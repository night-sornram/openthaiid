// Package doctor builds a redacted technician health snapshot for
// `openthaiid doctor` and GET /v1/helper/doctor.
//
// The snapshot never includes citizen ID, names, dates, address, portrait
// bytes, session tokens, GitHub tokens, or TLS private-key paths.
package doctor

import (
	"context"
	"strings"

	"github.com/night-sornram/openthaiid/internal/config"
	"github.com/night-sornram/openthaiid/internal/desktop"
	"github.com/night-sornram/openthaiid/internal/qualify"
)

const (
	HelperRunning = "running"
	HelperDown    = "down"

	PCSCOK   = "OK"
	PCSCFail = "FAIL"

	TLSYes = "yes"
	TLSNo  = "no"

	FailHelperDown = "HELPER_DOWN"
	FailPCSC       = "PCSC_UNAVAILABLE"

	UpdateNotChecked  = "not_checked"
	UpdateUpToDate    = "up_to_date"
	UpdateNewer       = "newer"
	UpdateNone        = "none"
	UpdateUnavailable = "unavailable"
)

// Reader is a PC/SC device row. Name is an OS reader label, never card data.
type Reader struct {
	Name     string `json:"name"`
	Presence string `json:"presence,omitempty"`
	Selected bool   `json:"selected"`
}

// Probe is the bridge.probe view: backend + readers, no card fields.
type Probe struct {
	OK      bool   `json:"ok"`
	Status  string `json:"status"`
	Backend string `json:"backend,omitempty"`
}

// Update is the last GitHub Releases check, or a skipped offline-safe row.
// Failures are always soft: they never set HardFail.
type Update struct {
	State  string `json:"state"`
	Reason string `json:"reason,omitempty"`
	Tag    string `json:"tag,omitempty"`
	Soft   bool   `json:"soft"`
}

// Snapshot is the desk health view printed by `openthaiid doctor`.
type Snapshot struct {
	Version        string   `json:"version"`
	Helper         string   `json:"helper"`
	HTTP           string   `json:"http"`
	TLS            string   `json:"tls"`
	TLSListen      string   `json:"tlsListen,omitempty"`
	ReadyURL       string   `json:"readyUrl"`
	PCSC           string   `json:"pcsc"`
	Backend        string   `json:"backend,omitempty"`
	BackendStatus  string   `json:"backendStatus,omitempty"`
	Readers        []Reader `json:"readers"`
	Probe          Probe    `json:"probe"`
	Update         Update   `json:"update"`
	NextTH         string   `json:"nextTh"`
	NextEN         string   `json:"nextEn"`
	HardFail       bool     `json:"hardFail"`
	FailCode       string   `json:"failCode,omitempty"`
	Practice       bool     `json:"practice,omitempty"`
	Hint           string   `json:"hint,omitempty"`
	PersonalFields string   `json:"personalFields"`
	// Text is the formatted snapshot for Technician-panel copy. Format()
	// does not print this field.
	Text string `json:"text,omitempty"`
}

// Inventory is an independent PC/SC list used only when the helper is down.
type Inventory struct {
	Backend string
	Readers []Reader
}

// HelperFetch returns the running helper's snapshot, or an error if it is down.
type HelperFetch func(ctx context.Context) (*Snapshot, error)

// PCSCList lists readers without opening a card or reading fields.
type PCSCList func(ctx context.Context) (Inventory, error)

// Options wires CLI collection. Tests inject fakes; production uses HTTP + OS PC/SC.
type Options struct {
	Version string
	Config  config.Config
	Helper  HelperFetch
	PCSC    PCSCList
}

// Collect builds a redacted snapshot. It talks to the helper when possible so
// a running helper is not given a second PC/SC context. Independent PC/SC is
// used only when the helper is down. Update-check rows stay soft.
func Collect(ctx context.Context, opts Options) Snapshot {
	if opts.Helper != nil {
		got, err := opts.Helper(ctx)
		if err == nil && got != nil && strings.TrimSpace(got.Helper) == HelperRunning {
			return Finalize(*got)
		}
	}
	return Finalize(downSnapshot(ctx, opts))
}

func downSnapshot(ctx context.Context, opts Options) Snapshot {
	cfg := opts.Config
	if cfg.Port == 0 {
		cfg = config.Default()
	}
	s := Snapshot{
		Version: strings.TrimSpace(opts.Version),
		Helper:  HelperDown,
		HTTP:    cfg.Addr(),
		TLS:     TLSNo,
		ReadyURL: func() string {
			u := strings.TrimSpace(cfg.ManagerURL())
			if u == "" {
				return "http://127.0.0.1:17891/"
			}
			return u
		}(),
		PCSC: PCSCFail,
		Probe: Probe{
			OK:     false,
			Status: HelperDown,
		},
		Update: Update{
			State: UpdateNotChecked,
			Soft:  true,
		},
		HardFail: true,
		FailCode: FailHelperDown,
	}
	if cfg.TLSEnabled() {
		s.TLS = TLSYes
		s.TLSListen = cfg.TLSAddr()
	}
	if opts.PCSC != nil {
		inv, err := opts.PCSC(ctx)
		if err != nil {
			s.PCSC = PCSCFail
			s.BackendStatus = "unavailable"
			s.Hint = pcscHint()
		} else {
			s.PCSC = PCSCOK
			s.Backend = inv.Backend
			s.BackendStatus = "available"
			s.Readers = inv.Readers
		}
	}
	return s
}

