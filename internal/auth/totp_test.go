package auth_test

import (
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/auth"
)

// TestTOTPCode_MatchesRFC6238 checks the SHA-1 test vectors of RFC 6238, appendix B. The RFC lists 8-digit
// codes; a 6-digit code is their last six digits.
func TestTOTPCode_MatchesRFC6238(t *testing.T) {
	secret := []byte("12345678901234567890")
	tests := []struct {
		want string
		unix int64
	}{
		{unix: 59, want: "287082"},          // 94287082
		{unix: 1111111109, want: "081804"},  // 07081804
		{unix: 1111111111, want: "050471"},  // 14050471
		{unix: 1234567890, want: "005924"},  // 89005924
		{unix: 2000000000, want: "279037"},  // 69279037
		{unix: 20000000000, want: "353130"}, // 65353130
	}
	for _, tt := range tests {
		if got := auth.TOTPCode(secret, time.Unix(tt.unix, 0)); got != tt.want {
			t.Errorf("TOTPCode(T = %d) = %s, want %s", tt.unix, got, tt.want)
		}
	}
}
