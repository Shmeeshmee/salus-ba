package redistest_test

import (
	"os"
	"os/exec"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/pkg/redis/redistest"
)

func TestNewClient_GivesEachTestAnEmptyDatabaseOfItsOwn(t *testing.T) {
	t.Parallel()
	a, b := redistest.NewClient(t), redistest.NewClient(t)

	if a.Options().DB == b.Options().DB {
		t.Fatalf("both clients use database %d", a.Options().DB)
	}
	if err := a.Set(t.Context(), "k", "a", time.Minute).Err(); err != nil {
		t.Fatal(err)
	}
	if n, err := b.Exists(t.Context(), "k").Result(); err != nil || n != 0 {
		t.Fatalf("key written through one client is visible through the other (n=%d, err=%v)", n, err)
	}
	if n, err := a.DBSize(t.Context()).Result(); err != nil || n != 1 {
		t.Fatalf("DBSize = %d, %v; want 1: a new database must start empty", n, err)
	}
}

// TestNewClient_FailsClearlyWithoutTestRedisURL runs one test of this binary again without TEST_REDIS_URL:
// it must fail, never skip, and say how to run the tests.
func TestNewClient_FailsClearlyWithoutTestRedisURL(t *testing.T) {
	t.Parallel()
	cmd := exec.Command(os.Args[0], "-test.run=^TestNewClient_GivesEachTestAnEmptyDatabaseOfItsOwn$") //nolint:gosec // re-runs this test binary
	cmd.Env = slices.DeleteFunc(os.Environ(), func(kv string) bool {
		return strings.HasPrefix(kv, "TEST_REDIS_URL=")
	})

	out, err := cmd.CombinedOutput()

	if err == nil {
		t.Fatalf("test binary exited 0 without TEST_REDIS_URL; output:\n%s", out)
	}
	if !strings.Contains(string(out), "TEST_REDIS_URL is not set") || !strings.Contains(string(out), "make test") {
		t.Errorf("output does not explain the missing TEST_REDIS_URL:\n%s", out)
	}
}
