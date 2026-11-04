package auth_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/Shmeeshmee/salus/internal/auth"
)

var testKey = bytes.Repeat([]byte{7}, 32) // synthetic

func TestAESSecretBox_OpensWhatItSealedForTheSameAccountOnly(t *testing.T) {
	box, err := auth.NewAESSecretBox(testKey)
	if err != nil {
		t.Fatal(err)
	}
	secret := []byte("twenty bytes secret!")

	sealed, err := box.Seal(secret, accountID)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(sealed, secret) {
		t.Error("the sealed secret holds the plaintext")
	}
	again, _ := box.Seal(secret, accountID)
	if bytes.Equal(sealed, again) {
		t.Error("two seals of the same secret are equal: the nonce is not fresh")
	}
	got, err := box.Open(sealed, accountID)
	if err != nil || !bytes.Equal(got, secret) {
		t.Errorf("Open() = %q, %v; want the secret", got, err)
	}

	tampered := bytes.Clone(sealed)
	tampered[len(tampered)-1] ^= 1
	for name, open := range map[string]func() ([]byte, error){
		"another account": func() ([]byte, error) { return box.Open(sealed, "another-account") },
		"tampered":        func() ([]byte, error) { return box.Open(tampered, accountID) },
		"too short":       func() ([]byte, error) { return box.Open(sealed[:5], accountID) },
	} {
		if _, err := open(); !errors.Is(err, auth.ErrSealedSecret) {
			t.Errorf("Open() of %s: error = %v, want ErrSealedSecret", name, err)
		}
	}
}

func TestNewAESSecretBox_RejectsAKeyThatIsNot32Bytes(t *testing.T) {
	for _, n := range []int{0, 16, 31, 33} {
		if _, err := auth.NewAESSecretBox(make([]byte, n)); err == nil {
			t.Errorf("NewAESSecretBox() with a %d-byte key: error = nil", n)
		}
	}
}
