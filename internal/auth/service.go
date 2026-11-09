package auth

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"unicode/utf8"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/practitioner"
	"github.com/Shmeeshmee/salus/internal/practitionerrole"
	"github.com/Shmeeshmee/salus/pkg/clock"
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

// Practitioners adds a practitioner to a clinic (practitioner.Service).
type Practitioners interface {
	Create(ctx context.Context, c practitioner.Caller, givenName, familyName string) (practitioner.Practitioner, error)
}

// PractitionerRoles gives a practitioner their role in the clinic (practitionerrole.Service).
type PractitionerRoles interface {
	Create(ctx context.Context, c practitionerrole.Caller, practitionerID string) (practitionerrole.PractitionerRole, error)
}

// Ports are what the Service runs on. A binary that calls only some methods may leave the ports of the
// others nil: the notifier issues invite tokens with Repo, Tx and Clock alone.
type Ports struct {
	Repo       Repository
	Hasher     Hasher
	Audit      Auditor
	Tx         Transactor
	Clock      clock.Clock
	Challenges ChallengeStore
	Counters   Counters
	Box        SecretBox
	Jobs       Jobs
	// Practitioners and PractitionerRoles create the directory entries of an accepted practitioner invite.
	Practitioners     Practitioners
	PractitionerRoles PractitionerRoles
	// RateLimitKey keys the HMAC that names the rate-limit counters, so neither e-mail nor IP is readable in
	// Redis (RATE_LIMIT_KEY).
	RateLimitKey []byte
}

// Service holds the account rules. Every access to an account writes an AuditEvent in the same
// transaction as the access, also when it fails.
type Service struct {
	repo       Repository
	hasher     Hasher
	audit      Auditor
	tx         Transactor
	clock      clock.Clock
	challenges ChallengeStore
	counters   Counters
	box        SecretBox
	jobs       Jobs
	directory  Practitioners
	roles      PractitionerRoles
	rateKey    []byte
}

// NewService returns a Service on its ports.
func NewService(p Ports) *Service {
	return &Service{
		repo: p.Repo, hasher: p.Hasher, audit: p.Audit, tx: p.Tx, clock: p.Clock,
		challenges: p.Challenges, counters: p.Counters, box: p.Box, jobs: p.Jobs, directory: p.Practitioners,
		roles: p.PractitionerRoles, rateKey: p.RateLimitKey,
	}
}

// Register creates a patient account. The e-mail is trimmed and lower-cased; the password is stored only as
// its argon2id hash. A taken e-mail gives ErrEmailTaken (SPEC.md "E-mail as login": this does reveal that
// the e-mail has an account). language is the language the user chose on the device and detected the one
// the app matched (docs/i18n/SPEC.md §3), each "" for none.
func (s *Service) Register(ctx context.Context, email, password, language, detected string) (Account, error) {
	if err := checkLanguages(language, detected); err != nil {
		return Account{}, err
	}
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
	a := Account{Email: email, Role: RolePatient, PasswordHash: hash, Language: language, DetectedLanguage: detected}
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

// recordLogin audits a user authentication by a: subtype is auditevent.CodeLogin or CodeLogout.
func (s *Service) recordLogin(ctx context.Context, a Account, subtype string, outcome auditevent.Outcome) error {
	return s.audit.Record(ctx, event(auditevent.Coding{System: auditevent.SystemDCM, Code: auditevent.CodeUserAuthentication},
		auditevent.Coding{System: auditevent.SystemDCM, Code: subtype}, auditevent.ActionExecute, a, outcome))
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
