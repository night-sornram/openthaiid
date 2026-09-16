package desktop

import (
	"strings"
	"testing"
)

func TestSummarizeNotRunning(t *testing.T) {
	s := Summarize(Lights{})
	if s.Helper || s.Reader || s.Card {
		t.Fatalf("lights %+v", s)
	}
	if !strings.Contains(s.HelperText, "not running") {
		t.Fatalf("helper text %q", s.HelperText)
	}
	out := Format(s)
	assertNoPII(t, out)
}

func TestSummarizeReady(t *testing.T) {
	s := Summarize(Lights{
		Helper:     true,
		Reader:     true,
		Card:       true,
		Backend:    "available",
		ReaderName: "SYNTHETIC ACR39U (fictional demo)",
		ReadyPage:  "http://127.0.0.1:17991/",
	})
	if !s.Helper || !s.Reader || !s.Card || s.Headline != "Ready" {
		t.Fatalf("%+v", s)
	}
	if s.TechnicianPage != "http://127.0.0.1:17991/#technician" {
		t.Fatalf("tech page %q", s.TechnicianPage)
	}
	out := Format(s)
	if !strings.Contains(out, "Reader connected: SYNTHETIC ACR39U") {
		t.Fatalf("format %q", out)
	}
	assertNoPII(t, out)
}

func TestSummarizePracticeAndNoReader(t *testing.T) {
	ready := Summarize(Lights{Helper: true, Reader: true, Card: true, Practice: true, Backend: "available"})
	if ready.Headline != "Ready (practice card)" {
		t.Fatalf("headline %q", ready.Headline)
	}
	noReader := Summarize(Lights{Helper: true, Backend: "available"})
	if noReader.Reader || !strings.Contains(noReader.Next, "Plug the card reader") {
		t.Fatalf("%+v", noReader)
	}
	stalled := Summarize(Lights{Helper: true, Reader: true, Card: true, Backend: "stalled"})
	if stalled.Reader || stalled.Card || !strings.Contains(stalled.ReaderText, "not responding") {
		t.Fatalf("%+v", stalled)
	}
}

func assertNoPII(t *testing.T, s string) {
	t.Helper()
	for _, n := range []string{"1234567890121", "สมชาย", "Somchai", "citizenId", "1101701234567"} {
		if strings.Contains(s, n) {
			t.Fatalf("status leaked %q: %s", n, s)
		}
	}
}
