package desktop

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestInstancePIDAndLog(t *testing.T) {
	dir := t.TempDir()
	inst := Instance{Dir: dir}
	if err := inst.WritePID(); err != nil {
		t.Fatal(err)
	}
	got, err := inst.PID()
	if err != nil {
		t.Fatal(err)
	}
	if got != os.Getpid() {
		t.Fatalf("pid %d want %d", got, os.Getpid())
	}
	raw, _ := os.ReadFile(inst.PIDPath())
	if strings.TrimSpace(string(raw)) != strconv.Itoa(os.Getpid()) {
		t.Fatalf("pid file %q", raw)
	}
	f, err := inst.OpenLog()
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.WriteString("bridge serve failed duration_ms=3\n")
	_ = f.Close()
	body, _ := os.ReadFile(inst.LogPath())
	assertNoPII(t, string(body))
	inst.RemovePID()
	if _, err := os.Stat(inst.PIDPath()); !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if filepath.Base(inst.LogPath()) != "helper.log" {
		t.Fatal(inst.LogPath())
	}
}
