package organization

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

const maxName = 200 // characters

// validName reports whether name (trimmed) is a clinic's name: 1 to 200 characters on one line, without a link. It
// goes into the e-mails Salus sends in the clinic's name (invites, the signup), as the signup's own check says.
func validName(name string) bool {
	return name != "" && utf8.RuneCountInString(name) <= maxName && !strings.ContainsFunc(name, unicode.IsControl) &&
		!strings.Contains(name, "://")
}

// staffLanguages are the languages a clinic's default language may be (docs/i18n/SPEC.md §3 "Tags").
var staffLanguages = []string{"bs", "hr", "sr-Latn", "sr-Cyrl", "en"}

// Transactor runs fn in a transaction carried by the context, joining one already there
// (postgres.Transactor).
type Transactor interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// Auditor records an AuditEvent inside the caller's transaction (auditevent.Service).
type Auditor interface {
	Record(ctx context.Context, e auditevent.AuditEvent) error
}

// Service holds the clinic rules: only staff of a clinic see it, only its admin changes it (its name, whether
// it is listed, its time zone, its default language), only the operator command creates one, and anyone may list the listed clinics.
// Reads are not audited (nothing in a clinic is personal data); every write is, also when it is refused.
type Service struct {
	repo   Repository
	audit  Auditor
	tx     Transactor
	clock  clock.Clock
	places Places
}

// NewService returns a Service on its ports. places checks and names a clinic's location; nil in the binaries
// that never read or change one (their Read names no city, their Update refuses a location).
func NewService(repo Repository, audit Auditor, tx Transactor, clk clock.Clock, places Places) *Service {
	return &Service{repo: repo, audit: audit, tx: tx, clock: clk, places: places}
}

// Read returns the clinic id to a member of it, or ErrNotFound to anyone else. A receptionist's read is audited (the
// front desk's reads are, docs/clinic-cms F1); other reads are not: nothing in a clinic is personal data.
func (s *Service) Read(ctx context.Context, c Caller, id string) (Organization, error) {
	if c.OrganizationID == "" || c.OrganizationID != id {
		return Organization{}, ErrNotFound
	}
	var o Organization
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.repo.SetOrganization(ctx, c.OrganizationID); err != nil {
			return err
		}
		var err error
		if o, err = s.repo.ByID(ctx, id); err != nil || !c.Reception {
			return err
		}
		return s.audit.Record(ctx, event(auditevent.CodeRead, auditevent.ActionRead, c, id, auditevent.OutcomeSuccess))
	})
	switch {
	case errors.Is(err, ErrNotFound):
		return Organization{}, ErrNotFound
	case err != nil:
		return Organization{}, fmt.Errorf("organization: read: %w", err)
	}
	if o.CityName, err = s.cityName(ctx, o.City); err != nil {
		return Organization{}, fmt.Errorf("organization: read: %w", err)
	}
	return o, nil
}

// Create adds a clinic, unlisted, through signup.Service's clinic creation: from salusctl create-clinic (as the owner)
// or a verified signup (as salus_app, only in the new clinic's own scope). Its audit event names agent, which is no
// account (auditevent.AgentOperator or AgentSignup). The name is trimmed; a blank one or one over 200 characters gives
// ErrInvalidName.
func (s *Service) Create(ctx context.Context, agent, name string) (Organization, error) {
	name = strings.TrimSpace(name)
	if !validName(name) {
		return Organization{}, ErrInvalidName
	}
	var o Organization
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		var err error
		if o, err = s.repo.Create(ctx, name); err != nil {
			return err
		}
		e := event(auditevent.CodeCreate, auditevent.ActionCreate, Caller{OrganizationID: o.ID}, o.ID, auditevent.OutcomeSuccess)
		e.AgentName = agent
		return s.audit.Record(ctx, e)
	})
	if err != nil {
		return Organization{}, fmt.Errorf("organization: create: %w", err)
	}
	return o, nil
}

// List puts the unlisted clinic id in the public directory: the operator's approval of a clinic that signed itself up
// (salusctl signup approve, docs/clinic-cms O1). A clinic that does not exist or is listed already gives ErrNotFound.
// Audited as the operator command.
func (s *Service) List(ctx context.Context, id string) error {
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		ok, err := s.repo.List(ctx, id)
		if err != nil {
			return err
		}
		if !ok {
			return ErrNotFound
		}
		e := event(auditevent.CodeUpdate, auditevent.ActionUpdate, Caller{OrganizationID: id}, id, auditevent.OutcomeSuccess)
		e.AgentName = auditevent.AgentOperator
		return s.audit.Record(ctx, e)
	})
	if err != nil {
		return fmt.Errorf("organization: list: %w", err)
	}
	return nil
}

// DefaultLanguage returns the clinic's default language (a staff language): for a new questionnaire's
// original language and for e-mails to staff with no language of their own (docs/i18n/SPEC.md §3). It sets the
// clinic's scope on the transaction in ctx, joining one already there. A clinic that does not exist gives
// ErrNotFound. Not audited: nothing in a clinic is personal data.
func (s *Service) DefaultLanguage(ctx context.Context, organizationID string) (string, error) {
	o, err := s.byID(ctx, organizationID)
	if err != nil {
		return "", fmt.Errorf("organization: default language: %w", err)
	}
	return o.DefaultLanguage, nil
}

