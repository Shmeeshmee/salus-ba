// Package redistest gives each test its own empty Redis database on the server at TEST_REDIS_URL (the
// compose Redis when run through `make test`). Without TEST_REDIS_URL the test fails; it never skips.
//
// Logical databases 1 to N-1 are handed out round-robin through a counter in database 0, and each is
// flushed before and after its test. N is the server's `databases` setting (256 in compose), so up to N-1
// tests can hold a database at once across all test packages.
package redistest

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"sync"
	"testing"

	goredis "github.com/redis/go-redis/v9"
)

const counterKey = "redistest:next_db"

// server is set up once per test binary, on first use; afterwards it is only read.
var server struct {
	err       error
	admin     *goredis.Client
	url       *url.URL
	once      sync.Once
	databases int64
}

func setup() {
	server.admin, server.url, server.databases, server.err = connect(os.Getenv("TEST_REDIS_URL"))
}

// connect opens an admin client on database 0 of the server at raw and reads how many databases it has.
func connect(raw string) (*goredis.Client, *url.URL, int64, error) {
	if raw == "" {
		return nil, nil, 0, errors.New("TEST_REDIS_URL is not set; run the tests with `make test`, which points it at the compose Redis")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("parse TEST_REDIS_URL: %w", err)
	}
	opts, err := goredis.ParseURL(raw)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("parse TEST_REDIS_URL: %w", err)
	}
	opts.DB = 0
	admin := goredis.NewClient(opts)
	cfg, err := admin.ConfigGet(context.Background(), "databases").Result()
	if err != nil {
		_ = admin.Close()
		return nil, nil, 0, fmt.Errorf("read the server's databases setting: %w", err)
	}
	n, err := strconv.ParseInt(cfg["databases"], 10, 64)
	if err != nil || n < 2 {
		_ = admin.Close()
		return nil, nil, 0, fmt.Errorf("the server has %q databases; redistest needs at least 2", cfg["databases"])
	}
	return admin, u, n, nil
}

// URL returns the URL of an empty database for this test. It is flushed again when the test ends.
func URL(t testing.TB) string {
	t.Helper()
	server.once.Do(setup)
	if server.err != nil {
		t.Fatalf("redistest: %v", server.err)
	}
	ctx := context.Background()
	next, err := server.admin.Incr(ctx, counterKey).Result()
	if err != nil {
		t.Fatalf("redistest: allocate a database: %v", err)
	}
	db := 1 + (next-1)%(server.databases-1)
	u := *server.url
	u.Path = "/" + strconv.FormatInt(db, 10)

	flush := func() error {
		opts, err := goredis.ParseURL(u.String())
		if err != nil {
			return err
		}
		c := goredis.NewClient(opts)
		defer func() { _ = c.Close() }()
		return c.FlushDB(ctx).Err()
	}
	if err := flush(); err != nil {
		t.Fatalf("redistest: flush database %d: %v", db, err)
	}
	t.Cleanup(func() {
		if err := flush(); err != nil {
			t.Errorf("redistest: flush database %d: %v", db, err)
		}
	})
	return u.String()
}

// NewClient is URL with a client on the new database, closed when the test ends.
func NewClient(t testing.TB) *goredis.Client {
	t.Helper()
	opts, err := goredis.ParseURL(URL(t))
	if err != nil {
		t.Fatalf("redistest: %v", err)
	}
	c := goredis.NewClient(opts)
	t.Cleanup(func() { _ = c.Close() })
	return c
}
