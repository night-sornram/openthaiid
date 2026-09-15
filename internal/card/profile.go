package card

// Profile describes a reviewed, fixed command set for one card family.
// Websites cannot supply bytes or offsets; only Profile implementations can.
type Profile interface {
	// ID is a stable profile identifier (e.g. "thai-citizen-mvp").
	ID() string
	// ATRHints are ATR byte prefixes that suggest this profile.
	// A match is a hint only, not proof of card identity or authenticity.
	ATRHints() [][]byte
	// MatchesATR reports whether atr is compatible with this profile.
	MatchesATR(atr []byte) bool
	// SelectCommand returns the APDU that selects the application.
	SelectCommand() []byte
	// ReadCommands returns the bounded APDU sequence needed for fields.
	ReadCommands(fields []Field) ([]APDU, error)
	// Parse turns a successful command trace into normalized fields.
	Parse(fields []Field, responses [][]byte) (*FieldValues, []Warning, error)
}

// APDU is an application protocol data unit sent to the card.
// Lengths and offsets come from the profile, never from the browser.
type APDU struct {
	Name    string
	Command []byte
	// ExpectSW lists accepted ISO 7816 status words for this step.
	ExpectSW []uint16
	// MaxResponse bounds the response buffer for this step.
	MaxResponse int
	// Retrieve is true when the profile uses GET RESPONSE after 61xx.
	Retrieve bool
}

// StatusOK is 0x9000.
const StatusOK uint16 = 0x9000

// SW extracts the status word from a response body.
func SW(resp []byte) (uint16, bool) {
	if len(resp) < 2 {
		return 0, false
	}
	return uint16(resp[len(resp)-2])<<8 | uint16(resp[len(resp)-1]), true
}

// IsSWMatch reports whether sw is acceptable for the APDU.
func IsSWMatch(a APDU, sw uint16) bool {
	if len(a.ExpectSW) == 0 {
		return sw == StatusOK
	}
	for _, want := range a.ExpectSW {
		if want == sw {
			return true
		}
	}
	return false
}
