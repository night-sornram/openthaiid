package bridge

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/night-sornram/openthaiid/internal/auth"
	"github.com/night-sornram/openthaiid/internal/card"
	"github.com/night-sornram/openthaiid/internal/config"
	"github.com/night-sornram/openthaiid/internal/pcsc"
	"github.com/night-sornram/openthaiid/internal/protocol"
	"github.com/night-sornram/openthaiid/internal/reader"
)

// startTestBridge starts a real bridge on an ephemeral loopback port.
func startTestBridge(t *testing.T, autoAllow bool) (url string, authMgr *auth.Manager, mon *pcsc.SyntheticBackend, rdb *pcsc.SyntheticBackend, cancel context.CancelFunc) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	_, portStr, _ := net.SplitHostPort(ln.Addr().String())
	_ = ln.Close()

	cfg := config.Default()
	fmt.Sscanf(portStr, "%d", &cfg.Port)
	cfg.ApprovalTimeout = 3 * time.Second
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	authMgr = auth.NewManager("bridge-it", cfg.TrustDuration, cfg.SessionDuration, cfg.ApprovalTimeout, cfg.MaxPendingApprovals)
	mon = pcsc.NewSynthetic()
	rdb = pcsc.NewSynthetic()
	ctx, cancel := context.WithCancel(context.Background())
	if err := mon.Establish(ctx); err != nil {
		t.Fatal(err)
	}
	if err := rdb.Establish(ctx); err != nil {
		t.Fatal(err)
	}
	mgr := reader.NewManager(mon, rdb)
	if err := mgr.Start(ctx); err != nil {
		t.Fatal(err)
	}
	s := New(cfg, log, authMgr, mgr)
	if err := s.Start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cancel()
		_ = s.Shutdown(context.Background())
		_ = mgr.Stop(context.Background())
	})
	wsURL := fmt.Sprintf("ws://127.0.0.1:%d/v1/ws", cfg.Port)

	if autoAllow {
		go func() {
			deadline := time.Now().Add(3 * time.Second)
			for time.Now().Before(deadline) {
				pending := authMgr.Pending(time.Now())
				if len(pending) > 0 {
					_, _ = authMgr.Resolve(pending[0].ID, true, time.Now())
					return
				}
				time.Sleep(10 * time.Millisecond)
			}
		}()
	}

	return wsURL, authMgr, mon, rdb, cancel
}

func scriptThaiCard(rdb *pcsc.SyntheticBackend) {
	p := card.ThaiCitizenMVP{}
	rdb.SetScriptResponse(string(p.SelectCommand()), append([]byte(nil), 0x90, 0x00))
	rdb.SetScriptResponse(string([]byte{0x80, 0xB0, 0x00, 0x04, 0x0D}), append([]byte("1101701234567"), 0x90, 0x00))
	// Minimal: only CID for this test.
}

func dial(t *testing.T, ctx context.Context, wsURL, origin string) (*websocket.Conn, *http.Response) {
	t.Helper()
	conn, resp, err := websocket.Dial(ctx, wsURL, &websocket.DialOptions{
		Subprotocols: []string{protocol.Subprotocol},
		HTTPHeader:   http.Header{"Origin": []string{origin}},
	})
	if err != nil {
		t.Fatalf("dial: %v (status %v)", err, resp)
	}
	return conn, resp
}

func readMsg(t *testing.T, ctx context.Context, c *websocket.Conn) map[string]any {
	t.Helper()
	_, data, err := c.Read(ctx)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal %s: %v", data, err)
	}
	return m
}

func writeReq(t *testing.T, ctx context.Context, c *websocket.Conn, id int64, typ string, token string, payload any) {
	t.Helper()
	raw, _ := json.Marshal(payload)
	msg := map[string]any{
		"v":    protocol.Major,
		"id":   id,
		"type": typ,
	}
	if token != "" {
		msg["token"] = token
	}
	if payload != nil {
		msg["payload"] = json.RawMessage(raw)
	}
	body, err := json.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Write(ctx, websocket.MessageText, body); err != nil {
		t.Fatal(err)
	}
}

