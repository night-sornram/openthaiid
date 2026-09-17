// Command openthaiid is the local Thai National ID bridge.
package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/night-sornram/openthaiid/internal/audit"
	"github.com/night-sornram/openthaiid/internal/auth"
	"github.com/night-sornram/openthaiid/internal/bridge"
	"github.com/night-sornram/openthaiid/internal/config"
	"github.com/night-sornram/openthaiid/internal/desktop"
	"github.com/night-sornram/openthaiid/internal/pcsc"
	"github.com/night-sornram/openthaiid/internal/reader"
	"github.com/night-sornram/openthaiid/internal/update"
)

// version is injected via -ldflags at release build time.
var version = "0.1.0-rc.9"

func main() {
	if maybeRunWindowsService() {
		return
	}
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "version":
			fmt.Println(version)
			return
		case "doctor":
			os.Exit(runDoctor(os.Args[2:]))
		case "help", "-h", "--help":
			usage()
			return
		case "autostart":
			os.Exit(runAutostart(os.Args[2:]))
		case "status":
			os.Exit(runStatus(os.Args[2:]))
		case "quit":
			os.Exit(runQuit(os.Args[2:]))
		case "open":
			os.Exit(runOpenCmd(os.Args[2:]))
		case "update":
			os.Exit(runUpdate(os.Args[2:]))
		case "qualify":
			os.Exit(runQualify(os.Args[2:]))
		case "bind-stress":
			os.Exit(runBindStress(os.Args[2:]))
		case "service":
			os.Exit(runService(os.Args[2:]))
		case "audit":
			os.Exit(runAudit(os.Args[2:]))
		}
	}

	runHelper(os.Args[1:])
}

func runHelper(args []string) {
	runHelperCtx(context.Background(), args)
}

