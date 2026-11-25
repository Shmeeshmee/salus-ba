package diagnosticreport

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

// Visits gives a visit with the caller's access facts, in the caller's scope, unaudited (appointment.Service): the
// report's own event covers the read.
type Visits interface {
	Attendance(ctx context.Context, c appointment.Caller, id string) (appointment.Attendance, error)
}

// Auditor records an AuditEvent inside the caller's transaction (auditevent.Service).
type Auditor interface {
	Record(ctx context.Context, e auditevent.AuditEvent) error
}

// Transactor runs fn in a transaction carried by the context, joining one already there (postgres.Transactor).
type Transactor interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// Ports are what the Service consumes.
type Ports struct {
	Repo   Repository
	Visits Visits
	Audit  Auditor
	Tx     Transactor
	Files  Files // the PDFs (binary.Service)
	Clock  clock.Clock
}

// Service is the lab results' business rules. Who may do what (docs/clinic-cms/WAVE6.md U5, 6.0's record): the
// clinic's admin and the visit's practitioner attach, change and read a visit's results (access.Appointment: with an
// active care consent to attach; to change or read, also after a revocation for a past visit); only a practitioner
// releases one; a receptionist is refused every route (no clinical content); the patient reads their own once
// released. Every read and write is audited in its transaction, every refusal in a transaction of its own.
type Service struct {
	p Ports
}

// NewService builds a Service on p.
func NewService(p Ports) *Service {
	return &Service{p: p}
}

// Attach attaches a report to a visit of the staff's clinic, preliminary. ErrInvalid for a draft that does not fit,
// ErrForbidden for a receptionist, ErrNotFound for a visit the caller may not see or whose patient has no active care
// consent.
func (s *Service) Attach(ctx context.Context, c Staff, appointmentID string, d Draft) (Report, error) {
	if err := d.validate(); err != nil {
		return Report{}, err
	}
	var out Report
	err := s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		att, err := s.visit(ctx, c, appointmentID)
		switch {
		case err != nil:
			return err
		case att.Level != access.Full:
			return ErrNotFound
		}
		a := att.Appointment
		if out, err = s.p.Repo.Insert(ctx, d.report(Report{OrganizationID: a.OrganizationID, PatientID: a.PatientID, AppointmentID: a.ID}), c.AccountID); err != nil {
			return fmt.Errorf("diagnosticreport: attach: %w", err)
		}
		return s.p.Audit.Record(ctx, staffEvent(c, auditevent.CodeCreate, out.ID, auditevent.OutcomeSuccess))
	})
	if err != nil {
		return Report{}, s.refused(ctx, c.AccountID, c.OrganizationID, auditevent.CodeCreate, "", err)
	}
	return out, nil
}

// Change changes a preliminary report at its version: ErrReleased once released, ErrVersionConflict when it moved on.
func (s *Service) Change(ctx context.Context, c Staff, id string, d Draft) (Report, error) {
	if err := d.validate(); err != nil {
		return Report{}, err
	}
	var out Report
	err := s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		r, err := s.report(ctx, c, id)
		switch {
		case err != nil:
			return err
		case r.Status == StatusFinal:
			return ErrReleased
		case r.VersionID != d.Version:
			return ErrVersionConflict
		}
		out = d.report(r)
		out.VersionID = r.VersionID + 1
		ok, err := s.p.Repo.Update(ctx, out)
		switch {
		case err != nil:
			return fmt.Errorf("diagnosticreport: change: %w", err)
		case !ok:
			return ErrVersionConflict
		}
		return s.p.Audit.Record(ctx, staffEvent(c, auditevent.CodeUpdate, id, auditevent.OutcomeSuccess))
	})
	if err != nil {
		return Report{}, s.refused(ctx, c.AccountID, c.OrganizationID, auditevent.CodeUpdate, id, err)
	}
	return out, nil
}

// Release releases a report at its version: the practitioner's review, after which the patient sees it. ErrForbidden
// to anyone but a practitioner who may see the visit's results; a revocation since the report was written does not
// hold it up.
func (s *Service) Release(ctx context.Context, c Staff, id string, version int) (Report, error) {
	if c.PractitionerID == "" || c.Reception {
		return Report{}, s.refused(ctx, c.AccountID, c.OrganizationID, auditevent.CodeUpdate, id, ErrForbidden)
	}
	var out Report
	err := s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		r, err := s.report(ctx, c, id)
		switch {
		case err != nil:
			return err
		case r.Status == StatusFinal:
			return ErrReleased
		case r.VersionID != version:
			return ErrVersionConflict
		}
		at := s.p.Clock.Now().Truncate(time.Microsecond) // as Postgres stores it
		ok, err := s.p.Repo.Release(ctx, id, c.AccountID, version, at)
		switch {
		case err != nil:
			return fmt.Errorf("diagnosticreport: release: %w", err)
		case !ok:
			return ErrVersionConflict
		}
		out = r
		out.Status, out.ReleasedAt, out.VersionID = StatusFinal, &at, version+1
		return s.p.Audit.Record(ctx, staffEvent(c, auditevent.CodeUpdate, id, auditevent.OutcomeSuccess))
	})
	if err != nil {
		return Report{}, s.refused(ctx, c.AccountID, c.OrganizationID, auditevent.CodeUpdate, id, err)
	}
	return out, nil
}