// Clinic is the clinic organizationID, for the clinic a visit is marked with (a shared visit's name and time zone,
// SAL-42). Read like DefaultLanguage: in the clinic's scope set on the transaction in ctx, unaudited (a clinic holds
// no personal data); ErrNotFound for none.
func (s *Service) Clinic(ctx context.Context, organizationID string) (Organization, error) {
	o, err := s.byID(ctx, organizationID)
	if err != nil {
		return Organization{}, fmt.Errorf("organization: clinic: %w", err)
	}
	return o, nil
}

// byID reads the clinic in its own scope, set on the transaction in ctx (joining one already there), unaudited;
// ErrNotFound for none.
func (s *Service) byID(ctx context.Context, organizationID string) (Organization, error) {
	if organizationID == "" {
		return Organization{}, ErrNotFound
	}
	var o Organization
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.repo.SetOrganization(ctx, organizationID); err != nil {
			return err
		}
		var err error
		o, err = s.repo.ByID(ctx, organizationID)
		return err
	})
	return o, err
}

// Names returns the name of each clinic ids names, by id, listed or not: the clinics of a patient's own records
// (their timeline, SAL-39), whose names the patient's scope cannot read. All are read in one transaction, each in its
// own clinic's scope, as DefaultLanguage reads it, so it is called outside the caller's transaction; unaudited, as
// nothing in a clinic is personal data. An unknown clinic is left out.
func (s *Service) Names(ctx context.Context, ids []string) (map[string]string, error) {
	out := make(map[string]string, len(ids))
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		for _, id := range ids {
			if _, done := out[id]; done || id == "" {
				continue
			}
			o, err := s.byID(ctx, id)
			switch {
			case errors.Is(err, ErrNotFound):
				continue
			case err != nil:
				return err
			}
			out[id] = o.Name
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("organization: names: %w", err)
	}
	return out, nil
}

// TranscriptCopy tells whether the clinic organizationID gives its patients a copy of their consultations'
// stored transcripts (PatientTranscriptCopy), for the transcript's rule (access.TranscriptCopy). Read like
// DefaultLanguage: in the clinic's scope set on the transaction in ctx, unaudited; ErrNotFound for none.
func (s *Service) TranscriptCopy(ctx context.Context, organizationID string) (bool, error) {
	o, err := s.byID(ctx, organizationID)
	if err != nil {
		return false, fmt.Errorf("organization: transcript copy: %w", err)
	}
	return o.PatientTranscriptCopy, nil
}

// ListListed returns the listed clinics, by name: the public directory, for anyone, signed in or not. Not
// audited: it holds only what the clinics chose to publish (docs/booking/SPEC.md "Public browse").
func (s *Service) ListListed(ctx context.Context) ([]Organization, error) {
	var out []Organization
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.repo.SetDirectory(ctx); err != nil {
			return err
		}
		all, err := s.repo.Listed(ctx)
		for _, o := range all {
			if o.Listed { // not on row-level security alone
				out = append(out, o)
			}
		}
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("organization: list listed: %w", err)
	}
	return out, nil
}

// Update changes the clinic id as ch says, at the version the caller last read, and returns it at its new
// version. Only the clinic's admin may: a member who is not gets ErrForbidden, anyone else ErrNotFound. A
// blank or overlong name gives ErrInvalidName, a time zone that is not an IANA name ErrInvalidTimeZone and a
// default language that is not a staff language ErrInvalidDefaultLanguage, before anything is accessed; a version that is not the current one ErrConflict;
// an address line or a postal code that is too long, or left to a clinic without a city, ErrInvalidLocation.
func (s *Service) Update(ctx context.Context, c Caller, id string, ch Changes, version int32) (Organization, error) {
	ch.Name = strings.TrimSpace(ch.Name)
	if !validName(ch.Name) {
		return Organization{}, ErrInvalidName
	}
	if ch.TimeZone != nil && !validTimeZone(*ch.TimeZone) {
		return Organization{}, ErrInvalidTimeZone
	}
	if ch.DefaultLanguage != nil && !slices.Contains(staffLanguages, *ch.DefaultLanguage) {
		return Organization{}, ErrInvalidDefaultLanguage
	}
	ch, err := trimAddress(ch)
	if err != nil {
		return Organization{}, err
	}
	cityName, err := s.checkLocation(ctx, ch)
	if err != nil {
		return Organization{}, err
	}
	if refused := refusal(c, id); refused != nil {
		// Nothing was accessed: the event names no entity.
		if err := s.record(ctx, c, "", auditevent.OutcomeMinorFailure); err != nil {
			return Organization{}, fmt.Errorf("organization: update: %w", err)
		}
		return Organization{}, refused
	}
	var o Organization
	var failed error // a refusal that is audited, and so commits
	err = s.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.repo.SetOrganization(ctx, c.OrganizationID); err != nil {
			return err
		}
		var err error
		o, failed, err = s.update(ctx, id, ch, version)
		if err != nil {
			return err
		}
		entity, outcome := o.ID, auditevent.OutcomeSuccess
		if failed != nil {
			outcome = auditevent.OutcomeMinorFailure
			if errors.Is(failed, ErrNotFound) {
				entity = ""
			}
		}
		return s.record(ctx, c, entity, outcome)
	})
	switch {
	case err != nil:
		return Organization{}, fmt.Errorf("organization: update: %w", err)
	case failed != nil:
		return Organization{}, failed
	}
	o.CityName = cityName
	if ch.City == nil {
		if o.CityName, err = s.cityName(ctx, o.City); err != nil {
			return Organization{}, fmt.Errorf("organization: update: %w", err)
		}
	}
	return o, nil
}

