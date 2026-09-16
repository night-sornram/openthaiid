package desktop

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	WindowsServiceName = "OpenThaiID"
	WindowsDisplayName = "OpenThaiID Helper"
	linuxUserUnitName  = "openthaiid.service"
	darwinDaemonName   = "org.openthaiid.helper.plist"
	windowsTaskName    = "OpenThaiID-Helper"
)

// Service is the always-on helper (no tray). Opt-in: Install must be called.
// Windows: SCM service. Linux: systemd --user unit. macOS: LaunchDaemon plist.
type Service struct {
	Executable string
	Dir        string
}

// ServeArgs is argv after the binary: headless helper, no tray, no browser.
func ServeArgs() []string { return []string{"serve"} }

func (s Service) executable() (string, error) {
	bin := strings.TrimSpace(s.Executable)
	if bin == "" {
		var err error
		bin, err = os.Executable()
		if err != nil {
			return "", err
		}
	}
	return filepath.Abs(bin)
}

// UnitFile is the unit / plist / (Windows) documented task path for this OS.
func (s Service) UnitFile() (string, error) {
	if s.Dir != "" {
		return filepath.Join(s.Dir, serviceFileName()), nil
	}
	dir, err := defaultServiceDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, serviceFileName()), nil
}

func serviceFileName() string {
	switch runtime.GOOS {
	case "windows":
		return windowsTaskName + ".xml"
	case "darwin":
		return darwinDaemonName
	default:
		return linuxUserUnitName
	}
}

func defaultServiceDir() (string, error) {
	switch runtime.GOOS {
	case "windows":
		appdata := os.Getenv("APPDATA")
		if appdata == "" {
			return "", fmt.Errorf("APPDATA is not set")
		}
		return filepath.Join(appdata, "OpenThaiID"), nil
	case "darwin":
		return "/Library/LaunchDaemons", nil
	default:
		if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
			return filepath.Join(x, "systemd", "user"), nil
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".config", "systemd", "user"), nil
	}
}

// Install writes the always-on unit (and on Windows, creates the SCM service).
func (s Service) Install() error {
	bin, err := s.executable()
	if err != nil {
		return err
	}
	if runtime.GOOS == "windows" {
		return installWindowsService(bin)
	}
	path, err := s.UnitFile()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	body, err := serviceContents(bin)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		return err
	}
	reloadUserSystemd()
	return nil
}

// Uninstall removes the always-on unit (and on Windows, deletes the SCM service).
func (s Service) Uninstall() error {
	if runtime.GOOS == "windows" {
		return uninstallWindowsService()
	}
	path, err := s.UnitFile()
	if err != nil {
		return err
	}
	_ = stopUserSystemd()
	_ = disableUserSystemd()
	err = os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	reloadUserSystemd()
	return nil
}

