package bridge

import (
	"crypto/rand"
	"crypto/subtle"
	"embed"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/night-sornram/openthaiid/internal/auth"
	"github.com/night-sornram/openthaiid/internal/card"
)

//go:embed managerui/index.html managerui/app.js managerui/style.css
var managerUI embed.FS

const managerTokenHeader = "X-OpenThaiID-Manager"
const managerBodyLimit = 4096

type probeView struct {
	Version    string          `json:"version"`
	InstanceID string          `json:"instanceId"`
	Synthetic  bool            `json:"synthetic"`
	Backend    managerBackend  `json:"backend"`
	Readers    []managerReader `json:"readers"`
	ManagerURL string          `json:"managerUrl"`
	WSURL      string          `json:"wsUrl"`
}

type managerStatus struct {
	Version       string           `json:"version"`
	InstanceID    string           `json:"instanceId"`
	Running       bool             `json:"running"`
	Synthetic     bool             `json:"synthetic"`
	ListenAddr    string           `json:"listenAddr"`
	ManagerURL    string           `json:"managerUrl"`
	WSURL         string           `json:"wsUrl"`
	Backend       managerBackend   `json:"backend"`
	Readers       []managerReader  `json:"readers"`
	Pending       []managerPending `json:"pending"`
	Trusted       []managerGrant   `json:"trusted"`
	TrustDuration string           `json:"trustDuration"`
}

type managerBackend struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

type managerReader struct {
	ID       string `json:"readerId"`
	Name     string `json:"name"`
	Presence string `json:"presence"`
	Selected bool   `json:"selected"`
}

type managerPending struct {
	ID             int64    `json:"id"`
	Origin         string   `json:"origin"`
	Fields         []string `json:"fields"`
	ComparisonCode string   `json:"comparisonCode"`
	ExpiresAt      string   `json:"expiresAt"`
}

type managerGrant struct {
	Origin    string   `json:"origin"`
	Fields    []string `json:"fields"`
	ExpiresAt string   `json:"expiresAt"`
}

func newManagerToken() string {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "uninitialized"
	}
	return hex.EncodeToString(b[:])
}

func (s *Server) managerHeaders(w http.ResponseWriter, csp string) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Content-Security-Policy", csp)
}

func (s *Server) handleManagerPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	raw, err := managerUI.ReadFile("managerui/index.html")
	if err != nil {
		http.Error(w, "manager unavailable", http.StatusInternalServerError)
		return
	}
	page := strings.Replace(string(raw), "__CSRF_TOKEN__", s.managerToken, 1)
	s.managerHeaders(w, "default-src 'self'; connect-src 'self'; img-src 'none'; object-src 'none'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.WriteString(w, page)
}

func (s *Server) handleManagerJS(w http.ResponseWriter, r *http.Request) {
	s.serveManagerStatic(w, r, "managerui/app.js", "text/javascript; charset=utf-8")
}

func (s *Server) handleManagerCSS(w http.ResponseWriter, r *http.Request) {
	s.serveManagerStatic(w, r, "managerui/style.css", "text/css; charset=utf-8")
}

func (s *Server) serveManagerStatic(w http.ResponseWriter, r *http.Request, name, ctype string) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	raw, err := managerUI.ReadFile(name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	s.managerHeaders(w, "default-src 'none'; frame-ancestors 'none'")
	w.Header().Set("Content-Type", ctype)
	_, _ = w.Write(raw)
}

func (s *Server) handleFavicon(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) requireManagerAPI(w http.ResponseWriter, r *http.Request) bool {
	if !s.managerOriginAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return false
	}
	got := r.Header.Get(managerTokenHeader)
	if subtle.ConstantTimeCompare([]byte(got), []byte(s.managerToken)) != 1 {
		http.Error(w, "forbidden", http.StatusForbidden)
		return false
	}
	return true
}

func (s *Server) managerOriginAllowed(r *http.Request) bool {
	raw := strings.TrimSpace(r.Header.Get("Origin"))
	if raw == "" {
		// Same-origin document navigation does not send Origin. Mutating
		// manager APIs still require the CSRF header, so a cross-site form
		// POST cannot succeed. GET status without Origin is loopback-only.
		return r.Method == http.MethodGet
	}
	return s.isManagerOrigin(raw)
}

func (s *Server) isManagerOrigin(raw string) bool {
	o, err := auth.NormalizeOrigin(raw)
	if err != nil {
		return false
	}
	port := portOnly(s.Addr())
	for _, host := range []string{"127.0.0.1", "localhost"} {
		want := "http://" + host + ":" + port
		if string(o) == want {
			return true
		}
	}
	return false
}

func (s *Server) handleManagerStatus(w http.ResponseWriter, r *http.Request) {
	if !s.requireManagerAPI(w, r) {
		return
	}
	s.writeJSON(w, http.StatusOK, s.managerStatus())
}

func (s *Server) handleManagerRefresh(w http.ResponseWriter, r *http.Request) {
	if !s.requireManagerAPI(w, r) {
		return
	}
	err := s.readers.Refresh(r.Context())
	if err != nil {
		s.writeJSON(w, http.StatusOK, map[string]any{
			"ok":      false,
			"code":    card.AsError(err).Code,
			"message": card.AsError(err).Message,
			"status":  s.managerStatus(),
		})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"ok": true, "status": s.managerStatus()})
}

