package bridge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
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

// testBridge is a running synthetic bridge with shared device state.
type testBridge struct {
	wsURL   string
	authMgr *auth.Manager
	mon     *pcsc.SyntheticBackend
	rdb     *pcsc.SyntheticBackend
	state   *pcsc.SyntheticState
	mgr     *reader.Manager
	srv     *Server
	cancel  context.CancelFunc
}

// startTestBridge starts a real bridge on an ephemeral loopback port.
// When seed is non-nil it is applied to shared synthetic state BEFORE the
// manager starts, so the first resync already observes the seeded card.
func startTestBridge(t *testing.T, autoAllow bool) *testBridge {
	t.Helper()
	return startTestBridgeSeed(t, autoAllow, func(mon, rdb *pcsc.SyntheticBackend) {
		// default: no pre-seeded hardware; tests add readers after start
	})
}

func startTestBridgeSeed(t *testing.T, autoAllow bool, seed func(mon, rdb *pcsc.SyntheticBackend)) *testBridge {
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
	authMgr := auth.NewManager("bridge-it", cfg.TrustDuration, cfg.SessionDuration, cfg.ApprovalTimeout, cfg.MaxPendingApprovals)
	shared := pcsc.NewSyntheticState()
	mon := pcsc.NewSyntheticWithState(shared)
	rdb := pcsc.NewSyntheticWithState(shared)

	if seed != nil {
		seed(mon, rdb)
	}

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
		shutCtx, c := context.WithTimeout(context.Background(), 2*time.Second)
		defer c()
		_ = s.Shutdown(shutCtx)
		_ = mgr.Stop(shutCtx)
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

	return &testBridge{
		wsURL:   wsURL,
		authMgr: authMgr,
		mon:     mon,
		rdb:     rdb,
		state:   shared,
		mgr:     mgr,
		srv:     s,
		cancel:  cancel,
	}
}

// waitCardReady waits until the manager has a card session for readerID.
func (tb *testBridge) waitCardReady(t *testing.T, readerID string) string {
	t.Helper()
	sid, ok := tb.mgr.WaitCardSession(readerID, time.Now().Add(2*time.Second))
	if !ok {
		t.Fatalf("timed out waiting for card session on %s", readerID)
	}
	return sid
}

func scriptThaiCard(rdb *pcsc.SyntheticBackend) {
	p := card.ThaiCitizenMVP{}
	rdb.SetScriptResponse(string(p.SelectCommand()), append([]byte(nil), 0x90, 0x00))
	// Documented READ BINARY: 80 B0 P1 P2 Lc=02 Data=00 Le
	rdb.SetScriptResponse(string([]byte{0x80, 0xB0, 0x00, 0x04, 0x02, 0x00, 0x0D}), append([]byte("1101701234567"), 0x90, 0x00))
}

