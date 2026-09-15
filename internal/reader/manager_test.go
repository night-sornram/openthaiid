package reader

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/night-sornram/openthaiid/internal/card"
	"github.com/night-sornram/openthaiid/internal/pcsc"
)

func startPair(t *testing.T) (*Manager, *pcsc.SyntheticBackend, *pcsc.SyntheticBackend, context.CancelFunc) {
	t.Helper()
	shared := pcsc.NewSyntheticState()
	mon := pcsc.NewSyntheticWithState(shared)
	rdb := pcsc.NewSyntheticWithState(shared)
	ctx, cancel := context.WithCancel(context.Background())
	if err := mon.Establish(ctx); err != nil {
		t.Fatal(err)
	}
	if err := rdb.Establish(ctx); err != nil {
		t.Fatal(err)
	}
	m := NewManager(mon, rdb)
	if err := m.Start(ctx); err != nil {
		t.Fatal(err)
	}
	if err := m.resync(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cancel()
		stopCtx, c := context.WithTimeout(context.Background(), 2*time.Second)
		defer c()
		_ = m.Stop(stopCtx)
	})
	return m, mon, rdb, cancel
}

func TestResyncEmitsEvents(t *testing.T) {
	ctx := context.Background()
	m, mon, _, _ := startPair(t)
	ch := m.Subscribe()
	defer m.Unsubscribe(ch)

	mon.AddReader("r1", "ACR39U")
	if err := m.resync(ctx); err != nil {
		t.Fatal(err)
	}
	ev := waitEvent(t, ch, "reader.connected")
	if ev.ReaderID != "r1" {
		t.Fatalf("got %+v", ev)
	}

	mon.InsertCard("r1")
	if err := m.resync(ctx); err != nil {
		t.Fatal(err)
	}
	ev = waitEvent(t, ch, "card.inserted")
	if ev.CardSessionID == "" {
		t.Fatal("missing card session id")
	}

	// Session stable while present.
	first := ev.CardSessionID
	if err := m.resync(ctx); err != nil {
		t.Fatal(err)
	}
	sid, ok := m.CardSession("r1")
	if !ok || sid != first {
		t.Fatalf("session changed: %q vs %q", sid, first)
	}

	mon.RemoveCard("r1")
	if err := m.resync(ctx); err != nil {
		t.Fatal(err)
	}
	ev = waitEvent(t, ch, "card.removed")
	if ev.CardSessionID != first {
		t.Fatal("removed session mismatch")
	}
	if _, ok := m.CardSession("r1"); ok {
		t.Fatal("session should be gone")
	}

	// Reinsert → new session.
	mon.InsertCard("r1")
	if err := m.resync(ctx); err != nil {
		t.Fatal(err)
	}
	ev = waitEvent(t, ch, "card.inserted")
	if ev.CardSessionID == first {
		t.Fatal("reinsert must create a new card session")
	}
}

