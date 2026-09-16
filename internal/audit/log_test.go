package audit

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/night-sornram/openthaiid/internal/qualify"
)

func TestRecordRingAndExport(t *testing.T) {
	l := New(3)
	l.Record(Event{Kind: KindHelperStart, Mode: "serve"})
	l.Record(Event{Kind: KindReaderConnected, Reader: "SYNTHETIC ACR39U (fictional demo)"})
	l.Record(Event{Kind: KindCardPresent, Presence: "present"})
	l.Record(Event{Kind: KindFillFailure, Code: "NO_CARD"})
	got := l.Snapshot()
	if len(got) != 3 {
		t.Fatalf("cap 3, got %d: %+v", len(got), got)
	}
	if got[0].Kind != KindReaderConnected || got[2].Kind != KindFillFailure {
		t.Fatalf("ring %+v", got)
	}
	if got[2].Code != "NO_CARD" {
		t.Fatalf("code %q", got[2].Code)
	}
	exp := l.ExportJSON()
	if exp.Schema != Schema || exp.Count != 3 || !strings.Contains(exp.PII, "never stored") {
		t.Fatalf("export %+v", exp)
	}
	raw, err := json.Marshal(exp)
	if err != nil {
		t.Fatal(err)
	}
	if qualify.ContainsForbidden(string(raw)) {
		t.Fatalf("export leaked forbidden: %s", raw)
	}
}

func TestSanitizeDropsPIIAndUnknownKinds(t *testing.T) {
	l := New(32)
	l.Record(Event{Kind: "citizen_name", Origin: "https://clinic.example"})
	l.Record(Event{
		Kind:   KindApprovalAllow,
		Origin: "https://clinic.example/1101701234567",
		Fields: []string{"citizenId", "สมชาย"},
	})
	l.Record(Event{Kind: KindFillFailure, Code: "NO_CARD: สมชาย 1101701234567"})
	l.Record(Event{Kind: KindFillFailure}) // missing code
	l.Record(Event{
		Kind:   KindApprovalAllow,
		Origin: "https://erp.clinic.example",
		Fields: []string{"citizenId", "thaiName", "notAField", "citizenId"},
	})
	got := l.Snapshot()
	if len(got) != 1 {
		t.Fatalf("expected 1 safe event, got %+v", got)
	}
	if got[0].Origin != "https://erp.clinic.example" {
		t.Fatalf("origin %q", got[0].Origin)
	}
	if strings.Join(got[0].Fields, ",") != "citizenId,thaiName" {
		t.Fatalf("fields %v", got[0].Fields)
	}
	raw, _ := json.Marshal(got)
	for _, s := range []string{"สมชาย", "1101701234567", "citizen_name"} {
		if bytes.Contains(raw, []byte(s)) {
			t.Fatalf("leaked %q: %s", s, raw)
		}
	}
}

func TestFillFailureCodeOnly(t *testing.T) {
	l := New(8)
	l.Record(Event{Kind: KindFillFailure, Code: "CARD_REMOVED", Origin: "https://erp.example"})
	got := l.Snapshot()
	if len(got) != 1 || got[0].Code != "CARD_REMOVED" {
		t.Fatalf("%+v", got)
	}
	raw, _ := json.Marshal(got[0])
	if bytes.Contains(raw, []byte("message")) {
		t.Fatalf("fill failure must not carry a message: %s", raw)
	}
}

func TestJSONLRoundTripAndForbiddenLinesDropped(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.jsonl")
	poison := []byte(`{"at":"2026-01-01T00:00:00Z","kind":"approval.allow","origin":"https://x.example","fields":["citizenId"]}
{"at":"2026-01-01T00:00:01Z","kind":"fill.failure","code":"NO_CARD","origin":"https://x.example","note":"สมชาย 1234567890121"}
{"at":"2026-01-01T00:00:02Z","kind":"helper.stop"}
`)
	if err := os.WriteFile(path, poison, 0o600); err != nil {
		t.Fatal(err)
	}
	l, err := Open(path, 16)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	got := l.Snapshot()
	if len(got) != 2 {
		t.Fatalf("loaded %+v", got)
	}
	if got[0].Kind != KindApprovalAllow || got[1].Kind != KindHelperStop {
		t.Fatalf("kinds %+v", got)
	}
	l.Record(Event{Kind: KindFillFailure, Code: "NO_READER"})
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if qualify.ContainsForbidden(string(raw)) {
		t.Fatalf("file leaked forbidden: %s", raw)
	}
	if !bytes.Contains(raw, []byte(`"kind":"fill.failure"`)) {
		t.Fatalf("missing append: %s", raw)
	}
}

func TestSanitizeModePresenceReader(t *testing.T) {
	ev, ok := sanitize(Event{
		Kind:     KindHelperStart,
		Mode:     "serve",
		Presence: "present",
		Reader:   "ACS ACR39U ICC Reader 0",
		ReaderID: "synthetic-acr39u",
	})
	if !ok || ev.Mode != "serve" || ev.Reader == "" || ev.ReaderID != "synthetic-acr39u" {
		t.Fatalf("%v %+v", ok, ev)
	}
	ev, ok = sanitize(Event{Kind: KindReaderSelected, ReaderID: "r2", Reader: "Reader Two"})
	if !ok || ev.Kind != KindReaderSelected || ev.ReaderID != "r2" {
		t.Fatalf("reader.selected: %v %+v", ok, ev)
	}
	ev, ok = sanitize(Event{Kind: KindHelperStart, Mode: "rootkit"})
	if !ok || ev.Mode != "" {
		t.Fatalf("bad mode kept: %v %+v", ok, ev)
	}
}
