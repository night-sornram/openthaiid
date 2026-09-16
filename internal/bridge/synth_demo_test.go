package bridge

import (
	"context"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/night-sornram/openthaiid/internal/card"
	"github.com/night-sornram/openthaiid/internal/pcsc"
	"github.com/night-sornram/openthaiid/internal/protocol"
)

func TestSyntheticDemoReadAllFields(t *testing.T) {
	// Seed the demo card BEFORE manager start so the first resync sees it.
	tb := startTestBridgeSeed(t, true, func(mon, rdb *pcsc.SyntheticBackend) {
		seedFullThaiCard(rdb)
	})
	tb.waitCardReady(t, "synthetic-acr39u")

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	conn, _ := dial(t, ctx, tb.wsURL, "https://clinic.example.com")
	defer func() { _ = conn.Close(websocket.StatusNormalClosure, "") }()

	writeReq(t, ctx, conn, 1, "session.open", "", protocol.SessionOpenPayload{
		Fields: []card.Field{card.FieldCitizenID, card.FieldThaiName, card.FieldBirthDate},
	})
	token := readToken(t, ctx, conn, 1)

	writeReq(t, ctx, conn, 2, "card.read", token, protocol.CardReadPayload{
		Fields: []card.Field{card.FieldCitizenID, card.FieldThaiName, card.FieldBirthDate},
	})
	for i := 0; i < 10; i++ {
		msg := readMsg(t, ctx, conn)
		if msg["id"] == nil || int64(msg["id"].(float64)) != 2 {
			continue
		}
		if msg["error"] != nil {
			t.Fatalf("read error: %v", msg["error"])
		}
		result := msg["result"].(map[string]any)
		fields := result["fields"].(map[string]any)
		if fields["citizenId"] != "1234567890121" {
			t.Fatalf("cid %v", fields["citizenId"])
		}
		name, _ := fields["thaiName"].(map[string]any)
		if name == nil || name["givenName"] != "สมชาย" {
			t.Fatalf("name %v", name)
		}
		if name["fullName"] != "นาย สมชาย ใจดี" {
			t.Fatalf("fullName %v", name["fullName"])
		}
		birth, _ := fields["birthDate"].(map[string]any)
		if birth == nil || birth["year"] != 1990.0 || birth["month"] != 4.0 || birth["day"] != 12.0 {
			t.Fatalf("birth %v (expected 1990-04-12)", birth)
		}
		// Extra textual fields must stay omitted when not requested.
		for _, leaked := range []string{"englishName", "gender", "issuer", "issueDate", "expireDate", "address", "portrait"} {
			if _, ok := fields[leaked]; ok {
				t.Fatalf("unrequested %s returned", leaked)
			}
		}
		return
	}
	t.Fatal("no read response")
}

