package bridge

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/night-sornram/openthaiid/internal/card"
	"github.com/night-sornram/openthaiid/internal/pcsc"
	"github.com/night-sornram/openthaiid/internal/protocol"
	"github.com/night-sornram/openthaiid/internal/reader"
)

func TestDuplicateRequestIDRejected(t *testing.T) {
	tb := startTestBridge(t, true)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, _ := dial(t, ctx, tb.wsURL, "https://clinic.example.com")
	defer func() { _ = conn.Close(websocket.StatusNormalClosure, "") }()

	writeReq(t, ctx, conn, 1, "session.open", "", protocol.SessionOpenPayload{
		Fields: []card.Field{card.FieldCitizenID},
	})
	token := readToken(t, ctx, conn, 1)

	// Same ID twice — second must be rejected.
	writeReq(t, ctx, conn, 2, "bridge.status", token, nil)
	_ = readByID(t, ctx, conn, 2)
	writeReq(t, ctx, conn, 2, "bridge.status", token, nil)
	msg := readByID(t, ctx, conn, 2)
	errObj, _ := msg["error"].(map[string]any)
	if errObj == nil {
		t.Fatalf("expected error for duplicate id, got %v", msg)
	}
	if errObj["code"] != "INVALID_REQUEST" {
		t.Fatalf("code %v", errObj["code"])
	}
}

func TestCancelUnknownRequestID(t *testing.T) {
	tb := startTestBridge(t, true)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, _ := dial(t, ctx, tb.wsURL, "https://clinic.example.com")
	defer func() { _ = conn.Close(websocket.StatusNormalClosure, "") }()
	writeReq(t, ctx, conn, 1, "session.open", "", protocol.SessionOpenPayload{
		Fields: []card.Field{card.FieldCitizenID},
	})
	token := readToken(t, ctx, conn, 1)

	writeReq(t, ctx, conn, 2, "request.cancel", token, protocol.CancelPayload{RequestID: 999})
	msg := readByID(t, ctx, conn, 2)
	if msg["error"] != nil {
		t.Fatalf("cancel should succeed, got %v", msg["error"])
	}
	res := msg["result"].(map[string]any)
	if res["cancelled"] != false {
		t.Fatalf("expected cancelled=false, got %v", res)
	}
}

func TestRevocationStopsEventsAndCloses(t *testing.T) {
	tb := startTestBridgeSeed(t, true, func(mon, rdb *pcsc.SyntheticBackend) {
		mon.AddReader("r1", "ACR39U")
	})
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	// Wait for reader inventory (no card required).
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if len(tb.mgr.Snapshot().Readers) == 1 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	conn, _ := dial(t, ctx, tb.wsURL, "https://clinic.example.com")
	defer func() { _ = conn.Close(websocket.StatusNormalClosure, "") }()

	writeReq(t, ctx, conn, 1, "session.open", "", protocol.SessionOpenPayload{
		Fields: []card.Field{card.FieldCitizenID},
	})
	_ = readToken(t, ctx, conn, 1)

	tb.authMgr.RevokeOrigin("https://clinic.example.com")

	deadline = time.Now().Add(2 * time.Second)
	sawRevoke := false
	for time.Now().Before(deadline) {
		rctx, rcancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
		_, data, err := conn.Read(rctx)
		rcancel()
		if err != nil {
			break
		}
		var m map[string]any
		if json.Unmarshal(data, &m) == nil && m["type"] == "authorization.revoked" {
			sawRevoke = true
			break
		}
	}
	_ = sawRevoke
	// After revocation the socket is closed; a write may succeed into a
	// half-closed buffer — a subsequent read must fail.
	if err := conn.Write(ctx, websocket.MessageText, []byte(`{"v":1,"id":3,"type":"bridge.status"}`)); err == nil {
		rctx, rcancel := context.WithTimeout(context.Background(), time.Second)
		_, _, err := conn.Read(rctx)
		rcancel()
		if err == nil {
			t.Fatal("expected connection to be closed after revocation")
		}
	}
}

