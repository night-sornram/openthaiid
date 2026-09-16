// Package protocol defines versioned browser messages.
package protocol

import (
	"encoding/json"

	"github.com/night-sornram/openthaiid/internal/card"
)

// Major is the protocol major version.
const Major = 1

// Minor is the protocol minor version. Additive field expansions bump minor.
const Minor = 1

// Subprotocol is the WebSocket subprotocol name.
const Subprotocol = "openthaiid.v1"

// Type identifies a message kind.
type Type string

const (
	// Client operations
	TypeSessionOpen   Type = "session.open"
	TypeSessionRenew  Type = "session.renew"
	TypeSessionClose  Type = "session.close"
	TypeBridgeStatus  Type = "bridge.status"
	TypeCardRead      Type = "card.read"
	TypeRequestCancel Type = "request.cancel"

	// Server events
	TypeApprovalRequired   Type = "approval.required"
	TypeStatusSnapshot     Type = "status.snapshot"
	TypeReaderConnected    Type = "reader.connected"
	TypeReaderDisconnected Type = "reader.disconnected"
	TypeCardInserted       Type = "card.inserted"
	TypeCardRemoved        Type = "card.removed"
	TypeAuthRevoked        Type = "authorization.revoked"
)

// Request is a client → server envelope.
type Request struct {
	V       int             `json:"v"`
	ID      int64           `json:"id"`
	Type    Type            `json:"type"`
	Token   string          `json:"token,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// Response is a server → client reply bound to a request id.
type Response struct {
	V      int             `json:"v"`
	ID     int64           `json:"id"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *card.Error     `json:"error,omitempty"`
}

// Event is a server → client unsolicited message.
type Event struct {
	V          int             `json:"v"`
	Type       Type            `json:"type"`
	InstanceID string          `json:"instanceId"`
	Sequence   uint64          `json:"sequence"`
	Payload    json.RawMessage `json:"payload,omitempty"`
}

// SessionOpenPayload is the session.open payload.
type SessionOpenPayload struct {
	Fields []card.Field `json:"fields"`
}

// SessionOpenResult is returned after authentication.
type SessionOpenResult struct {
	Token           string       `json:"token"`
	ExpiresAt       string       `json:"expiresAt"`
	Fields          []card.Field `json:"fields"`
	InstanceID      string       `json:"instanceId"`
	Protocol        ProtocolInfo `json:"protocol"`
	FieldsSupported []card.Field `json:"fieldsSupported"`
	// ApprovalID is set when the result is pairing-only (APPROVAL_REQUIRED).
	ApprovalID *int64 `json:"approvalId,omitempty"`
	Comparison string `json:"comparisonCode,omitempty"`
}

// ProtocolInfo advertises version and capabilities.
type ProtocolInfo struct {
	Major int `json:"major"`
	Minor int `json:"minor"`
}

// CardReadPayload is the card.read payload.
type CardReadPayload struct {
	ReaderID      string       `json:"readerId"`
	CardSessionID string       `json:"cardSessionId"`
	Fields        []card.Field `json:"fields"`
}

// CancelPayload cancels an in-flight request id.
type CancelPayload struct {
	RequestID int64 `json:"requestId"`
}

// StatusSnapshot is delivered as an event or bridge.status result.
type StatusSnapshot struct {
	Connection string       `json:"connection"`
	Backend    string       `json:"backend"`
	Readers    []ReaderView `json:"readers"`
}

// ReaderView is the public reader state for the protocol.
type ReaderView struct {
	ID            string `json:"readerId"`
	Name          string `json:"name"`
	Presence      string `json:"presence"`
	CardSessionID string `json:"cardSessionId,omitempty"`
}

// ApprovalRequiredPayload is the approval.required event body.
type ApprovalRequiredPayload struct {
	ApprovalID     int64        `json:"approvalId"`
	Fields         []card.Field `json:"fields"`
	ComparisonCode string       `json:"comparisonCode"`
	ExpiresAt      string       `json:"expiresAt"`
}

// ValidateRequest checks envelope basics.
func ValidateRequest(r Request) error {
	if r.V != Major {
		return card.NewError(card.ErrProtocolMismatch, "unsupported protocol version")
	}
	if r.ID <= 0 {
		return card.NewError(card.ErrInvalidRequest, "id must be positive")
	}
	switch r.Type {
	case TypeSessionOpen, TypeSessionRenew, TypeSessionClose,
		TypeBridgeStatus, TypeCardRead, TypeRequestCancel:
		return nil
	default:
		return card.NewError(card.ErrInvalidRequest, "unknown operation")
	}
}

// MarshalEvent builds an event envelope.
func MarshalEvent(seq uint64, instanceID string, t Type, payload any) ([]byte, error) {
	var raw json.RawMessage
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		raw = b
	}
	return json.Marshal(Event{
		V:          Major,
		Type:       t,
		InstanceID: instanceID,
		Sequence:   seq,
		Payload:    raw,
	})
}
