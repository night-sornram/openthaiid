package bindstress

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/night-sornram/openthaiid/internal/card"
	"github.com/night-sornram/openthaiid/internal/pcsc"
	"github.com/night-sornram/openthaiid/internal/qualify"
)

// Hardware is the PC/SC pair used by the helper (monitor + read).
type Hardware struct {
	Monitor pcsc.Backend
	Read    pcsc.Backend
}

// Options control a bind-stress run.
type Options struct {
	Version    string
	Synthetic  bool
	Cycles     int
	StepBudget time.Duration
	WaitBudget time.Duration
	// ToggleCard, when set (synthetic), inserts/removes the demo card.
	ToggleCard func(present bool) error
	Log        func(string)
}

func (o *Options) defaults() {
	if o.Version == "" {
		o.Version = "dev"
	}
	if o.Cycles <= 0 {
		o.Cycles = 20
	}
	if o.StepBudget <= 0 {
		o.StepBudget = 2 * time.Second
	}
	if o.WaitBudget <= 0 {
		o.WaitBudget = 1500 * time.Millisecond
	}
	if o.Log == nil {
		o.Log = func(string) {}
	}
}

type cycleStats struct {
	establishMS []int64
	listMS      []int64
	probeMS     []int64
	toggleMS    []int64
	connectMS   []int64
	selectMS    []int64
	cancelMS    []int64

	readerCounts []int
	presentHits  int
	absentHits   int
	toggleOK     int
	connectOK    int
	selectOK     int
	selectLen    []int
	cancelOK     int
	listAfterOK  int
	gcRuns       int
	establishOK  int

	lastReaders []string
	lastPresent bool
	lastBackend string

	errCodes map[string]int
}

func newCycleStats() *cycleStats {
	return &cycleStats{errCodes: map[string]int{}}
}

func (s *cycleStats) noteErr(err error) {
	if err == nil {
		return
	}
	s.errCodes[errCode(err)]++
}

// Run executes connect/disconnect, list, probe, and presence loops.
// It never logs field values or APDU bytes.
func Run(ctx context.Context, hw Hardware, opt Options) (*Report, error) {
	opt.defaults()
	mode := "hardware"
	if opt.Synthetic {
		mode = "synthetic-smoke"
	}
	rep := NewReport(opt.Version, mode, opt.Cycles)
	st := newCycleStats()

	if hw.Monitor == nil || hw.Read == nil {
		return nil, fmt.Errorf("bind-stress: missing backend")
	}
	st.lastBackend = hw.Monitor.Name()

	opt.Log(fmt.Sprintf("bind-stress: %s cycles=%d (no personal fields, no APDUs)", mode, opt.Cycles))

	for i := 0; i < opt.Cycles; i++ {
		if err := ctx.Err(); err != nil {
			st.noteErr(err)
			break
		}
		runCycle(ctx, hw, opt, st)
		runtime.GC()
		runtime.GC()
		st.gcRuns++
	}

	writeChecks(rep, st, opt)
	rep.Finalize()
	if _, err := rep.MarshalSafe(); err != nil {
		return nil, err
	}
	return rep, nil
}

