// Package auth is the identity context's domain: accounts, passwords, and (from package 3) sessions and
// MFA (docs/accounts/SPEC.md). It knows nothing about the directory beyond the opaque ids an account holds.
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

type accountIDKey struct{}

// WithAccountID returns ctx carrying the id of the signed-in account, as the session check sets it for a
// request.
func WithAccountID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, accountIDKey{}, id)
}

// AccountIDFrom returns the signed-in account's id in ctx, or "" when nobody is signed in.
func AccountIDFrom(ctx context.Context) string {
	id, _ := ctx.Value(accountIDKey{}).(string)
	return id
}
