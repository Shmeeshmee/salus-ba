package auth_test

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/auth"
)

const examplePassword = "correct horse battery"

// cheap are parameters far below the floor, for tests that only need two different settings.
var cheap = auth.Argon2Params{Memory: 64, Time: 1, Threads: 1}

func TestDefaultArgon2Params_AreAtLeastTheOWASPFloor(t *testing.T) {
	p := auth.DefaultArgon2Params()
	if p.Memory < 19*1024 || p.Time < 2 || p.Threads != 1 {
		t.Errorf("default %+v, want m >= 19 MiB, t >= 2, p = 1", p)
	}
}

func TestHash_IsAPHCStringWithTheParameters(t *testing.T) {
	h := auth.NewArgon2Hasher(auth.DefaultArgon2Params(), 1)
	got, err := h.Hash(t.Context(), examplePassword)
	if err != nil {
		t.Fatal(err)
	}
	p := auth.DefaultArgon2Params()
	want := regexp.MustCompile(fmt.Sprintf(`^\$argon2id\$v=19\$m=%d,t=%d,p=%d\$[A-Za-z0-9+/]{22}\$[A-Za-z0-9+/]{43}$`, p.Memory, p.Time, p.Threads))
	if !want.MatchString(got) {
		t.Errorf("Hash() = %q, want %s (16-byte salt, 32-byte key)", got, want)
	}
	if strings.Contains(got, examplePassword) {
		t.Error("the hash contains the password")
	}
}

func TestHash_SaltsEveryHash(t *testing.T) {
	h := auth.NewArgon2Hasher(cheap, 1)
	a, _ := h.Hash(t.Context(), examplePassword)
	b, _ := h.Hash(t.Context(), examplePassword)
	if a == b {
		t.Error("two hashes of one password are equal")
	}
}

func TestVerify(t *testing.T) {
	h := auth.NewArgon2Hasher(cheap, 1)
	encoded, err := h.Hash(t.Context(), examplePassword)
	if err != nil {
		t.Fatal(err)
	}
	stronger := auth.NewArgon2Hasher(auth.Argon2Params{Memory: 128, Time: 2, Threads: 1}, 1)
	tests := []struct {
		hasher    *auth.Argon2Hasher
		name      string
		password  string
		ok, stale bool
	}{
		{name: "right password", hasher: h, password: examplePassword, ok: true},
		{name: "wrong password", hasher: h, password: examplePassword + "!"},
		{name: "older parameters", hasher: stronger, password: examplePassword, ok: true, stale: true},
		{name: "older parameters, wrong password", hasher: stronger, password: "nope"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, stale, err := tt.hasher.Verify(t.Context(), tt.password, encoded)
			if err != nil || ok != tt.ok || stale != tt.stale {
				t.Errorf("Verify() = %v, %v, %v; want %v, %v, nil", ok, stale, err, tt.ok, tt.stale)
			}
		})
	}
}

func TestVerify_RejectsAMalformedHash(t *testing.T) {
	h := auth.NewArgon2Hasher(cheap, 1)
	for _, encoded := range []string{
		"",
		"plain",
		"$argon2i$v=19$m=64,t=1,p=1$c2FsdHNhbHRzYWx0c2FsdA$aGFzaA",
		"$argon2id$v=18$m=64,t=1,p=1$c2FsdHNhbHRzYWx0c2FsdA$aGFzaA",
		"$argon2id$v=19$m=x,t=1,p=1$c2FsdHNhbHRzYWx0c2FsdA$aGFzaA",
		"$argon2id$v=19$m=64,t=0,p=1$c2FsdHNhbHRzYWx0c2FsdA$aGFzaA",
		"$argon2id$v=19$m=64,t=1,p=1$!!$aGFzaA",
		"$argon2id$v=19$m=64,t=1,p=1$c2FsdHNhbHRzYWx0c2FsdA$!!",
		"$argon2id$v=19$m=64,t=1,p=1$c2FsdHNhbHRzYWx0c2FsdA$",
	} {
		ok, _, err := h.Verify(t.Context(), examplePassword, encoded)
		if ok || !errors.Is(err, auth.ErrMalformedHash) {
			t.Errorf("Verify(%q) = %v, %v; want false, ErrMalformedHash", encoded, ok, err)
		}
	}
}

func TestHasher_WaitsForASlotOnlyWhileTheContextLives(t *testing.T) {
	h := auth.NewArgon2Hasher(cheap, 1)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := h.Hash(ctx, examplePassword); !errors.Is(err, context.Canceled) {
		t.Errorf("Hash() with a cancelled context = %v, want context.Canceled", err)
	}
	if _, _, err := h.Verify(ctx, examplePassword, "$argon2id$v=19$m=64,t=1,p=1$c2FsdHNhbHRzYWx0c2FsdA$aGFzaA"); !errors.Is(err, context.Canceled) {
		t.Errorf("Verify() with a cancelled context = %v, want context.Canceled", err)
	}
}

func TestHasher_BoundsConcurrentHashes(t *testing.T) {
	// One slot, held by a slow hash: a second hash waits for it and gives up with its context.
	h := auth.NewArgon2Hasher(auth.Argon2Params{Memory: 32 * 1024, Time: 8, Threads: 1}, 1)
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = h.Hash(context.Background(), examplePassword)
	}()
	time.Sleep(20 * time.Millisecond) // let the first hash take the slot
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
	defer cancel()
	if _, err := h.Hash(ctx, examplePassword); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Hash() while the only slot is taken = %v, want context.DeadlineExceeded", err)
	}
	<-done
}

// BenchmarkHash measures one hash for each setting of the grid at p = 1 (docs/accounts/SPEC.md
// "Passwords"): run `go test -run '^$' -bench BenchmarkHash ./internal/auth`. The chosen setting and the
// numbers are in docs/accounts/PROGRESS.md.
func BenchmarkHash(b *testing.B) {
	for _, mib := range []uint32{19, 32, 46, 64} {
		for _, iterations := range []uint32{2, 3, 4} {
			p := auth.Argon2Params{Memory: mib * 1024, Time: iterations, Threads: 1}
			b.Run(fmt.Sprintf("m=%dMiB/t=%d", mib, iterations), func(b *testing.B) {
				h := auth.NewArgon2Hasher(p, 1)
				for b.Loop() {
					if _, err := h.Hash(b.Context(), examplePassword); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}

// BenchmarkHashParallel measures throughput of the default setting with several hashes at once, to size the
// semaphore: run with -cpu 1,2,4,8.
func BenchmarkHashParallel(b *testing.B) {
	h := auth.NewArgon2Hasher(auth.DefaultArgon2Params(), 64)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if _, err := h.Hash(context.Background(), examplePassword); err != nil {
				b.Error(err)
				return
			}
		}
	})
}
