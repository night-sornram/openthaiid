package card

import "fmt"

// Code is a stable machine-readable error code.
type Code string

const (
	// Connection
	ErrBridgeUnavailable Code = "BRIDGE_UNAVAILABLE"
	ErrConnectionFailed  Code = "CONNECTION_FAILED"
	ErrProtocolMismatch  Code = "PROTOCOL_MISMATCH"

	// Authorization
	ErrApprovalRequired Code = "APPROVAL_REQUIRED"
	ErrApprovalDenied   Code = "APPROVAL_DENIED"
	ErrApprovalTimeout  Code = "APPROVAL_TIMEOUT"
	ErrAuthExpired      Code = "AUTH_EXPIRED"
	ErrFieldNotAllowed  Code = "FIELD_NOT_ALLOWED"

	// Hardware
	ErrPCSCUnavailable         Code = "PCSC_UNAVAILABLE"
	ErrNoReader                Code = "NO_READER"
	ErrReaderSelectionRequired Code = "READER_SELECTION_REQUIRED"
	ErrNoCard                  Code = "NO_CARD"
	ErrCardRemoved             Code = "CARD_REMOVED"
	ErrReaderRemoved           Code = "READER_REMOVED"
	ErrBusy                    Code = "BUSY"

	// Reading
	ErrUnsupportedCard Code = "UNSUPPORTED_CARD"
	ErrReadFailed      Code = "READ_FAILED"
	ErrReadTimeout     Code = "READ_TIMEOUT"
	ErrCancelled       Code = "CANCELLED"
	ErrBackendStalled  Code = "BACKEND_STALLED"

	// Validation
	ErrInvalidRequest   Code = "INVALID_REQUEST"
	ErrUnsupportedField Code = "UNSUPPORTED_FIELD"
	ErrRateLimited      Code = "RATE_LIMITED"
)

// Error is a protocol-safe error. Messages never carry card field contents,
// raw APDUs, or native error dumps.
type Error struct {
	Code      Code   `json:"code"`
	Message   string `json:"message"`
	Retryable bool   `json:"retryable"`
	RequestID *int64 `json:"requestId,omitempty"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// NewError constructs an Error with a retryability derived from the code.
func NewError(code Code, message string) *Error {
	return &Error{
		Code:      code,
		Message:   message,
		Retryable: Retryable(code),
	}
}

// WithRequestID returns a copy bound to a protocol request id.
func (e *Error) WithRequestID(id int64) *Error {
	cp := *e
	cp.RequestID = &id
	return &cp
}

// Retryable classifies codes the SDK may surface as retryable.
func Retryable(code Code) bool {
	switch code {
	case ErrBridgeUnavailable, ErrConnectionFailed, ErrBusy,
		ErrReadTimeout, ErrPCSCUnavailable, ErrRateLimited:
		return true
	default:
		return false
	}
}

// AsError coerces err into *Error, wrapping unknowns as READ_FAILED.
func AsError(err error) *Error {
	if err == nil {
		return nil
	}
	if e, ok := err.(*Error); ok {
		return e
	}
	return NewError(ErrReadFailed, "operation failed")
}
