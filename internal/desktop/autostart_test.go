package desktop

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestAutostartEnableDisable(t *testing.T) {
	dir := t.TempDir()
	fake := filepath.Join(dir, "openthaiid")
	if runtime.GOOS == "windows" {
		fake += ".exe"
	}
	if err := os.WriteFile(fake, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	a := Autostart{Executable: fake, Dir: filepath.Join(dir, "login")}
	on, path, err := a.Enabled()
	if err != nil || on {
		t.Fatalf("enabled before: %v %v %s", on, err, path)
	}
	if err := a.Enable(); err != nil {
		t.Fatal(err)
	}
	on, path, err = a.Enabled()
	if err != nil || !on {
		t.Fatalf("enabled after: %v %v %s", on, err, path)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if !strings.Contains(text, fake) || !strings.Contains(text, "tray") {
		t.Fatalf("contents:\n%s", text)
	}
	if strings.Contains(text, "-open") {
		t.Fatal("login item must not auto-open a browser")
	}
	assertNoPII(t, text)
	if err := a.Disable(); err != nil {
		t.Fatal(err)
	}
	on, _, err = a.Enabled()
	if err != nil || on {
		t.Fatalf("enabled after disable: %v %v", on, err)
	}
	if err := a.Disable(); err != nil {
		t.Fatal(err)
	}
}

func TestAutostartContentsNoOpen(t *testing.T) {
	bin := `/opt/OpenThaiID/openthaiid`
	if runtime.GOOS == "windows" {
		if !strings.Contains(windowsStartupCmd(bin), `start "" "`+bin+`" tray`) {
			t.Fatal(windowsStartupCmd(bin))
		}
	}
	d := linuxDesktop(bin)
	if !strings.Contains(d, `Exec="`+bin+`" tray`) || strings.Contains(d, "-open") {
		t.Fatal(d)
	}
	p := darwinLaunchAgent(bin)
	if !strings.Contains(p, "<string>"+bin+"</string>") || !strings.Contains(p, "<string>tray</string>") {
		t.Fatal(p)
	}
	if strings.Contains(p, "<true/>") && strings.Contains(p, "KeepAlive") {
		if strings.Contains(p, "<key>KeepAlive</key>\n  <true/>") {
			t.Fatal("KeepAlive must stay false so tray Quit is not overridden")
		}
	}
}

func TestEscapeXML(t *testing.T) {
	got := darwinLaunchAgent(`/tmp/a&b<c>`)
	if !strings.Contains(got, `/tmp/a&amp;b&lt;c&gt;`) {
		t.Fatal(got)
	}
}
