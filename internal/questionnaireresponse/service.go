package questionnaireresponse

import (
	"context"
	"errors"
	"fmt"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/binary"
	"github.com/Shmeeshmee/salus/internal/questionnaire"
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

// Appointments gives a response its appointment with the caller's access facts, read in the caller's scope
// and unaudited, and to a job the appointment in the transaction's scope, unaudited; LookupLocked also locks
// it against a cancellation until the transaction ends, for a write that rests on it being booked
// (appointment.Service).
type Appointments interface {
	Attendance(ctx context.Context, c appointment.Caller, id string) (appointment.Attendance, error)
	Lookup(ctx context.Context, id string) (appointment.Appointment, error)
	LookupLocked(ctx context.Context, id string) (appointment.Appointment, error)
}

// Questionnaires reads a version of the caller's clinic, or one a patient's own response references, audited
// as the caller's read of it, and the active version of a role's default for a job, audited with the job as
// agent (questionnaire.Service).
type Questionnaires interface {
	Read(ctx context.Context, c questionnaire.Caller, id string) (questionnaire.Questionnaire, error)
	Referenced(ctx context.Context, c questionnaire.Caller, patientID, id string) (questionnaire.Questionnaire, error)
	ActiveDefault(ctx context.Context, agent, org, roleID string) (questionnaire.Questionnaire, error)
}

// Ports are the Service's dependencies.
type Ports struct {
	Repo           Repository
	Audit          Auditor
	Tx             Transactor
	Clock          clock.Clock
	Appointments   Appointments
	Questionnaires Questionnaires
	Jobs           Jobs
}

// Service holds the questionnaire response rules (SPEC.md §2 "Requests", "Access rule", "Audit").
type Service struct{ Ports }

// NewService returns a Service on its ports.
func NewService(p Ports) *Service {
	return &Service{Ports: p}
}

// Request asks the appointment's patient to fill in the questionnaire version questionnaireID, as the
// appointment's practitioner (access.RequestsQuestionnaire: with the patient's active consent), and returns
// the response: in progress, no answers, due at the appointment's start (source added). The appointment must
// be booked and not started, the version an active one of the clinic (read through the questionnaire's
// service, which audits that read), and the appointment without a live request of its questionnaire. In one
// transaction in the clinic's scope, which locks the appointment against a cancellation (a cancellation that
// committed first refuses the request; one that comes later waits, and its discard job deletes the response),
// audited as a create. ErrNotFound to anyone else and for an appointment or
// a version the caller may not see; ErrConsentRequired, ErrAppointmentClosed, ErrAppointmentStarted,
// ErrNotActive, ErrAlreadyRequested; each refusal is audited.
func (s *Service) Request(ctx context.Context, c appointment.Caller, appointmentID, questionnaireID string) (Response, error) {
	var out Response
	var refused error
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		var err error
		if out, refused, err = s.request(ctx, c, appointmentID, questionnaireID); err != nil {
			return err
		}
		if refused != nil {
			return s.record(ctx, c, auditevent.CodeCreate, "", auditevent.OutcomeMinorFailure)
		}
		return s.record(ctx, c, auditevent.CodeCreate, out.ID, auditevent.OutcomeSuccess)
	})
	switch {
	case err != nil:
		return Response{}, fmt.Errorf("questionnaireresponse: request: %w", err)
	case refused != nil:
		return Response{}, refused
	}
	return out, nil
}

// request makes the request in the transaction in ctx, or gives the refusal.
func (s *Service) request(ctx context.Context, c appointment.Caller, appointmentID, questionnaireID string) (out Response, refused, err error) {
	if c.OrganizationID == "" {
		return Response{}, ErrNotFound, nil
	}
	if err := s.Repo.SetClinic(ctx, c.OrganizationID); err != nil {
		return Response{}, nil, err
	}
	att, ok, err := s.attendance(ctx, c, appointmentID)
	if err != nil || !ok {
		return Response{}, ErrNotFound, err
	}
	switch v, f, visit := rule(c, att); {
	case !access.RequestsQuestionnaire(v, f, visit) && att.Practitioner && !att.ConsentActive:
		return Response{}, ErrConsentRequired, nil
	case !access.RequestsQuestionnaire(v, f, visit):
		return Response{}, ErrNotFound, nil
	}
	a, refused, err := s.booked(ctx, appointmentID)
	if err != nil || refused != nil {
		return Response{}, refused, err
	}
	now := s.Clock.Now()
	if !now.Before(a.Start) {
		return Response{}, ErrAppointmentStarted, nil
	}
	q, err := s.Questionnaires.Read(ctx, questionnaire.Caller{AccountID: c.AccountID, OrganizationID: c.OrganizationID, PractitionerID: c.PractitionerID}, questionnaireID)
	switch {
	case errors.Is(err, questionnaire.ErrNotFound):
		return Response{}, ErrNotFound, nil
	case err != nil:
		return Response{}, nil, err
	case q.Status != questionnaire.StatusActive:
		return Response{}, ErrNotActive, nil
	}
	out, created, err := s.Repo.Insert(ctx, Response{
		OrganizationID: a.OrganizationID, AppointmentID: a.ID, PatientID: a.PatientID, PractitionerRoleID: a.PractitionerRoleID,
		QuestionnaireID: q.ID, SeriesID: q.SeriesID, Source: SourceAdded, DueAt: a.Start,
	}, now)
	switch {
	case err != nil:
		return Response{}, nil, err
	case !created:
		return Response{}, ErrAlreadyRequested, nil
	}
	return out, nil, nil
}

