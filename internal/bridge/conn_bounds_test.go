package bridge

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/night-sornram/openthaiid/internal/auth"
	"github.com/night-sornram/openthaiid/internal/protocol"
)

// An eligible page that opens sockets and never authenticates must not hold
// MaxConnections slots forever.
func TestUnauthenticatedIdleConnectionIsClosed(t *testing.T) {
	tb := startTestBridge(t, false)
	tb.srv.unauthTimeout = 300 * time.Millisecond

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c, _ := dial(t, ctx, tb.wsURL, "http://localhost:3000")
	defer c.CloseNow()

	start := time.Now()
	_, _, err := c.Read(ctx)
	if err == nil {
		t.Fatal("expected the idle unauthenticated socket to be closed")
	}
	if got := websocket.CloseStatus(err); got != websocket.StatusPolicyViolation {
		t.Fatalf("close status = %v (err %v), want policy violation", got, err)
	}
	if el := time.Since(start); el > 3*time.Second {
		t.Fatalf("closed after %v; watchdog did not fire", el)
	}
}

// A session.open waiting on staff Allow is in flight: the unauthenticated
// bound must not cut it off before ApprovalTimeout.
func TestUnauthenticatedBoundSparesPendingApproval(t *testing.T) {
	tb := startTestBridge(t, false)
	tb.srv.unauthTimeout = 200 * time.Millisecond

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c, _ := dial(t, ctx, tb.wsURL, "http://localhost:3000")
	defer c.CloseNow()

	writeReq(t, ctx, c, 1, string(protocol.TypeSessionOpen), "", map[string]any{"fields": []string{"citizenId"}})
	var pendingID int64
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if p := tb.authMgr.Pending(time.Now()); len(p) > 0 {
			pendingID = p[0].ID
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if pendingID == 0 {
		t.Fatal("approval never became pending")
	}
	// Staff take longer than the unauthenticated bound to click Allow.
	time.Sleep(600 * time.Millisecond)
	if _, err := tb.authMgr.Resolve(pendingID, true, time.Now()); err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if tok := readToken(t, ctx, c, 1); tok == "" {
		t.Fatal("expected a session token after a slow Allow")
	}
}

// One client whose socket has stopped draining must not block enqueue for
// the broadcaster (which serves every client in turn).
func TestEnqueueNeverWaitsOnBlockedWrite(t *testing.T) {
	accepted := make(chan *clientConn, 1)
	hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		cc := newClientConn(ws, auth.Origin("http://localhost:3000"))
		go cc.writeLoop()
		accepted <- cc
		<-r.Context().Done()
	}))
	defer hs.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(hs.URL, "http"), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer client.CloseNow()
	// The client never reads, so the server's TCP send buffer fills and
	// ws.Write blocks inside writeLoop.
	cc := <-accepted
	defer cc.closeWS(websocket.StatusNormalClosure, "")

	frame := make([]byte, 1<<20)
	for i := 0; i < 16; i++ {
		cc.enqueue(frame)
	}
	time.Sleep(300 * time.Millisecond)

	done := make(chan struct{})
	go func() {
		cc.enqueue(frame)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("enqueue blocked behind a stalled ws.Write")
	}
}
