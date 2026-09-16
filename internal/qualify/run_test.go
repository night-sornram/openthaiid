package qualify

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/night-sornram/openthaiid/internal/pcsc"
)

func TestRunSyntheticSmokeE1E2E3(t *testing.T) {
	shared := pcsc.NewSyntheticState()
	mon := pcsc.NewSyntheticWithState(shared)
	rdb := pcsc.NewSyntheticWithState(shared)
	rdb.SeedDemoCard()

	toggle := func(present bool) error {
		const id = "synthetic-acr39u"
		if present {
			mon.InsertCard(id)
		} else {
			mon.RemoveCard(id)
		}
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rep, err := Run(ctx, Hardware{Monitor: mon, Read: rdb}, Options{
		Version:      "0.1.0-rc.1-test",
		Synthetic:    true,
		E2Count:      3,
		E3Count:      2,
		E1Budget:     2 * time.Second,
		StepTimeout:  2 * time.Second,
		SkipOperator: true,
		ToggleCard:   toggle,
	})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := rep.MarshalSafe()
	if err != nil {
		t.Fatal(err)
	}
	if hit := FirstForbidden(raw); hit != "" {
		t.Fatalf("report leaked %q: %s", hit, raw)
	}
	if checkResult(rep, "E1") != ResultPass {
		t.Fatalf("E1 %s", raw)
	}
	if checkResult(rep, "E2") != ResultPass {
		t.Fatalf("E2 %s", raw)
	}
	if checkResult(rep, "E3") != ResultPass {
		t.Fatalf("E3 %s", raw)
	}
	if checkResult(rep, "E7") != ResultSkip {
		t.Fatalf("E7 should skip without operator, got %s", checkResult(rep, "E7"))
	}
	if rep.Overall != ResultIncomplete {
		t.Fatalf("overall %s (want INCOMPLETE with operator skips)", rep.Overall)
	}
	text := rep.Text()
	if hit := FirstForbidden([]byte(text)); hit != "" {
		t.Fatalf("text leaked %q: %s", hit, text)
	}
	for _, want := range []string{"E1", "PASS", "technician diagnostics"} {
		if !strings.Contains(text, want) {
			t.Fatalf("text missing %q:\n%s", want, text)
		}
	}

	var parsed map[string]any
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed["pii"] == nil || parsed["pii"] == "" {
		t.Fatal("missing pii banner")
	}
}

func TestRunSyntheticNoCardFailsE1(t *testing.T) {
	mon := pcsc.NewSynthetic()
	rdb := pcsc.NewSyntheticWithState(mon.State())
	mon.AddReader("r1", "empty reader")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rep, err := Run(ctx, Hardware{Monitor: mon, Read: rdb}, Options{
		Version:      "test",
		Synthetic:    true,
		E2Count:      1,
		E3Count:      1,
		E1Budget:     200 * time.Millisecond,
		SkipOperator: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if checkResult(rep, "E1") != ResultFail {
		t.Fatalf("E1 want FAIL, got %s", checkResult(rep, "E1"))
	}
	var e1 Check
	for _, c := range rep.Checks {
		if c.ID == "E1" {
			e1 = c
		}
	}
	if e1.ErrorCode != "NO_CARD" {
		t.Fatalf("E1 code %q", e1.ErrorCode)
	}
	if checkResult(rep, "E2") != ResultSkip {
		t.Fatalf("E2 should skip after E1 fail")
	}
	raw, err := json.Marshal(rep)
	if err != nil {
		t.Fatal(err)
	}
	if hit := FirstForbidden(raw); hit != "" {
		t.Fatalf("leaked %q", hit)
	}
}

func TestMVPPresent(t *testing.T) {
	if mvpPresent(ReadShape{}) {
		t.Fatal("empty")
	}
	if !mvpPresent(ReadShape{FieldsPresent: []string{"citizenId", "thaiName", "birthDate", "portrait"}}) {
		t.Fatal("want true")
	}
}