// ForVisit returns a visit's reports, oldest first, to the staff who may see its results; each is audited as a
// search's match, none found once.
func (s *Service) ForVisit(ctx context.Context, c Staff, appointmentID string) ([]Report, error) {
	var out []Report
	err := s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		if _, err := s.visit(ctx, c, appointmentID); err != nil {
			return err
		}
		list, err := s.p.Repo.ForAppointment(ctx, appointmentID)
		if err != nil {
			return fmt.Errorf("diagnosticreport: for visit: %w", err)
		}
		if err := s.withPDFs(ctx, list); err != nil {
			return err
		}
		out = list
		return s.matches(ctx, list, func(id string) auditevent.AuditEvent {
			return staffEvent(c, auditevent.CodeSearchType, id, auditevent.OutcomeSuccess)
		})
	})
	if err != nil {
		return nil, s.refused(ctx, c.AccountID, c.OrganizationID, auditevent.CodeSearchType, "", err)
	}
	return out, nil
}

// Read returns one report of the staff's clinic to the staff who may see its visit's results.
func (s *Service) Read(ctx context.Context, c Staff, id string) (Report, error) {
	var out Report
	err := s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		r, err := s.report(ctx, c, id)
		if err != nil {
			return err
		}
		out = r
		return s.p.Audit.Record(ctx, staffEvent(c, auditevent.CodeRead, id, auditevent.OutcomeSuccess))
	})
	if err != nil {
		return Report{}, s.refused(ctx, c.AccountID, c.OrganizationID, auditevent.CodeRead, id, err)
	}
	return out, nil
}

// Mine returns the patient's released reports, newest first, each audited as a search's match (none found once). A
// patient without a profile has none.
func (s *Service) Mine(ctx context.Context, c Patient) ([]Report, error) {
	return s.MineAt(ctx, c, "")
}

// MineAt is Mine of the clinic org only, every clinic's when empty: the patient's export of one clinic reads, and
// audits, nothing of another (as consent.MineAt, docs/clinic-cms PX).
func (s *Service) MineAt(ctx context.Context, c Patient, org string) ([]Report, error) {
	if c.PatientID == "" {
		return []Report{}, nil
	}
	var out []Report
	err := s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.p.Repo.SetPatient(ctx, c.PatientID); err != nil {
			return err
		}
		list, err := s.p.Repo.ForPatient(ctx, c.PatientID)
		if err != nil {
			return fmt.Errorf("diagnosticreport: mine: %w", err)
		}
		if org != "" {
			list = slices.DeleteFunc(list, func(r Report) bool { return r.OrganizationID != org })
		}
		if err := s.withPDFs(ctx, list); err != nil {
			return err
		}
		out = list
		return s.matches(ctx, list, func(id string) auditevent.AuditEvent {
			return patientEvent(c, auditevent.CodeSearchType, id, auditevent.OutcomeSuccess)
		})
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// MyReport returns one of the patient's released reports; ErrNotFound (audited) for any other.
func (s *Service) MyReport(ctx context.Context, c Patient, id string) (Report, error) {
	var out Report
	err := s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		if c.PatientID == "" {
			return ErrNotFound
		}
		if err := s.p.Repo.SetPatient(ctx, c.PatientID); err != nil {
			return err
		}
		r, err := s.p.Repo.ByID(ctx, id)
		if err != nil {
			return err
		}
		one := []Report{r}
		if err := s.withPDFs(ctx, one); err != nil {
			return err
		}
		out = one[0]
		return s.p.Audit.Record(ctx, patientEvent(c, auditevent.CodeRead, id, auditevent.OutcomeSuccess))
	})
	if errors.Is(err, ErrNotFound) {
		return Report{}, s.refused(ctx, c.AccountID, "", auditevent.CodeRead, id, err)
	}
	return out, err
}

