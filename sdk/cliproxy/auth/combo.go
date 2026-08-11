package auth

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	internalconfig "github.com/therealtinhtute/llmhub/internal/config"
)

// ComboCandidate is one provider/model entry of a combo's candidate list.
type ComboCandidate struct {
	Provider string
	Model    string
}

// comboRotation tracks round-robin state for one combo: the current starting
// index and how many consecutive requests have been served from it.
type comboRotation struct {
	index int
	count int
}

// ComboResolver resolves combo names to ordered candidate lists with fallback
// or round-robin rotation. Cursor state is in-memory, mutex-guarded, and cleared
// whenever SetCombos replaces the definitions (i.e. on config reload).
type ComboResolver struct {
	mu     sync.Mutex
	combos map[string]internalconfig.ComboConfig
	cursor map[string]*comboRotation
}

// NewComboResolver returns an empty resolver.
func NewComboResolver() *ComboResolver {
	return &ComboResolver{
		combos: make(map[string]internalconfig.ComboConfig),
		cursor: make(map[string]*comboRotation),
	}
}

// SetCombos replaces the combo definitions and clears all rotation cursors.
func (r *ComboResolver) SetCombos(combos []internalconfig.ComboConfig) {
	r.mu.Lock()
	defer r.mu.Unlock()
	next := make(map[string]internalconfig.ComboConfig, len(combos))
	for _, combo := range combos {
		next[combo.Name] = combo
	}
	r.combos = next
	r.cursor = make(map[string]*comboRotation)
}

// ComboNames returns the names of the currently defined combos.
func (r *ComboResolver) ComboNames() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	names := make([]string, 0, len(r.combos))
	for name := range r.combos {
		names = append(names, name)
	}
	return names
}

// Resolve returns the candidate list in config order when name is a combo.
// fallback strategy always starts at index 0, so this is the fallback path.
func (r *ComboResolver) Resolve(name string) ([]ComboCandidate, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	combo, ok := r.combos[name]
	if !ok {
		return nil, false
	}
	return candidatesOf(combo), true
}

// Rotate returns the candidate list ordered for this request. round-robin keeps
// a per-combo cursor and serves the same starting candidate for stickyLimit
// consecutive requests before advancing; fallback always returns config order.
func (r *ComboResolver) Rotate(name, strategy string, stickyLimit int) []ComboCandidate {
	r.mu.Lock()
	defer r.mu.Unlock()
	combo, ok := r.combos[name]
	if !ok {
		return nil
	}
	if !strings.EqualFold(strings.TrimSpace(strategy), "round-robin") {
		return candidatesOf(combo)
	}
	return r.rotateLocked(name, combo, stickyLimit)
}

// Order returns the candidate list for this request, applying the combo's own
// strategy and sticky limit: fallback stays in config order, round-robin rotates.
func (r *ComboResolver) Order(name string) ([]ComboCandidate, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	combo, ok := r.combos[name]
	if !ok {
		return nil, false
	}
	if !strings.EqualFold(strings.TrimSpace(combo.Strategy), "round-robin") {
		return candidatesOf(combo), true
	}
	return r.rotateLocked(name, combo, combo.StickyLimit), true
}

// rotateLocked advances the per-combo rotation cursor for round-robin. Caller holds r.mu.
func (r *ComboResolver) rotateLocked(name string, combo internalconfig.ComboConfig, stickyLimit int) []ComboCandidate {
	if len(combo.Models) == 0 {
		return nil
	}
	if stickyLimit < 1 {
		stickyLimit = 1
	}
	state := r.cursor[name]
	if state == nil {
		state = &comboRotation{}
		r.cursor[name] = state
	}
	if state.count >= stickyLimit {
		state.index = (state.index + 1) % len(combo.Models)
		state.count = 0
	}
	state.count++
	return rotatedCandidates(combo.Models, state.index)
}

// ComboExhaustedError reports that every candidate of a combo failed.
type ComboExhaustedError struct {
	// Candidates is the number of candidates attempted.
	Candidates int
	// ResetAt is the earliest cooldown reset across the candidates (zero when unknown).
	ResetAt time.Time
	// Reset is the duration until ResetAt at error creation time (0 when unknown).
	Reset time.Duration
}

func (e *ComboExhaustedError) Error() string {
	if e == nil || e.Candidates <= 0 {
		return "all combo candidates failed"
	}
	if e.Reset <= 0 {
		return fmt.Sprintf("all %d combo candidates failed", e.Candidates)
	}
	return fmt.Sprintf("all %d combo candidates failed; reset after %s", e.Candidates, e.Reset.Truncate(time.Second))
}

// StatusCode reports the HTTP status for an exhausted combo request.
func (e *ComboExhaustedError) StatusCode() int {
	return http.StatusServiceUnavailable
}

func candidatesOf(combo internalconfig.ComboConfig) []ComboCandidate {
	out := make([]ComboCandidate, 0, len(combo.Models))
	for _, candidate := range combo.Models {
		if provider, model, ok := internalconfig.ParseComboCandidate(candidate); ok {
			out = append(out, ComboCandidate{Provider: provider, Model: model})
		}
	}
	return out
}

func rotatedCandidates(models []string, start int) []ComboCandidate {
	out := make([]ComboCandidate, 0, len(models))
	for i := 0; i < len(models); i++ {
		candidate := models[(start+i)%len(models)]
		if provider, model, ok := internalconfig.ParseComboCandidate(candidate); ok {
			out = append(out, ComboCandidate{Provider: provider, Model: model})
		}
	}
	return out
}
