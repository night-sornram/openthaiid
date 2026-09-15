// Command openthaiid is the local Thai National ID bridge.
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/night-sornram/openthaiid/internal/auth"
	"github.com/night-sornram/openthaiid/internal/bridge"
	"github.com/night-sornram/openthaiid/internal/config"
	"github.com/night-sornram/openthaiid/internal/pcsc"
	"github.com/night-sornram/openthaiid/internal/reader"
)

// version is injected via -ldflags at release build time.
var version = "0.0.1-dev"

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "version":
			fmt.Println(version)
			return
		case "doctor":
			os.Exit(runDoctor())
		case "help", "-h", "--help":
			usage()
			return
		}
	}

	fs := flag.NewFlagSet("openthaiid", flag.ExitOnError)
	var (
		cfgPath   string
		port      int
		bind      string
		logLevel  string
		synthetic bool
	)
	fs.StringVar(&cfgPath, "config", "", "path to config.json")
	fs.IntVar(&port, "port", 0, "listen port (default 17891)")
	fs.StringVar(&bind, "bind", "", "loopback bind address")
	fs.StringVar(&logLevel, "log-level", "", "debug|info|warn|error")
	fs.BoolVar(&synthetic, "synthetic", false, "use synthetic PC/SC backend (no hardware)")
	_ = fs.Parse(os.Args[1:])

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
			fatal("config", err)
		}
		cfg = loaded
	}
	if port != 0 {
		cfg.Port = port
	}
	if bind != "" {
		cfg.BindAddress = bind
	}
	if logLevel != "" {
		cfg.LogLevel = logLevel
	}
	if err := cfg.Validate(); err != nil {
		fatal("config", err)
	}

	log := newLogger(cfg.LogLevel)
	instanceID := bridge.NewInstanceID()
	authMgr := auth.NewManager(instanceID, cfg.TrustDuration, cfg.SessionDuration, cfg.ApprovalTimeout, cfg.MaxPendingApprovals)
	authMgr.NonInteractive = !stdinIsTTY()

	var factory readerFactories
	if synthetic {
		// Share one synthetic device state across monitor and read contexts
		// so inventory and card presence stay consistent. Seed one clearly
		// labeled fictional card for the documented demo path.
		shared := pcsc.NewSyntheticState()
		factory = readerFactories{
			monitor: func() (pcsc.Backend, error) {
				b := pcsc.NewSyntheticWithState(shared)
				return b, nil
			},
			read: func() (pcsc.Backend, error) {
				b := pcsc.NewSyntheticWithState(shared)
				b.SeedDemoCard()
				return b, nil
			},
		}
	} else {
		factory = readerFactories{
			monitor: pcsc.NewOSBackend,
			read:    pcsc.NewOSBackend,
		}
	}
	mon, err := factory.monitor()
	if err != nil {
		fatal("pcsc", err)
	}
	rdb, err := factory.read()
	if err != nil {
		fatal("pcsc", err)
	}
	mgr := reader.NewManager(mon, rdb)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := mgr.Start(ctx); err != nil {
		log.Warn("reader manager start failed", "error", err)
	} else {
		defer func() { _ = mgr.Stop(context.Background()) }()
	}

	srv := bridge.New(cfg, log, authMgr, mgr)
	if err := srv.Start(ctx); err != nil {
		fatal("bridge", err)
	}
	log.Info("openthaiid listening",
		"addr", srv.Addr(),
		"ws", cfg.WSURL(),
		"instance", instanceID,
		"backend", backendName(synthetic),
	)
	fmt.Printf("OpenThaiID bridge ready on %s\n", srv.Addr())
	fmt.Printf("WebSocket: %s\n", cfg.WSURL())
	fmt.Println("Terminal commands: status | trust | revoke <origin> | revoke-all | help | quit")
	fmt.Println("This is an independent open-source project and is not affiliated with or endorsed by the Department of Provincial Administration.")

	go terminalLoop(ctx, cancel, authMgr, log)

	<-ctx.Done()
	shutCtx, c2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer c2()
	_ = srv.Shutdown(shutCtx)
	log.Info("shutdown complete")
}

type readerFactories struct {
	monitor func() (pcsc.Backend, error)
	read    func() (pcsc.Backend, error)
}

func backendName(synthetic bool) string {
	if synthetic {
		return "synthetic"
	}
	return "pcsc"
}

