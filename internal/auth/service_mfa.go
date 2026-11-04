package auth

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"errors"
	"fmt"
	"net/url"
	"strconv"

	"github.com/Shmeeshmee/salus/internal/auditevent"
)

// totpIssuer names the service in authenticator apps.
const totpIssuer = "Salus"

var base32NoPad = base32.StdEncoding.WithPadding(base32.NoPadding)

// Enrol starts TOTP enrolment for the account of a login challenge (staff at their first login) or, without
// one, of a signed-in session (a patient who opts in). It stores a new secret, sealed and pending until
// Confirm, and returns it. Audited as an update of the account; an enrolled account gives
// ErrAlreadyEnrolled.
func (s *Service) Enrol(ctx context.Context, challenge string, sess Session) (Enrolment, error) {
	id, _, err := s.mfaAccount(ctx, challenge, sess)
	if err != nil {
		return Enrolment{}, fmt.Errorf("auth: enrol: %w", err)
	}
	secret := make([]byte, totpSecretLen)
	_, _ = rand.Read(secret) // crypto/rand.Read never returns an error
	var e Enrolment
	var already bool
	err = s.tx.WithTx(ctx, func(ctx context.Context) error {
		a, err := s.repo.ByID(ctx, id)
		if err != nil {
			return err
		}
		if already = a.MFAEnrolled(); already {
			return s.recordUpdate(ctx, a, auditevent.OutcomeMinorFailure)
		}
		sealed, err := s.box.Seal(secret, a.ID)
		if err != nil {
			return err
		}
		if err := s.repo.SetTOTPSecret(ctx, a.ID, sealed); err != nil {
			return err
		}
		e = newEnrolment(a.Email, secret)
		return s.recordUpdate(ctx, a, auditevent.OutcomeSuccess)
	})
	switch {
	case err != nil:
		return Enrolment{}, fmt.Errorf("auth: enrol: %w", err)
	case already:
		return Enrolment{}, ErrAlreadyEnrolled
	}
	return e, nil
}

// Confirm checks the first code against the pending secret and marks the account enrolled. From a
// challenge it also opens the session, MFA-verified; from a session it returns an empty result. Audited as
// an update of the account. A wrong code counts against the challenge.
func (s *Service) Confirm(ctx context.Context, challenge string, sess Session, code string) (LoginResult, error) {
	id, hash, err := s.mfaAccount(ctx, challenge, sess)
	if err != nil {
		return LoginResult{}, fmt.Errorf("auth: confirm: %w", err)
	}
	var res LoginResult
	var refused error
	err = s.tx.WithTx(ctx, func(ctx context.Context) error {
		a, secret, err := s.accountSecret(ctx, id)
		if err != nil {
			return err
		}
		now := s.clock.Now()
		step, matched := matchTOTP(secret, code, now)
		switch {
		case a.MFAEnrolled():
			refused = ErrAlreadyEnrolled
		case secret == nil:
			refused = ErrNotEnrolled
		case !matched:
			refused = ErrWrongCode
		}
		if refused != nil {
			return s.recordUpdate(ctx, a, auditevent.OutcomeMinorFailure)
		}
		if err := s.repo.ConfirmMFA(ctx, a.ID, step, now); err != nil {
			return err
		}
		if hash != nil {
			res.Next = NextDone
			if res.Session, res.Token, err = s.openSession(ctx, a, true); err != nil {
				return err
			}
		}
		return s.recordUpdate(ctx, a, auditevent.OutcomeSuccess)
	})
	if err != nil {
		return LoginResult{}, fmt.Errorf("auth: confirm: %w", err)
	}
	return s.afterCode(ctx, hash, res, refused)
}

// Verify checks a code for the account of a login challenge and opens its session, MFA-verified. A code
// whose step was already used is wrong (account.totp_last_step). Audited as a login. A wrong code counts
// against the challenge; the fifth ends it.
func (s *Service) Verify(ctx context.Context, challenge, code string) (LoginResult, error) {
	hash, ok := tokenHash(challenge)
	if !ok {
		return LoginResult{}, ErrInvalidChallenge
	}
	id, err := s.challenges.Get(ctx, hash)
	if err != nil {
		return LoginResult{}, fmt.Errorf("auth: verify: %w", err)
	}
	var res LoginResult
	var refused error
	err = s.tx.WithTx(ctx, func(ctx context.Context) error {
		a, secret, err := s.accountSecret(ctx, id)
		if err != nil {
			return err
		}
		if refused, err = s.useCode(ctx, a, secret, code); err != nil {
			return err
		}
		if refused != nil {
			return s.recordLogin(ctx, a, auditevent.CodeLogin, auditevent.OutcomeMinorFailure)
		}
		res.Next = NextDone
		if res.Session, res.Token, err = s.openSession(ctx, a, true); err != nil {
			return err
		}
		return s.recordLogin(ctx, a, auditevent.CodeLogin, auditevent.OutcomeSuccess)
	})
	if err != nil {
		return LoginResult{}, fmt.Errorf("auth: verify: %w", err)
	}
	return s.afterCode(ctx, hash, res, refused)
}

