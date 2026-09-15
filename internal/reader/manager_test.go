package reader

import (
	"context"
	"testing"
	"time"

	"github.com/night-sornram/openthaiid/internal/card"
	"github.com/night-sornram/openthaiid/internal/pcsc"
)

func startPair(t *testing.T) (*Manager, *pcsc.SyntheticBackend, *pcsc.SyntheticBackend, context.CancelFunc) {
	t.Helper()
	mon := pcsc.NewSynthetic()
	rdb := pcsc.NewSynthetic()
	// Share script via both backends for transmit; inventory is independent.
	// For lifecycle tests we only need the monitor side for presence.
	ctx, cancel := context.WithCancel(context.Background())
	if err := mon.Establish(ctx); err != nil {
		t.Fatal(err)
	}
	if err := rdb.Establish(ctx); err != nil {
		t.Fatal(err)
	}
	m := NewManager(mon, rdb)
	if err := m.resync(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cancel()
		_ = m.Stop(context.Background())
	})
	return m, mon, rdb, cancel
}

func TestResyncEmitsEvents(t *testing.T) {
	ctx := context.Background()
	m, mon, rdb, _ := startPair(t)
	_ = rdb
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
	rdb.AddReader("r1", "A")
	rdb.InsertCard("r1")
	if err := m.resync(ctx); err != nil {
		t.Fatal(err)
	}
	sid, ok := m.CardSession("r1")
	if !ok {
		t.Fatal("no session")
	}
	rdb.SetScriptResponse("CMD", []byte{0x90, 0x00})

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
