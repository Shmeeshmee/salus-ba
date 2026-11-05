package practitionerrole

import (
	"context"
	"errors"
	"fmt"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/internal/practitioner"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

// Transactor runs fn in a transaction carried by the context, joining one already there
// (postgres.Transactor).
type Transactor interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// Auditor records an AuditEvent inside the caller's transaction (auditevent.Service).
type Auditor interface {
	Record(ctx context.Context, e auditevent.AuditEvent) error
}

// Clinics reads a clinic in its own scope, unaudited inside the caller's access (organization.Service).
type Clinics interface {
	Read(ctx context.Context, c organization.Caller, id string) (organization.Organization, error)
}

// People gives the active practitioners of listed clinics among ids (practitioner.Service).
type People interface {
	PublicNames(ctx context.Context, ids []string) ([]practitioner.Practitioner, error)
}

// Ports are what the Service is built on.
type Ports struct {
	Repo    Repository
	Audit   Auditor
	Tx      Transactor
	Clock   clock.Clock
	Clinics Clinics
	People  People
}

// Service holds the practitioner role rules: only staff of a clinic see its roles, only its admin sets a
// role's specialty, and every read, search and write by staff is audited, also when it is refused. Anyone
// may read the active roles of a listed clinic whose practitioner is active (PublicByOrganization); that is
// not audited.
type Service struct {
	repo    Repository
	audit   Auditor
	tx      Transactor
	clock   clock.Clock
	clinics Clinics
	people  People
}

// NewService returns a Service on its ports.
func NewService(p Ports) *Service {
	return &Service{repo: p.Repo, audit: p.Audit, tx: p.Tx, clock: p.Clock, clinics: p.Clinics, people: p.People}
}

// Read returns the role id to staff of its clinic, or ErrNotFound to anyone else.
func (s *Service) Read(ctx context.Context, c Caller, id string) (PractitionerRole, error) {
	if c.OrganizationID == "" {
		if err := s.record(ctx, c, auditevent.CodeRead, "", auditevent.OutcomeMinorFailure); err != nil {
			return PractitionerRole{}, fmt.Errorf("practitionerrole: read: %w", err)
		}
		return PractitionerRole{}, ErrNotFound
	}
	var p PractitionerRole
	var missing bool
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.repo.SetOrganization(ctx, c.OrganizationID); err != nil {
			return err
		}
		var err error
		p, err = s.repo.ByID(ctx, id)
		if err != nil && !errors.Is(err, ErrNotFound) {
			return err
		}
		// The clinic check does not rest on row-level security alone.
		missing = err != nil || p.OrganizationID != c.OrganizationID
		if missing {
			return s.record(ctx, c, auditevent.CodeRead, "", auditevent.OutcomeMinorFailure)
		}
		return s.record(ctx, c, auditevent.CodeRead, p.ID, auditevent.OutcomeSuccess)
	})
	switch {
	case err != nil:
		return PractitionerRole{}, fmt.Errorf("practitionerrole: read: %w", err)
	case missing:
		return PractitionerRole{}, ErrNotFound
	}
	return p, nil
}

// Search returns the roles of the clinic org (PractitionerRole?organization=org), oldest first, to staff of
// that clinic, or ErrNotFound to anyone else. Each role returned is audited; a search that returns none
// writes one event without an entity.
func (s *Service) Search(ctx context.Context, c Caller, org string) ([]PractitionerRole, error) {
	if c.OrganizationID == "" || c.OrganizationID != org {
		if err := s.record(ctx, c, auditevent.CodeSearchType, "", auditevent.OutcomeMinorFailure); err != nil {
			return nil, fmt.Errorf("practitionerrole: search: %w", err)
		}
		return nil, ErrNotFound
	}
	var roles []PractitionerRole
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.repo.SetOrganization(ctx, c.OrganizationID); err != nil {
			return err
		}
		all, err := s.repo.ByOrganization(ctx, org)
		if err != nil {
			return err
		}
		for _, p := range all {
			if p.OrganizationID == c.OrganizationID { // not on row-level security alone
				roles = append(roles, p)
			}
		}
		if len(roles) == 0 {
			return s.record(ctx, c, auditevent.CodeSearchType, "", auditevent.OutcomeSuccess)
		}
		for _, p := range roles {
			if err := s.record(ctx, c, auditevent.CodeSearchType, p.ID, auditevent.OutcomeSuccess); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("practitionerrole: search: %w", err)
	}
	return roles, nil
}

