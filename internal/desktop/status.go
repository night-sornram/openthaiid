// Package desktop implements clinic-PC helper UX: tray, autostart, and
// staff-safe status text. It never formats citizen ID, names, dates, or
// other personal card fields.
package desktop

import "strings"

// Lights is helper / reader / card presence for clinic staff.
// ReaderName is a device label (for example "ACS ACR39U"), never card data.
type Lights struct {
	Helper         bool
	Reader         bool
	Card           bool
	Practice       bool
	Backend        string
	ReaderName     string
	ReadyPage      string
	TechnicianPage string
}

// Summary is the plain-language status shown in the tray, CLI, and
// GET /v1/helper/summary. JSON tags are for the loopback helper API.
type Summary struct {
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

// Summarize turns lights into clinic-staff copy. No personal field values.
func Summarize(l Lights) Summary {
	s := Summary{
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
		s.Headline = "Not ready yet"
		s.HelperText = "OpenThaiID is not running on this computer. Start it, then try again."
		s.ReaderText = "Card reader: unknown until OpenThaiID is running."
		s.CardText = "Card: unknown until OpenThaiID is running."
		s.Next = "Start OpenThaiID on this computer."
		return s
	}

	s.HelperText = "OpenThaiID is running on this computer."
	switch l.Backend {
	case "unavailable":
		s.Headline = "Not ready yet"
		s.ReaderText = "The card service is off. Ask a technician to start it."
		s.CardText = "Card: unknown until the card service is working."
		s.Next = "Ask a technician to fix the card service."
		s.Reader = false
		s.Card = false
		return s
	case "stalled":
		s.Headline = "Not ready yet"
		s.ReaderText = "The card service is not responding. Ask a technician."
		s.CardText = "Card: unknown until the card service is working."
		s.Next = "Ask a technician to fix the card service."
		s.Reader = false
		s.Card = false
		return s
	}

	if !l.Reader {
		s.Headline = "Not ready yet"
		s.ReaderText = "No card reader. Plug the reader into this computer."
		s.CardText = "Insert the card after the reader is connected."
		s.Next = "Plug the card reader into this computer."
		return s
	}

	name := strings.TrimSpace(l.ReaderName)
	if name == "" {
		name = "card reader"
	}
	s.ReaderText = "Reader connected: " + name
	if !l.Card {
		s.Headline = "Not ready yet"
		s.CardText = "Insert the Thai ID card."
		s.Next = "Insert the patient’s Thai ID card, then click Fill."
		return s
	}

	s.Headline = "Ready"
	if l.Practice {
		s.Headline = "Ready (practice card)"
	}
	s.CardText = "Card is inserted."
	s.Next = "Click Fill."
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
