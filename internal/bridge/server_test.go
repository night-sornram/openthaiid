package bridge

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/night-sornram/openthaiid/internal/auth"
	"github.com/night-sornram/openthaiid/internal/config"
	"github.com/night-sornram/openthaiid/internal/pcsc"
	"github.com/night-sornram/openthaiid/internal/reader"
)

func TestHealthz(t *testing.T) {
	cfg := config.Default()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	authMgr := auth.NewManager("bridge-test", time.Hour, time.Minute, time.Minute, 4)
	mgr := reader.NewManager(pcsc.NewSynthetic(), pcsc.NewSynthetic())
	s := New(cfg, log, authMgr, mgr)

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealthz)
	h := s.securityMiddleware(mux)

	req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:17891/healthz", nil)
	req.Host = "127.0.0.1:17891"
	req.RemoteAddr = "127.0.0.1:5555"
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestRejectsNonLoopbackPeer(t *testing.T) {
	cfg := config.Default()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	authMgr := auth.NewManager("bridge-test", time.Hour, time.Minute, time.Minute, 4)
	mgr := reader.NewManager(pcsc.NewSynthetic(), pcsc.NewSynthetic())
	s := New(cfg, log, authMgr, mgr)
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealthz)
	h := s.securityMiddleware(mux)

	req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:17891/healthz", nil)
	req.Host = "127.0.0.1:17891"
	req.RemoteAddr = "8.8.8.8:1234"
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status %d", rr.Code)
	}
}

func TestRejectsRebindingHost(t *testing.T) {
	cfg := config.Default()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	authMgr := auth.NewManager("bridge-test", time.Hour, time.Minute, time.Minute, 4)
	mgr := reader.NewManager(pcsc.NewSynthetic(), pcsc.NewSynthetic())
	s := New(cfg, log, authMgr, mgr)
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealthz)
	h := s.securityMiddleware(mux)

	req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:17891/healthz", nil)
	req.Host = "evil.example.com:17891"
	req.RemoteAddr = "127.0.0.1:5555"
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status %d", rr.Code)
	}
}

func TestWSRejectsBadOrigin(t *testing.T) {
	cfg := config.Default()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	authMgr := auth.NewManager("bridge-test", time.Hour, time.Minute, time.Minute, 4)
	mgr := reader.NewManager(pcsc.NewSynthetic(), pcsc.NewSynthetic())
	s := New(cfg, log, authMgr, mgr)
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/ws", s.handleWS)
	h := s.securityMiddleware(mux)

	for _, origin := range []string{"", "null", "http://evil.example.com", "https://*.example.com"} {
		req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:17891/v1/ws", nil)
		req.Host = "127.0.0.1:17891"
		req.RemoteAddr = "127.0.0.1:5555"
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusForbidden {
			t.Fatalf("origin %q: status %d", origin, rr.Code)
		}
	}
}

func TestWSAllowsEligibleOriginUpgradeAttempt(t *testing.T) {
	cfg := config.Default()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	authMgr := auth.NewManager("bridge-test", time.Hour, time.Minute, time.Minute, 4)
	mgr := reader.NewManager(pcsc.NewSynthetic(), pcsc.NewSynthetic())
	s := New(cfg, log, authMgr, mgr)
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/ws", s.handleWS)
	h := s.securityMiddleware(mux)

	req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:17891/v1/ws", nil)
	req.Host = "127.0.0.1:17891"
	req.RemoteAddr = "127.0.0.1:5555"
	req.Header.Set("Origin", "https://clinic.example.com")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	// Eligible origin is not 403. Without a WebSocket upgrade the Accept
	// helper returns 426 Upgrade Required.
	if rr.Code == http.StatusForbidden {
		t.Fatalf("eligible origin must not be forbidden, got %d", rr.Code)
	}
	if rr.Code != http.StatusUpgradeRequired && rr.Code != http.StatusOK {
		t.Fatalf("unexpected status %d", rr.Code)
	}
}
