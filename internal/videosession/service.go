package videosession

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/encounter"
	"github.com/Shmeeshmee/salus/internal/patient"
	"github.com/Shmeeshmee/salus/internal/practitioner"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

// The join window, the hard cap and the token (SPEC.md §2 "Join window", "Ending", "Token grant").
const (
	opensBefore     = 10 * time.Minute // the window opens before the start
	closesAfter     = 30 * time.Minute // and closes after the scheduled end
	hardCapAfter    = 2 * time.Hour    // the room is ended after the scheduled end, whoever is in it
	tokenValidFor   = 5 * time.Minute
	maxParticipants = 2
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

// Appointments gives the appointment, with the caller's access facts for a join or in the caller's scope for
// the webhook, and the practitioner of its role in the caller's scope for the room rule, in the caller's
// transaction, unaudited (appointment.Service).
type Appointments interface {
	Attendance(ctx context.Context, c appointment.Caller, id string) (appointment.Attendance, error)
	Lookup(ctx context.Context, id string) (appointment.Appointment, error)
	LookupPractitioner(ctx context.Context, id string) (string, error)
}

// Consents gives the state of a patient's consent with a clinic, in the caller's transaction and scope,
// unaudited (consent.Service).
type Consents interface {
	State(ctx context.Context, patientID, organizationID string) (access.Consent, error)
}

// Patients reads a profile in the caller's transaction and scope, unaudited (patient.Service).
type Patients interface {
	Lookup(ctx context.Context, id string) (patient.Patient, error)
}

// Practitioners gives the public names of practitioners, as the directory shows them (practitioner.Service).
type Practitioners interface {
	PublicNames(ctx context.Context, ids []string) ([]practitioner.Practitioner, error)
}

// Encounters starts and finishes the consultation's Encounter, and through it records the appointment's
// outcome, in the caller's transaction and clinic scope (encounter.Service).
type Encounters interface {
	Start(ctx context.Context, agent, appointmentID string, at time.Time) (encounter.Encounter, error)
	Finish(ctx context.Context, agent, appointmentID string, at time.Time, patientJoined bool) error
}

// Ports are the Service's dependencies. PublicURL is the LiveKit URL the clients connect to.
type Ports struct {
	Repo          Repository
	Provider      Provider
	Jobs          Jobs
	Encounters    Encounters
	Audit         Auditor
	Tx            Transactor
	Clock         clock.Clock
	Appointments  Appointments
	Consents      Consents
	Patients      Patients
	Practitioners Practitioners
	Listener      Listener
	PublicURL     string
}

// Service holds the consultation's room rules (SPEC.md §2 "Who may join", "Join response").
type Service struct{ Ports }

// NewService returns a Service on its ports.
func NewService(p Ports) *Service {
	return &Service{Ports: p}
}

// Join gives the caller a token for the appointment's room: to its patient, or to the practitioner of its
// role, while it is booked, the patient's consent with the clinic is active, the time is inside the window
// (10 minutes before the start to 30 minutes after the end) and the session has not ended. The first request
// stores the session and, in its transaction, enqueues the window's close; the room is created on every request, outside the transaction (it is idempotent, so a
// restarted LiveKit heals on the next request). The practitioner gets the patient's name at the access level
// (a Patient read, audited), the patient the practitioner's public name. ErrNotFound to anyone else,
// ErrConsentRequired, ErrJoinWindowNotOpen and ErrConsultationEnded; every outcome is audited.
func (s *Service) Join(ctx context.Context, c appointment.Caller, appointmentID string) (Join, error) {
	now := s.Clock.Now()
	var t ticket
	var refused error
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		var err error
		t, refused, err = s.join(ctx, c, appointmentID, now)
		return err
	})
	switch {
	case err != nil:
		return Join{}, fmt.Errorf("videosession: join: %w", err)
	case refused != nil:
		return Join{}, refused
	}
	if t.role == RolePatient {
		if t.peerName, err = s.publicName(ctx, t.practitionerID); err != nil {
			return Join{}, fmt.Errorf("videosession: join: %w", err)
		}
	}
	room := Room{Name: t.session.Room, MaxParticipants: maxParticipants, EmptyTimeout: t.hardCap.Sub(now)}
	if err := s.Provider.CreateRoom(ctx, room); err != nil {
		return Join{}, fmt.Errorf("videosession: join: %w", err)
	}
	identity := t.session.PatientIdentity
	if t.role == RolePractitioner {
		identity = t.session.PractitionerIdentity
	}
	token, err := s.Provider.Token(Grant{Room: t.session.Room, Identity: identity, ValidFor: tokenValidFor})
	if err != nil {
		return Join{}, fmt.Errorf("videosession: join: %w", err)
	}
	return Join{URL: s.PublicURL, Token: token, ExpiresAt: now.Add(tokenValidFor), PeerName: t.peerName, Role: t.role}, nil
}