func TestIntegrationFullReadFlow(t *testing.T) {
	wsURL, _, mon, rdb, _ := startTestBridge(t, true)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mon.AddReader("r1", "ACR39U")
	mon.InsertCard("r1")
	rdb.AddReader("r1", "ACR39U")
	rdb.InsertCard("r1")
	scriptThaiCard(rdb)

	// Give the manager loop a moment to resync.
	time.Sleep(50 * time.Millisecond)

	conn, _ := dial(t, ctx, wsURL, "https://clinic.example.com")
	defer func() { _ = conn.Close(websocket.StatusNormalClosure, "") }()

	writeReq(t, ctx, conn, 1, "session.open", "", protocol.SessionOpenPayload{
		Fields: []card.Field{card.FieldCitizenID},
	})

	token := ""
	for i := 0; i < 5; i++ {
		msg := readMsg(t, ctx, conn)
		if msg["type"] == "approval.required" || msg["type"] == "status.snapshot" {
			continue
		}
		if msg["id"] == nil {
			continue
		}
		if int64(msg["id"].(float64)) != 1 {
			continue
		}
		if msg["error"] != nil {
			t.Fatalf("session.open error: %v", msg["error"])
		}
		result := msg["result"].(map[string]any)
		token, _ = result["token"].(string)
		break
	}
	if token == "" {
		t.Fatal("no token after approval")
	}

	writeReq(t, ctx, conn, 2, "card.read", token, protocol.CardReadPayload{
		Fields: []card.Field{card.FieldCitizenID},
	})

	for i := 0; i < 6; i++ {
		msg := readMsg(t, ctx, conn)
		if msg["id"] == nil || int64(msg["id"].(float64)) != 2 {
			continue
		}
		if msg["error"] != nil {
			t.Fatalf("card.read error: %v", msg["error"])
		}
		result := msg["result"].(map[string]any)
		fields := result["fields"].(map[string]any)
		if fields["citizenId"] != "1101701234567" {
			t.Fatalf("citizenId %v", fields["citizenId"])
		}
		// Unrequested fields must be absent.
		if _, ok := fields["thaiName"]; ok {
			t.Fatal("thaiName should not be returned")
		}
		return
	}
	t.Fatal("no card.read response")
}

func TestIntegrationBadOriginRejected(t *testing.T) {
	wsURL, _, _, _, _ := startTestBridge(t, false)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, resp, err := websocket.Dial(ctx, wsURL, &websocket.DialOptions{
		Subprotocols: []string{protocol.Subprotocol},
		HTTPHeader:   http.Header{"Origin": []string{"http://evil.example.com"}},
	})
	if err == nil {
		t.Fatal("expected dial failure")
	}
	if resp != nil && resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status %d", resp.StatusCode)
	}
}

func TestIntegrationDenyApproval(t *testing.T) {
	wsURL, authMgr, _, _, _ := startTestBridge(t, false)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) {
			pending := authMgr.Pending(time.Now())
			if len(pending) > 0 {
				_, _ = authMgr.Resolve(pending[0].ID, false, time.Now())
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
	}()

	conn, _ := dial(t, ctx, wsURL, "https://clinic.example.com")
	defer func() { _ = conn.Close(websocket.StatusNormalClosure, "") }()
	writeReq(t, ctx, conn, 1, "session.open", "", protocol.SessionOpenPayload{
		Fields: []card.Field{card.FieldCitizenID},
	})
	for i := 0; i < 5; i++ {
		msg := readMsg(t, ctx, conn)
		if msg["id"] == nil || int64(msg["id"].(float64)) != 1 {
			continue
		}
		errObj, ok := msg["error"].(map[string]any)
		if !ok {
			t.Fatalf("expected error, got %v", msg)
		}
		if errObj["code"] != "APPROVAL_DENIED" {
			t.Fatalf("code %v", errObj["code"])
		}
		return
	}
	t.Fatal("no response")
}

func TestIntegrationNoFieldExpansion(t *testing.T) {
	wsURL, _, mon, rdb, _ := startTestBridge(t, true)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	mon.AddReader("r1", "A")
	mon.InsertCard("r1")
	rdb.AddReader("r1", "A")
	rdb.InsertCard("r1")
	scriptThaiCard(rdb)
	time.Sleep(30 * time.Millisecond)

	conn, _ := dial(t, ctx, wsURL, "https://clinic.example.com")
	defer func() { _ = conn.Close(websocket.StatusNormalClosure, "") }()
	writeReq(t, ctx, conn, 1, "session.open", "", protocol.SessionOpenPayload{
		Fields: []card.Field{card.FieldCitizenID},
	})
	token := ""
	for i := 0; i < 5; i++ {
		msg := readMsg(t, ctx, conn)
		if msg["id"] == nil || int64(msg["id"].(float64)) != 1 || msg["error"] != nil {
			continue
		}
		token, _ = msg["result"].(map[string]any)["token"].(string)
		break
	}
	if token == "" {
		t.Fatal("no token")
	}
	writeReq(t, ctx, conn, 2, "card.read", token, protocol.CardReadPayload{
		Fields: []card.Field{card.FieldCitizenID, card.FieldBirthDate},
	})
	for i := 0; i < 5; i++ {
		msg := readMsg(t, ctx, conn)
		if msg["id"] == nil || int64(msg["id"].(float64)) != 2 {
			continue
		}
		errObj, _ := msg["error"].(map[string]any)
		if errObj == nil || errObj["code"] != "FIELD_NOT_ALLOWED" {
			t.Fatalf("expected FIELD_NOT_ALLOWED, got %v", msg)
		}
		return
	}
	t.Fatal("no response")
}
