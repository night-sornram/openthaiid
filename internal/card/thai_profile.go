package card

import (
	"bytes"
	"encoding/hex"
	"fmt"
)

// ThaiCitizenMVP is the reviewed community-reference profile for textual
// Thai National ID fields. Command bytes and offsets follow the published
// APDU table in docs/provenance.md (dotnetthailand/ThaiNationalIDCard
// APDU.md). The implementation is independent; no upstream source was copied.
//
// READ BINARY command form (from the cited table):
//
//	CLA=0x80 INS=0xB0 P1=off_hi P2=off_lo Lc=0x02 Data=0x00 Le=length
//
// Field memory map (READ BINARY offsets):
//
//	citizenId    0x0004  13 bytes, ASCII digits
//	thaiName     0x0011  100 bytes, Windows-874 / TIS-620, '#'-delimited
//	englishName  0x0075  100 bytes, ASCII, '#'-delimited (same shape as thaiName)
//	birthDate    0x00D9  8 bytes, YYYYMMDD Buddhist Era
//	gender       0x00E1  1 byte, community convention "1"=male "2"=female
//	issuer       0x00F6  100 bytes, Windows-874 / TIS-620
//	issueDate    0x0167  8 bytes, YYYYMMDD Buddhist Era
//	expireDate   0x016F  8 bytes, YYYYMMDD Buddhist Era
//	address      0x1579  100 bytes, Windows-874 / TIS-620, '#'-delimited
//
// Portrait/photo blocks are intentionally not implemented.
type ThaiCitizenMVP struct{}

// Profile ID constants.
const (
	ProfileThaiCitizenMVP = "thai-citizen-mvp"
)

var (
	// SELECT FILE by DF name, Lc=0x08, DF = A0 00 00 00 54 48 00 01
	thaiSelectAPDU = []byte{0x00, 0xA4, 0x04, 0x00, 0x08, 0xA0, 0x00, 0x00, 0x00, 0x54, 0x48, 0x00, 0x01}

	// Community-documented ATR prefixes that often appear on Thai national
	// IDs. ATR is a hint only, never proof.
	thaiATRHints = [][]byte{
		{0x3B, 0x67},
		{0x3B, 0x8B},
	}
)

// Named offsets used by this profile (from the cited APDU table).
const (
	offCitizenID   = 0x0004
	lenCitizenID   = 0x0D
	offThaiName    = 0x0011
	lenThaiName    = 0x64
	offEnglishName = 0x0075
	lenEnglishName = 0x64
	offBirthDate   = 0x00D9
	lenBirthDate   = 0x08
	offGender      = 0x00E1
	lenGender      = 0x01
	offIssuer      = 0x00F6
	lenIssuer      = 0x64
	offIssueDate   = 0x0167
	lenIssueDate   = 0x08
	offExpireDate  = 0x016F
	lenExpireDate  = 0x08
	offAddress     = 0x1579
	lenAddress     = 0x64
)

// Command names produced by readBinary (Name@offset). Must stay in sync with Parse.
const (
	cmdCID     = "CID@0x0004"
	cmdTHName  = "TH_NAME@0x0011"
	cmdENName  = "EN_NAME@0x0075"
	cmdBirth   = "BIRTH@0x00D9"
	cmdGender  = "GENDER@0x00E1"
	cmdIssuer  = "ISSUER@0x00F6"
	cmdIssue   = "ISSUE@0x0167"
	cmdExpire  = "EXPIRE@0x016F"
	cmdAddress = "ADDRESS@0x1579"
)

// MaxReadLe is the maximum Le for a single READ BINARY.
// Thai cards commonly use Le up to 0xFF; we keep a conservative bound.
const MaxReadLe = 0xFF

func (ThaiCitizenMVP) ID() string { return ProfileThaiCitizenMVP }

func (ThaiCitizenMVP) ATRHints() [][]byte {
	out := make([][]byte, len(thaiATRHints))
	copy(out, thaiATRHints)
	return out
}

func (p ThaiCitizenMVP) MatchesATR(atr []byte) bool {
	for _, h := range p.ATRHints() {
		if len(atr) >= len(h) && bytes.Equal(atr[:len(h)], h) {
			return true
		}
	}
	return false
}

func (ThaiCitizenMVP) SelectCommand() []byte {
	out := make([]byte, len(thaiSelectAPDU))
	copy(out, thaiSelectAPDU)
	return out
}

