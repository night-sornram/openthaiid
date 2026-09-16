package bridge

import (
	"crypto/rand"
	"crypto/subtle"
	"embed"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/night-sornram/openthaiid/internal/audit"
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
	Version       string             `json:"version"`
	InstanceID    string             `json:"instanceId"`
	Running       bool               `json:"running"`
	Synthetic     bool               `json:"synthetic"`
	ListenAddr    string             `json:"listenAddr"`
	ManagerURL    string             `json:"managerUrl"`
	WSURL         string             `json:"wsUrl"`
	OS            string             `json:"os"`
	LogLevel      string             `json:"logLevel"`
	Connections   int                `json:"connections"`
	Backend       managerBackend     `json:"backend"`
	Readers       []managerReader    `json:"readers"`
	Pending       []managerPending   `json:"pending"`
	Trusted       []managerGrant     `json:"trusted"`
	TrustDuration string             `json:"trustDuration"`
	LastErrors    []managerLastError `json:"lastErrors"`
	Audit         []audit.Event      `json:"audit"`
}

// managerLastError is an operational failure for technicians. Messages never
// include citizen ID, names, dates, or other personal card field values.
type managerLastError struct {
	At      string `json:"at"`
	Source  string `json:"source"`
	Code    string `json:"code"`
	Message string `json:"message"`
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
		ce := card.AsError(err)
		s.recordError("refresh", string(ce.Code), ce.Message)
		s.writeJSON(w, http.StatusOK, map[string]any{
			"ok":      false,
			"code":    ce.Code,
			"message": ce.Message,
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
	name := body.ReaderID
	for _, r := range s.readers.Snapshot().Readers {
		if r.ID == body.ReaderID {
			name = r.Name
			break
		}
	}
	s.recordAudit(audit.Event{
		Kind:     audit.KindReaderSelected,
		Reader:   name,
		ReaderID: body.ReaderID,
	})
	s.broadcastStatusSnapshot()
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

func (s *Server) handleManagerReconnect(w http.ResponseWriter, r *http.Request) {
	if !s.requireManagerAPI(w, r) {
		return
	}
	err := s.readers.Reconnect(r.Context())
	if err != nil {
		ce := card.AsError(err)
		s.recordError("reconnect", string(ce.Code), ce.Message)
		s.writeJSON(w, http.StatusOK, map[string]any{
			"ok":      false,
			"code":    ce.Code,
			"message": ce.Message,
			"status":  s.managerStatus(),
		})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"ok": true, "status": s.managerStatus()})
}

func (s *Server) handleManagerLogLevel(w http.ResponseWriter, r *http.Request) {
	if !s.requireManagerAPI(w, r) {
		return
	}
	var body struct {
		Level   string `json:"level"`
		Verbose *bool  `json:"verbose"`
	}
	if err := decodeManagerBody(r, &body); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	level := strings.ToLower(strings.TrimSpace(body.Level))
	if body.Verbose != nil && level == "" {
		if *body.Verbose {
			level = "debug"
		} else {
			level = "info"
		}
	}
	if err := s.setLogLevel(level); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{
		"ok":       true,
		"logLevel": s.currentLogLevel(),
		"status":   s.managerStatus(),
	})
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

func selectedManagerReader(readers []managerReader) *managerReader {
	for i := range readers {
		if readers[i].Selected {
			return &readers[i]
		}
	}
	return nil
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
	s.clientsMu.Lock()
	connections := len(s.clients)
	s.clientsMu.Unlock()
	return managerStatus{
		Version:       version,
		InstanceID:    s.InstanceID(),
		Running:       true,
		Synthetic:     s.synthetic || backend.Name == "synthetic",
		ListenAddr:    s.Addr(),
		ManagerURL:    "http://" + s.Addr() + "/",
		WSURL:         s.cfg.WSURL(),
		OS:            runtime.GOOS + "/" + runtime.GOARCH,
		LogLevel:      s.currentLogLevel(),
		Connections:   connections,
		Backend:       backend,
		Readers:       readers,
		Pending:       outPending,
		Trusted:       outTrust,
		TrustDuration: s.cfg.TrustDuration.String(),
		LastErrors:    s.copyLastErrors(),
		Audit:         s.copyAudit(),
	}
}

const maxLastErrors = 8

func (s *Server) recordError(source, code, message string) {
	switch card.Code(code) {
	case card.ErrApprovalRequired, card.ErrCancelled:
		return
	}
	s.lastErrMu.Lock()
	defer s.lastErrMu.Unlock()
	s.lastErrs = append(s.lastErrs, managerLastError{
		At:      time.Now().UTC().Format(time.RFC3339),
		Source:  source,
		Code:    code,
		Message: message,
	})
	if len(s.lastErrs) > maxLastErrors {
		s.lastErrs = append([]managerLastError(nil), s.lastErrs[len(s.lastErrs)-maxLastErrors:]...)
	}
}

func (s *Server) copyLastErrors() []managerLastError {
	s.lastErrMu.Lock()
	defer s.lastErrMu.Unlock()
	if len(s.lastErrs) == 0 {
		return []managerLastError{}
	}
	out := make([]managerLastError, len(s.lastErrs))
	copy(out, s.lastErrs)
	return out
}

func (s *Server) copyAudit() []audit.Event {
	if s.audit == nil {
		return []audit.Event{}
	}
	return s.audit.Snapshot()
}

func (s *Server) recordAudit(ev audit.Event) {
	if s.audit == nil {
		return
	}
	s.audit.Record(ev)
}

func (s *Server) handleManagerAudit(w http.ResponseWriter, r *http.Request) {
	if !s.requireManagerAPI(w, r) {
		return
	}
	s.writeJSON(w, http.StatusOK, s.auditExport())
}

func (s *Server) auditExport() audit.Export {
	if s.audit == nil {
		return audit.Export{Schema: audit.Schema, PII: "omitted — citizen ID, names, address, portrait, raw APDUs are never stored", Events: []audit.Event{}}
	}
	return s.audit.ExportJSON()
}

// SetAudit replaces the operator trail. Tests may pass a memory-only log.
func (s *Server) SetAudit(l *audit.Log) {
	if l == nil {
		return
	}
	s.audit = l
}

// SetRunMode records console|tray|serve on helper.start / helper.stop.
func (s *Server) SetRunMode(mode string) {
	s.runMode = mode
}

// SetLogLeveler attaches the process-wide slog level so Technician verbose
// logging changes what the helper actually prints. Defaults stay info.
func (s *Server) SetLogLeveler(lv *slog.LevelVar) {
	if lv == nil {
		return
	}
	s.logLevel = lv
}

func (s *Server) currentLogLevel() string {
	if s.logLevel == nil {
		return "info"
	}
	switch s.logLevel.Level() {
	case slog.LevelDebug:
		return "debug"
	case slog.LevelWarn:
		return "warn"
	case slog.LevelError:
		return "error"
	default:
		return "info"
	}
}

func (s *Server) setLogLevel(level string) error {
	if s.logLevel == nil {
		s.logLevel = new(slog.LevelVar)
	}
	var lv slog.Level
	switch level {
	case "debug":
		lv = slog.LevelDebug
	case "info":
		lv = slog.LevelInfo
	case "warn":
		lv = slog.LevelWarn
	case "error":
		lv = slog.LevelError
	default:
		return card.NewError(card.ErrInvalidRequest, "invalid log level")
	}
	s.logLevel.Set(lv)
	if s.log != nil {
		s.log.Info("log level changed", "level", level)
	}
	return nil
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
