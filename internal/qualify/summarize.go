package qualify

import (
	"encoding/base64"
	"unicode"

	"github.com/night-sornram/openthaiid/internal/card"
)

// ReadShape is the PII-free summary of one card read.
// Values (citizen ID, names, address text, photo bytes) are never copied.
type ReadShape struct {
	FieldsPresent []string        `json:"fieldsPresent"`
	FieldsMissing []string        `json:"fieldsMissing,omitempty"`
	Warnings      []WarningShape  `json:"warnings,omitempty"`
	CitizenID     *CitizenIDShape `json:"citizenId,omitempty"`
	ThaiName      *NameShape      `json:"thaiName,omitempty"`
	EnglishName   *NameShape      `json:"englishName,omitempty"`
	BirthDate     *DateShape      `json:"birthDate,omitempty"`
	Gender        *GenderShape    `json:"gender,omitempty"`
	Issuer        *PresenceShape  `json:"issuer,omitempty"`
	IssueDate     *DateShape      `json:"issueDate,omitempty"`
	ExpireDate    *DateShape      `json:"expireDate,omitempty"`
	Address       *AddressShape   `json:"address,omitempty"`
	Portrait      *PortraitShape  `json:"portrait,omitempty"`
}

// WarningShape copies field-level warning codes only.
type WarningShape struct {
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message,omitempty"`
}

// CitizenIDShape records length, not the identifier.
type CitizenIDShape struct {
	Present    bool `json:"present"`
	DigitCount int  `json:"digitCount"`
}

// NameShape lists which structural parts were non-empty. No name text.
type NameShape struct {
	Present         bool     `json:"present"`
	PartsPresent    []string `json:"partsPresent,omitempty"`
	FullNamePresent bool     `json:"fullNamePresent"`
}

// DateShape records completeness, not the calendar values.
type DateShape struct {
	Present  bool `json:"present"`
	Year     bool `json:"year"`
	Month    bool `json:"month"`
	Day      bool `json:"day"`
	Complete bool `json:"complete"`
}

// GenderShape records the community 1/2 encoding hypothesis without
// personal identity. Code is a 1-byte card flag, not a citizen ID.
type GenderShape struct {
	Present   bool   `json:"present"`
	CodeLen   int    `json:"codeLen"`
	CodeDigit bool   `json:"codeDigit"`
	Code      string `json:"code,omitempty"`
	Value     string `json:"value,omitempty"` // male | female | unknown
}

// PresenceShape is a boolean with a character-count (not the text).
type PresenceShape struct {
	Present bool `json:"present"`
	CharLen int  `json:"charLen"`
}

// AddressShape records the 7-part '#' layout hypothesis: which components
// were non-empty. No house number, street, or province text.
type AddressShape struct {
	Present            bool     `json:"present"`
	PartsPresent       []string `json:"partsPresent,omitempty"`
	ComponentCount     int      `json:"componentCount"`
	FullAddressPresent bool     `json:"fullAddressPresent"`
}

// PortraitShape records JPEG assembly diagnostics. No Base64, no pixels.
type PortraitShape struct {
	Present         bool   `json:"present"`
	MimeType        string `json:"mimeType,omitempty"`
	Encoding        string `json:"encoding,omitempty"`
	Base64CharCount int    `json:"base64CharCount,omitempty"`
	DecodedBytes    int    `json:"decodedBytes,omitempty"`
	JPEGSOI         bool   `json:"jpegSOI"`
	JPEGEOI         bool   `json:"jpegEOI"`
	Decode          string `json:"decode,omitempty"`
}