// Finalize fills next-step copy, hard-fail flags, and redacts unsafe strings.
func Finalize(s Snapshot) Snapshot {
	s.PersonalFields = "not read"
	if strings.TrimSpace(s.Helper) != HelperRunning {
		s.Helper = HelperDown
		s.HardFail = true
		s.FailCode = FailHelperDown
		if s.Probe.Status == "" {
			s.Probe.Status = HelperDown
			s.Probe.OK = false
		}
	} else {
		s.Helper = HelperRunning
		if s.BackendStatus == "unavailable" || s.BackendStatus == "stalled" || s.PCSC == PCSCFail {
			s.PCSC = PCSCFail
			s.HardFail = true
			s.FailCode = FailPCSC
		} else {
			s.PCSC = PCSCOK
			s.HardFail = false
			s.FailCode = ""
		}
	}
	if s.Update.State == "" {
		s.Update.State = UpdateNotChecked
	}
	s.Update.Soft = true
	s.NextTH, s.NextEN = nextSteps(s)
	return redactSnapshot(s)
}

// ExitCode is 1 only for helper-down or PC/SC hard fail. Soft update rows
// never change the code.
func ExitCode(s Snapshot) int {
	if s.HardFail {
		return 1
	}
	return 0
}

func nextSteps(s Snapshot) (th, en string) {
	thCopy := desktop.StaffCopy(desktop.LangTH)
	enCopy := desktop.StaffCopy(desktop.LangEN)
	if s.Helper != HelperRunning {
		return thCopy.StartHelper, enCopy.StartHelper
	}
	switch s.BackendStatus {
	case "unavailable", "stalled":
		return thCopy.FixService, enCopy.FixService
	}
	if len(s.Readers) == 0 {
		return thCopy.PlugReader, enCopy.PlugReader
	}
	selected := 0
	var sel Reader
	for _, r := range s.Readers {
		if r.Selected {
			selected++
			sel = r
		}
	}
	if len(s.Readers) > 1 && selected != 1 {
		return thCopy.ChooseReader, enCopy.ChooseReader
	}
	present := sel.Presence == "present"
	if selected == 0 && len(s.Readers) == 1 {
		present = s.Readers[0].Presence == "present"
	}
	if !present {
		return thCopy.InsertThenFill, enCopy.InsertThenFill
	}
	return thCopy.ClickFill, enCopy.ClickFill
}

func pcscHint() string {
	return "Linux: start pcscd. Windows: start the Smart Card service. macOS: built-in PC/SC."
}

func redactSnapshot(s Snapshot) Snapshot {
	s.Version = sanitizeLine(s.Version)
	s.HTTP = sanitizeLine(s.HTTP)
	s.TLSListen = sanitizeListen(s.TLSListen)
	s.ReadyURL = sanitizeLine(s.ReadyURL)
	s.Backend = sanitizeLine(s.Backend)
	s.BackendStatus = sanitizeLine(s.BackendStatus)
	s.Hint = sanitizeLine(s.Hint)
	s.NextTH = sanitizeLine(s.NextTH)
	s.NextEN = sanitizeLine(s.NextEN)
	s.FailCode = sanitizeLine(s.FailCode)
	s.Probe.Status = sanitizeLine(s.Probe.Status)
	s.Probe.Backend = sanitizeLine(s.Probe.Backend)
	s.Update.State = sanitizeLine(s.Update.State)
	s.Update.Reason = sanitizeLine(s.Update.Reason)
	s.Update.Tag = sanitizeLine(s.Update.Tag)
	out := make([]Reader, 0, len(s.Readers))
	for _, r := range s.Readers {
		out = append(out, Reader{
			Name:     sanitizeLine(r.Name),
			Presence: sanitizeLine(r.Presence),
			Selected: r.Selected,
		})
	}
	s.Readers = out
	s.PersonalFields = "not read"
	return s
}

func sanitizeListen(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if looksLikeSecret(s) || qualify.ContainsForbidden(s) {
		return "[redacted]"
	}
	return s
}

func sanitizeLine(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if looksLikeSecret(s) || qualify.ContainsForbidden(s) {
		return "[redacted]"
	}
	return s
}

func looksLikeSecret(s string) bool {
	lower := strings.ToLower(s)
	for _, n := range []string{
		"begin private key",
		"begin rsa private key",
		"begin ec private key",
		"begin openssh private key",
		"github_pat_",
		"ghp_",
		"gho_",
		"ghu_",
		"ghs_",
		"openthaiid_github_token",
		"tlskeyfile",
		"tls-key",
		"private-key",
		"-----begin",
		"tok_",
		"cs_secret",
	} {
		if strings.Contains(lower, n) {
			return true
		}
	}
	if strings.Contains(lower, ".pem") && strings.Contains(lower, "key") {
		return true
	}
	if strings.Contains(lower, ".key") && (strings.Contains(lower, "/") || strings.Contains(lower, "\\")) {
		return true
	}
	return false
}

// ContainsForbidden reports whether formatted output would leak a sentinel.
func ContainsForbidden(s string) bool {
	return qualify.ContainsForbidden(s) || looksLikeSecret(s)
}
