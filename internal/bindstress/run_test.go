package bindstress

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/night-sornram/openthaiid/internal/pcsc"
	"github.com/night-sornram/openthaiid/internal/qualify"
)

func syntheticPair() (Hardware, func(bool) error) {
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
	return Hardware{Monitor: mon, Read: rdb}, toggle
}

func TestRunSyntheticSmoke(t *testing.T) {
	hw, toggle := syntheticPair()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	rep, err := Run(ctx, hw, Options{
		Version:    "0.1.0-rc.9-test",
		Synthetic:  true,
		Cycles:     3,
		StepBudget: 2 * time.Second,
		WaitBudget: 400 * time.Millisecond,
		ToggleCard: toggle,
	})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := rep.MarshalSafe()
	if err != nil {
		t.Fatal(err)
	}
	if hit := qualify.FirstForbidden(raw); hit != "" {
		t.Fatalf("report leaked %q: %s", hit, raw)
	}
	if hit := firstAPDUDump(raw); hit != "" {
		t.Fatalf("report leaked APDU %q: %s", hit, raw)
	}
	for _, id := range []string{"B1", "B2", "B3", "B4", "B5", "B6", "B7", "B8", "B9"} {
		if checkResult(rep, id) != ResultPass {
			t.Fatalf("%s %s\n%s", id, checkResult(rep, id), raw)
		}
	}
	if rep.Mode != "synthetic-smoke" {
		t.Fatalf("mode %s", rep.Mode)
	}
	if !rep.Evidence.NeedsACR39U {
		t.Fatal("synthetic must still need ACR39U")
	}
	if len(rep.Evidence.SyntheticDoesNot) != len(BindingIDs) {
		t.Fatalf("syntheticDoesNotClose %v", rep.Evidence.SyntheticDoesNot)
	}
	if len(rep.Evidence.ThisHostMayEvidence) != 0 {
		t.Fatalf("synthetic must not claim host evidence: %v", rep.Evidence.ThisHostMayEvidence)
	}
	if rep.Overall != ResultPass {
		t.Fatalf("overall %s\n%s", rep.Overall, raw)
	}

	text := rep.Text()
	if hit := qualify.FirstForbidden([]byte(text)); hit != "" {
		t.Fatalf("text leaked %q", hit)
	}
	for _, want := range []string{"B1", "PASS", "syntheticDoesNotClose", "needsACR39U"} {
		if !strings.Contains(text, want) {
			t.Fatalf("text missing %q:\n%s", want, text)
		}
	}

	var parsed map[string]any
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed["schema"] != Schema {
		t.Fatalf("schema %v", parsed["schema"])
	}
	ev, _ := parsed["evidence"].(map[string]any)
	if ev == nil {
		t.Fatal("missing evidence")
	}
	if _, ok := ev["thisHostMayEvidence"]; ok {
		t.Fatalf("synthetic report should omit thisHostMayEvidence, got %v", ev["thisHostMayEvidence"])
	}
	pii, _ := parsed["pii"].(string)
	if pii == "" || strings.Contains(strings.ToLower(pii), "stored") && !strings.Contains(pii, "never") {
		// pii field must say values are omitted
	}
	if !strings.Contains(pii, "omitted") {
		t.Fatalf("pii %q", pii)
	}

	by := checksByID(rep)
	b4 := by["B4"]
	if b4.Metrics["toggleOK"] != 3 {
		t.Fatalf("toggleOK %v", b4.Metrics["toggleOK"])
	}
	b6 := by["B6"]
	if b6.Metrics["selectOK"] != 3 {
		t.Fatalf("selectOK %v", b6.Metrics["selectOK"])
	}
	if b6.Metrics["respLenMin"] != 2 || b6.Metrics["respLenMax"] != 2 {
		t.Fatalf("select len %v", b6.Metrics)
	}
}

func TestRunSyntheticNoAPDUOrPIIInReport(t *testing.T) {
	hw, toggle := syntheticPair()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	rep, err := Run(ctx, hw, Options{
		Version:    "0.1.0-rc.9-test",
		Synthetic:  true,
		Cycles:     2,
		WaitBudget: 300 * time.Millisecond,
		ToggleCard: toggle,
	})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := rep.MarshalSafe()
	if err != nil {
		t.Fatal(err)
	}
	blob := string(raw)
	for _, s := range []string{
		"1234567890121", "สมชาย", "Somchai", "/9j/", "data:image",
		"00A4", "00a4", "A0 00 00 00 54 48", "80 B0", "80b00004",
	} {
		if strings.Contains(blob, s) {
			t.Fatalf("report contained %q", s)
		}
	}
}

