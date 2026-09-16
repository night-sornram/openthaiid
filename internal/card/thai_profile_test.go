package card

import (
	"context"
	"encoding/base64"
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
	wantENNameHex = "80b00075020064"
	wantBirthHex  = "80b000d9020008"
	wantGenderHex = "80b000e1020001"
	wantIssuerHex = "80b000f6020064"
	wantIssueHex  = "80b00167020008"
	wantExpireHex = "80b0016f020008"
	wantAddrHex   = "80b01579020064"
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

	cmds, err = p.ReadCommands([]Field{FieldEnglishName})
	if err != nil {
		t.Fatal(err)
	}
	if HexCommand(cmds[1]) != wantENNameHex {
		t.Fatalf("en name %s want %s (offset 0x0075)", HexCommand(cmds[1]), wantENNameHex)
	}

	cmds, err = p.ReadCommands([]Field{FieldGender})
	if err != nil {
		t.Fatal(err)
	}
	if HexCommand(cmds[1]) != wantGenderHex {
		t.Fatalf("gender %s want %s (offset 0x00E1)", HexCommand(cmds[1]), wantGenderHex)
	}

	cmds, err = p.ReadCommands([]Field{FieldIssuer})
	if err != nil {
		t.Fatal(err)
	}
	if HexCommand(cmds[1]) != wantIssuerHex {
		t.Fatalf("issuer %s want %s (offset 0x00F6)", HexCommand(cmds[1]), wantIssuerHex)
	}

	cmds, err = p.ReadCommands([]Field{FieldIssueDate})
	if err != nil {
		t.Fatal(err)
	}
	if HexCommand(cmds[1]) != wantIssueHex {
		t.Fatalf("issue %s want %s (offset 0x0167)", HexCommand(cmds[1]), wantIssueHex)
	}

	cmds, err = p.ReadCommands([]Field{FieldExpireDate})
	if err != nil {
		t.Fatal(err)
	}
	if HexCommand(cmds[1]) != wantExpireHex {
		t.Fatalf("expire %s want %s (offset 0x016F)", HexCommand(cmds[1]), wantExpireHex)
	}

	cmds, err = p.ReadCommands([]Field{FieldAddress})
	if err != nil {
		t.Fatal(err)
	}
	if HexCommand(cmds[1]) != wantAddrHex {
		t.Fatalf("address %s want %s (offset 0x1579)", HexCommand(cmds[1]), wantAddrHex)
	}

	cmds, err = p.ReadCommands(TextualFields)
	if err != nil {
		t.Fatal(err)
	}
	// SELECT + 9 textual field blocks.
	if len(cmds) != 10 {
		t.Fatalf("all textual fields should be SELECT + 9 reads, got %d", len(cmds))
	}
	got := []string{HexCommand(cmds[1]), HexCommand(cmds[2]), HexCommand(cmds[3]), HexCommand(cmds[4]), HexCommand(cmds[5]), HexCommand(cmds[6]), HexCommand(cmds[7]), HexCommand(cmds[8]), HexCommand(cmds[9])}
	want := []string{wantCIDHex, wantNameHex, wantENNameHex, wantBirthHex, wantGenderHex, wantIssuerHex, wantIssueHex, wantExpireHex, wantAddrHex}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("cmd %d %s want %s", i, got[i], want[i])
		}
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
	if got.EnglishName != nil || got.Gender != nil || got.Issuer != nil ||
		got.IssueDate != nil || got.ExpireDate != nil || got.Address != nil ||
		got.Portrait != nil {
		t.Fatal("unrequested extra textual fields present")
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

func TestParseGender(t *testing.T) {
	g, w := parseGender([]byte("1"))
	if w != nil || g == nil || g.Code != "1" || g.Value == nil || *g.Value != GenderMale {
		t.Fatalf("male %+v warn=%v", g, w)
	}
	g, w = parseGender([]byte("2"))
	if w != nil || g == nil || g.Value == nil || *g.Value != GenderFemale {
		t.Fatalf("female %+v warn=%v", g, w)
	}
	g, w = parseGender([]byte("9"))
	if g == nil || g.Code != "9" || g.Value != nil {
		t.Fatalf("unrecognized should keep code, null value: %+v", g)
	}
	if w == nil || w.Code != "UNRECOGNIZED" || w.Field != FieldGender {
		t.Fatalf("expected UNRECOGNIZED, got %v", w)
	}
	if w != nil && strings.Contains(w.Message, "9") {
		t.Fatal("warning must not echo the raw code")
	}
}

func TestParseEnglishNameBlock(t *testing.T) {
	raw := pad("Mr.#Somchai##Jaidee", 100)
	n, w := parseNameBlock(raw, FieldEnglishName)
	if w != nil {
		t.Fatalf("warn %v", w)
	}
	if n == nil || n.Title == nil || *n.Title != "Mr." ||
		n.GivenName == nil || *n.GivenName != "Somchai" ||
		n.MiddleName != nil ||
		n.FamilyName == nil || *n.FamilyName != "Jaidee" ||
		n.FullName != "Mr. Somchai Jaidee" {
		t.Fatalf("%+v", n)
	}
}

func TestParseIssuer(t *testing.T) {
	raw := pad(string(tis("เทศบาลนครนิทาน")), 100)
	s, w := parseIssuer(raw)
	if w != nil || s == nil || *s != "เทศบาลนครนิทาน" {
		t.Fatalf("issuer %+v warn=%v", s, w)
	}
}

func TestParseIssueAndExpireDates(t *testing.T) {
	d, w := parseCardDate([]byte("25600115"), FieldIssueDate)
	if w != nil {
		t.Fatalf("issue warn %v", w)
	}
	if d == nil || d.Year == nil || *d.Year != 2017 || d.Month == nil || *d.Month != 1 || d.Day == nil || *d.Day != 15 {
		t.Fatalf("issue %+v", d)
	}
	d, w = parseCardDate([]byte("25700114"), FieldExpireDate)
	if w != nil {
		t.Fatalf("expire warn %v", w)
	}
	if d == nil || d.Year == nil || *d.Year != 2027 || d.Day == nil || *d.Day != 14 {
		t.Fatalf("expire %+v", d)
	}
}

func TestParseAddressHashDelimited(t *testing.T) {
	raw := pad(string(tis("99#หมู่ที่ 1##ถนนนิทาน#ตำบลบ้านใหม่#อำเภอเมือง#จังหวัดนิทาน")), 100)
	a, w := parseAddressBlock(raw)
	if w != nil {
		t.Fatalf("warn %v", w)
	}
	if a == nil {
		t.Fatal("address nil")
	}
	if a.HouseNo == nil || *a.HouseNo != "99" {
		t.Fatalf("house %+v", a.HouseNo)
	}
	if a.VillageNo == nil || *a.VillageNo != "หมู่ที่ 1" {
		t.Fatalf("village %+v", a.VillageNo)
	}
	if a.Alley != nil {
		t.Fatalf("alley should be nil, got %+v", a.Alley)
	}
	if a.Street == nil || *a.Street != "ถนนนิทาน" {
		t.Fatalf("street %+v", a.Street)
	}
	if a.Subdistrict == nil || *a.Subdistrict != "ตำบลบ้านใหม่" {
		t.Fatalf("subdistrict %+v", a.Subdistrict)
	}
	if a.District == nil || *a.District != "อำเภอเมือง" {
		t.Fatalf("district %+v", a.District)
	}
	if a.Province == nil || *a.Province != "จังหวัดนิทาน" {
		t.Fatalf("province %+v", a.Province)
	}
	if a.FullAddress != "99 หมู่ที่ 1 ถนนนิทาน ตำบลบ้านใหม่ อำเภอเมือง จังหวัดนิทาน" {
		t.Fatalf("fullAddress %q", a.FullAddress)
	}
}

func TestParseAddressNoDelimiter(t *testing.T) {
	raw := pad(string(tis("99 หมู่ที่ 1 จังหวัดนิทาน")), 100)
	a, w := parseAddressBlock(raw)
	if w != nil {
		t.Fatalf("warn %v", w)
	}
	if a.HouseNo != nil || a.Province != nil {
		t.Fatalf("components should be nil without delimiter: %+v", a)
	}
	if a.FullAddress != "99 หมู่ที่ 1 จังหวัดนิทาน" {
		t.Fatalf("fullAddress %q", a.FullAddress)
	}
}

func TestEngineTextualFields(t *testing.T) {
	p := ThaiCitizenMVP{}
	conn := &fakeConn{responses: map[string][]byte{}}
	conn.responses[hexEncode(p.SelectCommand())] = swOK(nil)
	conn.responses[hexEncode([]byte{0x80, 0xB0, 0x00, 0x04, 0x02, 0x00, 0x0D})] = swOK([]byte("1234567890121"))
	conn.responses[hexEncode([]byte{0x80, 0xB0, 0x00, 0x11, 0x02, 0x00, 0x64})] = swOK(sampleNameBlock())
	conn.responses[hexEncode([]byte{0x80, 0xB0, 0x00, 0x75, 0x02, 0x00, 0x64})] = swOK(pad("Mr.#Somchai##Jaidee", 100))
	conn.responses[hexEncode([]byte{0x80, 0xB0, 0x00, 0xD9, 0x02, 0x00, 0x08})] = swOK([]byte("25330412"))
	conn.responses[hexEncode([]byte{0x80, 0xB0, 0x00, 0xE1, 0x02, 0x00, 0x01})] = swOK([]byte("1"))
	conn.responses[hexEncode([]byte{0x80, 0xB0, 0x00, 0xF6, 0x02, 0x00, 0x64})] = swOK(pad(string(tis("เทศบาลนครนิทาน")), 100))
	conn.responses[hexEncode([]byte{0x80, 0xB0, 0x01, 0x67, 0x02, 0x00, 0x08})] = swOK([]byte("25600115"))
	conn.responses[hexEncode([]byte{0x80, 0xB0, 0x01, 0x6F, 0x02, 0x00, 0x08})] = swOK([]byte("25700114"))
	conn.responses[hexEncode([]byte{0x80, 0xB0, 0x15, 0x79, 0x02, 0x00, 0x64})] = swOK(pad(string(tis("99#หมู่ที่ 1##ถนนนิทาน#ตำบลบ้านใหม่#อำเภอเมือง#จังหวัดนิทาน")), 100))

	got, warns, err := NewEngine(p).Execute(context.Background(), conn, TextualFields)
	if err != nil {
		t.Fatal(err)
	}
	if len(warns) != 0 {
		t.Fatalf("warns %v", warns)
	}
	if got.EnglishName == nil || got.EnglishName.GivenName == nil || *got.EnglishName.GivenName != "Somchai" {
		t.Fatalf("englishName %+v", got.EnglishName)
	}
	if got.Gender == nil || got.Gender.Value == nil || *got.Gender.Value != GenderMale {
		t.Fatalf("gender %+v", got.Gender)
	}
	if got.Issuer == nil || *got.Issuer != "เทศบาลนครนิทาน" {
		t.Fatalf("issuer %+v", got.Issuer)
	}
	if got.IssueDate == nil || got.IssueDate.Year == nil || *got.IssueDate.Year != 2017 {
		t.Fatalf("issue %+v", got.IssueDate)
	}
	if got.ExpireDate == nil || got.ExpireDate.Year == nil || *got.ExpireDate.Year != 2027 {
		t.Fatalf("expire %+v", got.ExpireDate)
	}
	if got.Address == nil || got.Address.HouseNo == nil || *got.Address.HouseNo != "99" {
		t.Fatalf("address %+v", got.Address)
	}

	raw, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	for _, key := range []string{"citizenId", "thaiName", "englishName", "birthDate", "gender", "issuer", "issueDate", "expireDate", "address"} {
		if !strings.Contains(s, `"`+key+`"`) {
			t.Fatalf("missing %s in %s", key, s)
		}
	}
}

func TestEngineDoesNotReadUnrequestedTextualFields(t *testing.T) {
	p := ThaiCitizenMVP{}
	conn := &fakeConn{responses: map[string][]byte{}}
	conn.responses[hexEncode(p.SelectCommand())] = swOK(nil)
	conn.responses[hexEncode([]byte{0x80, 0xB0, 0x00, 0xE1, 0x02, 0x00, 0x01})] = swOK([]byte("2"))

	got, _, err := NewEngine(p).Execute(context.Background(), conn, []Field{FieldGender})
	if err != nil {
		t.Fatal(err)
	}
	if got.Gender == nil || got.Gender.Value == nil || *got.Gender.Value != GenderFemale {
		t.Fatalf("gender %+v", got.Gender)
	}
	if len(conn.calls) != 2 {
		t.Fatalf("calls %v", conn.calls)
	}
	raw, _ := json.Marshal(got)
	s := string(raw)
	if !strings.Contains(s, `"gender"`) {
		t.Fatalf("missing gender in %s", s)
	}
	for _, leaked := range []string{"citizenId", "thaiName", "englishName", "birthDate", "issuer", "issueDate", "expireDate", "address", "portrait"} {
		if strings.Contains(s, leaked) {
			t.Fatalf("unrequested %s leaked into JSON: %s", leaked, s)
		}
	}
}

// Published Photo_Part1/20 … Photo_Part20/20 from APDU.md @ 4d803243715254cc93efe9040d8b1ecd85f520aa.
var wantPhotoHex = []string{
	"80b0017b0200ff",
	"80b0027a0200ff",
	"80b003790200ff",
	"80b004780200ff",
	"80b005770200ff",
	"80b006760200ff",
	"80b007750200ff",
	"80b008740200ff",
	"80b009730200ff",
	"80b00a720200ff",
	"80b00b710200ff",
	"80b00c700200ff",
	"80b00d6f0200ff",
	"80b00e6e0200ff",
	"80b00f6d0200ff",
	"80b0106c0200ff",
	"80b0116b0200ff",
	"80b0126a0200ff",
	"80b013690200ff",
	"80b014680200ff",
}

func TestThaiProfilePortraitAPDUMatchesPublishedTable(t *testing.T) {
	p := ThaiCitizenMVP{}
	cmds, err := p.ReadCommands([]Field{FieldPortrait})
	if err != nil {
		t.Fatal(err)
	}
	if len(cmds) != 1+portraitPartCount {
		t.Fatalf("portrait should be SELECT + %d parts, got %d", portraitPartCount, len(cmds))
	}
	if HexCommand(cmds[0]) != wantSelectHex {
		t.Fatalf("select %s", HexCommand(cmds[0]))
	}
	for i, want := range wantPhotoHex {
		got := HexCommand(cmds[i+1])
		if got != want {
			t.Fatalf("photo part %d/20 %s want %s", i+1, got, want)
		}
	}
}

func TestEnginePortraitJPEG(t *testing.T) {
	jpeg := FictionalPortraitJPEG()
	region := PadPortraitRegion(jpeg)
	p := ThaiCitizenMVP{}
	conn := &fakeConn{responses: map[string][]byte{}}
	conn.responses[hexEncode(p.SelectCommand())] = swOK(nil)
	for i := 0; i < portraitPartCount; i++ {
		off := portraitPartOffset(i)
		cmd := []byte{0x80, 0xB0, byte(off >> 8), byte(off & 0xFF), 0x02, 0x00, 0xFF}
		part := region[i*lenPortraitPart : (i+1)*lenPortraitPart]
		conn.responses[hexEncode(cmd)] = swOK(part)
	}

	got, warns, err := NewEngine(p).Execute(context.Background(), conn, []Field{FieldPortrait})
	if err != nil {
		t.Fatal(err)
	}
	if len(warns) != 0 {
		t.Fatalf("warns %v", warns)
	}
	if got.Portrait == nil {
		t.Fatal("portrait nil")
	}
	if got.Portrait.MimeType != PortraitMimeJPEG || got.Portrait.Encoding != PortraitEncodingBase64 {
		t.Fatalf("portrait meta %+v", got.Portrait)
	}
	if got.Portrait.Data == "" {
		t.Fatal("portrait data empty")
	}
	raw, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	if !strings.Contains(s, `"portrait"`) || !strings.Contains(s, `"image/jpeg"`) {
		t.Fatalf("json %s", s)
	}
	if strings.Contains(s, "citizenId") {
		t.Fatal("unrequested citizenId leaked")
	}

	back, err := base64.StdEncoding.DecodeString(got.Portrait.Data)
	if err != nil {
		t.Fatal(err)
	}
	if !bytesHasPrefix(back, []byte{0xFF, 0xD8}) || !bytesHasSuffix(back, []byte{0xFF, 0xD9}) {
		t.Fatal("decoded portrait is not a jpeg")
	}
	if !bytesContains(back, []byte(FictionalPortraitSentinel)) {
		t.Fatal("fictional sentinel missing from synthetic jpeg")
	}
}

func TestParsePortraitNotJPEG(t *testing.T) {
	p, w := parsePortrait(bytesRepeat(0x00, 64))
	if p != nil {
		t.Fatal("expected nil portrait")
	}
	if w == nil || w.Code != "NOT_JPEG" {
		t.Fatalf("warn %v", w)
	}
}

func bytesHasPrefix(b, pfx []byte) bool {
	if len(b) < len(pfx) {
		return false
	}
	for i := range pfx {
		if b[i] != pfx[i] {
			return false
		}
	}
	return true
}

func bytesHasSuffix(b, sfx []byte) bool {
	if len(b) < len(sfx) {
		return false
	}
	off := len(b) - len(sfx)
	for i := range sfx {
		if b[off+i] != sfx[i] {
			return false
		}
	}
	return true
}

func bytesContains(b, sub []byte) bool {
	if len(sub) == 0 {
		return true
	}
	for i := 0; i+len(sub) <= len(b); i++ {
		ok := true
		for j := range sub {
			if b[i+j] != sub[j] {
				ok = false
				break
			}
		}
		if ok {
			return true
		}
	}
	return false
}

func bytesRepeat(c byte, n int) []byte {
	out := make([]byte, n)
	for i := range out {
		out[i] = c
	}
	return out
}
