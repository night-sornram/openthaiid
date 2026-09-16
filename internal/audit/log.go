// Package audit is a redacted operator trail for technicians.
//
// It records helper start/stop, reader and card presence, approval
// allow/deny/revoke, and Fill failures by protocol error code only.
// Citizen ID, names, address, portrait bytes, and raw APDUs are never stored.
package audit

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/night-sornram/openthaiid/internal/card"
	"github.com/night-sornram/openthaiid/internal/qualify"
)

const (
	Schema     = "openthaiid.audit.v1"
	DefaultCap = 256
	maxFile    = 256 << 10 // 256 KiB, then rotate to .old
	piiNote    = "omitted — citizen ID, names, address, portrait, raw APDUs are never stored"
)

// Kinds that may appear in the trail. Anything else is dropped.
const (
	KindHelperStart        = "helper.start"
	KindHelperStop         = "helper.stop"
	KindReaderConnected    = "reader.connected"
	KindReaderDisconnected = "reader.disconnected"
	KindCardPresent        = "card.present"
	KindCardAbsent         = "card.absent"
	KindCardUnsupported    = "card.unsupported"
	KindApprovalAllow      = "approval.allow"
	KindApprovalDeny       = "approval.deny"
	KindApprovalRevoke     = "approval.revoke"
	KindFillFailure        = "fill.failure"
)

// Event is one redacted operator event. Field tokens (citizenId, thaiName, …)
// are names only — never values.
type Event struct {
	At       string   `json:"at"`
	Kind     string   `json:"kind"`
	Code     string   `json:"code,omitempty"`
	Origin   string   `json:"origin,omitempty"`
	Fields   []string `json:"fields,omitempty"`
	Reader   string   `json:"reader,omitempty"`
	Presence string   `json:"presence,omitempty"`
	Mode     string   `json:"mode,omitempty"`
}

// Export is the JSON technicians copy or download.
type Export struct {
	Schema string  `json:"schema"`
	PII    string  `json:"pii"`
	Count  int     `json:"count"`
	Events []Event `json:"events"`
}

// Log is a bounded in-memory ring, optionally mirrored to a JSONL file.
type Log struct {
	mu     sync.Mutex
	cap    int
	events []Event
	file   *os.File
	path   string
}

// New returns a memory-only ring of at most cap events (oldest dropped).
func New(cap int) *Log {
	if cap < 1 {
		cap = DefaultCap
	}
	return &Log{cap: cap, events: make([]Event, 0, cap)}
}

// Open loads existing JSONL (if any) into a ring and appends new events.
// A nil error with a usable log is always preferred: file failures fall back
// to memory-only so the helper still starts.
func Open(path string, cap int) (*Log, error) {
	l := New(cap)
	if strings.TrimSpace(path) == "" {
		return l, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return l, err
	}
	rewrite := false
	if raw, err := os.ReadFile(path); err == nil {
		kept, dropped := loadLines(raw, l.cap)
		l.events = kept
		if dropped > 0 {
			rewrite = true
		}
		if st, err := os.Stat(path); err == nil && st.Size() > maxFile {
			if dropped == 0 {
				_ = os.Rename(path, path+".old")
			}
			rewrite = true
		}
	}
	if rewrite {
		if err := writeJSONL(path, l.events); err != nil {
			return l, err
		}
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return l, err
	}
	l.file = f
	l.path = path
	return l, nil
}

