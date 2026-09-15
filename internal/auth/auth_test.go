package auth

import (
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
		{"", "", false},
		{"null", "", false},
		{"*", "", false},
		{"https://*.example.com", "", false},
		{"ftp://example.com", "", false},
		{"https://user:pass@example.com", "", false},
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

	p, err := m.RequestApproval(origin, fields, now)
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
	p, err := m.RequestApproval(origin, []card.Field{card.FieldCitizenID}, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.Resolve(p.ID, false, now); err != nil {
		t.Fatal(err)
	}
	if _, err := m.OpenSession(origin, []card.Field{card.FieldCitizenID}, now); err == nil {
		t.Fatal("denied origin must not get a session")
	}
	// Cooldown blocks new approvals.
	if _, err := m.RequestApproval(origin, []card.Field{card.FieldCitizenID}, now); err == nil {
		t.Fatal("expected cooldown rate limit")
	}
}

func TestNonInteractiveFailsClosed(t *testing.T) {
	now := time.Now()
	m := NewManager("b", time.Hour, time.Minute, 2*time.Second, 4)
	m.NonInteractive = true
	origin, _ := NormalizeOrigin("https://a.example")
	if _, err := m.RequestApproval(origin, []card.Field{card.FieldCitizenID}, now); err == nil {
		t.Fatal("noninteractive must deny new approvals")
	}
}

func TestTokenExpiry(t *testing.T) {
	now := time.Now()
	m := NewManager("b", time.Hour, time.Minute, 2*time.Second, 4)
	origin, _ := NormalizeOrigin("https://a.example")
	p, _ := m.RequestApproval(origin, []card.Field{card.FieldCitizenID}, now)
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
