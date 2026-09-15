package card

import (
	"context"
)

// Transmitter is the minimal connection surface the engine needs.
// It is satisfied by pcsc.CardConnection.
type Transmitter interface {
	Transmit(ctx context.Context, command []byte, maxResp int) ([]byte, error)
}

// Engine executes a reviewed profile against an open card connection.
type Engine struct {
	Profile Profile
}

// NewEngine wraps a profile.
func NewEngine(p Profile) *Engine {
	return &Engine{Profile: p}
}

// Execute runs SELECT + field READ BINARY commands and parses the result.
// Only the requested fields are returned; incidental adjacent bytes are
// discarded after parsing.
func (e *Engine) Execute(ctx context.Context, conn Transmitter, fields []Field) (*FieldValues, []Warning, error) {
	fields, err := NormalizeFields(fields)
	if err != nil {
		return nil, nil, err
	}
	cmds, err := e.Profile.ReadCommands(fields)
	if err != nil {
		return nil, nil, err
	}
	responses := make([][]byte, 0, len(cmds))
	for _, c := range cmds {
		if err := ctx.Err(); err != nil {
			return nil, nil, NewError(ErrCancelled, "read cancelled")
		}
		body, err := e.transmit(ctx, conn, c)
		if err != nil {
			return nil, nil, err
		}
		responses = append(responses, body)
	}
	return e.Profile.Parse(fields, responses)
}

// transmit sends one APDU, handling 61xx GET RESPONSE when Retrieve is set.
func (e *Engine) transmit(ctx context.Context, conn Transmitter, a APDU) ([]byte, error) {
	raw, err := conn.Transmit(ctx, a.Command, a.MaxResponse+16)
	if err != nil {
		return nil, AsError(err)
	}
	sw, ok := SW(raw)
	if !ok {
		return nil, NewError(ErrReadFailed, "short response")
	}
	body := raw[:len(raw)-2]

	// 61xx: more data available via GET RESPONSE.
	if a.Retrieve && sw&0xFF00 == 0x6100 {
		le := int(sw & 0xFF)
		if le == 0 {
			le = 0xFF
		}
		gr := []byte{0x00, 0xC0, 0x00, 0x00, byte(le)}
		more, err := conn.Transmit(ctx, gr, le+2)
		if err != nil {
			return nil, AsError(err)
		}
		msw, ok := SW(more)
		if !ok {
			return nil, NewError(ErrReadFailed, "short get-response")
		}
		if !IsSWMatch(APDU{ExpectSW: []uint16{StatusOK}}, msw) {
			return nil, NewError(ErrReadFailed, "get-response status")
		}
		body = append(body, more[:len(more)-2]...)
	} else if !IsSWMatch(a, sw) {
		if sw == 0x6A82 || sw == 0x6A86 {
			return nil, NewError(ErrUnsupportedCard, "application not found")
		}
		return nil, NewError(ErrReadFailed, "unexpected status")
	}

	if a.MaxResponse > 0 && len(body) > a.MaxResponse {
		body = body[:a.MaxResponse]
	}
	return body, nil
}

// Execute is the convenience used by the reader manager.
func Execute(ctx context.Context, conn Transmitter, fields []Field) (*FieldValues, []Warning, error) {
	return NewEngine(DefaultProfile()).Execute(ctx, conn, fields)
}
