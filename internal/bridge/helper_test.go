package bridge

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/night-sornram/openthaiid/internal/pcsc"
)

func TestHelperSummaryNoPII(t *testing.T) {
	t.Setenv("OPENTHAIID_LANG", "th")
	tb := startTestBridgeSeed(t, false, func(mon, rdb *pcsc.SyntheticBackend) {
		rdb.SeedDemoCard()
	})
	tb.srv.SetRuntime("0.0.1-test", true)
	waitReader(t, tb, "synthetic-acr39u")

	resp, err := http.Get(tb.httpBase() + "/v1/helper/summary")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range []string{"1234567890121", "สมชาย", "Somchai", "citizenId"} {
		if bytes.Contains(raw, []byte(n)) {
			t.Fatalf("summary leaked %q: %s", n, raw)
		}
	}
	var s map[string]any
	if err := json.Unmarshal(raw, &s); err != nil {
		t.Fatal(err)
	}
	if s["helper"] != true || s["reader"] != true || s["card"] != true {
		t.Fatalf("lights %+v", s)
	}
	if s["practice"] != true {
		t.Fatalf("practice %+v", s)
	}
	if s["lang"] != "th" {
		t.Fatalf("lang %v", s["lang"])
	}
	if !strings.Contains(s["headline"].(string), "พร้อม") {
		t.Fatalf("headline %v", s["headline"])
	}
	if _, ok := s["pending"]; ok {
		t.Fatal("summary must not include pending origins")
	}
	if _, ok := s["trusted"]; ok {
		t.Fatal("summary must not include trusted origins")
	}
}

func TestHelperSummaryMultiReaderNeedsSelection(t *testing.T) {
	t.Setenv("OPENTHAIID_LANG", "th")
	tb := startTestBridgeSeed(t, false, func(mon, rdb *pcsc.SyntheticBackend) {
		mon.AddReader("r1", "Reader One")
		mon.AddReader("r2", "Reader Two")
		mon.InsertCard("r1")
		mon.InsertCard("r2")
	})
	waitReader(t, tb, "r1")
	waitReader(t, tb, "r2")

	resp, err := http.Get(tb.httpBase() + "/v1/helper/summary")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	var s map[string]any
	if err := json.Unmarshal(raw, &s); err != nil {
		t.Fatal(err)
	}
	if s["card"] == true || s["headline"] == "พร้อม" {
		t.Fatalf("two readers with no selection must not be Ready: %+v", s)
	}
	next, _ := s["next"].(string)
	if !strings.Contains(next, "เลือกเครื่อง") {
		t.Fatalf("next %q", next)
	}

	if err := tb.mgr.SetPreferredReader("r2"); err != nil {
		t.Fatal(err)
	}
	resp2, err := http.Get(tb.httpBase() + "/v1/helper/summary")
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	raw, err = io.ReadAll(resp2.Body)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &s); err != nil {
		t.Fatal(err)
	}
	if s["card"] != true || s["headline"] != "พร้อม" {
		t.Fatalf("selected reader with card should be Ready: %+v", s)
	}
	if reader, _ := s["readerText"].(string); !strings.Contains(reader, "Reader Two") {
		t.Fatalf("readerText should name the selected reader: %+v", s)
	}
}

func TestHelperShutdownRejectsForeignOrigin(t *testing.T) {
	tb := startTestBridge(t, false)
	req, err := http.NewRequest(http.MethodPost, tb.httpBase()+"/v1/helper/shutdown", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Origin", "https://evil.example")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status %d", resp.StatusCode)
	}
}

func TestHelperShutdownNoOrigin(t *testing.T) {
	tb := startTestBridge(t, false)
	var called atomic.Bool
	tb.srv.SetShutdownRequest(func() { called.Store(true) })
	resp, err := http.Post(tb.httpBase()+"/v1/helper/shutdown", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if called.Load() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("shutdown callback not invoked")
}
