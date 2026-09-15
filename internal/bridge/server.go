// Package bridge is the loopback WebSocket server implementing protocol v1.
package bridge

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/night-sornram/openthaiid/internal/auth"
	"github.com/night-sornram/openthaiid/internal/card"
	"github.com/night-sornram/openthaiid/internal/config"
	"github.com/night-sornram/openthaiid/internal/protocol"
	"github.com/night-sornram/openthaiid/internal/reader"
)

// Server is the local bridge.
type Server struct {
	cfg     config.Config
	log     *slog.Logger
	auth    *auth.Manager
	readers *reader.Manager
	engine  *card.Engine

	mu       sync.Mutex
	conns    int
	httpSrv  *http.Server
	listener net.Listener

	clientsMu sync.Mutex
	clients   map[string]*clientConn

	opSem chan struct{}
}

type clientConn struct {
	id     string
	ws     *websocket.Conn
	origin auth.Origin

	// authMu guards token and sess across the receive loop and async ops.
	authMu sync.Mutex
	token  string
	sess   *auth.Session

	writeMu sync.Mutex
	closed  bool

	lastReqID int64
	eventSeq  uint64

	cancelOps    context.CancelFunc
	cancelOpsCtx context.Context

	pendingMu sync.Mutex
	pending   map[int64]context.CancelFunc

	outq    chan []byte
	dropped bool
}

func (cc *clientConn) setSession(sess *auth.Session) {
	cc.authMu.Lock()
	if sess == nil {
		cc.token = ""
		cc.sess = nil
	} else {
		cc.token = sess.Token
		cc.sess = sess
	}
	cc.authMu.Unlock()
}

func (cc *clientConn) clearSession() {
	cc.setSession(nil)
}

func (cc *clientConn) getToken() string {
	cc.authMu.Lock()
	defer cc.authMu.Unlock()
	return cc.token
}

func (cc *clientConn) getSession() *auth.Session {
	cc.authMu.Lock()
	defer cc.authMu.Unlock()
	return cc.sess
}

func (cc *clientConn) hasSession() bool {
	cc.authMu.Lock()
	defer cc.authMu.Unlock()
	return cc.token != "" && cc.sess != nil
}

// New constructs a bridge server.
func New(cfg config.Config, log *slog.Logger, authMgr *auth.Manager, readers *reader.Manager) *Server {
	n := cfg.MaxActiveReads
	if n < 1 {
		n = 1
	}
	s := &Server{
		cfg:     cfg,
		log:     log,
		auth:    authMgr,
		readers: readers,
		engine:  card.NewEngine(card.DefaultProfile()),
		clients: map[string]*clientConn{},
		opSem:   make(chan struct{}, n),
	}
	authMgr.OnRevoke = s.handleAuthRevoked
	return s
}

// InstanceID exposes the bridge instance id used in events.
func (s *Server) InstanceID() string { return s.auth.InstanceID() }

// Start begins listening on the configured loopback address.
func (s *Server) Start(ctx context.Context) error {
	if err := s.cfg.Validate(); err != nil {
		return err
	}
	ln, err := net.Listen("tcp", s.cfg.Addr())
	if err != nil {
		return card.NewError(card.ErrConnectionFailed, "listen failed")
	}
	s.listener = ln
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealthz)
	mux.HandleFunc("/v1/ws", s.handleWS)
	s.httpSrv = &http.Server{
		Handler:           s.securityMiddleware(mux),
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		if err := s.httpSrv.Serve(ln); err != nil && err != http.ErrServerClosed {
			s.log.Error("bridge serve failed", "error", err)
		}
	}()
	go s.forwardReaderEvents(ctx)
	return nil
}

// Addr returns the bound address once listening.
func (s *Server) Addr() string {
	if s.listener != nil {
		return s.listener.Addr().String()
	}
	return s.cfg.Addr()
}

