package qualify

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"
)

const Schema = "openthaiid.qualify.v1"

// Result is a check outcome. Operator free text is never stored.
type Result string

const (
	ResultPass       Result = "PASS"
	ResultFail       Result = "FAIL"
	ResultSkip       Result = "SKIP"
	ResultIncomplete Result = "INCOMPLETE"
)

// Report is the Friday hardware-qualification record.
// It must remain free of citizen ID, names, address text, and portrait bytes.
type Report struct {
	Schema      string         `json:"schema"`
	PII         string         `json:"pii"`
	Version     string         `json:"version"`
	StartedAt   string         `json:"startedAt"`
	FinishedAt  string         `json:"finishedAt,omitempty"`
	Host        HostInfo       `json:"host"`
	Mode        string         `json:"mode"` // hardware | synthetic-smoke
	Overall     Result         `json:"overall"`
	Checks      []Check        `json:"checks"`
	Diagnostics TechnicianCopy `json:"diagnostics"`
	Notes       []string       `json:"notes,omitempty"`
}

// HostInfo is machine metadata with no card data.
type HostInfo struct {
	GOOS   string `json:"goos"`
	GOARCH string `json:"goarch"`
	GoVer  string `json:"goVersion"`
}

// Check is one E-series (or M1) row.
type Check struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	Result     Result         `json:"result"`
	DurationMS int64          `json:"durationMs"`
	ErrorCode  string         `json:"errorCode,omitempty"`
	Metrics    map[string]any `json:"metrics,omitempty"`
	Detail     string         `json:"detail,omitempty"`
}

// TechnicianCopy is the paste-into-a-ticket block. Same privacy rule as
// the helper page "Copy diagnostics".
type TechnicianCopy struct {
	Version       string   `json:"version"`
	OS            string   `json:"os"`
	Backend       string   `json:"backend"`
	BackendStatus string   `json:"backendStatus,omitempty"`
	Practice      bool     `json:"practice"`
	ReaderCount   int      `json:"readerCount"`
	Readers       []string `json:"readers,omitempty"` // device names, not card data
	CardPresent   bool     `json:"cardPresent"`
	LastError     string   `json:"lastError,omitempty"`
}

// NewReport starts an empty report.
func NewReport(version, mode string) *Report {
	now := time.Now().UTC().Format(time.RFC3339)
	return &Report{
		Schema:    Schema,
		PII:       "omitted — citizen ID, names, address, portrait bytes are never stored",
		Version:   version,
		StartedAt: now,
		Host: HostInfo{
			GOOS:   runtime.GOOS,
			GOARCH: runtime.GOARCH,
			GoVer:  runtime.Version(),
		},
		Mode:    mode,
		Overall: ResultIncomplete,
		Checks:  []Check{},
		Diagnostics: TechnicianCopy{
			Version: version,
			OS:      runtime.GOOS + "/" + runtime.GOARCH,
		},
		Notes: []string{
			"Physical E1–E7 still require an ACR39U. Synthetic mode is a kit smoke, not hardware qualification.",
			"Do not paste ERP form values or screenshots of the card into tickets.",
		},
	}
}

// AddCheck appends a check and sanitizes Detail.
func (r *Report) AddCheck(c Check) {
	c.Detail = SanitizeLine(c.Detail)
	c.ErrorCode = SanitizeLine(c.ErrorCode)
	if c.Metrics != nil {
		c.Metrics = sanitizeMetrics(c.Metrics)
	}
	r.Checks = append(r.Checks, c)
}

func sanitizeMetrics(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		switch t := v.(type) {
		case string:
			out[k] = SanitizeLine(t)
		default:
			out[k] = v
		}
	}
	return out
}

// Finalize sets overall from checks and stamps FinishedAt.
func (r *Report) Finalize() {
	r.FinishedAt = time.Now().UTC().Format(time.RFC3339)
	anyFail := false
	anySkip := false
	anyPass := false
	for _, c := range r.Checks {
		switch c.Result {
		case ResultFail:
			anyFail = true
		case ResultSkip:
			anySkip = true
		case ResultPass:
			anyPass = true
		}
	}
	switch {
	case anyFail:
		r.Overall = ResultFail
	case anySkip && anyPass:
		r.Overall = ResultIncomplete
	case anyPass && !anySkip:
		r.Overall = ResultPass
	default:
		r.Overall = ResultIncomplete
	}
}

// MarshalSafe JSON-encodes the report and refuses to emit forbidden bytes.
func (r *Report) MarshalSafe() ([]byte, error) {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return nil, err
	}
	if hit := FirstForbidden(b); hit != "" {
		return nil, fmt.Errorf("qualify report blocked: output would contain personal or photo data")
	}
	return b, nil
}

// WriteFile writes the safe JSON report.
func (r *Report) WriteFile(path string) error {
	b, err := r.MarshalSafe()
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

// Text renders technician diagnostics copy (no personal fields).
func (r *Report) Text() string {
	var b strings.Builder
	fmt.Fprintf(&b, "OpenThaiID hardware qualification (%s)\n", r.Schema)
	fmt.Fprintf(&b, "pii: %s\n", r.PII)
	fmt.Fprintf(&b, "version: %s\n", r.Version)
	fmt.Fprintf(&b, "mode: %s\n", r.Mode)
	fmt.Fprintf(&b, "host: %s/%s %s\n", r.Host.GOOS, r.Host.GOARCH, r.Host.GoVer)
	fmt.Fprintf(&b, "started: %s\n", r.StartedAt)
	if r.FinishedAt != "" {
		fmt.Fprintf(&b, "finished: %s\n", r.FinishedAt)
	}
	fmt.Fprintf(&b, "overall: %s\n", r.Overall)
	fmt.Fprintf(&b, "\nchecks:\n")
	for _, c := range r.Checks {
		line := fmt.Sprintf("  %s %-8s %dms", c.ID, c.Result, c.DurationMS)
		if c.ErrorCode != "" {
			line += " code=" + c.ErrorCode
		}
		if c.Detail != "" {
			line += " " + c.Detail
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	d := r.Diagnostics
	fmt.Fprintf(&b, "\ntechnician diagnostics (no personal card fields)\n")
	fmt.Fprintf(&b, "  version: %s\n", d.Version)
	fmt.Fprintf(&b, "  os: %s\n", d.OS)
	fmt.Fprintf(&b, "  backend: %s %s\n", d.Backend, d.BackendStatus)
	fmt.Fprintf(&b, "  practiceReader: %v\n", d.Practice)
	fmt.Fprintf(&b, "  readers: %d\n", d.ReaderCount)
	for _, name := range d.Readers {
		fmt.Fprintf(&b, "    - %s\n", SanitizeLine(name))
	}
	fmt.Fprintf(&b, "  cardPresent: %v\n", d.CardPresent)
	if d.LastError != "" {
		fmt.Fprintf(&b, "  lastError: %s\n", SanitizeLine(d.LastError))
	}
	return b.String()
}
