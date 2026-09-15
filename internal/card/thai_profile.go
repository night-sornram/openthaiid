package card

import (
	"bytes"
	"encoding/hex"
	"fmt"
)

// ThaiCitizenMVP is the reviewed community-reference profile for the three
// MVP fields. Command bytes and offsets follow the published APDU table in
// docs/provenance.md (dotnetthailand/ThaiNationalIDCard APDU.md). The
// implementation is independent; no upstream source was copied.
//
// READ BINARY command form (from the cited table):
//
//	CLA=0x80 INS=0xB0 P1=off_hi P2=off_lo Lc=0x02 Data=0x00 Le=length
//
// Field memory map (READ BINARY offsets):
//
//	citizenId   0x0004  13 bytes, ASCII digits
//	thaiName    0x0011  100 bytes, Windows-874 / TIS-620, '#'-delimited
//	birthDate   0x00D9  8 bytes, YYYYMMDD Buddhist Era
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

// Named offsets used by this profile.
const (
	offCitizenID = 0x0004
	lenCitizenID = 0x0D
	offThaiName  = 0x0011
	lenThaiName  = 0x64
	offBirthDate = 0x00D9
	lenBirthDate = 0x08
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
	if want[FieldBirthDate] {
		cmds = append(cmds, readBinary("BIRTH", offBirthDate, lenBirthDate)...)
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
		raw, ok := bodies["CID@0x0004"]
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
		raw, ok := bodies["TH_NAME@0x0011"]
		if !ok || len(raw) == 0 {
			warns = append(warns, Warning{Field: FieldThaiName, Code: "TRUNCATED", Message: "thai name truncated"})
		} else {
			name, w := parseThaiNameBlock(raw)
			if w != nil {
				warns = append(warns, *w)
			}
			if name != nil {
				vals.ThaiName = name
			}
		}
	}

	if containsField(fields, FieldBirthDate) {
		raw, ok := bodies["BIRTH@0x00D9"]
		if !ok || len(raw) < lenBirthDate {
			warns = append(warns, Warning{Field: FieldBirthDate, Code: "TRUNCATED", Message: "birth date truncated"})
		} else {
			d, w := parseBirthDate(raw[:lenBirthDate])
			if w != nil {
				warns = append(warns, *w)
			}
			if d != nil {
				vals.BirthDate = d
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
