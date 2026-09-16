package desktop

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	autostartLabel     = "OpenThaiID"
	linuxDesktopName   = "openthaiid.desktop"
	darwinPlistName    = "org.openthaiid.helper.plist"
	windowsStartupName = "OpenThaiID.cmd"
)

// Autostart writes an opt-in login item for the local helper.
// Defaults stay off: Enable must be called explicitly.
type Autostart struct {
	// Executable is the helper binary. Empty uses os.Executable().
	Executable string
	// Override directories in tests. Empty uses the current user dirs.
	Dir string
}

// CommandLine is the argv used at login: helper tray (no browser popup).
func CommandLine(executable string) (bin string, args []string, err error) {
	bin = strings.TrimSpace(executable)
	if bin == "" {
		bin, err = os.Executable()
		if err != nil {
			return "", nil, err
		}
	}
	abs, err := filepath.Abs(bin)
	if err != nil {
		return "", nil, err
	}
	return abs, []string{"tray"}, nil
}

func (a Autostart) executable() (string, error) {
	bin, _, err := CommandLine(a.Executable)
	return bin, err
}

// File is the login-item path for this OS (cmd / desktop / plist).
func (a Autostart) File() (string, error) {
	if a.Dir != "" {
		return filepath.Join(a.Dir, autostartFileName()), nil
	}
	dir, err := defaultAutostartDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, autostartFileName()), nil
}

func autostartFileName() string {
	switch runtime.GOOS {
	case "windows":
		return windowsStartupName
	case "darwin":
		return darwinPlistName
	default:
		return linuxDesktopName
	}
}

func defaultAutostartDir() (string, error) {
	switch runtime.GOOS {
	case "windows":
		appdata := os.Getenv("APPDATA")
		if appdata == "" {
			return "", fmt.Errorf("APPDATA is not set")
		}
		return filepath.Join(appdata, "Microsoft", "Windows", "Start Menu", "Programs", "Startup"), nil
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, "Library", "LaunchAgents"), nil
	default:
		if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
			return filepath.Join(x, "autostart"), nil
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".config", "autostart"), nil
	}
}

// Enable writes the login item. It does not start a second helper now.
func (a Autostart) Enable() error {
	bin, err := a.executable()
	if err != nil {
		return err
	}
	path, err := a.File()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	body, err := autostartContents(bin)
	if err != nil {
		return err
	}
	return os.WriteFile(path, []byte(body), 0o644)
}

// Disable removes the login item if it exists.
func (a Autostart) Disable() error {
	path, err := a.File()
	if err != nil {
		return err
	}
	err = os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// Enabled reports whether a login item file is present.
func (a Autostart) Enabled() (bool, string, error) {
	path, err := a.File()
	if err != nil {
		return false, "", err
	}
	_, err = os.Stat(path)
	if err == nil {
		return true, path, nil
	}
	if os.IsNotExist(err) {
		return false, path, nil
	}
	return false, path, err
}

func autostartContents(bin string) (string, error) {
	switch runtime.GOOS {
	case "windows":
		return windowsStartupCmd(bin), nil
	case "darwin":
		return darwinLaunchAgent(bin), nil
	default:
		return linuxDesktop(bin), nil
	}
}

func windowsStartupCmd(bin string) string {
	return "@echo off\r\nstart \"\" \"" + bin + "\" tray\r\n"
}

func linuxDesktop(bin string) string {
	return `[Desktop Entry]
Type=Application
Name=OpenThaiID
Comment=Local Thai National ID helper (loopback only)
Exec="` + escapeDesktop(bin) + `" tray
Terminal=false
Categories=Utility;
X-GNOME-Autostart-enabled=true
`
}

func darwinLaunchAgent(bin string) string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>org.openthaiid.helper</string>
  <key>ProgramArguments</key>
  <array>
    <string>` + escapeXML(bin) + `</string>
    <string>tray</string>
  </array>
  <key>RunAtLoad</key>
  <true/>
  <key>KeepAlive</key>
  <false/>
</dict>
</plist>
`
}

func escapeDesktop(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}

func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	return s
}
