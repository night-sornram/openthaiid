package qualify

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/night-sornram/openthaiid/internal/card"
	"github.com/night-sornram/openthaiid/internal/pcsc"
	"github.com/night-sornram/openthaiid/internal/reader"
)

// OperatorIO is the Friday keyboard for E3 (hardware) and E4–E7 / M1.
// Implementations must refuse to store free text or personal data.
type OperatorIO interface {
	Ask(step OperatorStep) Result
	Pause(prompt string) error
}

// Options control a qualification run.
type Options struct {
	Version      string
	Synthetic    bool
	E2Count      int
	E3Count      int
	E1Budget     time.Duration
	StepTimeout  time.Duration
	SkipOperator bool
	// ToggleCard, when set (synthetic), inserts/removes the demo card.
	ToggleCard func(present bool) error
	Operator   OperatorIO
	Log        func(string)
}

func (o *Options) defaults() {
	if o.Version == "" {
		o.Version = "dev"
	}
	if o.E2Count <= 0 {
		o.E2Count = 100
	}
	if o.E3Count <= 0 {
		o.E3Count = 20
	}
	if o.E1Budget <= 0 {
		o.E1Budget = 2 * time.Second
	}
	if o.StepTimeout <= 0 {
		o.StepTimeout = 60 * time.Second
	}
	if o.Log == nil {
		o.Log = func(string) {}
	}
}

// Hardware is the PC/SC pair used by the helper.
type Hardware struct {
	Monitor pcsc.Backend
	Read    pcsc.Backend
}

// Run executes E1–E3 (automated) and records E4–E7 / M1 as SKIP when
// SkipOperator is set. It never logs field values.
func Run(ctx context.Context, hw Hardware, opt Options) (*Report, error) {
	opt.defaults()
	mode := "hardware"
	if opt.Synthetic {
		mode = "synthetic-smoke"
	}
	rep := NewReport(opt.Version, mode)
	engine := card.NewEngine(card.DefaultProfile())
	mgr := reader.NewManager(hw.Monitor, hw.Read)

	startErr := mgr.Start(ctx)
	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = mgr.Stop(stopCtx)
	}()

	runE1(ctx, rep, mgr, startErr, opt)
	fillDiagnostics(rep, mgr, opt)

	if checkResult(rep, "E1") == ResultPass {
		runE2(ctx, rep, mgr, engine, opt)
		runE3(ctx, rep, mgr, opt)
	} else {
		rep.AddCheck(Check{ID: "E2", Name: "repeated MVP reads", Result: ResultSkip, Detail: "skipped; E1 failed"})
		rep.AddCheck(Check{ID: "E3", Name: "insertion/removal cycles", Result: ResultSkip, Detail: "skipped; E1 failed"})
	}

	runOperatorChecks(rep, opt)

	fillDiagnostics(rep, mgr, opt)
	rep.Finalize()
	if _, err := rep.MarshalSafe(); err != nil {
		return nil, err
	}
	return rep, nil
}

func checkResult(r *Report, id string) Result {
	for _, c := range r.Checks {
		if c.ID == id {
			return c.Result
		}
	}
	return ResultSkip
}

func fillDiagnostics(rep *Report, mgr *reader.Manager, opt Options) {
	snap := mgr.Snapshot()
	rep.Diagnostics.Backend = snap.BackendName
	rep.Diagnostics.BackendStatus = snap.Backend
	rep.Diagnostics.Practice = opt.Synthetic || snap.BackendName == "synthetic"
	rep.Diagnostics.ReaderCount = len(snap.Readers)
	rep.Diagnostics.Readers = nil
	rep.Diagnostics.CardPresent = false
	for _, rd := range snap.Readers {
		rep.Diagnostics.Readers = append(rep.Diagnostics.Readers, SanitizeLine(rd.Name))
		if rd.Presence == pcsc.PresencePresent {
			rep.Diagnostics.CardPresent = true
		}
	}
}

