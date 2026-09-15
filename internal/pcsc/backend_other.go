//go:build !darwin && !linux && !windows

package pcsc

import (
	"context"
	"time"

	"github.com/night-sornram/openthaiid/internal/card"
)

// unsupportedBackend is compiled on GOOS values without a PC/SC adapter.
type unsupportedBackend struct{}

func newPlatformBackend() (Backend, error) {
	return &unsupportedBackend{}, nil
}

func (u *unsupportedBackend) Name() string { return "unsupported" }

func (u *unsupportedBackend) Establish(ctx context.Context) error {
	return card.NewError(card.ErrPCSCUnavailable, "platform not supported")
}

func (u *unsupportedBackend) Release(ctx context.Context) error { return nil }

func (u *unsupportedBackend) Readers(ctx context.Context) ([]ReaderInfo, error) {
	return nil, card.NewError(card.ErrPCSCUnavailable, "platform not supported")
}

func (u *unsupportedBackend) CardPresent(ctx context.Context, readerID string) (bool, error) {
	return false, card.NewError(card.ErrPCSCUnavailable, "platform not supported")
}

func (u *unsupportedBackend) WaitForChange(ctx context.Context, timeout time.Duration) error {
	return card.NewError(card.ErrPCSCUnavailable, "platform not supported")
}

func (u *unsupportedBackend) CancelWait() error { return nil }

func (u *unsupportedBackend) Open(ctx context.Context, readerID string) (CardConnection, error) {
	return nil, card.NewError(card.ErrPCSCUnavailable, "platform not supported")
}

func (u *unsupportedBackend) Supported() error {
	return card.NewError(card.ErrPCSCUnavailable, "platform not supported")
}
