package update

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
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

	StateUpToDate    = "up_to_date"
	StateNewer       = "newer"
	StateNone        = "none"
	StateUnavailable = "unavailable"

	ReasonOffline     = "offline"
	ReasonRateLimited = "rate_limited"
	ReasonHTTP        = "http"
)

// CheckResult is a comparison of the running binary against GitHub Releases.
// It never includes personal card data. Check never downloads an asset.
type CheckResult struct {
	Current      string
	Latest       string
	Newer        bool
	Prerelease   bool
	Tag          string
	HTMLURL      string
	ReleasesPage string
	AssetName    string
	AssetURL     string
	Published    string
	State        string
	Reason       string
	Message      string
	MessageTH    string
}

// PublicJSON is the loopback helper payload. Asset download URLs are omitted
// so the Ready page cannot auto-fetch an unsigned package.
type PublicJSON struct {
	OK           bool   `json:"ok"`
	State        string `json:"state"`
	Reason       string `json:"reason,omitempty"`
	Current      string `json:"current"`
	Latest       string `json:"latest,omitempty"`
	Newer        bool   `json:"newer"`
	Prerelease   bool   `json:"prerelease"`
	Tag          string `json:"tag,omitempty"`
	HTMLURL      string `json:"htmlUrl,omitempty"`
	ReleasesPage string `json:"releasesPage"`
	Message      string `json:"message"`
	MessageTH    string `json:"messageTh"`
	Unsigned     bool   `json:"unsigned"`
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
	// so an RC can see a newer RC.
	StableOnly bool
	HTTPClient *http.Client
	API        string
	GOOS       string
	GOARCH     string
	// Owner and Repo override the GitHub repository. Empty uses night-sornram/openthaiid.
	Owner string
	Repo  string
	// Repository is "owner/name" (OPENTHAIID_UPDATE_REPO). Ignored when Owner+Repo are set.
	Repository string
}

func (o Options) client() *http.Client {
	if o.HTTPClient != nil {
		return o.HTTPClient
	}
	return &http.Client{Timeout: 12 * time.Second}
}

func (o Options) ownerRepo() (owner, repo string) {
	if strings.TrimSpace(o.Owner) != "" && strings.TrimSpace(o.Repo) != "" {
		return strings.TrimSpace(o.Owner), strings.TrimSpace(o.Repo)
	}
	raw := strings.TrimSpace(o.Repository)
	if raw != "" {
		owner, name, ok := strings.Cut(raw, "/")
		owner, name = strings.TrimSpace(owner), strings.TrimSpace(name)
		if ok && owner != "" && name != "" && !strings.Contains(name, "/") {
			return owner, name
		}
	}
	return RepoOwner, RepoName
}

func (o Options) api() string {
	if o.API != "" {
		return o.API
	}
	owner, repo := o.ownerRepo()
	return "https://api.github.com/repos/" + owner + "/" + repo + "/releases"
}

