package bridge

import (
	"context"
	"net/http"
	"time"

	"github.com/night-sornram/openthaiid/internal/update"
)

const (
	managerUpdateWindow = 20 * time.Second
	managerUpdateBurst  = 2
	managerUpdateWait   = 10 * time.Second
)

func (s *Server) handleManagerUpdateCheck(w http.ResponseWriter, r *http.Request) {
	if !s.requireManagerAPI(w, r) {
		return
	}
	current := s.runtimeVersion()
	if !s.tryBeginUpdateCheck() {
		s.writeJSON(w, http.StatusOK, update.Unavailable(current, update.ReasonRateLimited, update.ReleasesPage).Public())
		return
	}
	defer s.endUpdateCheck()

	ctx, cancel := context.WithTimeout(r.Context(), managerUpdateWait)
	defer cancel()

	fn := s.updateChecker()
	var (
		res *update.CheckResult
		err error
	)
	if fn != nil {
		res, err = fn(ctx, current)
	} else {
		res, err = update.Check(ctx, current, update.DefaultOptions())
	}
	if err != nil || res == nil {
		res = update.Unavailable(current, update.ReasonOffline, update.ReleasesPage)
	}
	view := res.Public()
	view.Current = current
	s.writeJSON(w, http.StatusOK, view)
}

func (s *Server) runtimeVersion() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.version != "" {
		return s.version
	}
	return "dev"
}

func (s *Server) updateChecker() UpdateCheckFunc {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.updateCheck
}

func (s *Server) tryBeginUpdateCheck() bool {
	s.updateMu.Lock()
	defer s.updateMu.Unlock()
	if s.updateBusy {
		return false
	}
	now := time.Now()
	cutoff := now.Add(-managerUpdateWindow)
	kept := s.updateStamp[:0]
	for _, ts := range s.updateStamp {
		if ts.After(cutoff) {
			kept = append(kept, ts)
		}
	}
	s.updateStamp = kept
	if len(s.updateStamp) >= managerUpdateBurst {
		return false
	}
	s.updateBusy = true
	s.updateStamp = append(s.updateStamp, now)
	return true
}

func (s *Server) endUpdateCheck() {
	s.updateMu.Lock()
	s.updateBusy = false
	s.updateMu.Unlock()
}