// booked locks the appointment id in the transaction in ctx against a change and gives it, or the refusal:
// ErrAppointmentClosed once it is not booked (a cancellation that committed first included), ErrNotFound when
// the scope may not lock it.
func (s *Service) booked(ctx context.Context, id string) (a appointment.Appointment, refused, err error) {
	a, err = s.Appointments.LookupLocked(ctx, id)
	switch {
	case errors.Is(err, appointment.ErrNotFound):
		return appointment.Appointment{}, ErrNotFound, nil
	case err != nil:
		return appointment.Appointment{}, nil, err
	case a.Status != appointment.StatusBooked:
		return appointment.Appointment{}, ErrAppointmentClosed, nil
	}
	return a, nil, nil
}

// Withdraw withdraws the response id, as the practitioner of its appointment (access.RequestsQuestionnaire),
// while it is not submitted, and returns it: stopped, its answers and its list of files cleared at once, and
// the binary_discard job enqueued to delete the files. In one transaction, audited as an update. ErrNotFound to anyone who may not see it, ErrForbidden to anyone else who sees it
// (the patient, the clinic's admin), ErrSubmitted and ErrWithdrawn; each refusal is audited.
func (s *Service) Withdraw(ctx context.Context, c appointment.Caller, id string) (Response, error) {
	var out Response
	var refused error
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		var err error
		var seen bool
		if out, seen, refused, err = s.withdraw(ctx, c, id); err != nil {
			return err
		}
		switch {
		case refused != nil && seen:
			return s.record(ctx, c, auditevent.CodeUpdate, out.ID, auditevent.OutcomeMinorFailure)
		case refused != nil:
			return s.record(ctx, c, auditevent.CodeUpdate, "", auditevent.OutcomeMinorFailure)
		}
		return s.record(ctx, c, auditevent.CodeUpdate, out.ID, auditevent.OutcomeSuccess)
	})
	switch {
	case err != nil:
		return Response{}, fmt.Errorf("questionnaireresponse: withdraw: %w", err)
	case refused != nil:
		return Response{}, refused
	}
	return out, nil
}

// withdraw withdraws the response in the transaction in ctx, or gives the refusal and whether the caller
// sees the response it refused. The response is read to decide, and locked only for a withdrawal the caller
// may make: after a revocation the clinic still sees a submitted one (D9) but its scope may not lock it.
func (s *Service) withdraw(ctx context.Context, c appointment.Caller, id string) (out Response, seen bool, refused, err error) {
	r, att, level, err := s.view(ctx, c, id, s.Repo.ByID)
	switch {
	case err != nil:
		return Response{}, false, nil, err
	case level == access.None:
		return Response{}, false, ErrNotFound, nil
	}
	if v, f, visit := rule(c, att); !access.RequestsQuestionnaire(v, f, visit) {
		return r, true, ErrForbidden, nil
	}
	switch r, err = s.Repo.Lock(ctx, r.ID); {
	case errors.Is(err, ErrNotFound): // discarded or out of the scope since it was read
		return Response{}, false, ErrNotFound, nil
	case err != nil:
		return Response{}, false, nil, err
	case r.Status == StatusCompleted:
		return r, true, ErrSubmitted, nil
	case r.Status == StatusStopped:
		return r, true, ErrWithdrawn, nil
	}
	if out, err = s.Repo.Stop(ctx, r.ID, s.Clock.Now()); err != nil {
		return Response{}, false, nil, err
	}
	// The files went with the answers; the job deletes them (docs/attachments/SPEC.md §2 "Discard").
	return out, true, nil, s.Jobs.EnqueueDiscard(ctx, binary.DiscardArgs{ResponseID: out.ID, OrganizationID: out.OrganizationID})
}

