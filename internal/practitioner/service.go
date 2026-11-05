package practitioner

import (
	"context"
	"errors"
	"fmt"

	"github.com/Shmeeshmee/salus/internal/auditevent"
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

// Service holds the practitioner rules: only staff of a practitioner's clinic see them, and every read and
// create by staff is audited, also when it is refused. Anyone may read the names of the active
// practitioners of a listed clinic (PublicNames); that is not audited.
type Service struct {
	repo  Repository
	audit Auditor
	tx    Transactor
}

// NewService returns a Service on its ports.
func NewService(repo Repository, audit Auditor, tx Transactor) *Service {
	return &Service{repo: repo, audit: audit, tx: tx}
}

// Read returns the practitioner id to staff of their clinic, or ErrNotFound to anyone else.
func (s *Service) Read(ctx context.Context, c Caller, id string) (Practitioner, error) {
	if c.OrganizationID == "" {
		if err := s.record(ctx, c, auditevent.CodeRead, "", auditevent.OutcomeMinorFailure); err != nil {
			return Practitioner{}, fmt.Errorf("practitioner: read: %w", err)
		}
		return Practitioner{}, ErrNotFound
	}
	var p Practitioner
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
		return Practitioner{}, fmt.Errorf("practitioner: read: %w", err)
	case missing:
		return Practitioner{}, ErrNotFound
	}
	return p, nil
}

// Create adds an active practitioner to the caller's clinic and returns it (on invite acceptance, as the
// invited account). A caller without a clinic gets ErrForbidden. Audited, also when refused.
func (s *Service) Create(ctx context.Context, c Caller, givenName, familyName string) (Practitioner, error) {
	if c.OrganizationID == "" {
		if err := s.record(ctx, c, auditevent.CodeCreate, "", auditevent.OutcomeMinorFailure); err != nil {
			return Practitioner{}, fmt.Errorf("practitioner: create: %w", err)
		}
		return Practitioner{}, ErrForbidden
	}
	var p Practitioner
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.repo.SetOrganization(ctx, c.OrganizationID); err != nil {
			return err
		}
		var err error
		if p, err = s.repo.Create(ctx, c.OrganizationID, givenName, familyName); err != nil {
			return err
		}
		return s.record(ctx, c, auditevent.CodeCreate, p.ID, auditevent.OutcomeSuccess)
	})
	if err != nil {
		return Practitioner{}, fmt.Errorf("practitioner: create: %w", err)
	}
	return p, nil
}

// PublicNames returns the active practitioners among ids whose clinic is listed, with their name only (no
// clinic): the public directory, for anyone, signed in or not. It holds only what the clinic chose to
// publish, so it is not audited (docs/booking/SPEC.md "Public browse").
func (s *Service) PublicNames(ctx context.Context, ids []string) ([]Practitioner, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var out []Practitioner
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.repo.SetDirectory(ctx); err != nil {
			return err
		}
		all, err := s.repo.PublicByIDs(ctx, ids)
		for _, p := range all {
			if p.Active { // not on row-level security alone
				out = append(out, p)
			}
		}
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("practitioner: public names: %w", err)
	}
	return out, nil
}

// record audits a read or a create (interaction) by c of the practitioner id ("" when none was disclosed: no
// id from the request is stored).
func (s *Service) record(ctx context.Context, c Caller, interaction, id string, outcome auditevent.Outcome) error {
	action := auditevent.ActionRead
	if interaction == auditevent.CodeCreate {
		action = auditevent.ActionCreate
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
		e.EntityReference = "Practitioner/" + id
	}
	return s.audit.Record(ctx, e)
}
