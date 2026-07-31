package live

import (
	"regexp"
	"sync"
	"time"
)

const SessionLifetime = time.Hour

var callIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`)

type Session struct {
	CallID    string
	AuthID    string
	Model     string
	Resources *SessionResources
	token     uint64
}

type SessionResources struct {
	mu      sync.Mutex
	closed  bool
	closers []func() error
}

type storedSession struct {
	session Session
	claimed bool
	timer   *time.Timer
}

type Store struct {
	mu       sync.Mutex
	next     uint64
	lifetime time.Duration
	sessions map[string]*storedSession
}

type Claim int

const (
	ClaimMissing Claim = iota
	ClaimBusy
	ClaimAcquired
)

func NewStore() *Store {
	return &Store{
		lifetime: SessionLifetime,
		sessions: make(map[string]*storedSession),
	}
}

func (s *Store) Put(callID string, session Session) Session {
	if s == nil || !ValidCallID(callID) {
		EndSession(session)
		return Session{}
	}
	if session.Resources == nil {
		session.Resources = &SessionResources{}
	}

	s.mu.Lock()
	s.next++
	session.CallID = callID
	session.token = s.next
	previous := s.sessions[callID]
	entry := &storedSession{session: session}
	entry.timer = time.AfterFunc(s.expiryDuration(), func() {
		s.expire(callID, session.token)
	})
	s.sessions[callID] = entry
	s.mu.Unlock()

	if previous != nil {
		if previous.timer != nil {
			previous.timer.Stop()
		}
		if previous.session.Resources != nil && previous.session.Resources != session.Resources {
			previous.session.Resources.Close()
		}
	}
	return session
}

func (s *Store) Claim(callID string) (Session, Claim) {
	if s == nil || !ValidCallID(callID) {
		return Session{}, ClaimMissing
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	entry := s.sessions[callID]
	if entry == nil {
		return Session{}, ClaimMissing
	}
	if entry.claimed {
		return Session{}, ClaimBusy
	}
	entry.claimed = true
	if entry.timer != nil {
		entry.timer.Stop()
		entry.timer = nil
	}
	return entry.session, ClaimAcquired
}

func (s *Store) Release(session Session) {
	if s == nil || session.CallID == "" {
		return
	}
	s.mu.Lock()
	entry := s.sessions[session.CallID]
	if entry == nil || entry.session.token != session.token || !entry.claimed {
		s.mu.Unlock()
		return
	}
	entry.claimed = false
	entry.timer = time.AfterFunc(s.expiryDuration(), func() {
		s.expire(session.CallID, session.token)
	})
	s.mu.Unlock()
}

func (s *Store) Complete(session Session) {
	if s == nil || session.CallID == "" {
		EndSession(session)
		return
	}
	s.mu.Lock()
	entry := s.sessions[session.CallID]
	if entry == nil || entry.session.token != session.token {
		s.mu.Unlock()
		return
	}
	delete(s.sessions, session.CallID)
	if entry.timer != nil {
		entry.timer.Stop()
	}
	s.mu.Unlock()
	EndSession(entry.session)
}

func (s *Store) CompleteCall(callID string) {
	if s == nil || !ValidCallID(callID) {
		return
	}
	s.mu.Lock()
	entry := s.sessions[callID]
	if entry == nil {
		s.mu.Unlock()
		return
	}
	delete(s.sessions, callID)
	if entry.timer != nil {
		entry.timer.Stop()
	}
	s.mu.Unlock()
	EndSession(entry.session)
}

func (s *Store) CloseAll() {
	if s == nil {
		return
	}
	s.mu.Lock()
	entries := make([]*storedSession, 0, len(s.sessions))
	for callID, entry := range s.sessions {
		delete(s.sessions, callID)
		if entry.timer != nil {
			entry.timer.Stop()
		}
		entries = append(entries, entry)
	}
	s.mu.Unlock()
	for _, entry := range entries {
		EndSession(entry.session)
	}
}

func (s *Store) Peek(callID string) (Session, bool) {
	if s == nil {
		return Session{}, false
	}
	s.mu.Lock()
	entry := s.sessions[callID]
	s.mu.Unlock()
	if entry == nil {
		return Session{}, false
	}
	return entry.session, true
}

func (s *Store) SetLifetime(lifetime time.Duration) {
	if s == nil || lifetime <= 0 {
		return
	}
	s.mu.Lock()
	s.lifetime = lifetime
	s.mu.Unlock()
}

func (s *Store) expiryDuration() time.Duration {
	if s.lifetime > 0 {
		return s.lifetime
	}
	return SessionLifetime
}

func (s *Store) expire(callID string, token uint64) {
	s.mu.Lock()
	entry := s.sessions[callID]
	if entry == nil || entry.session.token != token || entry.claimed {
		s.mu.Unlock()
		return
	}
	delete(s.sessions, callID)
	s.mu.Unlock()
	EndSession(entry.session)
}

func EndSession(session Session) {
	if session.Resources != nil {
		session.Resources.Close()
	}
}

func (r *SessionResources) Add(closers ...func() error) {
	if r == nil {
		return
	}
	r.mu.Lock()
	if !r.closed {
		r.closers = append(r.closers, closers...)
		r.mu.Unlock()
		return
	}
	r.mu.Unlock()
	closeResources(closers)
}

func (r *SessionResources) Close() {
	if r == nil {
		return
	}
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return
	}
	r.closed = true
	closers := r.closers
	r.closers = nil
	r.mu.Unlock()
	closeResources(closers)
}

func closeResources(closers []func() error) {
	for _, closer := range closers {
		if closer != nil {
			_ = closer()
		}
	}
}

func ValidCallID(callID string) bool {
	return callIDPattern.MatchString(callID)
}
