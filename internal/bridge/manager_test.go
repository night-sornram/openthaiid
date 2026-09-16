package bridge

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/night-sornram/openthaiid/internal/auth"
	"github.com/night-sornram/openthaiid/internal/card"
	"github.com/night-sornram/openthaiid/internal/pcsc"
)

var csrfMeta = regexp.MustCompile(`name="openthaiid-csrf"\s+content="([a-f0-9]+)"`)

func TestManagerPageAndStatus(t *testing.T) {
	tb := startTestBridgeSeed(t, false, func(mon, rdb *pcsc.SyntheticBackend) {
		rdb.SeedDemoCard()
	})
	tb.srv.SetRuntime("0.0.1-test", true)
	waitReader(t, tb, "synthetic-acr39u")

	page := getBody(t, tb, "/")
	if !strings.Contains(page, "OpenThaiID helper") {
		t.Fatal("missing helper title")
	}
	if !strings.Contains(page, "Technician / developer") {
		t.Fatal("missing technician heading")
	}
	if !strings.Contains(page, `id="technician"`) {
		t.Fatal("missing technician section")
	}
	if !strings.Contains(page, "OpenThaiID บนเครื่องนี้") {
		t.Fatal("missing Thai user-mode helper light")
	}
	if !strings.Contains(page, `id="lang-th"`) || !strings.Contains(page, `id="lang-en"`) {
		t.Fatal("missing language toggle")
	}
	if !strings.Contains(page, `lang="th"`) {
		t.Fatal("default html lang should be Thai")
	}
	css := getBody(t, tb, "/manager/style.css")
	if !strings.Contains(css, ".tech[hidden]") {
		t.Fatal("technician section must stay display:none while hidden")
	}
	if strings.Contains(page, "1234567890121") || strings.Contains(page, "สมชาย") {
		t.Fatal("manager HTML must not embed card field values")
	}
	token := csrfFrom(t, page)

	st := managerGET(t, tb, token, "/v1/manager/status")
	if st["running"] != true {
		t.Fatalf("running=%v", st["running"])
	}
	if st["synthetic"] != true {
		t.Fatalf("synthetic=%v", st["synthetic"])
	}
	if st["version"] != "0.0.1-test" {
		t.Fatalf("version=%v", st["version"])
	}
	raw, _ := json.Marshal(st)
	if bytes.Contains(raw, []byte("1234567890121")) || bytes.Contains(raw, []byte("สมชาย")) {
		t.Fatalf("status leaked personal fields: %s", raw)
	}
	readers, _ := st["readers"].([]any)
	if len(readers) != 1 {
		t.Fatalf("readers=%v", st["readers"])
	}
	r0 := readers[0].(map[string]any)
	if r0["presence"] != "present" {
		t.Fatalf("presence=%v", r0["presence"])
	}
	if r0["name"] != "SYNTHETIC ACR39U (fictional demo)" {
		t.Fatalf("name=%v", r0["name"])
	}
	backend, _ := st["backend"].(map[string]any)
	if backend["name"] != "synthetic" || backend["status"] != "available" {
		t.Fatalf("backend=%v", backend)
	}
	if st["os"] == nil || st["os"] == "" {
		t.Fatal("expected os in status")
	}
	if st["logLevel"] != "info" {
		t.Fatalf("default logLevel=%v", st["logLevel"])
	}
	if _, ok := st["lastErrors"].([]any); !ok {
		t.Fatalf("lastErrors missing: %v", st["lastErrors"])
	}
}

func TestManagerCSRFRejectsForeignOrigin(t *testing.T) {
	tb := startTestBridge(t, false)
	token := csrfFrom(t, getBody(t, tb, "/"))
	req, err := http.NewRequest(http.MethodGet, tb.httpBase()+"/v1/manager/status", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Origin", "https://evil.example")
	req.Header.Set(managerTokenHeader, token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status %d", resp.StatusCode)
	}
}

func TestManagerCSRFRejectsMissingToken(t *testing.T) {
	tb := startTestBridge(t, false)
	req, err := http.NewRequest(http.MethodGet, tb.httpBase()+"/v1/manager/status", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Origin", tb.httpBase())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status %d", resp.StatusCode)
	}
}

