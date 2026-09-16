package bridge

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/night-sornram/openthaiid/internal/update"
)

func TestManagerUpdateCheckReportsNewer(t *testing.T) {
	tb := startTestBridge(t, false)
	tb.srv.SetRuntime("0.1.0-rc.3", true)
	var calls atomic.Int32
	tb.srv.SetUpdateCheck(func(ctx context.Context, current string) (*update.CheckResult, error) {
		calls.Add(1)
		if current != "0.1.0-rc.3" {
			t.Errorf("current %s", current)
		}
		return &update.CheckResult{
			Current:      current,
			Latest:       "0.1.0-rc.4",
			Newer:        true,
			Prerelease:   true,
			Tag:          "v0.1.0-rc.4",
			HTMLURL:      "https://github.com/night-sornram/openthaiid/releases/tag/v0.1.0-rc.4",
			ReleasesPage: update.ReleasesPage,
			AssetURL:     "https://example.test/secret-asset",
			State:        update.StateNewer,
			Message:      "A newer prerelease is available.",
			MessageTH:    "มีเวอร์ชันใหม่ (รุ่นทดลอง)",
		}, nil
	})
	token := csrfFrom(t, getBody(t, tb, "/"))

	st := managerGET(t, tb, token, "/v1/manager/status")
	if st["version"] != "0.1.0-rc.3" {
		t.Fatalf("status version %v", st["version"])
	}
	if calls.Load() != 0 {
		t.Fatal("status/load must not query GitHub")
	}
	_, _ = http.Get(tb.httpBase() + "/v1/helper/summary")
	if calls.Load() != 0 {
		t.Fatal("helper summary must not query GitHub")
	}

	out := managerPOST(t, tb, token, "/v1/manager/update/check", nil)
	if calls.Load() != 1 {
		t.Fatalf("calls %d", calls.Load())
	}
	if out["state"] != update.StateNewer || out["newer"] != true {
		t.Fatalf("%v", out)
	}
	if out["htmlUrl"] != "https://github.com/night-sornram/openthaiid/releases/tag/v0.1.0-rc.4" {
		t.Fatalf("url %v", out["htmlUrl"])
	}
	raw, _ := json.Marshal(out)
	if strings.Contains(string(raw), "secret-asset") {
		t.Fatalf("asset URL leaked: %s", raw)
	}
	if strings.Contains(string(raw), "1234567890121") {
		t.Fatal("pii")
	}
	msgTH, _ := out["messageTh"].(string)
	if !strings.Contains(msgTH, "มีเวอร์ชันใหม่") {
		t.Fatalf("thai %q", msgTH)
	}
}

func TestManagerUpdateCheckSoftFail(t *testing.T) {
	tb := startTestBridge(t, false)
	tb.srv.SetRuntime("0.1.0-rc.4", true)
	tb.srv.SetUpdateCheck(func(ctx context.Context, current string) (*update.CheckResult, error) {
		return update.Unavailable(current, update.ReasonOffline, update.ReleasesPage), nil
	})
	token := csrfFrom(t, getBody(t, tb, "/"))
	out := managerPOST(t, tb, token, "/v1/manager/update/check", nil)
	if out["state"] != update.StateUnavailable {
		t.Fatalf("%v", out)
	}
	msgTH, _ := out["messageTh"].(string)
	if !strings.Contains(msgTH, "กรอกข้อมูลได้ตามปกติ") {
		t.Fatalf("%q", msgTH)
	}
}

func TestManagerUpdateCheckInFlightDoesNotBlock(t *testing.T) {
	tb := startTestBridge(t, false)
	tb.srv.SetRuntime("0.1.0-rc.4", true)
	started := make(chan struct{})
	release := make(chan struct{})
	var n atomic.Int32
	tb.srv.SetUpdateCheck(func(ctx context.Context, current string) (*update.CheckResult, error) {
		n.Add(1)
		close(started)
		<-release
		return &update.CheckResult{Current: current, State: update.StateUpToDate, Message: "up to date", MessageTH: "ใช้เวอร์ชันล่าสุดแล้ว"}, nil
	})
	token := csrfFrom(t, getBody(t, tb, "/"))

	done := make(chan map[string]any, 1)
	go func() {
		done <- managerPOST(t, tb, token, "/v1/manager/update/check", nil)
	}()
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("first check did not start")
	}
	second := managerPOST(t, tb, token, "/v1/manager/update/check", nil)
	if second["state"] != update.StateUnavailable {
		t.Fatalf("in-flight should fail soft, got %v", second)
	}
	close(release)
	first := <-done
	if first["state"] != update.StateUpToDate {
		t.Fatalf("first %v", first)
	}
	if n.Load() != 1 {
		t.Fatalf("checker calls %d", n.Load())
	}
}

func TestManagerUpdateCheckRejectsForeignOrigin(t *testing.T) {
	tb := startTestBridge(t, false)
	var called atomic.Bool
	tb.srv.SetUpdateCheck(func(ctx context.Context, current string) (*update.CheckResult, error) {
		called.Store(true)
		return update.Unavailable(current, update.ReasonOffline, update.ReleasesPage), nil
	})
	token := csrfFrom(t, getBody(t, tb, "/"))
	resp := loopbackDo(t, tb, http.MethodPost, "/v1/manager/update/check", token, map[string]string{
		"Origin": "https://evil.example",
	}, nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("status %d body=%s", resp.StatusCode, b)
	}
	if called.Load() {
		t.Fatal("foreign origin must not query GitHub")
	}
}
