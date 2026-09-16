//go:build darwin || linux || windows

package pcsc

import (
	"strings"
	"testing"

	"github.com/ebfe/scard"
	"github.com/night-sornram/openthaiid/internal/card"
)

func TestMapNativeConnectAndIOErrors(t *testing.T) {
	if got := card.AsError(mapNativeConnectError(scard.ErrNoSmartcard)); got.Code != card.ErrNoCard {
		t.Fatalf("connect no smartcard: %s", got.Code)
	}
	if got := card.AsError(mapNativeConnectError(scard.ErrRemovedCard)); got.Code != card.ErrCardRemoved {
		t.Fatalf("connect removed: %s", got.Code)
	}
	if got := card.AsError(mapNativeConnectError(scard.ErrUnknownReader)); got.Code != card.ErrReaderRemoved {
		t.Fatalf("connect unknown reader: %s", got.Code)
	}
	if got := card.AsError(mapNativeIOError(scard.ErrRemovedCard, "transmit failed")); got.Code != card.ErrCardRemoved {
		t.Fatalf("io removed: %s", got.Code)
	}
	if got := card.AsError(mapNativeIOError(scard.ErrNoReadersAvailable, "transmit failed")); got.Code != card.ErrReaderRemoved {
		t.Fatalf("io no readers: %s", got.Code)
	}
	if got := card.AsError(mapNativeIOError(scard.ErrNotTransacted, "transmit failed")); got.Code != card.ErrReadFailed {
		t.Fatalf("io other: %s", got.Code)
	}
	for _, err := range []error{
		mapNativeConnectError(scard.ErrRemovedCard),
		mapNativeIOError(scard.ErrResetCard, "transmit failed"),
		mapNativeIOError(scard.ErrReaderUnavailable, "transmit failed"),
	} {
		msg := err.Error()
		for _, leak := range []string{"1234567890121", "สมชาย", "Somchai", "laser", "NHSO"} {
			if strings.Contains(msg, leak) {
				t.Fatalf("native map leaked %q in %q", leak, msg)
			}
		}
	}
}
