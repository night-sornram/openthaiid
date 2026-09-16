package desktop

import (
	"context"
	"errors"
	"runtime"
	"testing"
)

func TestRunTrayStub(t *testing.T) {
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		t.Skip("native tray on this OS")
	}
	err := RunTray(context.Background(), Host{})
	if !errors.Is(err, ErrNoTray) {
		t.Fatalf("got %v", err)
	}
}
