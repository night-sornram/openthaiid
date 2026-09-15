//go:build windows

package pcsc

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ebfe/scard"
	"github.com/night-sornram/openthaiid/internal/card"
)

// osBackend wraps ebfe/scard on Windows (no cgo; WinSCard).
type osBackend struct {
	mu     sync.Mutex
	ctx    *scard.Context
	closed bool
	// lastStates retains EventState between GetStatusChange calls so
	// subsequent waits do not use StateUnaware (which returns immediately).
	lastStates map[string]scard.StateFlag
}

func newPlatformBackend() (Backend, error) {
	return &osBackend{lastStates: map[string]scard.StateFlag{}}, nil
}

func (b *osBackend) Name() string { return "pcsc" }

func (b *osBackend) Supported() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed || b.ctx == nil {
		return card.NewError(card.ErrPCSCUnavailable, "pcsc context not established")
	}
	return nil
}

func (b *osBackend) Establish(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.ctx != nil {
		return nil
	}
	pcscCtx, err := scard.EstablishContext()
	if err != nil {
		return card.NewError(card.ErrPCSCUnavailable, "pcsc context unavailable")
	}
	b.ctx = pcscCtx
	b.closed = false
	b.lastStates = map[string]scard.StateFlag{}
	return nil
}

func (b *osBackend) Release(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.ctx != nil {
		_ = b.ctx.Cancel()
		_ = b.ctx.Release()
		b.ctx = nil
	}
	b.closed = true
	b.lastStates = map[string]scard.StateFlag{}
	return nil
}

func (b *osBackend) Readers(ctx context.Context) ([]ReaderInfo, error) {
	b.mu.Lock()
	pcscCtx := b.ctx
	b.mu.Unlock()
	if pcscCtx == nil {
		return nil, card.NewError(card.ErrPCSCUnavailable, "pcsc context not established")
	}
	names, err := pcscCtx.ListReaders()
	if err != nil {
		if err == scard.ErrNoReadersAvailable {
			return []ReaderInfo{}, nil
		}
		return nil, card.NewError(card.ErrPCSCUnavailable, "reader enumeration failed")
	}
	out := make([]ReaderInfo, 0, len(names))
	for _, n := range names {
		out = append(out, ReaderInfo{ID: readerIDFor(n), Name: n})
	}
	return out, nil
}

func (b *osBackend) CardPresent(ctx context.Context, readerID string) (bool, error) {
	b.mu.Lock()
	pcscCtx := b.ctx
	b.mu.Unlock()
	if pcscCtx == nil {
		return false, card.NewError(card.ErrPCSCUnavailable, "pcsc context not established")
	}
	name, err := b.nameFor(readerID)
	if err != nil {
		return false, err
	}
	states := []scard.ReaderState{{
		Reader:       name,
		CurrentState: scard.StateUnaware,
	}}
	err = pcscCtx.GetStatusChange(states, 0)
	if err != nil {
		if err == scard.ErrTimeout {
			return false, nil
		}
		return false, card.NewError(card.ErrPCSCUnavailable, "status query failed")
	}
	return states[0].EventState&scard.StatePresent != 0, nil
}

func (b *osBackend) WaitForChange(ctx context.Context, timeout time.Duration) error {
	b.mu.Lock()
	pcscCtx := b.ctx
	if pcscCtx == nil {
		b.mu.Unlock()
		return card.NewError(card.ErrPCSCUnavailable, "pcsc context not established")
	}
	last := make(map[string]scard.StateFlag, len(b.lastStates))
	for k, v := range b.lastStates {
		last[k] = v
	}
	b.mu.Unlock()

	names, err := pcscCtx.ListReaders()
	if err != nil && err != scard.ErrNoReadersAvailable {
		return card.NewError(card.ErrPCSCUnavailable, "reader enumeration failed")
	}
	states := make([]scard.ReaderState, 0, len(names))
	for _, n := range names {
		cur, ok := last[n]
		if !ok {
			cur = scard.StateUnaware
		}
		states = append(states, scard.ReaderState{Reader: n, CurrentState: cur})
	}
	wait := timeout
	if wait <= 0 {
		wait = time.Second
	}
	if len(states) == 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
			return context.DeadlineExceeded
		}
	}
	err = pcscCtx.GetStatusChange(states, wait)
	if err != nil {
		if err == scard.ErrTimeout {
			b.retainStates(states)
			return context.DeadlineExceeded
		}
		b.mu.Lock()
		b.lastStates = map[string]scard.StateFlag{}
		b.mu.Unlock()
		return card.NewError(card.ErrPCSCUnavailable, "status change failed")
	}
	b.retainStates(states)
	return nil
}

