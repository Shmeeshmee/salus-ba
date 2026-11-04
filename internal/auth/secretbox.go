package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
)

// aesKeyLen is AES-256's key length in bytes.
const aesKeyLen = 32

// AESSecretBox seals TOTP secrets with AES-256-GCM (SPEC.md "TOTP"). The account id is the additional data,
// so a sealed secret only opens for the account it was sealed for. Its key comes from the KMS in production
// (not decided yet) and from TOTP_KEY locally and in CI.
type AESSecretBox struct {
	aead cipher.AEAD
}

// NewAESSecretBox returns a box on key, which must be 32 bytes. The error never carries the key.
func NewAESSecretBox(key []byte) (*AESSecretBox, error) {
	if len(key) != aesKeyLen {
		return nil, errors.New("auth: the TOTP key must be 32 bytes")
	}
	block, _ := aes.NewCipher(key)  // fails only for a key length other than 16, 24 or 32
	aead, _ := cipher.NewGCM(block) // fails only for a block size other than 16
	return &AESSecretBox{aead: aead}, nil
}

// Seal returns nonce || ciphertext of plaintext, bound to accountID.
func (b *AESSecretBox) Seal(plaintext []byte, accountID string) ([]byte, error) {
	nonce := make([]byte, b.aead.NonceSize())
	_, _ = rand.Read(nonce) // crypto/rand.Read never returns an error
	return b.aead.Seal(nonce, nonce, plaintext, []byte(accountID)), nil
}

// Open returns the plaintext of sealed, or ErrSealedSecret when it was not sealed by this key for accountID.
func (b *AESSecretBox) Open(sealed []byte, accountID string) ([]byte, error) {
	n := b.aead.NonceSize()
	if len(sealed) < n {
		return nil, ErrSealedSecret
	}
	plaintext, err := b.aead.Open(nil, sealed[:n], sealed[n:], []byte(accountID))
	if err != nil {
		return nil, ErrSealedSecret
	}
	return plaintext, nil
}