// visit is the staff's attendance of the visit in the clinic's scope, set here: ErrForbidden for a receptionist or an
// account that is neither admin nor practitioner, ErrNotFound when the caller may not see the visit.
func (s *Service) visit(ctx context.Context, c Staff, appointmentID string) (appointment.Attendance, error) {
	if c.Reception || (!c.Admin && c.PractitionerID == "") || c.OrganizationID == "" {
		return appointment.Attendance{}, ErrForbidden
	}
	if err := s.p.Repo.SetOrganization(ctx, c.OrganizationID); err != nil {
		return appointment.Attendance{}, err
	}
	att, err := s.p.Visits.Attendance(ctx, appointment.Caller{
		AccountID: c.AccountID, OrganizationID: c.OrganizationID, PractitionerID: c.PractitionerID, Admin: c.Admin,
	}, appointmentID)
	switch {
	case errors.Is(err, appointment.ErrNotFound):
		return appointment.Attendance{}, ErrNotFound
	case err != nil:
		return appointment.Attendance{}, fmt.Errorf("diagnosticreport: visit: %w", err)
	case att.Level == access.None:
		return appointment.Attendance{}, ErrNotFound
	}
	return att, nil
}

// report is the report id of the staff's clinic, when the caller may see its visit.
func (s *Service) report(ctx context.Context, c Staff, id string) (Report, error) {
	if c.Reception || (!c.Admin && c.PractitionerID == "") || c.OrganizationID == "" {
		return Report{}, ErrForbidden
	}
	if err := s.p.Repo.SetOrganization(ctx, c.OrganizationID); err != nil {
		return Report{}, err
	}
	r, err := s.p.Repo.ByID(ctx, id)
	if err != nil {
		return Report{}, err
	}
	if _, err := s.visit(ctx, c, r.AppointmentID); err != nil {
		return Report{}, err
	}
	one := []Report{r}
	if err := s.withPDFs(ctx, one); err != nil {
		return Report{}, err
	}
	return one[0], nil
}

// matches audits each report of list with event, or one event without an entity when there is none.
func (s *Service) matches(ctx context.Context, list []Report, event func(id string) auditevent.AuditEvent) error {
	if len(list) == 0 {
		return s.p.Audit.Record(ctx, event(""))
	}
	for _, r := range list {
		if err := s.p.Audit.Record(ctx, event(r.ID)); err != nil {
			return err
		}
	}
	return nil
}

// refusals are the errors refused audits.
var refusals = []error{ErrForbidden, ErrNotFound, ErrReleased, ErrVersionConflict, ErrInvalid, ErrTooLarge, ErrBadBody}

// refused audits a refusal (one of refusals: a draft that does not fit is refused before, unaudited) in a transaction
// of its own, since the call's rolled back, and returns err; any other error is returned as it is.
func (s *Service) refused(ctx context.Context, account, org, interaction, id string, err error) error {
	if !slices.ContainsFunc(refusals, func(r error) bool { return errors.Is(err, r) }) {
		return err
	}
	e := event(account, org, interaction, id, auditevent.OutcomeMinorFailure)
	if aerr := s.p.Tx.WithTx(ctx, func(ctx context.Context) error { return s.p.Audit.Record(ctx, e) }); aerr != nil {
		return fmt.Errorf("diagnosticreport: refuse: %w", aerr)
	}
	return err
}

func staffEvent(c Staff, interaction, id string, outcome auditevent.Outcome) auditevent.AuditEvent {
	return event(c.AccountID, c.OrganizationID, interaction, id, outcome)
}

func patientEvent(c Patient, interaction, id string, outcome auditevent.Outcome) auditevent.AuditEvent {
	return event(c.AccountID, "", interaction, id, outcome)
}

// event is the AuditEvent of an interaction with the report id (none for a search with no match or a refusal before
// one is known): for staff, treatment (TREAT) in their clinic; for the patient, their own request (PATRQT).
func event(account, org, interaction, id string, outcome auditevent.Outcome) auditevent.AuditEvent {
	action := map[string]auditevent.Action{
		auditevent.CodeCreate: auditevent.ActionCreate, auditevent.CodeRead: auditevent.ActionRead,
		auditevent.CodeSearchType: auditevent.ActionExecute, auditevent.CodeUpdate: auditevent.ActionUpdate,
	}[interaction]
	e := auditevent.AuditEvent{
		Type:           auditevent.Coding{System: auditevent.SystemAuditEventType, Code: auditevent.CodeRest},
		Subtype:        auditevent.Coding{System: auditevent.SystemRestfulInteraction, Code: interaction},
		Purpose:        auditevent.Coding{System: auditevent.SystemActReason, Code: auditevent.CodeTREAT},
		Action:         action,
		Outcome:        outcome,
		AgentAccountID: account,
		OrganizationID: org,
	}
	if org == "" {
		e.Purpose.Code = auditevent.CodePATRQT
	}
	if id != "" {
		e.EntityReference = "DiagnosticReport/" + id
	}
	return e
}
