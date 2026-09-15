package bridge

import (
	"context"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/night-sornram/openthaiid/internal/card"
	"github.com/night-sornram/openthaiid/internal/protocol"
)

func TestSyntheticDemoReadAllFields(t *testing.T) {
	wsURL, _, mon, rdb, _ := startTestBridge(t, true)
	rdb.SeedDemoCard()
	// Force the monitor loop to pick up the seeded reader/card.
	mon.AddReader("synthetic-acr39u", "SYNTHETIC ACR39U (fictional demo)")
	mon.InsertCard("synthetic-acr39u")
	time.Sleep(80 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	conn, _ := dial(t, ctx, wsURL, "https://clinic.example.com")
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
		birth, _ := fields["birthDate"].(map[string]any)
		if birth == nil || birth["year"] != 1990.0 {
			t.Fatalf("birth %v", birth)
		}
		return
	}
	t.Fatal("no read response")
}