func runE1(ctx context.Context, rep *Report, mgr *reader.Manager, startErr error, opt Options) {
	opt.Log("E1 reader/card detection")
	t0 := time.Now()
	if startErr != nil {
		code, msg := errCode(startErr)
		rep.AddCheck(Check{
			ID:         "E1",
			Name:       "reader/card detection within 2s",
			Result:     ResultFail,
			DurationMS: time.Since(t0).Milliseconds(),
			ErrorCode:  code,
			Detail:     msg,
		})
		return
	}
	deadline := t0.Add(opt.E1Budget)
	var snap reader.Snapshot
	for {
		snap = mgr.Snapshot()
		if hasReader(snap) && hasCard(snap) && snap.Backend == "available" {
			break
		}
		if time.Now().After(deadline) {
			break
		}
		select {
		case <-ctx.Done():
			rep.AddCheck(Check{ID: "E1", Name: "reader/card detection within 2s", Result: ResultFail, DurationMS: time.Since(t0).Milliseconds(), ErrorCode: "CANCELLED"})
			return
		case <-time.After(20 * time.Millisecond):
		}
	}
	ms := time.Since(t0).Milliseconds()
	c := Check{
		ID:         "E1",
		Name:       "reader/card detection within 2s",
		DurationMS: ms,
		Metrics: map[string]any{
			"budgetMs":    opt.E1Budget.Milliseconds(),
			"readerCount": len(snap.Readers),
			"backend":     snap.Backend,
		},
	}
	switch {
	case snap.Backend != "available" && snap.Backend != "":
		c.Result = ResultFail
		c.ErrorCode = "PCSC_UNAVAILABLE"
		c.Detail = "card service " + snap.Backend
	case !hasReader(snap):
		c.Result = ResultFail
		c.ErrorCode = "NO_READER"
		c.Detail = "no PC/SC reader in inventory"
	case !hasCard(snap):
		c.Result = ResultFail
		c.ErrorCode = "NO_CARD"
		c.Detail = "reader present; insert a Thai ID card"
	case ms > opt.E1Budget.Milliseconds():
		c.Result = ResultFail
		c.ErrorCode = "READ_TIMEOUT"
		c.Detail = fmt.Sprintf("detected after budget (%dms > %dms)", ms, opt.E1Budget.Milliseconds())
	default:
		c.Result = ResultPass
		c.Detail = fmt.Sprintf("reader+card visible in %dms", ms)
	}
	rep.AddCheck(c)
}

func runE2(ctx context.Context, rep *Report, mgr *reader.Manager, engine *card.Engine, opt Options) {
	opt.Log(fmt.Sprintf("E2 %d repeated reads (field names / JPEG flags only)", opt.E2Count))
	t0 := time.Now()
	passed := 0
	failed := 0
	var minMS, maxMS, sumMS int64
	minMS = -1
	errorCodes := map[string]int{}
	var failures []map[string]any
	var sample *ReadShape

	fields := append([]card.Field(nil), card.SupportedFields...)
	for i := 1; i <= opt.E2Count; i++ {
		if ctx.Err() != nil {
			break
		}
		rt0 := time.Now()
		res, err := mgr.Read(ctx, reader.ReadRequest{
			Fields:  fields,
			Execute: engine.Execute,
		})
		dur := time.Since(rt0).Milliseconds()
		sumMS += dur
		if minMS < 0 || dur < minMS {
			minMS = dur
		}
		if dur > maxMS {
			maxMS = dur
		}
		if err != nil {
			failed++
			code, _ := errCode(err)
			errorCodes[code]++
			if len(failures) < 20 {
				failures = append(failures, map[string]any{"n": i, "code": code, "durationMs": dur})
			}
			continue
		}
		shape := SummarizeRead(res)
		if sample == nil {
			cp := shape
			sample = &cp
		}
		if !mvpPresent(shape) {
			failed++
			errorCodes["EMPTY"]++
			if len(failures) < 20 {
				failures = append(failures, map[string]any{"n": i, "code": "EMPTY", "durationMs": dur})
			}
			continue
		}
		passed++
	}
	avg := int64(0)
	n := passed + failed
	if n > 0 {
		avg = sumMS / int64(n)
	}
	c := Check{
		ID:         "E2",
		Name:       "repeated MVP reads (no field values stored)",
		DurationMS: time.Since(t0).Milliseconds(),
		Metrics: map[string]any{
			"requested": opt.E2Count,
			"passed":    passed,
			"failed":    failed,
			"durationMs": map[string]int64{
				"min": minOrZero(minMS),
				"max": maxMS,
				"avg": avg,
			},
			"errorCodes": errorCodes,
			"failures":   failures,
		},
	}
	if sample != nil {
		c.Metrics["sample"] = sample
	}
	if failed == 0 && passed == opt.E2Count {
		c.Result = ResultPass
		c.Detail = fmt.Sprintf("%d/%d reads ok", passed, opt.E2Count)
	} else {
		c.Result = ResultFail
		if len(errorCodes) == 1 {
			for code := range errorCodes {
				c.ErrorCode = code
			}
		} else if failed > 0 {
			c.ErrorCode = "READ_FAILED"
		}
		c.Detail = fmt.Sprintf("%d/%d reads ok", passed, opt.E2Count)
	}
	rep.AddCheck(c)
}

