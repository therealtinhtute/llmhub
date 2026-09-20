package translator

import (
	"context"
	"sync"

	log "github.com/sirupsen/logrus"
	"github.com/therealtinhtute/llmhub/internal/thinking"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// Registry manages translation functions across schemas.
type Registry struct {
	mu        sync.RWMutex
	requests  map[Format]map[Format]RequestEnvelopeTransform
	responses map[Format]map[Format]ResponseTransform
}

// NewRegistry constructs an empty translator registry.
func NewRegistry() *Registry {
	return &Registry{
		requests:  make(map[Format]map[Format]RequestEnvelopeTransform),
		responses: make(map[Format]map[Format]ResponseTransform),
	}
}

// Register stores request/response transforms between two formats.
func (r *Registry) Register(from, to Format, request RequestTransform, response ResponseTransform) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.requests[from]; !ok {
		r.requests[from] = make(map[Format]RequestEnvelopeTransform)
	}
	if request != nil {
		r.requests[from][to] = func(_ context.Context, req RequestEnvelope) RequestEnvelope {
			req.Body = request(req.Model, req.Body, req.Stream)
			return req
		}
	}

	if _, ok := r.responses[from]; !ok {
		r.responses[from] = make(map[Format]ResponseTransform)
	}
	r.responses[from][to] = response
}

// RegisterRequestEnvelope stores a request transform that consumes the complete
// request envelope, including request-scoped model metadata.
func (r *Registry) RegisterRequestEnvelope(from, to Format, request RequestEnvelopeTransform) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.requests[from]; !ok {
		r.requests[from] = make(map[Format]RequestEnvelopeTransform)
	}
	if request != nil {
		r.requests[from][to] = request
	}
}

// TranslateRequest converts a payload between schemas, returning the original payload
// if no translator is registered. When falling back to the original payload, the
// "model" field is still updated to match the resolved model name so that
// client-side prefixes (e.g. "copilot/gpt-5-mini") are not leaked upstream.
func (r *Registry) TranslateRequest(from, to Format, model string, rawJSON []byte, stream bool) []byte {
	req := r.TranslateRequestEnvelope(context.Background(), from, to, RequestEnvelope{
		Format: from,
		Model:  model,
		Stream: stream,
		Body:   rawJSON,
	})
	return req.Body
}

// TranslateRequestEnvelope translates a complete request envelope while preserving
// request-scoped metadata for the selected transform.
func (r *Registry) TranslateRequestEnvelope(ctx context.Context, from, to Format, req RequestEnvelope) RequestEnvelope {
	if ctx == nil {
		ctx = context.Background()
	}
	r.mu.RLock()
	var fn RequestEnvelopeTransform
	if byTarget, ok := r.requests[from]; ok {
		fn = byTarget[to]
	}
	r.mu.RUnlock()

	if fn != nil {
		summaryConfig := thinking.ExtractSummaryConfig(req.Body, from.String())
		req = fn(ctx, req)
		req.Body = thinking.ApplySummaryConfigForModel(req.Body, to.String(), req.Model, summaryConfig)
		req.Format = to
		return req
	}

	if req.Model != "" && gjson.GetBytes(req.Body, "model").String() != req.Model {
		if updated, err := sjson.SetBytes(req.Body, "model", req.Model); err != nil {
			log.Warnf("translator: failed to normalize model in request fallback: %v", err)
		} else {
			req.Body = updated
		}
	}
	req.Format = to
	return req
}

// HasResponseTransformer indicates whether a response translator exists.
func (r *Registry) HasResponseTransformer(from, to Format) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if byTarget, ok := r.responses[from]; ok {
		if _, isOk := byTarget[to]; isOk {
			return true
		}
	}
	return false
}

// TranslateStream applies the registered streaming response translator.
func (r *Registry) TranslateStream(ctx context.Context, from, to Format, model string, originalRequestRawJSON, requestRawJSON, rawJSON []byte, param *any) [][]byte {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if byTarget, ok := r.responses[to]; ok {
		if fn, isOk := byTarget[from]; isOk && fn.Stream != nil {
			return fn.Stream(ctx, model, originalRequestRawJSON, requestRawJSON, rawJSON, param)
		}
	}
	return [][]byte{rawJSON}
}

// TranslateNonStream applies the registered non-stream response translator.
func (r *Registry) TranslateNonStream(ctx context.Context, from, to Format, model string, originalRequestRawJSON, requestRawJSON, rawJSON []byte, param *any) []byte {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if byTarget, ok := r.responses[to]; ok {
		if fn, isOk := byTarget[from]; isOk && fn.NonStream != nil {
			return fn.NonStream(ctx, model, originalRequestRawJSON, requestRawJSON, rawJSON, param)
		}
	}
	return rawJSON
}

// TranslateTokenCount applies the registered token count response translator.
func (r *Registry) TranslateTokenCount(ctx context.Context, from, to Format, count int64, rawJSON []byte) []byte {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if byTarget, ok := r.responses[to]; ok {
		if fn, isOk := byTarget[from]; isOk && fn.TokenCount != nil {
			return fn.TokenCount(ctx, count)
		}
	}
	return rawJSON
}

var defaultRegistry = NewRegistry()

// Default exposes the package-level registry for shared use.
func Default() *Registry {
	return defaultRegistry
}

// Register attaches transforms to the default registry.
func Register(from, to Format, request RequestTransform, response ResponseTransform) {
	defaultRegistry.Register(from, to, request, response)
}

// RegisterRequestEnvelope stores an envelope-aware transform on the default registry.
func RegisterRequestEnvelope(from, to Format, request RequestEnvelopeTransform) {
	defaultRegistry.RegisterRequestEnvelope(from, to, request)
}

// TranslateRequest is a helper on the default registry.
func TranslateRequest(from, to Format, model string, rawJSON []byte, stream bool) []byte {
	return defaultRegistry.TranslateRequest(from, to, model, rawJSON, stream)
}

// TranslateRequestEnvelope translates a complete request envelope using the default registry.
func TranslateRequestEnvelope(ctx context.Context, from, to Format, req RequestEnvelope) RequestEnvelope {
	return defaultRegistry.TranslateRequestEnvelope(ctx, from, to, req)
}

// HasResponseTransformer inspects the default registry.
func HasResponseTransformer(from, to Format) bool {
	return defaultRegistry.HasResponseTransformer(from, to)
}

// TranslateStream is a helper on the default registry.
func TranslateStream(ctx context.Context, from, to Format, model string, originalRequestRawJSON, requestRawJSON, rawJSON []byte, param *any) [][]byte {
	return defaultRegistry.TranslateStream(ctx, from, to, model, originalRequestRawJSON, requestRawJSON, rawJSON, param)
}

// TranslateNonStream is a helper on the default registry.
func TranslateNonStream(ctx context.Context, from, to Format, model string, originalRequestRawJSON, requestRawJSON, rawJSON []byte, param *any) []byte {
	return defaultRegistry.TranslateNonStream(ctx, from, to, model, originalRequestRawJSON, requestRawJSON, rawJSON, param)
}

// TranslateTokenCount is a helper on the default registry.
func TranslateTokenCount(ctx context.Context, from, to Format, count int64, rawJSON []byte) []byte {
	return defaultRegistry.TranslateTokenCount(ctx, from, to, count, rawJSON)
}