// Installed reports whether the unit file (or Windows service) exists.
func (s Service) Installed() (bool, string, error) {
	if runtime.GOOS == "windows" {
		return windowsServiceInstalled()
	}
	path, err := s.UnitFile()
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

func serviceContents(bin string) (string, error) {
	switch runtime.GOOS {
	case "windows":
		return windowsTaskXML(bin), nil
	case "darwin":
		return darwinLaunchDaemon(bin), nil
	default:
		return linuxUserUnit(bin), nil
	}
}

func linuxUserUnit(bin string) string {
	return `[Unit]
Description=OpenThaiID local helper (loopback only)
Documentation=https://github.com/night-sornram/openthaiid/blob/dev/docs/installation.md
After=pcscd.service

[Service]
Type=simple
ExecStart=` + systemdQuote(bin) + ` serve
Restart=on-failure
RestartSec=3

# Always-on helper without a tray. Listens only on 127.0.0.1.
# No card data is stored. Survive logout with: loginctl enable-linger $USER
# Then: systemctl --user enable --now openthaiid.service

[Install]
WantedBy=default.target
`
}

func darwinLaunchDaemon(bin string) string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>org.openthaiid.helper</string>
  <key>ProgramArguments</key>
  <array>
    <string>` + escapeXML(bin) + `</string>
    <string>serve</string>
  </array>
  <key>RunAtLoad</key>
  <true/>
  <key>KeepAlive</key>
  <true/>
</dict>
</plist>
`
}

func windowsTaskXML(bin string) string {
	return `<?xml version="1.0" encoding="UTF-16"?>
<Task version="1.2" xmlns="http://schemas.microsoft.com/windows/2004/02/mit/task">
  <RegistrationInfo>
    <Description>OpenThaiID local helper (loopback only). No card data is stored. Prefer: openthaiid service install</Description>
  </RegistrationInfo>
  <Triggers>
    <BootTrigger>
      <Enabled>true</Enabled>
    </BootTrigger>
  </Triggers>
  <Principals>
    <Principal id="Author">
      <LogonType>Password</LogonType>
      <RunLevel>LeastPrivilege</RunLevel>
    </Principal>
  </Principals>
  <Settings>
    <MultipleInstancesPolicy>IgnoreNew</MultipleInstancesPolicy>
    <DisallowStartIfOnBatteries>false</DisallowStartIfOnBatteries>
    <StopIfGoingOnBatteries>false</StopIfGoingOnBatteries>
    <AllowStartOnDemand>true</AllowStartOnDemand>
    <Enabled>true</Enabled>
    <Hidden>false</Hidden>
    <RunOnlyIfIdle>false</RunOnlyIfIdle>
    <WakeToRun>false</WakeToRun>
    <ExecutionTimeLimit>PT0S</ExecutionTimeLimit>
    <RestartOnFailure>
      <Interval>PT1M</Interval>
      <Count>3</Count>
    </RestartOnFailure>
  </Settings>
  <Actions Context="Author">
    <Exec>
      <Command>` + escapeXML(bin) + `</Command>
      <Arguments>serve</Arguments>
    </Exec>
  </Actions>
</Task>
`
}

func systemdQuote(path string) string {
	if strings.ContainsAny(path, " \t\"'\\") {
		return `"` + strings.ReplaceAll(path, `\`, `\\`) + `"`
	}
	return path
}

func reloadUserSystemd() {
	if runtime.GOOS != "linux" {
		return
	}
	_ = exec.Command("systemctl", "--user", "daemon-reload").Run()
}

func stopUserSystemd() error {
	if runtime.GOOS != "linux" {
		return nil
	}
	return exec.Command("systemctl", "--user", "stop", linuxUserUnitName).Run()
}

func disableUserSystemd() error {
	if runtime.GOOS != "linux" {
		return nil
	}
	return exec.Command("systemctl", "--user", "disable", linuxUserUnitName).Run()
}

// EnableUserUnit runs systemctl --user enable --now when systemd is present.
func EnableUserUnit() error {
	if runtime.GOOS != "linux" {
		return nil
	}
	if err := exec.Command("systemctl", "--user", "daemon-reload").Run(); err != nil {
		return err
	}
	return exec.Command("systemctl", "--user", "enable", "--now", linuxUserUnitName).Run()
}

// DisableUserUnit stops and disables the user unit.
func DisableUserUnit() error {
	if runtime.GOOS != "linux" {
		return nil
	}
	_ = exec.Command("systemctl", "--user", "disable", "--now", linuxUserUnitName).Run()
	return nil
}

// LinuxSystemUnit is a template clinic IT can drop in /etc/systemd/system/.
func LinuxSystemUnit(bin string) string {
	if bin == "" {
		bin = "/usr/local/bin/openthaiid"
	}
	return `[Unit]
Description=OpenThaiID local helper (loopback only)
Documentation=https://github.com/night-sornram/openthaiid/blob/dev/docs/installation.md
After=pcscd.service
Requires=pcscd.service

[Service]
Type=simple
ExecStart=` + systemdQuote(bin) + ` serve
Restart=on-failure
RestartSec=3
# Clinic IT: set User= to an account that can use the USB reader, or omit
# for the default service user. Loopback only; no card data is stored.

[Install]
WantedBy=multi-user.target
`
}