func TestSyntheticDemoReadTextualFields(t *testing.T) {
	tb := startTestBridgeSeed(t, true, func(mon, rdb *pcsc.SyntheticBackend) {
		seedFullThaiCard(rdb)
	})
	tb.waitCardReady(t, "synthetic-acr39u")

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	conn, _ := dial(t, ctx, tb.wsURL, "https://clinic.example.com")
	defer func() { _ = conn.Close(websocket.StatusNormalClosure, "") }()

	writeReq(t, ctx, conn, 1, "session.open", "", protocol.SessionOpenPayload{
		Fields: card.SupportedFields,
	})
	open := readByID(t, ctx, conn, 1)
	if open["error"] != nil {
		t.Fatalf("session.open error: %v", open["error"])
	}
	openResult := open["result"].(map[string]any)
	token, _ := openResult["token"].(string)
	if token == "" {
		t.Fatal("missing token")
	}
	proto, _ := openResult["protocol"].(map[string]any)
	if proto["major"] != 1.0 || proto["minor"] != 4.0 {
		t.Fatalf("protocol %+v want major=1 minor=4", proto)
	}
	supported, _ := openResult["fieldsSupported"].([]any)
	if len(supported) != len(card.SupportedFields) {
		t.Fatalf("fieldsSupported %v", supported)
	}
	foundEN := false
	foundPortrait := false
	for _, f := range supported {
		if f == string(card.FieldEnglishName) {
			foundEN = true
		}
		if f == string(card.FieldPortrait) {
			foundPortrait = true
		}
	}
	if !foundEN {
		t.Fatalf("fieldsSupported missing englishName: %v", supported)
	}
	if !foundPortrait {
		t.Fatalf("fieldsSupported missing portrait: %v", supported)
	}

	writeReq(t, ctx, conn, 2, "card.read", token, protocol.CardReadPayload{
		Fields: card.SupportedFields,
	})
	msg := readByID(t, ctx, conn, 2)
	if msg["error"] != nil {
		t.Fatalf("read error: %v", msg["error"])
	}
	fields := msg["result"].(map[string]any)["fields"].(map[string]any)
	if fields["citizenId"] != "1234567890121" {
		t.Fatalf("cid %v", fields["citizenId"])
	}
	en, _ := fields["englishName"].(map[string]any)
	if en == nil || en["givenName"] != "Somchai" || en["familyName"] != "Jaidee" || en["fullName"] != "Mr. Somchai Jaidee" {
		t.Fatalf("englishName %v", en)
	}
	gender, _ := fields["gender"].(map[string]any)
	if gender == nil || gender["code"] != "1" || gender["value"] != "male" {
		t.Fatalf("gender %v", gender)
	}
	if fields["issuer"] != "เทศบาลนครนิทาน" {
		t.Fatalf("issuer %v", fields["issuer"])
	}
	issue, _ := fields["issueDate"].(map[string]any)
	if issue == nil || issue["year"] != 2017.0 || issue["month"] != 1.0 || issue["day"] != 15.0 {
		t.Fatalf("issueDate %v", issue)
	}
	expire, _ := fields["expireDate"].(map[string]any)
	if expire == nil || expire["year"] != 2027.0 || expire["day"] != 14.0 {
		t.Fatalf("expireDate %v", expire)
	}
	addr, _ := fields["address"].(map[string]any)
	if addr == nil || addr["houseNo"] != "99" || addr["alley"] != nil || addr["province"] != "จังหวัดนิทาน" {
		t.Fatalf("address %v", addr)
	}
	if addr["fullAddress"] != "99 หมู่ที่ 1 ถนนนิทาน ตำบลบ้านใหม่ อำเภอเมือง จังหวัดนิทาน" {
		t.Fatalf("fullAddress %v", addr["fullAddress"])
	}
	portrait, _ := fields["portrait"].(map[string]any)
	if portrait == nil {
		t.Fatal("portrait missing from full grant")
	}
	if portrait["mimeType"] != "image/jpeg" || portrait["encoding"] != "base64" {
		t.Fatalf("portrait meta %v", portrait)
	}
	data, _ := portrait["data"].(string)
	if data == "" || len(data) < 32 {
		t.Fatalf("portrait data missing")
	}
}

func TestSyntheticDemoReadPortraitOnly(t *testing.T) {
	tb := startTestBridgeSeed(t, true, func(mon, rdb *pcsc.SyntheticBackend) {
		seedFullThaiCard(rdb)
	})
	tb.waitCardReady(t, "synthetic-acr39u")

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	conn, _ := dial(t, ctx, tb.wsURL, "https://clinic.example.com")
	defer func() { _ = conn.Close(websocket.StatusNormalClosure, "") }()

	writeReq(t, ctx, conn, 1, "session.open", "", protocol.SessionOpenPayload{
		Fields: []card.Field{card.FieldPortrait},
	})
	token := readToken(t, ctx, conn, 1)

	writeReq(t, ctx, conn, 2, "card.read", token, protocol.CardReadPayload{
		Fields: []card.Field{card.FieldPortrait},
	})
	msg := readByID(t, ctx, conn, 2)
	if msg["error"] != nil {
		t.Fatalf("read error: %v", msg["error"])
	}
	fields := msg["result"].(map[string]any)["fields"].(map[string]any)
	portrait, _ := fields["portrait"].(map[string]any)
	if portrait == nil || portrait["mimeType"] != "image/jpeg" {
		t.Fatalf("portrait %v", portrait)
	}
	if _, ok := fields["citizenId"]; ok {
		t.Fatal("unrequested citizenId returned with portrait-only grant")
	}
}

