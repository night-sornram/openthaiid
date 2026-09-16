package update

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Download fetches an unsigned GitHub Release asset into destDir.
func Download(ctx context.Context, destDir, assetURL, assetName string, opts Options) (string, error) {
	if assetURL == "" || assetName == "" {
		return "", fmt.Errorf("no matching release asset for this OS/arch")
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(destDir, filepath.Base(assetName))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, assetURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/octet-stream")
	resp, err := opts.client().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download: HTTP %d", resp.StatusCode)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(f, io.LimitReader(resp.Body, 200<<20)); err != nil {
		_ = f.Close()
		return "", err
	}
	if err := f.Close(); err != nil {
		return "", err
	}
	return path, nil
}

// ApplyArchive extracts the helper binary from a package archive and replaces dest.
// SHA256SUMS is verified when present in the archive. dest is typically
// os.Executable(). On Windows the replacement is written next to the running
// exe as *.new because a locked image cannot be overwritten.
func ApplyArchive(archivePath, dest string) (applied string, err error) {
	binName := "openthaiid"
	if strings.HasSuffix(strings.ToLower(dest), ".exe") {
		binName = "openthaiid.exe"
	}
	tmp, err := os.MkdirTemp(filepath.Dir(dest), "openthaiid-update-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmp)

	files, err := extractArchive(archivePath, tmp)
	if err != nil {
		return "", err
	}
	sumsPath := ""
	binPath := ""
	for _, p := range files {
		base := filepath.Base(p)
		switch base {
		case "SHA256SUMS":
			sumsPath = p
		case binName:
			binPath = p
		}
		if strings.HasSuffix(p, string(filepath.Separator)+binName) {
			binPath = p
		}
	}
	if binPath == "" {
		// macOS .app layout
		cand := filepath.Join(tmp, "OpenThaiID.app", "Contents", "MacOS", "openthaiid")
		if st, statErr := os.Stat(cand); statErr == nil && !st.IsDir() {
			binPath = cand
		}
	}
	if binPath == "" {
		return "", fmt.Errorf("archive does not contain %s", binName)
	}
	if sumsPath != "" {
		if err := verifySHA256(sumsPath, binPath); err != nil {
			return "", err
		}
	}

	target := dest
	if strings.EqualFold(filepath.Ext(dest), ".exe") {
		target = dest + ".new"
	}
	if err := copyFile(binPath, target+".tmp", 0o755); err != nil {
		return "", err
	}
	if err := os.Rename(target+".tmp", target); err != nil {
		return "", err
	}
	return target, nil
}

func extractArchive(archivePath, dest string) ([]string, error) {
	lower := strings.ToLower(archivePath)
	switch {
	case strings.HasSuffix(lower, ".zip"):
		return extractZip(archivePath, dest)
	case strings.HasSuffix(lower, ".tar.gz") || strings.HasSuffix(lower, ".tgz"):
		return extractTarGz(archivePath, dest)
	default:
		return nil, fmt.Errorf("unsupported archive %s", filepath.Base(archivePath))
	}
}

func extractZip(archivePath, dest string) ([]string, error) {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	var out []string
	for _, f := range r.File {
		if err := safeJoin(dest, f.Name); err != nil {
			return nil, err
		}
		path := filepath.Join(dest, f.Name)
		if strings.HasSuffix(f.Name, "/") {
			if err := os.MkdirAll(path, 0o755); err != nil {
				return nil, err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, err
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		if err := writeLimited(path, rc, 80<<20); err != nil {
			_ = rc.Close()
			return nil, err
		}
		_ = rc.Close()
		out = append(out, path)
	}
	return out, nil
}

func extractTarGz(archivePath, dest string) ([]string, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	var out []string
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if err := safeJoin(dest, hdr.Name); err != nil {
			return nil, err
		}
		path := filepath.Join(dest, hdr.Name)
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(path, 0o755); err != nil {
				return nil, err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return nil, err
			}
			if err := writeLimited(path, tr, 80<<20); err != nil {
				return nil, err
			}
			_ = os.Chmod(path, os.FileMode(hdr.Mode)&0o777)
			out = append(out, path)
		}
	}
	return out, nil
}

func safeJoin(root, name string) error {
	path := filepath.Join(root, filepath.Clean(name))
	rel, err := filepath.Rel(root, path)
	if err != nil || strings.HasPrefix(rel, "..") || rel == ".." {
		return fmt.Errorf("refusing path %q", name)
	}
	return nil
}

func writeLimited(path string, r io.Reader, n int64) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, io.LimitReader(r, n))
	return err
}

func copyFile(src, dest string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

func verifySHA256(sumsPath, binPath string) error {
	raw, err := os.ReadFile(sumsPath)
	if err != nil {
		return err
	}
	wantName := filepath.Base(binPath)
	var want string
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		name := strings.TrimPrefix(fields[len(fields)-1], "*")
		if filepath.Base(name) == wantName {
			want = strings.ToLower(fields[0])
			break
		}
	}
	if want == "" {
		return fmt.Errorf("SHA256SUMS has no entry for %s", wantName)
	}
	f, err := os.Open(binPath)
	if err != nil {
		return err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	got := hex.EncodeToString(h.Sum(nil))
	if got != want {
		return fmt.Errorf("checksum mismatch for %s", wantName)
	}
	return nil
}
