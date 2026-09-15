package card

import (
	"bytes"
	"encoding/hex"
	"fmt"
)

// ThaiCitizenMVP is the reviewed community-reference profile for the three
// MVP fields. Offsets follow the widely published community command tables
// (see docs/provenance.md). Implementation is independent; no upstream source
// was copied.
//
// Field memory map (READ BINARY offsets):
//
//	citizenId   0x0004  13 bytes, ASCII digits
//	thaiTitle   0x0011  100 bytes, TIS-620
//	thaiGiven   0x0075  100 bytes
//	thaiMiddle  0x00D9  100 bytes
//	thaiFamily  0x013D  100 bytes
//	birthDate   0x0331  8 bytes, YYYYMMDD Buddhist Era
type ThaiCitizenMVP struct{}

// Profile ID constants.
const (
	ProfileThaiCitizenMVP = "thai-citizen-mvp"
)

var (
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
	offCitizenID  = 0x0004
	lenCitizenID  = 13
	offThaiTitle  = 0x0011
	offThaiGiven  = 0x0075
	offThaiMiddle = 0x00D9
	offThaiFamily = 0x013D
	lenNamePart   = 100
	offBirthDate  = 0x0331
	lenBirthDate  = 8
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
	// Unknown ATRs are not rejected outright at this layer; the engine
	// rejects after SELECT fails.
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
	// SELECT is always required before any READ BINARY.
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
		cmds = append(cmds, readBinary("TH_TITLE", offThaiTitle, lenNamePart)...)
		cmds = append(cmds, readBinary("TH_GIVEN", offThaiGiven, lenNamePart)...)
		cmds = append(cmds, readBinary("TH_MIDDLE", offThaiMiddle, lenNamePart)...)
		cmds = append(cmds, readBinary("TH_FAMILY", offThaiFamily, lenNamePart)...)
	}
	if want[FieldBirthDate] {
		cmds = append(cmds, readBinary("BIRTH", offBirthDate, lenBirthDate)...)
	}
	return cmds, nil
}

// readBinary builds one or more READ BINARY APDUs covering [offset, offset+length).
// Command form: CLA=80 INS=B0 P1=off_hi P2=off_lo Le=chunk.
// Subsequent chunks are issued as additional APDUs (profile does not rely on
// proprietary chaining).
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
			byte(le),
		}
		out = append(out, APDU{
			Name:        fmt.Sprintf("%s@0x%04X", name, off),
			Command:     cmd,
			ExpectSW:    []uint16{StatusOK},
			MaxResponse: le + 2,
			Retrieve:    true, // allow 61xx + GET RESPONSE
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

	// Map command name → body (SW already stripped by the engine).
	bodies := map[string][]byte{}
	for i, c := range cmds {
		bodies[c.Name] = responses[i]
	}

	vals := &FieldValues{}
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
		title := decodeThaiField(bodies["TH_TITLE@0x0011"], lenNamePart)
		given := decodeThaiField(bodies["TH_GIVEN@0x0075"], lenNamePart)
		middle := decodeThaiField(bodies["TH_MIDDLE@0x00D9"], lenNamePart)
		family := decodeThaiField(bodies["TH_FAMILY@0x013D"], lenNamePart)
		name, w := buildThaiName(title, given, middle, family)
		if w != nil {
			warns = append(warns, *w)
		}
		if name != nil {
			vals.ThaiName = name
		}
	}

	if containsField(fields, FieldBirthDate) {
		raw, ok := bodies["BIRTH@0x0331"]
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