// ReadCommands returns the bounded APDU sequence for the requested fields.
// Only blocks required for those fields are included.
func (ThaiCitizenMVP) ReadCommands(fields []Field) ([]APDU, error) {
	if _, err := NormalizeFields(fields); err != nil {
		return nil, err
	}
	want := map[Field]bool{}
	for _, f := range fields {
		want[f] = true
	}

	var cmds []APDU
	cmds = append(cmds, APDU{
		Name:        "SELECT",
		Command:     append([]byte(nil), thaiSelectAPDU...),
		ExpectSW:    []uint16{StatusOK},
		MaxResponse: 256,
	})

	if want[FieldCitizenID] {
		cmds = append(cmds, readBinary("CID", offCitizenID, lenCitizenID)...)
	}
	if want[FieldThaiName] {
		cmds = append(cmds, readBinary("TH_NAME", offThaiName, lenThaiName)...)
	}
	if want[FieldEnglishName] {
		cmds = append(cmds, readBinary("EN_NAME", offEnglishName, lenEnglishName)...)
	}
	if want[FieldBirthDate] {
		cmds = append(cmds, readBinary("BIRTH", offBirthDate, lenBirthDate)...)
	}
	if want[FieldGender] {
		cmds = append(cmds, readBinary("GENDER", offGender, lenGender)...)
	}
	if want[FieldIssuer] {
		cmds = append(cmds, readBinary("ISSUER", offIssuer, lenIssuer)...)
	}
	if want[FieldIssueDate] {
		cmds = append(cmds, readBinary("ISSUE", offIssueDate, lenIssueDate)...)
	}
	if want[FieldExpireDate] {
		cmds = append(cmds, readBinary("EXPIRE", offExpireDate, lenExpireDate)...)
	}
	if want[FieldAddress] {
		cmds = append(cmds, readBinary("ADDRESS", offAddress, lenAddress)...)
	}
	return cmds, nil
}

// readBinary builds READ BINARY APDUs covering [offset, offset+length).
// Command form matches the cited table:
//
//	CLA=0x80 INS=0xB0 P1=off_hi P2=off_lo Lc=0x02 Data=0x00 Le=chunk
//
// Lengths above 0xFF are split into additional APDUs.
func readBinary(name string, offset, length int) []APDU {
	var out []APDU
	remaining := length
	off := offset
	for remaining > 0 {
		le := remaining
		if le > MaxReadLe {
			le = MaxReadLe
		}
		cmd := []byte{
			0x80, 0xB0,
			byte(off >> 8), byte(off & 0xFF),
			0x02, 0x00,
			byte(le),
		}
		out = append(out, APDU{
			Name:        fmt.Sprintf("%s@0x%04X", name, off),
			Command:     cmd,
			ExpectSW:    []uint16{StatusOK},
			MaxResponse: le,
			Retrieve:    true,
		})
		off += le
		remaining -= le
	}
	return out
}

