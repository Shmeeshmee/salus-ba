package signup

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/mail"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/auth"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/internal/subscription"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

// TokenLifetime is how long a signup may be verified after its latest verification e-mail's token was issued (U8).
const TokenLifetime = 72 * time.Hour

// The rate limits: signups per client address an hour and per e-mail a day, verifications per address an hour.
const (
	maxSignupsPerIP    = 5
	maxSignupsPerEmail = 3
	maxVerifiesPerIP   = 20
	maxEmail           = 254
	maxClinicName      = 200
)

// staffLanguages are the languages a clinic's staff may use (docs/i18n/SPEC.md): a clinic's admin signs up in one.
var staffLanguages = []string{"bs", "hr", "sr-Latn", "sr-Cyrl", "en"}

// steps are the wizard's steps, in order.
var steps = []Step{StepDetails, StepBilling, StepHours, StepPractitioner, StepServices}

// Transactor runs fn in a transaction carried by the context, joining one already there.
type Transactor interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// Clinics creates a clinic and lists it in the public directory (organization.Service).
type Clinics interface {
	Create(ctx context.Context, agent, name string) (organization.Organization, error)
	List(ctx context.Context, id string) error
}

// Admins invites a clinic's first admin (auth.Service).
type Admins interface {
	InviteAdmin(ctx context.Context, agent, org, email string) (auth.Invite, error)
}

// Trials starts a clinic's trial (subscription.Service).
type Trials interface {
	StartTrial(ctx context.Context, agent, org string) (subscription.Subscription, error)
}

// Ports are what the Service consumes. salusctl's CreateClinic, Signups and Approve need Repo, Tx, Clock, Clinics,
// Admins and Trials only.
type Ports struct {
	Repo     Repository
	Jobs     Jobs
	Counters Counters
	Tx       Transactor
	Clock    clock.Clock
	Clinics  Clinics
	Admins   Admins
	Trials   Trials
	// RateLimitKey keys the HMAC that names the rate-limit counters, so neither e-mail nor address is readable.
	RateLimitKey []byte
}

// Service is the signup's business rules.
type Service struct {
	p Ports
}

// NewService builds a Service on p.
func NewService(p Ports) *Service {
	return &Service{p: p}
}

// Start records r and enqueues the verification e-mail. It answers the same whether or not the e-mail is known, so it
// enumerates nothing: every valid request is a new signup. ErrInvalid for a malformed request; a RateLimitedError over
// the limits of clientIP or the e-mail.
func (s *Service) Start(ctx context.Context, r Request, clientIP string) error {
	r, err := validRequest(r)
	if err != nil {
		return err
	}
	if err := s.limit(ctx, "signup:ip:", clientIP, time.Hour, maxSignupsPerIP); err != nil {
		return err
	}
	if err := s.limit(ctx, "signup:email:", r.Email, 24*time.Hour, maxSignupsPerEmail); err != nil {
		return err
	}
	// The first token is never sent: the e-mail job issues the link's (IssueVerification).
	_, hash, err := newToken()
	if err != nil {
		return fmt.Errorf("signup: start: %w", err)
	}
	err = s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		id, err := s.p.Repo.Insert(ctx, r, hash)
		if err != nil {
			return err
		}
		return s.p.Jobs.EnqueueEmail(ctx, id)
	})
	if err != nil {
		return fmt.Errorf("signup: start: %w", err)
	}
	return nil
}

func validRequest(r Request) (Request, error) {
	r.Email = strings.ToLower(strings.TrimSpace(r.Email))
	r.ClinicName = strings.TrimSpace(r.ClinicName)
	a, err := mail.ParseAddress(r.Email)
	if err != nil || a.Address != r.Email || a.Name != "" || len(r.Email) > maxEmail {
		return Request{}, ErrInvalid
	}
	if r.ClinicName == "" || utf8.RuneCountInString(r.ClinicName) > maxClinicName || !slices.Contains(staffLanguages, r.Language) {
		return Request{}, ErrInvalid
	}
	// The name goes into an e-mail Salus sends to the address given: one line of text, no link.
	if strings.ContainsFunc(r.ClinicName, unicode.IsControl) || strings.Contains(r.ClinicName, "://") {
		return Request{}, ErrInvalid
	}
	return r, nil
}

