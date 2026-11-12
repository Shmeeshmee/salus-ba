package pseudonym_test

import (
	"testing"

	"github.com/Shmeeshmee/salus/pkg/pseudonym"
)

func TestOf_IsKeyedAndStable(t *testing.T) {
	a, b := []byte("synthetic key a"), []byte("synthetic key b")
	if pseudonym.Of(a, "x") != pseudonym.Of(a, "x") {
		t.Error("the same id and key give two pseudonyms")
	}
	if pseudonym.Of(a, "x") == pseudonym.Of(a, "y") || pseudonym.Of(a, "x") == pseudonym.Of(b, "x") {
		t.Error("another id or another key gives the same pseudonym")
	}
	// HMAC-SHA256("key", "The quick brown fox jumps over the lazy dog"), RFC 4231's style of check.
	got := pseudonym.Of([]byte("key"), "The quick brown fox jumps over the lazy dog")
	if want := "f7bc83f430538424b13298e6aa6fb143ef4d59a14946175997479dbc2d1a3cd8"; hexOf(got) != want {
		t.Errorf("Of = %s, want %s", hexOf(got), want)
	}
}

func hexOf(b [32]byte) string {
	const digits = "0123456789abcdef"
	out := make([]byte, 64)
	for i, v := range b {
		out[2*i], out[2*i+1] = digits[v>>4], digits[v&15]
	}
	return string(out)
}
