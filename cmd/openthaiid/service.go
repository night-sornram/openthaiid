package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/night-sornram/openthaiid/internal/audit"
	"github.com/night-sornram/openthaiid/internal/config"
	"github.com/night-sornram/openthaiid/internal/desktop"
)

func runService(args []string) int {
	cmd := "status"
	if len(args) > 0 {
		cmd = args[0]
	}
	svc := desktop.Service{}
	switch cmd {
	case "install":
		if err := svc.Install(); err != nil {
			fmt.Fprintf(os.Stderr, "service install: %v\n", err)
			return 1
		}
		path, _ := svc.UnitFile()
		fmt.Println("Always-on helper: installed (opt-in). No tray. Loopback only.")
		if path != "" && runtime.GOOS != "windows" {
			fmt.Println(path)
		}
		if runtime.GOOS == "windows" {
			if err := desktop.StartWindowsService(); err != nil {
				fmt.Fprintf(os.Stderr, "service start: %v\nRun as Administrator: openthaiid service start\n", err)
				return 1
			}
			fmt.Println("Windows service OpenThaiID is running as LocalSystem.")
			fmt.Println("Ready page: http://127.0.0.1:17891/  (no tray icon in Session 0)")
			fmt.Println("Fallback without SCM: Register-OpenThaiID-Task.ps1 (see docs/installation.md)")
			return 0
		}
		if runtime.GOOS == "linux" {
			if err := desktop.EnableUserUnit(); err != nil {
				fmt.Println("Wrote the user systemd unit. Enable it with:")
				fmt.Println("  systemctl --user enable --now openthaiid.service")
				fmt.Println("Survive logout:")
				fmt.Println("  loginctl enable-linger $USER")
				if err != nil {
					fmt.Fprintf(os.Stderr, "systemctl: %v\n", err)
				}
				return 0
			}
			fmt.Println("systemd --user: openthaiid.service enabled and started.")
			fmt.Println("Survive logout: loginctl enable-linger $USER")
			return 0
		}
		fmt.Println("macOS: load the LaunchDaemon (needs root), for example:")
		fmt.Printf("  sudo launchctl load %s\n", path)
		return 0
	case "uninstall":
		if runtime.GOOS == "linux" {
			_ = desktop.DisableUserUnit()
		}
		if runtime.GOOS == "windows" {
			_ = desktop.StopWindowsService()
		}
		if err := svc.Uninstall(); err != nil {
			fmt.Fprintf(os.Stderr, "service uninstall: %v\n", err)
			return 1
		}
		fmt.Println("Always-on helper: removed")
		return 0
	case "start":
		if runtime.GOOS == "windows" {
			if err := desktop.StartWindowsService(); err != nil {
				fmt.Fprintf(os.Stderr, "service start: %v\n", err)
				return 1
			}
			fmt.Println("OpenThaiID service starting")
			return 0
		}
		if runtime.GOOS == "linux" {
			if err := desktop.EnableUserUnit(); err != nil {
				fmt.Fprintf(os.Stderr, "service start: %v\n", err)
				return 1
			}
			fmt.Println("openthaiid.service started")
			return 0
		}
		fmt.Println("Start the LaunchDaemon with launchctl, or run: openthaiid serve")
		return 0
	case "stop":
		if runtime.GOOS == "windows" {
			if err := desktop.StopWindowsService(); err != nil {
				fmt.Fprintf(os.Stderr, "service stop: %v\n", err)
				return 1
			}
			fmt.Println("OpenThaiID service stopping")
			return 0
		}
		if runtime.GOOS == "linux" {
			_ = desktop.DisableUserUnit()
			fmt.Println("openthaiid.service stopped")
			return 0
		}
		fmt.Println("Stop with launchctl, or: openthaiid quit")
		return 0
	case "status", "":
		on, path, err := svc.Installed()
		if err != nil && runtime.GOOS == "windows" {
			fmt.Fprintf(os.Stderr, "service status: %v\n", err)
			return 1
		}
		if runtime.GOOS == "windows" {
			line, qerr := desktop.QueryWindowsService()
			if qerr != nil {
				fmt.Fprintf(os.Stderr, "service status: %v\n", qerr)
				return 1
			}
			fmt.Println(line)
			return 0
		}
		if on {
			fmt.Println("Always-on helper: installed")
			if path != "" {
				fmt.Println(path)
			}
			if runtime.GOOS == "linux" {
				fmt.Println("Enable linger so it survives logout: loginctl enable-linger $USER")
			}
		} else {
			fmt.Println("Always-on helper: not installed (default is tray / start at login)")
			if path != "" {
				fmt.Println(path)
			}
		}
		return 0
	case "help", "-h", "--help":
		printServiceUsage()
		return 0
	default:
		printServiceUsage()
		return 1
	}
}

func printServiceUsage() {
	fmt.Println("usage: openthaiid service install|uninstall|start|stop|status")
	fmt.Println("Always-on helper without a logged-in tray. Opt-in. Loopback only.")
	fmt.Println("Windows: SCM service (Administrator). Fallback: Register-OpenThaiID-Task.ps1")
	fmt.Println("Linux: systemd --user unit + loginctl enable-linger")
	fmt.Println("macOS: LaunchDaemon plist (KeepAlive). Login-item autostart remains the tray path.")
}

func runAudit(args []string) int {
	dir, err := config.UserDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "audit: %v\n", err)
		return 1
	}
	path := filepath.Join(dir, audit.FileName())
	l, err := audit.Open(path, audit.DefaultCap)
	if err != nil {
		fmt.Fprintf(os.Stderr, "audit: %v\n", err)
		return 1
	}
	defer l.Close()
	exp := l.ExportJSON()
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(exp); err != nil {
		fmt.Fprintf(os.Stderr, "audit: %v\n", err)
		return 1
	}
	if strings.TrimSpace(path) != "" {
		fmt.Fprintf(os.Stderr, "file: %s\n", path)
	}
	return 0
}
