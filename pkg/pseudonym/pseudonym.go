// Package pseudonym keys an id to a pseudonym: HMAC-SHA256 of the id under a key. With a key that lives only as
// long as one computation (an aggregate build), the same id gives the same pseudonym within it, and nothing
// links a pseudonym back once the key is gone.
package pseudonym

import (
	"crypto/hmac"
	"crypto/sha256"
)

// Of returns HMAC-SHA256 of id under key.
func Of(key []byte, id string) [32]byte {
	m := hmac.New(sha256.New, key)
	m.Write([]byte(id))
	var out [32]byte
	copy(out[:], m.Sum(nil))
	return out
}
