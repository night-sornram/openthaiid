// Package reader coordinates inventory, card generations, and reads.
package reader

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	"github.com/night-sornram/openthaiid/internal/card"
	"github.com/night-sornram/openthaiid/internal/pcsc"
)

// StallGrace is how long after cancellation/timeout a blocked native call
// may remain outstanding before the backend is marked stalled.
const StallGrace = 2 * time.Second

// Presence mirrors pcsc.CardPresence in the public reader view.
type Presence = pcsc.CardPresence

// ReaderState is the public view of one reader.
type ReaderState struct {
	ID       string            `json:"readerId"`
	Name     string            `json:"name"`
	Presence pcsc.CardPresence `json:"presence"`
	// CardSessionID is set when Presence == present.
	CardSessionID string `json:"cardSessionId,omitempty"`
}

// Snapshot is a status view for authenticated clients.
type Snapshot struct {
	Readers []ReaderState `json:"readers"`
	// Backend: available | unavailable | stalled
	Backend string `json:"backend"`
	// BackendName is "pcsc" or "synthetic".
	BackendName string `json:"backendName,omitempty"`
	// PreferredReaderID is the operator-selected reader, if any.
	PreferredReaderID string `json:"preferredReaderId,omitempty"`
}

// ChangeEvent is emitted on inventory transitions.
type ChangeEvent struct {
	Kind          string // reader.connected | reader.disconnected | card.inserted | card.removed
	ReaderID      string
	Name          string
	CardSessionID string
}

// Manager owns the monitoring loop and the single active read slot.
type Manager struct {
	mu sync.RWMutex

	backend       pcsc.Backend
	readerBackend pcsc.Backend

	readers       map[string]*ReaderState
	backendStatus string
	preferredID   string

	reading bool
	stalled bool

	// activeWorker is non-nil while a native read worker owns the slot.
	activeWorker *readWorker
	// idle is closed when no native worker holds the slot.
	idle chan struct{}

	// eventOverflow is set when a subscriber dropped a change event.
	eventOverflow bool

	// resyncMu serializes inventory rebuilds.
	resyncMu sync.Mutex

	subscribers map[chan ChangeEvent]struct{}

	// monitorDone is closed when the monitoring goroutine exits.
	monitorDone    chan struct{}
	monitorStop    context.CancelFunc
	monitorStarted bool
}

// readWorker owns open → begin → execute → end → close for one read.
// The caller may abandon waiting; the worker always finishes cleanup.
type readWorker struct {
	done   chan struct{}
	result *card.ReadResult
	err    error
	// cleanups counts EndTransaction+Close pairs (exactly-once cleanup).
	cleanups int
}

// NewManager builds a manager around two backend instances (same factory).
func NewManager(monitor, reader pcsc.Backend) *Manager {
	idle := make(chan struct{})
	close(idle)
	return &Manager{
		backend:       monitor,
		readerBackend: reader,
		readers:       map[string]*ReaderState{},
		backendStatus: "unavailable",
		subscribers:   map[chan ChangeEvent]struct{}{},
		monitorDone:   make(chan struct{}),
		idle:          idle,
	}
}

// Start establishes both backends and begins the monitoring loop.
// A PC/SC failure at startup is non-fatal: the monitor loop retries recovery.
func (m *Manager) Start(ctx context.Context) error {
	mctx, cancel := context.WithCancel(context.Background())
	m.mu.Lock()
	if m.monitorStarted {
		m.mu.Unlock()
		cancel()
		return nil
	}
	m.monitorStop = cancel
	m.monitorStarted = true
	m.mu.Unlock()

	if err := m.backend.Establish(ctx); err != nil {
		m.mu.Lock()
		m.backendStatus = "unavailable"
		m.mu.Unlock()
		go m.monitorLoop(mctx)
		return nil
	}
	if err := m.readerBackend.Establish(ctx); err != nil {
		_ = m.backend.Release(ctx)
		m.mu.Lock()
		m.backendStatus = "unavailable"
		m.mu.Unlock()
		go m.monitorLoop(mctx)
		return nil
	}
	if err := m.resync(ctx); err == nil {
		m.mu.Lock()
		m.backendStatus = "available"
		m.mu.Unlock()
	}
	go m.monitorLoop(mctx)
	return nil
}

