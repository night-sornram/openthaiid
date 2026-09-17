package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/night-sornram/openthaiid/internal/config"
	"github.com/night-sornram/openthaiid/internal/doctor"
	"github.com/night-sornram/openthaiid/internal/pcsc"
)

func runDoctor(args []string) int {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var (
		cfgPath string
		port    int
		bind    string
	)
	fs.StringVar(&cfgPath, "config", "", "")
	fs.IntVar(&port, "port", 0, "")
	fs.StringVar(&bind, "bind", "", "")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "usage: openthaiid doctor [-config path] [-port n] [-bind addr]\n")
		return 2
	}
	cfg := config.Default()
	path := cfgPath
	if path == "" {
		if p, err := config.UserConfigPath(); err == nil {
			path = p
		}
	}
	if path != "" {
		loaded, err := config.LoadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "doctor: %v\n", err)
			return 1
		}
		cfg = loaded
	}
	config.OverlayEnv(&cfg)
	if port != 0 {
		cfg.Port = port
	}
	if bind != "" {
		cfg.BindAddress = bind
	}
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "doctor: %v\n", err)
		return 1
	}

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	base := "http://" + cfg.Addr()
	snap := doctor.Collect(ctx, doctor.Options{
		Version: version,
		Config:  cfg,
		Helper:  func(ctx context.Context) (*doctor.Snapshot, error) { return fetchDoctorSnapshot(ctx, base) },
		PCSC:    listPCSC,
	})
	fmt.Print(doctor.Format(snap))
	return doctor.ExitCode(snap)
}

func fetchDoctorSnapshot(ctx context.Context, base string) (*doctor.Snapshot, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(base, "/")+"/v1/helper/doctor", nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 800 * time.Millisecond}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	var snap doctor.Snapshot
	if err := json.NewDecoder(resp.Body).Decode(&snap); err != nil {
		return nil, err
	}
	if snap.Helper == "" {
		snap.Helper = doctor.HelperRunning
	}
	return &snap, nil
}

func listPCSC(ctx context.Context) (doctor.Inventory, error) {
	backend, err := pcsc.NewOSBackend()
	if err != nil {
		return doctor.Inventory{}, err
	}
	if err := backend.Establish(ctx); err != nil {
		return doctor.Inventory{}, err
	}
	defer func() { _ = backend.Release(context.Background()) }()
	readers, err := backend.Readers(ctx)
	if err != nil {
		return doctor.Inventory{}, err
	}
	inv := doctor.Inventory{Backend: backend.Name(), Readers: make([]doctor.Reader, 0, len(readers))}
	for _, r := range readers {
		row := doctor.Reader{Name: r.Name}
		if present, err := backend.CardPresent(ctx, r.ID); err == nil {
			if present {
				row.Presence = "present"
			} else {
				row.Presence = "absent"
			}
		}
		inv.Readers = append(inv.Readers, row)
	}
	return inv, nil
}