// Path is the JSONL file, or empty when memory-only.
func (l *Log) Path() string {
	if l == nil {
		return ""
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.path
}

// Close flushes the JSONL file if open.
func (l *Log) Close() error {
	if l == nil {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file == nil {
		return nil
	}
	err := l.file.Close()
	l.file = nil
	return err
}

// Record appends a sanitized event. Events that cannot be made safe are dropped.
func (l *Log) Record(ev Event) {
	if l == nil {
		return
	}
	if ev.At == "" {
		ev.At = time.Now().UTC().Format(time.RFC3339)
	}
	ev, ok := sanitize(ev)
	if !ok {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.events = append(l.events, ev)
	if len(l.events) > l.cap {
		l.events = append([]Event(nil), l.events[len(l.events)-l.cap:]...)
	}
	if l.file != nil {
		raw, err := json.Marshal(ev)
		if err != nil || qualify.ContainsForbidden(string(raw)) {
			return
		}
		_, _ = l.file.Write(append(raw, '\n'))
	}
}

// Snapshot returns a copy, oldest first.
func (l *Log) Snapshot() []Event {
	if l == nil {
		return []Event{}
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.events) == 0 {
		return []Event{}
	}
	out := make([]Event, len(l.events))
	copy(out, l.events)
	return out
}

// ExportJSON is the technician copy/download payload.
func (l *Log) ExportJSON() Export {
	ev := l.Snapshot()
	return Export{
		Schema: Schema,
		PII:    piiNote,
		Count:  len(ev),
		Events: ev,
	}
}

func loadLines(raw []byte, cap int) (events []Event, dropped int) {
	events = make([]Event, 0, cap)
	for _, line := range bytes.Split(raw, []byte("\n")) {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		if qualify.ContainsForbidden(string(line)) {
			dropped++
			continue
		}
		var ev Event
		if err := json.Unmarshal(line, &ev); err != nil {
			dropped++
			continue
		}
		ev, ok := sanitize(ev)
		if !ok {
			dropped++
			continue
		}
		events = append(events, ev)
	}
	if cap > 0 && len(events) > cap {
		dropped += len(events) - cap
		events = append([]Event(nil), events[len(events)-cap:]...)
	}
	return events, dropped
}

func writeJSONL(path string, events []Event) error {
	var b bytes.Buffer
	for _, ev := range events {
		raw, err := json.Marshal(ev)
		if err != nil || qualify.ContainsForbidden(string(raw)) {
			continue
		}
		b.Write(raw)
		b.WriteByte('\n')
	}
	return os.WriteFile(path, b.Bytes(), 0o600)
}

// FileName is the JSONL file stored under the user directory.
func FileName() string { return "audit.jsonl" }

func sanitize(ev Event) (Event, bool) {
	if !allowedKind(ev.Kind) {
		return Event{}, false
	}
	if blob, _ := json.Marshal(ev); qualify.ContainsForbidden(string(blob)) {
		// Drop rather than try to salvage a poisoned event.
		return Event{}, false
	}
	ev.Code = sanitizeCode(ev.Code)
	ev.Origin = sanitizeOrigin(ev.Origin)
	ev.Reader = sanitizeReader(ev.Reader)
	ev.Presence = sanitizePresence(ev.Presence)
	ev.Mode = sanitizeMode(ev.Mode)
	ev.Fields = sanitizeFields(ev.Fields)
	if ev.Kind == KindFillFailure && ev.Code == "" {
		return Event{}, false
	}
	return ev, true
}

func allowedKind(k string) bool {
	switch k {
	case KindHelperStart, KindHelperStop,
		KindReaderConnected, KindReaderDisconnected,
		KindCardPresent, KindCardAbsent, KindCardUnsupported,
		KindApprovalAllow, KindApprovalDeny, KindApprovalRevoke,
		KindFillFailure:
		return true
	default:
		return false
	}
}

func sanitizeCode(code string) string {
	code = strings.TrimSpace(code)
	if code == "" {
		return ""
	}
	if len(code) > 64 {
		return ""
	}
	for _, r := range code {
		if r >= 'A' && r <= 'Z' {
			continue
		}
		if r >= '0' && r <= '9' {
			continue
		}
		if r == '_' {
			continue
		}
		return ""
	}
	if qualify.ContainsForbidden(code) {
		return ""
	}
	return code
}

func sanitizeOrigin(origin string) string {
	origin = strings.TrimSpace(origin)
	if origin == "" {
		return ""
	}
	if origin == "*" {
		return "*"
	}
	if qualify.ContainsForbidden(origin) {
		return ""
	}
	lower := strings.ToLower(origin)
	if !strings.HasPrefix(lower, "https://") && !strings.HasPrefix(lower, "http://") {
		return ""
	}
	if strings.ContainsAny(origin, " \t\r\n") {
		return ""
	}
	if len(origin) > 256 {
		return origin[:256]
	}
	return origin
}

func sanitizeReader(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	if qualify.ContainsForbidden(name) {
		return ""
	}
	var b strings.Builder
	for _, r := range name {
		if r < 0x20 || r == 0x7f {
			continue
		}
		b.WriteRune(r)
	}
	out := b.String()
	if len(out) > 128 {
		out = out[:128]
	}
	return out
}

func sanitizePresence(p string) string {
	switch p {
	case "present", "absent", "unsupported":
		return p
	default:
		return ""
	}
}

func sanitizeMode(m string) string {
	switch m {
	case "console", "tray", "serve":
		return m
	default:
		return ""
	}
}

func sanitizeFields(fields []string) []string {
	if len(fields) == 0 {
		return nil
	}
	out := make([]string, 0, len(fields))
	seen := map[string]struct{}{}
	for _, f := range fields {
		f = strings.TrimSpace(f)
		if !card.Supported(card.Field(f)) {
			continue
		}
		if _, ok := seen[f]; ok {
			continue
		}
		seen[f] = struct{}{}
		out = append(out, f)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
