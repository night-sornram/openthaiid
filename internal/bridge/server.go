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
	"sync/atomic"
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

	seq atomic.Uint64

	// cancelRead is set while a card.read is in flight (single-read MVP).
	cancelReadMu sync.Mutex
	cancelRead   map[string]context.CancelFunc // key: token+requestID

	eventSubsMu sync.Mutex
	eventSubs   map[*clientConn]struct{}
}

type clientConn struct {
	ws      *websocket.Conn
	origin  auth.Origin
	token   string
	sess    *auth.Session
	writeMu sync.Mutex
	closed  bool
}

// New constructs a bridge server.
func New(cfg config.Config, log *slog.Logger, authMgr *auth.Manager, readers *reader.Manager) *Server {
	return &Server{
		cfg:        cfg,
		log:        log,
		auth:       authMgr,
		readers:    readers,
		engine:     card.NewEngine(card.DefaultProfile()),
		cancelRead: map[string]context.CancelFunc{},
		eventSubs:  map[*clientConn]struct{}{},
	}
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
	// Forward reader events to connected clients.
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

// Shutdown stops the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error {
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
	for _, s := range s.cfg.OriginAllowList {
		if o, err := auth.NormalizeOrigin(s); err == nil {
			out = append(out, o)
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
		Subprotocols: []string{protocol.Subprotocol},
		// Origin already validated above against the exact configured policy.
		// Disable coder/websocket's default same-origin check so we own it.
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

	cc := &clientConn{ws: c, origin: origin}
	s.eventSubsMu.Lock()
	s.eventSubs[cc] = struct{}{}
	s.eventSubsMu.Unlock()

	defer func() {
		s.eventSubsMu.Lock()
		delete(s.eventSubs, cc)
		s.eventSubsMu.Unlock()
		s.mu.Lock()
		s.conns--
		s.mu.Unlock()
		_ = c.Close(websocket.StatusNormalClosure, "")
	}()

	ctx := r.Context()
	c.SetReadLimit(int64(s.cfg.MaxFrameBytes))
	for {
		_, data, err := c.Read(ctx)
		if err != nil {
			return
		}
		s.handleMessage(ctx, cc, data)
	}
}

func (s *Server) handleMessage(ctx context.Context, cc *clientConn, data []byte) {
	var req protocol.Request
	if err := json.Unmarshal(data, &req); err != nil {
		s.writeError(cc, 0, card.NewError(card.ErrInvalidRequest, "malformed message"))
		return
	}
	if err := protocol.ValidateRequest(req); err != nil {
		s.writeError(cc, req.ID, card.AsError(err))
		return
	}

	switch req.Type {
	case protocol.TypeSessionOpen:
		s.opSessionOpen(ctx, cc, req)
	case protocol.TypeSessionRenew:
		s.opSessionRenew(ctx, cc, req)
	case protocol.TypeSessionClose:
		s.opSessionClose(cc, req)
	case protocol.TypeBridgeStatus:
		s.opBridgeStatus(ctx, cc, req)
	case protocol.TypeCardRead:
		s.opCardRead(ctx, cc, req)
	case protocol.TypeRequestCancel:
		s.opCancel(ctx, cc, req)
	default:
		s.writeError(cc, req.ID, card.NewError(card.ErrInvalidRequest, "unknown operation"))
	}
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
		cc.token = sess.Token
		cc.sess = sess
		s.writeResult(cc, req.ID, protocol.SessionOpenResult{
			Token:           sess.Token,
			ExpiresAt:       sess.ExpiresAt.UTC().Format(time.RFC3339),
			Fields:          sess.Fields,
			InstanceID:      sess.InstanceID,
			Protocol:        protocol.ProtocolInfo{Major: protocol.Major, Minor: 0},
			FieldsSupported: card.MVPFields,
		})
		s.sendStatusSnapshot(cc, protocol.TypeStatusSnapshot)
		return
	}
	cerr := card.AsError(err)
	if cerr.Code != card.ErrApprovalRequired {
		s.writeError(cc, req.ID, cerr)
		return
	}

	// Pairing-only: create a pending approval and wait for the operator.
	// One request produces exactly one response.
	pending, perr := s.auth.RequestApproval(cc.origin, fields, now)
	if perr != nil {
		s.writeError(cc, req.ID, card.AsError(perr).WithRequestID(req.ID))
		return
	}
	s.logApprovalPrompt(pending)
	// Tell only this connection about its own pending approval.
	// Pairing-only clients still receive this (it is about them).
	if body, err := json.Marshal(protocol.ApprovalRequiredPayload{
		ApprovalID:     pending.ID,
		Fields:         pending.Fields,
		ComparisonCode: pending.Comparison,
		ExpiresAt:      pending.ExpiresAt.UTC().Format(time.RFC3339),
	}); err == nil {
		s.writeEventTo(cc, protocol.TypeApprovalRequired, body)
	}

	allowed, werr := s.auth.Wait(pending, time.Now())
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
	cc.token = sess.Token
	cc.sess = sess
	s.writeResult(cc, req.ID, protocol.SessionOpenResult{
		Token:           sess.Token,
		ExpiresAt:       sess.ExpiresAt.UTC().Format(time.RFC3339),
		Fields:          sess.Fields,
		InstanceID:      sess.InstanceID,
		Protocol:        protocol.ProtocolInfo{Major: protocol.Major, Minor: 0},
		FieldsSupported: card.MVPFields,
	})
	s.sendStatusSnapshot(cc, protocol.TypeStatusSnapshot)
}

func (s *Server) logApprovalPrompt(p *auth.PendingRequest) {
	// Terminal-facing only; not a persistent structured log of card data.
	fmt.Printf("\n[approval] %s requests: %v\n", p.Origin, p.Fields)
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
	cc.token = fresh.Token
	cc.sess = fresh
	s.writeResult(cc, req.ID, protocol.SessionOpenResult{
		Token:           fresh.Token,
		ExpiresAt:       fresh.ExpiresAt.UTC().Format(time.RFC3339),
		Fields:          fresh.Fields,
		InstanceID:      fresh.InstanceID,
		Protocol:        protocol.ProtocolInfo{Major: protocol.Major, Minor: 0},
		FieldsSupported: card.MVPFields,
	})
}

func (s *Server) opSessionClose(cc *clientConn, req protocol.Request) {
	tok := tokenFrom(cc, req)
	if tok != "" {
		s.auth.CloseSession(tok)
	}
	cc.token = ""
	cc.sess = nil
	s.writeResult(cc, req.ID, map[string]bool{"closed": true})
}

func (s *Server) opBridgeStatus(ctx context.Context, cc *clientConn, req protocol.Request) {
	if _, err := s.requireAuth(cc, req); err != nil {
		s.writeError(cc, req.ID, card.AsError(err))
		return
	}
	snap := s.buildStatus()
	s.writeResult(cc, req.ID, snap)
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

	readCtx, cancel := context.WithTimeout(ctx, s.cfg.ReadTimeout)
	key := fmt.Sprintf("%s:%d", cc.token, req.ID)
	s.cancelReadMu.Lock()
	s.cancelRead[key] = cancel
	s.cancelReadMu.Unlock()
	defer func() {
		cancel()
		s.cancelReadMu.Lock()
		delete(s.cancelRead, key)
		s.cancelReadMu.Unlock()
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

	// Recheck authorization before delivery.
	if _, err := s.auth.Session(cc.token, cc.origin, time.Now()); err != nil {
		s.writeError(cc, req.ID, card.NewError(card.ErrAuthExpired, "authorization revoked").WithRequestID(req.ID))
		return
	}
	// Unicast result to this connection only.
	s.writeResult(cc, req.ID, res)
}

func (s *Server) opCancel(ctx context.Context, cc *clientConn, req protocol.Request) {
	var p protocol.CancelPayload
	if err := json.Unmarshal(req.Payload, &p); err != nil {
		s.writeError(cc, req.ID, card.NewError(card.ErrInvalidRequest, "bad payload"))
		return
	}
	key := fmt.Sprintf("%s:%d", cc.token, p.RequestID)
	s.cancelReadMu.Lock()
	cancel, ok := s.cancelRead[key]
	s.cancelReadMu.Unlock()
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
	cc.token = tok
	cc.sess = sess
	return sess, nil
}

func tokenFrom(cc *clientConn, req protocol.Request) string {
	if req.Token != "" {
		return req.Token
	}
	return cc.token
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
			s.broadcastEvent(t, payload, nil)
		}
	}
}