// Shutdown stops the HTTP server and closes upgraded WebSockets.
func (s *Server) Shutdown(ctx context.Context) error {
	s.clientsMu.Lock()
	clients := make([]*clientConn, 0, len(s.clients))
	for _, c := range s.clients {
		clients = append(clients, c)
	}
	s.clientsMu.Unlock()
	for _, c := range clients {
		c.closeWS(websocket.StatusGoingAway, "shutdown")
	}
	if s.httpSrv == nil {
		return nil
	}
	return s.httpSrv.Shutdown(ctx)
}

func (s *Server) securityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !auth.IsLoopbackPeer(r.RemoteAddr) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		want := s.cfg.Addr()
		if !auth.ValidHostAuthority(r.Host, want) && !auth.ValidHostAuthority(r.Host, "localhost:"+portOnly(want)) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func portOnly(addr string) string {
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return ""
	}
	return port
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) eligibleAllowList() []auth.Origin {
	var out []auth.Origin
	for _, o := range s.cfg.OriginAllowList {
		if no, err := auth.NormalizeOrigin(o); err == nil {
			out = append(out, no)
		}
	}
	return out
}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	rawOrigin := r.Header.Get("Origin")
	origin, err := auth.NormalizeOrigin(rawOrigin)
	if err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if !auth.Eligible(origin, s.cfg.EligibleHTTPLoopback, s.eligibleAllowList()) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	s.mu.Lock()
	if s.conns >= s.cfg.MaxConnections {
		s.mu.Unlock()
		http.Error(w, "too many connections", http.StatusTooManyRequests)
		return
	}
	s.conns++
	s.mu.Unlock()

	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		Subprotocols:   []string{protocol.Subprotocol},
		OriginPatterns: []string{"*"},
	})
	if err != nil {
		s.mu.Lock()
		s.conns--
		s.mu.Unlock()
		return
	}
	if c.Subprotocol() != protocol.Subprotocol {
		_ = c.Close(websocket.StatusProtocolError, "subprotocol")
		s.mu.Lock()
		s.conns--
		s.mu.Unlock()
		return
	}

	cc := newClientConn(c, origin)
	s.registerClient(cc)

	defer func() {
		s.unregisterClient(cc)
		s.auth.CancelPendingByConnection(cc.id)
		if cc.cancelOps != nil {
			cc.cancelOps()
		}
		s.mu.Lock()
		s.conns--
		s.mu.Unlock()
		cc.closeWS(websocket.StatusNormalClosure, "")
	}()

	go cc.writeLoop()
	s.sendStatusSnapshot(cc, protocol.TypeStatusSnapshot)

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	c.SetReadLimit(int64(s.cfg.MaxFrameBytes))

	unauthDeadline := time.Now().Add(30 * time.Second)
	for {
		if cc.getToken() == "" && time.Now().After(unauthDeadline) {
			cc.closeWS(websocket.StatusPolicyViolation, "authentication timeout")
			return
		}
		_, data, err := c.Read(ctx)
		if err != nil {
			return
		}
		if cc.getToken() != "" {
			// Authenticated connections are no longer subject to the
			// unauthenticated lifetime bound.
			unauthDeadline = time.Now().Add(24 * time.Hour)
		}
		s.dispatch(ctx, cc, data)
	}
}

func newClientConn(c *websocket.Conn, origin auth.Origin) *clientConn {
	id := "conn-" + randomID()
	ctx, cancel := context.WithCancel(context.Background())
	return &clientConn{
		id:           id,
		ws:           c,
		origin:       origin,
		pending:      map[int64]context.CancelFunc{},
		outq:         make(chan []byte, 64),
		cancelOps:    cancel,
		cancelOpsCtx: ctx,
	}
}

func (s *Server) registerClient(cc *clientConn) {
	s.clientsMu.Lock()
	s.clients[cc.id] = cc
	s.clientsMu.Unlock()
}

func (s *Server) unregisterClient(cc *clientConn) {
	s.clientsMu.Lock()
	delete(s.clients, cc.id)
	s.clientsMu.Unlock()
}

