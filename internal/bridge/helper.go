package bridge

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/night-sornram/openthaiid/internal/desktop"
	"github.com/night-sornram/openthaiid/internal/doctor"
	"github.com/night-sornram/openthaiid/internal/pcsc"
)

func (s *Server) handleHelperSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.helperLoopbackAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	s.writeJSON(w, http.StatusOK, s.HelperSummary())
}

func (s *Server) handleHelperShutdown(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.helperLoopbackAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin != "" {
		// Browser request: CSRF so a same-origin-looking form cannot stop
		// the helper. openthaiid quit sends no Origin and still works.
		got := r.Header.Get(managerTokenHeader)
		if subtle.ConstantTimeCompare([]byte(got), []byte(s.managerToken)) != 1 {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
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
// Browser origins other than the helper page are rejected. A browser
// Origin also requires the manager CSRF header.
func (s *Server) SetShutdownRequest(fn func()) {
	s.mu.Lock()
	s.onShutdownRequest = fn
	s.mu.Unlock()
}

func (s *Server) handleHelperDoctor(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.helperLoopbackAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	snap := s.DoctorSnapshot()
	snap.Text = doctor.Format(snap)
	s.writeJSON(w, http.StatusOK, snap)
}

// DoctorSnapshot is the technician desk health view. No personal card fields,
// tokens, or TLS private-key paths. Does not call GitHub.
func (s *Server) DoctorSnapshot() doctor.Snapshot {
	probe := s.probeView()
	httpAddr := s.Addr()
	snap := doctor.Snapshot{
		Version:       probe.Version,
		Helper:        doctor.HelperRunning,
		HTTP:          httpAddr,
		TLS:           doctor.TLSNo,
		ReadyURL:      probe.ManagerURL,
		Backend:       probe.Backend.Name,
		BackendStatus: probe.Backend.Status,
		Practice:      probe.Synthetic,
		Probe: doctor.Probe{
			OK:      true,
			Status:  "ok",
			Backend: probe.Backend.Status,
		},
		Update: s.doctorUpdate(),
	}
	if probe.TLS {
		snap.TLS = doctor.TLSYes
		snap.TLSListen = s.TLSAddr()
	}
	if probe.Backend.Status == "available" {
		snap.PCSC = doctor.PCSCOK
	} else {
		snap.PCSC = doctor.PCSCFail
	}
	snap.Readers = make([]doctor.Reader, 0, len(probe.Readers))
	for _, r := range probe.Readers {
		snap.Readers = append(snap.Readers, doctor.Reader{
			Name:     r.Name,
			Presence: r.Presence,
			Selected: r.Selected,
		})
	}
	return doctor.Finalize(snap)
}

func (s *Server) doctorUpdate() doctor.Update {
	last := s.copyLastUpdate()
	if last == nil {
		return doctor.Update{State: doctor.UpdateNotChecked, Soft: true}
	}
	return doctor.Update{
		State:  last.State,
		Reason: last.Reason,
		Tag:    last.Tag,
		Soft:   true,
	}
}

// HelperSummary is clinic-staff status with no personal card fields.
func (s *Server) HelperSummary() desktop.Summary {
	return desktop.Summarize(s.staffLights())
}

func (s *Server) staffLights() desktop.Lights {
	backend, readers := s.deviceView()
	selected := selectedManagerReader(readers)
	needSelection := len(readers) > 1 && selected == nil
	readerName := ""
	cardPresent := false
	if selected != nil {
		readerName = selected.Name
		cardPresent = selected.Presence == string(pcsc.PresencePresent)
	}
	return desktop.Lights{
		Helper:         true,
		Reader:         len(readers) > 0 && backend.Status == "available",
		Card:           cardPresent && backend.Status == "available" && !needSelection,
		NeedSelection:  needSelection,
		Practice:       s.synthetic || backend.Name == "synthetic",
		Backend:        backend.Status,
		ReaderName:     readerName,
		ReadyPage:      s.ManagerURL(),
		TechnicianPage: strings.TrimRight(s.ManagerURL(), "/") + "/#technician",
	}
}