// ticket is what the join's transaction decided.
type ticket struct {
	hardCap        time.Time
	session        Session
	practitionerID string
	peerName       string
	role           Role
}

// join decides the caller's join in the transaction in ctx, opens the session and audits, or gives the
// refusal (audited too).
func (s *Service) join(ctx context.Context, c appointment.Caller, id string, now time.Time) (t ticket, refused, err error) {
	att, err := s.Appointments.Attendance(ctx, c, id)
	if errors.Is(err, appointment.ErrNotFound) {
		return ticket{}, ErrNotFound, s.record(ctx, c, auditevent.CodeOperation, "", auditevent.OutcomeMinorFailure)
	}
	if err != nil {
		return ticket{}, nil, err
	}
	entity := "Appointment/" + att.Appointment.ID
	role, refused := decide(att, now)
	if errors.Is(refused, ErrConsultationEnded) {
		if refused, err = s.ended(ctx, c, att.Appointment.ID); err != nil {
			return ticket{}, nil, err
		}
	}
	if refused != nil {
		return ticket{}, refused, s.record(ctx, c, auditevent.CodeOperation, entity, auditevent.OutcomeMinorFailure)
	}
	session, err := s.open(ctx, c, att.Appointment, now)
	if err != nil {
		return ticket{}, nil, err
	}
	if !session.EndedAt.IsZero() {
		return ticket{}, session.refusal(), s.record(ctx, c, auditevent.CodeOperation, entity, auditevent.OutcomeMinorFailure)
	}
	t = ticket{session: session, role: role, practitionerID: att.PractitionerID, hardCap: att.Appointment.End.Add(hardCapAfter)}
	if role == RolePractitioner {
		if t.peerName, err = s.patientName(ctx, c, att); err != nil {
			return ticket{}, nil, err
		}
	}
	return t, nil, s.record(ctx, c, auditevent.CodeOperation, entity, auditevent.OutcomeSuccess)
}

// decide gives the caller's role in the appointment of att at now, or why they may not join: not a
// participant, no active consent (checked first: after a revocation the appointment may be cancelled too),
// no longer booked, or outside the window.
func decide(att appointment.Attendance, now time.Time) (Role, error) {
	a := att.Appointment
	switch {
	case !att.Patient && !att.Practitioner:
		return "", ErrNotFound
	case !att.ConsentActive:
		return "", ErrConsentRequired
	case a.Status != appointment.StatusBooked:
		return "", ErrConsultationEnded
	case now.Before(a.Start.Add(-opensBefore)):
		return "", ErrJoinWindowNotOpen
	case !now.Before(a.End.Add(closesAfter)):
		return "", ErrConsultationEnded
	case att.Patient:
		return RolePatient, nil
	}
	return RolePractitioner, nil
}

