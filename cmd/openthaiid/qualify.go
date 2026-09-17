package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/night-sornram/openthaiid/internal/pcsc"
	"github.com/night-sornram/openthaiid/internal/qualify"
)

const defaultQualifyReport = "openthaiid-qualify.json"

func runQualify(args []string) int {
	fs := flag.NewFlagSet("qualify", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var (
		synthetic    bool
		e2           int
		e3           int
		skipOperator bool
		operator     bool
		reportPath   string
		e1Budget     time.Duration
		stepTimeout  time.Duration
	)
	fs.BoolVar(&synthetic, "synthetic", false, "practice reader (kit smoke; not hardware qualification)")
	fs.IntVar(&e2, "e2-count", 100, "E2 repeated reads")
	fs.IntVar(&e3, "e3-count", 20, "E3 insert/remove cycles (synthetic toggle or -operator)")
	fs.BoolVar(&skipOperator, "skip-operator", true, "skip scoring E4–E7 and M1 (prompts are still printed in hardware mode)")
	fs.BoolVar(&operator, "operator", false, "prompt E3 (hardware) and E4–E7/M1; accepts only pass/fail/skip")
	fs.StringVar(&reportPath, "report", defaultQualifyReport, "write a single PII-free JSON report to this path")
	fs.DurationVar(&e1Budget, "e1-budget", 2*time.Second, "E1 detection budget")
	fs.DurationVar(&stepTimeout, "step-timeout", 60*time.Second, "timeout for one E3 presence wait")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "qualify: %v\n", err)
		fmt.Fprintln(os.Stderr, "usage: openthaiid qualify [-synthetic] [-operator] [-e2-count N] [-e3-count N] [-report path]")
		fmt.Fprintln(os.Stderr, "writes one redacted JSON report (default ./openthaiid-qualify.json). Never prints citizen ID, names, address, or photo bytes.")
		return 2
	}
	if operator {
		skipOperator = false
	}
	if reportPath == "" {
		reportPath = defaultQualifyReport
	}

	absReport, err := filepath.Abs(reportPath)
	if err != nil {
		absReport = reportPath
	}

	printSafe := func(w io.Writer, s string) bool {
		if hit := qualify.FirstForbidden([]byte(s)); hit != "" {
			fmt.Fprintln(os.Stderr, "qualify: refused to print (would contain personal or photo data)")
			return false
		}
		fmt.Fprint(w, s)
		return true
	}

	banner := "qualify: hardware mode — never prints citizen ID, names, address, or portrait bytes\n"
	if synthetic {
		banner = "qualify: synthetic smoke (does not claim ACR39U / E1–E7 hardware pass)\n"
	}
	banner += "A single redacted JSON report will be written to:\n  " + absReport + "\n"
	banner += "If output would contain personal or photo data, it is refused and not written.\n"
	banner += "E4–E7 / M1: " + operatorHint(synthetic, skipOperator) + "\n"
	if !printSafe(os.Stdout, banner) {
		return 1
	}

	var hw qualify.Hardware
	var toggle func(present bool) error
	if synthetic {
		shared := pcsc.NewSyntheticState()
		mon := pcsc.NewSyntheticWithState(shared)
		rdb := pcsc.NewSyntheticWithState(shared)
		rdb.SeedDemoCard()
		hw = qualify.Hardware{Monitor: mon, Read: rdb}
		toggle = func(present bool) error {
			const id = "synthetic-acr39u"
			if present {
				mon.InsertCard(id)
			} else {
				mon.RemoveCard(id)
			}
			return nil
		}
	} else {
		mon, err := pcsc.NewOSBackend()
		if err != nil {
			fmt.Fprintf(os.Stderr, "qualify: pcsc monitor: %v\n", err)
			return 1
		}
		rdb, err := pcsc.NewOSBackend()
		if err != nil {
			fmt.Fprintf(os.Stderr, "qualify: pcsc read: %v\n", err)
			return 1
		}
		hw = qualify.Hardware{Monitor: mon, Read: rdb}
	}

	op := qualify.StdinOperator{
		In:    os.Stdin,
		Out:   os.Stdout,
		IsTTY: stdinIsTTY(),
		Log:   func(s string) { _ = printSafe(os.Stdout, s+"\n") },
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	rep, err := qualify.Run(ctx, hw, qualify.Options{
		Version:      version,
		Synthetic:    synthetic,
		E2Count:      e2,
		E3Count:      e3,
		E1Budget:     e1Budget,
		StepTimeout:  stepTimeout,
		SkipOperator: skipOperator,
		ToggleCard:   toggle,
		Operator:     op,
		Log:          func(s string) { _ = printSafe(os.Stdout, s+"\n") },
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "qualify: %v\n", err)
		_ = os.Remove(absReport)
		return 1
	}

	text := rep.Text()
	if !printSafe(os.Stdout, text) {
		_ = os.Remove(absReport)
		return 1
	}

	if err := rep.WriteFile(absReport); err != nil {
		fmt.Fprintf(os.Stderr, "qualify: write report: %v\n", err)
		_ = os.Remove(absReport)
		return 1
	}
	if !printSafe(os.Stdout, "report: "+absReport+"\n") {
		_ = os.Remove(absReport)
		return 1
	}

	switch rep.Overall {
	case qualify.ResultFail:
		return 1
	default:
		return 0
	}
}

func operatorHint(synthetic, skip bool) string {
	if synthetic {
		return "not scored on synthetic smoke. Hardware: openthaiid qualify -operator"
	}
	if skip {
		return "prompts printed below; not scored. Record pass/fail with -operator"
	}
	return "interactive pass/fail/skip (type those words only)"
}

func stdinIsTTY() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}
