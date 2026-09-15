package card

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

type fakeConn struct {
	// keyed by full command hex
	responses map[string][]byte
	calls     []string
}

func (f *fakeConn) Transmit(ctx context.Context, command []byte, maxResp int) ([]byte, error) {
	key := hexEncode(command)
	f.calls = append(f.calls, key)
	if r, ok := f.responses[key]; ok {
		return append([]byte(nil), r...), nil
	}
	return nil, NewError(ErrReadFailed, "no scripted response")
}

func hexEncode(b []byte) string {
	const digits = "0123456789abcdef"
	out := make([]byte, len(b)*2)
	for i, c := range b {
		out[i*2] = digits[c>>4]
		out[i*2+1] = digits[c&0x0F]
	}
	return string(out)
}

// Documented command bytes from dotnetthailand/ThaiNationalIDCard APDU.md
// (immutable revision 4d803243715254cc93efe9040d8b1ecd85f520aa).
// Form: CLA INS P1 P2 Lc Data Le
const (
	wantSelectHex = "00a4040008a000000054480001"
	wantCIDHex    = "80b0000402000d"
	wantNameHex   = "80b00011020064"
	wantBirthHex  = "80b000d9020008"
)

func TestThaiProfileCommandPlanMatchesPublishedAPDU(t *testing.T) {
	p := ThaiCitizenMVP{}

	cmds, err := p.ReadCommands([]Field{FieldCitizenID})
	if err != nil {
		t.Fatal(err)
	}
	if len(cmds) != 2 {
		t.Fatalf("got %d cmds", len(cmds))
	}
	if HexCommand(cmds[0]) != wantSelectHex {
		t.Fatalf("select %s want %s", HexCommand(cmds[0]), wantSelectHex)
	}
	if HexCommand(cmds[1]) != wantCIDHex {
		t.Fatalf("cid %s want %s (Lc=02 Data=00 required)", HexCommand(cmds[1]), wantCIDHex)
	}

	cmds, err = p.ReadCommands([]Field{FieldThaiName})
	if err != nil {
		t.Fatal(err)
	}
	if len(cmds) != 2 {
		t.Fatalf("thai name should be SELECT + 1 block, got %d", len(cmds))
	}
	if HexCommand(cmds[1]) != wantNameHex {
		t.Fatalf("name %s want %s (single block at 0x0011)", HexCommand(cmds[1]), wantNameHex)
	}

	cmds, err = p.ReadCommands([]Field{FieldBirthDate})
	if err != nil {
		t.Fatal(err)
	}
	if HexCommand(cmds[1]) != wantBirthHex {
		t.Fatalf("birth %s want %s (offset 0x00D9)", HexCommand(cmds[1]), wantBirthHex)
	}

	cmds, err = p.ReadCommands([]Field{FieldThaiName, FieldBirthDate})
	if err != nil {
		t.Fatal(err)
	}
	if len(cmds) != 3 {
		t.Fatalf("got %d cmds", len(cmds))
	}
}

func pad(s string, n int) []byte {
	b := make([]byte, n)
	copy(b, []byte(s))
	for i := len(s); i < n; i++ {
		b[i] = 0x20
	}
	return b
}

func tis(s string) []byte {
	var out []byte
	for _, r := range s {
		if r < 0x80 {
			out = append(out, byte(r))
			continue
		}
		if r >= 0x0E01 && r <= 0x0E5B {
			out = append(out, byte(0xA1+(r-0x0E01)))
			continue
		}
		out = append(out, '?')
	}
	return out
}

func swOK(body []byte) []byte {
	return append(append([]byte(nil), body...), 0x90, 0x00)
}

// Independently specified name block (not derived from implementation constants).
// Thai: title นาย, given สมชาย, empty middle, family ใจดี
func sampleNameBlock() []byte {
	// "นาย#สมชาย##ใจดี" encoded as TIS-620
	block := append(tis("นาย"), '#')
	block = append(block, tis("สมชาย")...)
	block = append(block, '#', '#')
	block = append(block, tis("ใจดี")...)
	return pad(string(block), 100)
}

