package qualify

import (
	"bytes"
	"strings"

	"github.com/night-sornram/openthaiid/internal/card"
)

// ForbiddenSubstrings must never appear in a qualification report or
// technician diagnostics copy. They cover the synthetic fixture plus
// common encodings of personal payloads (JPEG Base64, data URLs).
//
// Real citizen IDs, names, and addresses are not listed here (unknown);
// reports are built so those values are never copied in. This list is
// the last-chance blocker for the seeded fictional card and photo bytes.
var ForbiddenSubstrings = []string{
	"1234567890121",
	"1101701234567",
	"สมชาย",
	"ใจดี",
	"Somchai",
	"Jaidee",
	"เทศบาลนครนิทาน",
	"ถนนนิทาน",
	"ตำบลบ้านใหม่",
	"อำเภอเมือง",
	"จังหวัดนิทาน",
	"25330412",
	"25600115",
	"25700114",
	card.FictionalPortraitSentinel,
	"/9j/", // JPEG SOI in standard Base64
	"data:image",
	"image/jpeg;base64,",
}

// FirstForbidden reports the first forbidden substring in p, or "".
func FirstForbidden(p []byte) string {
	for _, s := range ForbiddenSubstrings {
		if s == "" {
			continue
		}
		if bytes.Contains(p, []byte(s)) {
			return s
		}
	}
	// 13-digit runs are almost always a citizen ID. Reports may include
	// counts like "digitCount":13 — that is a short number, not a 13-digit
	// identifier. Flag only contiguous 13-digit tokens.
	if id := thirteenDigitToken(p); id != "" {
		return id
	}
	return ""
}

func thirteenDigitToken(p []byte) string {
	n := 0
	start := 0
	for i, b := range p {
		if b >= '0' && b <= '9' {
			if n == 0 {
				start = i
			}
			n++
			continue
		}
		if n == 13 {
			return string(p[start:i])
		}
		n = 0
	}
	if n == 13 {
		return string(p[start:])
	}
	return ""
}

// ContainsForbidden is a convenience wrapper around FirstForbidden.
func ContainsForbidden(s string) bool {
	return FirstForbidden([]byte(s)) != ""
}

// SanitizeLine drops a technician line rather than storing personal data.
func SanitizeLine(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if ContainsForbidden(s) {
		return "[redacted]"
	}
	return s
}