func TestManagerAllowDenyAndRevoke(t *testing.T) {
	tb := startTestBridge(t, false)
	token := csrfFrom(t, getBody(t, tb, "/"))
	origin := auth.MustOrigin("https://clinic.example.com")
	p, err := tb.authMgr.RequestApproval(origin, []card.Field{card.FieldCitizenID, card.FieldThaiName}, "conn-mgr", time.Now())
	if err != nil {
		t.Fatal(err)
	}

	st := managerGET(t, tb, token, "/v1/manager/status")
	pending, _ := st["pending"].([]any)
	if len(pending) != 1 {
		t.Fatalf("pending=%v", st["pending"])
	}

	_ = managerPOST(t, tb, token, "/v1/manager/approvals/"+itoa(p.ID)+"/allow", nil)
	if _, ok := tb.authMgr.GrantFor(origin, time.Now()); !ok {
		t.Fatal("expected grant after allow")
	}
	st = managerGET(t, tb, token, "/v1/manager/status")
	if pending, _ = st["pending"].([]any); len(pending) != 0 {
		t.Fatalf("pending after allow: %v", pending)
	}
	trusted, _ := st["trusted"].([]any)
	if len(trusted) != 1 {
		t.Fatalf("trusted=%v", trusted)
	}

	_ = managerPOST(t, tb, token, "/v1/manager/trust/revoke", map[string]string{"origin": "https://clinic.example.com"})
	if _, ok := tb.authMgr.GrantFor(origin, time.Now()); ok {
		t.Fatal("grant should be revoked")
	}

	p2, err := tb.authMgr.RequestApproval(origin, []card.Field{card.FieldCitizenID}, "conn-mgr-2", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	_ = managerPOST(t, tb, token, "/v1/manager/approvals/"+itoa(p2.ID)+"/deny", nil)
	if _, ok := tb.authMgr.GrantFor(origin, time.Now()); ok {
		t.Fatal("deny must not create a grant")
	}
}

func TestManagerRefreshAndSelectReader(t *testing.T) {
	tb := startTestBridge(t, false)
	token := csrfFrom(t, getBody(t, tb, "/"))
	tb.mon.AddReader("r1", "Reader One")
	tb.mon.AddReader("r2", "Reader Two")
	tb.mon.InsertCard("r2")
	_ = managerPOST(t, tb, token, "/v1/manager/refresh", nil)
	st := managerGET(t, tb, token, "/v1/manager/status")
	readers, _ := st["readers"].([]any)
	if len(readers) != 2 {
		t.Fatalf("readers after refresh=%v", st["readers"])
	}
	_ = managerPOST(t, tb, token, "/v1/manager/reader/select", map[string]string{"readerId": "r2"})
	st = managerGET(t, tb, token, "/v1/manager/status")
	found := false
	for _, raw := range st["readers"].([]any) {
		r := raw.(map[string]any)
		if r["readerId"] == "r2" && r["selected"] == true && r["presence"] == "present" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected r2 selected: %v", st["readers"])
	}
}

func TestManagerNoCORS(t *testing.T) {
	tb := startTestBridge(t, false)
	resp, err := http.Get(tb.httpBase() + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if aco := resp.Header.Get("Access-Control-Allow-Origin"); aco != "" {
		t.Fatalf("unexpected CORS %q", aco)
	}
}

func TestManagerHealthzUnchanged(t *testing.T) {
	tb := startTestBridge(t, false)
	resp, err := http.Get(tb.httpBase() + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !bytes.Contains(body, []byte(`"status":"ok"`)) && !bytes.Contains(body, []byte(`"status": "ok"`)) {
		t.Fatalf("healthz body %s", body)
	}
}

func TestManagerDiagnosticsLogLevelAndReconnect(t *testing.T) {
	tb := startTestBridgeSeed(t, false, func(mon, rdb *pcsc.SyntheticBackend) {
		rdb.SeedDemoCard()
	})
	tb.srv.SetRuntime("0.0.1-test", true)
	waitReader(t, tb, "synthetic-acr39u")
	token := csrfFrom(t, getBody(t, tb, "/"))

	tb.srv.recordError("websocket", "NO_CARD", "no card present")
	tb.srv.recordError("websocket", "APPROVAL_REQUIRED", "should be skipped")

	st := managerGET(t, tb, token, "/v1/manager/status")
	raw, _ := json.Marshal(st)
	if bytes.Contains(raw, []byte("1234567890121")) || bytes.Contains(raw, []byte("สมชาย")) {
		t.Fatalf("diagnostics leaked personal fields: %s", raw)
	}
	errs, _ := st["lastErrors"].([]any)
	if len(errs) != 1 {
		t.Fatalf("lastErrors=%v", st["lastErrors"])
	}
	e0 := errs[0].(map[string]any)
	if e0["code"] != "NO_CARD" || e0["source"] != "websocket" {
		t.Fatalf("last error=%v", e0)
	}

	out := managerPOST(t, tb, token, "/v1/manager/log-level", map[string]any{"verbose": true})
	if out["logLevel"] != "debug" {
		t.Fatalf("logLevel after verbose=%v", out["logLevel"])
	}
	st = managerGET(t, tb, token, "/v1/manager/status")
	if st["logLevel"] != "debug" {
		t.Fatalf("status logLevel=%v", st["logLevel"])
	}
	out = managerPOST(t, tb, token, "/v1/manager/log-level", map[string]any{"verbose": false})
	if out["logLevel"] != "info" {
		t.Fatalf("logLevel after quiet=%v", out["logLevel"])
	}

	out = managerPOST(t, tb, token, "/v1/manager/reconnect", nil)
	if out["ok"] != true {
		t.Fatalf("reconnect=%v", out)
	}
	st = managerGET(t, tb, token, "/v1/manager/status")
	readers, _ := st["readers"].([]any)
	if len(readers) != 1 {
		t.Fatalf("readers after reconnect=%v", st["readers"])
	}
}

func (tb *testBridge) httpBase() string {
	return "http://" + tb.srv.Addr()
}

func getBody(t *testing.T, tb *testBridge, path string) string {
	t.Helper()
	resp, err := http.Get(tb.httpBase() + path)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("%s status %d", path, resp.StatusCode)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func csrfFrom(t *testing.T, html string) string {
	t.Helper()
	m := csrfMeta.FindStringSubmatch(html)
	if len(m) != 2 {
		t.Fatalf("csrf meta missing")
	}
	return m[1]
}

func managerGET(t *testing.T, tb *testBridge, token, path string) map[string]any {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, tb.httpBase()+path, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set(managerTokenHeader, token)
	req.Header.Set("Origin", tb.httpBase())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("GET %s status %d body=%s", path, resp.StatusCode, b)
	}
	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	return out
}

func managerPOST(t *testing.T, tb *testBridge, token, path string, body any) map[string]any {
	t.Helper()
	var rdr io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		rdr = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(http.MethodPost, tb.httpBase()+path, rdr)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set(managerTokenHeader, token)
	req.Header.Set("Origin", tb.httpBase())
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("POST %s status %d body=%s", path, resp.StatusCode, b)
	}
	if len(b) == 0 {
		return nil
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func waitReader(t *testing.T, tb *testBridge, id string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		for _, r := range tb.mgr.Snapshot().Readers {
			if r.ID == id {
				return
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("reader %s not visible", id)
}

func itoa(n int64) string {
	return strconv.FormatInt(n, 10)
}
