package pcsc

import (
	"context"
	"encoding/hex"
	"sync"
	"time"

	"github.com/night-sornram/openthaiid/internal/card"
)

// SyntheticState is shared device state for the synthetic backend family.
// Separate Backend contexts (monitor vs read) share one state so inventory
// and card presence stay consistent.
type SyntheticState struct {
	mu       sync.Mutex
	readers  map[string]string // id → name
	present  map[string]bool
	script   map[string][]byte
	failOpen map[string]bool
	stalled  bool

	// block maps a lifecycle phase → channel closed to unblock it.
	// Phases: open, begin, transmit, end, close.
	block map[string]chan struct{}
	// cleanupCounts records EndTransaction / Close invocations for
	// exactly-once cleanup assertions.
	endTxCalls  int
	closeCalls  int
	openCalls   int
	blockHooked bool
}

// NewSyntheticState creates empty shared device state.
func NewSyntheticState() *SyntheticState {
	return &SyntheticState{
		readers:  map[string]string{},
		present:  map[string]bool{},
		script:   map[string][]byte{},
		failOpen: map[string]bool{},
		block:    map[string]chan struct{}{},
	}
}

// SyntheticBackend is a deterministic Backend for tests and CI.
// It never touches a real reader.
type SyntheticBackend struct {
	mu     sync.Mutex
	ready  bool
	closed bool

	state *SyntheticState

	waiters []chan struct{}
}

// NewSynthetic returns a synthetic backend with private state.
func NewSynthetic() *SyntheticBackend {
	return NewSyntheticWithState(NewSyntheticState())
}

// NewSyntheticWithState returns a synthetic backend sharing device state.
func NewSyntheticWithState(state *SyntheticState) *SyntheticBackend {
	return &SyntheticBackend{state: state}
}

// State exposes the shared device state (test / CLI controls).
func (s *SyntheticBackend) State() *SyntheticState { return s.state }

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
	ready, closed := s.ready, s.closed
	s.mu.Unlock()
	s.state.mu.Lock()
	stalled := s.state.stalled
	s.state.mu.Unlock()
	if stalled {
		return card.NewError(card.ErrBackendStalled, "synthetic backend stalled")
	}
	if closed || !ready {
		return card.NewError(card.ErrPCSCUnavailable, "synthetic backend not established")
	}
	return nil
}

func (s *SyntheticBackend) Readers(ctx context.Context) ([]ReaderInfo, error) {
	if err := s.Supported(); err != nil {
		return nil, err
	}
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	out := make([]ReaderInfo, 0, len(s.state.readers))
	for id, name := range s.state.readers {
		out = append(out, ReaderInfo{ID: id, Name: name})
	}
	return out, nil
}

func (s *SyntheticBackend) CardPresent(ctx context.Context, readerID string) (bool, error) {
	if err := s.Supported(); err != nil {
		return false, err
	}
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	if _, ok := s.state.readers[readerID]; !ok {
		return false, card.NewError(card.ErrReaderRemoved, "reader not found")
	}
	return s.state.present[readerID], nil
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
	if err := s.state.waitPhase(ctx, "open"); err != nil {
		return nil, err
	}
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	if s.state.failOpen[readerID] {
		return nil, card.NewError(card.ErrReadFailed, "open failed")
	}
	if _, ok := s.state.readers[readerID]; !ok {
		return nil, card.NewError(card.ErrReaderRemoved, "reader not found")
	}
	if !s.state.present[readerID] {
		return nil, card.NewError(card.ErrNoCard, "no card")
	}
	s.state.openCalls++
	return &syntheticConn{sb: s, readerID: readerID}, nil
}

// --- shared-state controls (also used by the CLI synthetic demo) ---

// AddReader inserts a reader and notifies watchers.
func (s *SyntheticBackend) AddReader(id, name string) {
	s.state.AddReader(id, name)
	s.notifyAll()
}

// RemoveReader removes a reader and notifies watchers.
func (s *SyntheticBackend) RemoveReader(id string) {
	s.state.RemoveReader(id)
	s.notifyAll()
}

