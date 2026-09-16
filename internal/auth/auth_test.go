package auth

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/night-sornram/openthaiid/internal/card"
)

func TestNormalizeOrigin(t *testing.T) {
	cases := []struct {
		in   string
		want string
		ok   bool
	}{
		{"https://Clinic.Example.com", "https://clinic.example.com", true},
		{"https://clinic.example.com:443", "https://clinic.example.com", true},
		{"http://127.0.0.1:5173", "http://127.0.0.1:5173", true},
		{"http://[::1]:5173", "http://[::1]:5173", true},
		{"", "", false},
		{"null", "", false},
		{"*", "", false},
		{"https://*.example.com", "", false},
		{"ftp://example.com", "", false},
		{"https://user:pass@example.com", "", false},
		{"https://clinic.example.com/path", "", false},
		{"https://clinic.example.com?q=1", "", false},
		{"https://clinic.example.com#frag", "", false},
		{"https://clinic.example.com:99999", "", false},
		{"https://clinic.example.com:abc", "", false},
		{"https://clinic.example.com:0", "", false},
	}
	for _, tc := range cases {
		got, err := NormalizeOrigin(tc.in)
		if tc.ok && err != nil {
			t.Fatalf("%q: unexpected err %v", tc.in, err)
		}
		if !tc.ok && err == nil {
			t.Fatalf("%q: expected error", tc.in)
		}
		if tc.ok && string(got) != tc.want {
			t.Fatalf("%q: got %q want %q", tc.in, got, tc.want)
		}
	}
}

func TestEligible(t *testing.T) {
	https, _ := NormalizeOrigin("https://app.example.com")
	httpLocal, _ := NormalizeOrigin("http://127.0.0.1:5173")
	httpRemote, _ := NormalizeOrigin("http://evil.example.com")
	if !Eligible(https, false, nil) {
		t.Fatal("https should be eligible")
	}
	if !Eligible(httpLocal, true, nil) {
		t.Fatal("http loopback should be eligible when allowed")
	}
	if Eligible(httpLocal, false, nil) {
		t.Fatal("http loopback should be ineligible when disabled")
	}
	if Eligible(httpRemote, true, nil) {
		t.Fatal("http non-loopback should be ineligible")
	}
}

func TestValidHostAuthority(t *testing.T) {
	if !ValidHostAuthority("127.0.0.1:17891", "127.0.0.1:17891") {
		t.Fatal("exact host should match")
	}
	if ValidHostAuthority("attacker.example:17891", "127.0.0.1:17891") {
		t.Fatal("rebinding host must be rejected")
	}
}

func TestIsLoopbackPeer(t *testing.T) {
	if !IsLoopbackPeer("127.0.0.1:54321") {
		t.Fatal("127.0.0.1 should be loopback")
	}
	if IsLoopbackPeer("10.0.0.5:54321") {
		t.Fatal("10.0.0.5 should not be loopback")
	}
}

func TestApprovalFlow(t *testing.T) {
	now := time.Now()
	m := NewManager("bridge-test", time.Hour, time.Minute, 2*time.Second, 4)
	origin, _ := NormalizeOrigin("https://clinic.example.com")
	fields := []card.Field{card.FieldThaiName}

	p, err := m.RequestApproval(origin, fields, "conn-1", now)
	if err != nil {
		t.Fatal(err)
	}
	if p.Comparison == "" || p.ID == 0 {
		t.Fatal("pending request incomplete")
	}
	if _, err := m.OpenSession(origin, fields, now); err == nil {
		t.Fatal("session before approval should fail")
	}
	if _, err := m.Resolve(p.ID, true, now); err != nil {
		t.Fatal(err)
	}
	s, err := m.OpenSession(origin, fields, now)
	if err != nil {
		t.Fatal(err)
	}
	if s.Token == "" || s.InstanceID != "bridge-test" {
		t.Fatal("session incomplete")
	}
	if _, err := m.Session(s.Token, origin, now); err != nil {
		t.Fatal(err)
	}
	// Field expansion denied.
	if _, err := m.OpenSession(origin, []card.Field{card.FieldCitizenID, card.FieldThaiName, card.FieldBirthDate}, now); err == nil {
		t.Fatal("field expansion should require new approval")
	}
	m.RevokeOrigin(origin)
	if _, err := m.Session(s.Token, origin, now); err == nil {
		t.Fatal("session after revoke should fail")
	}
}