func runOperatorChecks(rep *Report, opt Options) {
	if opt.Synthetic {
		opt.Log("E4–E7 / M1 skipped (synthetic smoke). Friday hardware: openthaiid qualify -operator")
		opt.Log("Never record citizen ID, names, address, or photo bytes. See docs/hardware-qualification.md")
	} else {
		opt.Log("E4–E7 / M1 operator prompts (never type card data / never paste a photo)")
	}
	for _, step := range OperatorChecks() {
		if opt.Synthetic {
			rep.AddCheck(Check{ID: step.ID, Name: step.Name, Result: ResultSkip, Detail: "synthetic smoke; not a hardware pass. " + step.Detail})
			continue
		}
		if !opt.SkipOperator && opt.Operator != nil {
			opt.Log("prompt " + step.ID)
			res := opt.Operator.Ask(step)
			detail := step.Detail
			switch res {
			case ResultPass:
				detail = "operator recorded PASS — " + step.Detail
			case ResultFail:
				detail = "operator recorded FAIL — " + step.Detail
			default:
				res = ResultSkip
				detail = "operator recorded SKIP — " + step.Detail
			}
			rep.AddCheck(Check{ID: step.ID, Name: step.Name, Result: res, Detail: detail})
			continue
		}
		opt.Log(step.Prompt)
		opt.Log("Recording SKIP. Re-run with -operator on the clinic PC to store pass/fail/skip.")
		rep.AddCheck(Check{
			ID:     step.ID,
			Name:   step.Name,
			Result: ResultSkip,
			Detail: "printed operator prompt; not scored (-skip-operator). " + step.Detail,
		})
	}
}

func runE3(ctx context.Context, rep *Report, mgr *reader.Manager, opt Options) {
	opt.Log(fmt.Sprintf("E3 %d insertion/removal cycles", opt.E3Count))
	t0 := time.Now()
	if opt.ToggleCard == nil {
		if !opt.SkipOperator && opt.Operator != nil {
			runE3Hardware(ctx, rep, mgr, opt, t0)
			return
		}
		opt.Log(e3HardwarePrompt)
		rep.AddCheck(Check{
			ID:     "E3",
			Name:   "insertion/removal cycles",
			Result: ResultSkip,
			Detail: "needs operator card remove/insert: openthaiid qualify -operator (or -synthetic). Never type card data. See docs/hardware-qualification.md",
		})
		return
	}

	ok := 0
	var lastCode string
	var maxCycle int64
	for i := 1; i <= opt.E3Count; i++ {
		ct0 := time.Now()
		if err := opt.ToggleCard(false); err != nil {
			lastCode, _ = errCode(err)
			break
		}
		if err := waitPresence(ctx, mgr, false, opt.StepTimeout); err != nil {
			lastCode, _ = errCode(err)
			break
		}
		if err := opt.ToggleCard(true); err != nil {
			lastCode, _ = errCode(err)
			break
		}
		if err := waitPresence(ctx, mgr, true, opt.StepTimeout); err != nil {
			lastCode, _ = errCode(err)
			break
		}
		cycle := time.Since(ct0).Milliseconds()
		if cycle > maxCycle {
			maxCycle = cycle
		}
		ok++
	}
	c := Check{
		ID:         "E3",
		Name:       "insertion/removal cycles",
		DurationMS: time.Since(t0).Milliseconds(),
		ErrorCode:  lastCode,
		Metrics: map[string]any{
			"requested":  opt.E3Count,
			"ok":         ok,
			"maxCycleMs": maxCycle,
		},
	}
	if ok >= opt.E3Count {
		c.Result = ResultPass
		c.Detail = fmt.Sprintf("%d/%d cycles ok", ok, opt.E3Count)
	} else {
		c.Result = ResultFail
		if c.ErrorCode == "" {
			c.ErrorCode = "CARD_REMOVED"
		}
		c.Detail = fmt.Sprintf("%d/%d cycles ok", ok, opt.E3Count)
	}
	rep.AddCheck(c)
}

const e3HardwarePrompt = `E3 — Insertion / removal cycles (≥20)
ถอดบัตรจนสถานะบอกให้เสียบ แล้วเสียบให้สุด ทำซ้ำ

For each cycle: remove the card fully, then seat it fully.
Do not type citizen ID, names, address, or paste a photo.
Run: openthaiid qualify -operator
Synthetic kit toggles a practice card automatically.`

