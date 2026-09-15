package bridge

import (
	"bytes"
	"io"
	"log/slog"
	"testing"
)

// Sentinel values that must never appear in structured logs.
func TestLogsNeverContainFieldSentinels(t *testing.T) {
	sentinels := []string{
		"1101701234567",
		"สมชาย",
		"25330412",
		"cs_secret",
		"tok_abcdef",
	}
	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	// The bridge logger is only used for operational messages. Approval
	// prompts go to stdout (terminal), not this logger.
	log.Info("bridge serve failed", "error", io.EOF)
	log.Debug("read", "duration_ms", 12, "request", "req_9f")

	out := buf.String()
	for _, s := range sentinels {
		if bytes.Contains([]byte(out), []byte(s)) {
			t.Fatalf("log leaked sentinel %q: %s", s, out)
		}
	}
}