// Stop joins the monitoring loop and releases backends.
// If a native worker still holds the read slot when ctx expires, Stop returns
// without releasing a PC/SC context that is still in use.
func (m *Manager) Stop(ctx context.Context) error {
	m.mu.Lock()
	stop := m.monitorStop
	started := m.monitorStarted
	done := m.monitorDone
	idle := m.idle
	m.mu.Unlock()
	if stop != nil {
		stop()
	}
	_ = m.backend.CancelWait()
	if started {
		select {
		case <-done:
		case <-ctx.Done():
		}
	}
	// Never release a context still owned by a blocked native worker.
	select {
	case <-idle:
	case <-ctx.Done():
		return ctx.Err()
	}
	_ = m.backend.Release(ctx)
	return m.readerBackend.Release(ctx)
}

// Subscribe returns a buffered change channel. Caller must Unsubscribe.
func (m *Manager) Subscribe() chan ChangeEvent {
	ch := make(chan ChangeEvent, 32)
	m.mu.Lock()
	m.subscribers[ch] = struct{}{}
	m.mu.Unlock()
	return ch
}

// Unsubscribe removes the channel.
func (m *Manager) Unsubscribe(ch chan ChangeEvent) {
	m.mu.Lock()
	delete(m.subscribers, ch)
	m.mu.Unlock()
}

// ConsumeEventOverflow reports and clears a dropped-event condition.
func (m *Manager) ConsumeEventOverflow() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	v := m.eventOverflow
	m.eventOverflow = false
	return v
}

func (m *Manager) emit(ev ChangeEvent) {
	m.mu.RLock()
	subs := make([]chan ChangeEvent, 0, len(m.subscribers))
	for ch := range m.subscribers {
		subs = append(subs, ch)
	}
	m.mu.RUnlock()

	overflow := false
	for _, ch := range subs {
		select {
		case ch <- ev:
		default:
			overflow = true
		}
	}
	if overflow {
		m.mu.Lock()
		m.eventOverflow = true
		m.mu.Unlock()
		// The monitor loop's next resync rebuilds inventory; subscribers
		// that overflowed are closed by the bridge with WS 1013.
	}
}

// Snapshot returns the current reader view under synchronization.
func (m *Manager) Snapshot() Snapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]ReaderState, 0, len(m.readers))
	for _, r := range m.readers {
		out = append(out, *r)
	}
	name := ""
	if m.backend != nil {
		name = m.backend.Name()
	}
	return Snapshot{
		Readers:           out,
		Backend:           m.backendStatus,
		BackendName:       name,
		PreferredReaderID: m.preferredID,
	}
}

// BackendName identifies the native adapter ("pcsc" or "synthetic").
func (m *Manager) BackendName() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.backend == nil {
		return ""
	}
	return m.backend.Name()
}

// Refresh rebuilds inventory from the monitoring backend.
func (m *Manager) Refresh(ctx context.Context) error {
	return m.resync(ctx)
}

// Reconnect releases idle PC/SC contexts and tries to establish them again.
// Used by technician tools when the card service looks stuck. A read that
// still owns a context is left alone (tryReleaseBackends is a no-op then).
func (m *Manager) Reconnect(ctx context.Context) error {
	m.tryReleaseBackends(ctx)
	if !m.recoverBackend(ctx) {
		return card.NewError(card.ErrPCSCUnavailable, "could not reconnect to the card service")
	}
	return nil
}

// SetPreferredReader records which reader the operator selected.
// An empty id clears the selection (auto when exactly one card is present).
func (m *Manager) SetPreferredReader(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if id == "" {
		m.preferredID = ""
		return nil
	}
	if _, ok := m.readers[id]; !ok {
		return card.NewError(card.ErrNoReader, "reader not found")
	}
	m.preferredID = id
	return nil
}

// PreferredReader returns the operator-selected reader id, if any.
func (m *Manager) PreferredReader() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.preferredID
}

func (m *Manager) monitorLoop(ctx context.Context) {
	defer close(m.monitorDone)
	for ctx.Err() == nil {
		m.mu.RLock()
		stalled := m.stalled
		m.mu.RUnlock()
		if stalled || m.backend.Supported() != nil {
			// Stalled or backend not established: try recovery.
			if !m.recoverBackend(ctx) {
				select {
				case <-ctx.Done():
					return
				case <-time.After(2 * time.Second):
				}
				continue
			}
		}
		err := m.backend.WaitForChange(ctx, time.Second)
		if ctx.Err() != nil {
			return
		}
		if err != nil && err != context.DeadlineExceeded {
			m.mu.Lock()
			if !m.stalled {
				m.backendStatus = "unavailable"
			}
			m.mu.Unlock()
			m.invalidateAllSessions()
			select {
			case <-ctx.Done():
				return
			case <-time.After(2 * time.Second):
			}
			m.tryReleaseBackends(ctx)
			if e2 := m.backend.Establish(ctx); e2 == nil {
				_ = m.readerBackend.Establish(ctx)
				m.mu.Lock()
				if !m.stalled {
					m.backendStatus = "available"
				}
				m.mu.Unlock()
			}
		}
		_ = m.resync(ctx)
	}
}

