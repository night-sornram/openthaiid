package desktop

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestLinuxUserUnitServeNoOpen(t *testing.T) {
	bin := `/opt/OpenThaiID/openthaiid`
	u := linuxUserUnit(bin)
	execLine := ""
	for _, line := range strings.Split(u, "\n") {
		if strings.HasPrefix(line, "ExecStart=") {
			execLine = line
			break
		}
	}
	if execLine != "ExecStart=/opt/OpenThaiID/openthaiid serve" {
		t.Fatalf("ExecStart %q", execLine)
	}
	if strings.Contains(execLine, "-open") || strings.Contains(execLine, "tray") {
		t.Fatal("always-on unit must use serve, not tray -open")
	}
	if !strings.Contains(u, "Restart=on-failure") {
		t.Fatal(u)
	}
	if !strings.Contains(u, "loginctl enable-linger") {
		t.Fatal("missing linger hint")
	}
	assertNoPII(t, u)
}

func TestLinuxSystemUnitTemplate(t *testing.T) {
	u := LinuxSystemUnit("/usr/local/bin/openthaiid")
	if !strings.Contains(u, "ExecStart=/usr/local/bin/openthaiid serve") {
		t.Fatal(u)
	}
	if strings.Contains(u, "tray") {
		t.Fatal(u)
	}
	assertNoPII(t, u)
}

func TestDarwinLaunchDaemonKeepAlive(t *testing.T) {
	p := darwinLaunchDaemon(`/Applications/OpenThaiID.app/Contents/MacOS/openthaiid`)
	if !strings.Contains(p, "<string>serve</string>") {
		t.Fatal(p)
	}
	if !strings.Contains(p, "<key>KeepAlive</key>\n  <true/>") {
		t.Fatal("LaunchDaemon KeepAlive must be true for always-on")
	}
	assertNoPII(t, p)
}

func TestWindowsTaskXMLServe(t *testing.T) {
	xml := windowsTaskXML(`C:\Program Files\OpenThaiID\openthaiid.exe`)
	if !strings.Contains(xml, `openthaiid.exe</Command>`) {
		t.Fatal(xml)
	}
	if !strings.Contains(xml, "<Arguments>serve</Arguments>") {
		t.Fatal(xml)
	}
	if strings.Contains(xml, "tray") {
		t.Fatal(xml)
	}
	assertNoPII(t, xml)
}

func TestServiceInstallWritesUnit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows install talks to SCM")
	}
	dir := t.TempDir()
	fake := filepath.Join(dir, "openthaiid")
	if err := os.WriteFile(fake, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	svc := Service{Executable: fake, Dir: filepath.Join(dir, "units")}
	on, path, err := svc.Installed()
	if err != nil || on {
		t.Fatalf("before: %v %v %s", on, err, path)
	}
	if err := svc.Install(); err != nil {
		t.Fatal(err)
	}
	on, path, err = svc.Installed()
	if err != nil || !on {
		t.Fatalf("after: %v %v %s", on, err, path)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if !strings.Contains(text, fake) || !strings.Contains(text, "serve") {
		t.Fatalf("contents:\n%s", text)
	}
	if strings.Contains(text, "-open") {
		t.Fatal("always-on must not open a browser")
	}
	assertNoPII(t, text)
	if err := svc.Uninstall(); err != nil {
		t.Fatal(err)
	}
	on, _, err = svc.Installed()
	if err != nil || on {
		t.Fatalf("after uninstall: %v %v", on, err)
	}
}

func TestServeArgs(t *testing.T) {
	if got := strings.Join(ServeArgs(), " "); got != "serve" {
		t.Fatalf("%q", got)
	}
}
