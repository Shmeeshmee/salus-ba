// Package wire is the envelope of a message pushed to a live client: {"type", "version", "payload"}. The same
// bytes travel on a Redis Stream and in a WebSocket frame, so a fan-out forwards them without decoding the
// payload. The package knows no message type; each sender names its own.
package wire

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
)

// ErrInvalid is returned for an envelope without a type, a version of at least 1, or a payload.
var ErrInvalid = errors.New("wire: invalid envelope")

// Envelope is a decoded message. Payload is left as JSON for the receiver of that type and version to decode.
type Envelope struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
	Version int             `json:"version"`
}

// Encode marshals payload into an envelope of type typ and version. typ is lower case letters and
// underscores, so it needs no escaping; the envelope is written by hand around the marshalled payload.
func Encode(typ string, version int, payload any) ([]byte, error) {
	if !validType(typ) || version < 1 {
		return nil, ErrInvalid
	}
	p, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("wire: marshal %s payload: %w", typ, err)
	}
	if string(p) == "null" {
		return nil, ErrInvalid
	}
	b := make([]byte, 0, len(`{"type":"","version":,"payload":}`)+len(typ)+4+len(p))
	b = append(b, `{"type":"`...)
	b = append(b, typ...)
	b = append(b, `","version":`...)
	b = strconv.AppendInt(b, int64(version), 10)
	b = append(b, `,"payload":`...)
	b = append(b, p...)
	return append(b, '}'), nil
}

// Decode parses an envelope and checks its type, version and payload are present; it leaves the payload as
// it is.
func Decode(b []byte) (Envelope, error) {
	var e Envelope
	if err := json.Unmarshal(b, &e); err != nil {
		return Envelope{}, fmt.Errorf("%w: %w", ErrInvalid, err)
	}
	if !validType(e.Type) || e.Version < 1 || len(e.Payload) == 0 || string(e.Payload) == "null" {
		return Envelope{}, ErrInvalid
	}
	return e, nil
}

func validType(typ string) bool {
	if typ == "" {
		return false
	}
	for _, r := range typ {
		if (r < 'a' || r > 'z') && r != '_' {
			return false
		}
	}
	return true
}
