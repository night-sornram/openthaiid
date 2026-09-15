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

	reading bool
	stalled bool

	subscribers map[chan ChangeEvent]struct{}

	// monitorDone is closed when the monitoring goroutine exits.
	monitorDone    chan struct{}
	monitorStop    context.CancelFunc
	monitorStarted bool
}

// NewManager builds a manager around two backend instances (same factory).
func NewManager(monitor, reader pcsc.Backend) *Manager {
	return &Manager{
		backend:       monitor,
		readerBackend: reader,
		readers:       map[string]*ReaderState{},
		backendStatus: "unavailable",
		subscribers:   map[chan ChangeEvent]struct{}{},
		monitorDone:   make(chan struct{}),
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
func (m *Manager) Stop(ctx context.Context) error {
	m.mu.Lock()
	stop := m.monitorStop
	started := m.monitorStarted
	done := m.monitorDone
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

func (m *Manager) emit(ev ChangeEvent) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for ch := range m.subscribers {
		select {
		case ch <- ev:
		default:
			// Drop if the subscriber is slow; they should resync.
		}
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
	return Snapshot{Readers: out, Backend: m.backendStatus}
}

func (m *Manager) monitorLoop(ctx context.Context) {
	defer close(m.monitorDone)
	for ctx.Err() == nil {
		if m.backend.Supported() != nil {
			// Backend not established: try recovery.
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
			_ = m.backend.Release(ctx)
			_ = m.readerBackend.Release(ctx)
			if e2 := m.backend.Establish(ctx); e2 == nil {
				_ = m.readerBackend.Establish(ctx)
				m.mu.Lock()
				m.backendStatus = "available"
				m.mu.Unlock()
			}
		}
		_ = m.resync(ctx)
	}
}

func (m *Manager) recoverBackend(ctx context.Context) bool {
	if err := m.backend.Establish(ctx); err != nil {
		return false
	}
	if err := m.readerBackend.Establish(ctx); err != nil {
		_ = m.backend.Release(ctx)
		return false
	}
	m.mu.Lock()
	m.backendStatus = "available"
	m.stalled = false
	m.mu.Unlock()
	_ = m.resync(ctx)
	return true
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
	m.readers = next
	m.mu.Unlock()

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
		} else if nowPresent {
			// Preserve existing session id while the card remains present.
			st.CardSessionID = old.CardSessionID
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

// SelectReader picks a reader when exactly one exists.
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
// The native slot stays occupied until the blocked call returns; late
// results from a cancelled context are discarded.
func (m *Manager) Read(ctx context.Context, req ReadRequest) (*card.ReadResult, error) {
	fields, err := card.NormalizeFields(req.Fields)
	if err != nil {
		return nil, err
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
	m.reading = true
	m.mu.Unlock()

	// Occupies the single native read slot until the call returns, even if
	// ctx is already cancelled — the native work cannot be interrupted.
	done := make(chan struct{})
	defer close(done)
	defer func() {
		m.mu.Lock()
		m.reading = false
		m.mu.Unlock()
	}()

	conn, err := m.readerBackend.Open(ctx, readerID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = conn.Close() }()

	if err := conn.BeginTransaction(ctx); err != nil {
		return nil, err
	}
	defer func() { _ = conn.EndTransaction(context.Background()) }()

	if cur, ok := m.CardSession(readerID); !ok || cur != sid {
		return nil, card.NewError(card.ErrCardRemoved, "card session changed")
	}

	type execResult struct {
		vals  *card.FieldValues
		warns []card.Warning
		err   error
	}
	resCh := make(chan execResult, 1)
	go func() {
		vals, warns, err := req.Execute(ctx, conn, fields)
		resCh <- execResult{vals, warns, err}
	}()

	var res execResult
	select {
	case res = <-resCh:
	case <-ctx.Done():
		// Wait for the native call to finish so the slot stays honest;
		// discard the late result.
		go func() {
			<-resCh
			_ = conn.Close()
		}()
		return nil, card.NewError(card.ErrCancelled, "read cancelled")
	}
	if res.err != nil {
		return nil, res.err
	}
	if cur, ok := m.CardSession(readerID); !ok || cur != sid {
		return nil, card.NewError(card.ErrCardRemoved, "card session changed")
	}
	return &card.ReadResult{
		ReaderID:      readerID,
		CardSessionID: sid,
		Fields:        *res.vals,
		Warnings:      res.warns,
	}, nil
}

func newCardSessionID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "cs_fallback"
	}
	return "cs_" + hex.EncodeToString(b[:])
}