func (s *Server) handleManagerSelectReader(w http.ResponseWriter, r *http.Request) {
	if !s.requireManagerAPI(w, r) {
		return
	}
	var body struct {
		ReaderID string `json:"readerId"`
	}
	if err := decodeManagerBody(r, &body); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if err := s.readers.SetPreferredReader(body.ReaderID); err != nil {
		ce := card.AsError(err)
		s.writeJSON(w, http.StatusConflict, map[string]any{"error": ce.Message, "code": ce.Code})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"ok": true, "status": s.managerStatus()})
}

func (s *Server) handleManagerAllow(w http.ResponseWriter, r *http.Request) {
	s.resolveManagerApproval(w, r, true)
}

func (s *Server) handleManagerDeny(w http.ResponseWriter, r *http.Request) {
	s.resolveManagerApproval(w, r, false)
}

func (s *Server) resolveManagerApproval(w http.ResponseWriter, r *http.Request, allow bool) {
	if !s.requireManagerAPI(w, r) {
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if _, err := s.auth.Resolve(id, allow, time.Now()); err != nil {
		ce := card.AsError(err)
		s.writeJSON(w, http.StatusConflict, map[string]any{"error": ce.Message, "code": ce.Code})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"ok": true, "status": s.managerStatus()})
}

func (s *Server) handleManagerRevoke(w http.ResponseWriter, r *http.Request) {
	if !s.requireManagerAPI(w, r) {
		return
	}
	var body struct {
		Origin string `json:"origin"`
	}
	if err := decodeManagerBody(r, &body); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	origin, err := auth.NormalizeOrigin(body.Origin)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	n := s.auth.RevokeOrigin(origin)
	s.writeJSON(w, http.StatusOK, map[string]any{"ok": true, "revoked": n, "status": s.managerStatus()})
}

func (s *Server) handleManagerRevokeAll(w http.ResponseWriter, r *http.Request) {
	if !s.requireManagerAPI(w, r) {
		return
	}
	s.auth.RevokeAll()
	s.writeJSON(w, http.StatusOK, map[string]any{"ok": true, "status": s.managerStatus()})
}

func (s *Server) probeView() probeView {
	backend, readers := s.deviceView()
	version := s.version
	if version == "" {
		version = "dev"
	}
	return probeView{
		Version:    version,
		InstanceID: s.InstanceID(),
		Synthetic:  s.synthetic || backend.Name == "synthetic",
		Backend:    backend,
		Readers:    readers,
		ManagerURL: s.ManagerURL(),
		WSURL:      s.cfg.WSURL(),
	}
}

func (s *Server) deviceView() (managerBackend, []managerReader) {
	snap := s.readers.Snapshot()
	preferred := snap.PreferredReaderID
	readers := make([]managerReader, 0, len(snap.Readers))
	for _, r := range snap.Readers {
		readers = append(readers, managerReader{
			ID:       r.ID,
			Name:     r.Name,
			Presence: string(r.Presence),
			Selected: preferred != "" && r.ID == preferred,
		})
	}
	sort.Slice(readers, func(i, j int) bool {
		if readers[i].Name != readers[j].Name {
			return readers[i].Name < readers[j].Name
		}
		return readers[i].ID < readers[j].ID
	})
	if preferred == "" && len(readers) == 1 {
		readers[0].Selected = true
	}
	backendName := snap.BackendName
	if backendName == "" {
		backendName = s.readers.BackendName()
	}
	return managerBackend{Name: backendName, Status: snap.Backend}, readers
}

func (s *Server) managerStatus() managerStatus {
	now := time.Now()
	backend, readers := s.deviceView()
	pending := s.auth.Pending(now)
	outPending := make([]managerPending, 0, len(pending))
	for _, p := range pending {
		outPending = append(outPending, managerPending{
			ID:             p.ID,
			Origin:         string(p.Origin),
			Fields:         fieldsToStrings(p.Fields),
			ComparisonCode: p.Comparison,
			ExpiresAt:      p.ExpiresAt.UTC().Format(time.RFC3339),
		})
	}
	grants := s.auth.TrustedOrigins(now)
	outTrust := make([]managerGrant, 0, len(grants))
	for _, g := range grants {
		outTrust = append(outTrust, managerGrant{
			Origin:    string(g.Origin),
			Fields:    fieldsToStrings(g.Fields),
			ExpiresAt: g.ExpiresAt.UTC().Format(time.RFC3339),
		})
	}

	version := s.version
	if version == "" {
		version = "dev"
	}
	return managerStatus{
		Version:       version,
		InstanceID:    s.InstanceID(),
		Running:       true,
		Synthetic:     s.synthetic || backend.Name == "synthetic",
		ListenAddr:    s.Addr(),
		ManagerURL:    "http://" + s.Addr() + "/",
		WSURL:         s.cfg.WSURL(),
		Backend:       backend,
		Readers:       readers,
		Pending:       outPending,
		Trusted:       outTrust,
		TrustDuration: s.cfg.TrustDuration.String(),
	}
}

func fieldsToStrings(fields []card.Field) []string {
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		out = append(out, string(f))
	}
	return out
}

func decodeManagerBody(r *http.Request, dest any) error {
	defer r.Body.Close()
	dec := json.NewDecoder(io.LimitReader(r.Body, managerBodyLimit))
	return dec.Decode(dest)
}

func (s *Server) writeJSON(w http.ResponseWriter, status int, v any) {
	s.managerHeaders(w, "default-src 'none'; frame-ancestors 'none'")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// SetRuntime records CLI version and whether the synthetic backend is in use.
func (s *Server) SetRuntime(version string, synthetic bool) {
	s.version = version
	s.synthetic = synthetic
}

// ManagerURL is the loopback HTML manager.
func (s *Server) ManagerURL() string {
	return "http://" + s.Addr() + "/"
}