// InsertCard marks a card present.
func (s *SyntheticBackend) InsertCard(id string) {
	s.state.InsertCard(id)
	s.notifyAll()
}

// RemoveCard marks a card absent.
func (s *SyntheticBackend) RemoveCard(id string) {
	s.state.RemoveCard(id)
	s.notifyAll()
}

// SetStalled simulates a hung native call surface.
func (s *SyntheticBackend) SetStalled(v bool) {
	s.state.SetStalled(v)
}

// BlockPhase parks the named lifecycle phase until UnblockPhase is called.
// Phases: open | begin | transmit | end | close.
func (s *SyntheticBackend) BlockPhase(phase string) {
	s.state.BlockPhase(phase)
}

// UnblockPhase releases a previously blocked phase.
func (s *SyntheticBackend) UnblockPhase(phase string) {
	s.state.UnblockPhase(phase)
}

// CleanupStats returns end-transaction and close call counts.
func (s *SyntheticBackend) CleanupStats() (endTx, closes int) {
	return s.state.CleanupStats()
}

// OpenStats returns the number of Open calls.
func (s *SyntheticBackend) OpenStats() int {
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	return s.state.openCalls
}

func (st *SyntheticState) BlockPhase(phase string) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if _, ok := st.block[phase]; !ok {
		st.block[phase] = make(chan struct{})
	}
}

func (st *SyntheticState) UnblockPhase(phase string) {
	st.mu.Lock()
	ch, ok := st.block[phase]
	delete(st.block, phase)
	st.mu.Unlock()
	if ok {
		close(ch)
	}
}

func (st *SyntheticState) CleanupStats() (endTx, closes int) {
	st.mu.Lock()
	defer st.mu.Unlock()
	return st.endTxCalls, st.closeCalls
}

func (st *SyntheticState) waitPhase(ctx context.Context, phase string) error {
	st.mu.Lock()
	ch, ok := st.block[phase]
	st.mu.Unlock()
	if !ok {
		return nil
	}
	// Non-cooperative: wait is not cancelled by the request context.
	// Use a long safety bound only.
	timer := time.NewTimer(60 * time.Second)
	defer timer.Stop()
	select {
	case <-ch:
		return nil
	case <-timer.C:
		return card.NewError(card.ErrBackendStalled, "blocked phase timed out")
	case <-ctx.Done():
		// Still honor an explicit caller cancel for cooperative tests that
		// pass a cancellable context into Transmit/Open helpers.
		return ctx.Err()
	}
}

// SetScriptResponse programs the response for a command string.
func (s *SyntheticBackend) SetScriptResponse(cmd string, resp []byte) {
	s.state.SetScriptResponse(cmd, resp)
}

// SeedDemoCard installs one clearly labeled fictional reader + Thai ID card
// with independently specified command responses for the three MVP fields.
func (s *SyntheticBackend) SeedDemoCard() {
	seedDemoCard(s.state)
	s.notifyAll()
}

func (st *SyntheticState) AddReader(id, name string) {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.readers[id] = name
	st.present[id] = false
}

func (st *SyntheticState) RemoveReader(id string) {
	st.mu.Lock()
	defer st.mu.Unlock()
	delete(st.readers, id)
	delete(st.present, id)
}

func (st *SyntheticState) InsertCard(id string) {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.present[id] = true
}

func (st *SyntheticState) RemoveCard(id string) {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.present[id] = false
}

func (st *SyntheticState) SetStalled(v bool) {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.stalled = v
}

func (st *SyntheticState) SetScriptResponse(cmd string, resp []byte) {
	st.mu.Lock()
	defer st.mu.Unlock()
	cp := make([]byte, len(resp))
	copy(cp, resp)
	st.script[cmd] = cp
}

func (st *SyntheticState) responseFor(cmd string) ([]byte, error) {
	st.mu.Lock()
	defer st.mu.Unlock()
	resp, ok := st.script[cmd]
	if !ok {
		// Also accept hex form for convenience.
		if r, ok2 := st.script[hex.EncodeToString([]byte(cmd))]; ok2 {
			out := make([]byte, len(r))
			copy(out, r)
			return out, nil
		}
		return nil, card.NewError(card.ErrReadFailed, "no synthetic response")
	}
	out := make([]byte, len(resp))
	copy(out, resp)
	return out, nil
}

