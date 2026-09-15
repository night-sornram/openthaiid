// Package pcsc owns native smart-card backend contracts.
// Production uses OS PC/SC; tests use a synthetic implementation of the
// same interfaces.
package pcsc

import (
	"context"
	"time"
)

// ReaderInfo describes one physical or synthetic reader.
type ReaderInfo struct {
	// ID is an opaque, stable-for-this-process identifier.
	ID string `json:"readerId"`
	// Name is the OS-provided reader name (not personal data).
	Name string `json:"name"`
}

// CardPresence is the presence dimension for one reader.
type CardPresence string

const (
	PresenceAbsent      CardPresence = "absent"
	PresencePresent     CardPresence = "present"
	PresenceUnsupported CardPresence = "unsupported"
)

// Snapshot is a point-in-time reader inventory.
type Snapshot struct {
	Readers []ReaderInfo `json:"readers"`
	// Present maps reader ID → true when a card is present.
	Present map[string]bool `json:"present"`
}

// Event is emitted by the monitoring loop.
type Event struct {
	Kind      string // reader.connected | reader.disconnected | card.inserted | card.removed
	ReaderID  string
	Name      string
	Timestamp time.Time
}

// CardConnection is a live connection to one inserted card.
// Implementations serialize Transmit calls on a given connection.
type CardConnection interface {
	// Transmit sends command and returns the response body including SW.
	Transmit(ctx context.Context, command []byte, maxResp int) ([]byte, error)
	// BeginTransaction starts exclusive access (PC/SC transaction).
	BeginTransaction(ctx context.Context) error
	// EndTransaction releases exclusive access.
	EndTransaction(ctx context.Context) error
	// Close releases the connection. Safe to call once.
	Close() error
}

// Backend enumerates readers, observes state changes, and opens cards.
// Monitoring and reading must use separate Backend instances so each owns
// its native PC/SC context.
type Backend interface {
	// Name identifies the backend ("pcsc" or "synthetic").
	Name() string
	// Establish prepares native resources.
	Establish(ctx context.Context) error
	// Release tears down native resources.
	Release(ctx context.Context) error
	// Readers returns the current inventory.
	Readers(ctx context.Context) ([]ReaderInfo, error)
	// CardPresent reports whether a card is currently present in readerID.
	// Implementations should use status flags, not a full connect/read.
	CardPresent(ctx context.Context, readerID string) (bool, error)
	// WaitForChange blocks until the inventory changes, ctx is cancelled,
	// or timeout elapses. timeout 0 means a short default poll.
	WaitForChange(ctx context.Context, timeout time.Duration) error
	// CancelWait unblocks a concurrent WaitForChange.
	CancelWait() error
	// Open connects to the card currently present in readerID.
	Open(ctx context.Context, readerID string) (CardConnection, error)
	// Supported reports whether the OS PC/SC stack is usable.
	Supported() error
}

// Factory constructs a Backend. The synthetic factory ignores hardware.
type Factory func() (Backend, error)