// useCode checks code against an enrolled account's secret and records its step as used. refused is
// ErrNotEnrolled or ErrWrongCode when the code does not open the session.
func (s *Service) useCode(ctx context.Context, a Account, secret []byte, code string) (refused, err error) {
	if !a.MFAEnrolled() || secret == nil {
		return ErrNotEnrolled, nil
	}
	step, ok := matchTOTP(secret, code, s.clock.Now())
	if !ok {
		return ErrWrongCode, nil
	}
	fresh, err := s.repo.UseTOTPStep(ctx, a.ID, step)
	if err != nil || !fresh {
		return ErrWrongCode, err
	}
	return nil, nil
}

// mfaAccount returns the account a challenge acts for, with the challenge's hash, or without a challenge the
// session's account.
func (s *Service) mfaAccount(ctx context.Context, challenge string, sess Session) (string, []byte, error) {
	if challenge == "" {
		if sess.AccountID == "" {
			return "", nil, ErrUnauthenticated
		}
		return sess.AccountID, nil, nil
	}
	hash, ok := tokenHash(challenge)
	if !ok {
		return "", nil, ErrInvalidChallenge
	}
	id, err := s.challenges.Get(ctx, hash)
	return id, hash, err
}

// accountSecret reads the account and opens its TOTP secret; the secret is nil when none is stored.
func (s *Service) accountSecret(ctx context.Context, id string) (Account, []byte, error) {
	a, err := s.repo.ByID(ctx, id)
	if err != nil {
		return Account{}, nil, err
	}
	sealed, err := s.repo.TOTPSecret(ctx, id)
	if err != nil || sealed == nil {
		return a, nil, err
	}
	secret, err := s.box.Open(sealed, id)
	return a, secret, err
}

// afterCode ends a challenge that opened a session and counts a wrong code against it. Without a
// challenge (hash nil) there is nothing to do.
func (s *Service) afterCode(ctx context.Context, hash []byte, res LoginResult, refused error) (LoginResult, error) {
	var err error
	switch {
	case hash == nil:
	case refused == nil:
		err = s.challenges.Delete(ctx, hash)
	case errors.Is(refused, ErrWrongCode):
		err = s.challenges.Fail(ctx, hash, maxWrongCodes)
	}
	switch {
	case err != nil:
		return LoginResult{}, fmt.Errorf("auth: challenge: %w", err)
	case refused != nil:
		return LoginResult{}, refused
	}
	return res, nil
}

func (s *Service) recordUpdate(ctx context.Context, a Account, outcome auditevent.Outcome) error {
	return s.audit.Record(ctx, event(restCoding, interaction(auditevent.CodeUpdate), auditevent.ActionUpdate, a, outcome))
}

// newEnrolment returns the secret as base32 and as an otpauth:// URI (Key Uri Format) labelled with the
// e-mail.
func newEnrolment(email string, secret []byte) Enrolment {
	b32 := base32NoPad.EncodeToString(secret)
	q := url.Values{
		"secret": {b32}, "issuer": {totpIssuer}, "algorithm": {"SHA1"},
		"digits": {strconv.Itoa(totpDigits)}, "period": {strconv.Itoa(totpStep)},
	}
	u := url.URL{Scheme: "otpauth", Host: "totp", Path: "/" + totpIssuer + ":" + email, RawQuery: q.Encode()}
	return Enrolment{Secret: b32, URI: u.String()}
}

// ResetMFA recovers a staff member who lost their device (salusctl reset-mfa): in one transaction it clears
// the TOTP secret, the last used step and the enrolment, revokes the account's open sessions and audits the
// change with the operator command as the agent. The next login asks for enrolment. A patient account gives
// ErrNotStaff (audited); an unknown id ErrNotFound (not audited: no account was accessed).
func (s *Service) ResetMFA(ctx context.Context, id string) error {
	var refused error
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		a, err := s.repo.ByID(ctx, id)
		if errors.Is(err, ErrNotFound) {
			refused = ErrNotFound
			return nil
		}
		if err != nil {
			return err
		}
		if a.Role == RolePatient {
			refused = ErrNotStaff
			return s.audit.Record(ctx, operatorUpdate(a, auditevent.OutcomeMinorFailure))
		}
		if err := s.repo.ResetMFA(ctx, a.ID); err != nil {
			return err
		}
		if err := s.repo.RevokeSessions(ctx, a.ID, s.clock.Now()); err != nil {
			return err
		}
		return s.audit.Record(ctx, operatorUpdate(a, auditevent.OutcomeSuccess))
	})
	if err != nil {
		return fmt.Errorf("auth: reset MFA: %w", err)
	}
	return refused
}

// operatorUpdate is an update of account a by the operator command, which is no account.
func operatorUpdate(a Account, outcome auditevent.Outcome) auditevent.AuditEvent {
	e := event(restCoding, interaction(auditevent.CodeUpdate), auditevent.ActionUpdate, a, outcome)
	e.AgentAccountID, e.AgentName = "", auditevent.AgentOperator
	return e
}
