package card

import (
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
)

// decodeThaiField trims padding from a fixed-width TIS-620/Windows-874 field.
// Trailing spaces and NULs are removed; internal spaces are preserved.
func decodeThaiField(raw []byte, expect int) *string {
	if len(raw) == 0 {
		return nil
	}
	if expect > 0 && len(raw) > expect {
		raw = raw[:expect]
	}
	end := len(raw)
	for end > 0 && (raw[end-1] == 0x00 || raw[end-1] == 0x20) {
		end--
	}
	raw = raw[:end]
	if len(raw) == 0 {
		return nil
	}
	s := decodeWindows874(raw)
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}

// decodeWindows874 maps bytes to Unicode using the reviewed Windows-874
// decoder (superset of TIS-620 used by Thai identity cards).
func decodeWindows874(b []byte) string {
	out, err := charmap.Windows874.NewDecoder().Bytes(b)
	if err != nil {
		// Replacement path: ASCII pass-through plus U+FFFD for undecodable bytes.
		var sb strings.Builder
		sb.Grow(len(b))
		for i := 0; i < len(b); i++ {
			c := b[i]
			if c < 0x80 {
				sb.WriteByte(c)
			} else {
				sb.WriteRune(utf8.RuneError)
			}
		}
		return sb.String()
	}
	return string(out)
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

// parseBirthDate parses an 8-byte YYYYMMDD birth-date field.
func parseBirthDate(raw []byte) (*CardDate, *Warning) {
	return parseCardDate(raw, FieldBirthDate)
}

// parseCardDate parses an 8-byte YYYYMMDD field (birth, issue, or expire).
// Card years are Buddhist Era (typically 2400+); output year is Gregorian.
// Unknown/invalid components become null with a warning; a zero day is never
// replaced with 1. Leap-year and month-length rules are validated on the
// Gregorian calendar after conversion.
func parseCardDate(raw []byte, field Field) (*CardDate, *Warning) {
	noun := dateNoun(field)
	s := strings.TrimSpace(string(raw))
	s = strings.Trim(s, "\x00")
	if len(s) != 8 {
		w := Warning{Field: field, Code: "BAD_LENGTH", Message: noun + " must be 8 digits"}
		return nil, &w
	}
	for i := 0; i < 8; i++ {
		if s[i] < '0' || s[i] > '9' {
			w := Warning{Field: field, Code: "NOT_DIGITS", Message: noun + " must be digits"}
			return nil, &w
		}
	}
	y, _ := atoi2(s[0:4])
	mo, _ := atoi2(s[4:6])
	day, _ := atoi2(s[6:8])

	// Buddhist Era → Common Era for realistic Thai ID years.
	if y >= 2400 {
		y = y - 543
	}

	d := &CardDate{}
	var warns []Warning

	if y == 0 {
		d.Year = nil
	} else {
		d.Year = &y
	}
	if mo == 0 || mo > 12 {
		d.Month = nil
		if mo != 0 {
			warns = append(warns, Warning{Field: field, Code: "BAD_MONTH", Message: noun + " month invalid"})
		}
	} else {
		d.Month = &mo
	}

	maxDay := 0
	if mo >= 1 && mo <= 12 && y != 0 {
		maxDay = daysInGregorianMonth(y, mo)
	} else if mo >= 1 && mo <= 12 {
		// Year unknown: reject only clearly impossible days.
		maxDay = 31
	}

	if day == 0 {
		d.Day = nil
	} else if maxDay > 0 && day > maxDay {
		d.Day = nil
		warns = append(warns, Warning{Field: field, Code: "BAD_DAY", Message: noun + " day invalid"})
	} else {
		d.Day = &day
	}

	// Fully unknown date is an unavailable field, not an empty object.
	if d.Year == nil && d.Month == nil && d.Day == nil {
		if len(warns) == 0 {
			warns = append(warns, Warning{Field: field, Code: "EMPTY", Message: noun + " empty"})
		}
		return nil, &warns[0]
	}

	if len(warns) > 0 {
		return d, &warns[0]
	}
	return d, nil
}

func dateNoun(f Field) string {
	switch f {
	case FieldIssueDate:
		return "issue date"
	case FieldExpireDate:
		return "expire date"
	default:
		return "birth date"
	}
}

func atoi2(s string) (int, bool) {
	n := 0
	if len(s) == 0 {
		return 0, false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return 0, false
		}
		n = n*10 + int(s[i]-'0')
	}
	return n, true
}

func isGregorianLeap(y int) bool {
	return y%4 == 0 && (y%100 != 0 || y%400 == 0)
}

func daysInGregorianMonth(y, m int) int {
	switch m {
	case 1, 3, 5, 7, 8, 10, 12:
		return 31
	case 4, 6, 9, 11:
		return 30
	case 2:
		if isGregorianLeap(y) {
			return 29
		}
		return 28
	default:
		return 0
	}
}

// parseThaiNameBlock splits a single 100-byte Thai-name block.
func parseThaiNameBlock(raw []byte) (*ThaiName, *Warning) {
	return parseNameBlock(raw, FieldThaiName)
}

