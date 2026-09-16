package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/night-sornram/openthaiid/internal/config"
	"github.com/night-sornram/openthaiid/internal/desktop"
	"github.com/night-sornram/openthaiid/internal/update"
)

func runUpdate(args []string) int {
	cmd := "check"
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		cmd = args[0]
		args = args[1:]
	}
	switch cmd {
	case "check":
		return updateCheck(args)
	case "download":
		return updateDownload(args, false)
	case "apply":
		return updateDownload(args, true)
	case "help", "-h", "--help":
		fmt.Print(updateUsage)
		return 0
	default:
		fmt.Fprintf(os.Stderr, "usage: openthaiid update check|download|apply\n")
		return 1
	}
}

const updateUsage = `openthaiid update — look up GitHub Releases (no auto-install)

Usage:
  openthaiid update check [--stable]
  openthaiid update download [--stable] [--dir path]
  openthaiid update apply [--stable] [--yes]

check asks GitHub whether a newer unsigned package exists and prints the
release URL. It does not download. Offline / rate-limited checks fail soft
(Fill still works). download/apply stay explicit technician steps.

Packages are unsigned. No card data is sent.
Repo: night-sornram/openthaiid (override with OPENTHAIID_UPDATE_REPO=owner/name)
Future Store/code-signing steps: docs/signing.md
`

func updateOptionsFromArgs(args []string) (update.Options, string, bool, error) {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	stable := fs.Bool("stable", false, "ignore GitHub prereleases")
	dir := fs.String("dir", "", "download directory")
	yes := fs.Bool("yes", false, "replace the running helper (apply only)")
	if err := fs.Parse(args); err != nil {
		return update.Options{}, "", false, err
	}
	return update.Options{StableOnly: *stable}.WithEnv(), *dir, *yes, nil
}

func updateCheck(args []string) int {
	opts, _, _, err := updateOptionsFromArgs(args)
	if err != nil {
		return 1
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	res, err := update.Check(ctx, version, opts)
	if err != nil {
		res = update.Unavailable(version, update.ReasonOffline, update.ReleasesPage)
	}
	fmt.Println(res.StaffMessage(string(desktop.StaffLang())))
	return 0
}

func updateDownload(args []string, apply bool) int {
	opts, dir, yes, err := updateOptionsFromArgs(args)
	if err != nil {
		return 1
	}
	if apply && !yes {
		fmt.Fprintln(os.Stderr, "Refusing to replace the helper without --yes (packages are unsigned).")
		fmt.Fprintln(os.Stderr, "Download only: openthaiid update download")
		fmt.Fprintln(os.Stderr, "Or open", update.ReleasesPage)
		return 2
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	res, err := update.Check(ctx, version, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "update: %v\n", err)
		return 1
	}
	fmt.Println(res.Message)
	if res.AssetURL == "" {
		return 1
	}
	if dir == "" {
		userDir, err := config.UserDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "update: %v\n", err)
			return 1
		}
		dir = filepath.Join(userDir, "updates")
	}
	path, err := update.Download(ctx, dir, res.AssetURL, res.AssetName, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "download: %v\n", err)
		return 1
	}
	fmt.Println("Downloaded", path)
	if !apply {
		fmt.Println("Verify SHA256SUMS inside the archive, then: openthaiid update apply --yes")
		return 0
	}
	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "apply: %v\n", err)
		return 1
	}
	applied, err := update.ApplyArchive(path, exe)
	if err != nil {
		fmt.Fprintf(os.Stderr, "apply: %v\n", err)
		return 1
	}
	fmt.Println("Replaced", applied)
	if applied != exe {
		fmt.Println("Restart the helper to run the new file (Windows writes a .new next to the running exe).")
	} else {
		fmt.Println("Restart the helper to run the new binary.")
	}
	return 0
}
