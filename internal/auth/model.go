// Package auth is the identity context's domain: accounts, passwords, sessions and MFA
// (docs/accounts/SPEC.md). It knows nothing about the directory beyond the opaque ids an account holds.
package auth

import (
	"context"
	"time"
)

// Role is what an account may do: SPEC.md "Roles".
type Role string

// The roles.
const (
	RoleAdmin        Role = "admin"
	RolePractitioner Role = "practitioner"
	RolePatient      Role = "patient"
)

// Account is one person who can sign in. Staff (admin, practitioner) belong to exactly one clinic; a patient
// to none.
type Account struct {
	MFAEnrolledAt  time.Time // zero until MFA is set up
	ID             string
	Email          string // trimmed and lower-cased: the login
	OrganizationID string // the clinic of a staff account; empty for a patient
	PasswordHash   string // argon2id PHC string; never leaves the domain
	Role           Role
}

// MFAEnrolled reports whether the account has set up TOTP.
func (a Account) MFAEnrolled() bool { return !a.MFAEnrolledAt.IsZero() }

// Session is one sign-in. The token that identifies it is sent to the client once; only its SHA-256 is
// stored.
type Session struct {
	ExpiresAt      time.Time // absolute: 12 h for staff, 30 days for a patient
	ID             string
	AccountID      string
	OrganizationID string // the clinic of a staff session; empty for a patient's
	MFAVerified    bool   // the sign-in passed a TOTP code
}

// Next is the step a login asks for.
type Next string

// The steps.
const (
	NextDone   Next = "done"   // the session is open
	NextVerify Next = "verify" // send a TOTP code with the challenge (POST /auth/mfa/verify)
	NextEnrol  Next = "enrol"  // staff without MFA: enrol with the challenge (POST /auth/mfa/enrol, confirm)
)

// LoginResult is what a login step gives back: an open session, or a challenge for the next step.
type LoginResult struct {
	Token     string // the session token, when Next is NextDone
	Challenge string // when Next is NextVerify or NextEnrol
	Next      Next
	Session   Session // when Next is NextDone
}

// Enrolment is a new TOTP secret, pending until a first code confirms it.
type Enrolment struct {
	Secret string // base32, for manual entry
	URI    string // otpauth://totp/…, for a QR code
}

type sessionKey struct{}

// WithSession returns ctx carrying the signed-in session, as the session middleware sets it for a request.
func WithSession(ctx context.Context, s Session) context.Context {
	return context.WithValue(ctx, sessionKey{}, s)
}

// SessionFrom returns the signed-in session in ctx, and false when nobody is signed in.
func SessionFrom(ctx context.Context) (Session, bool) {
	s, ok := ctx.Value(sessionKey{}).(Session)
	return s, ok
}

// AccountIDFrom returns the signed-in account's id in ctx, or "" when nobody is signed in.
func AccountIDFrom(ctx context.Context) string {
	s, _ := SessionFrom(ctx)
	return s.AccountID
}
