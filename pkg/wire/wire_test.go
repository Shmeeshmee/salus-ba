package wire_test

import (
	"errors"
	"testing"

	"github.com/Shmeeshmee/salus/pkg/wire"
)

func TestEncode_WrapsThePayloadWithItsTypeAndVersion(t *testing.T) {
	t.Parallel()
	got, err := wire.Encode("transcript", 1, struct {
		Text string `json:"text"`
	}{Text: "placeholder"})
	if err != nil {
		t.Fatal(err)
	}
	if want := `{"type":"transcript","version":1,"payload":{"text":"placeholder"}}`; string(got) != want {
		t.Errorf("Encode() = %s, want %s", got, want)
	}
}

func TestEncode_RefusesAnEnvelopeWithoutATypeOrVersion(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name    string
		typ     string
		version int
	}{
		{name: "no type", version: 1},
		{name: "a type that would need escaping", typ: `a"b`, version: 1},
		{name: "an upper-case type", typ: "Transcript", version: 1},
		{name: "version 0", typ: "transcript"},
		{name: "negative version", typ: "transcript", version: -1},
	} {
		if _, err := wire.Encode(tt.typ, tt.version, struct{}{}); !errors.Is(err, wire.ErrInvalid) {
			t.Errorf("%s: err = %v, want ErrInvalid", tt.name, err)
		}
	}
}

// A payload that marshals to null makes an envelope Decode refuses, so Encode refuses it too.
func TestEncode_RefusesAPayloadThatMarshalsToNull(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		payload any
		name    string
	}{
		{name: "nil"},
		{name: "a nil pointer", payload: (*struct{})(nil)},
		{name: "a nil map", payload: map[string]string(nil)},
		{name: "a nil slice", payload: []string(nil)},
	} {
		if _, err := wire.Encode("transcript", 1, tt.payload); !errors.Is(err, wire.ErrInvalid) {
			t.Errorf("%s: err = %v, want ErrInvalid", tt.name, err)
		}
	}
}

func TestEncode_ReturnsThePayloadsMarshalError(t *testing.T) {
	t.Parallel()
	if _, err := wire.Encode("transcript", 1, make(chan int)); err == nil || errors.Is(err, wire.ErrInvalid) {
		t.Errorf("err = %v, want the marshal error", err)
	}
}

func TestDecode_ReturnsTheEnvelopeWithThePayloadUndecoded(t *testing.T) {
	t.Parallel()
	got, err := wire.Decode([]byte(`{"type":"draft","version":2,"payload":{"a":1}}`))
	if err != nil {
		t.Fatal(err)
	}
	if got.Type != "draft" || got.Version != 2 || string(got.Payload) != `{"a":1}` {
		t.Errorf("Decode() = %+v (payload %s)", got, got.Payload)
	}
}

func TestDecode_RefusesWhatIsNoEnvelope(t *testing.T) {
	t.Parallel()
	for _, in := range []string{
		``, `[]`, `{`,
		`{"version":1,"payload":{}}`,
		`{"type":"draft","payload":{}}`,
		`{"type":"draft","version":0,"payload":{}}`,
		`{"type":"draft","version":1}`,
		`{"type":"draft","version":1,"payload":null}`,
	} {
		if _, err := wire.Decode([]byte(in)); !errors.Is(err, wire.ErrInvalid) {
			t.Errorf("Decode(%s): err = %v, want ErrInvalid", in, err)
		}
	}
}

func TestEncodeThenDecode_RoundTrips(t *testing.T) {
	t.Parallel()
	b, err := wire.Encode("suggestion", 1, map[string]string{"text": "placeholder"})
	if err != nil {
		t.Fatal(err)
	}
	env, err := wire.Decode(b)
	if err != nil {
		t.Fatal(err)
	}
	if env.Type != "suggestion" || env.Version != 1 || string(env.Payload) != `{"text":"placeholder"}` {
		t.Errorf("round trip = %+v (payload %s)", env, env.Payload)
	}
}