// seedDemoCard is independent of parser constants: command bytes match
// testdata/synthetic/fixtures.json and APDU.md.
func seedDemoCard(st *SyntheticState) {
	const readerID = "synthetic-acr39u"
	st.AddReader(readerID, "SYNTHETIC ACR39U (fictional demo)")
	st.InsertCard(readerID)

	selectCmd := string([]byte{0x00, 0xA4, 0x04, 0x00, 0x08, 0xA0, 0x00, 0x00, 0x00, 0x54, 0x48, 0x00, 0x01})
	cidCmd := string([]byte{0x80, 0xB0, 0x00, 0x04, 0x02, 0x00, 0x0D})
	nameCmd := string([]byte{0x80, 0xB0, 0x00, 0x11, 0x02, 0x00, 0x64})
	birthCmd := string([]byte{0x80, 0xB0, 0x00, 0xD9, 0x02, 0x00, 0x08})

	// Fictional citizen id with a valid check digit (computed independently).
	// 1 1017 0123456 7 is the historical sample; use a clearly fictional one:
	// 0000000000000 would fail length semantics; use 1234567890121 (checksum advisory).
	cid := []byte("1234567890121")

	// Thai name: นาย#สมชาย##ใจดี  (TIS-620)
	name := tis620("นาย")
	name = append(name, '#')
	name = append(name, tis620("สมชาย")...)
	name = append(name, '#', '#')
	name = append(name, tis620("ใจดี")...)
	padded := make([]byte, 100)
	copy(padded, name)
	for i := len(name); i < 100; i++ {
		padded[i] = 0x20
	}

	// Birth: 2533-04-12 BE → 1990-04-12 CE
	birth := []byte("25330412")

	st.SetScriptResponse(selectCmd, []byte{0x90, 0x00})
	st.SetScriptResponse(cidCmd, append(append([]byte(nil), cid...), 0x90, 0x00))
	st.SetScriptResponse(nameCmd, append(append([]byte(nil), padded...), 0x90, 0x00))
	st.SetScriptResponse(birthCmd, append(append([]byte(nil), birth...), 0x90, 0x00))
}

func tis620(s string) []byte {
	var out []byte
	for _, r := range s {
		if r < 0x80 {
			out = append(out, byte(r))
			continue
		}
		if r >= 0x0E01 && r <= 0x0E5B {
			out = append(out, byte(0xA1+(r-0x0E01)))
			continue
		}
		out = append(out, '?')
	}
	return out
}

func (s *SyntheticBackend) notifyAll() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.notifyLocked()
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
	if err := c.sb.state.waitPhase(ctx, "transmit"); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil, card.NewError(card.ErrCancelled, "connection closed")
	}
	return c.sb.state.responseFor(string(command))
}

func (c *syntheticConn) BeginTransaction(ctx context.Context) error {
	if err := c.sb.state.waitPhase(ctx, "begin"); err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return card.NewError(card.ErrCancelled, "connection closed")
	}
	c.inTx = true
	return nil
}

func (c *syntheticConn) EndTransaction(ctx context.Context) error {
	// Cleanup is non-cooperative: do not honor request cancellation here.
	// UnblockPhase (or the safety timeout in waitPhase) is the only way out.
	_ = c.sb.state.waitPhase(context.Background(), "end")
	c.mu.Lock()
	c.inTx = false
	c.sb.state.mu.Lock()
	c.sb.state.endTxCalls++
	c.sb.state.mu.Unlock()
	c.mu.Unlock()
	return nil
}

func (c *syntheticConn) Close() error {
	_ = c.sb.state.waitPhase(context.Background(), "close")
	c.mu.Lock()
	c.closed = true
	c.inTx = false
	c.sb.state.mu.Lock()
	c.sb.state.closeCalls++
	c.sb.state.mu.Unlock()
	c.mu.Unlock()
	return nil
}

var _ Backend = (*SyntheticBackend)(nil)
var _ CardConnection = (*syntheticConn)(nil)
