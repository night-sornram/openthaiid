package bridge

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/night-sornram/openthaiid/internal/audit"
	"github.com/night-sornram/openthaiid/internal/auth"
	"github.com/night-sornram/openthaiid/internal/card"
	"github.com/night-sornram/openthaiid/internal/config"
)

// SetConfigPath sets the JSON file used to persist IT pre-approved origins.
// Tests must point this at a temp file so the real user config is not written.
func (s *Server) SetConfigPath(path string) {
	s.mu.Lock()
	s.configPath = path
	s.mu.Unlock()
}

func (s *Server) configFilePath() string {
	s.mu.Lock()
	path := s.configPath
	s.mu.Unlock()
	if strings.TrimSpace(path) != "" {
		return path
	}
	p, err := config.UserConfigPath()
	if err != nil {
		return ""
	}
	return p
}

func (s *Server) copyPreapproved() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.cfg.PreapprovedOrigins) == 0 {
		return []string{}
	}
	out := make([]string, len(s.cfg.PreapprovedOrigins))
	copy(out, s.cfg.PreapprovedOrigins)
	return out
}

func (s *Server) addPreapprovedOrigin(raw string) (string, error) {
	origin, err := auth.NormalizeOrigin(raw)
	if err != nil {
		return "", card.NewError(card.ErrInvalidRequest, "invalid origin")
	}
	s.trustMu.Lock()
	defer s.trustMu.Unlock()
	next, err := config.AddPreapproved(s.copyPreapproved(), string(origin))
	if err != nil {
		return "", card.NewError(card.ErrInvalidRequest, err.Error())
	}
	path := s.configFilePath()
	if path == "" {
		return "", fmt.Errorf("config path unavailable")
	}
	if err := config.SavePreapprovedOrigins(path, next); err != nil {
		return "", err
	}
	s.mu.Lock()
	s.cfg.PreapprovedOrigins = next
	s.mu.Unlock()
	if err := s.auth.Preapprove(origin, s.cfg.Fields, time.Now()); err != nil {
		return "", err
	}
	s.recordAudit(audit.Event{
		Kind:   audit.KindTrustPreapprove,
		Origin: string(origin),
		Fields: fieldsToStrings(s.cfg.Fields),
	})
	return string(origin), nil
}

func (s *Server) removePreapprovedOrigin(raw string) (string, error) {
	origin, err := auth.NormalizeOrigin(raw)
	if err != nil {
		return "", card.NewError(card.ErrInvalidRequest, "invalid origin")
	}
	s.trustMu.Lock()
	defer s.trustMu.Unlock()
	next := config.RemovePreapproved(s.copyPreapproved(), string(origin))
	path := s.configFilePath()
	if path != "" {
		if err := config.SavePreapprovedOrigins(path, next); err != nil {
			return "", err
		}
	}
	s.mu.Lock()
	s.cfg.PreapprovedOrigins = next
	s.mu.Unlock()
	s.recordAudit(audit.Event{
		Kind:   audit.KindTrustUnpreapprove,
		Origin: string(origin),
	})
	return string(origin), nil
}

func (s *Server) handleManagerPreapprove(w http.ResponseWriter, r *http.Request) {
	if !s.requireManagerAPI(w, r) {
		return
	}
	var body struct {
		Origin string `json:"origin"`
	}
	if err := decodeManagerBody(r, &body); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	origin, err := s.addPreapprovedOrigin(body.Origin)
	if err != nil {
		s.writeTrustError(w, err)
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{
		"ok":     true,
		"origin": origin,
		"status": s.managerStatus(),
	})
}

func (s *Server) handleManagerUnpreapprove(w http.ResponseWriter, r *http.Request) {
	if !s.requireManagerAPI(w, r) {
		return
	}
	var body struct {
		Origin string `json:"origin"`
	}
	if err := decodeManagerBody(r, &body); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	origin, err := s.removePreapprovedOrigin(body.Origin)
	if err != nil {
		s.writeTrustError(w, err)
		return
	}
	n := s.auth.RevokeOrigin(auth.Origin(origin))
	s.writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"origin":  origin,
		"revoked": n,
		"status":  s.managerStatus(),
	})
}

func (s *Server) writeTrustError(w http.ResponseWriter, err error) {
	if ce, ok := err.(*card.Error); ok {
		status := http.StatusInternalServerError
		if ce.Code == card.ErrInvalidRequest {
			status = http.StatusBadRequest
		}
		s.writeJSON(w, status, map[string]any{"ok": false, "error": ce.Message, "code": ce.Code})
		return
	}
	s.writeJSON(w, http.StatusInternalServerError, map[string]any{
		"ok":    false,
		"error": "could not save trusted clinic websites",
		"code":  card.ErrConnectionFailed,
	})
}