func runE3Hardware(ctx context.Context, rep *Report, mgr *reader.Manager, opt Options, t0 time.Time) {
	ok := 0
	var lastCode string
	var maxCycle int64
	opt.Log(e3HardwarePrompt)
	for i := 1; i <= opt.E3Count; i++ {
		ct0 := time.Now()
		prompt := fmt.Sprintf("E3 cycle %d/%d — remove the card fully, then press Enter.\nรอบที่ %d/%d — ถอดบัตรออกให้สุด แล้วกด Enter", i, opt.E3Count, i, opt.E3Count)
		if err := opt.Operator.Pause(prompt); err != nil {
			lastCode, _ = errCode(err)
			if lastCode == "" {
				lastCode = "OPERATOR_REFUSED"
			}
			break
		}
		if err := waitPresence(ctx, mgr, false, opt.StepTimeout); err != nil {
			lastCode, _ = errCode(err)
			break
		}
		prompt = fmt.Sprintf("E3 cycle %d/%d — seat the card fully, then press Enter.\nรอบที่ %d/%d — เสียบบัตรให้สุด แล้วกด Enter", i, opt.E3Count, i, opt.E3Count)
		if err := opt.Operator.Pause(prompt); err != nil {
			lastCode, _ = errCode(err)
			if lastCode == "" {
				lastCode = "OPERATOR_REFUSED"
			}
			break
		}
		if err := waitPresence(ctx, mgr, true, opt.StepTimeout); err != nil {
			lastCode, _ = errCode(err)
			break
		}
		cycle := time.Since(ct0).Milliseconds()
		if cycle > maxCycle {
			maxCycle = cycle
		}
		ok++
	}
	c := Check{
		ID:         "E3",
		Name:       "insertion/removal cycles",
		DurationMS: time.Since(t0).Milliseconds(),
		ErrorCode:  lastCode,
		Metrics: map[string]any{
			"requested":  opt.E3Count,
			"ok":         ok,
			"maxCycleMs": maxCycle,
			"mode":       "operator",
		},
	}
	if ok >= opt.E3Count {
		c.Result = ResultPass
		c.Detail = fmt.Sprintf("%d/%d cycles ok", ok, opt.E3Count)
	} else {
		c.Result = ResultFail
		if c.ErrorCode == "" {
			c.ErrorCode = "CARD_REMOVED"
		}
		c.Detail = fmt.Sprintf("%d/%d cycles ok", ok, opt.E3Count)
	}
	rep.AddCheck(c)
}

func waitPresence(ctx context.Context, mgr *reader.Manager, wantPresent bool, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		_ = mgr.Refresh(ctx)
		snap := mgr.Snapshot()
		if wantPresent && hasCard(snap) {
			return nil
		}
		if !wantPresent && hasReader(snap) && !hasCard(snap) {
			return nil
		}
		time.Sleep(15 * time.Millisecond)
	}
	if wantPresent {
		return card.NewError(card.ErrNoCard, "timed out waiting for card present")
	}
	return card.NewError(card.ErrCardRemoved, "timed out waiting for card absent")
}

func hasReader(snap reader.Snapshot) bool {
	return len(snap.Readers) > 0
}

func hasCard(snap reader.Snapshot) bool {
	for _, r := range snap.Readers {
		if r.Presence == pcsc.PresencePresent {
			return true
		}
	}
	return false
}

func mvpPresent(s ReadShape) bool {
	need := map[string]bool{"citizenId": false, "thaiName": false, "birthDate": false}
	for _, n := range s.FieldsPresent {
		if _, ok := need[n]; ok {
			need[n] = true
		}
	}
	for _, ok := range need {
		if !ok {
			return false
		}
	}
	return true
}

func errCode(err error) (string, string) {
	if err == nil {
		return "", ""
	}
	if e := card.AsError(err); e != nil {
		return string(e.Code), SanitizeLine(e.Message)
	}
	msg := SanitizeLine(err.Error())
	low := strings.ToLower(msg)
	if strings.Contains(low, "personal") || strings.Contains(low, "photo") {
		return "OPERATOR_REFUSED", msg
	}
	if strings.Contains(low, "not a terminal") {
		return "NO_TTY", msg
	}
	return "READ_FAILED", "operation failed"
}

func minOrZero(v int64) int64 {
	if v < 0 {
		return 0
	}
	return v
}