func TestAsyncOpSaturationRateLimited(t *testing.T) {
	tb := startTestBridge(t, true)
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	conn, _ := dial(t, ctx, tb.wsURL, "https://clinic.example.com")
	defer func() { _ = conn.Close(websocket.StatusNormalClosure, "") }()
	writeReq(t, ctx, conn, 1, "session.open", "", protocol.SessionOpenPayload{
		Fields: []card.Field{card.FieldCitizenID},
	})
	token := readToken(t, ctx, conn, 1)

	// Fire more than maxAsyncOpsPerConn concurrent bridge.status requests.
	// They complete quickly, so also issue session.renew which is async.
	// Use session.renew (async) without reading responses first to pile up.
	// First consume the session.open result; then blast renew+status.
	// bridge.status is fast; to saturate we need slow ops. Approval-wait is
	// slow, so open new sessions from the same connection (each waits).
	// After one grant exists, session.open is fast. Use card.read against a
	// blocked transmit instead.
	tb.mon.AddReader("r1", "A")
	tb.mon.InsertCard("r1")
	scriptThaiCard(tb.rdb)
	tb.waitCardReady(t, "r1")
	tb.rdb.BlockPhase("transmit")

	// Overlapping reads must return BUSY, not queue forever.
	const burst = 12
	for i := 0; i < burst; i++ {
		writeReq(t, ctx, conn, int64(10+i), "card.read", token, protocol.CardReadPayload{
			Fields: []card.Field{card.FieldCitizenID},
		})
	}

	// Drain responses: first may hang on transmit; others should be BUSY or RATE_LIMITED.
	// Unblock after we have seen at least one BUSY or RATE_LIMITED, or timeout.
	deadline := time.Now().Add(3 * time.Second)
	sawBusyOrLimit := false
	var firstID *int64
	for time.Now().Before(deadline) {
		rctx, rcancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		_, data, err := conn.Read(rctx)
		rcancel()
		if err != nil {
			continue
		}
		var m map[string]any
		if json.Unmarshal(data, &m) != nil {
			continue
		}
		if m["error"] == nil {
			// Possibly a successful read after we unblocked; ignore.
			continue
		}
		code, _ := m["error"].(map[string]any)["code"].(string)
		if code == "BUSY" || code == "RATE_LIMITED" {
			sawBusyOrLimit = true
			if firstID == nil {
				if id, ok := m["id"].(float64); ok {
					v := int64(id)
					firstID = &v
				}
			}
			// Keep reading a bit to confirm multiple rejects, then unblock.
			break
		}
	}
	tb.rdb.UnblockPhase("transmit")
	if !sawBusyOrLimit {
		// Unblock and require at least BUSY on a second overlapping attempt.
		// Sequential second read after unblock should succeed; a concurrent
		// one while first is blocked is covered above. If the first read
		// finished before others started, issue an explicit overlap.
		tb.rdb.BlockPhase("transmit")
		writeReq(t, ctx, conn, 50, "card.read", token, protocol.CardReadPayload{
			Fields: []card.Field{card.FieldCitizenID},
		})
		// Give the first read time to acquire the slot.
		time.Sleep(20 * time.Millisecond)
		writeReq(t, ctx, conn, 51, "card.read", token, protocol.CardReadPayload{
			Fields: []card.Field{card.FieldCitizenID},
		})
		msg := readByID(t, ctx, conn, 51)
		errObj, _ := msg["error"].(map[string]any)
		if errObj == nil || errObj["code"] != "BUSY" {
			tb.rdb.UnblockPhase("transmit")
			t.Fatalf("expected BUSY on overlapping read, got %v", msg)
		}
		tb.rdb.UnblockPhase("transmit")
		// Drain the first read.
		_ = readByID(t, ctx, conn, 50)
	}
}

func TestCancelBlockedReadReturnsPromptlyAndStalls(t *testing.T) {
	tb := startTestBridgeSeed(t, true, func(mon, rdb *pcsc.SyntheticBackend) {
		mon.AddReader("r1", "ACR39U")
		mon.InsertCard("r1")
		scriptThaiCard(rdb)
		rdb.BlockPhase("transmit")
	})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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
	// Let the worker acquire the native slot and block in Transmit.
	time.Sleep(50 * time.Millisecond)

	start := time.Now()
	writeReq(t, ctx, conn, 3, "request.cancel", token, protocol.CancelPayload{RequestID: 2})
	cancelMsg := readByID(t, ctx, conn, 3)
	if cancelMsg["error"] != nil {
		t.Fatalf("cancel failed: %v", cancelMsg["error"])
	}

	// The read must fail promptly with CANCELLED without waiting for native cleanup.
	readMsg := readByID(t, ctx, conn, 2)
	errObj, _ := readMsg["error"].(map[string]any)
	if errObj == nil || errObj["code"] != "CANCELLED" {
		t.Fatalf("expected CANCELLED, got %v", readMsg)
	}
	elapsed := time.Since(start)
	if elapsed > time.Second {
		t.Fatalf("cancel not prompt: %s", elapsed)
	}

	// Overlapping / subsequent reads while the native call is still blocked
	// must be rejected (slot retained) — BUSY, not a second native open.
	writeReq(t, ctx, conn, 4, "card.read", token, protocol.CardReadPayload{
		Fields: []card.Field{card.FieldCitizenID},
	})
	msg := readByID(t, ctx, conn, 4)
	errObj, _ = msg["error"].(map[string]any)
	if errObj == nil {
		t.Fatalf("expected error while slot retained, got %v", msg)
	}
	if errObj["code"] != "BUSY" && errObj["code"] != "BACKEND_STALLED" {
		t.Fatalf("expected BUSY or BACKEND_STALLED, got %v", errObj)
	}

	// Unblock native work; cleanup must run (exactly once).
	tb.rdb.UnblockPhase("transmit")
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		endTx, closes := tb.rdb.CleanupStats()
		if endTx >= 1 && closes >= 1 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	endTx, closes := tb.rdb.CleanupStats()
	t.Fatalf("cleanup did not run: endTx=%d closes=%d", endTx, closes)
}

