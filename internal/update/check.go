package update

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"
	"time"
)

const (
	// Repo is the GitHub repository used for unsigned GitHub Releases.
	RepoOwner = "night-sornram"
	RepoName  = "openthaiid"

	ReleasesAPI  = "https://api.github.com/repos/" + RepoOwner + "/" + RepoName + "/releases"
	ReleasesPage = "https://github.com/" + RepoOwner + "/" + RepoName + "/releases"

	userAgent = "openthaiid-update"
)

// CheckResult is a comparison of the running binary against GitHub Releases.
// It never includes personal card data.
type CheckResult struct {
	Current    string
	Latest     string
	Newer      bool
	Prerelease bool
	Tag        string
	HTMLURL    string
	AssetName  string
	AssetURL   string
	Published  string
	Message    string
}

type ghRelease struct {
	TagName    string    `json:"tag_name"`
	Name       string    `json:"name"`
	Draft      bool      `json:"draft"`
	Prerelease bool      `json:"prerelease"`
	HTMLURL    string    `json:"html_url"`
	Published  time.Time `json:"published_at"`
	Assets     []ghAsset `json:"assets"`
}

type ghAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// Options control which releases count as candidates.
type Options struct {
	// StableOnly ignores GitHub prereleases (rc/beta). Default includes them
	// so an RC can update to a newer RC.
	StableOnly bool
	HTTPClient *http.Client
	API        string
	GOOS       string
	GOARCH     string
}

func (o Options) client() *http.Client {
	if o.HTTPClient != nil {
		return o.HTTPClient
	}
	return &http.Client{Timeout: 15 * time.Second}
}

func (o Options) api() string {
	if o.API != "" {
		return o.API
	}
	return ReleasesAPI
}

func (o Options) goos() string {
	if o.GOOS != "" {
		return o.GOOS
	}
	return runtime.GOOS
}

func (o Options) goarch() string {
	if o.GOARCH != "" {
		return o.GOARCH
	}
	return runtime.GOARCH
}

// Check lists GitHub Releases and reports whether a newer package exists.
func Check(ctx context.Context, current string, opts Options) (*CheckResult, error) {
	rels, err := listReleases(ctx, opts)
	if err != nil {
		return nil, err
	}
	var best *ghRelease
	for i := range rels {
		r := &rels[i]
		if r.Draft {
			continue
		}
		if opts.StableOnly && r.Prerelease {
			continue
		}
		if r.TagName == "" {
			continue
		}
		if best == nil || CompareVersions(r.TagName, best.TagName) > 0 {
			best = r
		}
	}
	if best == nil {
		return &CheckResult{
			Current: current,
			Message: "No GitHub Releases published yet. Packages stay unsigned until a human tags a release.\n" + ReleasesPage,
		}, nil
	}
	assetName, assetURL := matchAsset(best.Assets, opts.goos(), opts.goarch())
	newer := CompareVersions(best.TagName, current) > 0
	res := &CheckResult{
		Current:    current,
		Latest:     strings.TrimPrefix(strings.TrimPrefix(best.TagName, "v"), "V"),
		Newer:      newer,
		Prerelease: best.Prerelease,
		Tag:        best.TagName,
		HTMLURL:    best.HTMLURL,
		AssetName:  assetName,
		AssetURL:   assetURL,
		Published:  best.Published.UTC().Format(time.RFC3339),
	}
	res.Message = res.UserMessage()
	return res, nil
}

func (r CheckResult) UserMessage() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Running: %s\n", r.Current)
	if r.Latest == "" {
		b.WriteString(r.Message)
		return strings.TrimSpace(b.String())
	}
	fmt.Fprintf(&b, "GitHub:  %s", r.Tag)
	if r.Prerelease {
		b.WriteString(" (prerelease)")
	}
	b.WriteByte('\n')
	if r.Newer {
		b.WriteString("A newer package is available.\n")
	} else {
		b.WriteString("This helper is up to date (or newer than the latest GitHub Release).\n")
	}
	if r.AssetName != "" {
		fmt.Fprintf(&b, "Asset:   %s\n", r.AssetName)
	} else if r.Newer {
		b.WriteString("No matching OS/arch asset on that release.\n")
	}
	b.WriteString("Packages are unsigned. Verify SHA256SUMS inside the archive.\n")
	if r.HTMLURL != "" {
		fmt.Fprintf(&b, "%s\n", r.HTMLURL)
	} else {
		fmt.Fprintf(&b, "%s\n", ReleasesPage)
	}
	b.WriteString("Apply with: openthaiid update apply --yes\n")
	return strings.TrimSpace(b.String())
}

func listReleases(ctx context.Context, opts Options) ([]ghRelease, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, opts.api(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := opts.client().Do(req)
	if err != nil {
		return nil, fmt.Errorf("github releases: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github releases: HTTP %d", resp.StatusCode)
	}
	var rels []ghRelease
	if err := json.Unmarshal(body, &rels); err != nil {
		return nil, fmt.Errorf("github releases: %w", err)
	}
	return rels, nil
}

func matchAsset(assets []ghAsset, goos, goarch string) (name, url string) {
	want := fmt.Sprintf("-%s-%s", goos, goarch)
	ext := ".tar.gz"
	if goos == "windows" {
		ext = ".zip"
	}
	for _, a := range assets {
		if strings.Contains(a.Name, want) && strings.HasSuffix(a.Name, ext) && strings.HasPrefix(a.Name, "openthaiid-") {
			return a.Name, a.BrowserDownloadURL
		}
	}
	return "", ""
}
