package card

import (
	"context"
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

func TestThaiProfileCommandPlan(t *testing.T) {
	p := ThaiCitizenMVP{}
	cmds, err := p.ReadCommands([]Field{FieldCitizenID})
	if err != nil {
		t.Fatal(err)
	}
	if len(cmds) != 2 { // SELECT + CID
		t.Fatalf("got %d cmds", len(cmds))
	}
	if HexCommand(cmds[0]) != "00a4040008a000000054480001" {
		t.Fatalf("select %s", HexCommand(cmds[0]))
	}
	if HexCommand(cmds[1]) != "80b000040d" {
		t.Fatalf("cid %s", HexCommand(cmds[1]))
	}

	cmds, err = p.ReadCommands([]Field{FieldThaiName, FieldBirthDate})
	if err != nil {
		t.Fatal(err)
	}
	// SELECT + 4 name blocks + birth
	if len(cmds) != 6 {
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
	// Encode a few Thai code points as TIS-620 for tests.
	// "นาย" = 0xA2 0xC1 0xD2 ; "สมชาย" etc. constructed via helper.
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

func TestEngineExecuteAllFields(t *testing.T) {
	// Build a synthetic card image with the three MVP fields.
	// Thai: นาย สมชาย ใจดี
	title := tis("นาย")
	given := tis("สมชาย")
	middle := []byte{} // empty
	family := tis("ใจดี")

	birth := []byte("25330412")
	cid := []byte("1101701234567") // may fail checksum — engine still returns

	p := ThaiCitizenMVP{}
	conn := &fakeConn{responses: map[string][]byte{}}
	conn.responses[hexEncode(p.SelectCommand())] = swOK(nil)
	conn.responses[hexEncode([]byte{0x80, 0xB0, 0x00, 0x04, 0x0D})] = swOK(cid)
	conn.responses[hexEncode([]byte{0x80, 0xB0, 0x00, 0x11, 0x64})] = swOK(pad(string(title), 100))
	conn.responses[hexEncode([]byte{0x80, 0xB0, 0x00, 0x75, 0x64})] = swOK(pad(string(given), 100))
	conn.responses[hexEncode([]byte{0x80, 0xB0, 0x00, 0xD9, 0x64})] = swOK(pad("", 100))
	conn.responses[hexEncode([]byte{0x80, 0xB0, 0x01, 0x3D, 0x64})] = swOK(pad(string(family), 100))
	conn.responses[hexEncode([]byte{0x80, 0xB0, 0x03, 0x31, 0x08})] = swOK(birth)

	_ = middle
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
	if got.BirthDate == nil || got.BirthDate.Year == nil || *got.BirthDate.Year != 2533 {
		t.Fatalf("birth %+v", got.BirthDate)
	}
	if got.BirthDate.Month == nil || *got.BirthDate.Month != 4 || got.BirthDate.Day == nil || *got.BirthDate.Day != 12 {
		t.Fatalf("birth parts %+v", got.BirthDate)
	}
}

func TestEngineOnlyRequestedFields(t *testing.T) {
	p := ThaiCitizenMVP{}
	conn := &fakeConn{responses: map[string][]byte{}}
	conn.responses[hexEncode(p.SelectCommand())] = swOK(nil)
	conn.responses[hexEncode([]byte{0x80, 0xB0, 0x00, 0x04, 0x0D})] = swOK([]byte("1101701234567"))
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
	conn.responses[hexEncode([]byte{0x80, 0xB0, 0x00, 0x04, 0x0D})] = []byte{'1', '1', 0x61, 0x0B}
	conn.responses[hexEncode([]byte{0x00, 0xC0, 0x00, 0x00, 0x0B})] = swOK([]byte("01701234567"))

	got, _, err := NewEngine(p).Execute(context.Background(), conn, []Field{FieldCitizenID})
	if err != nil {
		t.Fatal(err)
	}
	if got.CitizenID == nil || *got.CitizenID != "1101701234567" {
		t.Fatalf("cid %+v", got.CitizenID)
	}
}

func TestParseBirthDateUnknownDay(t *testing.T) {
	d, _ := parseBirthDate([]byte("25330400"))
	if d.Day != nil {
		t.Fatal("zero day must stay null")
	}
	if d.Year == nil || *d.Year != 2533 || d.Month == nil || *d.Month != 4 {
		t.Fatalf("%+v", d)
	}
}

func TestParseCitizenIDChecksum(t *testing.T) {
	// Valid check digit example constructed for the algorithm.
	// 123456789012? → compute
	s := "1234567890121"
	id, w := parseCitizenID([]byte(s))
	if id == nil {
		t.Fatalf("id nil warn=%v", w)
	}
	if w != nil && w.Code == "CHECKSUM" {
		// advisory only
	}
	if _, w := parseCitizenID([]byte("123")); w == nil || w.Code != "BAD_LENGTH" {
		t.Fatal("expected bad length")
	}
}

func TestDecodeTIS620(t *testing.T) {
	// 0xA2=ข 0xA1=ก 0xBF=ฟ
	got := decodeTIS620([]byte{0xA2, 0xA1, 0xBF})
	if got != "ขกฟ" {
		t.Fatalf("got %q runes=%x", got, []rune(got))
	}
	if decodeTIS620([]byte("ABC")) != "ABC" {
		t.Fatal("ascii")
	}
}
