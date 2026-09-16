package bridge

import (
	"net/http"
	"strings"

	"github.com/night-sornram/openthaiid/internal/desktop"
	"github.com/night-sornram/openthaiid/internal/pcsc"
)

func (s *Server) handleHelperSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.writeJSON(w, http.StatusOK, s.HelperSummary())
}

func (s *Server) handleHelperShutdown(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin != "" && !s.isManagerOrigin(origin) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	s.mu.Lock()
	fn := s.onShutdownRequest
	s.mu.Unlock()
	if fn != nil {
		go fn()
	}
}

// SetShutdownRequest is invoked by POST /v1/helper/shutdown (loopback).
// Browser origins other than the helper page are rejected.
func (s *Server) SetShutdownRequest(fn func()) {
	s.mu.Lock()
	s.onShutdownRequest = fn
	s.mu.Unlock()
}

// HelperSummary is clinic-staff status with no personal card fields.
func (s *Server) HelperSummary() desktop.Summary {
	return desktop.Summarize(s.staffLights())
}

func (s *Server) staffLights() desktop.Lights {
	backend, readers := s.deviceView()
	cardPresent := false
	readerName := ""
	for _, r := range readers {
		if r.Selected && r.Name != "" {
			readerName = r.Name
		}
		if r.Presence == string(pcsc.PresencePresent) {
			cardPresent = true
		}
	}
	if readerName == "" && len(readers) > 0 {
		readerName = readers[0].Name
	}
	return desktop.Lights{
		Helper:         true,
		Reader:         len(readers) > 0 && backend.Status == "available",
		Card:           cardPresent && backend.Status == "available",
		Practice:       s.synthetic || backend.Name == "synthetic",
		Backend:        backend.Status,
		ReaderName:     readerName,
		ReadyPage:      s.ManagerURL(),
		TechnicianPage: strings.TrimRight(s.ManagerURL(), "/") + "/#technician",
	}
}
