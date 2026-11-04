package auth

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/practitioner"
	"github.com/Shmeeshmee/salus/internal/practitionerrole"
)

const (
	inviteLifetime = 72 * time.Hour
	maxName        = 200 // characters, each of the given and the family name
)

// Invite makes an invite for in (e-mail, role and, for a practitioner, the given and family name) in the
// clinic of the admin signed in with sess, and enqueues its e-mail in the same transaction. An open invite
// for the same e-mail in the clinic is replaced. Anyone but an admin of the clinic gets ErrForbidden.
// Audited as a create of the invite, also when refused. Whether the e-mail already has an account is not
// checked here: an admin must not learn that someone, perhaps a patient, has one.
func (s *Service) Invite(ctx context.Context, sess Session, in Invite) (Invite, error) {
	in, err := validInvite(in)
	if err != nil {
		return Invite{}, err
	}
	var out Invite
	var refused bool
	err = s.tx.WithTx(ctx, func(ctx context.Context) error {
		admin, ok, err := s.clinicAdmin(ctx, sess)
		if err != nil {
			return err
		}
		if refused = !ok; refused {
			return s.audit.Record(ctx, inviteEvent(auditevent.CodeCreate, auditevent.ActionCreate, admin, "", auditevent.OutcomeMinorFailure))
		}
		in.OrganizationID, in.InvitedBy = admin.OrganizationID, admin.ID
		if out, err = s.putInvite(ctx, in); err != nil {
			return err
		}
		return s.audit.Record(ctx, inviteEvent(auditevent.CodeCreate, auditevent.ActionCreate, admin, out.ID, auditevent.OutcomeSuccess))
	})
	switch {
	case err != nil:
		return Invite{}, fmt.Errorf("auth: invite: %w", err)
	case refused:
		return Invite{}, ErrForbidden
	}
	return out, nil
}

// InviteAdmin invites the first admin of the clinic org. Only the operator command calls it (salusctl
// create-clinic); its audit event names the operator command as the agent.
func (s *Service) InviteAdmin(ctx context.Context, org, email string) (Invite, error) {
	in, err := validInvite(Invite{Email: email, Role: RoleAdmin})
	if err != nil {
		return Invite{}, err
	}
	in.OrganizationID = org
	var out Invite
	err = s.tx.WithTx(ctx, func(ctx context.Context) error {
		var err error
		if out, err = s.putInvite(ctx, in); err != nil {
			return err
		}
		e := inviteEvent(auditevent.CodeCreate, auditevent.ActionCreate, Account{OrganizationID: org}, out.ID, auditevent.OutcomeSuccess)
		e.AgentName = auditevent.AgentOperator
		return s.audit.Record(ctx, e)
	})
	if err != nil {
		return Invite{}, fmt.Errorf("auth: invite admin: %w", err)
	}
	return out, nil
}

// putInvite stores in, open for inviteLifetime, in its clinic and enqueues its e-mail, inside the caller's
// transaction.
func (s *Service) putInvite(ctx context.Context, in Invite) (Invite, error) {
	if err := s.repo.SetOrganization(ctx, in.OrganizationID); err != nil {
		return Invite{}, err
	}
	now := s.clock.Now()
	in.CreatedAt, in.ExpiresAt = now, now.Add(inviteLifetime)
	id, err := s.repo.PutInvite(ctx, in)
	if err != nil {
		return Invite{}, err
	}
	in.ID = id
	if err := s.jobs.EnqueueInviteEmail(ctx, in.ID, in.OrganizationID); err != nil {
		return Invite{}, err
	}
	return in, nil
}

// ListInvites returns the open invites of the clinic of the admin signed in with sess, oldest first (GET
// /invites). Anyone but an admin of the clinic gets ErrForbidden. Audited as a search: one event per invite
// returned, or one without an entity for none.
func (s *Service) ListInvites(ctx context.Context, sess Session) ([]Invite, error) {
	var invites []Invite
	var refused bool
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		admin, ok, err := s.clinicAdmin(ctx, sess)
		if err != nil {
			return err
		}
		if refused = !ok; refused {
			return s.audit.Record(ctx, inviteEvent(auditevent.CodeSearchType, auditevent.ActionExecute, admin, "", auditevent.OutcomeMinorFailure))
		}
		if err := s.repo.SetOrganization(ctx, admin.OrganizationID); err != nil {
			return err
		}
		if invites, err = s.repo.OpenInvites(ctx, admin.OrganizationID, s.clock.Now()); err != nil {
			return err
		}
		if len(invites) == 0 {
			return s.audit.Record(ctx, inviteEvent(auditevent.CodeSearchType, auditevent.ActionExecute, admin, "", auditevent.OutcomeSuccess))
		}
		for _, inv := range invites {
			if err := s.audit.Record(ctx, inviteEvent(auditevent.CodeSearchType, auditevent.ActionExecute, admin, inv.ID, auditevent.OutcomeSuccess)); err != nil {
				return err
			}
		}
		return nil
	})
	switch {
	case err != nil:
		return nil, fmt.Errorf("auth: list invites: %w", err)
	case refused:
		return nil, ErrForbidden
	}
	return invites, nil
}