func (s *Server) nextSeq() uint64 { return s.seq.Add(1) }

func (s *Server) broadcastEvent(t protocol.Type, payload any, only *clientConn) {
	body, err := json.Marshal(payload)
	if err != nil {
		return
	}
	s.eventSubsMu.Lock()
	targets := make([]*clientConn, 0, len(s.eventSubs))
	for cc := range s.eventSubs {
		// Unauthenticated / pairing-only clients receive no device events.
		if cc.token == "" || cc.sess == nil {
			continue
		}
		if only != nil && cc != only {
			continue
		}
		targets = append(targets, cc)
	}
	s.eventSubsMu.Unlock()

	for _, cc := range targets {
		s.writeEventTo(cc, t, body)
	}
}

func (s *Server) writeEventTo(cc *clientConn, t protocol.Type, payload json.RawMessage) {
	raw, err := json.Marshal(protocol.Event{
		V:          protocol.Major,
		Type:       t,
		InstanceID: s.InstanceID(),
		Sequence:   s.nextSeq(),
		Payload:    payload,
	})
	if err != nil {
		return
	}
	_ = s.writeRaw(cc, raw)
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
	_ = s.writeRaw(cc, body)
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
	_ = s.writeRaw(cc, body)
}

func (s *Server) writeRaw(cc *clientConn, body []byte) error {
	cc.writeMu.Lock()
	defer cc.writeMu.Unlock()
	if cc.closed {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return cc.ws.Write(ctx, websocket.MessageText, body)
}

// NewInstanceID creates a random bridge instance id.
func NewInstanceID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "bridge-unknown"
	}
	return "bridge-" + hex.EncodeToString(b[:])
}