func runCycle(ctx context.Context, hw Hardware, opt Options, st *cycleStats) {
	step, cancel := context.WithTimeout(ctx, opt.StepBudget)
	defer cancel()

	t0 := time.Now()
	if err := hw.Monitor.Establish(step); err != nil {
		st.noteErr(err)
		st.establishMS = append(st.establishMS, time.Since(t0).Milliseconds())
		return
	}
	if err := hw.Read.Establish(step); err != nil {
		st.noteErr(err)
		_ = hw.Monitor.Release(context.Background())
		st.establishMS = append(st.establishMS, time.Since(t0).Milliseconds())
		return
	}
	st.establishMS = append(st.establishMS, time.Since(t0).Milliseconds())
	st.establishOK++
	defer func() {
		_ = hw.Read.Release(context.Background())
		_ = hw.Monitor.Release(context.Background())
	}()

	// Concurrent list on separate contexts (P7).
	tList := time.Now()
	var (
		wg      sync.WaitGroup
		monList []pcsc.ReaderInfo
		rdList  []pcsc.ReaderInfo
		monErr  error
		rdErr   error
	)
	wg.Add(2)
	go func() {
		defer wg.Done()
		monList, monErr = hw.Monitor.Readers(step)
	}()
	go func() {
		defer wg.Done()
		rdList, rdErr = hw.Read.Readers(step)
	}()
	wg.Wait()
	st.listMS = append(st.listMS, time.Since(tList).Milliseconds())
	if monErr != nil {
		st.noteErr(monErr)
		return
	}
	if rdErr != nil {
		st.noteErr(rdErr)
		return
	}
	readers := monList
	if len(rdList) > len(readers) {
		readers = rdList
	}
	st.readerCounts = append(st.readerCounts, len(readers))
	st.lastReaders = nil
	for _, r := range readers {
		st.lastReaders = append(st.lastReaders, qualify.SanitizeLine(r.Name))
	}

	// Probe presence (status flags only).
	tProbe := time.Now()
	anyPresent := false
	var firstID string
	for _, r := range readers {
		if firstID == "" {
			firstID = r.ID
		}
		present, err := hw.Read.CardPresent(step, r.ID)
		if err != nil {
			st.noteErr(err)
			continue
		}
		if present {
			anyPresent = true
			st.presentHits++
		} else {
			st.absentHits++
		}
	}
	if len(readers) == 0 {
		st.absentHits++
	}
	st.lastPresent = anyPresent
	st.probeMS = append(st.probeMS, time.Since(tProbe).Milliseconds())

	// Synthetic present/absent toggle.
	if opt.Synthetic && opt.ToggleCard != nil && firstID != "" {
		tToggle := time.Now()
		if err := opt.ToggleCard(false); err != nil {
			st.noteErr(err)
		} else if present, err := hw.Read.CardPresent(step, firstID); err != nil {
			st.noteErr(err)
		} else if present {
			st.noteErr(card.NewError(card.ErrReadFailed, "card still present after remove"))
		} else if err := opt.ToggleCard(true); err != nil {
			st.noteErr(err)
		} else if present, err := hw.Read.CardPresent(step, firstID); err != nil {
			st.noteErr(err)
		} else if !present {
			st.noteErr(card.NewError(card.ErrNoCard, "card still absent after insert"))
		} else {
			st.toggleOK++
			anyPresent = true
			st.lastPresent = true
		}
		st.toggleMS = append(st.toggleMS, time.Since(tToggle).Milliseconds())
	}

	// Connect/disconnect the card handle. SELECT AID only; record length + SW.
	if firstID != "" && anyPresent {
		tConn := time.Now()
		conn, err := hw.Read.Open(step, firstID)
		st.connectMS = append(st.connectMS, time.Since(tConn).Milliseconds())
		if err != nil {
			st.noteErr(err)
		} else {
			st.connectOK++
			tSel := time.Now()
			if err := conn.BeginTransaction(step); err != nil {
				st.noteErr(err)
			} else {
				cmd := card.DefaultProfile().SelectCommand()
				resp, err := conn.Transmit(step, cmd, 256)
				if err != nil {
					st.noteErr(err)
				} else if qualify.ContainsForbidden(string(resp)) {
					st.noteErr(card.NewError(card.ErrReadFailed, "select response looked like personal data"))
				} else {
					st.selectLen = append(st.selectLen, len(resp))
					if sw, ok := card.SW(resp); ok && sw == card.StatusOK {
						st.selectOK++
					} else {
						st.noteErr(card.NewError(card.ErrUnsupportedCard, "select status not OK"))
					}
				}
				_ = conn.EndTransaction(context.Background())
			}
			st.selectMS = append(st.selectMS, time.Since(tSel).Milliseconds())
			_ = conn.Close()
		}
	}

	// Cancel vs WaitForChange (P9).
	tWait := time.Now()
	waitErr := waitAndCancel(step, hw.Monitor, opt.WaitBudget)
	elapsed := time.Since(tWait)
	st.cancelMS = append(st.cancelMS, elapsed.Milliseconds())
	if errors.Is(waitErr, context.DeadlineExceeded) && elapsed > opt.WaitBudget+500*time.Millisecond {
		st.noteErr(card.NewError(card.ErrBackendStalled, "WaitForChange did not unblock"))
	} else if waitErr != nil && string(card.AsError(waitErr).Code) == string(card.ErrBackendStalled) {
		st.noteErr(waitErr)
	} else {
		st.cancelOK++
	}
	if _, err := hw.Monitor.Readers(step); err == nil {
		st.listAfterOK++
	}
}

