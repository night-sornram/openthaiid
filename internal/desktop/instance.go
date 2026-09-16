package desktop

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const pidFileName = "helper.pid"

// Instance tracks a running helper via a pid file under the user dir.
type Instance struct {
	Dir string
}

func (i Instance) PIDPath() string {
	dir := i.Dir
	if dir == "" {
		var err error
		dir, err = defaultUserDir()
		if err != nil {
			return ""
		}
	}
	return filepath.Join(dir, pidFileName)
}

func defaultUserDir() (string, error) {
	if x := os.Getenv("OPENTHAIID_HOME"); x != "" {
		return x, nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "openthaiid"), nil
}

// WritePID records this process after a successful listen.
func (i Instance) WritePID() error {
	path := i.PIDPath()
	if path == "" {
		return fmt.Errorf("pid path unavailable")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(strconv.Itoa(os.Getpid())+"\n"), 0o600)
}

// RemovePID deletes the pid file. Safe if missing.
func (i Instance) RemovePID() {
	_ = os.Remove(i.PIDPath())
}

// PID returns the recorded pid if the file exists.
func (i Instance) PID() (int, error) {
	raw, err := os.ReadFile(i.PIDPath())
	if err != nil {
		return 0, err
	}
	n, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	if err != nil || n < 1 {
		return 0, fmt.Errorf("invalid pid file")
	}
	return n, nil
}

// LogPath is the tray-mode helper log (operational messages only).
func (i Instance) LogPath() string {
	dir := i.Dir
	if dir == "" {
		var err error
		dir, err = defaultUserDir()
		if err != nil {
			return ""
		}
	}
	return filepath.Join(dir, "helper.log")
}

// OpenLog appends to helper.log, rotating once past 1 MiB. Callers must Close.
func (i Instance) OpenLog() (*os.File, error) {
	path := i.LogPath()
	if path == "" {
		return nil, fmt.Errorf("log path unavailable")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	if st, err := os.Stat(path); err == nil && st.Size() > 1<<20 {
		_ = os.Rename(path, path+".old")
	}
	return os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
}
