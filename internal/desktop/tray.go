package desktop

import (
	"context"
	"errors"
	"log/slog"
)

// ErrNoTray is returned when this session has no notification-area host
// (typical on Linux without a desktop, or Darwin without cgo). The helper
// should keep running in the background.
var ErrNoTray = errors.New("desktop tray unavailable")

// Host is the tray menu. Callbacks must not include personal card fields.
type Host struct {
	ReadyURL      string
	TechnicianURL string
	Status        func() string
	OpenURL       func(url string) error
	AutostartOn   func() bool
	SetAutostart  func(enabled bool) error
	Quit          func()
	Log           *slog.Logger
}

// RunTray shows a notification-area icon when the OS supports it and blocks
// until ctx is cancelled or the user chooses Quit. On platforms with no tray,
// it returns ErrNoTray without blocking.
func RunTray(ctx context.Context, host Host) error {
	if host.Log == nil {
		host.Log = slog.Default()
	}
	if host.ReadyURL == "" {
		host.ReadyURL = "http://127.0.0.1:17891/"
	}
	if host.TechnicianURL == "" {
		host.TechnicianURL = host.ReadyURL + "#technician"
	}
	if host.Status == nil {
		host.Status = func() string { return "OpenThaiID" }
	}
	if host.OpenURL == nil {
		host.OpenURL = func(string) error { return nil }
	}
	if host.AutostartOn == nil {
		host.AutostartOn = func() bool { return false }
	}
	if host.SetAutostart == nil {
		host.SetAutostart = func(bool) error { return nil }
	}
	if host.Quit == nil {
		host.Quit = func() {}
	}
	return runNativeTray(ctx, host)
}

func (h Host) openReady() {
	if err := h.OpenURL(h.ReadyURL); err != nil && h.Log != nil {
		h.Log.Warn("open ready page", "error", err)
	}
}

func (h Host) openTechnician() {
	if err := h.OpenURL(h.TechnicianURL); err != nil && h.Log != nil {
		h.Log.Warn("open technician page", "error", err)
	}
}

func (h Host) toggleAutostart() {
	on := h.AutostartOn()
	if err := h.SetAutostart(!on); err != nil && h.Log != nil {
		h.Log.Warn("autostart", "error", err)
	}
}
