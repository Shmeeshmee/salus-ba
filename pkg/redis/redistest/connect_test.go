package redistest

import (
	"strings"
	"testing"
)

func TestConnect_ReportsWhyTheServerIsUnusable(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name, raw, want string
	}{
		{"unset", "", "TEST_REDIS_URL is not set"},
		{"not a URL", "redis://[::1", "parse TEST_REDIS_URL"},
		{"not a Redis URL", "http://redis:6379/0", "parse TEST_REDIS_URL"},
		{"unreachable", "redis://127.0.0.1:1/0", "read the server's databases setting"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			admin, _, _, err := connect(tt.raw)

			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("connect(%q) error = %v, want one containing %q", tt.raw, err, tt.want)
			}
			if admin != nil {
				t.Errorf("connect(%q) returned a client along with the error", tt.raw)
			}
		})
	}
}
