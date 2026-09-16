package pcsc

import (
	"context"
	"testing"
	"time"
)

func TestSyntheticLifecycle(t *testing.T) {
	ctx := context.Background()
	sb := NewSynthetic()
	if err := sb.Establish(ctx); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = sb.Release(ctx) }()

	sb.AddReader("r1", "ACR39U")
	readers, err := sb.Readers(ctx)
	if err != nil || len(readers) != 1 {
		t.Fatalf("readers=%v err=%v", readers, err)
	}

	present, err := sb.CardPresent(ctx, "r1")
	if err != nil || present {
		t.Fatalf("expected absent, present=%v err=%v", present, err)
	}

	sb.InsertCard("r1")
	present, err = sb.CardPresent(ctx, "r1")
	if err != nil || !present {
		t.Fatalf("expected present, present=%v err=%v", present, err)
	}

	conn, err := sb.Open(ctx, "r1")
	if err != nil {
		t.Fatal(err)
	}
	if err := conn.BeginTransaction(ctx); err != nil {
		t.Fatal(err)
	}
	sb.SetScriptResponse("SELECT", []byte{0x90, 0x00})
	resp, err := conn.Transmit(ctx, []byte("SELECT"), 256)
	if err != nil || len(resp) != 2 {
		t.Fatalf("resp=%v err=%v", resp, err)
	}
	if err := conn.EndTransaction(ctx); err != nil {
		t.Fatal(err)
	}
	_ = conn.Close()

	sb.RemoveCard("r1")
	present, _ = sb.CardPresent(ctx, "r1")
	if present {
		t.Fatal("expected absent after remove")
	}
	_, err = sb.Open(ctx, "r1")
	if err == nil {
		t.Fatal("open without card must fail")
	}
}

func TestSyntheticReestablishAfterRelease(t *testing.T) {
	ctx := context.Background()
	sb := NewSynthetic()
	if err := sb.Establish(ctx); err != nil {
		t.Fatal(err)
	}
	sb.AddReader("r1", "ACR39U")
	if err := sb.Release(ctx); err != nil {
		t.Fatal(err)
	}
	if err := sb.Establish(ctx); err != nil {
		t.Fatal(err)
	}
	readers, err := sb.Readers(ctx)
	if err != nil || len(readers) != 1 {
		t.Fatalf("readers after reconnect=%v err=%v", readers, err)
	}
}

func TestSyntheticTwoReaders(t *testing.T) {
	ctx := context.Background()
	sb := NewSynthetic()
	if err := sb.Establish(ctx); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = sb.Release(ctx) }()

	sb.AddReader("r1", "SYNTHETIC ACR39U (fictional demo)")
	sb.AddReader("r2", "SYNTHETIC second reader (fictional)")
	sb.InsertCard("r2")

	readers, err := sb.Readers(ctx)
	if err != nil || len(readers) != 2 {
		t.Fatalf("readers=%v err=%v", readers, err)
	}
	present1, err := sb.CardPresent(ctx, "r1")
	if err != nil || present1 {
		t.Fatalf("r1 present=%v err=%v", present1, err)
	}
	present2, err := sb.CardPresent(ctx, "r2")
	if err != nil || !present2 {
		t.Fatalf("r2 present=%v err=%v", present2, err)
	}
	if _, err := sb.Open(ctx, "r1"); err == nil {
		t.Fatal("open r1 without a card must fail")
	}
	conn, err := sb.Open(ctx, "r2")
	if err != nil {
		t.Fatal(err)
	}
	_ = conn.Close()
}

func TestSeedSecondDemoReader(t *testing.T) {
	ctx := context.Background()
	sb := NewSynthetic()
	if err := sb.Establish(ctx); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = sb.Release(ctx) }()
	sb.SeedDemoCard()
	sb.SeedSecondDemoReader()
	readers, err := sb.Readers(ctx)
	if err != nil || len(readers) != 2 {
		t.Fatalf("readers=%v err=%v", readers, err)
	}
}

func TestSyntheticStalled(t *testing.T) {
	ctx := context.Background()
	sb := NewSynthetic()
	_ = sb.Establish(ctx)
	sb.SetStalled(true)
	if err := sb.Supported(); err == nil {
		t.Fatal("stalled backend must report unsupported")
	}
}

func TestSyntheticWaitForChange(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	sb := NewSynthetic()
	_ = sb.Establish(ctx)
	sb.AddReader("r1", "reader")

	errCh := make(chan error, 1)
	go func() {
		errCh <- sb.WaitForChange(ctx, time.Second)
	}()
	time.Sleep(20 * time.Millisecond)
	sb.InsertCard("r1")
	select {
	case err := <-errCh:
		if err != nil && err != context.DeadlineExceeded && err != context.Canceled {
			t.Fatalf("wait err %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("wait did not return")
	}
}
