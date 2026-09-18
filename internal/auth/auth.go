// Package auth implements in-memory origin grants and session tokens.
package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
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
	// Persistent is an IT pre-approve. It does not expire until removed.
	Persistent bool
}

// Session binds a token to an origin grant and bridge instance.
// Session expiry is always capped at the underlying grant expiry.
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
	// ConnectionID binds the approval to the originating connection.
	ConnectionID string

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
	// recentApprovals tracks recent approval creations.
	recentApprovals []time.Time

	// NonInteractive denies new approvals when true (stdin not a TTY).
	NonInteractive bool

	// OnRevoke is invoked (outside the manager lock) with the origin and
	// tokens that were just revoked. Tokens may be empty when only a grant
	// existed.
	OnRevoke func(origin Origin, tokens []string)

	// OnDecision is invoked (outside the manager lock) after Allow or Deny.
	OnDecision func(origin Origin, fields []card.Field, allow bool)
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

// RequestApproval creates a pending request bound to connectionID and returns it.
func (m *Manager) RequestApproval(origin Origin, fields []card.Field, connectionID string, now time.Time) (*PendingRequest, error) {
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
	// Replace any still-open approval from the same connection.
	for id, p := range m.pending {
		if p.ConnectionID != "" && p.ConnectionID == connectionID && !p.decided {
			p.decided = true
			p.allowed = false
			close(p.done)
			delete(m.pending, id)
		}
	}
	if len(m.pending) >= m.maxPending {
		return nil, card.NewError(card.ErrRateLimited, "too many pending approvals")
	}

	comp := comparisonCode()
	id := m.nextID
	m.nextID++
	p := &PendingRequest{
		ID:           id,
		Origin:       origin,
		Fields:       append([]card.Field(nil), fields...),
		Comparison:   comp,
		ExpiresAt:    now.Add(m.approvalTTL),
		CreatedAt:    now,
		ConnectionID: connectionID,
		done:         make(chan struct{}),
	}
	m.pending[id] = p
	m.recentApprovals = append(m.recentApprovals, now)
	return p, nil
}