func (m *Manager) recoverBackend(ctx context.Context) bool {
	m.mu.RLock()
	stalled := m.stalled
	idle := m.idle
	m.mu.RUnlock()
	if stalled {
		select {
		case <-idle:
		case <-ctx.Done():
			return false
		case <-time.After(50 * time.Millisecond):
			return false
		}
	}
	if err := m.backend.Establish(ctx); err != nil {
		return false
	}
	if err := m.readerBackend.Establish(ctx); err != nil {
		_ = m.backend.Release(ctx)
		return false
	}
	m.mu.Lock()
	if m.activeWorker == nil {
		m.backendStatus = "available"
		m.stalled = false
	}
	m.mu.Unlock()
	_ = m.resync(ctx)
	return true
}

// tryReleaseBackends releases only when no native worker holds the slot.
func (m *Manager) tryReleaseBackends(ctx context.Context) {
	m.mu.RLock()
	idle := m.idle
	m.mu.RUnlock()
	select {
	case <-idle:
		_ = m.backend.Release(ctx)
		_ = m.readerBackend.Release(ctx)
	default:
		// A worker still owns a PC/SC context; do not release it.
	}
}

// invalidateAllSessions clears card generations (reader loss / service failure).
func (m *Manager) invalidateAllSessions() {
	m.mu.Lock()
	prev := m.readers
	next := make(map[string]*ReaderState, len(m.readers))
	for id, r := range m.readers {
		cp := *r
		if cp.Presence == pcsc.PresencePresent {
			cp.Presence = pcsc.PresenceAbsent
		}
		cp.CardSessionID = ""
		next[id] = &cp
	}
	m.readers = next
	m.mu.Unlock()

	for id, old := range prev {
		if old.CardSessionID != "" {
			m.emit(ChangeEvent{Kind: "card.removed", ReaderID: id, CardSessionID: old.CardSessionID})
		}
	}
}

// resync diffs the backend inventory and emits events.
func (m *Manager) resync(ctx context.Context) error {
	m.resyncMu.Lock()
	defer m.resyncMu.Unlock()
	if err := m.backend.Supported(); err != nil {
		m.mu.Lock()
		if !m.stalled && m.backendStatus != "stalled" {
			m.backendStatus = "unavailable"
		}
		m.mu.Unlock()
		return err
	}
	list, err := m.backend.Readers(ctx)
	if err != nil {
		return err
	}
	next := map[string]*ReaderState{}
	for _, r := range list {
		present := m.cardPresent(ctx, r.ID)
		st := &ReaderState{
			ID:       r.ID,
			Name:     r.Name,
			Presence: pcsc.PresenceAbsent,
		}
		if present {
			st.Presence = pcsc.PresencePresent
		}
		next[r.ID] = st
	}

	m.mu.Lock()
	prev := m.readers
	// Copy previous session ids into next before publishing so Snapshot
	// never observes a half-built ReaderState.
	for id, st := range next {
		if old, ok := prev[id]; ok && old.Presence == pcsc.PresencePresent && st.Presence == pcsc.PresencePresent {
			st.CardSessionID = old.CardSessionID
		}
	}
	var events []ChangeEvent
	for id, st := range next {
		old, ok := prev[id]
		if !ok {
			events = append(events, ChangeEvent{Kind: "reader.connected", ReaderID: id, Name: st.Name})
			if st.Presence == pcsc.PresencePresent {
				sid := newCardSessionID()
				st.CardSessionID = sid
				events = append(events, ChangeEvent{Kind: "card.inserted", ReaderID: id, CardSessionID: sid})
			}
			continue
		}
		wasPresent := old.Presence == pcsc.PresencePresent
		nowPresent := st.Presence == pcsc.PresencePresent
		if !wasPresent && nowPresent {
			sid := newCardSessionID()
			st.CardSessionID = sid
			events = append(events, ChangeEvent{Kind: "card.inserted", ReaderID: id, CardSessionID: sid})
		} else if wasPresent && !nowPresent {
			events = append(events, ChangeEvent{Kind: "card.removed", ReaderID: id, CardSessionID: old.CardSessionID})
			st.CardSessionID = ""
		}
	}
	for id, old := range prev {
		if _, ok := next[id]; !ok {
			if old.Presence == pcsc.PresencePresent {
				events = append(events, ChangeEvent{Kind: "card.removed", ReaderID: id, CardSessionID: old.CardSessionID})
			}
			events = append(events, ChangeEvent{Kind: "reader.disconnected", ReaderID: id, Name: old.Name})
		}
	}
	m.readers = next
	if m.preferredID != "" {
		if _, ok := next[m.preferredID]; !ok {
			m.preferredID = ""
		}
	}
	m.mu.Unlock()

	for _, ev := range events {
		m.emit(ev)
	}
	return nil
}

