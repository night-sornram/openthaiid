package pcsc

import (
	"context"
	"sync"
	"time"

	"github.com/night-sornram/openthaiid/internal/card"
)

// SyntheticBackend is a deterministic Backend for tests and CI.
// It never touches a real reader.
type SyntheticBackend struct {
	mu      sync.Mutex
	ready   bool
	stalled bool
	closed  bool

	readers map[string]string // id → name
	present map[string]bool

	waiters []chan struct{}

	// Script maps APDU hex-ish name or full command string to responses.
	// For the synthetic MVP we key on command string.
	script map[string][]byte
	// FailOpen makes Open fail for a reader.
	failOpen map[string]bool
}

// NewSynthetic returns an empty synthetic backend.
func NewSynthetic() *SyntheticBackend {
	return &SyntheticBackend{
		readers:  map[string]string{},
		present:  map[string]bool{},
		script:   map[string][]byte{},
		failOpen: map[string]bool{},
	}
}

func (s *SyntheticBackend) Name() string { return "synthetic" }

func (s *SyntheticBackend) Establish(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return card.NewError(card.ErrPCSCUnavailable, "synthetic backend closed")
	}
	s.ready = true
	return nil
}

func (s *SyntheticBackend) Release(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	s.ready = false
	s.notifyLocked()
	return nil
}

func (s *SyntheticBackend) Supported() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stalled {
		return card.NewError(card.ErrBackendStalled, "synthetic backend stalled")
	}
	if !s.ready {
		return card.NewError(card.ErrPCSCUnavailable, "synthetic backend not established")
	}
	return nil
}

func (s *SyntheticBackend) Readers(ctx context.Context) ([]ReaderInfo, error) {
	if err := s.Supported(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]ReaderInfo, 0, len(s.readers))
	for id, name := range s.readers {
		out = append(out, ReaderInfo{ID: id, Name: name})
	}
	return out, nil
}

func (s *SyntheticBackend) CardPresent(ctx context.Context, readerID string) (bool, error) {
	if err := s.Supported(); err != nil {
		return false, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.readers[readerID]; !ok {
		return false, card.NewError(card.ErrReaderRemoved, "reader not found")
	}
	return s.present[readerID], nil
}

func (s *SyntheticBackend) WaitForChange(ctx context.Context, timeout time.Duration) error {
	if err := s.Supported(); err != nil {
		return err
	}
	s.mu.Lock()
	ch := make(chan struct{}, 1)
	s.waiters = append(s.waiters, ch)
	s.mu.Unlock()

	var timeoutCh <-chan time.Time
	if timeout > 0 {
		t := time.NewTimer(timeout)
		defer t.Stop()
		timeoutCh = t.C
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-ch:
		return nil
	case <-timeoutCh:
		return context.DeadlineExceeded
	}
}

func (s *SyntheticBackend) CancelWait() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.notifyLocked()
	return nil
}

func (s *SyntheticBackend) Open(ctx context.Context, readerID string) (CardConnection, error) {
	if err := s.Supported(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failOpen[readerID] {
		return nil, card.NewError(card.ErrReadFailed, "open failed")
	}
	name, ok := s.readers[readerID]
	if !ok {
		return nil, card.NewError(card.ErrReaderRemoved, "reader not found")
	}
	if !s.present[readerID] {
		return nil, card.NewError(card.ErrNoCard, "no card")
	}
	_ = name
	return &syntheticConn{sb: s, readerID: readerID}, nil
}

// --- test controls ---

// AddReader inserts a reader and notifies watchers.
func (s *SyntheticBackend) AddReader(id, name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.readers[id] = name
	s.present[id] = false
	s.notifyLocked()
}

// RemoveReader removes a reader and notifies watchers.
func (s *SyntheticBackend) RemoveReader(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.readers, id)
	delete(s.present, id)
	s.notifyLocked()
}

// InsertCard marks a card present.
func (s *SyntheticBackend) InsertCard(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.present[id] = true
	s.notifyLocked()
}

// RemoveCard marks a card absent.
func (s *SyntheticBackend) RemoveCard(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.present[id] = false
	s.notifyLocked()
}

// SetStalled simulates a hung native call surface.
func (s *SyntheticBackend) SetStalled(v bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stalled = v
}

// SetScriptResponse programs the response for a command string.
func (s *SyntheticBackend) SetScriptResponse(cmd string, resp []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.script[cmd] = resp
}

func (s *SyntheticBackend) responseFor(cmd string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	resp, ok := s.script[cmd]
	if !ok {
		return nil, card.NewError(card.ErrReadFailed, "no synthetic response")
	}
	out := make([]byte, len(resp))
	copy(out, resp)
	return out, nil
}

func (s *SyntheticBackend) notifyLocked() {
	for _, ch := range s.waiters {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
	s.waiters = nil
}

type syntheticConn struct {
	sb       *SyntheticBackend
	readerID string
	mu       sync.Mutex
	closed   bool
	inTx     bool
}

func (c *syntheticConn) Transmit(ctx context.Context, command []byte, maxResp int) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil, card.NewError(card.ErrCancelled, "connection closed")
	}
	return c.sb.responseFor(string(command))
}

func (c *syntheticConn) BeginTransaction(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return card.NewError(card.ErrCancelled, "connection closed")
	}
	c.inTx = true
	return nil
}

func (c *syntheticConn) EndTransaction(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.inTx = false
	return nil
}

func (c *syntheticConn) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
	c.inTx = false
	return nil
}

var _ Backend = (*SyntheticBackend)(nil)
var _ CardConnection = (*syntheticConn)(nil)
