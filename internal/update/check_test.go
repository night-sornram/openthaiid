package update

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCheckPicksNewerPrerelease(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == "" {
			t.Errorf("missing User-Agent")
		}
		_ = json.NewEncoder(w).Encode([]ghRelease{
			{
				TagName:    "v0.1.0-rc.1",
				Prerelease: true,
				HTMLURL:    "https://example.test/r/v0.1.0-rc.1",
				Published:  time.Date(2026, 9, 16, 0, 0, 0, 0, time.UTC),
				Assets: []ghAsset{{
					Name:               "openthaiid-0.1.0-rc.1-linux-amd64.tar.gz",
					BrowserDownloadURL: "https://example.test/dl",
				}},
			},
			{
				TagName:   "v0.0.1",
				HTMLURL:   "https://example.test/r/v0.0.1",
				Published: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		})
	}))
	defer srv.Close()

	res, err := Check(context.Background(), "0.0.1-dev", Options{
		API:        srv.URL,
		HTTPClient: srv.Client(),
		GOOS:       "linux",
		GOARCH:     "amd64",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Newer || res.Tag != "v0.1.0-rc.1" || res.State != StateNewer {
		t.Fatalf("%+v", res)
	}
	if res.AssetName != "openthaiid-0.1.0-rc.1-linux-amd64.tar.gz" {
		t.Fatalf("asset %s", res.AssetName)
	}
	if strings.Contains(res.Message, "1234567890121") || strings.Contains(res.MessageTH, "1234567890121") {
		t.Fatal("update message leaked a personal sentinel")
	}
	if !strings.Contains(res.Message, "unsigned") {
		t.Fatalf("expected unsigned notice: %s", res.Message)
	}
	if !strings.Contains(res.Message, "does not download") {
		t.Fatalf("check must not auto-install: %s", res.Message)
	}
	if strings.Contains(res.Message, "update apply") || strings.Contains(res.MessageTH, "update apply") {
		t.Fatalf("staff/check copy must not push apply: %s", res.Message)
	}
	if !strings.Contains(res.MessageTH, "มีเวอร์ชันใหม่") {
		t.Fatalf("thai newer copy: %s", res.MessageTH)
	}
	pub := res.Public()
	if pub.HTMLURL == "" {
		t.Fatal("public JSON needs the release URL")
	}
	if strings.Contains(pub.Message, "example.test/dl") || strings.Contains(pub.MessageTH, "example.test/dl") {
		t.Fatalf("public JSON must omit asset download URL: %+v", pub)
	}
}

func TestCheckStableOnlySkipsPrerelease(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]ghRelease{
			{TagName: "v0.1.0-rc.1", Prerelease: true},
			{TagName: "v0.0.2"},
		})
	}))
	defer srv.Close()
	res, err := Check(context.Background(), "0.0.1", Options{
		API:        srv.URL,
		HTTPClient: srv.Client(),
		StableOnly: true,
		GOOS:       "linux",
		GOARCH:     "amd64",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Tag != "v0.0.2" || res.Prerelease || res.State != StateNewer {
		t.Fatalf("%+v", res)
	}
}

func TestCheckUpToDate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]ghRelease{
			{TagName: "v0.1.0-rc.4", Prerelease: true, HTMLURL: "https://example.test/r/v0.1.0-rc.4"},
		})
	}))
	defer srv.Close()
	res, err := Check(context.Background(), "0.1.0-rc.4", Options{API: srv.URL, HTTPClient: srv.Client()})
	if err != nil {
		t.Fatal(err)
	}
	if res.Newer || res.State != StateUpToDate {
		t.Fatalf("%+v", res)
	}
	if !strings.Contains(res.Message, "up to date") {
		t.Fatalf("%s", res.Message)
	}
	if !strings.Contains(res.MessageTH, "ใช้เวอร์ชันล่าสุดแล้ว") {
		t.Fatalf("%s", res.MessageTH)
	}
}

func TestCheckNoReleases(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]ghRelease{})
	}))
	defer srv.Close()
	res, err := Check(context.Background(), "0.1.0-rc.4", Options{API: srv.URL, HTTPClient: srv.Client()})
	if err != nil {
		t.Fatal(err)
	}
	if res.State != StateNone || res.Newer {
		t.Fatalf("%+v", res)
	}
}

