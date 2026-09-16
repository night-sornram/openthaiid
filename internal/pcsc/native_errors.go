//go:build darwin || linux || windows

package pcsc

import (
	"github.com/ebfe/scard"
	"github.com/night-sornram/openthaiid/internal/card"
)

// mapNativeConnectError classifies PC/SC Connect failures without dumping
// native strings. A missing card at connect is NO_CARD; a pulled card or
// unplug during I/O is CARD_REMOVED / READER_REMOVED.
func mapNativeConnectError(err error) error {
	if err == nil {
		return nil
	}
	switch err {
	case scard.ErrNoSmartcard:
		return card.NewError(card.ErrNoCard, "no card present")
	case scard.ErrRemovedCard, scard.ErrResetCard:
		return card.NewError(card.ErrCardRemoved, "card removed")
	case scard.ErrUnknownReader, scard.ErrReaderUnavailable, scard.ErrNoReadersAvailable:
		return card.NewError(card.ErrReaderRemoved, "reader unplugged")
	default:
		return card.NewError(card.ErrReadFailed, "card connect failed")
	}
}

// mapNativeIOError classifies Transmit / BeginTransaction failures.
func mapNativeIOError(err error, fallbackMsg string) error {
	if err == nil {
		return nil
	}
	switch err {
	case scard.ErrNoSmartcard, scard.ErrRemovedCard, scard.ErrResetCard:
		return card.NewError(card.ErrCardRemoved, "card removed")
	case scard.ErrUnknownReader, scard.ErrReaderUnavailable, scard.ErrNoReadersAvailable:
		return card.NewError(card.ErrReaderRemoved, "reader unplugged")
	default:
		return card.NewError(card.ErrReadFailed, fallbackMsg)
	}
}
