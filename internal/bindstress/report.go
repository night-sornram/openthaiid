// Package bindstress is the pre-hardware PC/SC / ebfe/scard binding
// stress harness. It repeatedly exercises context connect/disconnect,
// reader list, presence probe, and (on synthetic) card present/absent.
// Reports never include citizen ID, names, address, portrait bytes, or
// raw APDUs.
package bindstress

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/night-sornram/openthaiid/internal/qualify"
)

const Schema = "openthaiid.bindstress.v1"

// Result is a check outcome. Same vocabulary as the qualify kit.
type Result = qualify.Result

const (
	ResultPass       = qualify.ResultPass
	ResultFail       = qualify.ResultFail
	ResultSkip       = qualify.ResultSkip
	ResultIncomplete = qualify.ResultIncomplete
)

// Report is the binding-audit stress record.
type Report struct {
	Schema     string   `json:"schema"`
	PII        string   `json:"pii"`
	Version    string   `json:"version"`
	StartedAt  string   `json:"startedAt"`
	FinishedAt string   `json:"finishedAt,omitempty"`
	Host       HostInfo `json:"host"`
	Mode       string   `json:"mode"` // hardware | synthetic-smoke
	Overall    Result   `json:"overall"`
	Cycles     int      `json:"cycles"`
	Checks     []Check  `json:"checks"`
	Evidence   Evidence `json:"evidence"`
	Notes      []string `json:"notes,omitempty"`
}

// HostInfo is machine metadata with no card data.
type HostInfo struct {
	GOOS   string `json:"goos"`
	GOARCH string `json:"goarch"`
	GoVer  string `json:"goVersion"`
}

// Check is one harness row (context, list, probe, presence, …).
type Check struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	Result     Result         `json:"result"`
	DurationMS int64          `json:"durationMs"`
	ErrorCode  string         `json:"errorCode,omitempty"`
	Metrics    map[string]any `json:"metrics,omitempty"`
	Detail     string         `json:"detail,omitempty"`
}

// Evidence records which binding-audit IDs this run can speak to.
// Synthetic never closes P2–P10. Hardware on this GOOS/GOARCH can
// evidence a subset; other hosts still need their own run.
type Evidence struct {
	ThisHostMayEvidence []string `json:"thisHostMayEvidence,omitempty"`
	StillNeedsOtherHost []string `json:"stillNeedsOtherHost,omitempty"`
	SyntheticDoesNot    []string `json:"syntheticDoesNotClose,omitempty"`
	NeedsACR39U         bool     `json:"needsACR39U"`
}

// NewReport starts an empty redacted report.
func NewReport(version, mode string, cycles int) *Report {
	now := time.Now().UTC().Format(time.RFC3339)
	ev := HostEvidence(runtime.GOOS, runtime.GOARCH, mode != "hardware")
	notes := []string{
		"Never paste citizen ID, names, address, portrait bytes, raw APDUs, or card.read payloads.",
		"Synthetic mode proves the harness, not ebfe/scard native layouts.",
		"When the ACR39U arrives: openthaiid bind-stress -report ./openthaiid-bind-stress.json (no -synthetic).",
	}
	if mode != "hardware" {
		notes = append(notes, "This synthetic run does not close P2–P10.")
	} else {
		notes = append(notes, "Hardware run: attach this JSON. A pass on this OS/arch does not close other hosts.")
	}
	return &Report{
		Schema:    Schema,
		PII:       "omitted — citizen ID, names, address, portrait bytes, and raw APDUs are never stored",
		Version:   version,
		StartedAt: now,
		Host: HostInfo{
			GOOS:   runtime.GOOS,
			GOARCH: runtime.GOARCH,
			GoVer:  runtime.Version(),
		},
		Mode:     mode,
		Overall:  ResultIncomplete,
		Cycles:   cycles,
		Checks:   []Check{},
		Evidence: ev,
		Notes:    notes,
	}
}

// AddCheck appends a check and sanitizes strings.
func (r *Report) AddCheck(c Check) {
	c.Detail = qualify.SanitizeLine(c.Detail)
	c.ErrorCode = qualify.SanitizeLine(c.ErrorCode)
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
			out[k] = qualify.SanitizeLine(t)
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

// MarshalSafe JSON-encodes the report and refuses forbidden bytes.
func (r *Report) MarshalSafe() ([]byte, error) {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return nil, err
	}
	if hit := qualify.FirstForbidden(b); hit != "" {
		return nil, fmt.Errorf("bind-stress report blocked: output would contain personal or photo data")
	}
	if hit := firstAPDUDump(b); hit != "" {
		return nil, fmt.Errorf("bind-stress report blocked: output would contain a raw APDU dump")
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

// Text renders technician copy (no personal fields, no APDUs).
func (r *Report) Text() string {
	var b strings.Builder
	fmt.Fprintf(&b, "OpenThaiID PC/SC bind-stress (%s)\n", r.Schema)
	fmt.Fprintf(&b, "pii: %s\n", r.PII)
	fmt.Fprintf(&b, "version: %s\n", r.Version)
	fmt.Fprintf(&b, "mode: %s\n", r.Mode)
	fmt.Fprintf(&b, "host: %s/%s %s\n", r.Host.GOOS, r.Host.GOARCH, r.Host.GoVer)
	fmt.Fprintf(&b, "cycles: %d\n", r.Cycles)
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
	fmt.Fprintf(&b, "\nevidence:\n")
	if len(r.Evidence.SyntheticDoesNot) > 0 {
		fmt.Fprintf(&b, "  syntheticDoesNotClose: %s\n", strings.Join(r.Evidence.SyntheticDoesNot, ", "))
	}
	if len(r.Evidence.ThisHostMayEvidence) > 0 {
		fmt.Fprintf(&b, "  thisHostMayEvidence: %s\n", strings.Join(r.Evidence.ThisHostMayEvidence, ", "))
	}
	if len(r.Evidence.StillNeedsOtherHost) > 0 {
		fmt.Fprintf(&b, "  stillNeedsOtherHost: %s\n", strings.Join(r.Evidence.StillNeedsOtherHost, ", "))
	}
	fmt.Fprintf(&b, "  needsACR39U: %v\n", r.Evidence.NeedsACR39U)
	return b.String()
}

func checkResult(r *Report, id string) Result {
	for _, c := range r.Checks {
		if c.ID == id {
			return c.Result
		}
	}
	return ResultSkip
}
