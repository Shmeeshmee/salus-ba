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
	PractitionerID string // the directory's practitioner of a practitioner account; empty otherwise
	PasswordHash   string // argon2id PHC string; never leaves the domain
	// Language is the user's own choice of language, a BCP 47 tag of the enum language (docs/i18n/SPEC.md §3);
	// empty while they follow their browser or device.
	Language string
	// DetectedLanguage is the language the client matched at the last registration or sign-in; empty before.
	DetectedLanguage string
	Role             Role
}

// Member is what a staff session is in its clinic: whether its account is the admin, and the practitioner it
// is (empty for none).
type Member struct {
	PractitionerID string
	Admin          bool
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

// Invite asks someone to join a clinic as an admin or a practitioner (docs/accounts/SPEC.md "Invites"). The
// token of its link is issued by the invite_email job and never stored, only its SHA-256.
type Invite struct {
	CreatedAt      time.Time
	ExpiresAt      time.Time // 72 h after it was made or last replaced
	AcceptedAt     time.Time // zero while open
	ID             string
	OrganizationID string
	InvitedBy      string // the admin's account; empty when the operator command invited
	Email          string // trimmed and lower-cased: the login the new account gets
	GivenName      string // a practitioner's one HumanName; empty for an admin
	FamilyName     string
	Role           Role // RoleAdmin or RolePractitioner
}

// open reports whether the invite can still be accepted at now.
func (inv Invite) open(now time.Time) bool {
	return inv.AcceptedAt.IsZero() && now.Before(inv.ExpiresAt)
}

// Acceptance is an accepted invite: the new account, its TOTP secret (pending until a first code confirms
// it), and the challenge for POST /auth/mfa/confirm, which also opens the first session.
type Acceptance struct {
	Enrolment Enrolment
	Account   Account
	Challenge string
}

// InviteEmailArgs is the invite_email job, on the notifier's "email" queue. Its arguments are opaque ids
// only; the job loads the rest.
type InviteEmailArgs struct {
	InviteID       string `json:"invite_id"`
	OrganizationID string `json:"organization_id"`
}

// Kind names the job in river_job.
func (InviteEmailArgs) Kind() string { return "invite_email" }

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