func TestCheckGitHub404IsUnavailableNotEmpty(t *testing.T) {
	// Unauthenticated private-repo 404, or /releases/latest when every
	// release is a prerelease, must not print "no releases published."
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"Not Found"}`))
	}))
	defer srv.Close()
	res, err := Check(context.Background(), "0.1.0-rc.4", Options{API: srv.URL, HTTPClient: srv.Client()})
	if err != nil {
		t.Fatal(err)
	}
	if res.State != StateUnavailable || res.Reason != ReasonNotFound {
		t.Fatalf("%+v", res)
	}
	if strings.Contains(res.Message, "No GitHub Releases published") {
		t.Fatalf("404 must not look like an empty catalog: %s", res.Message)
	}
	if !strings.Contains(res.Message, "Prereleases count") {
		t.Fatalf("expected prerelease hint: %s", res.Message)
	}
	if !strings.Contains(res.MessageTH, "กรอกข้อมูลได้ตามปกติ") {
		t.Fatalf("%s", res.MessageTH)
	}
}

func TestCheckListEndpointNotLatest(t *testing.T) {
	var gotPath, gotQuery, gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		gotAuth = r.Header.Get("Authorization")
		_ = json.NewEncoder(w).Encode([]ghRelease{
			{TagName: "v0.1.0-rc.4", Prerelease: true, HTMLURL: "https://example.test/r/v0.1.0-rc.4"},
		})
	}))
	defer srv.Close()
	res, err := Check(context.Background(), "0.1.0-rc.1", Options{
		API:        srv.URL + "/releases",
		HTTPClient: srv.Client(),
		Token:      "test-token",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.State != StateNewer || res.Tag != "v0.1.0-rc.4" || !res.Prerelease {
		t.Fatalf("%+v", res)
	}
	if gotPath != "/releases" || strings.Contains(gotPath, "latest") {
		t.Fatalf("must list /releases not /releases/latest, path=%s", gotPath)
	}
	if !strings.Contains(gotQuery, "per_page=100") {
		t.Fatalf("query %s", gotQuery)
	}
	if gotAuth != "Bearer test-token" {
		t.Fatalf("auth %q", gotAuth)
	}
}

func TestCheckPaginatesAndKeepsPrereleases(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("page")
		if page == "2" {
			_ = json.NewEncoder(w).Encode([]ghRelease{
				{TagName: "v0.1.0-rc.4", Prerelease: true, HTMLURL: "https://example.test/r/v0.1.0-rc.4"},
			})
			return
		}
		w.Header().Set("Link", fmt.Sprintf(`<%s?page=2>; rel="next"`, "http://"+r.Host+"/releases"))
		_ = json.NewEncoder(w).Encode([]ghRelease{
			{TagName: "v0.1.0-rc.1", Prerelease: true},
			{TagName: "v0.1.0-rc.3", Prerelease: true},
		})
	}))
	defer srv.Close()
	res, err := Check(context.Background(), "0.1.0-rc.2", Options{
		API:        srv.URL + "/releases",
		HTTPClient: srv.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Tag != "v0.1.0-rc.4" || !res.Newer || !res.Prerelease {
		t.Fatalf("pagination missed rc.4: %+v", res)
	}
}

func TestCheckStableOnlyMentionsSkippedPrereleases(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]ghRelease{
			{TagName: "v0.1.0-rc.1", Prerelease: true},
			{TagName: "v0.1.0-rc.4", Prerelease: true},
		})
	}))
	defer srv.Close()
	res, err := Check(context.Background(), "0.1.0-rc.4", Options{
		API:        srv.URL,
		HTTPClient: srv.Client(),
		StableOnly: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.State != StateNone || !res.SkippedPrereleases {
		t.Fatalf("%+v", res)
	}
	if !strings.Contains(res.Message, "omit --stable") {
		t.Fatalf("%s", res.Message)
	}
	if !strings.Contains(res.MessageTH, "รุ่นทดลอง") {
		t.Fatalf("%s", res.MessageTH)
	}
}

func TestCheckSoftFailOffline(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	cli := srv.Client()
	srv.Close()
	res, err := Check(context.Background(), "0.1.0-rc.4", Options{API: srv.URL, HTTPClient: cli})
	if err != nil {
		t.Fatal(err)
	}
	if res.State != StateUnavailable || res.Reason != ReasonOffline {
		t.Fatalf("%+v", res)
	}
	if !strings.Contains(res.MessageTH, "กรอกข้อมูลได้ตามปกติ") {
		t.Fatalf("%s", res.MessageTH)
	}
}

func TestCheckSoftFailRateLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"message":"API rate limit exceeded"}`))
	}))
	defer srv.Close()
	res, err := Check(context.Background(), "0.1.0-rc.4", Options{API: srv.URL, HTTPClient: srv.Client()})
	if err != nil {
		t.Fatal(err)
	}
	if res.State != StateUnavailable || res.Reason != ReasonRateLimited {
		t.Fatalf("%+v", res)
	}
	if !strings.Contains(res.Message, "rate limited") {
		t.Fatalf("%s", res.Message)
	}
}

func TestCheckSoftFailHTTPServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()
	res, err := Check(context.Background(), "0.1.0-rc.4", Options{API: srv.URL, HTTPClient: srv.Client()})
	if err != nil {
		t.Fatal(err)
	}
	if res.State != StateUnavailable || res.Reason != ReasonHTTP {
		t.Fatalf("%+v", res)
	}
}

func TestOptionsOwnerRepo(t *testing.T) {
	o := Options{Owner: "acme", Repo: "fork"}
	if got := o.api(); got != "https://api.github.com/repos/acme/fork/releases" {
		t.Fatalf("api %s", got)
	}
	if got := o.releasesPage(); got != "https://github.com/acme/fork/releases" {
		t.Fatalf("page %s", got)
	}
	t.Setenv("OPENTHAIID_UPDATE_REPO", "other/tree")
	fromEnv := Options{}.WithEnv()
	owner, repo := fromEnv.ownerRepo()
	if owner != "other" || repo != "tree" {
		t.Fatalf("env %s/%s", owner, repo)
	}
}

func TestApplyArchiveVerifiesChecksum(t *testing.T) {
	dir := t.TempDir()
	bin := []byte("fake-openthaiid-binary")
	sum := sha256.Sum256(bin)
	archive := filepath.Join(dir, "pkg.tar.gz")
	if err := writeTarGz(archive, map[string][]byte{
		"openthaiid":     bin,
		"SHA256SUMS":     []byte(hex.EncodeToString(sum[:]) + "  openthaiid\n"),
		"START-HERE.txt": []byte("unsigned\n"),
	}); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(dir, "install", "openthaiid")
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dest, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	applied, err := ApplyArchive(archive, dest)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(applied)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(bin) {
		t.Fatalf("applied %q", got)
	}
}

func TestApplyArchiveRejectsChecksumMismatch(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "pkg.tar.gz")
	if err := writeTarGz(archive, map[string][]byte{
		"openthaiid": []byte("bin"),
		"SHA256SUMS": []byte("0000000000000000000000000000000000000000000000000000000000000000  openthaiid\n"),
	}); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(dir, "openthaiid")
	if err := os.WriteFile(dest, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := ApplyArchive(archive, dest); err == nil {
		t.Fatal("expected checksum error")
	}
}

func writeTarGz(path string, files map[string][]byte) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	gz := gzip.NewWriter(f)
	defer gz.Close()
	tw := tar.NewWriter(gz)
	defer tw.Close()
	for name, body := range files {
		hdr := &tar.Header{Name: name, Mode: 0o644, Size: int64(len(body))}
		if name == "openthaiid" || strings.HasSuffix(name, ".exe") {
			hdr.Mode = 0o755
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if _, err := tw.Write(body); err != nil {
			return err
		}
	}
	return nil
}

func TestDownloadWritesAsset(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "archive-bytes")
	}))
	defer srv.Close()
	dir := t.TempDir()
	path, err := Download(context.Background(), dir, srv.URL+"/file", "openthaiid-x.tar.gz", Options{HTTPClient: srv.Client()})
	if err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(path)
	if string(got) != "archive-bytes" {
		t.Fatalf("got %q", got)
	}
}
