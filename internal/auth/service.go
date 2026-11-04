package auth

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"unicode/utf8"

	"github.com/Shmeeshmee/salus/internal/auditevent"
)

// Password length limits in characters, without composition rules (NIST SP 800-63B).
const (
	minPassword = 12
	maxPassword = 128
	maxEmail    = 254
)

// dummyPassword is hashed for an unknown e-mail, so a login takes as long whether or not the e-mail exists.
const dummyPassword = "dummy password for an unknown e-mail"

// Transactor runs fn in a transaction carried by the context, joining one already there
// (postgres.Transactor).
type Transactor interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// Auditor records an AuditEvent inside the caller's transaction (auditevent.Service).
type Auditor interface {
	Record(ctx context.Context, e auditevent.AuditEvent) error
}

// Service holds the account rules. Every access to an account writes an AuditEvent in the same
// transaction as the access, also when it fails.
type Service struct {
	repo   Repository
	hasher Hasher
	audit  Auditor
	tx     Transactor
}

// NewService returns a Service on its ports.
func NewService(repo Repository, hasher Hasher, audit Auditor, tx Transactor) *Service {
	return &Service{repo: repo, hasher: hasher, audit: audit, tx: tx}
}

// Register creates a patient account. The e-mail is trimmed and lower-cased; the password is stored only as
// its argon2id hash. A taken e-mail gives ErrEmailTaken (SPEC.md "E-mail as login": this does reveal that
// the e-mail has an account).
func (s *Service) Register(ctx context.Context, email, password string) (Account, error) {
	email = normaliseEmail(email)
	if !validEmail(email) {
		return Account{}, ErrInvalidEmail
	}
	if n := utf8.RuneCountInString(password); n < minPassword || n > maxPassword {
		return Account{}, ErrInvalidPassword
	}
	hash, err := s.hasher.Hash(ctx, password) // outside the transaction: it holds no connection while hashing
	if err != nil {
		return Account{}, fmt.Errorf("auth: register: %w", err)
	}
	a := Account{Email: email, Role: RolePatient, PasswordHash: hash}
	var taken bool
	err = s.tx.WithTx(ctx, func(ctx context.Context) error {
		id, err := s.repo.Create(ctx, a)
		taken = errors.Is(err, ErrEmailTaken)
		if err != nil && !taken {
			return err
		}
		a.ID = id
		subject, outcome := a, auditevent.OutcomeSuccess
		if taken {
			subject, outcome = Account{}, auditevent.OutcomeMinorFailure // nobody yet: no agent, no entity
		}
		return s.audit.Record(ctx, event(restCoding, interaction(auditevent.CodeCreate), auditevent.ActionCreate, subject, outcome))
	})
	switch {
	case err != nil:
		return Account{}, fmt.Errorf("auth: register: %w", err)
	case taken:
		return Account{}, ErrEmailTaken
	}
	a.PasswordHash = ""
	return a, nil
}

// CheckPassword returns the account whose e-mail and password match, or ErrWrongCredentials for an unknown
// e-mail and a wrong password alike. An unknown e-mail costs one hash too, so timing does not tell them
// apart. A hash made with older parameters is replaced. Every attempt is audited as a login.
func (s *Service) CheckPassword(ctx context.Context, email, password string) (Account, error) {
	email = normaliseEmail(email)
	var a Account
	var wrong bool
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		var err error
		a, err = s.repo.ByEmail(ctx, email)
		if errors.Is(err, ErrNotFound) {
			if _, err := s.hasher.Hash(ctx, dummyPassword); err != nil {
				return err
			}
			wrong, a = true, Account{}
			return s.recordLogin(ctx, a, auditevent.OutcomeMinorFailure)
		}
		if err != nil {
			return err
		}
		ok, stale, err := s.hasher.Verify(ctx, password, a.PasswordHash)
		if err != nil {
			return err
		}
		if !ok {
			wrong = true
			return s.recordLogin(ctx, a, auditevent.OutcomeMinorFailure)
		}
		if stale {
			if err := s.rehash(ctx, a.ID, password); err != nil {
				return err
			}
		}
		return s.recordLogin(ctx, a, auditevent.OutcomeSuccess)
	})
	switch {
	case err != nil:
		return Account{}, fmt.Errorf("auth: check password: %w", err)
	case wrong:
		return Account{}, ErrWrongCredentials
	}
	a.PasswordHash = ""
	return a, nil
}

func (s *Service) rehash(ctx context.Context, id, password string) error {
	hash, err := s.hasher.Hash(ctx, password)
	if err != nil {
		return err
	}
	return s.repo.SetPasswordHash(ctx, id, hash)
}

func (s *Service) recordLogin(ctx context.Context, a Account, outcome auditevent.Outcome) error {
	return s.audit.Record(ctx, event(auditevent.Coding{System: auditevent.SystemDCM, Code: auditevent.CodeUserAuthentication},
		auditevent.Coding{System: auditevent.SystemDCM, Code: auditevent.CodeLogin}, auditevent.ActionExecute, a, outcome))
}

// Me returns the signed-in account (GET /auth/me), or ErrNotFound. The read is audited.
func (s *Service) Me(ctx context.Context, id string) (Account, error) {
	var a Account
	var missing bool
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		var err error
		a, err = s.repo.ByID(ctx, id)
		missing = errors.Is(err, ErrNotFound)
		if err != nil && !missing {
			return err
		}
		subject, outcome := a, auditevent.OutcomeSuccess
		if missing {
			subject, outcome = Account{ID: id}, auditevent.OutcomeMinorFailure
		}
		return s.audit.Record(ctx, event(restCoding, interaction(auditevent.CodeRead), auditevent.ActionRead, subject, outcome))
	})
	switch {
	case err != nil:
		return Account{}, fmt.Errorf("auth: me: %w", err)
	case missing:
		return Account{}, ErrNotFound
	}
	a.PasswordHash = ""
	return a, nil
}

var restCoding = auditevent.Coding{System: auditevent.SystemAuditEventType, Code: auditevent.CodeRest}

func interaction(code string) auditevent.Coding {
	return auditevent.Coding{System: auditevent.SystemRestfulInteraction, Code: code}
}

// event is an access by account a to itself, for operations (HOPERAT). An empty a gives an event with no
// agent and no entity.
func event(typ, subtype auditevent.Coding, action auditevent.Action, a Account, outcome auditevent.Outcome) auditevent.AuditEvent {
	e := auditevent.AuditEvent{
		Type:           typ,
		Subtype:        subtype,
		Purpose:        auditevent.Coding{System: auditevent.SystemActReason, Code: auditevent.CodeHOPERAT},
		Action:         action,
		Outcome:        outcome,
		AgentAccountID: a.ID,
		OrganizationID: a.OrganizationID,
	}
	if a.ID != "" {
		e.EntityReference = "Account/" + a.ID
	}
	return e
}

func normaliseEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// validEmail accepts a plain address (no display name, no comments) of at most maxEmail characters.
func validEmail(email string) bool {
	if email == "" || len(email) > maxEmail {
		return false
	}
	addr, err := mail.ParseAddress(email)
	return err == nil && addr.Name == "" && addr.Address == email
}
