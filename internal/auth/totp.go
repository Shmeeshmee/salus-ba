package auth

import (
	"crypto/hmac"
	"crypto/sha1" // RFC 6238 TOTP is HMAC-SHA-1; authenticator apps expect it (SPEC.md "TOTP")
	"crypto/subtle"
	"encoding/binary"
	"fmt"
	"time"
)

// TOTP parameters (RFC 6238, SPEC.md "TOTP"): HMAC-SHA-1, 6 digits, 30-second steps, one step of drift
// either way.
const (
	totpStep      = 30 // seconds
	totpDigits    = 6
	totpModulo    = 1_000_000 // 10^totpDigits
	totpDrift     = 1         // steps
	totpSecretLen = 20        // bytes, the HMAC-SHA-1 output size (RFC 4226 section 4)
)

// TOTPCode returns the code for secret at time at.
func TOTPCode(secret []byte, at time.Time) string {
	return hotp(secret, totpStepAt(at))
}

func totpStepAt(at time.Time) int64 { return at.Unix() / totpStep }

// hotp is RFC 4226's HOTP value for counter, with dynamic truncation.
func hotp(secret []byte, counter int64) string {
	mac := hmac.New(sha1.New, secret)
	_ = binary.Write(mac, binary.BigEndian, counter) // a hash.Hash never returns an error
	sum := mac.Sum(nil)
	off := sum[len(sum)-1] & 0x0f
	v := binary.BigEndian.Uint32(sum[off:off+4]) & 0x7fffffff
	return fmt.Sprintf("%0*d", totpDigits, v%totpModulo)
}

// matchTOTP returns the step whose code is code, within one step of now's, comparing in constant time.
func matchTOTP(secret []byte, code string, now time.Time) (step int64, ok bool) {
	cur := totpStepAt(now)
	for s := cur - totpDrift; s <= cur+totpDrift; s++ {
		if subtle.ConstantTimeCompare([]byte(hotp(secret, s)), []byte(code)) == 1 {
			return s, true
		}
	}
	return 0, false
}