// update reads the clinic and changes it when version is current. A missing clinic or a stale version is
// returned as failed, with the clinic read when there is one; err is any other error.
func (s *Service) update(ctx context.Context, id string, ch Changes, version int32) (o Organization, failed, err error) {
	cur, err := s.repo.ByID(ctx, id)
	if errors.Is(err, ErrNotFound) {
		return Organization{}, ErrNotFound, nil
	}
	if err != nil {
		return Organization{}, nil, err
	}
	if cur.VersionID != version {
		return cur, ErrConflict, nil
	}
	next := cur
	next.Name = ch.Name
	if ch.Listed != nil {
		next.Listed = *ch.Listed
	}
	if ch.TimeZone != nil {
		next.TimeZone = *ch.TimeZone
	}
	if ch.DefaultLanguage != nil {
		next.DefaultLanguage = *ch.DefaultLanguage
	}
	if ch.PatientTranscriptCopy != nil {
		next.PatientTranscriptCopy = *ch.PatientTranscriptCopy
	}
	next = withLocation(next, ch, s.clock.Now())
	if next, err = withAddress(next, ch); err != nil {
		return cur, err, nil
	}
	o, err = s.repo.Update(ctx, next, s.clock.Now())
	if errors.Is(err, ErrConflict) { // changed since the read
		return cur, ErrConflict, nil
	}
	if errors.Is(err, ErrNotApproved) || errors.Is(err, ErrInvalidLocation) {
		return cur, err, nil
	}
	return o, nil, err
}

// validTimeZone reports whether zone is an IANA time zone name (time.LoadLocation; the binaries embed the
// zone data with time/tzdata). "" and "Local" load too, but name no zone.
func validTimeZone(zone string) bool {
	if zone == "" || zone == "Local" {
		return false
	}
	_, err := time.LoadLocation(zone)
	return err == nil
}

// refusal is why c may not change the clinic id, or nil when c is its admin.
func refusal(c Caller, id string) error {
	switch {
	case c.OrganizationID == "" || c.OrganizationID != id:
		return ErrNotFound
	case !c.Admin:
		return ErrForbidden
	}
	return nil
}

// record audits an update by c of the clinic id ("" when none was accessed).
func (s *Service) record(ctx context.Context, c Caller, id string, outcome auditevent.Outcome) error {
	return s.audit.Record(ctx, event(auditevent.CodeUpdate, auditevent.ActionUpdate, c, id, outcome))
}

// event is a write (interaction) by c of the clinic id ("" when none was accessed), for operations.
func event(interaction string, action auditevent.Action, c Caller, id string, outcome auditevent.Outcome) auditevent.AuditEvent {
	e := auditevent.AuditEvent{
		Type:           auditevent.Coding{System: auditevent.SystemAuditEventType, Code: auditevent.CodeRest},
		Subtype:        auditevent.Coding{System: auditevent.SystemRestfulInteraction, Code: interaction},
		Purpose:        auditevent.Coding{System: auditevent.SystemActReason, Code: auditevent.CodeHOPERAT},
		Action:         action,
		Outcome:        outcome,
		AgentAccountID: c.AccountID,
		OrganizationID: c.OrganizationID,
	}
	if id != "" {
		e.EntityReference = "Organization/" + id
	}
	return e
}

// ScopeNames returns the names of the clinics ids, by id, as the scope of the clinic org shows them (its own, and those
// of other clinics' visits shared with it: organization_shared), for the history's items; one it does not show is
// left out. It sets that scope on the transaction in ctx, joining one already there. Not audited: nothing in a
// clinic is personal data.
func (s *Service) ScopeNames(ctx context.Context, org string, ids []string) (map[string]string, error) {
	out := make(map[string]string, len(ids))
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.repo.SetOrganization(ctx, org); err != nil {
			return err
		}
		for _, id := range ids {
			if _, done := out[id]; done {
				continue
			}
			o, err := s.repo.ByID(ctx, id)
			switch {
			case errors.Is(err, ErrNotFound):
				continue
			case err != nil:
				return err
			}
			out[id] = o.Name
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("organization: scope names: %w", err)
	}
	return out, nil
}
