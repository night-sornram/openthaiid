package bridge

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/night-sornram/openthaiid/internal/audit"
	"github.com/night-sornram/openthaiid/internal/card"
	"github.com/night-sornram/openthaiid/internal/pcsc"
	"github.com/night-sornram/openthaiid/internal/protocol"
	"github.com/night-sornram/openthaiid/internal/qualify"
)

func TestFillFailureAuditCodeOnly(t *testing.T) {
	tb := startTestBridgeSeed(t, true, func(mon, rdb *pcsc.SyntheticBackend) {
		mon.AddReader("r1", "SYNTHETIC ACR39U (fictional demo)")
	})
	waitReader(t, tb, "r1")

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	conn, _ := dial(t, ctx, tb.wsURL, "https://clinic.example.com")
	defer func() { _ = conn.Close(websocket.StatusNormalClosure, "") }()

	writeReq(t, ctx, conn, 1, "session.open", "", protocol.SessionOpenPayload{
		Fields: []card.Field{card.FieldCitizenID, card.FieldThaiName, card.FieldBirthDate},
	})
	token := readToken(t, ctx, conn, 1)
	writeReq(t, ctx, conn, 2, "card.read", token, protocol.CardReadPayload{})
	msg := readByID(t, ctx, conn, 2)
	errObj, _ := msg["error"].(map[string]any)
	if errObj == nil {
		t.Fatalf("expected fill error, got %v", msg)
	}
	code, _ := errObj["code"].(string)
	if code == "" {
		t.Fatalf("missing code: %v", errObj)
	}

	exp := tb.srv.auditExport()
	raw, err := json.Marshal(exp)
	if err != nil {
		t.Fatal(err)
	}
	if qualify.ContainsForbidden(string(raw)) {
		t.Fatalf("audit leaked forbidden: %s", raw)
	}
	found := false
	for _, ev := range exp.Events {
		if ev.Kind == audit.KindFillFailure && ev.Code == code {
			found = true
			if ev.Code != code {
				t.Fatalf("fill failure stored extra: %+v", ev)
			}
		}
	}
	if !found {
		t.Fatalf("missing fill.failure %s in %+v", code, exp.Events)
	}
	if bytes.Contains(raw, []byte("message")) && bytes.Contains(raw, []byte("no card")) {
		t.Fatalf("fill failure must not copy messages: %s", raw)
	}
}