// SummarizeRead builds a PII-free shape from a successful (or partial) read.
func SummarizeRead(res *card.ReadResult) ReadShape {
	out := ReadShape{}
	if res == nil {
		return out
	}
	f := res.Fields
	out.CitizenID = summarizeCitizenID(f.CitizenID)
	out.ThaiName = summarizeName(f.ThaiName)
	out.EnglishName = summarizeName(f.EnglishName)
	out.BirthDate = summarizeDate(f.BirthDate)
	out.Gender = summarizeGender(f.Gender)
	out.Issuer = summarizePresence(f.Issuer)
	out.IssueDate = summarizeDate(f.IssueDate)
	out.ExpireDate = summarizeDate(f.ExpireDate)
	out.Address = summarizeAddress(f.Address)
	out.Portrait = summarizePortrait(f.Portrait)

	type item struct {
		name    string
		present bool
	}
	for _, it := range []item{
		{"citizenId", out.CitizenID != nil && out.CitizenID.Present},
		{"thaiName", out.ThaiName != nil && out.ThaiName.Present},
		{"englishName", out.EnglishName != nil && out.EnglishName.Present},
		{"birthDate", out.BirthDate != nil && out.BirthDate.Present},
		{"gender", out.Gender != nil && out.Gender.Present},
		{"issuer", out.Issuer != nil && out.Issuer.Present},
		{"issueDate", out.IssueDate != nil && out.IssueDate.Present},
		{"expireDate", out.ExpireDate != nil && out.ExpireDate.Present},
		{"address", out.Address != nil && out.Address.Present},
		{"portrait", out.Portrait != nil && out.Portrait.Present},
	} {
		if it.present {
			out.FieldsPresent = append(out.FieldsPresent, it.name)
		} else {
			out.FieldsMissing = append(out.FieldsMissing, it.name)
		}
	}
	for _, w := range res.Warnings {
		out.Warnings = append(out.Warnings, WarningShape{
			Field:   string(w.Field),
			Code:    w.Code,
			Message: SanitizeLine(w.Message),
		})
	}
	return out
}

func summarizeCitizenID(v *string) *CitizenIDShape {
	if v == nil {
		return &CitizenIDShape{Present: false}
	}
	n := 0
	for _, r := range *v {
		if unicode.IsDigit(r) {
			n++
		}
	}
	return &CitizenIDShape{Present: *v != "", DigitCount: n}
}

func summarizeName(n *card.ThaiName) *NameShape {
	if n == nil {
		return &NameShape{Present: false}
	}
	s := &NameShape{Present: true, FullNamePresent: n.FullName != ""}
	add := func(label string, p *string) {
		if p != nil && *p != "" {
			s.PartsPresent = append(s.PartsPresent, label)
		}
	}
	add("title", n.Title)
	add("givenName", n.GivenName)
	add("middleName", n.MiddleName)
	add("familyName", n.FamilyName)
	return s
}

func summarizeDate(d *card.CardDate) *DateShape {
	if d == nil {
		return &DateShape{Present: false}
	}
	s := &DateShape{
		Present: true,
		Year:    d.Year != nil,
		Month:   d.Month != nil,
		Day:     d.Day != nil,
	}
	s.Complete = s.Year && s.Month && s.Day
	return s
}

func summarizeGender(g *card.Gender) *GenderShape {
	if g == nil {
		return &GenderShape{Present: false}
	}
	s := &GenderShape{
		Present: true,
		CodeLen: len(g.Code),
		Code:    g.Code,
	}
	if len(g.Code) == 1 && g.Code[0] >= '0' && g.Code[0] <= '9' {
		s.CodeDigit = true
	}
	s.Value = "unknown"
	if g.Value != nil {
		switch *g.Value {
		case card.GenderMale, card.GenderFemale:
			s.Value = *g.Value
		}
	}
	return s
}

func summarizePresence(v *string) *PresenceShape {
	if v == nil {
		return &PresenceShape{Present: false}
	}
	return &PresenceShape{Present: *v != "", CharLen: len([]rune(*v))}
}

func summarizeAddress(a *card.ThaiAddress) *AddressShape {
	if a == nil {
		return &AddressShape{Present: false}
	}
	s := &AddressShape{Present: true, FullAddressPresent: a.FullAddress != ""}
	add := func(label string, p *string) {
		if p != nil && *p != "" {
			s.PartsPresent = append(s.PartsPresent, label)
		}
	}
	add("houseNo", a.HouseNo)
	add("villageNo", a.VillageNo)
	add("alley", a.Alley)
	add("street", a.Street)
	add("subdistrict", a.Subdistrict)
	add("district", a.District)
	add("province", a.Province)
	s.ComponentCount = len(s.PartsPresent)
	return s
}

func summarizePortrait(p *card.Portrait) *PortraitShape {
	if p == nil || p.Data == "" {
		return &PortraitShape{Present: false}
	}
	s := &PortraitShape{
		Present:         true,
		MimeType:        p.MimeType,
		Encoding:        p.Encoding,
		Base64CharCount: len(p.Data),
	}
	raw, err := base64.StdEncoding.DecodeString(p.Data)
	if err != nil {
		s.Decode = "invalid-base64"
		return s
	}
	s.DecodedBytes = len(raw)
	s.JPEGSOI = len(raw) >= 2 && raw[0] == 0xFF && raw[1] == 0xD8
	s.JPEGEOI = len(raw) >= 2 && raw[len(raw)-2] == 0xFF && raw[len(raw)-1] == 0xD9
	return s
}