func (m *Manager) cardPresent(ctx context.Context, readerID string) bool {
	ok, err := m.backend.CardPresent(ctx, readerID)
	if err != nil {
		return false
	}
	return ok
}

// CardSession returns the current session id for a reader, if any.
func (m *Manager) CardSession(readerID string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	st, ok := m.readers[readerID]
	if !ok || st.Presence != pcsc.PresencePresent || st.CardSessionID == "" {
		return "", false
	}
	return st.CardSessionID, true
}

// WaitCardSession blocks until readerID has a card session or deadline.
func (m *Manager) WaitCardSession(readerID string, deadline time.Time) (string, bool) {
	for time.Now().Before(deadline) {
		if sid, ok := m.CardSession(readerID); ok {
			return sid, true
		}
		time.Sleep(5 * time.Millisecond)
	}
	return m.CardSession(readerID)
}

// SelectReader picks a reader when exactly one exists, or the preferred
// reader when the operator has selected one in the local manager.
func (m *Manager) SelectReader(requested string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if requested != "" {
		st, ok := m.readers[requested]
		if !ok {
			return "", card.NewError(card.ErrNoReader, "reader not found")
		}
		if st.Presence != pcsc.PresencePresent {
			return "", card.NewError(card.ErrNoCard, "no card")
		}
		return requested, nil
	}
	if m.preferredID != "" {
		st, ok := m.readers[m.preferredID]
		if !ok {
			return "", card.NewError(card.ErrNoReader, "selected reader not found")
		}
		if st.Presence != pcsc.PresencePresent {
			return "", card.NewError(card.ErrNoCard, "no card")
		}
		return m.preferredID, nil
	}
	var only string
	n := 0
	for id, st := range m.readers {
		if st.Presence == pcsc.PresencePresent {
			only = id
			n++
		}
	}
	if n == 0 {
		return "", card.NewError(card.ErrNoReader, "no reader with a card")
	}
	if n > 1 {
		return "", card.NewError(card.ErrReaderSelectionRequired, "multiple readers")
	}
	return only, nil
}

// ReadRequest is a coordinated single-field-set read.
type ReadRequest struct {
	ReaderID      string
	CardSessionID string
	Fields        []card.Field
	// Execute runs the profile read on an open connection.
	Execute func(ctx context.Context, conn card.Transmitter, fields []card.Field) (*card.FieldValues, []card.Warning, error)
}

