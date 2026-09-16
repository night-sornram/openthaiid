package update

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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
	if !res.Newer || res.Tag != "v0.1.0-rc.1" {
		t.Fatalf("%+v", res)
	}
	if res.AssetName != "openthaiid-0.1.0-rc.1-linux-amd64.tar.gz" {
		t.Fatalf("asset %s", res.AssetName)
	}
	if strings.Contains(res.Message, "1234567890121") {
		t.Fatal("update message leaked a personal sentinel")
	}
	if !strings.Contains(res.Message, "unsigned") {
		t.Fatalf("expected unsigned notice: %s", res.Message)
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
	if res.Tag != "v0.0.2" || res.Prerelease {
		t.Fatalf("%+v", res)
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
