// Package desktop implements clinic-PC helper UX: tray, autostart, and
// staff-safe status text. It never formats citizen ID, names, dates, or
// other personal card fields.
package desktop

import (
	"fmt"
	"strings"
)

// Lights is helper / reader / card presence for clinic staff.
// ReaderName is a device label (for example "ACS ACR39U"), never card data.
type Lights struct {
	Helper         bool
	Reader         bool
	Card           bool
	Practice       bool
	Backend        string
	ReaderName     string
	NeedSelection  bool
	ReadyPage      string
	TechnicianPage string
	Lang           Lang // empty → StaffLang() (Thai default)
}

// Summary is the plain-language status shown in the tray, CLI, and
// GET /v1/helper/summary. JSON tags are for the loopback helper API.
type Summary struct {
	Lang           string `json:"lang"`
	Helper         bool   `json:"helper"`
	Reader         bool   `json:"reader"`
	Card           bool   `json:"card"`
	Practice       bool   `json:"practice"`
	Backend        string `json:"backend,omitempty"`
	ReadyPage      string `json:"readyPage"`
	TechnicianPage string `json:"technicianPage"`
	Headline       string `json:"headline"`
	HelperText     string `json:"helperText"`
	ReaderText     string `json:"readerText"`
	CardText       string `json:"cardText"`
	Next           string `json:"next"`
}

// Summarize turns lights into clinic-staff copy (Thai by default).
// No personal field values. Pass Lights.Lang = LangEN for English.
func Summarize(l Lights) Summary {
	lang := l.Lang
	if lang == "" {
		lang = StaffLang()
	}
	c := StaffCopy(lang)
	s := Summary{
		Lang:           string(c.Lang),
		Helper:         l.Helper,
		Reader:         l.Reader,
		Card:           l.Card,
		Practice:       l.Practice,
		Backend:        l.Backend,
		ReadyPage:      l.ReadyPage,
		TechnicianPage: l.TechnicianPage,
	}
	if s.ReadyPage == "" {
		s.ReadyPage = "http://127.0.0.1:17891/"
	}
	if s.TechnicianPage == "" {
		s.TechnicianPage = strings.TrimRight(s.ReadyPage, "/") + "/#technician"
	}

	if !l.Helper {
		s.Headline = c.NotReady
		s.HelperText = c.HelperOff
		s.ReaderText = c.ReaderUnknown
		s.CardText = c.CardUnknown
		s.Next = c.StartHelper
		return s
	}

	s.HelperText = c.HelperOn
	switch l.Backend {
	case "unavailable":
		s.Headline = c.NotReady
		s.ReaderText = c.ServiceOff
		s.CardText = c.CardUntilService
		s.Next = c.FixService
		s.Reader = false
		s.Card = false
		return s
	case "stalled":
		s.Headline = c.NotReady
		s.ReaderText = c.ServiceStalled
		s.CardText = c.CardUntilService
		s.Next = c.FixService
		s.Reader = false
		s.Card = false
		return s
	}

	if !l.Reader {
		s.Headline = c.NotReady
		s.ReaderText = c.NoReader
		s.CardText = c.InsertAfterReader
		s.Next = c.PlugReader
		return s
	}

	name := strings.TrimSpace(l.ReaderName)
	if name == "" {
		if lang == LangEN {
			name = "card reader"
		} else {
			name = "เครื่องอ่านบัตร"
		}
	}
	s.ReaderText = fmt.Sprintf(c.ReaderConnected, name)
	if l.NeedSelection {
		s.Headline = c.NotReady
		s.Card = false
		s.CardText = c.InsertAfterReader
		s.Next = c.ChooseReader
		return s
	}
	if !l.Card {
		s.Headline = c.NotReady
		s.CardText = c.InsertCard
		s.Next = c.InsertThenFill
		return s
	}

	s.Headline = c.Ready
	if l.Practice {
		s.Headline = c.ReadyPractice
	}
	s.CardText = c.CardInserted
	s.Next = c.ClickFill
	return s
}

// Format returns a short multi-line status for the tray dialog and CLI.
func Format(s Summary) string {
	var b strings.Builder
	b.WriteString(s.Headline)
	b.WriteByte('\n')
	if s.Next != "" {
		b.WriteString(s.Next)
		b.WriteByte('\n')
	}
	b.WriteString(s.HelperText)
	b.WriteByte('\n')
	b.WriteString(s.ReaderText)
	b.WriteByte('\n')
	b.WriteString(s.CardText)
	return b.String()
}
