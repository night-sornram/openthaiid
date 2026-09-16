package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/night-sornram/openthaiid/internal/pcsc"
	"github.com/night-sornram/openthaiid/internal/qualify"
)

func runQualify(args []string) int {
	fs := flag.NewFlagSet("qualify", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var (
		synthetic    bool
		e2           int
		e3           int
		skipOperator bool
		reportPath   string
		e1Budget     time.Duration
		stepTimeout  time.Duration
	)
	fs.BoolVar(&synthetic, "synthetic", false, "practice reader (kit smoke; not hardware qualification)")
	fs.IntVar(&e2, "e2-count", 100, "E2 repeated reads")
	fs.IntVar(&e3, "e3-count", 20, "E3 insert/remove cycles (synthetic toggle)")
	fs.BoolVar(&skipOperator, "skip-operator", true, "skip E4–E7 and M1 (operator checklist)")
	fs.StringVar(&reportPath, "report", "", "write PII-free JSON report to this path")
	fs.DurationVar(&e1Budget, "e1-budget", 2*time.Second, "E1 detection budget")
	fs.DurationVar(&stepTimeout, "step-timeout", 60*time.Second, "timeout for one E3 presence wait")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "qualify: %v\n", err)
		fmt.Fprintln(os.Stderr, "usage: openthaiid qualify [-synthetic] [-e2-count N] [-e3-count N] [-report path]")
		return 2
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
		fmt.Println("qualify: synthetic smoke (does not claim ACR39U / E1–E7 hardware pass)")
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
		fmt.Println("qualify: hardware mode — never prints citizen ID, names, address, or portrait bytes")
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
		Log:          func(s string) { fmt.Println(s) },
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "qualify: %v\n", err)
		return 1
	}

	text := rep.Text()
	if hit := qualify.FirstForbidden([]byte(text)); hit != "" {
		fmt.Fprintln(os.Stderr, "qualify: refused to print diagnostics (would contain personal or photo data)")
		return 1
	}
	fmt.Print(text)

	if reportPath != "" {
		if err := rep.WriteFile(reportPath); err != nil {
			fmt.Fprintf(os.Stderr, "qualify: write report: %v\n", err)
			return 1
		}
		fmt.Println("report:", reportPath)
	}

	switch rep.Overall {
	case qualify.ResultFail:
		return 1
	default:
		return 0
	}
}
