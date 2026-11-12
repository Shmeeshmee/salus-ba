package clinicstats

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/Shmeeshmee/salus/internal/auditevent"
)

// maxRange is the longest range a query may span: ten years of local days.
const maxRange = 3653

// breakdowns are the diagnosis breakdowns a query may ask for.
var breakdowns = []Breakdown{ByGroup, BySubgroup, ByCode, ByPractitionerRole, BySpecialty, ByMonth}

// Auditor records an AuditEvent inside the caller's transaction (auditevent.Service).
type Auditor interface {
	Record(ctx context.Context, e auditevent.AuditEvent) error
}

// Ports are the Service's dependencies, on salus_app.
type Ports struct {
	Repo  Repository
	Audit Auditor
	Tx    Transactor
}

// Service reads a clinic's figures for its staff (rules C1 to C3, docs/plans/analytics.md §4): only their own
// clinic's, in its scope; an admin everything of it; a practitioner who is not an admin the clinic's appointment
// figures (or their own role's) and only their own role's diagnoses. Every read and refusal is audited.
type Service struct{ p Ports }

// NewService returns the Service on its ports.
func NewService(p Ports) *Service { return &Service{p: p} }

// Appointments returns the clinic's appointments by month, the heatmap of their weekdays and hours and, for an
// admin, the fulfilled ones per role. ErrInvalidQuery for a bad range (not audited: nothing is read);
// ErrForbidden, audited, to a caller who is neither the clinic's admin nor a practitioner of it, and to a
// practitioner who is not an admin asking for another role or for a specialty.
func (s *Service) Appointments(ctx context.Context, c Caller, q AppointmentQuery) (AppointmentFigures, error) {
	if !validRange(q.From, q.To) {
		return AppointmentFigures{}, ErrInvalidQuery
	}
	if !staff(c) || (!c.Admin && !ownAppointmentFilter(c, q)) {
		return AppointmentFigures{}, s.refuse(ctx, c)
	}
	out := AppointmentFigures{PerPractitioner: c.Admin}
	err := s.read(ctx, c, func(ctx context.Context) error {
		var err error
		if out.Series, err = s.p.Repo.Series(ctx, c.OrganizationID, q); err != nil {
			return err
		}
		if out.Heatmap, err = s.p.Repo.Heatmap(ctx, c.OrganizationID, q); err != nil {
			return err
		}
		if c.Admin {
			out.Practitioners, err = s.p.Repo.Practitioners(ctx, c.OrganizationID, q)
		}
		return err
	})
	if err != nil {
		return AppointmentFigures{}, fmt.Errorf("clinicstats: appointments: %w", err)
	}
	return out, nil
}

// Diagnoses returns the clinic's confirmed diagnoses counted by the query's breakdown, most first: for an admin the
// clinic's, for a practitioner who is not an admin their own role's only. ErrInvalidQuery for a bad range or
// breakdown (not audited); ErrForbidden, audited, to a caller who is neither admin nor practitioner, to a
// practitioner without a role and to one asking for another role.
func (s *Service) Diagnoses(ctx context.Context, c Caller, q DiagnosisQuery) ([]Count, error) {
	if !validRange(q.From, q.To) || !slices.Contains(breakdowns, q.By) {
		return nil, ErrInvalidQuery
	}
	if !c.Admin {
		if !staff(c) || c.PractitionerRoleID == "" || (q.PractitionerRoleID != "" && q.PractitionerRoleID != c.PractitionerRoleID) {
			return nil, s.refuse(ctx, c)
		}
		q.PractitionerRoleID = c.PractitionerRoleID
	}
	var out []Count
	err := s.read(ctx, c, func(ctx context.Context) error {
		var err error
		out, err = s.p.Repo.Diagnoses(ctx, c.OrganizationID, q)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("clinicstats: diagnoses: %w", err)
	}
	return out, nil
}

// ownAppointmentFilter reports whether q narrows a non-admin's appointment figures at most to their own role (C2).
// A specialty is never theirs to filter by: in a clinic with one cardiologist it is that colleague's figures, and
// with their own role's subtracted it is the others' of their specialty.
func ownAppointmentFilter(c Caller, q AppointmentQuery) bool {
	return q.Specialty == "" && (q.PractitionerRoleID == "" || q.PractitionerRoleID == c.PractitionerRoleID)
}

// staff reports whether c is the admin or a practitioner of a clinic.
func staff(c Caller) bool { return c.OrganizationID != "" && (c.Admin || c.Practitioner) }

// validRange reports whether from to to is a range of at most maxRange days.
func validRange(from, to time.Time) bool {
	return !from.IsZero() && !to.IsZero() && !to.Before(from) && to.Sub(from) < maxRange*24*time.Hour
}

// read runs fn in a transaction in the caller's clinic's scope and audits the read in it.
func (s *Service) read(ctx context.Context, c Caller, fn func(ctx context.Context) error) error {
	return s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.p.Repo.SetClinic(ctx, c.OrganizationID); err != nil {
			return err
		}
		if err := fn(ctx); err != nil {
			return err
		}
		return s.p.Audit.Record(ctx, event(c, auditevent.OutcomeSuccess))
	})
}

// refuse audits a refused read by c in a transaction of its own and returns ErrForbidden.
func (s *Service) refuse(ctx context.Context, c Caller) error {
	err := s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		return s.p.Audit.Record(ctx, event(c, auditevent.OutcomeMinorFailure))
	})
	if err != nil {
		return fmt.Errorf("clinicstats: refuse: %w", err)
	}
	return ErrForbidden
}

// event is the AuditEvent of a read of c's clinic's figures (C3): a search of Organization/<clinic> for the
// clinic's operations (v3-ActReason HOPERAT; the code system has no code for statistics).
func event(c Caller, outcome auditevent.Outcome) auditevent.AuditEvent {
	e := auditevent.AuditEvent{
		Type:           auditevent.Coding{System: auditevent.SystemAuditEventType, Code: auditevent.CodeRest},
		Subtype:        auditevent.Coding{System: auditevent.SystemRestfulInteraction, Code: auditevent.CodeSearchType},
		Purpose:        auditevent.Coding{System: auditevent.SystemActReason, Code: auditevent.CodeHOPERAT},
		Action:         auditevent.ActionExecute,
		Outcome:        outcome,
		AgentAccountID: c.AccountID,
		OrganizationID: c.OrganizationID,
	}
	if c.OrganizationID != "" {
		e.EntityReference = "Organization/" + c.OrganizationID
	}
	return e
}