func TestEngineExecuteAllFields(t *testing.T) {
	cid := []byte("1101701234567") // may fail checksum — engine still returns
	birth := []byte("25330412")    // BE → CE 1990-04-12

	p := ThaiCitizenMVP{}
	conn := &fakeConn{responses: map[string][]byte{}}
	conn.responses[hexEncode(p.SelectCommand())] = swOK(nil)
	conn.responses[hexEncode([]byte{0x80, 0xB0, 0x00, 0x04, 0x02, 0x00, 0x0D})] = swOK(cid)
	conn.responses[hexEncode([]byte{0x80, 0xB0, 0x00, 0x11, 0x02, 0x00, 0x64})] = swOK(sampleNameBlock())
	conn.responses[hexEncode([]byte{0x80, 0xB0, 0x00, 0xD9, 0x02, 0x00, 0x08})] = swOK(birth)

	got, warns, err := NewEngine(p).Execute(context.Background(), conn, []Field{FieldCitizenID, FieldThaiName, FieldBirthDate})
	if err != nil {
		t.Fatal(err)
	}
	if got.CitizenID == nil || *got.CitizenID != "1101701234567" {
		t.Fatalf("cid %+v warn=%v", got.CitizenID, warns)
	}
	if got.ThaiName == nil {
		t.Fatal("name nil")
	}
	if got.ThaiName.Title == nil || *got.ThaiName.Title != "นาย" {
		t.Fatalf("title %+v", got.ThaiName)
	}
	if got.ThaiName.GivenName == nil || *got.ThaiName.GivenName != "สมชาย" {
		t.Fatalf("given %+v", got.ThaiName)
	}
	if got.ThaiName.MiddleName != nil {
		t.Fatalf("middle should be nil, got %+v", *got.ThaiName.MiddleName)
	}
	if got.ThaiName.FamilyName == nil || *got.ThaiName.FamilyName != "ใจดี" {
		t.Fatalf("family %+v", got.ThaiName)
	}
	if !strings.Contains(got.ThaiName.FullName, "สมชาย") {
		t.Fatalf("fullName %q", got.ThaiName.FullName)
	}
	// Gregorian year, not Buddhist Era.
	if got.BirthDate == nil || got.BirthDate.Year == nil || *got.BirthDate.Year != 1990 {
		t.Fatalf("birth year %+v want 1990", got.BirthDate)
	}
	if got.BirthDate.Month == nil || *got.BirthDate.Month != 4 || got.BirthDate.Day == nil || *got.BirthDate.Day != 12 {
		t.Fatalf("birth parts %+v", got.BirthDate)
	}
}

func TestEngineOnlyRequestedFields(t *testing.T) {
	p := ThaiCitizenMVP{}
	conn := &fakeConn{responses: map[string][]byte{}}
	conn.responses[hexEncode(p.SelectCommand())] = swOK(nil)
	conn.responses[hexEncode([]byte{0x80, 0xB0, 0x00, 0x04, 0x02, 0x00, 0x0D})] = swOK([]byte("1101701234567"))
	// Intentionally do not script name/birth — they must not be requested.

	got, _, err := NewEngine(p).Execute(context.Background(), conn, []Field{FieldCitizenID})
	if err != nil {
		t.Fatal(err)
	}
	if got.ThaiName != nil || got.BirthDate != nil {
		t.Fatal("unrequested fields present")
	}
	if len(conn.calls) != 2 {
		t.Fatalf("calls %v", conn.calls)
	}
	raw, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	if !strings.Contains(s, `"citizenId"`) {
		t.Fatalf("missing citizenId in %s", s)
	}
	if strings.Contains(s, "thaiName") || strings.Contains(s, "birthDate") {
		t.Fatalf("unrequested fields leaked into JSON: %s", s)
	}
}

