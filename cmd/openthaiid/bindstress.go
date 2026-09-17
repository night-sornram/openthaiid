package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/night-sornram/openthaiid/internal/bindstress"
	"github.com/night-sornram/openthaiid/internal/pcsc"
	"github.com/night-sornram/openthaiid/internal/qualify"
)

const defaultBindStressReport = "openthaiid-bind-stress.json"

func runBindStress(args []string) int {
	fs := flag.NewFlagSet("bind-stress", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var (
		synthetic  bool
		cycles     int
		reportPath string
	)
	fs.BoolVar(&synthetic, "synthetic", false, "practice backend (harness smoke; not a hardware binding pass)")
	fs.IntVar(&cycles, "cycles", 20, "establish/list/probe/release loops")
	fs.StringVar(&reportPath, "report", defaultBindStressReport, "write a single PII-free JSON report to this path")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "bind-stress: %v\n", err)
		fmt.Fprintln(os.Stderr, "usage: openthaiid bind-stress [-synthetic] [-cycles N] [-report path]")
		fmt.Fprintln(os.Stderr, "writes one redacted JSON report (default ./openthaiid-bind-stress.json).")
		fmt.Fprintln(os.Stderr, "Never prints citizen ID, names, address, photo bytes, or raw APDUs.")
		return 2
	}
	if reportPath == "" {
		reportPath = defaultBindStressReport
	}
	absReport, err := filepath.Abs(reportPath)
	if err != nil {
		absReport = reportPath
	}

	printSafe := func(w io.Writer, s string) bool {
		if hit := qualify.FirstForbidden([]byte(s)); hit != "" {
			fmt.Fprintln(os.Stderr, "bind-stress: refused to print (would contain personal or photo data)")
			return false
		}
		fmt.Fprint(w, s)
		return true
	}

	banner := "bind-stress: hardware mode — PC/SC connect/list/probe only; never prints card fields or APDUs\n"
	if synthetic {
		banner = "bind-stress: synthetic smoke (does not close P2–P10 / ACR39U binding audit)\n"
	}
	banner += "A single redacted JSON report will be written to:\n  " + absReport + "\n"
	banner += "See docs/bind-stress.md. Hardware: omit -synthetic.\n"
	if !printSafe(os.Stdout, banner) {
		return 1
	}

	var hw bindstress.Hardware
	var toggle func(present bool) error
	if synthetic {
		shared := pcsc.NewSyntheticState()
		mon := pcsc.NewSyntheticWithState(shared)
		rdb := pcsc.NewSyntheticWithState(shared)
		rdb.SeedDemoCard()
		hw = bindstress.Hardware{Monitor: mon, Read: rdb}
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
			fmt.Fprintf(os.Stderr, "bind-stress: pcsc monitor: %v\n", err)
			return 1
		}
		rdb, err := pcsc.NewOSBackend()
		if err != nil {
			fmt.Fprintf(os.Stderr, "bind-stress: pcsc read: %v\n", err)
			return 1
		}
		hw = bindstress.Hardware{Monitor: mon, Read: rdb}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	rep, err := bindstress.Run(ctx, hw, bindstress.Options{
		Version:    version,
		Synthetic:  synthetic,
		Cycles:     cycles,
		ToggleCard: toggle,
		Log:        func(s string) { _ = printSafe(os.Stdout, s+"\n") },
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "bind-stress: %v\n", err)
		_ = os.Remove(absReport)
		return 1
	}

	text := rep.Text()
	if !printSafe(os.Stdout, text) {
		_ = os.Remove(absReport)
		return 1
	}

	if err := rep.WriteFile(absReport); err != nil {
		fmt.Fprintf(os.Stderr, "bind-stress: write report: %v\n", err)
		_ = os.Remove(absReport)
		return 1
	}
	if !printSafe(os.Stdout, "report: "+absReport+"\n") {
		_ = os.Remove(absReport)
		return 1
	}

	switch rep.Overall {
	case bindstress.ResultFail:
		return 1
	default:
		return 0
	}
}
