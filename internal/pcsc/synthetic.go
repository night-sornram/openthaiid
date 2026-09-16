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
	mu          sync.Mutex
	readers     map[string]string // id → name
	present     map[string]bool
	script      map[string][]byte
	failOpen    map[string]bool
	stalled     bool
	serviceDown bool

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
	s.state.mu.Lock()
	down := s.state.serviceDown
	s.state.mu.Unlock()
	if down {
		return card.NewError(card.ErrPCSCUnavailable, "synthetic card service is down")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	// Mirror the OS backend: Release + Establish is a reconnect, not a
	// permanent close. Card handles still use their own closed flag.
	s.closed = false
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
	down := s.state.serviceDown
	s.state.mu.Unlock()
	if stalled {
		return card.NewError(card.ErrBackendStalled, "synthetic backend stalled")
	}
	if down {
		return card.NewError(card.ErrPCSCUnavailable, "synthetic card service is down")
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

// SetServiceDown simulates a stopped PC/SC daemon / Smart Card service.
// Establish and Supported fail with PCSC_UNAVAILABLE until it is cleared.
// Shared across monitor/read backends. Does not model OS sleep itself.
func (s *SyntheticBackend) SetServiceDown(v bool) {
	s.state.SetServiceDown(v)
	s.notifyAll()
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
// with independently specified command responses for every supported textual field.
// A second reader can be added with AddReader for multi-reader tests; the
// default `-synthetic` CLI path stays the single-reader auto-select flow.
func (s *SyntheticBackend) SeedDemoCard() {
	seedDemoCard(s.state)
	s.notifyAll()
}

// SeedSecondDemoReader adds a second fictional reader sharing the same
// scripted card responses. Used by tests and OPENTHAIID_SYNTHETIC_READERS=2.
func (s *SyntheticBackend) SeedSecondDemoReader() {
	const readerID = "synthetic-second"
	s.AddReader(readerID, "SYNTHETIC second reader (fictional)")
	s.InsertCard(readerID)
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

func (st *SyntheticState) SetServiceDown(v bool) {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.serviceDown = v
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
	enNameCmd := string([]byte{0x80, 0xB0, 0x00, 0x75, 0x02, 0x00, 0x64})
	birthCmd := string([]byte{0x80, 0xB0, 0x00, 0xD9, 0x02, 0x00, 0x08})
	genderCmd := string([]byte{0x80, 0xB0, 0x00, 0xE1, 0x02, 0x00, 0x01})
	issuerCmd := string([]byte{0x80, 0xB0, 0x00, 0xF6, 0x02, 0x00, 0x64})
	issueCmd := string([]byte{0x80, 0xB0, 0x01, 0x67, 0x02, 0x00, 0x08})
	expireCmd := string([]byte{0x80, 0xB0, 0x01, 0x6F, 0x02, 0x00, 0x08})
	addrCmd := string([]byte{0x80, 0xB0, 0x15, 0x79, 0x02, 0x00, 0x64})

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

	// English name: Mr.#Somchai##Jaidee (ASCII, same '#' layout as Thai name)
	enName := []byte("Mr.#Somchai##Jaidee")

	// Birth: 2533-04-12 BE → 1990-04-12 CE
	birth := []byte("25330412")

	// Gender community convention: '1' = male
	gender := []byte{'1'}

	// Issuer: clearly fictional municipality (TIS-620)
	issuer := tis620("เทศบาลนครนิทาน")

	// Issue 2560-01-15 BE → 2017-01-15 CE; expire 2570-01-14 BE → 2027-01-14 CE
	issue := []byte("25600115")
	expire := []byte("25700114")

	// Address: 7-part '#' layout; empty alley (consecutive ##).
	addr := tis620("99")
	addr = append(addr, '#')
	addr = append(addr, tis620("หมู่ที่ 1")...)
	addr = append(addr, '#', '#')
	addr = append(addr, tis620("ถนนนิทาน")...)
	addr = append(addr, '#')
	addr = append(addr, tis620("ตำบลบ้านใหม่")...)
	addr = append(addr, '#')
	addr = append(addr, tis620("อำเภอเมือง")...)
	addr = append(addr, '#')
	addr = append(addr, tis620("จังหวัดนิทาน")...)

	st.SetScriptResponse(selectCmd, []byte{0x90, 0x00})
	st.SetScriptResponse(cidCmd, append(append([]byte(nil), cid...), 0x90, 0x00))
	st.SetScriptResponse(nameCmd, append(padBlock(name, 100), 0x90, 0x00))
	st.SetScriptResponse(enNameCmd, append(padBlock(enName, 100), 0x90, 0x00))
	st.SetScriptResponse(birthCmd, append(append([]byte(nil), birth...), 0x90, 0x00))
	st.SetScriptResponse(genderCmd, append(append([]byte(nil), gender...), 0x90, 0x00))
	st.SetScriptResponse(issuerCmd, append(padBlock(issuer, 100), 0x90, 0x00))
	st.SetScriptResponse(issueCmd, append(append([]byte(nil), issue...), 0x90, 0x00))
	st.SetScriptResponse(expireCmd, append(append([]byte(nil), expire...), 0x90, 0x00))
	st.SetScriptResponse(addrCmd, append(padBlock(addr, 100), 0x90, 0x00))
	seedFictionalPortrait(st)
}

func seedFictionalPortrait(st *SyntheticState) {
	region := card.PadPortraitRegion(card.FictionalPortraitJPEG())
	const part = 0xFF
	const base = 0x017B
	for i := 0; i < 20; i++ {
		off := base + i*part
		cmd := string([]byte{0x80, 0xB0, byte(off >> 8), byte(off & 0xFF), 0x02, 0x00, 0xFF})
		st.SetScriptResponse(cmd, append(append([]byte(nil), region[i*part:(i+1)*part]...), 0x90, 0x00))
	}
}

func padBlock(b []byte, n int) []byte {
	padded := make([]byte, n)
	copy(padded, b)
	for i := len(b); i < n; i++ {
		padded[i] = 0x20
	}
	return padded
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
	if err := c.presenceError(); err != nil {
		return nil, err
	}
	return c.sb.state.responseFor(string(command))
}

func (c *syntheticConn) presenceError() error {
	c.sb.state.mu.Lock()
	defer c.sb.state.mu.Unlock()
	if _, ok := c.sb.state.readers[c.readerID]; !ok {
		return card.NewError(card.ErrReaderRemoved, "reader unplugged")
	}
	if !c.sb.state.present[c.readerID] {
		return card.NewError(card.ErrCardRemoved, "card removed")
	}
	return nil
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
	if err := c.presenceError(); err != nil {
		return err
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
