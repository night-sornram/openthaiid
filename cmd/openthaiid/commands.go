package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/night-sornram/openthaiid/internal/config"
	"github.com/night-sornram/openthaiid/internal/desktop"
)

func runAutostart(args []string) int {
	cmd := "status"
	if len(args) > 0 {
		cmd = args[0]
	}
	a := desktop.Autostart{}
	switch cmd {
	case "enable":
		if err := a.Enable(); err != nil {
			fmt.Fprintf(os.Stderr, "autostart enable: %v\n", err)
			return 1
		}
		path, _ := a.File()
		fmt.Println("Start at login: on (opt-in). Takes effect the next time you sign in.")
		fmt.Println(path)
		fmt.Println("The helper still listens only on 127.0.0.1. This does not open a browser at login.")
		return 0
	case "disable":
		if err := a.Disable(); err != nil {
			fmt.Fprintf(os.Stderr, "autostart disable: %v\n", err)
			return 1
		}
		fmt.Println("Start at login: off")
		return 0
	case "status", "":
		on, path, err := a.Enabled()
		if err != nil {
			fmt.Fprintf(os.Stderr, "autostart status: %v\n", err)
			return 1
		}
		if on {
			fmt.Println("Start at login: on")
			fmt.Println(path)
		} else {
			fmt.Println("Start at login: off (default)")
			fmt.Println(path)
		}
		return 0
	case "help", "-h", "--help":
		fmt.Println("usage: openthaiid autostart enable|disable|status")
		fmt.Println("Opt-in only. Off by default. Windows: Startup folder. macOS: LaunchAgent. Linux: XDG autostart.")
		return 0
	default:
		fmt.Fprintf(os.Stderr, "usage: openthaiid autostart enable|disable|status\n")
		return 1
	}
}

func runStatus(args []string) int {
	base, err := helperBaseFromArgs("status", args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "status: %v\n", err)
		return 1
	}
	sum, err := fetchHelperSummary(base)
	if err != nil {
		fmt.Println(desktop.Format(desktop.Summarize(desktop.Lights{
			ReadyPage:      strings.TrimRight(base, "/") + "/",
			TechnicianPage: strings.TrimRight(base, "/") + "/#technician",
		})))
		return 1
	}
	fmt.Println(desktop.Format(sum))
	return 0
}

func runQuit(args []string) int {
	base, err := helperBaseFromArgs("quit", args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "quit: %v\n", err)
		return 1
	}
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Post(base+"/v1/helper/shutdown", "application/json", nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, "OpenThaiID does not appear to be running on this computer.")
		return 1
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "quit: HTTP %d\n", resp.StatusCode)
		return 1
	}
	fmt.Println("OpenThaiID is stopping.")
	return 0
}

func runOpenCmd(args []string) int {
	tech := false
	rest := args
	if len(rest) > 0 && (rest[0] == "technician" || rest[0] == "tech" || rest[0] == "--technician") {
		tech = true
		rest = rest[1:]
	}
	base, err := helperBaseFromArgs("open", rest)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open: %v\n", err)
		return 1
	}
	url := strings.TrimRight(base, "/") + "/"
	if tech {
		url += "#technician"
	}
	if _, err := fetchHelperSummary(base); err != nil && !helperHealthz(base) {
		fmt.Fprintf(os.Stderr, "OpenThaiID is not running. Start it, then open %s\n", url)
		return 1
	}
	if err := openBrowser(url); err != nil {
		fmt.Fprintf(os.Stderr, "could not open browser: %v\nOpen %s\n", err, url)
		return 1
	}
	fmt.Println(url)
	return 0
}

func helperBaseFromArgs(name string, args []string) (string, error) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
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
		return "", err
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
			return "", err
		}
		cfg = loaded
	}
	if port != 0 {
		cfg.Port = port
	}
	if bind != "" {
		cfg.BindAddress = bind
	}
	if err := cfg.Validate(); err != nil {
		return "", err
	}
	return "http://" + net.JoinHostPort(cfg.BindAddress, strconv.Itoa(cfg.Port)), nil
}

func fetchHelperSummary(base string) (desktop.Summary, error) {
	client := &http.Client{Timeout: 800 * time.Millisecond}
	resp, err := client.Get(strings.TrimRight(base, "/") + "/v1/helper/summary")
	if err != nil {
		return desktop.Summary{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return desktop.Summary{}, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	var sum desktop.Summary
	if err := json.NewDecoder(resp.Body).Decode(&sum); err != nil {
		return desktop.Summary{}, err
	}
	return sum, nil
}

func helperHealthz(base string) bool {
	client := &http.Client{Timeout: 800 * time.Millisecond}
	resp, err := client.Get(strings.TrimRight(base, "/") + "/healthz")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}
