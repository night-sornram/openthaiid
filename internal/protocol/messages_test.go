package protocol

import (
	"encoding/json"
	"testing"

	"github.com/night-sornram/openthaiid/internal/card"
)

func TestValidateRequest(t *testing.T) {
	ok := Request{V: Major, ID: 1, Type: TypeSessionOpen}
	if err := ValidateRequest(ok); err != nil {
		t.Fatal(err)
	}
	if err := ValidateRequest(Request{V: 2, ID: 1, Type: TypeSessionOpen}); err == nil {
		t.Fatal("bad major must fail")
	}
	if err := ValidateRequest(Request{V: Major, ID: 0, Type: TypeSessionOpen}); err == nil {
		t.Fatal("id must be positive")
	}
	if err := ValidateRequest(Request{V: Major, ID: 1, Type: "card.exfiltrate"}); err == nil {
		t.Fatal("unknown type must fail")
	}
}

func TestMarshalEvent(t *testing.T) {
	b, err := MarshalEvent(7, "bridge-x", TypeCardInserted, map[string]string{
		"readerId": "r1",
	})
	if err != nil {
		t.Fatal(err)
	}
	var ev Event
	if err := json.Unmarshal(b, &ev); err != nil {
		t.Fatal(err)
	}
	if ev.V != Major || ev.Sequence != 7 || ev.InstanceID != "bridge-x" || ev.Type != TypeCardInserted {
		t.Fatalf("got %+v", ev)
	}
}

func TestCardReadPayloadRoundTrip(t *testing.T) {
	raw := `{"readerId":"r1","cardSessionId":"cs_1","fields":["thaiName"]}`
	var p CardReadPayload
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		t.Fatal(err)
	}
	if p.ReaderID != "r1" || len(p.Fields) != 1 || p.Fields[0] != card.FieldThaiName {
		t.Fatalf("got %+v", p)
	}
}

func TestProtocolMinorIsAdditive(t *testing.T) {
	if Major != 1 {
		t.Fatalf("major %d", Major)
	}
	if Minor != 3 {
		t.Fatalf("minor %d want 3 (portrait field)", Minor)
	}
}