func runHelperCtx(parent context.Context, args []string) {
	openManager := false
	tray := false
	serve := false
	if len(args) > 0 {
		switch args[0] {
		case "manager":
			openManager = true
			args = args[1:]
		case "tray":
			tray = true
			args = args[1:]
		case "serve":
			serve = true
			args = args[1:]
		}
	}

	fs := flag.NewFlagSet("openthaiid", flag.ExitOnError)
	var (
		cfgPath   string
		port      int
		bind      string
		logLevel  string
		synthetic bool
		tlsCert   string
		tlsKey    string
		tlsPort   int
	)
	fs.StringVar(&cfgPath, "config", "", "path to config.json")
	fs.IntVar(&port, "port", 0, "listen port (default 17891)")
	fs.StringVar(&bind, "bind", "", "loopback bind address")
	fs.StringVar(&logLevel, "log-level", "", "debug|info|warn|error (default info; quiet for clinic staff)")
	fs.BoolVar(&synthetic, "synthetic", false, "use synthetic PC/SC backend (no hardware)")
	fs.BoolVar(&openManager, "open", openManager, "open the local manager in a browser")
	fs.BoolVar(&tray, "tray", tray, "run as a background helper (tray icon on Windows/macOS)")
	fs.BoolVar(&serve, "serve", serve, "run headless (no tray, no terminal) for a service / always-on helper")
	fs.StringVar(&tlsCert, "tls-cert", "", "PEM cert for optional loopback HTTPS/WSS (with -tls-key)")
	fs.StringVar(&tlsKey, "tls-key", "", "PEM private key for optional loopback HTTPS/WSS")
	fs.IntVar(&tlsPort, "tls-port", 0, "HTTPS/WSS port (default 17892; requires cert+key)")
	_ = fs.Parse(args)

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
	config.OverlayEnv(&cfg)
	if port != 0 {
		cfg.Port = port
	}
	if bind != "" {
		cfg.BindAddress = bind
	}
	if logLevel != "" {
		cfg.LogLevel = logLevel
	}
	if tlsCert != "" {
		cfg.TLSCertFile = tlsCert
	}
	if tlsKey != "" {
		cfg.TLSKeyFile = tlsKey
	}
	if tlsPort != 0 {
		cfg.TLSPort = tlsPort
	}
	if err := cfg.Validate(); err != nil {
		fatal("config", err)
	}

	userDir, _ := config.UserDir()
	inst := desktop.Instance{Dir: userDir}
	headless := tray || serve

	var logExtra io.Writer
	if headless {
		if f, err := inst.OpenLog(); err == nil {
			defer f.Close()
			logExtra = f
		}
	}

	levelVar := new(slog.LevelVar)
	log := newLogger(cfg.LogLevel, levelVar, logExtra)

	if headless {
		if existing, ok := lookupRunning(cfg); ok {
			log.Info("helper already running", "page", existing.ReadyPage)
			fmt.Println("OpenThaiID is already running on this computer.")
			if openManager {
				if err := openBrowser(existing.ReadyPage); err != nil {
					fmt.Printf("Open the helper at %s\n", existing.ReadyPage)
				}
			}
			return
		}
	}

	instanceID := bridge.NewInstanceID()
	authMgr := auth.NewManager(instanceID, cfg.TrustDuration, cfg.SessionDuration, cfg.ApprovalTimeout, cfg.MaxPendingApprovals)
	// Approvals are handled in the local Manager UI (and still in the
	// terminal when a TTY is attached). Do not fail closed just because
	// stdin is not a console — clinics often start the bridge from a
	// shortcut or service.

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
				if os.Getenv("OPENTHAIID_SYNTHETIC_READERS") == "2" {
					b.SeedSecondDemoReader()
				}
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

	ctx, cancel := signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := mgr.Start(ctx); err != nil {
		log.Warn("reader manager start failed", "error", err)
	} else {
		defer func() { _ = mgr.Stop(context.Background()) }()
	}

	srv := bridge.New(cfg, log, authMgr, mgr)
	srv.SetRuntime(version, synthetic)
	srv.SetLogLeveler(levelVar)
	srv.SetShutdownRequest(cancel)
	mode := "console"
	if tray {
		mode = "tray"
	}
	if serve {
		mode = "serve"
	}
	srv.SetRunMode(mode)
	if userDir != "" {
		if trail, err := audit.Open(filepath.Join(userDir, audit.FileName()), audit.DefaultCap); err != nil {
			log.Warn("audit log", "error", err)
		} else {
			srv.SetAudit(trail)
			defer trail.Close()
		}
	}
	if err := srv.Start(ctx); err != nil {
		if headless {
			if existing, ok := lookupRunning(cfg); ok {
				log.Info("helper already running", "page", existing.ReadyPage)
				fmt.Println("OpenThaiID is already running on this computer.")
				if openManager {
					_ = openBrowser(existing.ReadyPage)
				}
				return
			}
		}
		fatal("bridge", err)
	}
	if headless {
		if err := inst.WritePID(); err != nil {
			log.Warn("pid file", "error", err)
		} else {
			defer inst.RemovePID()
		}
		if tray {
			desktop.HideConsole()
		}
	}

	managerURL := srv.ManagerURL()
	log.Info("openthaiid listening",
		"addr", srv.Addr(),
		"manager", managerURL,
		"ws", cfg.WSURL(),
		"tls", cfg.TLSEnabled(),
		"wss", cfg.WSSURL(),
		"instance", instanceID,
		"backend", backendName(synthetic),
		"tray", tray,
		"serve", serve,
	)
	if !headless {
		fmt.Printf("OpenThaiID is ready on this computer (%s).\n", srv.Addr())
		fmt.Printf("Clinic staff: keep this window open, plug in the reader, then Fill in the ERP.\n")
		fmt.Printf("Helper page:  %s  (Technician / developer tools are hidden there)\n", managerURL)
		fmt.Printf("WebSocket:    %s\n", cfg.WSURL())
		if cfg.TLSEnabled() {
			fmt.Printf("HTTPS/WSS:    %s  (technician-managed local cert; Firefox/Safari https ERP)\n", cfg.WSSURL())
		}
		fmt.Println("The first time a clinic page asks to read the card, Allow it on the helper page.")
		fmt.Println("Technician commands: status | trust | allow <id> | deny <id> | revoke <origin> | revoke-all | help | quit")
		fmt.Println("This is an independent open-source project and is not affiliated with or endorsed by the Department of Provincial Administration.")
	} else if serve {
		log.Info("headless helper running (no tray)")
	}

	if openManager {
		if err := openBrowser(managerURL); err != nil {
			log.Warn("could not open manager", "error", err, "url", managerURL)
			fmt.Printf("Open the manager at %s\n", managerURL)
		}
	}

	if tray {
		host := desktop.Host{
			ReadyURL:      managerURL,
			TechnicianURL: strings.TrimRight(managerURL, "/") + "/#technician",
			Status: func() string {
				return desktop.Format(srv.HelperSummary())
			},
			OpenURL: openBrowser,
			AutostartOn: func() bool {
				on, _, err := (desktop.Autostart{}).Enabled()
				return err == nil && on
			},
			SetAutostart: func(enabled bool) error {
				a := desktop.Autostart{}
				if enabled {
					return a.Enable()
				}
				return a.Disable()
			},
			Quit: cancel,
			Log:  log,
			CheckUpdate: func() string {
				ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
				defer cancel()
				res, err := update.Check(ctx, version, update.DefaultOptions())
				lang := string(desktop.StaffLang())
				if err != nil {
					return update.Unavailable(version, update.ReasonOffline, update.ReleasesPage).StaffMessage(lang)
				}
				return res.StaffMessage(lang)
			},
			ReleasesURL: update.ReleasesPage,
		}
		err := desktop.RunTray(ctx, host)
		if errors.Is(err, desktop.ErrNoTray) {
			log.Info("no notification area; helper running in the background")
			fmt.Printf("OpenThaiID is running in the background on %s.\n", srv.Addr())
			fmt.Printf("Ready page: %s\n", managerURL)
			fmt.Println("This session has no tray icon (typical on Linux without a desktop). Use the Ready page, or: openthaiid status | openthaiid quit")
			<-ctx.Done()
		} else if err != nil {
			log.Warn("tray", "error", err)
			<-ctx.Done()
		}
	} else if serve {
		<-ctx.Done()
	} else {
		go terminalLoop(ctx, cancel, authMgr, log)
		<-ctx.Done()
	}

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
			origin, err := auth.NormalizeOrigin(fields[1])
			if err != nil {
				fmt.Println("error: invalid origin")
				continue
			}
			n := authMgr.RevokeOrigin(origin)
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

func newLogger(level string, lv *slog.LevelVar, extra io.Writer) *slog.Logger {
	if lv == nil {
		lv = new(slog.LevelVar)
	}
	switch level {
	case "debug":
		lv.Set(slog.LevelDebug)
	case "warn":
		lv.Set(slog.LevelWarn)
	case "error":
		lv.Set(slog.LevelError)
	default:
		lv.Set(slog.LevelInfo)
	}
	var w io.Writer = os.Stderr
	if extra != nil {
		w = io.MultiWriter(os.Stderr, extra)
	}
	return slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{Level: lv}))
}

