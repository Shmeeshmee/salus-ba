package organization

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

const maxName = 200 // characters

// Transactor runs fn in a transaction carried by the context, joining one already there
// (postgres.Transactor).
type Transactor interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// Auditor records an AuditEvent inside the caller's transaction (auditevent.Service).
type Auditor interface {
	Record(ctx context.Context, e auditevent.AuditEvent) error
}

// Service holds the clinic rules: only staff of a clinic see it, only its admin changes it, only the operator
// command creates one. Reads are not audited (a clinic's name is not personal data); every write is, also
// when it is refused.
type Service struct {
	repo  Repository
	audit Auditor
	tx    Transactor
	clock clock.Clock
}

// NewService returns a Service on its ports.
func NewService(repo Repository, audit Auditor, tx Transactor, clk clock.Clock) *Service {
	return &Service{repo: repo, audit: audit, tx: tx, clock: clk}
}

// Read returns the clinic id to a member of it, or ErrNotFound to anyone else.
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
		o, err = s.repo.ByID(ctx, id)
		return err
	})
	switch {
	case errors.Is(err, ErrNotFound):
		return Organization{}, ErrNotFound
	case err != nil:
		return Organization{}, fmt.Errorf("organization: read: %w", err)
	}
	return o, nil
}

// Create adds a clinic. Only the operator command calls it (cmd/salusctl, connected as the owner: the
// application role may not insert a clinic); its audit event names the operator command as the agent. The
// name is trimmed; a blank one or one over 200 characters gives ErrInvalidName.
func (s *Service) Create(ctx context.Context, name string) (Organization, error) {
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > maxName {
		return Organization{}, ErrInvalidName
	}
	var o Organization
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		var err error
		if o, err = s.repo.Create(ctx, name); err != nil {
			return err
		}
		e := event(auditevent.CodeCreate, auditevent.ActionCreate, Caller{OrganizationID: o.ID}, o.ID, auditevent.OutcomeSuccess)
		e.AgentName = auditevent.AgentOperator
		return s.audit.Record(ctx, e)
	})
	if err != nil {
		return Organization{}, fmt.Errorf("organization: create: %w", err)
	}
	return o, nil
}

// Update renames the clinic id, at the version the caller last read, and returns it at its new version.
// Only the clinic's admin may: a member who is not gets ErrForbidden, anyone else ErrNotFound. A version that
// is not the current one gives ErrConflict.
func (s *Service) Update(ctx context.Context, c Caller, id, name string, version int32) (Organization, error) {
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > maxName {
		return Organization{}, ErrInvalidName
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
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.repo.SetOrganization(ctx, c.OrganizationID); err != nil {
			return err
		}
		var err error
		o, failed, err = s.rename(ctx, id, name, version)
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
	return o, nil
}

// rename reads the clinic and renames it when version is current. A missing clinic or a stale version is
// returned as failed, with the clinic read when there is one; err is any other error.
func (s *Service) rename(ctx context.Context, id, name string, version int32) (o Organization, failed, err error) {
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
	o, err = s.repo.Rename(ctx, id, name, version, s.clock.Now())
	if errors.Is(err, ErrConflict) { // changed since the read
		return cur, ErrConflict, nil
	}
	return o, nil, err
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