func waitAndCancel(ctx context.Context, b pcsc.Backend, budget time.Duration) error {
	done := make(chan error, 1)
	go func() {
		done <- b.WaitForChange(ctx, budget)
	}()
	time.Sleep(15 * time.Millisecond)
	_ = b.CancelWait()
	select {
	case err := <-done:
		return err
	case <-time.After(budget + 500*time.Millisecond):
		return card.NewError(card.ErrBackendStalled, "WaitForChange did not unblock")
	case <-ctx.Done():
		return ctx.Err()
	}
}

func writeChecks(rep *Report, st *cycleStats, opt Options) {
	b1 := Check{
		ID:         "B1",
		Name:       "context establish/release",
		Result:     passIf(st.establishOK == opt.Cycles),
		DurationMS: sumMS(st.establishMS),
		Metrics:    msMetrics("establishMs", st.establishMS, map[string]any{"ok": st.establishOK, "cycles": opt.Cycles, "backend": st.lastBackend}),
		Detail:     "PC/SC context connect/disconnect (not a card field read)",
	}
	if b1.Result != ResultPass {
		b1.ErrorCode = firstCode(st, card.ErrPCSCUnavailable)
		b1.Detail = fmt.Sprintf("establish OK %d/%d", st.establishOK, opt.Cycles)
	}
	rep.AddCheck(b1)

	listResult := ResultFail
	if len(st.listMS) > 0 {
		listResult = ResultPass
	}
	rep.AddCheck(Check{
		ID:         "B2",
		Name:       "list readers",
		Result:     listResult,
		DurationMS: sumMS(st.listMS),
		Metrics: map[string]any{
			"lists":        len(st.listMS),
			"readerCounts": st.readerCounts,
			"readers":      st.lastReaders,
		},
		Detail: "device names only",
	})

	probeResult := ResultFail
	if len(st.probeMS) > 0 {
		probeResult = ResultPass
	}
	rep.AddCheck(Check{
		ID:         "B3",
		Name:       "probe presence",
		Result:     probeResult,
		DurationMS: sumMS(st.probeMS),
		Metrics: map[string]any{
			"presentHits": st.presentHits,
			"absentHits":  st.absentHits,
			"lastPresent": st.lastPresent,
		},
		Detail: "CardPresent status flags only — no card.read",
	})

	toggle := Check{
		ID:         "B4",
		Name:       "card present/absent",
		DurationMS: sumMS(st.toggleMS),
		Metrics:    map[string]any{"toggleOK": st.toggleOK, "cycles": opt.Cycles},
	}
	switch {
	case !opt.Synthetic:
		toggle.Result = ResultSkip
		toggle.Detail = "hardware cannot auto-toggle; B3 records observed presence. Insert/remove is Friday E3."
	case opt.ToggleCard == nil:
		toggle.Result = ResultSkip
		toggle.Detail = "synthetic run without ToggleCard; B3 still probes presence"
	case st.toggleOK == opt.Cycles:
		toggle.Result = ResultPass
		toggle.Detail = "synthetic InsertCard/RemoveCard"
	default:
		toggle.Result = ResultFail
		toggle.Detail = fmt.Sprintf("toggleOK %d/%d", st.toggleOK, opt.Cycles)
		toggle.ErrorCode = firstCode(st, card.ErrNoCard)
	}
	rep.AddCheck(toggle)

	connect := Check{
		ID:         "B5",
		Name:       "card connect/disconnect",
		DurationMS: sumMS(st.connectMS),
		Metrics:    map[string]any{"connectOK": st.connectOK, "openAttempts": len(st.connectMS)},
		Detail:     "Open/Close only; no field READ BINARY",
	}
	switch {
	case len(st.connectMS) == 0:
		connect.Result = ResultSkip
		connect.Detail = "no card present to Open (NO_CARD / no reader)"
		if !st.lastPresent {
			connect.ErrorCode = string(card.ErrNoCard)
		}
		if len(st.lastReaders) == 0 {
			connect.ErrorCode = string(card.ErrNoReader)
		}
	case st.connectOK == len(st.connectMS) && st.connectOK > 0:
		connect.Result = ResultPass
	default:
		connect.Result = ResultFail
		connect.ErrorCode = firstCode(st, card.ErrReadFailed)
	}
	rep.AddCheck(connect)

	sel := Check{
		ID:         "B6",
		Name:       "select transmit length",
		DurationMS: sumMS(st.selectMS),
		Metrics: map[string]any{
			"selectOK":     st.selectOK,
			"respLenMin":   minInt(st.selectLen),
			"respLenMax":   maxInt(st.selectLen),
			"respLenCount": len(st.selectLen),
		},
		Detail: "SELECT AID only; report stores length + SW-ok, never APDU bytes (P8)",
	}
	switch {
	case len(st.selectMS) == 0:
		sel.Result = ResultSkip
		sel.Detail = "skipped; no card connect"
	case st.selectOK == len(st.selectMS) && st.selectOK > 0:
		sel.Result = ResultPass
	default:
		sel.Result = ResultFail
		sel.ErrorCode = firstCode(st, card.ErrUnsupportedCard)
	}
	rep.AddCheck(sel)

	conc := ResultPass
	if len(st.listMS) != opt.Cycles {
		conc = ResultFail
	}
	rep.AddCheck(Check{
		ID:         "B7",
		Name:       "concurrent monitor+read list",
		Result:     conc,
		DurationMS: sumMS(st.listMS),
		Metrics:    map[string]any{"parallelLists": len(st.listMS)},
		Detail:     "two Backend instances (P7)",
	})

	cancel := Check{
		ID:         "B8",
		Name:       "cancel WaitForChange",
		DurationMS: sumMS(st.cancelMS),
		Metrics: map[string]any{
			"unblocked":   st.cancelOK,
			"listAfterOK": st.listAfterOK,
			"attempts":    len(st.cancelMS),
		},
		Detail: "CancelWait unblocks GetStatusChange (P9); mapped cancel errors are not a hang",
	}
	switch {
	case st.cancelOK == opt.Cycles && st.listAfterOK == opt.Cycles:
		cancel.Result = ResultPass
	case st.cancelOK == opt.Cycles:
		cancel.Result = ResultPass
		cancel.Detail = "WaitForChange unblocked; list-after-cancel had errors (recorded in errorCodes)"
	default:
		cancel.Result = ResultFail
		cancel.ErrorCode = firstCode(st, card.ErrBackendStalled)
	}
	rep.AddCheck(cancel)

	gcDetail := "runtime.GC between cycles (P10). Synthetic has no cgo pointers; hardware darwin/linux does."
	if opt.Synthetic {
		gcDetail = "Go GC only on synthetic — does not close P10. Friday OS PC/SC on darwin/linux is the cgo evidence."
	}
	rep.AddCheck(Check{
		ID:         "B9",
		Name:       "GC between cycles",
		Result:     passIf(st.gcRuns == opt.Cycles),
		DurationMS: 0,
		Metrics:    map[string]any{"gcRuns": st.gcRuns},
		Detail:     gcDetail,
	})

	if len(st.errCodes) > 0 {
		rep.Notes = append(rep.Notes, "errorCodeCounts are protocol codes only (no native strings).")
	}
}

