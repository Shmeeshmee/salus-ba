package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Argon2Params are argon2id's cost parameters.
type Argon2Params struct {
	Memory  uint32 // KiB
	Time    uint32 // iterations
	Threads uint8
}

// Lengths of the salt and the derived key, in bytes (docs/accounts/SPEC.md "Passwords").
const (
	saltLen = 16
	keyLen  = 32
)

// DefaultArgon2Params is the setting new hashes use: the cheapest setting at or above the OWASP floor
// (m = 19 MiB, t = 2, p = 1) that takes 50-100 ms, chosen with BenchmarkHash (docs/accounts/PROGRESS.md,
// package 2).
func DefaultArgon2Params() Argon2Params {
	return Argon2Params{Memory: 32 * 1024, Time: 4, Threads: 1}
}

// MaxConcurrentHashes bounds the hashes running at once, so a burst of logins uses at most
// MaxConcurrentHashes × Memory (4 × 32 MiB = 128 MiB) and queues the rest (BenchmarkHashParallel, PROGRESS.md).
const MaxConcurrentHashes = 4

// Argon2Hasher hashes and verifies passwords with argon2id, stored as PHC strings
// ($argon2id$v=19$m=…,t=…,p=…$salt$key) so the parameters can change. A semaphore bounds the hashes running
// at once.
type Argon2Hasher struct {
	slots  chan struct{}
	params Argon2Params
}

// NewArgon2Hasher returns a hasher that hashes with p and runs at most concurrent hashes at once.
func NewArgon2Hasher(p Argon2Params, concurrent int) *Argon2Hasher {
	return &Argon2Hasher{params: p, slots: make(chan struct{}, concurrent)}
}

// Hash returns the PHC string of password with a fresh random salt.
func (h *Argon2Hasher) Hash(ctx context.Context, password string) (string, error) {
	salt := make([]byte, saltLen)
	_, _ = rand.Read(salt) // crypto/rand.Read never returns an error
	key, err := h.derive(ctx, password, salt, h.params, keyLen)
	if err != nil {
		return "", err
	}
	return encode(h.params, salt, key), nil
}

// Verify reports whether password matches encoded, compared in constant time, and whether encoded was made
// with other parameters than the current ones (stale: re-hash it).
func (h *Argon2Hasher) Verify(ctx context.Context, password, encoded string) (ok, stale bool, err error) {
	p, salt, want, err := decode(encoded)
	if err != nil {
		return false, false, err
	}
	got, err := h.derive(ctx, password, salt, p, uint32(len(want)))
	if err != nil {
		return false, false, err
	}
	if subtle.ConstantTimeCompare(got, want) != 1 {
		return false, false, nil
	}
	return true, p != h.params || len(salt) != saltLen || len(want) != keyLen, nil
}

// derive runs argon2id once a slot is free, or gives up when ctx ends first.
func (h *Argon2Hasher) derive(ctx context.Context, password string, salt []byte, p Argon2Params, n uint32) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	select {
	case h.slots <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	defer func() { <-h.slots }()
	return argon2.IDKey([]byte(password), salt, p.Time, p.Memory, p.Threads, n), nil
}

var b64 = base64.RawStdEncoding

func encode(p Argon2Params, salt, key []byte) string {
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, p.Memory, p.Time, p.Threads, b64.EncodeToString(salt), b64.EncodeToString(key))
}

func decode(encoded string) (Argon2Params, []byte, []byte, error) {
	var p Argon2Params
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" || parts[2] != fmt.Sprintf("v=%d", argon2.Version) {
		return p, nil, nil, ErrMalformedHash
	}
	var tail string
	n, _ := fmt.Sscanf(parts[3]+"$", "m=%d,t=%d,p=%d%s", &p.Memory, &p.Time, &p.Threads, &tail)
	if n != 4 || tail != "$" || p.Memory == 0 || p.Time == 0 || p.Threads == 0 {
		return p, nil, nil, ErrMalformedHash
	}
	salt, err := b64.DecodeString(parts[4])
	if err != nil || len(salt) == 0 {
		return p, nil, nil, ErrMalformedHash
	}
	key, err := b64.DecodeString(parts[5])
	if err != nil || len(key) == 0 {
		return p, nil, nil, ErrMalformedHash
	}
	return p, salt, key, nil
}