// parseNameBlock splits a 100-byte name block (Thai or English).
// Community card layout uses '#' between components:
//
//	title # given # middle # family
//
// An empty middle name appears as consecutive delimiters. When '#' is absent
// the block is split on whitespace (title given [middle] family).
func parseNameBlock(raw []byte, field Field) (*ThaiName, *Warning) {
	s := decodeThaiField(raw, 0)
	if s == nil {
		return nil, &Warning{Field: field, Code: "EMPTY", Message: "name fields empty"}
	}
	var parts []string
	if strings.Contains(*s, "#") {
		parts = strings.Split(*s, "#")
		// Drop a trailing empty component from a trailing delimiter.
		if n := len(parts); n > 0 && parts[n-1] == "" {
			parts = parts[:n-1]
		}
	} else {
		parts = strings.Fields(*s)
	}
	if len(parts) == 0 {
		return nil, &Warning{Field: field, Code: "EMPTY", Message: "name fields empty"}
	}

	trim := func(p string) *string {
		p = strings.TrimSpace(p)
		if p == "" {
			return nil
		}
		return &p
	}

	var title, given, middle, family *string
	switch len(parts) {
	case 1:
		// Only a single token — treat as given name.
		given = trim(parts[0])
	case 2:
		title = trim(parts[0])
		given = trim(parts[1])
	case 3:
		title = trim(parts[0])
		given = trim(parts[1])
		family = trim(parts[2])
	default:
		title = trim(parts[0])
		given = trim(parts[1])
		middle = trim(parts[2])
		family = trim(parts[3])
	}

	return buildThaiName(field, title, given, middle, family)
}

// parseGender maps the 1-byte community gender code.
// Hypothesis (not in APDU.md; used by independent community readers):
//
//	"1" → male, "2" → female
//
// Other values keep the raw code and set Value to null.
func parseGender(raw []byte) (*Gender, *Warning) {
	s := strings.TrimSpace(string(raw))
	s = strings.Trim(s, "\x00")
	if s == "" {
		return nil, &Warning{Field: FieldGender, Code: "EMPTY", Message: "gender empty"}
	}
	g := &Gender{Code: s}
	switch s {
	case "1":
		v := GenderMale
		g.Value = &v
	case "2":
		v := GenderFemale
		g.Value = &v
	default:
		return g, &Warning{Field: FieldGender, Code: "UNRECOGNIZED", Message: "gender code unrecognized"}
	}
	return g, nil
}

// parseIssuer decodes the 100-byte TIS-620 issuer field.
func parseIssuer(raw []byte) (*string, *Warning) {
	s := decodeThaiField(raw, 0)
	if s == nil {
		return nil, &Warning{Field: FieldIssuer, Code: "EMPTY", Message: "issuer empty"}
	}
	return s, nil
}

// parseAddressBlock splits a 100-byte address block.
// Hypothesis: '#' between house / village / alley / street / subdistrict /
// district / province, matching the name-block delimiter already used for
// thaiName. Prefixes such as "จังหวัด" are preserved (not stripped).
func parseAddressBlock(raw []byte) (*ThaiAddress, *Warning) {
	s := decodeThaiField(raw, 0)
	if s == nil {
		return nil, &Warning{Field: FieldAddress, Code: "EMPTY", Message: "address empty"}
	}
	addr := &ThaiAddress{}
	if strings.Contains(*s, "#") {
		parts := strings.Split(*s, "#")
		if n := len(parts); n > 0 && parts[n-1] == "" {
			parts = parts[:n-1]
		}
		get := func(i int) *string {
			if i >= len(parts) {
				return nil
			}
			p := strings.TrimSpace(parts[i])
			if p == "" {
				return nil
			}
			return &p
		}
		addr.HouseNo = get(0)
		addr.VillageNo = get(1)
		addr.Alley = get(2)
		addr.Street = get(3)
		addr.Subdistrict = get(4)
		addr.District = get(5)
		addr.Province = get(6)
		nonEmpty := make([]string, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				nonEmpty = append(nonEmpty, p)
			}
		}
		addr.FullAddress = strings.Join(nonEmpty, " ")
	} else {
		addr.FullAddress = *s
	}
	if addr.FullAddress == "" {
		return nil, &Warning{Field: FieldAddress, Code: "EMPTY", Message: "address empty"}
	}
	return addr, nil
}

// buildThaiName assembles a ThaiName and fullName.
func buildThaiName(field Field, title, given, middle, family *string) (*ThaiName, *Warning) {
	if title == nil && given == nil && middle == nil && family == nil {
		return nil, &Warning{Field: field, Code: "EMPTY", Message: "name fields empty"}
	}
	parts := make([]string, 0, 4)
	for _, p := range []*string{title, given, middle, family} {
		if p != nil && *p != "" {
			parts = append(parts, *p)
		}
	}
	if len(parts) == 0 {
		return nil, &Warning{Field: field, Code: "EMPTY", Message: "name fields empty"}
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