func TestSelectReader(t *testing.T) {
	ctx := context.Background()
	m, mon, _, _ := startPair(t)
	mon.AddReader("r1", "A")
	mon.AddReader("r2", "B")
	mon.InsertCard("r1")
	mon.InsertCard("r2")
	if err := m.resync(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := m.SelectReader(""); err == nil {
		t.Fatal("multiple readers must require selection")
	}
	id, err := m.SelectReader("r1")
	if err != nil || id != "r1" {
		t.Fatalf("id=%q err=%v", id, err)
	}
}

func TestReadBusyAndGeneration(t *testing.T) {
	ctx := context.Background()
	m, mon, rdb, _ := startPair(t)
	mon.AddReader("r1", "A")
	mon.InsertCard("r1")
	if err := m.resync(ctx); err != nil {
		t.Fatal(err)
	}
	sid, ok := m.CardSession("r1")
	if !ok {
		t.Fatal("no session")
	}
	_ = rdb

	// Wrong generation fails.
	_, err := m.Read(ctx, ReadRequest{
		ReaderID:      "r1",
		CardSessionID: "cs_wrong",
		Fields:        []card.Field{card.FieldCitizenID},
		Execute: func(ctx context.Context, conn card.Transmitter, fields []card.Field) (*card.FieldValues, []card.Warning, error) {
			return &card.FieldValues{}, nil, nil
		},
	})
	if err == nil {
		t.Fatal("wrong generation should fail")
	}

	// Matching generation succeeds.
	res, err := m.Read(ctx, ReadRequest{
		ReaderID:      "r1",
		CardSessionID: sid,
		Fields:        []card.Field{card.FieldCitizenID},
		Execute: func(ctx context.Context, conn card.Transmitter, fields []card.Field) (*card.FieldValues, []card.Warning, error) {
			id := "1234567890123"
			return &card.FieldValues{CitizenID: &id}, nil, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.CardSessionID != sid || res.Fields.CitizenID == nil {
		t.Fatalf("bad result %+v", res)
	}
}

func TestOverlappingReadBusy(t *testing.T) {
	m, mon, rdb, _ := startPair(t)
	mon.AddReader("r1", "A")
	mon.InsertCard("r1")
	if err := m.resync(context.Background()); err != nil {
		t.Fatal(err)
	}
	sid, _ := m.CardSession("r1")
	rdb.BlockPhase("transmit")

	started := make(chan struct{})
	errCh := make(chan error, 1)
	go func() {
		close(started)
		_, err := m.Read(context.Background(), ReadRequest{
			ReaderID:      "r1",
			CardSessionID: sid,
			Fields:        []card.Field{card.FieldCitizenID},
			Execute: func(ctx context.Context, conn card.Transmitter, fields []card.Field) (*card.FieldValues, []card.Warning, error) {
				_, err := conn.Transmit(ctx, []byte("CMD"), 8)
				if err != nil {
					return nil, nil, err
				}
				id := "1"
				return &card.FieldValues{CitizenID: &id}, nil, nil
			},
		})
		errCh <- err
	}()
	<-started
	// Wait until the first read holds the slot.
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		m.mu.RLock()
		reading := m.reading
		m.mu.RUnlock()
		if reading {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	_, err := m.Read(context.Background(), ReadRequest{
		ReaderID: "r1",
		Fields:   []card.Field{card.FieldCitizenID},
		Execute: func(ctx context.Context, conn card.Transmitter, fields []card.Field) (*card.FieldValues, []card.Warning, error) {
			return &card.FieldValues{}, nil, nil
		},
	})
	if err == nil || card.AsError(err).Code != card.ErrBusy {
		t.Fatalf("expected BUSY, got %v", err)
	}
	rdb.SetScriptResponse("CMD", []byte{0x90, 0x00})
	rdb.UnblockPhase("transmit")
	if err := <-errCh; err != nil {
		t.Fatal(err)
	}
}

func TestCancelRetainsSlotDiscardsResultAndCleansUpOnce(t *testing.T) {
	m, mon, rdb, _ := startPair(t)
	mon.AddReader("r1", "A")
	mon.InsertCard("r1")
	if err := m.resync(context.Background()); err != nil {
		t.Fatal(err)
	}
	sid, _ := m.CardSession("r1")
	rdb.BlockPhase("transmit")

	var executes int64
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := m.Read(ctx, ReadRequest{
		ReaderID:      "r1",
		CardSessionID: sid,
		Fields:        []card.Field{card.FieldCitizenID},
		Execute: func(ctx context.Context, conn card.Transmitter, fields []card.Field) (*card.FieldValues, []card.Warning, error) {
			atomic.AddInt64(&executes, 1)
			_, err := conn.Transmit(context.Background(), []byte("CMD"), 8)
			if err != nil {
				return nil, nil, err
			}
			id := "late-result-should-be-discarded"
			return &card.FieldValues{CitizenID: &id}, nil, nil
		},
	})
	if err == nil {
		t.Fatal("expected cancel error")
	}
	code := card.AsError(err).Code
	if code != card.ErrCancelled && code != card.ErrReadTimeout {
		t.Fatalf("expected CANCELLED/READ_TIMEOUT, got %v", err)
	}
	if time.Since(start) > time.Second {
		t.Fatalf("cancel not prompt: %s", time.Since(start))
	}

	// Slot retained while the blocked native call is outstanding.
	m.mu.RLock()
	reading := m.reading
	m.mu.RUnlock()
	if !reading {
		t.Fatal("slot must stay occupied until cleanup finishes")
	}

	rdb.UnblockPhase("transmit")
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		m.mu.RLock()
		reading = m.reading
		m.mu.RUnlock()
		if !reading {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if reading {
		t.Fatal("slot not released after cleanup")
	}
	endTx, closes := rdb.CleanupStats()
	if endTx != 1 || closes != 1 {
		t.Fatalf("exactly-once cleanup violated: endTx=%d closes=%d", endTx, closes)
	}
}

func TestStallAfterGraceThenRecovery(t *testing.T) {
	m, mon, rdb, _ := startPair(t)
	mon.AddReader("r1", "A")
	mon.InsertCard("r1")
	if err := m.resync(context.Background()); err != nil {
		t.Fatal(err)
	}
	sid, _ := m.CardSession("r1")
	rdb.BlockPhase("transmit")

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, err := m.Read(ctx, ReadRequest{
		ReaderID:      "r1",
		CardSessionID: sid,
		Fields:        []card.Field{card.FieldCitizenID},
		Execute: func(ctx context.Context, conn card.Transmitter, fields []card.Field) (*card.FieldValues, []card.Warning, error) {
			_, err := conn.Transmit(context.Background(), []byte("CMD"), 8)
			return &card.FieldValues{}, nil, err
		},
	})
	if err == nil {
		t.Fatal("expected cancel")
	}

	time.Sleep(StallGrace + 50*time.Millisecond)
	_, err = m.Read(context.Background(), ReadRequest{
		ReaderID: "r1",
		Fields:   []card.Field{card.FieldCitizenID},
		Execute: func(ctx context.Context, conn card.Transmitter, fields []card.Field) (*card.FieldValues, []card.Warning, error) {
			return &card.FieldValues{}, nil, nil
		},
	})
	if err == nil || card.AsError(err).Code != card.ErrBackendStalled {
		t.Fatalf("expected BACKEND_STALLED, got %v", err)
	}
	if snap := m.Snapshot(); snap.Backend != "stalled" {
		t.Fatalf("backend status %q", snap.Backend)
	}

	rdb.UnblockPhase("transmit")
	// Worker finishes; recovery clears stall.
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if m.Snapshot().Backend != "stalled" {
			// Re-seed card for a successful subsequent read after recovery.
			mon.InsertCard("r1")
			if err := m.resync(context.Background()); err == nil {
				if _, ok := m.CardSession("r1"); ok {
					return
				}
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("backend did not recover: %+v", m.Snapshot())
}

func TestStopRefusesReleaseWhileWorkerBlocked(t *testing.T) {
	m, mon, rdb, _ := startPair(t)
	mon.AddReader("r1", "A")
	mon.InsertCard("r1")
	if err := m.resync(context.Background()); err != nil {
		t.Fatal(err)
	}
	sid, _ := m.CardSession("r1")
	rdb.BlockPhase("transmit")

	go func() {
		_, _ = m.Read(context.Background(), ReadRequest{
			ReaderID:      "r1",
			CardSessionID: sid,
			Fields:        []card.Field{card.FieldCitizenID},
			Execute: func(ctx context.Context, conn card.Transmitter, fields []card.Field) (*card.FieldValues, []card.Warning, error) {
				_, err := conn.Transmit(context.Background(), []byte("CMD"), 8)
				return &card.FieldValues{}, nil, err
			},
		})
	}()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		m.mu.RLock()
		reading := m.reading
		m.mu.RUnlock()
		if reading {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	stopCtx, stopCancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer stopCancel()
	err := m.Stop(stopCtx)
	if err == nil {
		// Stop succeeded only if it did not release while blocked — either
		// deadline error or success after worker finished. Worker is blocked.
		t.Fatal("expected Stop to hit deadline while worker holds native context")
	}
	rdb.UnblockPhase("transmit")
}

func waitEvent(t *testing.T, ch chan ChangeEvent, kind string) ChangeEvent {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for {
		select {
		case ev := <-ch:
			if ev.Kind == kind {
				return ev
			}
		case <-deadline:
			t.Fatalf("timeout waiting for %s", kind)
		}
	}
}
