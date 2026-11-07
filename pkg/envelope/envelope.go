// Package envelope encrypts a file with a data key of its own and wraps that key with a key-encryption key
// (docs/attachments/SPEC.md §2 "Encryption").
//
// The plaintext is cut into segments of SegmentSize bytes, the last one shorter (and empty for an empty file).
// Each segment is sealed with AES-256-GCM under the data key; the nonce is the segment's index (big-endian,
// 96 bits), which is safe because a data key encrypts one file only; the additional data is the file's id, the
// index and a flag on the last segment. A segment therefore opens only at its own place in its own file, and a
// truncated or extended file does not open. Ciphertext segment i starts at i × (SegmentSize + Overhead), which
// lets OpenRange decrypt a range without reading the file from the start.
package envelope

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"errors"
)

const (
	// SegmentSize is the plaintext length of every segment but the last.
	SegmentSize = 64 << 10
	// Overhead is what sealing adds to each segment: the GCM tag.
	Overhead = 16
	// KeySize is the length of a data key and of a key-encryption key: AES-256.
	KeySize = 32

	cipherSegment = SegmentSize + Overhead
)

var (
	// ErrOpen means the ciphertext was altered, truncated, extended, or belongs to another key or file.
	ErrOpen = errors.New("envelope: the ciphertext does not open")
	// ErrUnwrap means the wrapped key was altered, or belongs to another key-encryption key or file.
	ErrUnwrap = errors.New("envelope: the data key does not unwrap")
	// ErrRange means the requested plaintext range is not inside the file.
	ErrRange = errors.New("envelope: the range is outside the file")
	// ErrKeySize means a key is not KeySize bytes.
	ErrKeySize = errors.New("envelope: a key must be 32 bytes")
)

// NewDataKey returns a random data key for one file.
func NewDataKey() ([]byte, error) {
	k := make([]byte, KeySize)
	if _, err := rand.Read(k); err != nil {
		return nil, err
	}
	return k, nil
}

func newGCM(key []byte) (cipher.AEAD, error) {
	if len(key) != KeySize {
		return nil, ErrKeySize
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

// segmentNonce is the segment index, big-endian, in 96 bits.
func segmentNonce(nonce []byte, index uint64) []byte {
	clear(nonce[:4])
	binary.BigEndian.PutUint64(nonce[4:], index)
	return nonce
}

// segmentAD is the file id, the segment index and the last-segment flag. The id comes first and the rest has a
// fixed length, so no two (id, index, flag) give the same bytes.
func segmentAD(id string, index uint64, last bool) []byte {
	ad := make([]byte, 0, len(id)+9)
	ad = append(ad, id...)
	ad = binary.BigEndian.AppendUint64(ad, index)
	if last {
		return append(ad, 1)
	}
	return append(ad, 0)
}

// KEK is a key-encryption key and its id, which is stored with every key it wraps.
type KEK struct {
	aead cipher.AEAD
	id   string
}

// NewKEK takes the 32-byte key-encryption key and its id.
func NewKEK(id string, key []byte) (*KEK, error) {
	aead, err := newGCM(key)
	if err != nil {
		return nil, err
	}
	return &KEK{aead: aead, id: id}, nil
}

// ID is the key-encryption key's id.
func (k *KEK) ID() string { return k.id }

// Wrap seals a data key with the file's id as additional data. The result is a random nonce followed by the
// sealed key; a key-encryption key wraps many keys, so its nonces are random, never counted.
func (k *KEK) Wrap(dek []byte, id string) ([]byte, error) {
	if len(dek) != KeySize {
		return nil, ErrKeySize
	}
	nonce := make([]byte, k.aead.NonceSize(), k.aead.NonceSize()+KeySize+Overhead)
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	return k.aead.Seal(nonce, nonce, dek, []byte(id)), nil
}

// Unwrap opens a key Wrap sealed for the same file.
func (k *KEK) Unwrap(wrapped []byte, id string) ([]byte, error) {
	n := k.aead.NonceSize()
	if len(wrapped) < n {
		return nil, ErrUnwrap
	}
	dek, err := k.aead.Open(nil, wrapped[:n], wrapped[n:], []byte(id))
	if err != nil || len(dek) != KeySize {
		return nil, ErrUnwrap
	}
	return dek, nil
}
