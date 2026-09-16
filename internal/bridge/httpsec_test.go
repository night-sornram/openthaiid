package bridge

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/night-sornram/openthaiid/internal/audit"
	"github.com/night-sornram/openthaiid/internal/pcsc"
	"github.com/night-sornram/openthaiid/internal/qualify"
)

func TestLoopbackMutationsRejectForeignOrigin(t *testing.T) {
	tb := startTestBridge(t, false)
	token := csrfFrom(t, getBody(t, tb, "/"))
	tb.mon.AddReader("r1", "Reader One")
	tb.mon.AddReader("r2", "Reader Two")
	_ = tb.mgr.Refresh(t.Context())

	cases := []struct {
		method string
		path   string
		body   []byte
	}{
		{http.MethodGet, "/v1/manager/status", nil},
		{http.MethodGet, "/v1/manager/audit", nil},
		{http.MethodPost, "/v1/manager/refresh", nil},
		{http.MethodPost, "/v1/manager/reconnect", nil},
		{http.MethodPost, "/v1/manager/reader/select", []byte(`{"readerId":"r1"}`)},
		{http.MethodPost, "/v1/manager/trust/revoke-all", nil},
		{http.MethodPost, "/v1/manager/log-level", []byte(`{"level":"info"}`)},
		{http.MethodPost, "/v1/helper/shutdown", nil},
		{http.MethodGet, "/v1/helper/summary", nil},
		{http.MethodPost, "/v1/manager/update/check", nil},
	}
	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			var called atomic.Bool
			tb.srv.SetShutdownRequest(func() { called.Store(true) })
			resp := loopbackDo(t, tb, tc.method, tc.path, token, map[string]string{
				"Origin": "https://evil.example",
			}, tc.body)
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusForbidden {
				b, _ := io.ReadAll(resp.Body)
				t.Fatalf("status %d body=%s", resp.StatusCode, b)
			}
			if aco := resp.Header.Get("Access-Control-Allow-Origin"); aco != "" {
				t.Fatalf("CORS %q", aco)
			}
			if called.Load() {
				t.Fatal("shutdown must not run for a foreign Origin")
			}
		})
	}
}

func TestLoopbackMutationsRejectCrossSiteFetch(t *testing.T) {
	tb := startTestBridge(t, false)
	token := csrfFrom(t, getBody(t, tb, "/"))
	tb.mon.AddReader("r1", "Reader One")
	_ = tb.mgr.Refresh(t.Context())

	cases := []struct {
		method string
		path   string
		body   []byte
	}{
		{http.MethodPost, "/v1/manager/reconnect", nil},
		{http.MethodPost, "/v1/manager/refresh", nil},
		{http.MethodPost, "/v1/manager/reader/select", []byte(`{"readerId":"r1"}`)},
		{http.MethodGet, "/v1/manager/audit", nil},
		{http.MethodPost, "/v1/helper/shutdown", nil},
		{http.MethodGet, "/v1/helper/summary", nil},
		{http.MethodPost, "/v1/manager/update/check", nil},
	}
	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			var called atomic.Bool
			tb.srv.SetShutdownRequest(func() { called.Store(true) })
			resp := loopbackDo(t, tb, tc.method, tc.path, token, map[string]string{
				"Origin":         tb.httpBase(),
				"Sec-Fetch-Site": "cross-site",
			}, tc.body)
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusForbidden {
				b, _ := io.ReadAll(resp.Body)
				t.Fatalf("status %d body=%s", resp.StatusCode, b)
			}
			if called.Load() {
				t.Fatal("cross-site fetch must not stop the helper")
			}
		})
	}
}

func TestManagerPOSTRejectsMissingOrigin(t *testing.T) {
	tb := startTestBridge(t, false)
	token := csrfFrom(t, getBody(t, tb, "/"))
	tb.mon.AddReader("r1", "Reader One")
	_ = tb.mgr.Refresh(t.Context())

	for _, path := range []string{
		"/v1/manager/refresh",
		"/v1/manager/reconnect",
		"/v1/manager/reader/select",
		"/v1/manager/trust/revoke-all",
		"/v1/manager/update/check",
	} {
		resp := loopbackDo(t, tb, http.MethodPost, path, token, nil, nil)
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("%s status %d", path, resp.StatusCode)
		}
	}
}

func TestHelperShutdownRequiresCSRFWhenOriginPresent(t *testing.T) {
	tb := startTestBridge(t, false)
	var called atomic.Bool
	tb.srv.SetShutdownRequest(func() { called.Store(true) })

	resp := loopbackDo(t, tb, http.MethodPost, "/v1/helper/shutdown", "", map[string]string{
		"Origin": tb.httpBase(),
	}, nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status %d", resp.StatusCode)
	}
	time.Sleep(50 * time.Millisecond)
	if called.Load() {
		t.Fatal("shutdown without CSRF must not run")
	}

	token := csrfFrom(t, getBody(t, tb, "/"))
	resp2 := loopbackDo(t, tb, http.MethodPost, "/v1/helper/shutdown", token, map[string]string{
		"Origin": tb.httpBase(),
	}, nil)
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp2.Body)
		t.Fatalf("manager origin + CSRF status %d body=%s", resp2.StatusCode, b)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if called.Load() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("shutdown callback not invoked")
}