func TestIntegrationUngrantedTextualFieldRejected(t *testing.T) {
	tb := startTestBridgeSeed(t, true, func(mon, rdb *pcsc.SyntheticBackend) {
		seedFullThaiCard(rdb)
	})
	tb.waitCardReady(t, "synthetic-acr39u")

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	conn, _ := dial(t, ctx, tb.wsURL, "https://clinic.example.com")
	defer func() { _ = conn.Close(websocket.StatusNormalClosure, "") }()

	writeReq(t, ctx, conn, 1, "session.open", "", protocol.SessionOpenPayload{
		Fields: card.DefaultFields,
	})
	token := readToken(t, ctx, conn, 1)

	writeReq(t, ctx, conn, 2, "card.read", token, protocol.CardReadPayload{
		Fields: []card.Field{card.FieldGender},
	})
	msg := readByID(t, ctx, conn, 2)
	errObj, _ := msg["error"].(map[string]any)
	if errObj == nil || errObj["code"] != "FIELD_NOT_ALLOWED" {
		t.Fatalf("expected FIELD_NOT_ALLOWED, got %v", msg["error"])
	}

	writeReq(t, ctx, conn, 3, "card.read", token, protocol.CardReadPayload{
		Fields: []card.Field{card.FieldCitizenID},
	})
	ok := readByID(t, ctx, conn, 3)
	if ok["error"] != nil {
		t.Fatalf("granted read failed: %v", ok["error"])
	}
	fields := ok["result"].(map[string]any)["fields"].(map[string]any)
	if fields["citizenId"] != "1234567890121" {
		t.Fatalf("cid %v", fields["citizenId"])
	}
	if _, present := fields["gender"]; present {
		t.Fatal("gender must not appear without a grant")
	}
}

func TestIntegrationSessionOpenFieldExpansionPrompts(t *testing.T) {
	tb := startTestBridgeSeed(t, true, func(mon, rdb *pcsc.SyntheticBackend) {
		seedFullThaiCard(rdb)
	})
	tb.waitCardReady(t, "synthetic-acr39u")

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	conn1, _ := dial(t, ctx, tb.wsURL, "https://clinic.example.com")
	defer func() { _ = conn1.Close(websocket.StatusNormalClosure, "") }()
	writeReq(t, ctx, conn1, 1, "session.open", "", protocol.SessionOpenPayload{
		Fields: card.DefaultFields,
	})
	token1 := readToken(t, ctx, conn1, 1)
	if token1 == "" {
		t.Fatal("missing default-grant token")
	}

	conn2, _ := dial(t, ctx, tb.wsURL, "https://clinic.example.com")
	defer func() { _ = conn2.Close(websocket.StatusNormalClosure, "") }()
	expanded := append(append([]card.Field(nil), card.DefaultFields...), card.FieldGender)
	writeReq(t, ctx, conn2, 1, "session.open", "", protocol.SessionOpenPayload{
		Fields: expanded,
	})

	sawApproval := false
	var open map[string]any
	deadline := time.Now().Add(4 * time.Second)
	for open == nil && time.Now().Before(deadline) {
		msg := readMsg(t, ctx, conn2)
		typ, _ := msg["type"].(string)
		if typ == "approval.required" {
			sawApproval = true
			pending := tb.authMgr.Pending(time.Now())
			if len(pending) == 0 {
				t.Fatal("approval.required without pending request")
			}
			if _, err := tb.authMgr.Resolve(pending[0].ID, true, time.Now()); err != nil {
				t.Fatalf("resolve expansion: %v", err)
			}
			continue
		}
		if msg["id"] != nil && int64(msg["id"].(float64)) == 1 {
			open = msg
		}
	}
	if !sawApproval {
		t.Fatal("session.open field expansion must prompt for a new approval")
	}
	if open == nil {
		t.Fatal("missing session.open response after expansion approval")
	}
	if open["error"] != nil {
		t.Fatalf("expansion session.open error: %v", open["error"])
	}
	token2, _ := open["result"].(map[string]any)["token"].(string)
	if token2 == "" {
		t.Fatal("missing expansion token")
	}

	writeReq(t, ctx, conn2, 2, "card.read", token2, protocol.CardReadPayload{
		Fields: []card.Field{card.FieldGender},
	})
	read := readByID(t, ctx, conn2, 2)
	if read["error"] != nil {
		t.Fatalf("expanded read error: %v", read["error"])
	}
	fields := read["result"].(map[string]any)["fields"].(map[string]any)
	gender, _ := fields["gender"].(map[string]any)
	if gender == nil || gender["code"] != "1" || gender["value"] != "male" {
		t.Fatalf("gender %v", gender)
	}

	writeReq(t, ctx, conn1, 2, "card.read", token1, protocol.CardReadPayload{
		Fields: []card.Field{card.FieldCitizenID},
	})
	prior := readByID(t, ctx, conn1, 2)
	if prior["error"] != nil {
		t.Fatalf("original grant read failed after expansion: %v", prior["error"])
	}
}