// dispatch validates the envelope and runs the operation without blocking
// the receive loop for long-running work (approval wait, card read).
func (s *Server) dispatch(ctx context.Context, cc *clientConn, data []byte) {
	var req protocol.Request
	if err := json.Unmarshal(data, &req); err != nil {
		s.writeError(cc, 0, card.NewError(card.ErrInvalidRequest, "malformed message"))
		return
	}
	if err := protocol.ValidateRequest(req); err != nil {
		s.writeError(cc, req.ID, card.AsError(err))
		return
	}
	cc.writeMu.Lock()
	if req.ID <= cc.lastReqID {
		cc.writeMu.Unlock()
		s.writeError(cc, req.ID, card.NewError(card.ErrInvalidRequest, "request id must increase"))
		return
	}
	cc.lastReqID = req.ID
	cc.writeMu.Unlock()

	switch req.Type {
	case protocol.TypeSessionOpen:
		s.runAsync(ctx, cc, req, s.opSessionOpen)
	case protocol.TypeSessionRenew:
		s.runAsync(ctx, cc, req, s.opSessionRenew)
	case protocol.TypeSessionClose:
		s.opSessionClose(cc, req)
	case protocol.TypeBridgeStatus:
		s.runAsync(ctx, cc, req, s.opBridgeStatus)
	case protocol.TypeCardRead:
		s.runAsync(ctx, cc, req, s.opCardRead)
	case protocol.TypeRequestCancel:
		s.opCancel(cc, req)
	default:
		s.writeError(cc, req.ID, card.NewError(card.ErrInvalidRequest, "unknown operation"))
	}
}

func (s *Server) runAsync(ctx context.Context, cc *clientConn, req protocol.Request, fn func(context.Context, *clientConn, protocol.Request)) {
	opCtx, cancel := context.WithCancel(ctx)
	cc.pendingMu.Lock()
	cc.pending[req.ID] = cancel
	cc.pendingMu.Unlock()
	go func() {
		defer func() {
			cc.pendingMu.Lock()
			delete(cc.pending, req.ID)
			cc.pendingMu.Unlock()
			cancel()
		}()
		fn(opCtx, cc, req)
	}()
}

func (s *Server) opSessionOpen(ctx context.Context, cc *clientConn, req protocol.Request) {
	var p protocol.SessionOpenPayload
	if len(req.Payload) > 0 {
		if err := json.Unmarshal(req.Payload, &p); err != nil {
			s.writeError(cc, req.ID, card.NewError(card.ErrInvalidRequest, "bad payload"))
			return
		}
	}
	if len(p.Fields) == 0 {
		p.Fields = card.DefaultFields
	}
	fields, err := card.NormalizeFields(p.Fields)
	if err != nil {
		s.writeError(cc, req.ID, card.AsError(err))
		return
	}

	now := time.Now()
	sess, err := s.auth.OpenSession(cc.origin, fields, now)
	if err == nil {
		cc.setSession(sess)
		s.writeResult(cc, req.ID, sessionResult(sess))
		s.sendStatusSnapshot(cc, protocol.TypeStatusSnapshot)
		return
	}
	cerr := card.AsError(err)
	if cerr.Code != card.ErrApprovalRequired {
		s.writeError(cc, req.ID, cerr)
		return
	}

	pending, perr := s.auth.RequestApproval(cc.origin, fields, cc.id, time.Now())
	if perr != nil {
		s.writeError(cc, req.ID, card.AsError(perr).WithRequestID(req.ID))
		return
	}
	s.logApprovalPrompt(pending)
	if body, err := json.Marshal(protocol.ApprovalRequiredPayload{
		ApprovalID:     pending.ID,
		Fields:         pending.Fields,
		ComparisonCode: pending.Comparison,
		ExpiresAt:      pending.ExpiresAt.UTC().Format(time.RFC3339),
	}); err == nil {
		s.writeEventTo(cc, protocol.TypeApprovalRequired, body)
	}

	waitCtx := ctx
	if cc.cancelOpsCtx != nil {
		waitCtx = cc.cancelOpsCtx
	}
	allowed, werr := s.auth.Wait(waitCtx, pending)
	if werr != nil {
		s.writeError(cc, req.ID, card.AsError(werr).WithRequestID(req.ID))
		return
	}
	if !allowed {
		s.writeError(cc, req.ID, card.NewError(card.ErrApprovalDenied, "operator denied").WithRequestID(req.ID))
		return
	}
	sess, err = s.auth.OpenSession(cc.origin, fields, time.Now())
	if err != nil {
		s.writeError(cc, req.ID, card.AsError(err).WithRequestID(req.ID))
		return
	}
	cc.setSession(sess)
	s.writeResult(cc, req.ID, sessionResult(sess))
	s.sendStatusSnapshot(cc, protocol.TypeStatusSnapshot)
}