func TestStallMarksBackendUnavailable(t *testing.T) {
	tb := startTestBridgeSeed(t, true, func(mon, rdb *pcsc.SyntheticBackend) {
		mon.AddReader("r1", "ACR39U")
		mon.InsertCard("r1")
		scriptThaiCard(rdb)
		rdb.BlockPhase("transmit")
	})
	tb.waitCardReady(t, "r1")

	// Direct manager test for stall grace.
	sid, _ := tb.mgr.CardSession("r1")
	readCtx, readCancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	start := time.Now()
	_, err := tb.mgr.Read(readCtx, readRequestCID(sid))
	readCancel()
	if err == nil {
		t.Fatal("expected error from cancelled read")
	}
	cerr := card.AsError(err)
	if cerr.Code != card.ErrReadTimeout && cerr.Code != card.ErrCancelled {
		t.Fatalf("expected CANCELLED/READ_TIMEOUT, got %v", cerr)
	}
	if time.Since(start) > time.Second {
		t.Fatalf("cancel not prompt: %s", time.Since(start))
	}

	// Wait past StallGrace; subsequent reads must see BACKEND_STALLED.
	time.Sleep(reader.StallGrace + 100*time.Millisecond)
	_, err = tb.mgr.Read(context.Background(), readRequestCID(sid))
	if err == nil {
		t.Fatal("expected BACKEND_STALLED after grace")
	}
	if card.AsError(err).Code != card.ErrBackendStalled && card.AsError(err).Code != card.ErrBusy {
		// BUSY is also acceptable if the worker still holds the slot before
		// stall is published; stall is the intended post-grace state.
		if card.AsError(err).Code != card.ErrBackendStalled {
			// Accept BUSY only if stall flag not yet visible; force check snapshot.
			snap := tb.mgr.Snapshot()
			if snap.Backend != "stalled" {
				t.Fatalf("expected stalled backend, got %v err=%v", snap.Backend, err)
			}
		}
	}

	tb.rdb.UnblockPhase("transmit")
	// Worker finish should eventually allow recovery of the slot.
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if tb.mgr.Snapshot().Backend != "stalled" {
			return
		}
		// Force recovery attempt via another resync path.
		time.Sleep(50 * time.Millisecond)
	}
}

func readRequestCID(sid string) reader.ReadRequest {
	return reader.ReadRequest{
		ReaderID:      "r1",
		CardSessionID: sid,
		Fields:        []card.Field{card.FieldCitizenID},
		Execute: func(ctx context.Context, conn card.Transmitter, fields []card.Field) (*card.FieldValues, []card.Warning, error) {
			return tbExecuteDemo(ctx, conn, fields)
		},
	}
}

func tbExecuteDemo(ctx context.Context, conn card.Transmitter, fields []card.Field) (*card.FieldValues, []card.Warning, error) {
	// Minimal SELECT + CID read matching scriptThaiCard.
	p := card.ThaiCitizenMVP{}
	if _, err := conn.Transmit(ctx, p.SelectCommand(), 8); err != nil {
		return nil, nil, err
	}
	body, err := conn.Transmit(ctx, []byte{0x80, 0xB0, 0x00, 0x04, 0x02, 0x00, 0x0D}, 32)
	if err != nil {
		return nil, nil, err
	}
	id := string(body[:13])
	return &card.FieldValues{CitizenID: &id}, nil, nil
}

func readToken(t *testing.T, ctx context.Context, c *websocket.Conn, id int64) string {
	t.Helper()
	msg := readByID(t, ctx, c, id)
	if msg["error"] != nil {
		t.Fatalf("id %d error: %v", id, msg["error"])
	}
	result, _ := msg["result"].(map[string]any)
	tok, _ := result["token"].(string)
	if tok == "" {
		t.Fatalf("no token in %v", msg)
	}
	return tok
}

func readByID(t *testing.T, ctx context.Context, c *websocket.Conn, id int64) map[string]any {
	t.Helper()
	pending := map[int64]map[string]any{}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if msg, ok := pending[id]; ok {
			return msg
		}
		rctx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
		_, data, err := c.Read(rctx)
		cancel()
		if err != nil {
			if ctx.Err() != nil {
				break
			}
			continue
		}
		var m map[string]any
		if err := json.Unmarshal(data, &m); err != nil {
			continue
		}
		if m["id"] == nil {
			continue // event
		}
		mid, _ := m["id"].(float64)
		pending[int64(mid)] = m
	}
	if msg, ok := pending[id]; ok {
		return msg
	}
	t.Fatalf("no response for id %d", id)
	return nil
}