// Create gives the practitioner practitionerID their role in the caller's clinic and returns it (on invite
// acceptance, as the invited account). A caller without a clinic gets ErrForbidden. Audited, also when
// refused.
func (s *Service) Create(ctx context.Context, c Caller, practitionerID string) (PractitionerRole, error) {
	if c.OrganizationID == "" {
		if err := s.record(ctx, c, auditevent.CodeCreate, "", auditevent.OutcomeMinorFailure); err != nil {
			return PractitionerRole{}, fmt.Errorf("practitionerrole: create: %w", err)
		}
		return PractitionerRole{}, ErrForbidden
	}
	var p PractitionerRole
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.repo.SetOrganization(ctx, c.OrganizationID); err != nil {
			return err
		}
		var err error
		if p, err = s.repo.Create(ctx, c.OrganizationID, practitionerID); err != nil {
			return err
		}
		return s.record(ctx, c, auditevent.CodeCreate, p.ID, auditevent.OutcomeSuccess)
	})
	if err != nil {
		return PractitionerRole{}, fmt.Errorf("practitionerrole: create: %w", err)
	}
	return p, nil
}

// SetSpecialty gives the role id the specialty code (one of Specialties), at the version the caller last
// read, and returns it at its new version. Only the clinic's admin may: a member who is not gets
// ErrForbidden, anyone else ErrNotFound. A code not in the list gives ErrInvalidSpecialty before anything is
// accessed; a version that is not the current one ErrConflict. Audited, also when refused.
func (s *Service) SetSpecialty(ctx context.Context, c Caller, id, code string, version int32) (PractitionerRole, error) {
	if _, ok := SpecialtyByCode(code); !ok {
		return PractitionerRole{}, ErrInvalidSpecialty
	}
	if refused := updateRefusal(c); refused != nil {
		if err := s.record(ctx, c, auditevent.CodeUpdate, "", auditevent.OutcomeMinorFailure); err != nil {
			return PractitionerRole{}, fmt.Errorf("practitionerrole: set specialty: %w", err)
		}
		return PractitionerRole{}, refused
	}
	var p PractitionerRole
	var failed error // a refusal that is audited, and so commits
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.repo.SetOrganization(ctx, c.OrganizationID); err != nil {
			return err
		}
		var err error
		p, failed, err = s.setSpecialty(ctx, c, id, code, version)
		if err != nil {
			return err
		}
		entity, outcome := p.ID, auditevent.OutcomeSuccess
		if failed != nil {
			outcome = auditevent.OutcomeMinorFailure
		}
		return s.record(ctx, c, auditevent.CodeUpdate, entity, outcome)
	})
	switch {
	case err != nil:
		return PractitionerRole{}, fmt.Errorf("practitionerrole: set specialty: %w", err)
	case failed != nil:
		return PractitionerRole{}, failed
	}
	return p, nil
}

// setSpecialty reads the role and sets its code when version is current. A role outside c's clinic or a
// stale version is returned as failed, with the role read when c may see it; err is any other error.
func (s *Service) setSpecialty(ctx context.Context, c Caller, id, code string, version int32) (p PractitionerRole, failed, err error) {
	cur, err := s.repo.ByID(ctx, id)
	switch {
	case errors.Is(err, ErrNotFound):
		return PractitionerRole{}, ErrNotFound, nil
	case err != nil:
		return PractitionerRole{}, nil, err
	case cur.OrganizationID != c.OrganizationID: // not on row-level security alone
		return PractitionerRole{}, ErrNotFound, nil
	case cur.VersionID != version:
		return cur, ErrConflict, nil
	}
	p, err = s.repo.SetSpecialty(ctx, id, code, version, s.clock.Now())
	if errors.Is(err, ErrConflict) { // changed since the read
		return cur, ErrConflict, nil
	}
	return p, nil, err
}

// updateRefusal is why c may not change a role, or nil when c is an admin of a clinic (whose roles only are
// then in scope).
func updateRefusal(c Caller) error {
	switch {
	case c.OrganizationID == "":
		return ErrNotFound
	case !c.Admin:
		return ErrForbidden
	}
	return nil
}