// IssueVerification issues a new token for the verification e-mail of the unverified signup id, replacing the last
// one, and returns what the e-mail needs (the notifier's job: only the newest link works). ErrInvalidToken when the
// signup is verified or unknown.
func (s *Service) IssueVerification(ctx context.Context, id string) (Verification, error) {
	token, hash, err := newToken()
	if err != nil {
		return Verification{}, fmt.Errorf("signup: issue verification: %w", err)
	}
	var v Verification
	err = s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		sg, err := s.p.Repo.ByID(ctx, id)
		if errors.Is(err, ErrNotFound) {
			return ErrInvalidToken
		}
		if err != nil {
			return err
		}
		ok, err := s.p.Repo.SetToken(ctx, id, hash, s.p.Clock.Now())
		if err != nil {
			return err
		}
		if !ok {
			return ErrInvalidToken
		}
		v = Verification{Email: sg.Email, ClinicName: sg.ClinicName, Language: sg.Language, Token: token}
		return nil
	})
	if err != nil {
		return Verification{}, fmt.Errorf("signup: issue verification: %w", err)
	}
	return v, nil
}

// Verify checks token, creates the clinic unlisted, invites its admin and starts the trial, in one transaction, each
// audited with the signup as the agent.
// ErrInvalidToken for a token that is unknown, replaced, used or older than TokenLifetime; a RateLimitedError over the
// limit of clientIP.
func (s *Service) Verify(ctx context.Context, token, clientIP string) (Signup, error) {
	if err := s.limit(ctx, "signup:verify:", clientIP, time.Hour, maxVerifiesPerIP); err != nil {
		return Signup{}, err
	}
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil || len(raw) != 32 {
		return Signup{}, ErrInvalidToken
	}
	hash := sha256.Sum256(raw)
	var out Signup
	err = s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		sg, err := s.p.Repo.ByTokenHash(ctx, hash[:])
		if errors.Is(err, ErrNotFound) {
			return ErrInvalidToken
		}
		if err != nil {
			return err
		}
		now := s.p.Clock.Now()
		if !sg.VerifiedAt.IsZero() || now.Sub(sg.TokenIssuedAt) > TokenLifetime {
			return ErrInvalidToken
		}
		created, err := s.createClinic(ctx, auditevent.AgentSignup, sg.ClinicName, sg.Email)
		if err != nil {
			return err
		}
		ok, err := s.p.Repo.MarkVerified(ctx, sg.ID, created.OrganizationID, now)
		if err != nil {
			return err
		}
		if !ok {
			return ErrInvalidToken // verified by a concurrent request: this one's clinic is rolled back
		}
		sg.VerifiedAt, sg.OrganizationID = now, created.OrganizationID
		out = sg
		return nil
	})
	if err != nil {
		return Signup{}, fmt.Errorf("signup: verify: %w", err)
	}
	return out, nil
}

// CreateClinic creates the clinic name, unlisted, with its first admin's invite to email and its 30-day trial, in one
// transaction, audited as salusctl (salusctl create-clinic). Without the owner's default plan the clinic is created
// without a trial (Created.Trial false).
func (s *Service) CreateClinic(ctx context.Context, name, email string) (Created, error) {
	return s.createClinic(ctx, auditevent.AgentOperator, name, email)
}

// createClinic is the one path for a clinic, from salusctl create-clinic and from a verified signup; its writes are
// audited as agent.
func (s *Service) createClinic(ctx context.Context, agent, name, email string) (Created, error) {
	var out Created
	err := s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		clinic, err := s.p.Clinics.Create(ctx, agent, name)
		if err != nil {
			return err
		}
		inv, err := s.p.Admins.InviteAdmin(ctx, agent, clinic.ID, email)
		if err != nil {
			return err
		}
		out = Created{OrganizationID: clinic.ID, InviteID: inv.ID, Trial: true}
		if _, err = s.p.Trials.StartTrial(ctx, agent, clinic.ID); errors.Is(err, subscription.ErrNoPlan) {
			out.Trial, err = false, nil
		}
		return err
	})
	if err != nil {
		return Created{}, fmt.Errorf("signup: create clinic: %w", err)
	}
	return out, nil
}

// AdminJoined forgets the admin's e-mail on the verified signup of the clinic organizationID (U8), in the transaction
// in ctx: auth.Service calls it as the clinic's admin accepts their invite. The account holds the e-mail from then on.
func (s *Service) AdminJoined(ctx context.Context, organizationID string) error {
	err := s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		return s.p.Repo.ForgetEmail(ctx, organizationID)
	})
	if err != nil {
		return fmt.Errorf("signup: admin joined: %w", err)
	}
	return nil
}