func usage() {
	fmt.Println(`openthaiid — local Thai National ID helper

Usage:
  openthaiid                 Run the helper in a terminal (technicians / developers)
  openthaiid tray            Run as a background helper (tray on Windows/macOS)
  openthaiid serve           Headless helper (no tray) for a Windows service / systemd / Task Scheduler
  openthaiid manager         Run and open the Ready page in a browser
  openthaiid status          Staff-safe helper / reader / card status (no personal data)
  openthaiid quit            Stop a running helper on this computer
  openthaiid open            Open the Ready page
  openthaiid open technician Open technician / developer tools
  openthaiid autostart       enable | disable | status   (opt-in start at login)
  openthaiid service         install | uninstall | start | stop | status   (always-on, no tray)
  openthaiid audit           Print the redacted operator audit trail JSON (no personal fields)
  openthaiid update          check | download | apply   (GitHub Releases lookup; unsigned; check does not install)
  openthaiid doctor          Desk snapshot: helper, ports, PC/SC, probe, TLS yes/no
                             (no personal fields, tokens, or key paths). Exit 1 only
                             if the helper is down or PC/SC is unavailable.
  openthaiid qualify         Friday kit: E1–E3 + E4–E7/M1 prompts; one JSON report (no personal fields)
                             default report: ./openthaiid-qualify.json
                             -operator records pass/fail/skip only (never type card data)
  openthaiid bind-stress     PC/SC binding stress: connect/list/probe/presence; one JSON report
                             (no personal fields, no APDUs). -synthetic is harness smoke only.
                             Friday with ACR39U: omit -synthetic. See docs/bind-stress.md
  openthaiid version         Print version

Flags (run / tray mode):
  -config path        JSON config file
  -port n             Listen port (default 17891)
  -bind addr          Loopback bind address (default 127.0.0.1)
  -log-level level    debug|info|warn|error (default info; quiet for clinic staff)
  -synthetic          Use a practice reader (no hardware)
  -open               Open the Ready page in a browser
  -tray               Same as the tray subcommand
  -serve              Same as the serve subcommand
  -tls-cert path      Optional loopback HTTPS/WSS certificate (PEM)
  -tls-key path       Matching private key (never commit; see docs/loopback-tls.md)
  -tls-port n         HTTPS/WSS port (default 17892)

Helper page:   http://127.0.0.1:17891/
               Default view is for clinic staff. Technician / developer tools
               are behind a toggle (or #technician).
WebSocket:     ws://127.0.0.1:17891/v1/ws
Optional WSS:  wss://127.0.0.1:17892/v1/ws  (technician-managed local cert)

Clinic staff: install once, start the helper (tray or Start menu), plug reader, Fill.
Technicians:  see docs/technician.md, docs/friday-checklist.md,
              docs/hardware-qualification.md, and docs/bind-stress.md.

There is no backend or frontend to deploy. The ERP calls autofillThaiID.
The helper listens on loopback only. Approvals are unchanged.

This is an independent open-source project and is not affiliated with or
endorsed by the Department of Provincial Administration.`)
}

func fatal(what string, err error) {
	fmt.Fprintf(os.Stderr, "%s: %v\n", what, err)
	os.Exit(1)
}

func lookupRunning(cfg config.Config) (desktop.Summary, bool) {
	base := "http://" + net.JoinHostPort(cfg.BindAddress, fmt.Sprintf("%d", cfg.Port))
	sum, err := fetchHelperSummary(base)
	if err == nil && sum.Helper {
		return sum, true
	}
	if helperHealthz(base) {
		return desktop.Summarize(desktop.Lights{Helper: true, ReadyPage: base + "/", TechnicianPage: base + "/#technician"}), true
	}
	return desktop.Summary{}, false
}