// CancelPendingByConnection closes undecided approvals for a connection.
func (m *Manager) CancelPendingByConnection(connectionID string) []int64 {
	if connectionID == "" {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	var ids []int64
	for id, p := range m.pending {
		if p.ConnectionID == connectionID && !p.decided {
			p.decided = true
			p.allowed = false
			close(p.done)
			delete(m.pending, id)
			ids = append(ids, id)
		}
	}
	return ids
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
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// Resolve applies an operator decision.
func (m *Manager) Resolve(id int64, allow bool, now time.Time) (*PendingRequest, error) {
	m.mu.Lock()
	p, ok := m.pending[id]
	if !ok {
		m.mu.Unlock()
		return nil, card.NewError(card.ErrInvalidRequest, "unknown approval request")
	}
	if now.After(p.ExpiresAt) {
		delete(m.pending, id)
		if !p.decided {
			p.decided = true
			p.allowed = false
			close(p.done)
		}
		m.mu.Unlock()
		return nil, card.NewError(card.ErrApprovalTimeout, "approval expired")
	}
	p.decided = true
	p.allowed = allow
	close(p.done)
	delete(m.pending, id)
	if allow {
		exp := now.Add(m.trustTTL)
		persistent := false
		fields := append([]card.Field(nil), p.Fields...)
		if existing, ok := m.grants[p.Origin]; ok && existing.Persistent {
			persistent = true
			exp = existing.ExpiresAt
			fields = unionFields(existing.Fields, fields)
		}
		m.grants[p.Origin] = &Grant{
			Origin:     p.Origin,
			Fields:     fields,
			ExpiresAt:  exp,
			Comparison: p.Comparison,
			Persistent: persistent,
		}
	} else {
		m.denialCooldownUntil = now.Add(5 * time.Second)
	}
	origin := p.Origin
	fields := append([]card.Field(nil), p.Fields...)
	m.mu.Unlock()
	if m.OnDecision != nil {
		m.OnDecision(origin, fields, allow)
	}
	return p, nil
}

// Wait blocks until the pending request is resolved, expires, or ctx is done.
func (m *Manager) Wait(ctx context.Context, p *PendingRequest) (bool, error) {
	timer := time.NewTimer(time.Until(p.ExpiresAt))
	defer timer.Stop()
	select {
	case <-p.done:
		return p.allowed, nil
	case <-ctx.Done():
		m.CancelPendingByConnection(p.ConnectionID)
		return false, card.NewError(card.ErrCancelled, "wait cancelled")
	case <-timer.C:
		m.mu.Lock()
		if existing, ok := m.pending[p.ID]; ok && existing == p && !p.decided {
			p.decided = true
			p.allowed = false
			close(p.done)
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
	if !ok || (!g.Persistent && !now.Before(g.ExpiresAt)) {
		return nil, false
	}
	cp := *g
	cp.Fields = append([]card.Field(nil), g.Fields...)
	return &cp, true
}

// OpenSession authenticates a connection under an existing live grant.
// Field expansion beyond the grant is rejected.
// Session expiry is capped at the grant expiry.
func (m *Manager) OpenSession(origin Origin, fields []card.Field, now time.Time) (*Session, error) {
	fields, err := card.NormalizeFields(fields)
	if err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.gcLocked(now)

	g, ok := m.grants[origin]
	if !ok || (!g.Persistent && !now.Before(g.ExpiresAt)) {
		return nil, card.NewError(card.ErrApprovalRequired, "origin not trusted")
	}
	if !card.Subset(fields, g.Fields) {
		return nil, card.NewError(card.ErrFieldNotAllowed, "field not in grant")
	}
	tok, err := randomToken()
	if err != nil {
		return nil, card.NewError(card.ErrConnectionFailed, "token generation failed")
	}
	exp := now.Add(m.sessionTTL)
	if exp.After(g.ExpiresAt) {
		exp = g.ExpiresAt
	}
	s := &Session{
		Token:      tok,
		Origin:     origin,
		Fields:     append([]card.Field(nil), fields...),
		InstanceID: m.instanceID,
		ExpiresAt:  exp,
	}
	m.sessions[tok] = s
	cp := *s
	cp.Fields = append([]card.Field(nil), s.Fields...)
	return &cp, nil
}

// Session validates a bearer token against origin, instance, live grant,
// and field set. The grant must still be live; an expired grant invalidates
// an otherwise unexpired token.
func (m *Manager) Session(token string, origin Origin, now time.Time) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.gcLocked(now)
	s, ok := m.sessions[token]
	if !ok || !now.Before(s.ExpiresAt) || s.Origin != origin || s.InstanceID != m.instanceID {
		return nil, card.NewError(card.ErrAuthExpired, "session invalid")
	}
	g, ok := m.grants[origin]
	if !ok || (!g.Persistent && !now.Before(g.ExpiresAt)) {
		delete(m.sessions, token)
		return nil, card.NewError(card.ErrAuthExpired, "grant expired")
	}
	if !card.Subset(s.Fields, g.Fields) {
		delete(m.sessions, token)
		return nil, card.NewError(card.ErrFieldNotAllowed, "grant fields narrowed")
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

// RevokeOrigin cancels the grant, pending approvals, and all sessions for an
// origin. It returns the revoked session tokens and fires OnRevoke outside
// the lock.
func (m *Manager) RevokeOrigin(origin Origin) int {
	if n, err := NormalizeOrigin(string(origin)); err == nil {
		origin = n
	}
	m.mu.Lock()
	delete(m.grants, origin)
	var tokens []string
	for tok, s := range m.sessions {
		if s.Origin == origin {
			delete(m.sessions, tok)
			tokens = append(tokens, tok)
		}
	}
	for id, p := range m.pending {
		if p.Origin == origin && !p.decided {
			p.decided = true
			p.allowed = false
			close(p.done)
			delete(m.pending, id)
		}
	}
	m.mu.Unlock()
	if m.OnRevoke != nil {
		m.OnRevoke(origin, tokens)
	}
	return len(tokens)
}

// RevokeAll clears every grant, pending approval, and session.
func (m *Manager) RevokeAll() {
	m.mu.Lock()
	type revoked struct {
		origin Origin
		tokens []string
	}
	byOrigin := map[Origin][]string{}
	for tok, s := range m.sessions {
		g := m.grants[s.Origin]
		if g != nil && g.Persistent {
			continue
		}
		byOrigin[s.Origin] = append(byOrigin[s.Origin], tok)
		delete(m.sessions, tok)
	}
	for _, p := range m.pending {
		if !p.decided {
			p.decided = true
			p.allowed = false
			close(p.done)
		}
	}
	m.pending = map[int64]*PendingRequest{}
	kept := map[Origin]*Grant{}
	for o, g := range m.grants {
		if g != nil && g.Persistent {
			kept[o] = g
			continue
		}
		if _, ok := byOrigin[o]; !ok {
			byOrigin[o] = nil
		}
	}
	m.grants = kept
	m.mu.Unlock()
	if m.OnRevoke != nil {
		for o, toks := range byOrigin {
			m.OnRevoke(o, toks)
		}
	}
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
	sort.Slice(out, func(i, j int) bool { return out[i].Origin < out[j].Origin })
	return out
}

func (m *Manager) gcLocked(now time.Time) {
	for id, p := range m.pending {
		if now.After(p.ExpiresAt) {
			if !p.decided {
				p.decided = true
				p.allowed = false
				close(p.done)
			}
			delete(m.pending, id)
		}
	}
	for o, g := range m.grants {
		if g.Persistent {
			continue
		}
		if !now.Before(g.ExpiresAt) {
			delete(m.grants, o)
		}
	}
	for tok, s := range m.sessions {
		if !now.Before(s.ExpiresAt) {
			delete(m.sessions, tok)
			continue
		}
		g, ok := m.grants[s.Origin]
		if !ok || (!g.Persistent && !now.Before(g.ExpiresAt)) {
			delete(m.sessions, tok)
		}
	}
}

// PersistentUntil is the grant expiry stored for IT pre-approve. gc and
// OpenSession skip TTL for Persistent grants; session tokens still use
// sessionTTL.
func PersistentUntil() time.Time {
	return time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)
}

func unionFields(a, b []card.Field) []card.Field {
	seen := make(map[card.Field]struct{}, len(a)+len(b))
	out := make([]card.Field, 0, len(a)+len(b))
	for _, f := range a {
		if _, ok := seen[f]; ok {
			continue
		}
		seen[f] = struct{}{}
		out = append(out, f)
	}
	for _, f := range b {
		if _, ok := seen[f]; ok {
			continue
		}
		seen[f] = struct{}{}
		out = append(out, f)
	}
	return out
}

func (m *Manager) preapproveLocked(origin Origin, fields []card.Field, now time.Time) error {
	_ = now
	fields, err := card.NormalizeFields(fields)
	if err != nil {
		return err
	}
	if existing, ok := m.grants[origin]; ok {
		fields = unionFields(existing.Fields, fields)
	}
	m.grants[origin] = &Grant{
		Origin:     origin,
		Fields:     append([]card.Field(nil), fields...),
		ExpiresAt:  PersistentUntil(),
		Persistent: true,
	}
	return nil
}

// Preapprove records an explicit IT clinic-website grant. It does not expire
// until RevokeOrigin. Wildcards are rejected by NormalizeOrigin.
func (m *Manager) Preapprove(origin Origin, fields []card.Field, now time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.gcLocked(now)
	return m.preapproveLocked(origin, fields, now)
}

// ApplyPreapproved seeds persistent grants for each explicit origin. Invalid
// entries are skipped. This is not “allow all”.
func (m *Manager) ApplyPreapproved(origins []string, fields []card.Field, now time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.gcLocked(now)
	for _, raw := range origins {
		o, err := NormalizeOrigin(raw)
		if err != nil {
			continue
		}
		_ = m.preapproveLocked(o, fields, now)
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
	n := int(b[0])<<16 | int(b[1])<<8 | int(b[2])
	n %= 1000000
	digits := []byte("000000")
	for i := 5; i >= 0; i-- {
		digits[i] = byte('0' + n%10)
		n /= 10
	}
	return string(digits)
}

// EscapeTerminal renders untrusted text for terminal display.
func EscapeTerminal(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r < 0x20 || r == 0x7f {
			fmt.Fprintf(&b, "\\x%02x", r)
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