// Read performs a single active read against the reader backend.
//
// One native worker owns open → transaction → transmit → end → close.
// Cancellation and timeout return promptly (CANCELLED / READ_TIMEOUT) without
// waiting for blocked native cleanup. The global read slot stays occupied
// until that worker finishes; late results are discarded. Overlapping reads
// are rejected with BUSY. If native work is still blocked two seconds after
// cancellation, the backend is marked stalled until the worker finishes and
// recovery completes.
func (m *Manager) Read(ctx context.Context, req ReadRequest) (*card.ReadResult, error) {
	fields, err := card.NormalizeFields(req.Fields)
	if err != nil {
		return nil, err
	}

	m.mu.RLock()
	stalled := m.stalled
	m.mu.RUnlock()
	if stalled {
		return nil, card.NewError(card.ErrBackendStalled, "backend stalled")
	}

	readerID, err := m.SelectReader(req.ReaderID)
	if err != nil {
		return nil, err
	}
	sid, ok := m.CardSession(readerID)
	if !ok {
		return nil, card.NewError(card.ErrNoCard, "no card")
	}
	if req.CardSessionID != "" && req.CardSessionID != sid {
		return nil, card.NewError(card.ErrCardRemoved, "card session changed")
	}

	m.mu.Lock()
	if m.reading {
		m.mu.Unlock()
		return nil, card.NewError(card.ErrBusy, "another read is in progress")
	}
	if m.stalled {
		m.mu.Unlock()
		return nil, card.NewError(card.ErrBackendStalled, "backend stalled")
	}
	w := &readWorker{done: make(chan struct{})}
	m.reading = true
	m.activeWorker = w
	m.idle = make(chan struct{})
	m.mu.Unlock()

	// Worker owns the entire native lifecycle. It is not cancelled by ctx.
	go m.runNativeWorker(w, readerID, sid, fields, req.Execute)

	select {
	case <-w.done:
		m.finishWorker(w)
		return w.result, w.err
	case <-ctx.Done():
		// Prompt client-visible failure. Do not wait for blocked native cleanup.
		// After StallGrace, if the worker is still outstanding, mark the
		// backend stalled and keep ownership until the worker finishes.
		err := cancelError(ctx)
		go func() {
			stall := time.NewTimer(StallGrace)
			defer stall.Stop()
			select {
			case <-w.done:
				m.finishWorker(w)
			case <-stall.C:
				m.markStalled()
				<-w.done
				m.finishWorker(w)
			}
		}()
		return nil, err
	}
}

func cancelError(ctx context.Context) *card.Error {
	if ctx.Err() == context.DeadlineExceeded {
		return card.NewError(card.ErrReadTimeout, "read timed out")
	}
	return card.NewError(card.ErrCancelled, "read cancelled")
}

func (m *Manager) finishWorker(w *readWorker) {
	m.mu.Lock()
	if m.activeWorker == w {
		m.reading = false
		m.activeWorker = nil
		close(m.idle)
		// A finished worker unblocks stalled recovery.
		if m.stalled {
			m.backendStatus = "stalled"
		}
	}
	m.mu.Unlock()
}

func (m *Manager) markStalled() {
	m.mu.Lock()
	m.stalled = true
	m.backendStatus = "stalled"
	m.mu.Unlock()
}

// CleanupCount reports completed native cleanup operations for tests.
func (m *Manager) CleanupCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.activeWorker == nil {
		return 0
	}
	return m.activeWorker.cleanups
}

// runNativeWorker performs the full native read under a single owner.
// The worker context is intentionally not the caller's ctx: native PC/SC
// calls cannot be interrupted, so ownership (and cleanup) must outlive
// client cancellation.
func (m *Manager) runNativeWorker(w *readWorker, readerID, sid string, fields []card.Field, execute func(context.Context, card.Transmitter, []card.Field) (*card.FieldValues, []card.Warning, error)) {
	defer close(w.done)

	// Detached from the request context; the worker always finishes.
	ctx := context.Background()

	conn, err := m.readerBackend.Open(ctx, readerID)
	if err != nil {
		w.err = err
		return
	}

	if err := conn.BeginTransaction(ctx); err != nil {
		_ = conn.Close()
		w.cleanups++
		w.err = err
		return
	}

	// Exactly-once cleanup of transaction + connection.
	defer func() {
		_ = conn.EndTransaction(context.Background())
		w.cleanups++
		_ = conn.Close()
		w.cleanups++
	}()

	if cur, ok := m.CardSession(readerID); !ok || cur != sid {
		w.err = card.NewError(card.ErrCardRemoved, "card session changed")
		return
	}

	// Execute may block inside native code. A soft deadline only applies to
	// cooperative backends (synthetic); real PC/SC finishes on its own.
	execCtx, cancelExec := context.WithTimeout(ctx, 15*time.Second)
	defer cancelExec()
	vals, warns, err := execute(execCtx, conn, fields)
	if err != nil {
		w.err = err
		return
	}
	if cur, ok := m.CardSession(readerID); !ok || cur != sid {
		w.err = card.NewError(card.ErrCardRemoved, "card session changed")
		return
	}
	if vals == nil {
		w.err = card.NewError(card.ErrReadFailed, "empty result")
		return
	}
	w.result = &card.ReadResult{
		ReaderID:      readerID,
		CardSessionID: sid,
		Fields:        *vals,
		Warnings:      warns,
	}
}

func newCardSessionID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "cs_fallback"
	}
	return "cs_" + hex.EncodeToString(b[:])
}