func sessionResult(sess *auth.Session) protocol.SessionOpenResult {
	return protocol.SessionOpenResult{
		Token:           sess.Token,
		ExpiresAt:       sess.ExpiresAt.UTC().Format(time.RFC3339),
		Fields:          sess.Fields,
		InstanceID:      sess.InstanceID,
		Protocol:        protocol.ProtocolInfo{Major: protocol.Major, Minor: 0},
		FieldsSupported: card.MVPFields,
	}
}

func (s *Server) logApprovalPrompt(p *auth.PendingRequest) {
	fmt.Printf("\n[approval] %s requests: %v\n", auth.EscapeTerminal(string(p.Origin)), p.Fields)
	fmt.Printf("Trust ends when OpenThaiID stops or after %s.\n", s.cfg.TrustDuration)
	fmt.Printf("Request: %d   Comparison code: %s\n\n", p.ID, p.Comparison)
	fmt.Printf("allow %d\n", p.ID)
	fmt.Printf("deny %d\n\n", p.ID)
}

func (s *Server) opSessionRenew(ctx context.Context, cc *clientConn, req protocol.Request) {
	sess, err := s.auth.Session(tokenFrom(cc, req), cc.origin, time.Now())
	if err != nil {
		s.writeError(cc, req.ID, card.AsError(err))
		return
	}
	fresh, err := s.auth.OpenSession(cc.origin, sess.Fields, time.Now())
	if err != nil {
		s.writeError(cc, req.ID, card.AsError(err))
		return
	}
	s.auth.CloseSession(sess.Token)
	cc.setSession(fresh)
	s.writeResult(cc, req.ID, sessionResult(fresh))
}

func (s *Server) opSessionClose(cc *clientConn, req protocol.Request) {
	tok := tokenFrom(cc, req)
	if tok != "" {
		s.auth.CloseSession(tok)
	}
	cc.clearSession()
	s.writeResult(cc, req.ID, map[string]bool{"closed": true})
}

func (s *Server) opBridgeStatus(ctx context.Context, cc *clientConn, req protocol.Request) {
	if _, err := s.requireAuth(cc, req); err != nil {
		s.writeError(cc, req.ID, card.AsError(err))
		return
	}
	s.writeResult(cc, req.ID, s.buildStatus())
}

func (s *Server) opCardRead(ctx context.Context, cc *clientConn, req protocol.Request) {
	sess, err := s.requireAuth(cc, req)
	if err != nil {
		s.writeError(cc, req.ID, card.AsError(err))
		return
	}
	var p protocol.CardReadPayload
	if err := json.Unmarshal(req.Payload, &p); err != nil {
		s.writeError(cc, req.ID, card.NewError(card.ErrInvalidRequest, "bad payload"))
		return
	}
	fields := p.Fields
	if len(fields) == 0 {
		fields = sess.Fields
	}
	fields, err = card.NormalizeFields(fields)
	if err != nil {
		s.writeError(cc, req.ID, card.AsError(err))
		return
	}
	if !card.Subset(fields, sess.Fields) {
		s.writeError(cc, req.ID, card.NewError(card.ErrFieldNotAllowed, "field not in grant"))
		return
	}

	select {
	case s.opSem <- struct{}{}:
	case <-ctx.Done():
		s.writeError(cc, req.ID, card.NewError(card.ErrCancelled, "read cancelled").WithRequestID(req.ID))
		return
	}
	defer func() { <-s.opSem }()

	readCtx, cancel := context.WithTimeout(ctx, s.cfg.ReadTimeout)
	cc.pendingMu.Lock()
	cc.pending[req.ID] = cancel
	cc.pendingMu.Unlock()
	defer func() {
		cancel()
		cc.pendingMu.Lock()
		delete(cc.pending, req.ID)
		cc.pendingMu.Unlock()
	}()

	res, err := s.readers.Read(readCtx, reader.ReadRequest{
		ReaderID:      p.ReaderID,
		CardSessionID: p.CardSessionID,
		Fields:        fields,
		Execute:       s.engine.Execute,
	})
	if err != nil {
		s.writeError(cc, req.ID, card.AsError(err).WithRequestID(req.ID))
		return
	}

	if _, err := s.auth.Session(cc.getToken(), cc.origin, time.Now()); err != nil {
		s.writeError(cc, req.ID, card.NewError(card.ErrAuthExpired, "authorization revoked").WithRequestID(req.ID))
		return
	}
	s.writeResult(cc, req.ID, res)
}