// Parse decodes responses into the requested fields.
// responses align 1:1 with ReadCommands(fields).
func (ThaiCitizenMVP) Parse(fields []Field, responses [][]byte) (*FieldValues, []Warning, error) {
	cmds, err := ThaiCitizenMVP{}.ReadCommands(fields)
	if err != nil {
		return nil, nil, err
	}
	if len(responses) != len(cmds) {
		return nil, nil, NewError(ErrReadFailed, "response count mismatch")
	}

	bodies := map[string][]byte{}
	for i, c := range cmds {
		bodies[c.Name] = responses[i]
	}

	vals := NewFieldValues(fields)
	var warns []Warning

	if containsField(fields, FieldCitizenID) {
		raw, ok := bodies[cmdCID]
		if !ok || len(raw) < lenCitizenID {
			warns = append(warns, Warning{Field: FieldCitizenID, Code: "TRUNCATED", Message: "citizen id truncated"})
		} else if id, w := parseCitizenID(raw[:lenCitizenID]); w != nil {
			warns = append(warns, *w)
			if id != nil {
				vals.CitizenID = id
			}
		} else {
			vals.CitizenID = id
		}
	}

	if containsField(fields, FieldThaiName) {
		raw, ok := bodies[cmdTHName]
		if !ok || len(raw) == 0 {
			warns = append(warns, Warning{Field: FieldThaiName, Code: "TRUNCATED", Message: "thai name truncated"})
		} else {
			name, w := parseNameBlock(raw, FieldThaiName)
			if w != nil {
				warns = append(warns, *w)
			}
			if name != nil {
				vals.ThaiName = name
			}
		}
	}

	if containsField(fields, FieldEnglishName) {
		raw, ok := bodies[cmdENName]
		if !ok || len(raw) == 0 {
			warns = append(warns, Warning{Field: FieldEnglishName, Code: "TRUNCATED", Message: "english name truncated"})
		} else {
			name, w := parseNameBlock(raw, FieldEnglishName)
			if w != nil {
				warns = append(warns, *w)
			}
			if name != nil {
				vals.EnglishName = name
			}
		}
	}

	if containsField(fields, FieldBirthDate) {
		raw, ok := bodies[cmdBirth]
		if !ok || len(raw) < lenBirthDate {
			warns = append(warns, Warning{Field: FieldBirthDate, Code: "TRUNCATED", Message: "birth date truncated"})
		} else {
			d, w := parseCardDate(raw[:lenBirthDate], FieldBirthDate)
			if w != nil {
				warns = append(warns, *w)
			}
			if d != nil {
				vals.BirthDate = d
			}
		}
	}

	if containsField(fields, FieldGender) {
		raw, ok := bodies[cmdGender]
		if !ok || len(raw) < lenGender {
			warns = append(warns, Warning{Field: FieldGender, Code: "TRUNCATED", Message: "gender truncated"})
		} else {
			g, w := parseGender(raw[:lenGender])
			if w != nil {
				warns = append(warns, *w)
			}
			if g != nil {
				vals.Gender = g
			}
		}
	}

	if containsField(fields, FieldIssuer) {
		raw, ok := bodies[cmdIssuer]
		if !ok || len(raw) == 0 {
			warns = append(warns, Warning{Field: FieldIssuer, Code: "TRUNCATED", Message: "issuer truncated"})
		} else {
			iss, w := parseIssuer(raw)
			if w != nil {
				warns = append(warns, *w)
			}
			if iss != nil {
				vals.Issuer = iss
			}
		}
	}

	if containsField(fields, FieldIssueDate) {
		raw, ok := bodies[cmdIssue]
		if !ok || len(raw) < lenIssueDate {
			warns = append(warns, Warning{Field: FieldIssueDate, Code: "TRUNCATED", Message: "issue date truncated"})
		} else {
			d, w := parseCardDate(raw[:lenIssueDate], FieldIssueDate)
			if w != nil {
				warns = append(warns, *w)
			}
			if d != nil {
				vals.IssueDate = d
			}
		}
	}

	if containsField(fields, FieldExpireDate) {
		raw, ok := bodies[cmdExpire]
		if !ok || len(raw) < lenExpireDate {
			warns = append(warns, Warning{Field: FieldExpireDate, Code: "TRUNCATED", Message: "expire date truncated"})
		} else {
			d, w := parseCardDate(raw[:lenExpireDate], FieldExpireDate)
			if w != nil {
				warns = append(warns, *w)
			}
			if d != nil {
				vals.ExpireDate = d
			}
		}
	}

	if containsField(fields, FieldAddress) {
		raw, ok := bodies[cmdAddress]
		if !ok || len(raw) == 0 {
			warns = append(warns, Warning{Field: FieldAddress, Code: "TRUNCATED", Message: "address truncated"})
		} else {
			addr, w := parseAddressBlock(raw)
			if w != nil {
				warns = append(warns, *w)
			}
			if addr != nil {
				vals.Address = addr
			}
		}
	}

	return vals, warns, nil
}

func containsField(fields []Field, f Field) bool {
	for _, x := range fields {
		if x == f {
			return true
		}
	}
	return false
}

// DefaultProfile returns the MVP Thai profile.
func DefaultProfile() Profile {
	return ThaiCitizenMVP{}
}

// ProfileByATR selects a profile from an ATR hint, or nil.
func ProfileByATR(atr []byte) Profile {
	var p ThaiCitizenMVP
	if p.MatchesATR(atr) {
		return p
	}
	return nil
}

// HexCommand is a test helper rendering an APDU as uppercase hex.
func HexCommand(a APDU) string {
	return hex.EncodeToString(a.Command)
}