// view reads the response id with read in the caller's scope, the patient's own or the staff caller's
// clinic, in the transaction in ctx, and decides the caller's level through access; a response the caller may
// not see (or none, or a patient without a profile) is at None. The check does not rest on row-level security
// alone.
func (s *Service) view(ctx context.Context, c appointment.Caller, id string, read func(context.Context, string) (Response, error)) (Response, appointment.Attendance, access.Level, error) {
	ok, err := s.scope(ctx, c)
	if err != nil || !ok {
		return Response{}, appointment.Attendance{}, access.None, err
	}
	r, err := read(ctx, id)
	switch {
	case errors.Is(err, ErrNotFound):
		return Response{}, appointment.Attendance{}, access.None, nil
	case err != nil:
		return Response{}, appointment.Attendance{}, access.None, err
	}
	att, ok, err := s.attendance(ctx, c, r.AppointmentID)
	if err != nil || !ok {
		return Response{}, appointment.Attendance{}, access.None, err
	}
	return r, att, level(c, att, r), nil
}

// scope scopes the transaction in ctx to the caller: a patient's own, or the staff caller's clinic; false for
// a patient without a profile, who has none.
func (s *Service) scope(ctx context.Context, c appointment.Caller) (bool, error) {
	if c.OrganizationID != "" {
		return true, s.Repo.SetClinic(ctx, c.OrganizationID)
	}
	if c.PatientID == "" {
		return false, nil
	}
	return true, s.Repo.SetPatient(ctx, c.PatientID)
}

// attendance gives the caller's attendance of the appointment id, and false when they may not see it.
func (s *Service) attendance(ctx context.Context, c appointment.Caller, id string) (appointment.Attendance, bool, error) {
	att, err := s.Appointments.Attendance(ctx, c, id)
	switch {
	case errors.Is(err, appointment.ErrNotFound):
		return appointment.Attendance{}, false, nil
	case err != nil:
		return appointment.Attendance{}, false, err
	}
	return att, true, nil
}

// rule is the caller, the patient's facts and the appointment as access sees them, from the caller's
// attendance of the appointment.
func rule(c appointment.Caller, att appointment.Attendance) (access.Viewer, access.Facts, access.Visit) {
	a := att.Appointment
	visit := access.Visit{PractitionerRole: a.PractitionerRoleID, Start: a.Start}
	f := access.Facts{
		Patient: a.PatientID, Clinic: a.OrganizationID, Visits: []access.Visit{visit},
		Consent: access.Consent{Active: att.ConsentActive, RevokedAt: att.RevokedAt},
	}
	var v access.Viewer
	switch {
	case att.Patient:
		v = access.Viewer{Role: access.RolePatient, Patient: c.PatientID}
	case c.Admin:
		v = access.Viewer{Role: access.RoleAdmin, Clinic: c.OrganizationID}
	default:
		v = access.Viewer{Role: access.RolePractitioner, Clinic: c.OrganizationID}
		if att.Practitioner {
			v.PractitionerRole = a.PractitionerRoleID
		}
	}
	return v, f, visit
}

// level is what the caller may see of r (access.QuestionnaireResponse).
func level(c appointment.Caller, att appointment.Attendance, r Response) access.Level {
	v, f, visit := rule(c, att)
	return access.QuestionnaireResponse(v, f, visit, r.Status == StatusCompleted)
}

// record audits interaction by c on QuestionnaireResponse/id ("" when none was disclosed: no id from the
// request is stored). The purpose is the patient's own request for a patient and treatment for staff.
func (s *Service) record(ctx context.Context, c appointment.Caller, interaction, id string, outcome auditevent.Outcome) error {
	return s.Audit.Record(ctx, callerEvent(c, interaction, id, outcome))
}

// callerEvent is the audit event record writes.
func callerEvent(c appointment.Caller, interaction, id string, outcome auditevent.Outcome) auditevent.AuditEvent {
	purpose := auditevent.CodePATRQT
	if c.OrganizationID != "" {
		purpose = auditevent.CodeTREAT
	}
	action := map[string]auditevent.Action{
		auditevent.CodeCreate: auditevent.ActionCreate, auditevent.CodeUpdate: auditevent.ActionUpdate,
		auditevent.CodeSearchType: auditevent.ActionExecute,
	}[interaction]
	if action == "" {
		action = auditevent.ActionRead
	}
	e := auditevent.AuditEvent{
		Type:           auditevent.Coding{System: auditevent.SystemAuditEventType, Code: auditevent.CodeRest},
		Subtype:        auditevent.Coding{System: auditevent.SystemRestfulInteraction, Code: interaction},
		Purpose:        auditevent.Coding{System: auditevent.SystemActReason, Code: purpose},
		Action:         action,
		Outcome:        outcome,
		AgentAccountID: c.AccountID,
		OrganizationID: c.OrganizationID,
	}
	if id != "" {
		e.EntityReference = "QuestionnaireResponse/" + id
	}
	return e
}