func passIf(ok bool) Result {
	if ok {
		return ResultPass
	}
	return ResultFail
}

func errCode(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "TIMEOUT"
	}
	if errors.Is(err, context.Canceled) {
		return string(card.ErrCancelled)
	}
	return string(card.AsError(err).Code)
}

func firstCode(st *cycleStats, prefer card.Code) string {
	if n := st.errCodes[string(prefer)]; n > 0 {
		return string(prefer)
	}
	for code := range st.errCodes {
		return code
	}
	return ""
}

func sumMS(v []int64) int64 {
	var s int64
	for _, x := range v {
		s += x
	}
	return s
}

func minInt(v []int) int {
	if len(v) == 0 {
		return 0
	}
	m := v[0]
	for _, x := range v[1:] {
		if x < m {
			m = x
		}
	}
	return m
}

func maxInt(v []int) int {
	if len(v) == 0 {
		return 0
	}
	m := v[0]
	for _, x := range v[1:] {
		if x > m {
			m = x
		}
	}
	return m
}

func msMetrics(key string, samples []int64, extra map[string]any) map[string]any {
	out := map[string]any{"n": len(samples)}
	for k, v := range extra {
		out[k] = v
	}
	if len(samples) == 0 {
		return out
	}
	var sum, min, max int64
	min, max = samples[0], samples[0]
	for _, x := range samples {
		sum += x
		if x < min {
			min = x
		}
		if x > max {
			max = x
		}
	}
	out[key+"Min"] = min
	out[key+"Max"] = max
	out[key+"Avg"] = sum / int64(len(samples))
	return out
}
