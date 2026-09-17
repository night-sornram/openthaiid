package bridge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/night-sornram/openthaiid/internal/audit"
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
	if !strings.Contains(page, `id="audit-events"`) {
		t.Fatal("missing operator audit trail")
	}
	if !strings.Contains(page, "Operator audit trail") {
		t.Fatal("missing audit heading")
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
	if !strings.Contains(page, "ฝึกใช้ก่อนได้เครื่องจริง") {
		t.Fatal("missing Thai training heading")
	}
	if !strings.Contains(page, `id="training-card"`) {
		t.Fatal("missing training card")
	}
	if !strings.Contains(page, `id="test-connection"`) {
		t.Fatal("missing test-connection button")
	}
	if !strings.Contains(page, "ทดสอบการเชื่อมต่อ") {
		t.Fatal("missing Thai test-connection label")
	}
	if !strings.Contains(page, `id="check-update"`) {
		t.Fatal("missing check-update button")
	}
	if !strings.Contains(page, "ตรวจเวอร์ชัน") {
		t.Fatal("missing Thai check-update label")
	}
	if !strings.Contains(page, `id="staff-readers"`) {
		t.Fatal("missing staff multi-reader list")
	}
	if !strings.Contains(page, `class="light-state"`) {
		t.Fatal("lights must include text status, not color alone")
	}
	if !strings.Contains(page, `class="btn btn-staff"`) {
		t.Fatal("test-connection must be a large staff tap target")
	}
	if !strings.Contains(page, `aria-label="ภาษา / Language"`) {
		t.Fatal("language toggle must have a bilingual accessible name")
	}
	js := getBody(t, tb, "/manager/app.js")
	if !strings.Contains(js, "trainingTitle") || !strings.Contains(js, "Practice before the real reader arrives") {
		t.Fatal("missing training i18n")
	}
	if !strings.Contains(js, "refreshCsrfFromPage") || !strings.Contains(js, "testConnection") {
		t.Fatal("missing helper restart recovery or test-connection")
	}
	if !strings.Contains(js, "checkGitHubReleases") || !strings.Contains(js, "/v1/manager/update/check") {
		t.Fatal("missing operator GitHub Releases check")
	}
	if !strings.Contains(page, `id="desk-snapshot"`) {
		t.Fatal("missing technician desk snapshot button")
	}
	if !strings.Contains(js, "copyDeskSnapshot") || !strings.Contains(js, "/v1/helper/doctor") {
		t.Fatal("missing desk snapshot copy")
	}
	if !strings.Contains(js, "clearChildren") || !strings.Contains(js, "execCommand") {
		t.Fatal("Ready page needs replaceChildren/clipboard fallbacks for Firefox/Safari")
	}
	if strings.Contains(js, "api.github.com") {
		t.Fatal("Ready page must not call GitHub from the browser")
	}
	if !strings.Contains(js, "refreshTestResultCopy") {
		t.Fatal("missing test-result language refresh")
	}
	if !strings.Contains(js, "helperLostHeadline") || !strings.Contains(js, "bridge.probe") {
		t.Fatal("missing reconnect copy or WS probe test")
	}
	if !strings.Contains(js, "gentleWakeRefresh") || !strings.Contains(js, "pageshow") {
		t.Fatal("missing sleep/wake re-probe")
	}
	if !strings.Contains(js, "useThisReader") || !strings.Contains(js, "renderStaffReaders") {
		t.Fatal("missing staff multi-reader picker")
	}
	if !strings.Contains(js, "lightOk") || !strings.Contains(js, "lightStateWord") {
		t.Fatal("lights must expose text status words, not color alone")
	}
	if !strings.Contains(js, "btn-staff") {
		t.Fatal("ใช้เครื่องนี้ must be a large staff tap target")
	}
	if strings.Contains(js, "card.read") {
		t.Fatal("Ready-page test connection must not issue card.read")
	}
	if strings.Contains(js, "1234567890121") || strings.Contains(js, "สมชาย") {
		t.Fatal("manager JS must not embed card field values")
	}
	css := getBody(t, tb, "/manager/style.css")
	if !strings.Contains(css, ".tech[hidden]") {
		t.Fatal("technician section must stay display:none while hidden")
	}
	if !strings.Contains(css, ":focus-visible") {
		t.Fatal("missing focus-visible styles")
	}
	if !strings.Contains(css, "min-height: 2.75rem") {
		t.Fatal("staff tap targets should be at least 44px")
	}
	if !strings.Contains(css, "#update-result") || !strings.Contains(css, "pre-wrap") {
		t.Fatal("version-check result should keep line breaks")
	}
	if !strings.Contains(css, ".light-state") {
		t.Fatal("lights must style a text status, not color alone")
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
	assertNoForbiddenJSON(t, raw)
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
	if _, ok := st["audit"].([]any); !ok {
		t.Fatalf("audit missing: %v", st["audit"])
	}
	kinds := auditKinds(st)
	if !containsKind(kinds, "helper.start") {
		t.Fatalf("audit missing helper.start: %v", kinds)
	}
	if !containsKind(kinds, "reader.connected") || !containsKind(kinds, "card.present") {
		t.Fatalf("audit missing reader/card snapshot: %v", kinds)
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

	st = managerGET(t, tb, token, "/v1/manager/status")
	kinds := auditKinds(st)
	if !containsKind(kinds, "approval.allow") || !containsKind(kinds, "approval.deny") || !containsKind(kinds, "approval.revoke") {
		t.Fatalf("audit kinds %v", kinds)
	}
}

func auditKinds(st map[string]any) []string {
	raw, _ := st["audit"].([]any)
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		ev, _ := item.(map[string]any)
		if k, _ := ev["kind"].(string); k != "" {
			out = append(out, k)
		}
	}
	return out
}

func containsKind(kinds []string, want string) bool {
	for _, k := range kinds {
		if k == want {
			return true
		}
	}
	return false
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
		if r["readerId"] == "r1" && r["selected"] == true {
			t.Fatalf("r1 must not stay selected: %v", st["readers"])
		}
	}
	if !found {
		t.Fatalf("expected r2 selected: %v", st["readers"])
	}
	kinds := auditKinds(st)
	if !containsKind(kinds, "reader.selected") {
		t.Fatalf("audit missing reader.selected: %v", kinds)
	}
	foundID := false
	for _, item := range st["audit"].([]any) {
		ev := item.(map[string]any)
		if ev["kind"] == "reader.selected" && ev["readerId"] == "r2" {
			foundID = true
		}
	}
	if !foundID {
		t.Fatalf("reader.selected must note readerId: %v", st["audit"])
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
	assertNoForbiddenJSON(t, raw)
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

func TestManagerReconnectRecoversPCSCUnavailable(t *testing.T) {
	tb := startTestBridgeSeed(t, false, func(mon, rdb *pcsc.SyntheticBackend) {
		rdb.SeedDemoCard()
	})
	tb.srv.SetRuntime("0.0.1-test", true)
	waitReader(t, tb, "synthetic-acr39u")
	token := csrfFrom(t, getBody(t, tb, "/"))

	tb.mon.SetServiceDown(true)
	if err := tb.mgr.Refresh(context.Background()); err == nil {
		t.Fatal("refresh while card service is down must fail")
	}
	st := managerGET(t, tb, token, "/v1/manager/status")
	backend, _ := st["backend"].(map[string]any)
	if backend["status"] != "unavailable" {
		t.Fatalf("status while down=%v", st["backend"])
	}

	out := managerPOST(t, tb, token, "/v1/manager/reconnect", nil)
	if out["ok"] != false {
		t.Fatalf("reconnect while down=%v", out)
	}
	if out["code"] != "PCSC_UNAVAILABLE" {
		t.Fatalf("reconnect code=%v", out["code"])
	}
	raw, _ := json.Marshal(out)
	assertNoForbiddenJSON(t, raw)

	tb.mon.SetServiceDown(false)
	out = managerPOST(t, tb, token, "/v1/manager/reconnect", nil)
	if out["ok"] != true {
		t.Fatalf("reconnect after restore=%v", out)
	}
	st = managerGET(t, tb, token, "/v1/manager/status")
	backend, _ = st["backend"].(map[string]any)
	if backend["status"] != "available" {
		t.Fatalf("status after restore=%v", st["backend"])
	}
	readers, _ := st["readers"].([]any)
	if len(readers) != 1 {
		t.Fatalf("readers after restore=%v", st["readers"])
	}
	r0 := readers[0].(map[string]any)
	if r0["presence"] != "present" {
		t.Fatalf("presence after restore=%v", r0["presence"])
	}
	if bytes.Contains([]byte(fmt.Sprintf("%v", st)), []byte("1234567890121")) {
		t.Fatalf("status leaked personal fields")
	}

	js := getBody(t, tb, "/manager/app.js")
	if !strings.Contains(js, "pageshow") || !strings.Contains(js, "gentleWakeRefresh") {
		t.Fatal("Ready page must re-probe after sleep/wake without card.read")
	}
	if !strings.Contains(js, "serviceBack") || !strings.Contains(js, "Reconnect card service") {
		t.Fatal("Ready page missing card-service recovery copy")
	}
}

func TestManagerAuditExportRedacted(t *testing.T) {
	tb := startTestBridge(t, false)
	tb.srv.SetRuntime("0.0.1-test", true)
	tb.srv.recordAudit(audit.Event{Kind: audit.KindFillFailure, Code: "NO_CARD", Origin: "https://erp.clinic.example"})
	tb.srv.recordAudit(audit.Event{Kind: audit.KindFillFailure, Code: "NO_CARD สมชาย 1234567890121", Origin: "https://evil.example"})
	token := csrfFrom(t, getBody(t, tb, "/"))

	st := managerGET(t, tb, token, "/v1/manager/status")
	raw, _ := json.Marshal(st)
	assertNoForbiddenJSON(t, raw)
	exp := managerGET(t, tb, token, "/v1/manager/audit")
	if exp["schema"] != "openthaiid.audit.v1" {
		t.Fatalf("schema %v", exp["schema"])
	}
	pii, _ := exp["pii"].(string)
	if !strings.Contains(pii, "never stored") {
		t.Fatalf("pii note %v", exp["pii"])
	}
	body, _ := json.Marshal(exp)
	assertNoForbiddenJSON(t, body)
	js := getBody(t, tb, "/manager/app.js")
	if !strings.Contains(js, "openthaiid.audit.v1") || !strings.Contains(js, "downloadAuditJSON") {
		t.Fatal("manager JS missing audit export")
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
