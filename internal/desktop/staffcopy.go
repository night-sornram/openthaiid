package desktop

import (
	"os"
	"strings"
)

// Lang is clinic-staff UI language. Technician/debug copy stays English.
type Lang string

const (
	LangTH Lang = "th"
	LangEN Lang = "en"
)

// StaffLang is the default staff language: Thai, unless OPENTHAIID_LANG
// (or OPENTHAIID_LOCALE) is an English tag. Explicit "en" is the fallback.
func StaffLang() Lang {
	return ParseLang(firstNonEmpty(os.Getenv("OPENTHAIID_LANG"), os.Getenv("OPENTHAIID_LOCALE")))
}

// ParseLang maps a BCP-47-ish tag to Thai or English. Empty / unknown → Thai.
func ParseLang(tag string) Lang {
	t := strings.ToLower(strings.TrimSpace(tag))
	t = strings.ReplaceAll(t, "_", "-")
	if t == "en" || strings.HasPrefix(t, "en-") {
		return LangEN
	}
	return LangTH
}

// Copy is staff-facing Ready / status strings. No personal card fields.
type Copy struct {
	Lang              Lang
	Ready             string
	ReadyPractice     string
	NotReady          string
	Cancelled         string
	HelperOn          string
	HelperOff         string
	ReaderUnknown     string
	CardUnknown       string
	StartHelper       string
	ServiceOff        string
	ServiceStalled    string
	CardUntilService  string
	FixService        string
	NoReader          string
	InsertAfterReader string
	PlugReader        string
	ReaderConnected   string // %s = device name
	ChooseReader      string
	InsertCard        string
	InsertThenFill    string
	CardInserted      string
	ClickFill         string
	UnsupportedCard   string
	InsertThaiID      string
	StatusTitle       string
	MenuReady         string
	MenuTechnician    string
	MenuStatus        string
	MenuAutostart     string
	MenuUpdate        string
	MenuReleases      string
	MenuQuit          string
	Tooltip           string
}

// StaffCopy returns clinic-staff strings. Default Thai; LangEN is the fallback.
func StaffCopy(lang Lang) Copy {
	if lang == LangEN {
		return englishCopy()
	}
	return thaiCopy()
}

func thaiCopy() Copy {
	return Copy{
		Lang:              LangTH,
		Ready:             "พร้อม",
		ReadyPractice:     "พร้อม (บัตรฝึกซ้อม)",
		NotReady:          "ยังไม่พร้อม",
		Cancelled:         "ยกเลิกแล้ว",
		HelperOn:          "OpenThaiID กำลังทำงานบนเครื่องนี้",
		HelperOff:         "OpenThaiID ยังไม่ทำงานบนเครื่องนี้ กรุณาเปิดโปรแกรม แล้วลองอีกครั้ง",
		ReaderUnknown:     "เครื่องอ่านบัตร: ยังไม่ทราบ จนกว่าจะเปิด OpenThaiID",
		CardUnknown:       "บัตร: ยังไม่ทราบ จนกว่าจะเปิด OpenThaiID",
		StartHelper:       "เปิด OpenThaiID บนเครื่องนี้",
		ServiceOff:        "บริการอ่านบัตรยังไม่พร้อม (หลังเครื่องหลับก็เจอได้) ไม่ต้องรีสตาร์ทเครื่อง",
		ServiceStalled:    "บริการอ่านบัตรไม่ตอบสนอง ให้ช่างเทคนิคช่วยตรวจ — ไม่ต้องรีสตาร์ทเครื่อง",
		CardUntilService:  "บัตร: ยังไม่ทราบ จนกว่าบริการอ่านบัตรจะทำงาน",
		FixService:        "กด ทดสอบการเชื่อมต่อ หรือให้ช่างเทคนิคกด Reconnect card service แล้วรอ พร้อม",
		NoReader:          "ไม่พบเครื่องอ่านบัตร เสียบเครื่องอ่านเข้าเครื่องนี้",
		InsertAfterReader: "เสียบบัตรหลังจากต่อเครื่องอ่านแล้ว",
		PlugReader:        "เสียบเครื่องอ่านบัตรเข้าเครื่องนี้",
		ReaderConnected:   "ต่อเครื่องอ่านแล้ว: %s",
		ChooseReader:      "มีเครื่องอ่านมากกว่าหนึ่งเครื่อง เลือกเครื่องบนหน้า พร้อม",
		InsertCard:        "เสียบบัตรประชาชนไทย",
		InsertThenFill:    "เสียบบัตรประชาชนของผู้ป่วย แล้วกด กรอกข้อมูล",
		CardInserted:      "เสียบบัตรแล้ว",
		ClickFill:         "กด กรอกข้อมูล",
		UnsupportedCard:   "บัตรนี้ไม่ใช่บัตรประชาชนไทย",
		InsertThaiID:      "เสียบบัตรประชาชนไทย",
		StatusTitle:       "สถานะ OpenThaiID",
		MenuReady:         "เปิดหน้า พร้อม",
		MenuTechnician:    "Technician tools",
		MenuStatus:        "แสดงสถานะ",
		MenuAutostart:     "เริ่มเมื่อเข้าสู่ระบบ",
		MenuUpdate:        "ตรวจเวอร์ชัน",
		MenuReleases:      "GitHub Releases",
		MenuQuit:          "ออก",
		Tooltip:           "ตัวช่วย OpenThaiID",
	}
}

func englishCopy() Copy {
	return Copy{
		Lang:              LangEN,
		Ready:             "Ready",
		ReadyPractice:     "Ready (practice card)",
		NotReady:          "Not ready yet",
		Cancelled:         "Cancelled",
		HelperOn:          "OpenThaiID is running on this computer.",
		HelperOff:         "OpenThaiID is not running on this computer. Start it, then try again.",
		ReaderUnknown:     "Card reader: unknown until OpenThaiID is running.",
		CardUnknown:       "Card: unknown until OpenThaiID is running.",
		StartHelper:       "Start OpenThaiID on this computer.",
		ServiceOff:        "The card service is not ready (this can happen after sleep). Do not reboot.",
		ServiceStalled:    "The card service is not responding. Ask a technician — do not reboot.",
		CardUntilService:  "Card: unknown until the card service is working.",
		FixService:        "Click Test connection, or ask a technician to click Reconnect card service, then wait until Ready.",
		NoReader:          "No card reader. Plug the reader into this computer.",
		InsertAfterReader: "Insert the card after the reader is connected.",
		PlugReader:        "Plug the card reader into this computer.",
		ReaderConnected:   "Reader connected: %s",
		ChooseReader:      "More than one reader is connected. Choose one on the Ready page.",
		InsertCard:        "Insert the Thai ID card.",
		InsertThenFill:    "Insert the patient’s Thai ID card, then click Fill.",
		CardInserted:      "Card is inserted.",
		ClickFill:         "Click Fill.",
		UnsupportedCard:   "This card is not a Thai National ID.",
		InsertThaiID:      "Insert a Thai National ID card.",
		StatusTitle:       "OpenThaiID status",
		MenuReady:         "Open Ready page",
		MenuTechnician:    "Open Technician tools",
		MenuStatus:        "Show status",
		MenuAutostart:     "Start at login",
		MenuUpdate:        "Check version",
		MenuReleases:      "Open GitHub Releases",
		MenuQuit:          "Quit",
		Tooltip:           "OpenThaiID helper",
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