func TestDenial(t *testing.T) {
	now := time.Now()
	m := NewManager("b", time.Hour, time.Minute, 2*time.Second, 4)
	origin, _ := NormalizeOrigin("https://a.example")
	p, err := m.RequestApproval(origin, []card.Field{card.FieldCitizenID}, "c1", now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.Resolve(p.ID, false, now); err != nil {
		t.Fatal(err)
	}
	if _, err := m.OpenSession(origin, []card.Field{card.FieldCitizenID}, now); err == nil {
		t.Fatal("denied origin must not get a session")
	}
	if _, err := m.RequestApproval(origin, []card.Field{card.FieldCitizenID}, "c1", now); err == nil {
		t.Fatal("expected cooldown rate limit")
	}
}

func TestNonInteractiveFailsClosed(t *testing.T) {
	now := time.Now()
	m := NewManager("b", time.Hour, time.Minute, 2*time.Second, 4)
	m.NonInteractive = true
	origin, _ := NormalizeOrigin("https://a.example")
	if _, err := m.RequestApproval(origin, []card.Field{card.FieldCitizenID}, "c1", now); err == nil {
		t.Fatal("noninteractive must deny new approvals")
	}
}

func TestTokenExpiry(t *testing.T) {
	now := time.Now()
	m := NewManager("b", time.Hour, time.Minute, 2*time.Second, 4)
	origin, _ := NormalizeOrigin("https://a.example")
	p, _ := m.RequestApproval(origin, []card.Field{card.FieldCitizenID}, "c1", now)
	_, _ = m.Resolve(p.ID, true, now)
	s, err := m.OpenSession(origin, []card.Field{card.FieldCitizenID}, now)
	if err != nil {
		t.Fatal(err)
	}
	later := now.Add(2 * time.Minute)
	if _, err := m.Session(s.Token, origin, later); err == nil {
		t.Fatal("expired session must fail")
	}
}

func TestSessionCappedAtGrantExpiry(t *testing.T) {
	now := time.Now()
	m := NewManager("b", 30*time.Second, time.Hour, 2*time.Second, 4)
	origin, _ := NormalizeOrigin("https://a.example")
	p, _ := m.RequestApproval(origin, []card.Field{card.FieldCitizenID}, "c1", now)
	_, _ = m.Resolve(p.ID, true, now)
	s, err := m.OpenSession(origin, []card.Field{card.FieldCitizenID}, now)
	if err != nil {
		t.Fatal(err)
	}
	grantEnd := now.Add(30 * time.Second)
	if s.ExpiresAt.After(grantEnd) {
		t.Fatalf("session expiry %s exceeds grant %s", s.ExpiresAt, grantEnd)
	}
}

func TestExpiredGrantInvalidatesLiveToken(t *testing.T) {
	now := time.Now()
	m := NewManager("b", 30*time.Second, time.Hour, 2*time.Second, 4)
	origin, _ := NormalizeOrigin("https://a.example")
	p, _ := m.RequestApproval(origin, []card.Field{card.FieldCitizenID}, "c1", now)
	_, _ = m.Resolve(p.ID, true, now)
	// Craft a session that outlives the grant by temporarily using a long TTL
	// manager state: reopen after grant would expire is refused; Session() must
	// also refuse an existing token once the grant is gone.
	s, err := m.OpenSession(origin, []card.Field{card.FieldCitizenID}, now)
	if err != nil {
		t.Fatal(err)
	}
	// Manually expire the grant.
	m.mu.Lock()
	m.grants[origin].ExpiresAt = now.Add(time.Second)
	m.mu.Unlock()
	later := now.Add(2 * time.Second)
	if _, err := m.Session(s.Token, origin, later); err == nil {
		t.Fatal("expired grant must invalidate session token")
	}
}

func TestRevocationCallback(t *testing.T) {
	now := time.Now()
	m := NewManager("b", time.Hour, time.Minute, 2*time.Second, 4)
	var gotOrigin Origin
	var gotTokens []string
	m.OnRevoke = func(o Origin, tokens []string) {
		gotOrigin = o
		gotTokens = tokens
	}
	origin, _ := NormalizeOrigin("https://a.example")
	p, _ := m.RequestApproval(origin, []card.Field{card.FieldCitizenID}, "c1", now)
	_, _ = m.Resolve(p.ID, true, now)
	s, err := m.OpenSession(origin, []card.Field{card.FieldCitizenID}, now)
	if err != nil {
		t.Fatal(err)
	}
	if n := m.RevokeOrigin(origin); n != 1 {
		t.Fatalf("revoked %d", n)
	}
	if gotOrigin != origin || len(gotTokens) != 1 || gotTokens[0] != s.Token {
		t.Fatalf("callback %v %v", gotOrigin, gotTokens)
	}
}

func TestDecisionCallback(t *testing.T) {
	now := time.Now()
	m := NewManager("b", time.Hour, time.Minute, 2*time.Second, 4)
	var gotOrigin Origin
	var gotAllow bool
	var n int
	m.OnDecision = func(o Origin, fields []card.Field, allow bool) {
		gotOrigin = o
		gotAllow = allow
		n++
	}
	origin, _ := NormalizeOrigin("https://a.example")
	p, _ := m.RequestApproval(origin, []card.Field{card.FieldCitizenID}, "c1", now)
	if _, err := m.Resolve(p.ID, true, now); err != nil {
		t.Fatal(err)
	}
	if n != 1 || gotOrigin != origin || !gotAllow {
		t.Fatalf("allow callback n=%d origin=%s allow=%v", n, gotOrigin, gotAllow)
	}
	p2, _ := m.RequestApproval(origin, []card.Field{card.FieldThaiName}, "c2", now.Add(time.Second))
	if _, err := m.Resolve(p2.ID, false, now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if n != 2 || gotAllow {
		t.Fatalf("deny callback n=%d allow=%v", n, gotAllow)
	}
}

func TestRevokeOriginNormalizesCase(t *testing.T) {
	now := time.Now()
	m := NewManager("b", time.Hour, time.Minute, 2*time.Second, 4)
	origin, _ := NormalizeOrigin("https://clinic.example.com")
	p, _ := m.RequestApproval(origin, []card.Field{card.FieldCitizenID}, "c1", now)
	_, _ = m.Resolve(p.ID, true, now)
	if _, err := m.OpenSession(origin, []card.Field{card.FieldCitizenID}, now); err != nil {
		t.Fatal(err)
	}
	if n := m.RevokeOrigin(Origin("https://Clinic.Example.com")); n != 1 {
		t.Fatalf("revoked %d, want 1", n)
	}
	if _, ok := m.GrantFor(origin, now); ok {
		t.Fatal("grant should be gone after mixed-case revoke")
	}
}

func TestCancelPendingOnDisconnect(t *testing.T) {
	now := time.Now()
	m := NewManager("b", time.Hour, time.Minute, 5*time.Second, 4)
	origin, _ := NormalizeOrigin("https://a.example")
	p, err := m.RequestApproval(origin, []card.Field{card.FieldCitizenID}, "conn-x", now)
	if err != nil {
		t.Fatal(err)
	}
	ids := m.CancelPendingByConnection("conn-x")
	if len(ids) != 1 || ids[0] != p.ID {
		t.Fatalf("ids %v", ids)
	}
	if len(m.Pending(now)) != 0 {
		t.Fatal("pending should be empty after cancel")
	}
}

func TestWaitContextCancel(t *testing.T) {
	now := time.Now()
	m := NewManager("b", time.Hour, time.Minute, 10*time.Second, 4)
	origin, _ := NormalizeOrigin("https://a.example")
	p, err := m.RequestApproval(origin, []card.Field{card.FieldCitizenID}, "conn-y", now)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := m.Wait(ctx, p)
		done <- err
	}()
	cancel()
	select {
	case err := <-done:
		if err == nil || card.AsError(err).Code != card.ErrCancelled {
			t.Fatalf("err %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("wait did not honor context")
	}
}

func TestEscapeTerminal(t *testing.T) {
	got := EscapeTerminal("ok\x1b[31mred")
	if strings.Contains(got, "\x1b") {
		t.Fatalf("control char not escaped: %q", got)
	}
	if !strings.Contains(got, `\x1b`) {
		t.Fatalf("expected escaped form, got %q", got)
	}
}