func (s *Server) opCancel(cc *clientConn, req protocol.Request) {
	var p protocol.CancelPayload
	if err := json.Unmarshal(req.Payload, &p); err != nil {
		s.writeError(cc, req.ID, card.NewError(card.ErrInvalidRequest, "bad payload"))
		return
	}
	cc.pendingMu.Lock()
	cancel, ok := cc.pending[p.RequestID]
	cc.pendingMu.Unlock()
	if ok {
		cancel()
	}
	s.writeResult(cc, req.ID, map[string]bool{"cancelled": ok})
}

func (s *Server) requireAuth(cc *clientConn, req protocol.Request) (*auth.Session, error) {
	tok := tokenFrom(cc, req)
	if tok == "" {
		return nil, card.NewError(card.ErrAuthExpired, "not authenticated")
	}
	sess, err := s.auth.Session(tok, cc.origin, time.Now())
	if err != nil {
		return nil, card.AsError(err)
	}
	cc.setSession(sess)
	return sess, nil
}

func tokenFrom(cc *clientConn, req protocol.Request) string {
	if req.Token != "" {
		return req.Token
	}
	return cc.getToken()
}

func (s *Server) buildStatus() protocol.StatusSnapshot {
	snap := s.readers.Snapshot()
	out := protocol.StatusSnapshot{
		Connection: "connected",
		Backend:    snap.Backend,
		Readers:    make([]protocol.ReaderView, 0, len(snap.Readers)),
	}
	for _, r := range snap.Readers {
		out.Readers = append(out.Readers, protocol.ReaderView{
			ID:            r.ID,
			Name:          r.Name,
			Presence:      string(r.Presence),
			CardSessionID: r.CardSessionID,
		})
	}
	return out
}

func (s *Server) sendStatusSnapshot(cc *clientConn, t protocol.Type) {
	payload, err := json.Marshal(s.buildStatus())
	if err != nil {
		return
	}
	s.writeEventTo(cc, t, payload)
}

func (s *Server) forwardReaderEvents(ctx context.Context) {
	ch := s.readers.Subscribe()
	defer s.readers.Unsubscribe(ch)
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-ch:
			if !ok {
				return
			}
			var t protocol.Type
			var payload any
			switch ev.Kind {
			case "reader.connected":
				t = protocol.TypeReaderConnected
				payload = map[string]string{"readerId": ev.ReaderID, "name": ev.Name}
			case "reader.disconnected":
				t = protocol.TypeReaderDisconnected
				payload = map[string]string{"readerId": ev.ReaderID}
			case "card.inserted":
				t = protocol.TypeCardInserted
				payload = map[string]string{"readerId": ev.ReaderID, "cardSessionId": ev.CardSessionID}
			case "card.removed":
				t = protocol.TypeCardRemoved
				payload = map[string]string{"readerId": ev.ReaderID, "cardSessionId": ev.CardSessionID}
			default:
				continue
			}
			s.broadcastEvent(t, payload)
		}
	}
}