func terminalLoop(ctx context.Context, cancel context.CancelFunc, authMgr *auth.Manager, log *slog.Logger) {
	sc := bufio.NewScanner(os.Stdin)
	now := time.Now
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		if !sc.Scan() {
			return
		}
		line := strings.TrimSpace(sc.Text())
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		switch fields[0] {
		case "help":
			fmt.Println("commands: status | trust | allow <id> | deny <id> | revoke <origin> | revoke-all | quit")
		case "status":
			for _, g := range authMgr.TrustedOrigins(now()) {
				fmt.Printf("trusted %s fields=%v until=%s\n", g.Origin, g.Fields, g.ExpiresAt.Format(time.RFC3339))
			}
			for _, p := range authMgr.Pending(now()) {
				fmt.Printf("pending id=%d origin=%s fields=%v code=%s\n", p.ID, p.Origin, p.Fields, p.Comparison)
			}
		case "trust":
			for _, g := range authMgr.TrustedOrigins(now()) {
				fmt.Printf("%s  fields=%v  until=%s\n", g.Origin, g.Fields, g.ExpiresAt.Format(time.RFC3339))
			}
			if len(authMgr.TrustedOrigins(now())) == 0 {
				fmt.Println("(no trusted origins)")
			}
		case "allow", "deny":
			if len(fields) < 2 {
				fmt.Println("usage: allow|deny <request-id>")
				continue
			}
			var id int64
			if _, err := fmt.Sscanf(fields[1], "%d", &id); err != nil {
				fmt.Println("invalid request id")
				continue
			}
			_, err := authMgr.Resolve(id, fields[0] == "allow", now())
			if err != nil {
				fmt.Println("error:", err)
			} else {
				fmt.Printf("%sd %d\n", fields[0], id)
			}
		case "revoke":
			if len(fields) < 2 {
				fmt.Println("usage: revoke <origin>")
				continue
			}
			n := authMgr.RevokeOrigin(auth.Origin(fields[1]))
			fmt.Printf("revoked %d session(s)\n", n)
		case "revoke-all":
			authMgr.RevokeAll()
			fmt.Println("revoked all grants and sessions")
		case "quit", "exit":
			cancel()
			return
		default:
			fmt.Println("unknown command; try help")
		}
	}
}

func runDoctor() int {
	fmt.Printf("openthaiid doctor (%s)\n", version)
	fmt.Println("config: OK (defaults)")
	backend, err := pcsc.NewOSBackend()
	if err != nil {
		fmt.Printf("pcsc: FAIL create backend: %v\n", err)
		return 1
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := backend.Establish(ctx); err != nil {
		fmt.Printf("pcsc: FAIL establish: %v\n", err)
		fmt.Println("hint: on Linux ensure pcscd is running and libpcsclite is installed")
		fmt.Println("hint: on macOS ensure PC/SC.framework is available")
		fmt.Println("hint: on Windows ensure the Smart Card service is running")
		return 1
	}
	defer func() { _ = backend.Release(context.Background()) }()
	readers, err := backend.Readers(ctx)
	if err != nil {
		fmt.Printf("pcsc: FAIL list readers: %v\n", err)
		return 1
	}
	fmt.Printf("pcsc: OK backend=%s readers=%d\n", backend.Name(), len(readers))
	for _, r := range readers {
		// Name only — never card contents.
		fmt.Printf("  reader: %s\n", r.Name)
	}
	fmt.Println("personal fields: not read")
	return 0
}

func newLogger(level string) *slog.Logger {
	var lv slog.Level
	switch level {
	case "debug":
		lv = slog.LevelDebug
	case "warn":
		lv = slog.LevelWarn
	case "error":
		lv = slog.LevelError
	default:
		lv = slog.LevelInfo
	}
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: lv}))
}

func stdinIsTTY() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

func usage() {
	fmt.Println(`openthaiid — local Thai National ID bridge

Usage:
  openthaiid              Run the bridge
  openthaiid doctor       Check PC/SC availability (no personal fields)
  openthaiid version      Print version

Flags (run mode):
  -config path        JSON config file
  -port n             Listen port (default 17891)
  -bind addr          Loopback bind address (default 127.0.0.1)
  -log-level level    debug|info|warn|error
  -synthetic          Use synthetic backend (no hardware)

This is an independent open-source project and is not affiliated with or
endorsed by the Department of Provincial Administration.`)
}

func fatal(what string, err error) {
	fmt.Fprintf(os.Stderr, "%s: %v\n", what, err)
	os.Exit(1)
}