// PublicByOrganization returns the active roles of the clinic org whose practitioner is active, oldest
// first, when the clinic is listed, and none otherwise: the public directory, for anyone, signed in or not.
// The service checks the listing and the practitioner itself; row-level security is the second guard. It
// holds only what the clinic chose to publish, so it is not audited (docs/booking/SPEC.md "Public browse").
func (s *Service) PublicByOrganization(ctx context.Context, org string) ([]PractitionerRole, error) {
	listed, err := s.listed(ctx, org)
	if err != nil || !listed {
		return nil, err
	}
	var roles []PractitionerRole
	err = s.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.repo.SetDirectory(ctx); err != nil {
			return err
		}
		all, err := s.repo.PublicByOrganization(ctx, org)
		if err != nil {
			return err
		}
		ids := make([]string, 0, len(all))
		for _, p := range all {
			ids = append(ids, p.PractitionerID)
		}
		people, err := s.people.PublicNames(ctx, ids)
		if err != nil {
			return err
		}
		active := make(map[string]bool, len(people))
		for _, p := range people {
			active[p.ID] = p.Active
		}
		for _, p := range all {
			if p.OrganizationID == org && p.Active && active[p.PractitionerID] {
				roles = append(roles, p)
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("practitionerrole: public search: %w", err)
	}
	return roles, nil
}

// listed reports whether the clinic org is listed, read in its own scope and transaction before the
// directory's (a clinic that does not exist is not).
func (s *Service) listed(ctx context.Context, org string) (bool, error) {
	clinic, err := s.clinics.Read(ctx, organization.Caller{OrganizationID: org}, org)
	switch {
	case errors.Is(err, organization.ErrNotFound):
		return false, nil
	case err != nil:
		return false, fmt.Errorf("practitionerrole: public search: %w", err)
	}
	return clinic.Listed, nil
}

// Lookup returns the role id of the clinic org, or ErrNotFound: for a service that shows whose an
// appointment is, inside its own audited access. It joins the transaction in ctx, sets the clinic's scope and
// writes no event.
func (s *Service) Lookup(ctx context.Context, org, id string) (PractitionerRole, error) {
	return s.lookup(ctx, org, func(ctx context.Context) (PractitionerRole, error) { return s.repo.ByID(ctx, id) })
}

// OfPractitioner returns the role of the practitioner practitionerID in the clinic org, or ErrNotFound: to
// know which role a signed-in practitioner acts in, or which one a search names. Like Lookup, unaudited.
func (s *Service) OfPractitioner(ctx context.Context, org, practitionerID string) (PractitionerRole, error) {
	return s.lookup(ctx, org, func(ctx context.Context) (PractitionerRole, error) { return s.repo.ByPractitioner(ctx, practitionerID) })
}

func (s *Service) lookup(ctx context.Context, org string, find func(ctx context.Context) (PractitionerRole, error)) (PractitionerRole, error) {
	var p PractitionerRole
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.repo.SetOrganization(ctx, org); err != nil {
			return err
		}
		var err error
		p, err = find(ctx)
		return err
	})
	switch {
	case errors.Is(err, ErrNotFound) || (err == nil && p.OrganizationID != org): // not on row-level security alone
		return PractitionerRole{}, ErrNotFound
	case err != nil:
		return PractitionerRole{}, fmt.Errorf("practitionerrole: lookup: %w", err)
	}
	return p, nil
}

// record audits a read, a search, a create or an update (interaction) by c that disclosed the role id ("" for none: no id from
// the request is stored). A search is an execute (FHIR AuditEvent.action), a read a read, a create a create.
func (s *Service) record(ctx context.Context, c Caller, interaction, id string, outcome auditevent.Outcome) error {
	action := auditevent.ActionRead
	switch interaction {
	case auditevent.CodeSearchType:
		action = auditevent.ActionExecute
	case auditevent.CodeCreate:
		action = auditevent.ActionCreate
	case auditevent.CodeUpdate:
		action = auditevent.ActionUpdate
	}
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
		e.EntityReference = "PractitionerRole/" + id
	}
	return s.audit.Record(ctx, e)
}