func TestHelperSummaryAllowsCLIAndManagerOrigin(t *testing.T) {
	tb := startTestBridgeSeed(t, false, func(mon, rdb *pcsc.SyntheticBackend) {
		rdb.SeedDemoCard()
	})
	waitReader(t, tb, "synthetic-acr39u")

	resp, err := http.Get(tb.httpBase() + "/v1/helper/summary")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("cli summary %d", resp.StatusCode)
	}
	raw, _ := io.ReadAll(resp.Body)
	assertNoForbiddenJSON(t, raw)

	token := csrfFrom(t, getBody(t, tb, "/"))
	resp2 := loopbackDo(t, tb, http.MethodGet, "/v1/helper/summary", token, map[string]string{
		"Origin": tb.httpBase(),
	}, nil)
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("manager origin summary %d", resp2.StatusCode)
	}
}

func TestManagerDeviceActionsRateLimited(t *testing.T) {
	tb := startTestBridge(t, false)
	token := csrfFrom(t, getBody(t, tb, "/"))

	var n429, nOK int
	for i := 0; i < managerDeviceBurst+2; i++ {
		resp := loopbackDo(t, tb, http.MethodPost, "/v1/manager/reconnect", token, map[string]string{
			"Origin": tb.httpBase(),
		}, nil)
		raw, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		switch resp.StatusCode {
		case http.StatusOK:
			nOK++
		case http.StatusTooManyRequests:
			n429++
			var out map[string]any
			if err := json.Unmarshal(raw, &out); err != nil {
				t.Fatalf("429 json: %v", err)
			}
			if out["code"] != "RATE_LIMITED" {
				t.Fatalf("429 code=%v", out["code"])
			}
		default:
			t.Fatalf("status %d body=%s", resp.StatusCode, raw)
		}
	}
	if nOK != managerDeviceBurst || n429 == 0 {
		t.Fatalf("burst=%d ok=%d 429=%d", managerDeviceBurst, nOK, n429)
	}

	resp := loopbackDo(t, tb, http.MethodPost, "/v1/manager/refresh", token, map[string]string{
		"Origin": tb.httpBase(),
	}, nil)
	raw, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("refresh should share the device bound, status %d body=%s", resp.StatusCode, raw)
	}
}

func TestLoopbackJSONOmitsPII(t *testing.T) {
	tb := startTestBridgeSeed(t, false, func(mon, rdb *pcsc.SyntheticBackend) {
		rdb.SeedDemoCard()
	})
	tb.srv.SetRuntime("0.0.1-test", true)
	waitReader(t, tb, "synthetic-acr39u")
	token := csrfFrom(t, getBody(t, tb, "/"))

	tb.srv.recordError("websocket", "NO_CARD", "no card สมชาย 1234567890121 Somchai")
	tb.srv.recordError("reconnect", "NO_CARD สมชาย", "should drop whole event")
	tb.srv.recordAudit(audit.Event{
		Kind:   audit.KindFillFailure,
		Code:   "NO_CARD สมชาย 1234567890121",
		Origin: "https://evil.example",
	})
	tb.srv.recordAudit(audit.Event{
		Kind:   audit.KindFillFailure,
		Code:   "NO_CARD",
		Origin: "https://erp.clinic.example",
	})

	paths := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/v1/helper/summary"},
		{http.MethodGet, "/v1/manager/status"},
		{http.MethodGet, "/v1/manager/audit"},
		{http.MethodPost, "/v1/manager/reconnect"},
	}
	for _, tc := range paths {
		resp := loopbackDo(t, tb, tc.method, tc.path, token, map[string]string{
			"Origin": tb.httpBase(),
		}, nil)
		raw, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%s %s status %d body=%s", tc.method, tc.path, resp.StatusCode, raw)
		}
		assertNoForbiddenJSON(t, raw)
		if tc.path == "/v1/helper/summary" {
			var s map[string]any
			if err := json.Unmarshal(raw, &s); err != nil {
				t.Fatal(err)
			}
			for _, leak := range []string{"pending", "trusted", "audit", "lastErrors", "citizenId", "fields"} {
				if _, ok := s[leak]; ok {
					t.Fatalf("summary must not include %s: %s", leak, raw)
				}
			}
		}
	}

	st := managerGET(t, tb, token, "/v1/manager/status")
	errs, _ := st["lastErrors"].([]any)
	if len(errs) != 1 {
		t.Fatalf("poisoned lastErrors should drop bad code and strip message: %v", st["lastErrors"])
	}
	e0 := errs[0].(map[string]any)
	if e0["code"] != "NO_CARD" {
		t.Fatalf("last error=%v", e0)
	}
	if msg, _ := e0["message"].(string); msg != "" {
		t.Fatalf("poisoned last-error message must be stripped: %q", msg)
	}
}

func TestManagerCORSPreflightHasNoWildcard(t *testing.T) {
	tb := startTestBridge(t, false)
	req, err := http.NewRequest(http.MethodOptions, tb.httpBase()+"/v1/manager/reconnect", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Origin", "https://evil.example")
	req.Header.Set("Access-Control-Request-Method", "POST")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if aco := resp.Header.Get("Access-Control-Allow-Origin"); aco != "" {
		t.Fatalf("preflight CORS %q", aco)
	}
	if resp.StatusCode == http.StatusOK {
		t.Fatalf("OPTIONS must not succeed for manager mutations, got %d", resp.StatusCode)
	}
}

func loopbackDo(t *testing.T, tb *testBridge, method, path, token string, extra map[string]string, body []byte) *http.Response {
	t.Helper()
	var rdr io.Reader
	if body != nil {
		rdr = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, tb.httpBase()+path, rdr)
	if err != nil {
		t.Fatal(err)
	}
	if token != "" {
		req.Header.Set(managerTokenHeader, token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range extra {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func assertNoForbiddenJSON(t *testing.T, raw []byte) {
	t.Helper()
	if needle := qualify.FirstForbidden(raw); needle != "" {
		t.Fatalf("json leaked %q: %s", needle, raw)
	}
}