// open returns the appointment a's session, stored now with random names unless it has one, in the
// caller's scope; a session it stores gets its window-close job, enqueued in the same transaction.
func (s *Service) open(ctx context.Context, c appointment.Caller, a appointment.Appointment, now time.Time) (Session, error) {
	if err := s.scope(ctx, c); err != nil {
		return Session{}, err
	}
	session, created, err := s.Repo.Open(ctx, Session{
		AppointmentID: a.ID, OrganizationID: a.OrganizationID, ClosesAt: a.End.Add(closesAfter), CreatedAt: now,
		Room: randomName("r_"), PatientIdentity: randomName("p_"), PractitionerIdentity: randomName("p_"),
	})
	if err != nil || !created {
		return session, err
	}
	return session, s.Jobs.EnqueueWindowClose(ctx, session.ID, session.OrganizationID, session.ClosesAt)
}

// scope scopes the transaction in ctx to the caller: the patient's own sessions, or the staff caller's clinic.
func (s *Service) scope(ctx context.Context, c appointment.Caller) error {
	if c.OrganizationID == "" {
		return s.Repo.SetPatient(ctx, c.PatientID)
	}
	return s.Repo.SetClinic(ctx, c.OrganizationID)
}

// ended is why the ended consultation of the appointment id refuses the caller's join, read in their scope:
// ErrPractitionerDidNotJoin when its session ended without the practitioner, else ErrConsultationEnded (also
// with no session). It opens none.
func (s *Service) ended(ctx context.Context, c appointment.Caller, id string) (error, error) {
	if err := s.scope(ctx, c); err != nil {
		return nil, err
	}
	session, err := s.Repo.ByAppointment(ctx, id)
	switch {
	case errors.Is(err, ErrNotFound):
		return ErrConsultationEnded, nil
	case err != nil:
		return nil, err
	}
	return session.refusal(), nil
}

// randomName is prefix and 128 random bits in lower-case base32, derived from nothing of ours.
func randomName(prefix string) string {
	return prefix + strings.ToLower(rand.Text())
}

// patientName is the patient's name as the practitioner may see it, audited as a Patient read; empty when
// the access level gives none.
func (s *Service) patientName(ctx context.Context, c appointment.Caller, att appointment.Attendance) (string, error) {
	if att.Profile == access.None {
		return "", nil
	}
	p, err := s.Patients.Lookup(ctx, att.Appointment.PatientID)
	if err != nil {
		return "", err
	}
	if err := s.record(ctx, c, auditevent.CodeRead, "Patient/"+p.ID, auditevent.OutcomeSuccess); err != nil {
		return "", err
	}
	return p.GivenName + " " + p.FamilyName, nil
}

// publicName is the practitioner's name as the public directory shows it; empty when it does not.
func (s *Service) publicName(ctx context.Context, practitionerID string) (string, error) {
	all, err := s.Practitioners.PublicNames(ctx, []string{practitionerID})
	if err != nil {
		return "", err
	}
	for _, p := range all {
		if p.ID == practitionerID {
			return p.GivenName + " " + p.FamilyName, nil
		}
	}
	return "", nil
}

// record audits interaction by c on entity ("" when none was disclosed: no id from the request is stored).
// The purpose is the patient's own request for a patient and treatment for staff.
func (s *Service) record(ctx context.Context, c appointment.Caller, interaction, entity string, outcome auditevent.Outcome) error {
	action, purpose := auditevent.ActionExecute, auditevent.CodePATRQT
	if interaction == auditevent.CodeRead {
		action = auditevent.ActionRead
	}
	if c.OrganizationID != "" {
		purpose = auditevent.CodeTREAT
	}
	return s.Audit.Record(ctx, auditevent.AuditEvent{
		Type:            auditevent.Coding{System: auditevent.SystemAuditEventType, Code: auditevent.CodeRest},
		Subtype:         auditevent.Coding{System: auditevent.SystemRestfulInteraction, Code: interaction},
		Purpose:         auditevent.Coding{System: auditevent.SystemActReason, Code: purpose},
		Action:          action,
		Outcome:         outcome,
		AgentAccountID:  c.AccountID,
		OrganizationID:  c.OrganizationID,
		EntityReference: entity,
	})
}
