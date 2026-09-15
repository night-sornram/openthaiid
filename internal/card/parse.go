package card

import (
	"strconv"
	"strings"
	"unicode/utf8"
)

// decodeThaiField trims padding from a fixed-width TIS-620/Windows-874 field.
// Trailing spaces and NULs are removed; internal spaces are preserved.
// Invalid bytes become U+FFFD via replacement decoding but still yield a value
// with a warning at a higher layer when needed.
func decodeThaiField(raw []byte, expect int) *string {
	if len(raw) == 0 {
		return nil
	}
	// Only use up to expect bytes if the buffer is longer.
	if expect > 0 && len(raw) > expect {
		raw = raw[:expect]
	}
	// Trim trailing NULs and ASCII spaces (common card padding).
	end := len(raw)
	for end > 0 && (raw[end-1] == 0x00 || raw[end-1] == 0x20) {
		end--
	}
	raw = raw[:end]
	if len(raw) == 0 {
		return nil
	}
	s := decodeTIS620(raw)
	// Collapse only leading/trailing whitespace after decode.
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}

// decodeTIS620 maps bytes to Unicode.
// TIS-620 is the 0xA1–0xFB Thai range; 0x00–0x7F are ASCII.
// Bytes 0x80–0xA0 and 0xFC–0xFF use Windows-874 extensions when valid.
func decodeTIS620(b []byte) string {
	var sb strings.Builder
	sb.Grow(len(b))
	for i := 0; i < len(b); i++ {
		c := b[i]
		switch {
		case c < 0x80:
			sb.WriteByte(c)
		case c >= 0xA1 && c <= 0xFB:
			// TIS-620 Thai: U+0E01 + (c - 0xA1)
			sb.WriteRune(rune(0x0E01 + int(c-0xA1)))
		case c == 0xA0:
			sb.WriteRune(0x00A0) // NBSP (Win-874)
		case c >= 0x80 && c <= 0x9F:
			// Win-874 specials (quotes, ellipsis, etc.) — map a few common ones.
			sb.WriteString(win874Special(c))
		default:
			sb.WriteRune(utf8.RuneError)
		}
	}
	return sb.String()
}

func win874Special(c byte) string {
	switch c {
	case 0x80:
		return "€"
	case 0x85:
		return "…"
	case 0x91:
		return "‘"
	case 0x92:
		return "’"
	case 0x93:
		return "“"
	case 0x94:
		return "”"
	case 0x95:
		return "•"
	case 0x96:
		return "–"
	case 0x97:
		return "—"
	default:
		return string(rune(c))
	}
}

// parseCitizenID validates a 13-digit citizen id.
// Checksum is a data check, not identity verification.
func parseCitizenID(raw []byte) (*string, *Warning) {
	s := strings.TrimSpace(string(raw))
	s = strings.Trim(s, "\x00")
	if len(s) != 13 {
		w := Warning{Field: FieldCitizenID, Code: "BAD_LENGTH", Message: "citizen id must be 13 digits"}
		return nil, &w
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			w := Warning{Field: FieldCitizenID, Code: "NOT_DIGITS", Message: "citizen id must be digits"}
			return nil, &w
		}
	}
	if !citizenIDChecksumOK(s) {
		// Still return the value; checksum is advisory.
		w := Warning{Field: FieldCitizenID, Code: "CHECKSUM", Message: "citizen id checksum failed"}
		return &s, &w
	}
	return &s, nil
}

// citizenIDChecksumOK implements the public Thai ID check-digit algorithm:
// sum(d[i] * (13-i)) for i in 0..11, check = (11 - sum%11) % 10 == d[12].
func citizenIDChecksumOK(s string) bool {
	if len(s) != 13 {
		return false
	}
	sum := 0
	for i := 0; i < 12; i++ {
		d := int(s[i] - '0')
		sum += d * (13 - i)
	}
	check := (11 - sum%11) % 10
	return check == int(s[12]-'0')
}

// parseBirthDate parses an 8-byte YYYYMMDD field.
// Year is Buddhist Era when it looks like BE (>= 2400).
// Unknown/invalid components become null with a warning; a zero day is never
// replaced with 1.
func parseBirthDate(raw []byte) (*CardDate, *Warning) {
	s := strings.TrimSpace(string(raw))
	s = strings.Trim(s, "\x00")
	if len(s) != 8 {
		w := Warning{Field: FieldBirthDate, Code: "BAD_LENGTH", Message: "birth date must be 8 digits"}
		return nil, &w
	}
	for i := 0; i < 8; i++ {
		if s[i] < '0' || s[i] > '9' {
			w := Warning{Field: FieldBirthDate, Code: "NOT_DIGITS", Message: "birth date must be digits"}
			return nil, &w
		}
	}
	y, _ := strconv.Atoi(s[0:4])
	mo, _ := strconv.Atoi(s[4:6])
	day, _ := strconv.Atoi(s[6:8])

	d := &CardDate{}
	var warns []Warning

	// Buddhist Era years are 2400+; Gregorian would be 1900–2099 for this domain.
	if y == 0 {
		d.Year = nil
	} else {
		d.Year = &y
	}
	if mo == 0 || mo > 12 {
		d.Month = nil
		if mo != 0 {
			warns = append(warns, Warning{Field: FieldBirthDate, Code: "BAD_MONTH", Message: "birth month invalid"})
		}
	} else {
		d.Month = &mo
	}
	if day == 0 || day > 31 {
		// Zero day means unknown — do not default to 1.
		d.Day = nil
		if day != 0 {
			warns = append(warns, Warning{Field: FieldBirthDate, Code: "BAD_DAY", Message: "birth day invalid"})
		}
	} else {
		d.Day = &day
	}
	if len(warns) > 0 {
		return d, &warns[0]
	}
	return d, nil
}

// buildThaiName assembles a ThaiName and fullName.
func buildThaiName(title, given, middle, family *string) (*ThaiName, *Warning) {
	if title == nil && given == nil && middle == nil && family == nil {
		return nil, &Warning{Field: FieldThaiName, Code: "EMPTY", Message: "name fields empty"}
	}
	parts := make([]string, 0, 4)
	for _, p := range []*string{title, given, middle, family} {
		if p != nil && *p != "" {
			parts = append(parts, *p)
		}
	}
	if len(parts) == 0 {
		return nil, &Warning{Field: FieldThaiName, Code: "EMPTY", Message: "name fields empty"}
	}
	return &ThaiName{
		Title:      title,
		GivenName:  given,
		MiddleName: middle,
		FamilyName: family,
		FullName:   strings.Join(parts, " "),
	}, nil
}

// BEToCE converts a Buddhist Era year to Common Era.
func BEToCE(be int) int { return be - 543 }
