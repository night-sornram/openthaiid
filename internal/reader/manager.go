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

	backend pcsc.Backend
	// readerBackend is a second context used only for Open/Transmit so
	// monitoring can continue during a read.
	readerBackend pcsc.Backend

	readers       map[string]*ReaderState
	backendStatus string

	reading bool
	stalled bool

	subscribers map[chan ChangeEvent]struct{}

	// pendingGeneration invalidated card sessions on removal.
	// cardSessionID is the insertion generation token.
}

// NewManager builds a manager around two backend instances (same factory).
func NewManager(monitor, reader pcsc.Backend) *Manager {
	return &Manager{
		backend:       monitor,
		readerBackend: reader,
		readers:       map[string]*ReaderState{},
		backendStatus: "unavailable",
		subscribers:   map[chan ChangeEvent]struct{}{},
	}
}

// Start establishes both backends and begins the monitoring loop.
func (m *Manager) Start(ctx context.Context) error {
	if err := m.backend.Establish(ctx); err != nil {
		return err
	}
	if err := m.readerBackend.Establish(ctx); err != nil {
		_ = m.backend.Release(ctx)
		return err
	}
	if err := m.resync(ctx); err != nil {
		return err
	}
	m.mu.Lock()
	m.backendStatus = "available"
	m.mu.Unlock()
	go m.monitorLoop(ctx)
	return nil
}

// Stop releases backends.
func (m *Manager) Stop(ctx context.Context) error {
	_ = m.backend.CancelWait()
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

// Snapshot returns the current reader view.
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
	for ctx.Err() == nil {
		err := m.backend.WaitForChange(ctx, time.Second)
		if ctx.Err() != nil {
			return
		}
		if err != nil && err != context.DeadlineExceeded {
			m.mu.Lock()
			if m.backendStatus != "stalled" {
				m.backendStatus = "unavailable"
			}
			m.mu.Unlock()
			// Recovery: try to re-establish after a short pause.
			select {
			case <-ctx.Done():
				return
			case <-time.After(2 * time.Second):
			}
			_ = m.backend.Release(ctx)
			if e2 := m.backend.Establish(ctx); e2 == nil {
				m.mu.Lock()
				m.backendStatus = "available"
				m.mu.Unlock()
			}
		}
		_ = m.resync(ctx)
	}
}

// resync diffs the backend inventory and emits events.
func (m *Manager) resync(ctx context.Context) error {
	if m.backend.Supported() != nil {
		m.mu.Lock()
		if !m.stalled {
			m.backendStatus = "unavailable"
		}
		m.mu.Unlock()
		return nil
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
	m.reading = true
	m.mu.Unlock()
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

	// Re-check generation after open; removal may have raced.
	if cur, ok := m.CardSession(readerID); !ok || cur != sid {
		return nil, card.NewError(card.ErrCardRemoved, "card session changed")
	}

	vals, warns, err := req.Execute(ctx, conn, fields)
	if err != nil {
		return nil, err
	}
	// Re-check before returning a result.
	if cur, ok := m.CardSession(readerID); !ok || cur != sid {
		return nil, card.NewError(card.ErrCardRemoved, "card session changed")
	}
	return &card.ReadResult{
		ReaderID:      readerID,
		CardSessionID: sid,
		Fields:        *vals,
		Warnings:      warns,
	}, nil
}

func newCardSessionID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "cs_fallback"
	}
	return "cs_" + hex.EncodeToString(b[:])
}