func TestMarshalSafeRefusesPII(t *testing.T) {
	rep := NewReport("test", "synthetic-smoke", 1)
	rep.Checks = []Check{{
		ID:     "B1",
		Name:   "x",
		Result: ResultPass,
		Detail: "1234567890121",
	}}
	if _, err := rep.MarshalSafe(); err == nil {
		t.Fatal("expected PII block")
	}
}

func TestMarshalSafeRefusesAPDUHex(t *testing.T) {
	rep := NewReport("test", "synthetic-smoke", 1)
	rep.Checks = []Check{{
		ID:     "B6",
		Name:   "x",
		Result: ResultPass,
		Detail: "00A4040008A000000054480001",
	}}
	if _, err := rep.MarshalSafe(); err == nil {
		t.Fatal("expected APDU block")
	}
}

func TestHostEvidenceSyntheticVsHardware(t *testing.T) {
	syn := HostEvidence("linux", "amd64", true)
	if len(syn.ThisHostMayEvidence) != 0 {
		t.Fatalf("synthetic evidence %v", syn.ThisHostMayEvidence)
	}
	if !syn.NeedsACR39U {
		t.Fatal("needs ACR39U")
	}
	hw := HostEvidence("linux", "amd64", false)
	if !contains(hw.ThisHostMayEvidence, "P4") || !contains(hw.ThisHostMayEvidence, "P10") {
		t.Fatalf("linux/amd64 %v", hw.ThisHostMayEvidence)
	}
	if contains(hw.ThisHostMayEvidence, "P2") {
		t.Fatal("linux must not claim P2")
	}
	if !contains(hw.StillNeedsOtherHost, "P2") || !contains(hw.StillNeedsOtherHost, "P6") {
		t.Fatalf("missing other hosts %v", hw.StillNeedsOtherHost)
	}
	win := HostEvidence("windows", "amd64", false)
	if !contains(win.ThisHostMayEvidence, "P6") {
		t.Fatalf("windows %v", win.ThisHostMayEvidence)
	}
	if contains(win.ThisHostMayEvidence, "P10") {
		t.Fatal("windows must not claim cgo P10")
	}
	mac := HostEvidence("darwin", "arm64", false)
	if !contains(mac.ThisHostMayEvidence, "P2") || !contains(mac.ThisHostMayEvidence, "P10") {
		t.Fatalf("darwin/arm64 %v", mac.ThisHostMayEvidence)
	}
}

func TestHardwareModeSkipsPresenceToggle(t *testing.T) {
	hw, _ := syntheticPair()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	rep, err := Run(ctx, hw, Options{
		Version:    "0.1.0-rc.9-test",
		Synthetic:  false,
		Cycles:     2,
		WaitBudget: 300 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	if checkResult(rep, "B4") != ResultSkip {
		t.Fatalf("B4 want SKIP got %s", checkResult(rep, "B4"))
	}
	if checkResult(rep, "B1") != ResultPass {
		t.Fatalf("B1 %s", checkResult(rep, "B1"))
	}
	if checkResult(rep, "B5") != ResultPass {
		// seeded card is still present; Open should work
		t.Fatalf("B5 %s", checkResult(rep, "B5"))
	}
	if !rep.Evidence.NeedsACR39U {
		t.Fatal("still needs ACR39U for a real claim")
	}
	if len(rep.Evidence.SyntheticDoesNot) != 0 {
		t.Fatalf("hardware mode should not fill syntheticDoesNotClose: %v", rep.Evidence.SyntheticDoesNot)
	}
	if rep.Mode != "hardware" {
		t.Fatalf("mode %s", rep.Mode)
	}
}

func TestNoReadersSkipsConnect(t *testing.T) {
	shared := pcsc.NewSyntheticState()
	mon := pcsc.NewSyntheticWithState(shared)
	rdb := pcsc.NewSyntheticWithState(shared)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	rep, err := Run(ctx, Hardware{Monitor: mon, Read: rdb}, Options{
		Version:    "test",
		Synthetic:  true,
		Cycles:     2,
		WaitBudget: 200 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	if checkResult(rep, "B1") != ResultPass {
		t.Fatalf("B1 %s", checkResult(rep, "B1"))
	}
	if checkResult(rep, "B2") != ResultPass {
		t.Fatalf("B2 %s", checkResult(rep, "B2"))
	}
	if checkResult(rep, "B4") != ResultSkip {
		t.Fatalf("B4 want SKIP without ToggleCard, got %s", checkResult(rep, "B4"))
	}
	if checkResult(rep, "B5") != ResultSkip {
		t.Fatalf("B5 want SKIP got %s", checkResult(rep, "B5"))
	}
	if checkResult(rep, "B6") != ResultSkip {
		t.Fatalf("B6 want SKIP got %s", checkResult(rep, "B6"))
	}
}

func checksByID(rep *Report) map[string]Check {
	out := map[string]Check{}
	for _, c := range rep.Checks {
		out[c.ID] = c
	}
	return out
}

func contains(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}
