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
		return
	}
	t.Fatal("no read response")
}