// handleAuthRevoked delivers authorization.revoked then closes affected clients.
func (s *Server) handleAuthRevoked(origin auth.Origin, tokens []string) {
	s.clientsMu.Lock()
	targets := make([]*clientConn, 0)
	for _, cc := range s.clients {
		tok := cc.getToken()
		if tok == "" {
			continue
		}
		for _, t := range tokens {
			if tok == t {
				targets = append(targets, cc)
				break
			}
		}
	}
	s.clientsMu.Unlock()
	for _, cc := range targets {
		body, _ := json.Marshal(map[string]string{"reason": "origin revoked"})
		s.writeEventTo(cc, protocol.TypeAuthRevoked, body)
		cc.clearSession()
		if cc.cancelOps != nil {
			cc.cancelOps()
		}
		cc.closeWS(websocket.StatusPolicyViolation, "authorization revoked")
	}
}

func (s *Server) broadcastEvent(t protocol.Type, payload any) {
	body, err := json.Marshal(payload)
	if err != nil {
		return
	}
	s.clientsMu.Lock()
	targets := make([]*clientConn, 0, len(s.clients))
	for _, cc := range s.clients {
		if !cc.hasSession() {
			continue
		}
		targets = append(targets, cc)
	}
	s.clientsMu.Unlock()

	now := time.Now()
	for _, cc := range targets {
		if _, err := s.auth.Session(cc.getToken(), cc.origin, now); err != nil {
			cc.clearSession()
			cc.closeWS(websocket.StatusPolicyViolation, "authorization expired")
			continue
		}
		s.writeEventTo(cc, t, body)
	}
}

func (s *Server) writeEventTo(cc *clientConn, t protocol.Type, payload json.RawMessage) {
	cc.writeMu.Lock()
	cc.eventSeq++
	seq := cc.eventSeq
	cc.writeMu.Unlock()

	raw, err := json.Marshal(protocol.Event{
		V:          protocol.Major,
		Type:       t,
		InstanceID: s.InstanceID(),
		Sequence:   seq,
		Payload:    payload,
	})
	if err != nil {
		return
	}
	cc.enqueue(raw)
}

func (s *Server) writeResult(cc *clientConn, id int64, result any) {
	raw, err := json.Marshal(result)
	if err != nil {
		s.writeError(cc, id, card.NewError(card.ErrConnectionFailed, "encode failed"))
		return
	}
	body, err := json.Marshal(protocol.Response{
		V:      protocol.Major,
		ID:     id,
		Result: raw,
	})
	if err != nil {
		return
	}
	cc.enqueue(body)
}

func (s *Server) writeError(cc *clientConn, id int64, e *card.Error) {
	if e == nil {
		e = card.NewError(card.ErrConnectionFailed, "unknown error")
	}
	body, err := json.Marshal(protocol.Response{
		V:     protocol.Major,
		ID:    id,
		Error: e,
	})
	if err != nil {
		return
	}
	cc.enqueue(body)
}

// enqueue queues a frame. If the client is too slow, the connection is marked
// dropped and closed — other clients are never delayed.
func (cc *clientConn) enqueue(body []byte) {
	cc.writeMu.Lock()
	if cc.closed || cc.dropped {
		cc.writeMu.Unlock()
		return
	}
	select {
	case cc.outq <- body:
		cc.writeMu.Unlock()
	default:
		cc.dropped = true
		cc.writeMu.Unlock()
		go cc.closeWS(websocket.StatusPolicyViolation, "client too slow")
	}
}

func (cc *clientConn) writeLoop() {
	for body := range cc.outq {
		cc.writeMu.Lock()
		if cc.closed {
			cc.writeMu.Unlock()
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := cc.ws.Write(ctx, websocket.MessageText, body)
		cancel()
		if err != nil {
			cc.closed = true
			cc.writeMu.Unlock()
			_ = cc.ws.Close(websocket.StatusInternalError, "write failed")
			return
		}
		cc.writeMu.Unlock()
	}
}

func (cc *clientConn) closeWS(code websocket.StatusCode, reason string) {
	cc.writeMu.Lock()
	if cc.closed {
		cc.writeMu.Unlock()
		return
	}
	cc.closed = true
	cc.writeMu.Unlock()
	_ = cc.ws.Close(code, reason)
}

func randomID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "unknown"
	}
	return hex.EncodeToString(b[:])
}

// NewInstanceID creates a random bridge instance id.
func NewInstanceID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "bridge-unknown"
	}
	return "bridge-" + hex.EncodeToString(b[:])
}