func seedFullThaiCard(rdb *pcsc.SyntheticBackend) {
	rdb.SeedDemoCard()
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
	tb := startTestBridgeSeed(t, true, func(mon, rdb *pcsc.SyntheticBackend) {
		mon.AddReader("r1", "ACR39U")
		mon.InsertCard("r1")
		scriptThaiCard(rdb)
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	tb.waitCardReady(t, "r1")

	conn, _ := dial(t, ctx, tb.wsURL, "https://clinic.example.com")
	defer func() { _ = conn.Close(websocket.StatusNormalClosure, "") }()

	writeReq(t, ctx, conn, 1, "session.open", "", protocol.SessionOpenPayload{
		Fields: []card.Field{card.FieldCitizenID},
	})

	token := readToken(t, ctx, conn, 1)

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

func TestIntegrationSeededFullCardFields(t *testing.T) {
	// Seed the full demo card before startup; assert exact synthetic values.
	tb := startTestBridgeSeed(t, true, func(mon, rdb *pcsc.SyntheticBackend) {
		seedFullThaiCard(rdb)
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	tb.waitCardReady(t, "synthetic-acr39u")

	conn, _ := dial(t, ctx, tb.wsURL, "https://clinic.example.com")
	defer func() { _ = conn.Close(websocket.StatusNormalClosure, "") }()

	writeReq(t, ctx, conn, 1, "session.open", "", protocol.SessionOpenPayload{
		Fields: []card.Field{card.FieldCitizenID, card.FieldThaiName, card.FieldBirthDate},
	})
	token := readToken(t, ctx, conn, 1)

	writeReq(t, ctx, conn, 2, "card.read", token, protocol.CardReadPayload{})
	for i := 0; i < 8; i++ {
		msg := readMsg(t, ctx, conn)
		if msg["id"] == nil || int64(msg["id"].(float64)) != 2 {
			continue
		}
		if msg["error"] != nil {
			t.Fatalf("card.read error: %v", msg["error"])
		}
		fields := msg["result"].(map[string]any)["fields"].(map[string]any)
		if fields["citizenId"] != "1234567890121" {
			t.Fatalf("citizenId %v", fields["citizenId"])
		}
		name, _ := fields["thaiName"].(map[string]any)
		if name == nil {
			t.Fatalf("missing thaiName: %v", fields)
		}
		if name["fullName"] != "นาย สมชาย ใจดี" {
			t.Fatalf("thaiName.fullName %v", name["fullName"])
		}
		if name["givenName"] != "สมชาย" || name["familyName"] != "ใจดี" {
			t.Fatalf("thaiName parts %v", name)
		}
		birth, _ := fields["birthDate"].(map[string]any)
		if birth == nil {
			t.Fatalf("missing birthDate: %v", fields)
		}
		if birth["year"] != float64(1990) || birth["month"] != float64(4) || birth["day"] != float64(12) {
			t.Fatalf("birthDate %v (expected 1990-04-12 Gregorian)", birth)
		}
		return
	}
	t.Fatal("no card.read response")
}

func TestIntegrationInsertionAfterStartup(t *testing.T) {
	tb := startTestBridgeSeed(t, true, func(mon, rdb *pcsc.SyntheticBackend) {
		mon.AddReader("r1", "ACR39U")
		// Card starts absent; insert after the manager is running.
		scriptThaiCard(rdb)
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Wait for reader inventory first.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		snap := tb.mgr.Snapshot()
		if len(snap.Readers) == 1 && snap.Readers[0].ID == "r1" {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if len(tb.mgr.Snapshot().Readers) != 1 {
		t.Fatal("reader not observed")
	}
	if _, ok := tb.mgr.CardSession("r1"); ok {
		t.Fatal("card should start absent")
	}

	tb.mon.InsertCard("r1")
	tb.waitCardReady(t, "r1")

	conn, _ := dial(t, ctx, tb.wsURL, "https://clinic.example.com")
	defer func() { _ = conn.Close(websocket.StatusNormalClosure, "") }()
	writeReq(t, ctx, conn, 1, "session.open", "", protocol.SessionOpenPayload{
		Fields: []card.Field{card.FieldCitizenID},
	})
	token := readToken(t, ctx, conn, 1)
	writeReq(t, ctx, conn, 2, "card.read", token, protocol.CardReadPayload{
		Fields: []card.Field{card.FieldCitizenID},
	})
	msg := readByID(t, ctx, conn, 2)
	if msg["error"] != nil {
		t.Fatalf("card.read error: %v", msg["error"])
	}
	fields := msg["result"].(map[string]any)["fields"].(map[string]any)
	if fields["citizenId"] != "1101701234567" {
		t.Fatalf("citizenId %v", fields["citizenId"])
	}
}

func TestIntegrationBadOriginRejected(t *testing.T) {
	tb := startTestBridge(t, false)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, resp, err := websocket.Dial(ctx, tb.wsURL, &websocket.DialOptions{
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
	tb := startTestBridge(t, false)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) {
			pending := tb.authMgr.Pending(time.Now())
			if len(pending) > 0 {
				_, _ = tb.authMgr.Resolve(pending[0].ID, false, time.Now())
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
	}()

	conn, _ := dial(t, ctx, tb.wsURL, "https://clinic.example.com")
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

func TestIntegrationUnicastReadIsolation(t *testing.T) {
	// Two authenticated connections: a card.read result is delivered only to
	// the requester. The peer may receive metadata events, never field values.
	tb := startTestBridgeSeed(t, true, func(mon, rdb *pcsc.SyntheticBackend) {
		seedFullThaiCard(rdb)
	})
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	tb.waitCardReady(t, "synthetic-acr39u")

	connA, _ := dial(t, ctx, tb.wsURL, "https://clinic.example.com")
	defer func() { _ = connA.Close(websocket.StatusNormalClosure, "") }()
	writeReq(t, ctx, connA, 1, "session.open", "", protocol.SessionOpenPayload{
		Fields: []card.Field{card.FieldCitizenID, card.FieldThaiName, card.FieldBirthDate},
	})
	tokenA := readToken(t, ctx, connA, 1)

	connB, _ := dial(t, ctx, tb.wsURL, "https://clinic.example.com")
	defer func() { _ = connB.Close(websocket.StatusNormalClosure, "") }()
	writeReq(t, ctx, connB, 1, "session.open", "", protocol.SessionOpenPayload{
		Fields: []card.Field{card.FieldCitizenID, card.FieldThaiName, card.FieldBirthDate},
	})
	tokenB := readToken(t, ctx, connB, 1)
	if tokenA == tokenB {
		t.Fatal("competing clients must receive distinct session tokens")
	}

	// Take over connB reads for the rest of the test. Timed Read contexts
	// abort the WebSocket, so the peer is drained from a goroutine using the
	// parent context instead.
	peer := startFrameCollector(ctx, connB)

	writeReq(t, ctx, connA, 2, "card.read", tokenA, protocol.CardReadPayload{})
	msgA := readByID(t, ctx, connA, 2)
	if msgA["error"] != nil {
		t.Fatalf("requester card.read error: %v", msgA["error"])
	}
	fieldsA := msgA["result"].(map[string]any)["fields"].(map[string]any)
	if fieldsA["citizenId"] != "1234567890121" {
		t.Fatalf("requester citizenId %v", fieldsA["citizenId"])
	}

	time.Sleep(250 * time.Millisecond)
	assertNoCardFields(t, peer.snapshot())

	writeReq(t, ctx, connB, 2, "card.read", tokenB, protocol.CardReadPayload{})
	msgB, ok := peer.waitID(2, 3*time.Second)
	if !ok {
		t.Fatal("peer independent card.read timed out")
	}
	if msgB["error"] != nil {
		t.Fatalf("peer independent card.read error: %v", msgB["error"])
	}
	fieldsB := msgB["result"].(map[string]any)["fields"].(map[string]any)
	if fieldsB["citizenId"] != "1234567890121" {
		t.Fatalf("peer citizenId %v", fieldsB["citizenId"])
	}
}

type frameCollector struct {
	mu     sync.Mutex
	frames []map[string]any
}

func startFrameCollector(ctx context.Context, c *websocket.Conn) *frameCollector {
	fc := &frameCollector{}
	go func() {
		for {
			_, data, err := c.Read(ctx)
			if err != nil {
				return
			}
			var m map[string]any
			if json.Unmarshal(data, &m) != nil {
				continue
			}
			fc.mu.Lock()
			fc.frames = append(fc.frames, m)
			fc.mu.Unlock()
		}
	}()
	return fc
}

func (fc *frameCollector) snapshot() []map[string]any {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	out := make([]map[string]any, len(fc.frames))
	copy(out, fc.frames)
	return out
}

func (fc *frameCollector) waitID(id int64, timeout time.Duration) (map[string]any, bool) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		for _, m := range fc.snapshot() {
			if m["id"] == nil {
				continue
			}
			mid, _ := m["id"].(float64)
			if int64(mid) == id {
				return m, true
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	return nil, false
}

func assertNoCardFields(t *testing.T, frames []map[string]any) {
	t.Helper()
	raw, err := json.Marshal(frames)
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	for _, needle := range []string{"1234567890121", "สมชาย", "ใจดี"} {
		if strings.Contains(s, needle) {
			t.Fatalf("competing client received unicast card field %q", needle)
		}
	}
}

func TestIntegrationPairingOnlyNoInventory(t *testing.T) {
	// Unapproved connections must not receive reader inventory, card-session
	// ids, or personal fields (docs/security.md pairing-only; acceptance B1).
	tb := startTestBridgeSeed(t, false, func(mon, rdb *pcsc.SyntheticBackend) {
		seedFullThaiCard(rdb)
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	tb.waitCardReady(t, "synthetic-acr39u")

	conn, _ := dial(t, ctx, tb.wsURL, "https://clinic.example.com")
	defer func() { _ = conn.Close(websocket.StatusNormalClosure, "") }()

	writeReq(t, ctx, conn, 1, "session.open", "", protocol.SessionOpenPayload{
		Fields: []card.Field{card.FieldCitizenID, card.FieldThaiName, card.FieldBirthDate},
	})

	for i := 0; i < 8; i++ {
		msg := readMsg(t, ctx, conn)
		raw, _ := json.Marshal(msg)
		s := string(raw)
		for _, needle := range []string{"1234567890121", "สมชาย", "ใจดี"} {
			if strings.Contains(s, needle) {
				t.Fatalf("pairing-only client received card field %q", needle)
			}
		}
		typ, _ := msg["type"].(string)
		if typ == "status.snapshot" {
			payload, _ := msg["payload"].(map[string]any)
			readers, _ := payload["readers"].([]any)
			if len(readers) > 0 {
				t.Fatalf("pairing-only client received reader inventory: %v", payload)
			}
		}
		if typ == "card.inserted" || typ == "reader.connected" || typ == "card.removed" {
			t.Fatalf("pairing-only client received device event %s", typ)
		}
		if typ == "approval.required" {
			return
		}
		if msg["id"] != nil {
			t.Fatalf("unexpected session.open response before approval: %v", msg)
		}
	}
	t.Fatal("expected approval.required without inventory")
}

func TestIntegrationProbeMetadataOnly(t *testing.T) {
	tb := startTestBridgeSeed(t, false, func(mon, rdb *pcsc.SyntheticBackend) {
		rdb.SeedDemoCard()
	})
	tb.srv.SetRuntime("0.0.1-test", true)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	tb.waitCardReady(t, "synthetic-acr39u")

	conn, _ := dial(t, ctx, tb.wsURL, "https://clinic.example.com")
	defer func() { _ = conn.Close(websocket.StatusNormalClosure, "") }()

	writeReq(t, ctx, conn, 1, "bridge.probe", "", nil)
	msg := readByID(t, ctx, conn, 1)
	if msg["error"] != nil {
		t.Fatalf("probe error %v", msg["error"])
	}
	raw, _ := json.Marshal(msg["result"])
	if bytes.Contains(raw, []byte("1234567890121")) || bytes.Contains(raw, []byte("สมชาย")) {
		t.Fatalf("probe leaked personal fields: %s", raw)
	}
	res := msg["result"].(map[string]any)
	if res["synthetic"] != true {
		t.Fatalf("synthetic=%v", res["synthetic"])
	}
	readers, _ := res["readers"].([]any)
	if len(readers) != 1 {
		t.Fatalf("readers=%v", res["readers"])
	}
	r0 := readers[0].(map[string]any)
	if r0["presence"] != "present" {
		t.Fatalf("presence=%v", r0["presence"])
	}
	if _, ok := r0["cardSessionId"]; ok {
		t.Fatal("probe must not include cardSessionId")
	}
}

func TestIntegrationNoFieldExpansion(t *testing.T) {
	tb := startTestBridgeSeed(t, true, func(mon, rdb *pcsc.SyntheticBackend) {
		mon.AddReader("r1", "A")
		mon.InsertCard("r1")
		scriptThaiCard(rdb)
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	tb.waitCardReady(t, "r1")

	conn, _ := dial(t, ctx, tb.wsURL, "https://clinic.example.com")
	defer func() { _ = conn.Close(websocket.StatusNormalClosure, "") }()
	writeReq(t, ctx, conn, 1, "session.open", "", protocol.SessionOpenPayload{
		Fields: []card.Field{card.FieldCitizenID},
	})
	token := readToken(t, ctx, conn, 1)
	writeReq(t, ctx, conn, 2, "card.read", token, protocol.CardReadPayload{
		Fields: []card.Field{card.FieldCitizenID, card.FieldBirthDate},
	})
	msg := readByID(t, ctx, conn, 2)
	errObj, _ := msg["error"].(map[string]any)
	if errObj == nil || errObj["code"] != "FIELD_NOT_ALLOWED" {
		t.Fatalf("expected FIELD_NOT_ALLOWED, got %v", msg)
	}
}
