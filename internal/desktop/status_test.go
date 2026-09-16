package desktop

import (
	"strings"
	"testing"
)

func TestSummarizeNotRunningThaiDefault(t *testing.T) {
	t.Setenv("OPENTHAIID_LANG", "")
	t.Setenv("OPENTHAIID_LOCALE", "")
	s := Summarize(Lights{})
	if s.Helper || s.Reader || s.Card {
		t.Fatalf("lights %+v", s)
	}
	if s.Lang != "th" {
		t.Fatalf("lang %q", s.Lang)
	}
	if s.Headline != "ยังไม่พร้อม" {
		t.Fatalf("headline %q", s.Headline)
	}
	if !strings.Contains(s.HelperText, "ยังไม่ทำงาน") {
		t.Fatalf("helper text %q", s.HelperText)
	}
	out := Format(s)
	assertNoPII(t, out)
}

func TestSummarizeReadyThai(t *testing.T) {
	s := Summarize(Lights{
		Helper:     true,
		Reader:     true,
		Card:       true,
		Backend:    "available",
		ReaderName: "SYNTHETIC ACR39U (fictional demo)",
		ReadyPage:  "http://127.0.0.1:17991/",
		Lang:       LangTH,
	})
	if !s.Helper || !s.Reader || !s.Card || s.Headline != "พร้อม" {
		t.Fatalf("%+v", s)
	}
	if s.TechnicianPage != "http://127.0.0.1:17991/#technician" {
		t.Fatalf("tech page %q", s.TechnicianPage)
	}
	out := Format(s)
	if !strings.Contains(out, "ต่อเครื่องอ่านแล้ว: SYNTHETIC ACR39U") {
		t.Fatalf("format %q", out)
	}
	assertNoPII(t, out)
}

func TestSummarizeEnglishFallback(t *testing.T) {
	t.Setenv("OPENTHAIID_LANG", "en")
	s := Summarize(Lights{
		Helper:     true,
		Reader:     true,
		Card:       true,
		Backend:    "available",
		ReaderName: "SYNTHETIC ACR39U (fictional demo)",
	})
	if s.Lang != "en" || s.Headline != "Ready" {
		t.Fatalf("%+v", s)
	}
	if !strings.Contains(s.ReaderText, "Reader connected: SYNTHETIC ACR39U") {
		t.Fatalf("reader %q", s.ReaderText)
	}
	explicit := Summarize(Lights{Helper: true, Lang: LangEN, Backend: "available"})
	if explicit.Headline != "Not ready yet" {
		t.Fatalf("explicit en %+v", explicit)
	}
}

func TestSummarizePracticeAndNoReader(t *testing.T) {
	ready := Summarize(Lights{Helper: true, Reader: true, Card: true, Practice: true, Backend: "available", Lang: LangTH})
	if ready.Headline != "พร้อม (บัตรฝึกซ้อม)" {
		t.Fatalf("headline %q", ready.Headline)
	}
	enReady := Summarize(Lights{Helper: true, Reader: true, Card: true, Practice: true, Backend: "available", Lang: LangEN})
	if enReady.Headline != "Ready (practice card)" {
		t.Fatalf("en headline %q", enReady.Headline)
	}
	noReader := Summarize(Lights{Helper: true, Backend: "available", Lang: LangEN})
	if noReader.Reader || !strings.Contains(noReader.Next, "Plug the card reader") {
		t.Fatalf("%+v", noReader)
	}
	stalled := Summarize(Lights{Helper: true, Reader: true, Card: true, Backend: "stalled", Lang: LangEN})
	if stalled.Reader || stalled.Card || !strings.Contains(stalled.ReaderText, "not responding") {
		t.Fatalf("%+v", stalled)
	}
	thStall := Summarize(Lights{Helper: true, Backend: "stalled", Lang: LangTH})
	if !strings.Contains(thStall.ReaderText, "ไม่ตอบสนอง") {
		t.Fatalf("%+v", thStall)
	}
	unavailable := Summarize(Lights{Helper: true, Reader: true, Card: true, Backend: "unavailable", Lang: LangTH})
	if unavailable.Headline != "ยังไม่พร้อม" || unavailable.Reader || unavailable.Card {
		t.Fatalf("unavailable must not be Ready: %+v", unavailable)
	}
	if !strings.Contains(unavailable.Next, "ทดสอบการเชื่อมต่อ") || !strings.Contains(unavailable.Next, "Reconnect card service") {
		t.Fatalf("next %q", unavailable.Next)
	}
	if strings.Contains(Format(unavailable), "PC/SC") || strings.Contains(Format(unavailable), "1234567890121") {
		t.Fatalf("unavailable leaked jargon or PII: %s", Format(unavailable))
	}
	needPick := Summarize(Lights{
		Helper:        true,
		Reader:        true,
		Card:          true,
		NeedSelection: true,
		Backend:       "available",
		ReaderName:    "Reader A",
		Lang:          LangTH,
	})
	if needPick.Card || needPick.Headline != "ยังไม่พร้อม" {
		t.Fatalf("multi-reader must not be Ready: %+v", needPick)
	}
	if !strings.Contains(needPick.Next, "เลือกเครื่อง") {
		t.Fatalf("next %q", needPick.Next)
	}
	assertNoPII(t, Format(needPick))
}

func TestParseLang(t *testing.T) {
	if ParseLang("") != LangTH || ParseLang("th-TH") != LangTH {
		t.Fatal("default thai")
	}
	if ParseLang("en") != LangEN || ParseLang("en-GB") != LangEN {
		t.Fatal("english fallback")
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