func TestRequestedUnavailableFieldIsNullWithWarning(t *testing.T) {
	p := ThaiCitizenMVP{}
	conn := &fakeConn{responses: map[string][]byte{}}
	conn.responses[hexEncode(p.SelectCommand())] = swOK(nil)
	// Truncated citizen id → warning + null field.
	conn.responses[hexEncode([]byte{0x80, 0xB0, 0x00, 0x04, 0x02, 0x00, 0x0D})] = swOK([]byte("123"))
	conn.responses[hexEncode([]byte{0x80, 0xB0, 0x00, 0x11, 0x02, 0x00, 0x64})] = swOK(pad("", 100))
	conn.responses[hexEncode([]byte{0x80, 0xB0, 0x00, 0xD9, 0x02, 0x00, 0x08})] = swOK([]byte("00000000"))

	got, warns, err := NewEngine(p).Execute(context.Background(), conn, []Field{FieldCitizenID, FieldThaiName, FieldBirthDate})
	if err != nil {
		t.Fatal(err)
	}
	if got.CitizenID != nil || got.ThaiName != nil || got.BirthDate != nil {
		t.Fatalf("expected all nil, got %+v", got)
	}
	if len(warns) < 2 {
		t.Fatalf("expected warnings, got %v", warns)
	}
	raw, _ := json.Marshal(got)
	s := string(raw)
	if !strings.Contains(s, `"citizenId":null`) || !strings.Contains(s, `"thaiName":null`) || !strings.Contains(s, `"birthDate":null`) {
		t.Fatalf("requested fields must be null, got %s", s)
	}
}

func TestEngineUnsupportedCard(t *testing.T) {
	p := ThaiCitizenMVP{}
	conn := &fakeConn{responses: map[string][]byte{}}
	conn.responses[hexEncode(p.SelectCommand())] = []byte{0x6A, 0x82}
	_, _, err := NewEngine(p).Execute(context.Background(), conn, []Field{FieldCitizenID})
	if err == nil || AsError(err).Code != ErrUnsupportedCard {
		t.Fatalf("err %v", err)
	}
}

func TestEngineGetResponse(t *testing.T) {
	p := ThaiCitizenMVP{}
	conn := &fakeConn{responses: map[string][]byte{}}
	conn.responses[hexEncode(p.SelectCommand())] = swOK(nil)
	conn.responses[hexEncode([]byte{0x80, 0xB0, 0x00, 0x04, 0x02, 0x00, 0x0D})] = []byte{'1', '1', 0x61, 0x0B}
	conn.responses[hexEncode([]byte{0x00, 0xC0, 0x00, 0x00, 0x0B})] = swOK([]byte("01701234567"))

	got, _, err := NewEngine(p).Execute(context.Background(), conn, []Field{FieldCitizenID})
	if err != nil {
		t.Fatal(err)
	}
	if got.CitizenID == nil || *got.CitizenID != "1101701234567" {
		t.Fatalf("cid %+v", got.CitizenID)
	}
}

func TestEngineRejectsOversizedResponse(t *testing.T) {
	p := ThaiCitizenMVP{}
	conn := &fakeConn{responses: map[string][]byte{}}
	conn.responses[hexEncode(p.SelectCommand())] = swOK(nil)
	// Return more than Le=0x0D bytes for citizen id.
	conn.responses[hexEncode([]byte{0x80, 0xB0, 0x00, 0x04, 0x02, 0x00, 0x0D})] = swOK(make([]byte, 32))
	_, _, err := NewEngine(p).Execute(context.Background(), conn, []Field{FieldCitizenID})
	if err == nil || AsError(err).Code != ErrReadFailed {
		t.Fatalf("expected oversized rejection, got %v", err)
	}
}

func TestParseBirthDateGregorianAndUnknownDay(t *testing.T) {
	d, _ := parseBirthDate([]byte("25330400"))
	if d.Day != nil {
		t.Fatal("zero day must stay null")
	}
	if d.Year == nil || *d.Year != 1990 || d.Month == nil || *d.Month != 4 {
		t.Fatalf("%+v", d)
	}
}