func (b *osBackend) retainStates(states []scard.ReaderState) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.lastStates == nil {
		b.lastStates = map[string]scard.StateFlag{}
	}
	for _, st := range states {
		es := st.EventState &^ scard.StateUnaware
		b.lastStates[st.Reader] = es
	}
}

func (b *osBackend) CancelWait() error {
	b.mu.Lock()
	pcscCtx := b.ctx
	b.mu.Unlock()
	if pcscCtx == nil {
		return nil
	}
	return pcscCtx.Cancel()
}

func (b *osBackend) Open(ctx context.Context, readerID string) (CardConnection, error) {
	b.mu.Lock()
	pcscCtx := b.ctx
	b.mu.Unlock()
	if pcscCtx == nil {
		return nil, card.NewError(card.ErrPCSCUnavailable, "pcsc context not established")
	}
	name, err := b.nameFor(readerID)
	if err != nil {
		return nil, err
	}
	sc, err := pcscCtx.Connect(name, scard.ShareShared, scard.ProtocolAny)
	if err != nil {
		if err == scard.ErrNoSmartcard {
			return nil, card.NewError(card.ErrNoCard, "no card present")
		}
		return nil, card.NewError(card.ErrReadFailed, "card connect failed")
	}
	return &osConn{conn: sc, readerID: readerID}, nil
}

func (b *osBackend) nameFor(readerID string) (string, error) {
	readers, err := b.Readers(context.Background())
	if err != nil {
		return "", err
	}
	for _, r := range readers {
		if r.ID == readerID {
			return r.Name, nil
		}
	}
	return "", card.NewError(card.ErrReaderRemoved, "reader not found")
}

func readerIDFor(name string) string {
	return fmt.Sprintf("rd_%x", simpleHash(name))
}

func simpleHash(s string) uint64 {
	var h uint64 = 14695981039346656037
	for i := 0; i < len(s); i++ {
		h ^= uint64(s[i])
		h *= 1099511628211
	}
	return h
}

type osConn struct {
	mu       sync.Mutex
	conn     *scard.Card
	tx       bool
	closed   bool
	readerID string
}

func (c *osConn) Transmit(ctx context.Context, command []byte, maxResp int) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed || c.conn == nil {
		return nil, card.NewError(card.ErrCancelled, "connection closed")
	}
	resp, err := c.conn.Transmit(command)
	if err != nil {
		return nil, card.NewError(card.ErrReadFailed, "transmit failed")
	}
	if maxResp > 0 && len(resp) > maxResp {
		return nil, card.NewError(card.ErrReadFailed, "response exceeds bound")
	}
	return resp, nil
}

func (c *osConn) BeginTransaction(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed || c.conn == nil {
		return card.NewError(card.ErrCancelled, "connection closed")
	}
	if err := c.conn.BeginTransaction(); err != nil {
		return card.NewError(card.ErrBusy, "transaction unavailable")
	}
	c.tx = true
	return nil
}

func (c *osConn) EndTransaction(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed || c.conn == nil || !c.tx {
		return nil
	}
	_ = c.conn.EndTransaction(scard.LeaveCard)
	c.tx = false
	return nil
}

func (c *osConn) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil
	}
	c.closed = true
	if c.conn != nil {
		if c.tx {
			_ = c.conn.EndTransaction(scard.LeaveCard)
			c.tx = false
		}
		_ = c.conn.Disconnect(scard.LeaveCard)
		c.conn = nil
	}
	return nil
}

var _ Backend = (*osBackend)(nil)
var _ CardConnection = (*osConn)(nil)
