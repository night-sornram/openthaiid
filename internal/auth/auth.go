// Package auth implements in-memory origin grants and session tokens.
package auth

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	"github.com/night-sornram/openthaiid/internal/card"
)

// Origin is a normalized browser origin (scheme://host[:port]).
type Origin string

// Grant is an in-memory trust decision for one origin and field set.
type Grant struct {
	Origin     Origin
	Fields     []card.Field
	ExpiresAt  time.Time
	Comparison string
}

// Session binds a token to an origin grant and bridge instance.
type Session struct {
	Token      string
	Origin     Origin
	Fields     []card.Field
	InstanceID string
	ExpiresAt  time.Time
}

// PendingRequest is an approval waiting on the terminal.
type PendingRequest struct {
	ID         int64
	Origin     Origin
	Fields     []card.Field
	Comparison string
	ExpiresAt  time.Time
	CreatedAt  time.Time
	// Result is closed when the operator allows or denies.
	// Value is stored in decided/allowed.
	done    chan struct{}
	allowed bool
	decided bool
}

// Manager holds grants, sessions, and pending approvals.
type Manager struct {
	mu sync.Mutex

	instanceID  string
	trustTTL    time.Duration
	sessionTTL  time.Duration
	approvalTTL time.Duration
	maxPending  int

	pending  map[int64]*PendingRequest
	nextID   int64
	grants   map[Origin]*Grant
	sessions map[string]*Session

	// denialCooldownUntil blocks new approvals briefly after a denial.
	denialCooldownUntil time.Time
	// approvalRate tracks recent approval creations.
	recentApprovals []time.Time

	// NonInteractive denies new approvals when true (stdin not a TTY).
	NonInteractive bool
}

// NewManager constructs an authorization manager.
func NewManager(instanceID string, trustTTL, sessionTTL, approvalTTL time.Duration, maxPending int) *Manager {
	if maxPending < 1 {
		maxPending = 1
	}
	return &Manager{
		instanceID:  instanceID,
		trustTTL:    trustTTL,
		sessionTTL:  sessionTTL,
		approvalTTL: approvalTTL,
		maxPending:  maxPending,
		pending:     map[int64]*PendingRequest{},
		grants:      map[Origin]*Grant{},
		sessions:    map[string]*Session{},
		nextID:      1,
	}
}

// InstanceID returns the bridge instance identifier.
func (m *Manager) InstanceID() string { return m.instanceID }

// RequestApproval creates a pending request and returns it.
func (m *Manager) RequestApproval(origin Origin, fields []card.Field, now time.Time) (*PendingRequest, error) {
	fields, err := card.NormalizeFields(fields)
	if err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.gcLocked(now)

	if m.NonInteractive {
		return nil, card.NewError(card.ErrApprovalDenied, "noninteractive process")
	}
	if now.Before(m.denialCooldownUntil) {
		return nil, card.NewError(card.ErrRateLimited, "approval cooldown")
	}
	// Rate limit: at most 4 approvals in the last minute.
	cut := now.Add(-time.Minute)
	kept := m.recentApprovals[:0]
	for _, t := range m.recentApprovals {
		if t.After(cut) {
			kept = append(kept, t)
		}
	}
	m.recentApprovals = kept
	if len(m.recentApprovals) >= 4 {
		return nil, card.NewError(card.ErrRateLimited, "too many approval requests")
	}
	if len(m.pending) >= m.maxPending {
		return nil, card.NewError(card.ErrRateLimited, "too many pending approvals")
	}

	// Existing valid grant with a superset field set can be reused by the
	// caller without a new terminal prompt — handled in OpenSession.
	comp := comparisonCode()
	id := m.nextID
	m.nextID++
	p := &PendingRequest{
		ID:         id,
		Origin:     origin,
		Fields:     append([]card.Field(nil), fields...),
		Comparison: comp,
		ExpiresAt:  now.Add(m.approvalTTL),
		CreatedAt:  now,
		done:       make(chan struct{}),
	}
	m.pending[id] = p
	m.recentApprovals = append(m.recentApprovals, now)
	return p, nil
}

// Pending returns open pending requests (for terminal display).
func (m *Manager) Pending(now time.Time) []*PendingRequest {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.gcLocked(now)
	out := make([]*PendingRequest, 0, len(m.pending))
	for _, p := range m.pending {
		out = append(out, p)
	}
	return out
}

// Resolve applies an operator decision.
func (m *Manager) Resolve(id int64, allow bool, now time.Time) (*PendingRequest, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.pending[id]
	if !ok {
		return nil, card.NewError(card.ErrInvalidRequest, "unknown approval request")
	}
	if now.After(p.ExpiresAt) {
		delete(m.pending, id)
		return nil, card.NewError(card.ErrApprovalTimeout, "approval expired")
	}
	p.decided = true
	p.allowed = allow
	close(p.done)
	delete(m.pending, id)
	if allow {
		m.grants[p.Origin] = &Grant{
			Origin:     p.Origin,
			Fields:     append([]card.Field(nil), p.Fields...),
			ExpiresAt:  now.Add(m.trustTTL),
			Comparison: p.Comparison,
		}
	} else {
		m.denialCooldownUntil = now.Add(5 * time.Second)
	}
	return p, nil
}