// IsClinicAdmin reports whether the account signed in with sess is an admin of the session's clinic, as read
// from the database (the session does not hold the role), for the directory's FHIR handlers. Not audited: it
// is part of the request's authentication, and the directory access it is for is audited there.
func (s *Service) IsClinicAdmin(ctx context.Context, sess Session) (bool, error) {
	_, ok, err := s.clinicAdmin(ctx, sess)
	if err != nil {
		return false, fmt.Errorf("auth: clinic admin: %w", err)
	}
	return ok, nil
}

// clinicAdmin reads the account signed in with sess and reports whether it is an admin of the session's
// clinic. The returned account is the agent to audit: the session's when the account is gone.
func (s *Service) clinicAdmin(ctx context.Context, sess Session) (Account, bool, error) {
	a, err := s.repo.ByID(ctx, sess.AccountID)
	if errors.Is(err, ErrNotFound) {
		return Account{ID: sess.AccountID, OrganizationID: sess.OrganizationID}, false, nil
	}
	if err != nil {
		return Account{}, false, err
	}
	ok := a.Role == RoleAdmin && a.OrganizationID != "" && a.OrganizationID == sess.OrganizationID
	a.OrganizationID = sess.OrganizationID
	return a, ok, nil
}

// IssueInviteToken issues a new token for the open invite inviteID of the clinic org and returns it with the
// invite's e-mail, for the invite_email job. Only the token's SHA-256 is stored, replacing an earlier one,
// so a retried job makes the link of an earlier e-mail stop working (accepted by the owner, docs/accounts/
// PROGRESS.md). An invite that is unknown, accepted or expired gives ErrInvalidInvite. Not audited: a system
// step, not one of the accesses SPEC.md §2 lists.
func (s *Service) IssueInviteToken(ctx context.Context, inviteID, org string) (email, token string, err error) {
	token, hash := newToken()
	err = s.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.repo.SetOrganization(ctx, org); err != nil {
			return err
		}
		inv, err := s.repo.InviteByID(ctx, inviteID)
		if err != nil {
			return err
		}
		if !inv.open(s.clock.Now()) {
			return ErrInvalidInvite
		}
		email = inv.Email
		return s.repo.SetInviteToken(ctx, inv.ID, hash)
	})
	if err != nil {
		return "", "", fmt.Errorf("auth: issue invite token: %w", err)
	}
	return email, token, nil
}

// AcceptInvite accepts the invite of token with the new account's password, in one transaction: it creates
// the account in the invite's clinic with a pending TOTP secret, marks the invite accepted and, for a
// practitioner, creates the Practitioner and its PractitionerRole through their services and links the
// account to it. It returns the enrolment and a challenge: POST /auth/mfa/confirm with the first code
// finishes enrolment and opens the session. An unknown token gives ErrInvalidInvite (not audited: nothing
// was accessed); an expired or used invite ErrInvalidInvite and a taken e-mail ErrEmailTaken, both audited.
func (s *Service) AcceptInvite(ctx context.Context, token, password string) (Acceptance, error) {
	hash, ok := tokenHash(token)
	if !ok {
		return Acceptance{}, ErrInvalidInvite
	}
	if n := utf8.RuneCountInString(password); n < minPassword || n > maxPassword {
		return Acceptance{}, ErrInvalidPassword
	}
	pwHash, err := s.hasher.Hash(ctx, password) // outside the transaction: it holds no connection while hashing
	if err != nil {
		return Acceptance{}, fmt.Errorf("auth: accept invite: %w", err)
	}
	secret := make([]byte, totpSecretLen)
	_, _ = rand.Read(secret) // crypto/rand.Read never returns an error
	var a Account
	var refused error
	err = s.tx.WithTx(ctx, func(ctx context.Context) error {
		var err error
		a, refused, err = s.acceptInvite(ctx, hash, pwHash, secret)
		return err
	})
	switch {
	case err != nil:
		return Acceptance{}, fmt.Errorf("auth: accept invite: %w", err)
	case refused != nil:
		return Acceptance{}, refused
	}
	challenge, err := s.newChallenge(ctx, a.ID)
	if err != nil {
		return Acceptance{}, fmt.Errorf("auth: accept invite: %w", err)
	}
	a.PasswordHash = ""
	return Acceptance{Account: a, Challenge: challenge, Enrolment: newEnrolment(a.Email, secret)}, nil
}