// Signups returns the verified signups, those waiting for approval first (salusctl signup list).
func (s *Service) Signups(ctx context.Context) ([]Signup, error) {
	out, err := s.p.Repo.Verified(ctx)
	if err != nil {
		return nil, fmt.Errorf("signup: list: %w", err)
	}
	return out, nil
}

// Approve lists the clinic organizationID of a verified signup in the public directory (salusctl signup approve).
// ErrNotFound when no verified signup of that clinic is waiting.
func (s *Service) Approve(ctx context.Context, organizationID string) error {
	err := s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		ok, err := s.p.Repo.Approve(ctx, organizationID, s.p.Clock.Now())
		if err != nil {
			return err
		}
		if !ok {
			return ErrNotFound
		}
		return s.p.Clinics.List(ctx, organizationID)
	})
	if err != nil {
		return fmt.Errorf("signup: approve: %w", err)
	}
	return nil
}

// Setup returns the setup wizard's progress of the caller's clinic: every step, open unless saved done or skipped.
// ErrNotFound for a caller with no clinic, ErrForbidden for staff who are not its admin (a practitioner, a
// receptionist): the wizard is the admin's. Not personal data; not audited.
func (s *Service) Setup(ctx context.Context, c Caller) (Setup, error) {
	if c.OrganizationID == "" {
		return Setup{}, ErrNotFound
	}
	if !c.Admin {
		return Setup{}, ErrForbidden
	}
	var out Setup
	err := s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		var err error
		out, err = s.steps(ctx, c.OrganizationID)
		return err
	})
	if err != nil {
		return Setup{}, fmt.Errorf("signup: setup: %w", err)
	}
	return out, nil
}

// SaveSetup stores the wizard's progress of the admin caller's clinic: each step named in in, open removing what was
// saved. ErrForbidden for a member who is not the admin, ErrInvalidSetup for an unknown step or state.
func (s *Service) SaveSetup(ctx context.Context, c Caller, in Setup) (Setup, error) {
	if c.OrganizationID == "" {
		return Setup{}, ErrNotFound
	}
	if !c.Admin {
		return Setup{}, ErrForbidden
	}
	for step, state := range in.Steps {
		if !slices.Contains(steps, step) || (state != StepOpen && state != StepDone && state != StepSkipped) {
			return Setup{}, ErrInvalidSetup
		}
	}
	var out Setup
	err := s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.p.Repo.SetOrganization(ctx, c.OrganizationID); err != nil {
			return err
		}
		now := s.p.Clock.Now()
		for _, step := range steps {
			if state, ok := in.Steps[step]; ok {
				if err := s.p.Repo.PutStep(ctx, c.OrganizationID, step, state, now); err != nil {
					return err
				}
			}
		}
		var err error
		out, err = s.steps(ctx, c.OrganizationID)
		return err
	})
	if err != nil {
		return Setup{}, fmt.Errorf("signup: save setup: %w", err)
	}
	return out, nil
}

// steps sets the clinic's scope and returns its progress, every step named.
func (s *Service) steps(ctx context.Context, org string) (Setup, error) {
	if err := s.p.Repo.SetOrganization(ctx, org); err != nil {
		return Setup{}, err
	}
	saved, err := s.p.Repo.Steps(ctx, org)
	if err != nil {
		return Setup{}, err
	}
	out := Setup{Steps: make(map[Step]StepState, len(steps))}
	for _, step := range steps {
		out.Steps[step] = StepOpen
		if state, ok := saved[step]; ok {
			out.Steps[step] = state
		}
	}
	return out, nil
}

// limit counts a request of value under prefix and returns a RateLimitedError when the window's count is over max.
func (s *Service) limit(ctx context.Context, prefix, value string, window time.Duration, max int64) error {
	mac := hmac.New(sha256.New, s.p.RateLimitKey)
	mac.Write([]byte(value))
	n, left, err := s.p.Counters.Incr(ctx, prefix+hex.EncodeToString(mac.Sum(nil)), window)
	if err != nil {
		return fmt.Errorf("signup: rate limit: %w", err)
	}
	if n > max {
		return &RateLimitedError{RetryAfter: left}
	}
	return nil
}

// newToken returns a random token for a link (base64url of 32 bytes) and its SHA-256.
func newToken() (string, []byte, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", nil, err
	}
	hash := sha256.Sum256(raw)
	return base64.RawURLEncoding.EncodeToString(raw), hash[:], nil
}