func (o Options) releasesPage() string {
	owner, repo := o.ownerRepo()
	if owner == RepoOwner && repo == RepoName {
		return ReleasesPage
	}
	return "https://github.com/" + owner + "/" + repo + "/releases"
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

// WithEnv fills empty Owner/Repo/API from OPENTHAIID_* environment variables.
func (o Options) WithEnv() Options {
	if o.Repository == "" {
		o.Repository = strings.TrimSpace(os.Getenv("OPENTHAIID_UPDATE_REPO"))
	}
	if o.Owner == "" {
		o.Owner = strings.TrimSpace(os.Getenv("OPENTHAIID_GITHUB_OWNER"))
	}
	if o.Repo == "" {
		o.Repo = strings.TrimSpace(os.Getenv("OPENTHAIID_GITHUB_REPO"))
	}
	if o.API == "" {
		o.API = strings.TrimSpace(os.Getenv("OPENTHAIID_UPDATE_API"))
	}
	return o
}

// DefaultOptions reads optional OPENTHAIID_UPDATE_REPO / owner / API overrides.
func DefaultOptions() Options {
	return Options{}.WithEnv()
}

type apiError struct {
	status      int
	rateLimited bool
	offline     bool
	msg         string
}

func (e *apiError) Error() string {
	if e == nil {
		return "github releases"
	}
	if e.msg != "" {
		return e.msg
	}
	if e.status != 0 {
		return fmt.Sprintf("github releases: HTTP %d", e.status)
	}
	return "github releases"
}

func classifyError(err error) string {
	var ae *apiError
	if errors.As(err, &ae) {
		if ae.rateLimited {
			return ReasonRateLimited
		}
		if ae.offline || ae.status == 0 {
			return ReasonOffline
		}
		return ReasonHTTP
	}
	return ReasonOffline
}

// Unavailable builds a soft-fail result. Ready/Fill must keep working.
func Unavailable(current, reason string, page string) *CheckResult {
	if reason == "" {
		reason = ReasonOffline
	}
	if page == "" {
		page = ReleasesPage
	}
	res := &CheckResult{
		Current:      current,
		State:        StateUnavailable,
		Reason:       reason,
		ReleasesPage: page,
	}
	res.Message = res.englishMessage()
	res.MessageTH = res.thaiMessage()
	return res
}

// Check lists GitHub Releases and reports whether a newer package exists.
// Network and GitHub rate-limit failures return a result with StateUnavailable
// and a nil error so Ready/Fill never depend on GitHub.
func Check(ctx context.Context, current string, opts Options) (*CheckResult, error) {
	opts = opts.WithEnv()
	page := opts.releasesPage()
	rels, err := listReleases(ctx, opts)
	if err != nil {
		return Unavailable(current, classifyError(err), page), nil
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
		res := &CheckResult{
			Current:      current,
			State:        StateNone,
			ReleasesPage: page,
		}
		res.Message = res.englishMessage()
		res.MessageTH = res.thaiMessage()
		return res, nil
	}
	assetName, assetURL := matchAsset(best.Assets, opts.goos(), opts.goarch())
	newer := CompareVersions(best.TagName, current) > 0
	state := StateUpToDate
	if newer {
		state = StateNewer
	}
	html := best.HTMLURL
	if html == "" {
		html = page
	}
	res := &CheckResult{
		Current:      current,
		Latest:       strings.TrimPrefix(strings.TrimPrefix(best.TagName, "v"), "V"),
		Newer:        newer,
		Prerelease:   best.Prerelease,
		Tag:          best.TagName,
		HTMLURL:      html,
		ReleasesPage: page,
		AssetName:    assetName,
		AssetURL:     assetURL,
		Published:    best.Published.UTC().Format(time.RFC3339),
		State:        state,
	}
	res.Message = res.englishMessage()
	res.MessageTH = res.thaiMessage()
	return res, nil
}

func (r CheckResult) Public() PublicJSON {
	url := r.HTMLURL
	if url == "" {
		url = r.ReleasesPage
	}
	if url == "" {
		url = ReleasesPage
	}
	page := r.ReleasesPage
	if page == "" {
		page = ReleasesPage
	}
	return PublicJSON{
		OK:           true, // HTTP 200 after the operator click; state tells the UI
		State:        r.State,
		Reason:       r.Reason,
		Current:      r.Current,
		Latest:       r.Latest,
		Newer:        r.Newer,
		Prerelease:   r.Prerelease,
		Tag:          r.Tag,
		HTMLURL:      url,
		ReleasesPage: page,
		Message:      r.StaffMessage("en"),
		MessageTH:    r.StaffMessage("th"),
		Unsigned:     true,
	}
}

func (r CheckResult) UserMessage() string {
	return r.englishMessage()
}

func (r CheckResult) StaffMessage(lang string) string {
	if strings.EqualFold(strings.TrimSpace(lang), "en") {
		return r.englishMessage()
	}
	return r.thaiMessage()
}

func (r CheckResult) englishMessage() string {
	page := r.displayURL()
	var b strings.Builder
	if r.Current != "" {
		fmt.Fprintf(&b, "Running: %s\n", r.Current)
	}
	switch r.State {
	case StateUnavailable:
		switch r.Reason {
		case ReasonRateLimited:
			b.WriteString("Could not check GitHub Releases (rate limited).\n")
		default:
			b.WriteString("Could not check GitHub Releases (offline or unreachable).\n")
		}
		b.WriteString("Fill still works. This check does not download or install.\n")
		b.WriteString(page)
		return strings.TrimSpace(b.String())
	case StateNone:
		b.WriteString("No GitHub Releases published yet.\n")
		b.WriteString("Packages stay unsigned until a human tags a release.\n")
		b.WriteString("This check does not download or install.\n")
		b.WriteString(page)
		return strings.TrimSpace(b.String())
	}
	if r.Tag != "" {
		fmt.Fprintf(&b, "GitHub:  %s", r.Tag)
		if r.Prerelease {
			b.WriteString(" (prerelease)")
		}
		b.WriteByte('\n')
	}
	if r.Newer {
		if r.Prerelease {
			b.WriteString("A newer prerelease is available.\n")
		} else {
			b.WriteString("A newer release is available.\n")
		}
	} else {
		b.WriteString("This helper is up to date (or newer than the latest GitHub Release).\n")
	}
	b.WriteString("Packages are unsigned. This check does not download or install.\n")
	b.WriteString(page)
	return strings.TrimSpace(b.String())
}

func (r CheckResult) thaiMessage() string {
	page := r.displayURL()
	var b strings.Builder
	if r.Current != "" {
		fmt.Fprintf(&b, "ใช้เวอร์ชัน: %s\n", r.Current)
	}
	switch r.State {
	case StateUnavailable:
		switch r.Reason {
		case ReasonRateLimited:
			b.WriteString("ตรวจเวอร์ชันจาก GitHub ไม่ได้ (จำกัดคำขอชั่วคราว)\n")
		default:
			b.WriteString("ตรวจเวอร์ชันจาก GitHub ไม่ได้ (ไม่มีเน็ตหรือติดต่อไม่ได้)\n")
		}
		b.WriteString("กรอกข้อมูลได้ตามปกติ — คำสั่งนี้ไม่ดาวน์โหลดและไม่ติดตั้ง\n")
		b.WriteString(page)
		return strings.TrimSpace(b.String())
	case StateNone:
		b.WriteString("ยังไม่มี GitHub Release\n")
		b.WriteString("แพ็กเกจยังไม่เซ็นชื่อ จนกว่าจะมีคนติดป้ายรุ่น\n")
		b.WriteString("คำสั่งนี้ไม่ดาวน์โหลดและไม่ติดตั้ง\n")
		b.WriteString(page)
		return strings.TrimSpace(b.String())
	}
	if r.Tag != "" {
		fmt.Fprintf(&b, "GitHub: %s", r.Tag)
		if r.Prerelease {
			b.WriteString(" (รุ่นทดลอง)")
		}
		b.WriteByte('\n')
	}
	if r.Newer {
		if r.Prerelease {
			b.WriteString("มีเวอร์ชันใหม่ (รุ่นทดลอง)\n")
		} else {
			b.WriteString("มีเวอร์ชันใหม่\n")
		}
	} else {
		b.WriteString("ใช้เวอร์ชันล่าสุดแล้ว\n")
	}
	b.WriteString("แพ็กเกจยังไม่เซ็นชื่อ — คำสั่งนี้ไม่ดาวน์โหลดและไม่ติดตั้ง\n")
	b.WriteString(page)
	return strings.TrimSpace(b.String())
}

func (r CheckResult) displayURL() string {
	if r.HTMLURL != "" {
		return r.HTMLURL
	}
	if r.ReleasesPage != "" {
		return r.ReleasesPage
	}
	return ReleasesPage
}

func listReleases(ctx context.Context, opts Options) ([]ghRelease, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, opts.api(), nil)
	if err != nil {
		return nil, &apiError{offline: true, msg: err.Error()}
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := opts.client().Do(req)
	if err != nil {
		return nil, &apiError{offline: true, msg: fmt.Sprintf("github releases: %v", err)}
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, &apiError{offline: true, msg: err.Error()}
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, &apiError{status: resp.StatusCode, rateLimited: true}
	}
	if resp.StatusCode == http.StatusForbidden {
		// Unauthenticated GitHub 403 is typically a rate limit.
		return nil, &apiError{status: resp.StatusCode, rateLimited: true}
	}
	if resp.StatusCode != http.StatusOK {
		return nil, &apiError{status: resp.StatusCode, msg: fmt.Sprintf("github releases: HTTP %d", resp.StatusCode)}
	}
	var rels []ghRelease
	if err := json.Unmarshal(body, &rels); err != nil {
		return nil, &apiError{status: resp.StatusCode, msg: fmt.Sprintf("github releases: %v", err)}
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