// acceptInvite is AcceptInvite's transaction. refused is a refusal that is audited and commits.
func (s *Service) acceptInvite(ctx context.Context, hash []byte, pwHash string, secret []byte) (a Account, refused, err error) {
	org, err := s.repo.InviteOrganization(ctx, hash)
	if errors.Is(err, ErrInvalidInvite) {
		return Account{}, ErrInvalidInvite, nil
	}
	if err != nil {
		return Account{}, nil, err
	}
	if err := s.repo.SetOrganization(ctx, org); err != nil {
		return Account{}, nil, err
	}
	inv, err := s.repo.InviteByTokenHash(ctx, hash)
	if err != nil {
		return Account{}, nil, err
	}
	now := s.clock.Now()
	a = Account{Email: inv.Email, Role: inv.Role, OrganizationID: inv.OrganizationID, PasswordHash: pwHash}
	refused = ErrInvalidInvite
	if inv.open(now) {
		a.ID, err = s.repo.Create(ctx, a)
		refused = nil
		if errors.Is(err, ErrEmailTaken) {
			refused, err = ErrEmailTaken, nil
		}
	}
	if err != nil {
		return Account{}, nil, err
	}
	if refused != nil {
		no := Account{OrganizationID: inv.OrganizationID} // no account: no agent
		return Account{}, refused, s.audit.Record(ctx, inviteEvent(auditevent.CodeCreate, auditevent.ActionCreate, no, inv.ID, auditevent.OutcomeMinorFailure))
	}
	// A concurrent acceptance of the same invite fails here, and its account rolls back.
	if err := s.repo.AcceptInvite(ctx, inv.ID, now); err != nil {
		return Account{}, nil, err
	}
	if err := s.setUpAccount(ctx, a, inv, secret); err != nil {
		return Account{}, nil, err
	}
	return a, nil, s.audit.Record(ctx, event(restCoding, interaction(auditevent.CodeCreate), auditevent.ActionCreate, a, auditevent.OutcomeSuccess))
}

// setUpAccount stores the new account's pending TOTP secret and, for a practitioner, creates their
// Practitioner and PractitionerRole as the new account and links the account to it.
func (s *Service) setUpAccount(ctx context.Context, a Account, inv Invite, secret []byte) error {
	sealed, err := s.box.Seal(secret, a.ID)
	if err != nil {
		return err
	}
	if err := s.repo.SetTOTPSecret(ctx, a.ID, sealed); err != nil {
		return err
	}
	if a.Role != RolePractitioner {
		return nil
	}
	p, err := s.directory.Create(ctx, practitioner.Caller{AccountID: a.ID, OrganizationID: a.OrganizationID}, inv.GivenName, inv.FamilyName)
	if err != nil {
		return err
	}
	if _, err := s.roles.Create(ctx, practitionerrole.Caller{AccountID: a.ID, OrganizationID: a.OrganizationID}, p.ID); err != nil {
		return err
	}
	return s.repo.SetPractitioner(ctx, a.ID, p.ID)
}

// validInvite normalises in and checks it: a plain e-mail; an admin, or a practitioner with a given and a
// family name.
func validInvite(in Invite) (Invite, error) {
	in.Email = normaliseEmail(in.Email)
	in.GivenName, in.FamilyName = strings.TrimSpace(in.GivenName), strings.TrimSpace(in.FamilyName)
	if !validEmail(in.Email) {
		return Invite{}, ErrInvalidEmail
	}
	switch in.Role {
	case RoleAdmin:
		if in.GivenName != "" || in.FamilyName != "" {
			return Invite{}, ErrInvalidName
		}
	case RolePractitioner:
		if !validName(in.GivenName) || !validName(in.FamilyName) {
			return Invite{}, ErrInvalidName
		}
	default:
		return Invite{}, ErrInvalidRole
	}
	return in, nil
}

func validName(name string) bool {
	return name != "" && utf8.RuneCountInString(name) <= maxName
}

// inviteEvent is an access (interaction) by agent to the invite id ("" when none was accessed), in the
// agent's clinic, for operations.
func inviteEvent(code string, action auditevent.Action, agent Account, id string, outcome auditevent.Outcome) auditevent.AuditEvent {
	e := event(restCoding, interaction(code), action, agent, outcome)
	e.EntityReference = ""
	if id != "" {
		e.EntityReference = "Invite/" + id
	}
	return e
}