// Wait blocks until the pending request is resolved or expires.
func (m *Manager) Wait(p *PendingRequest, now time.Time) (bool, error) {
	timer := time.NewTimer(time.Until(p.ExpiresAt))
	defer timer.Stop()
	select {
	case <-p.done:
		return p.allowed, nil
	case <-timer.C:
		m.mu.Lock()
		if _, ok := m.pending[p.ID]; ok {
			delete(m.pending, p.ID)
		}
		m.mu.Unlock()
		return false, card.NewError(card.ErrApprovalTimeout, "approval expired")
	}
}

// GrantFor returns a live grant for origin, if any.
func (m *Manager) GrantFor(origin Origin, now time.Time) (*Grant, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.gcLocked(now)
	g, ok := m.grants[origin]
	if !ok {
		return nil, false
	}
	cp := *g
	cp.Fields = append([]card.Field(nil), g.Fields...)
	return &cp, true
}

// OpenSession authenticates a connection under an existing grant.
// Field expansion beyond the grant is rejected.
func (m *Manager) OpenSession(origin Origin, fields []card.Field, now time.Time) (*Session, error) {
	fields, err := card.NormalizeFields(fields)
	if err != nil {
		return nil, err
	}
	g, ok := m.GrantFor(origin, now)
	if !ok {
		return nil, card.NewError(card.ErrApprovalRequired, "origin not trusted")
	}
	if !card.Subset(fields, g.Fields) {
		return nil, card.NewError(card.ErrFieldNotAllowed, "field not in grant")
	}
	tok, err := randomToken()
	if err != nil {
		return nil, card.NewError(card.ErrConnectionFailed, "token generation failed")
	}
	s := &Session{
		Token:      tok,
		Origin:     origin,
		Fields:     append([]card.Field(nil), fields...),
		InstanceID: m.instanceID,
		ExpiresAt:  now.Add(m.sessionTTL),
	}
	m.mu.Lock()
	m.sessions[tok] = s
	m.mu.Unlock()
	cp := *s
	cp.Fields = append([]card.Field(nil), s.Fields...)
	return &cp, nil
}

// Session validates a bearer token against origin and instance.
func (m *Manager) Session(token string, origin Origin, now time.Time) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.gcLocked(now)
	s, ok := m.sessions[token]
	if !ok || s.ExpiresAt.Before(now) || s.Origin != origin || s.InstanceID != m.instanceID {
		return nil, card.NewError(card.ErrAuthExpired, "session invalid")
	}
	cp := *s
	cp.Fields = append([]card.Field(nil), s.Fields...)
	return &cp, nil
}

// CloseSession removes a token.
func (m *Manager) CloseSession(token string) {
	m.mu.Lock()
	delete(m.sessions, token)
	m.mu.Unlock()
}

// RevokeOrigin cancels the grant and all sessions for an origin.
func (m *Manager) RevokeOrigin(origin Origin) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.grants, origin)
	n := 0
	for tok, s := range m.sessions {
		if s.Origin == origin {
			delete(m.sessions, tok)
			n++
		}
	}
	return n
}

// RevokeAll clears every grant and session.
func (m *Manager) RevokeAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.grants = map[Origin]*Grant{}
	m.sessions = map[string]*Session{}
}

// TrustedOrigins lists live grants (for terminal `trust`).
func (m *Manager) TrustedOrigins(now time.Time) []Grant {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.gcLocked(now)
	out := make([]Grant, 0, len(m.grants))
	for _, g := range m.grants {
		cp := *g
		cp.Fields = append([]card.Field(nil), g.Fields...)
		out = append(out, cp)
	}
	return out
}

func (m *Manager) gcLocked(now time.Time) {
	for id, p := range m.pending {
		if now.After(p.ExpiresAt) {
			// Do not close(p.done) here if already decided — only undelivered.
			if !p.decided {
				p.decided = true
				p.allowed = false
				close(p.done)
			}
			delete(m.pending, id)
		}
	}
	for o, g := range m.grants {
		if now.After(g.ExpiresAt) {
			delete(m.grants, o)
		}
	}
	for tok, s := range m.sessions {
		if now.After(s.ExpiresAt) {
			delete(m.sessions, tok)
		}
	}
}

func randomToken() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

func comparisonCode() string {
	var b [3]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "000000"
	}
	// Six decimal digits from 3 bytes.
	n := int(b[0])<<16 | int(b[1])<<8 | int(b[2])
	n %= 1000000
	s := "000000"
	digits := []byte(s)
	for i := 5; i >= 0; i-- {
		digits[i] = byte('0' + n%10)
		n /= 10
	}
	return string(digits)
}
