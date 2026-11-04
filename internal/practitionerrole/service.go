package practitionerrole

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

// Service holds the practitioner role rules: only staff of a clinic see its roles, and every read and
// search is audited, also when it is refused.
type Service struct {
	repo  Repository
	audit Auditor
	tx    Transactor
}

// NewService returns a Service on its ports.
func NewService(repo Repository, audit Auditor, tx Transactor) *Service {
	return &Service{repo: repo, audit: audit, tx: tx}
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

// record audits a read or a search (interaction) by c that disclosed the role id ("" for none: no id from
// the request is stored). A search is an execute (FHIR AuditEvent.action), a read a read.
func (s *Service) record(ctx context.Context, c Caller, interaction, id string, outcome auditevent.Outcome) error {
	action := auditevent.ActionRead
	if interaction == auditevent.CodeSearchType {
		action = auditevent.ActionExecute
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
