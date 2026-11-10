package auth

import (
	"context"
	"time"
)

// Repository stores accounts.
type Repository interface {
	// Create stores a and returns its new id, or ErrEmailTaken when the e-mail has an account. It leaves the
	// transaction usable either way, so the failure can be audited in it.
	Create(ctx context.Context, a Account) (string, error)
	// ByEmail returns the account with the normalised e-mail, or ErrNotFound.
	ByEmail(ctx context.Context, email string) (Account, error)
	// ByID returns the account, or ErrNotFound (also for an id that is not a UUID).
	ByID(ctx context.Context, id string) (Account, error)
	// SetPasswordHash replaces the account's password hash.
	SetPasswordHash(ctx context.Context, id, hash string) error
	// SetLanguage sets the account's chosen language, or clears it for "", or returns ErrNotFound.
	SetLanguage(ctx context.Context, id, language string) error
	// SetDetectedLanguage sets the language the client matched, or returns ErrNotFound.
	SetDetectedLanguage(ctx context.Context, id, language string) error

	// TOTPSecret returns the account's sealed TOTP secret, or nil when there is none.
	TOTPSecret(ctx context.Context, id string) ([]byte, error)
	// SetTOTPSecret stores a sealed secret, pending until ConfirmMFA, in place of any earlier pending one.
	SetTOTPSecret(ctx context.Context, id string, sealed []byte) error
	// ConfirmMFA marks the account enrolled at at, with step as its last used TOTP step, or returns
	// ErrAlreadyEnrolled when it is enrolled.
	ConfirmMFA(ctx context.Context, id string, step int64, at time.Time) error
	// UseTOTPStep records step as the last used one and reports whether it was newer than the last; false for
	// an account that does not exist.
	UseTOTPStep(ctx context.Context, id string, step int64) (bool, error)

	// CreateSession stores s under the token's hash and returns its id.
	CreateSession(ctx context.Context, s Session, tokenHash []byte, at time.Time) (string, error)
	// SessionByTokenHash returns the session with the token hash that is neither revoked nor expired at now,
	// or ErrNotFound.
	SessionByTokenHash(ctx context.Context, tokenHash []byte, now time.Time) (Session, error)
	// RevokeSession ends the session at at, or returns ErrNotFound when it is not open.
	RevokeSession(ctx context.Context, id string, at time.Time) error
	// RevokeSessions ends every open session of the account at at.
	RevokeSessions(ctx context.Context, accountID string, at time.Time) error
	// ResetMFA clears the account's TOTP secret, its last used step and its enrolment.
	ResetMFA(ctx context.Context, id string) error
	// SetPractitioner links the account to its Practitioner.
	SetPractitioner(ctx context.Context, id, practitionerID string) error
	// SetTermReviewer grants (true) or revokes the account's reviewer permission of the term list; granting it
	// again or revoking it when there is none changes nothing. Only the owner's connection may (salusctl).
	SetTermReviewer(ctx context.Context, id string, reviewer bool) error

	// SetOrganization scopes the rest of the transaction in ctx to the clinic id: invites are row-level
	// secured.
	SetOrganization(ctx context.Context, id string) error
	// PutInvite stores inv as the open invite for its e-mail in its clinic, replacing an open one there (its
	// token too), and returns its id.
	PutInvite(ctx context.Context, inv Invite) (string, error)
	// OpenInvites returns the clinic's invites that are neither accepted nor expired at now, oldest first.
	OpenInvites(ctx context.Context, org string, now time.Time) ([]Invite, error)
	// InviteByID returns the invite, or ErrInvalidInvite.
	InviteByID(ctx context.Context, id string) (Invite, error)
	// SetInviteToken replaces the invite's token hash.
	SetInviteToken(ctx context.Context, id string, tokenHash []byte) error
	// InviteOrganization returns the clinic of the invite with the token hash, before any clinic is set
	// (invite_organization, SECURITY DEFINER), or ErrInvalidInvite.
	InviteOrganization(ctx context.Context, tokenHash []byte) (string, error)
	// InviteByTokenHash returns the invite with the token hash in the clinic set, or ErrInvalidInvite.
	InviteByTokenHash(ctx context.Context, tokenHash []byte) (Invite, error)
	// AcceptInvite marks the invite accepted at at, or returns ErrInvalidInvite when it was accepted or
	// expired by then.
	AcceptInvite(ctx context.Context, id string, at time.Time) error

	// DeleteSessionsEndedBefore deletes the sessions revoked or expired before before, and counts them. Only
	// the retention role may (built on its pool in internal/worker).
	DeleteSessionsEndedBefore(ctx context.Context, before time.Time) (int64, error)
	// DeleteInvitesEndedBefore deletes the invites of every clinic accepted or expired before before, and
	// counts them. Only the retention role may: it bypasses row-level security.
	DeleteInvitesEndedBefore(ctx context.Context, before time.Time) (int64, error)
}

// Jobs enqueues background jobs in the transaction in the context (River's InsertTx), so a job exists
// exactly when the write that causes it commits.
type Jobs interface {
	EnqueueInviteEmail(ctx context.Context, inviteID, organizationID string) error
}

// ChallengeStore keeps MFA challenges (Redis): the step between password and code, by the challenge's hash.
type ChallengeStore interface {
	// Put stores the challenge for accountID for ttl.
	Put(ctx context.Context, hash []byte, accountID string, ttl time.Duration) error
	// Get returns the challenge's account id, or ErrInvalidChallenge.
	Get(ctx context.Context, hash []byte) (string, error)
	// Fail counts a wrong code and deletes the challenge at the limit-th.
	Fail(ctx context.Context, hash []byte, limit int) error
	// Delete removes the challenge.
	Delete(ctx context.Context, hash []byte) error
}

// Counters are fixed-window counters (Redis), for the login rate limit.
type Counters interface {
	// Incr adds one to the counter key, starting a window of window when there is none, and returns the
	// count and how long the window has left.
	Incr(ctx context.Context, key string, window time.Duration) (int64, time.Duration, error)
	// Get returns the counter key and how long its window has left; zero for none.
	Get(ctx context.Context, key string) (int64, time.Duration, error)
}

// SecretBox seals TOTP secrets at rest, bound to their account (AESSecretBox; a KMS in production).
type SecretBox interface {
	Seal(plaintext []byte, accountID string) ([]byte, error)
	Open(sealed []byte, accountID string) ([]byte, error)
}

// Hasher hashes and verifies passwords (Argon2Hasher). Both wait for a free slot and give up when ctx ends.
type Hasher interface {
	Hash(ctx context.Context, password string) (string, error)
	// Verify reports whether password matches encoded and whether encoded uses older parameters.
	Verify(ctx context.Context, password, encoded string) (ok, stale bool, err error)
}