func TestParseBirthDateLeapYear(t *testing.T) {
	// 2527-02-29 BE → 1984-02-29 CE (leap)
	d, w := parseBirthDate([]byte("25270229"))
	if w != nil {
		t.Fatalf("unexpected warning %v", w)
	}
	if d == nil || d.Year == nil || *d.Year != 1984 || d.Day == nil || *d.Day != 29 {
		y, mo, dy := -1, -1, -1
		if d != nil {
			if d.Year != nil {
				y = *d.Year
			}
			if d.Month != nil {
				mo = *d.Month
			}
			if d.Day != nil {
				dy = *d.Day
			}
		}
		t.Fatalf("y=%d m=%d d=%d", y, mo, dy)
	}
	// 2528-02-29 BE → 1985-02-29 CE (not leap)
	d, w = parseBirthDate([]byte("25280229"))
	if d != nil && d.Day != nil {
		t.Fatalf("non-leap Feb 29 must be null, got day=%d", *d.Day)
	}
	if w == nil || w.Code != "BAD_DAY" {
		t.Fatalf("expected BAD_DAY, got %v", w)
	}
	if d == nil || d.Year == nil || *d.Year != 1985 {
		t.Fatalf("year should survive day rejection, got %+v", d)
	}
}

func TestParseThaiNameBlockHashDelimiters(t *testing.T) {
	// "นาย#สมชาย#วร#ใจดี"
	raw := pad(string(tis("นาย#สมชาย#วร#ใจดี")), 100)
	n, w := parseThaiNameBlock(raw)
	if w != nil {
		t.Fatalf("warn %v", w)
	}
	if n == nil || n.Title == nil || *n.Title != "นาย" ||
		n.GivenName == nil || *n.GivenName != "สมชาย" ||
		n.MiddleName == nil || *n.MiddleName != "วร" ||
		n.FamilyName == nil || *n.FamilyName != "ใจดี" {
		t.Fatalf("%+v", n)
	}
	if n.FullName != "นาย สมชาย วร ใจดี" {
		t.Fatalf("fullName %q", n.FullName)
	}
}

func TestParseThaiNameBlockTrailingDelimiter(t *testing.T) {
	raw := pad(string(tis("นาย#สมชาย##ใจดี#")), 100)
	n, _ := parseThaiNameBlock(raw)
	if n.MiddleName != nil {
		t.Fatalf("middle %+v", n.MiddleName)
	}
	if n.FamilyName == nil || *n.FamilyName != "ใจดี" {
		t.Fatalf("family %+v", n)
	}
}

func TestParseThaiNameSpaceFallback(t *testing.T) {
	raw := pad(string(tis("นางสาว สมหญิง ใจงาม")), 100)
	n, _ := parseThaiNameBlock(raw)
	if n.Title == nil || *n.Title != "นางสาว" {
		t.Fatalf("title %+v", n)
	}
	if n.GivenName == nil || *n.GivenName != "สมหญิง" {
		t.Fatalf("given %+v", n)
	}
	if n.FamilyName == nil || *n.FamilyName != "ใจงาม" {
		t.Fatalf("family %+v", n)
	}
}

func TestParseCitizenIDChecksum(t *testing.T) {
	s := "1234567890121"
	id, w := parseCitizenID([]byte(s))
	if id == nil {
		t.Fatalf("id nil warn=%v", w)
	}
	if _, w := parseCitizenID([]byte("123")); w == nil || w.Code != "BAD_LENGTH" {
		t.Fatal("expected bad length")
	}
}

func TestDecodeWindows874(t *testing.T) {
	// 0xA2=ข 0xA1=ก 0xBF=ฟ
	got := decodeWindows874([]byte{0xA2, 0xA1, 0xBF})
	if got != "ขกฟ" {
		t.Fatalf("got %q runes=%x", got, []rune(got))
	}
	if decodeWindows874([]byte("ABC")) != "ABC" {
		t.Fatal("ascii")
	}
}
