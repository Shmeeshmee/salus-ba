package appointment

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/internal/patient"
	"github.com/Shmeeshmee/salus/internal/practitionerrole"
	"github.com/Shmeeshmee/salus/internal/slot"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/payments"
)

// patientCancelBefore is the patient's cancellation window: until 24 hours before the start.
const patientCancelBefore = 24 * time.Hour

// Transactor runs fn in a transaction carried by the context, joining one already there
// (postgres.Transactor).
type Transactor interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// Auditor records an AuditEvent inside the caller's transaction (auditevent.Service).
type Auditor interface {
	Record(ctx context.Context, e auditevent.AuditEvent) error
}

// Consents gives the state of the patient's consents with a clinic, in the caller's transaction and scope
// (consent.Service).
type Consents interface {
	State(ctx context.Context, patientID, organizationID string) (access.Consent, error)
}

// Slots resolves the slot to book in the directory, and holds and releases it in the appointment's
// transaction (slot.Service).
type Slots interface {
	Bookable(ctx context.Context, id string) (slot.Bookable, error)
	Hold(ctx context.Context, org, id string) (slot.Slot, error)
	Release(ctx context.Context, org, id string) (bool, error)
}

// Patients reads a profile in the caller's transaction and scope, unaudited (patient.Service).
type Patients interface {
	Lookup(ctx context.Context, id string) (patient.Patient, error)
}

// Clinics reads a clinic in its own scope, to check it is listed (organization.Service).
type Clinics interface {
	Read(ctx context.Context, c organization.Caller, id string) (organization.Organization, error)
}

// Roles reads a practitioner role of the caller's clinic, to know whose it is, and looks up a role of a
// clinic, or a practitioner's, in the transaction in ctx, unaudited (practitionerrole.Service).
type Roles interface {
	Read(ctx context.Context, c practitionerrole.Caller, id string) (practitionerrole.PractitionerRole, error)
	Lookup(ctx context.Context, org, id string) (practitionerrole.PractitionerRole, error)
	OfPractitioner(ctx context.Context, org, practitionerID string) (practitionerrole.PractitionerRole, error)
}

// Jobs enqueues the appointment's e-mails, the request of its role's default questionnaire on a booking, and
// the end of its consultation's access and the discard of its questionnaire responses on a cancellation, in
// the transaction in ctx.
type Jobs interface {
	EnqueueBooked(ctx context.Context, id, org string) error
	EnqueueCancelled(ctx context.Context, id, org string) error
	EnqueueAccessEnd(ctx context.Context, id, org string) error
	EnqueueReminder(ctx context.Context, id, org string, at time.Time) error
	EnqueueQuestionnaireRequest(ctx context.Context, id, org string) error
	EnqueueQuestionnaireDiscard(ctx context.Context, id, org string) error
	// EnqueueSlotFreed hands a freed slot still ahead to the waitlist (waitlist_slot_freed, SAL-35).
	EnqueueSlotFreed(ctx context.Context, slotID, org string) error
}

// StatsQueue enqueues the refresh of a clinic's statistics for the local day holding day, in the transaction in
// ctx (clinicstats' River queue, docs/plans/analytics.md §4): every appointment write enqueues its start's day.
type StatsQueue interface {
	EnqueueStatsRefresh(ctx context.Context, orgID string, day time.Time) error
}

// Ports are the Service's dependencies. The notifier builds it for Notice only (Repo, Audit, Tx, Clock,
// Patients).
type Ports struct {
	Repo     Repository
	Audit    Auditor
	Tx       Transactor
	Clock    clock.Clock
	Consents Consents
	Slots    Slots
	Patients Patients
	Clinics  Clinics
	Roles    Roles
	Jobs     Jobs
	Stats    StatsQueue // every write; not needed by a Service that only reads
	// Prices gives a booking's price and Payments takes it (service_purchase.go): no price exists before the payments
	// epic (SAL-23), so both are nil in every binary and nothing is charged.
	Prices   Prices
	Payments payments.Provider
	// Tokens stores the one-time links of reminders and LinkURL is their base (the api's public origin): needed only
	// by a Service that issues links (IssueLink, the notifier's reminders).
	Tokens TokenRepository
	// NoShows corrects a no-show or an attendance, and Rules gives the clinic's repeat no-show flag (SAL-36,
	// service_noshow.go): needed only by the api's Service.
	NoShows NoShowRepository
	Rules   ClinicRules
	LinkURL string
}

// Service holds the appointment rules (SPEC.md "Booking rules", "Who may do what with an appointment"). A
// patient books, cancels until 24 hours before and reads their own; staff read and cancel through access;
// the practitioner marks their own fulfilled or no-show. Every read and write is audited, refusals included.
type Service struct{ Ports }

// NewService returns a Service on its ports.
func NewService(p Ports) *Service {
	return &Service{Ports: p}
}

// Book books the free slot slotID for the calling patient and returns the appointment. The slot is resolved
// in the public directory first (free, of an active role of a listed clinic); then, in one transaction in
// the patient's scope with the slot's clinic set, the clinic is checked listed and the patient's consent with
// it active, the slot is held, the appointment stored, the confirmation and the request of the role's default
// questionnaire (questionnaire_default_request) enqueued (and the reminder, 24 hours before the start, when that
// is ahead), and the booking audited. ErrForbidden to staff, ErrProfileRequired
// without a profile, ErrSlotUnavailable for a slot that is not free and ahead or a time the practitioner is
// booked at, ErrConsentRequired without an active consent; each refusal is audited.
func (s *Service) Book(ctx context.Context, c Caller, slotID string) (Appointment, error) {
	switch {
	case !c.patient():
		return Appointment{}, s.refuse(ctx, c, auditevent.CodeCreate, ErrForbidden)
	case c.PatientID == "":
		return Appointment{}, s.refuse(ctx, c, auditevent.CodeCreate, ErrProfileRequired)
	}
	now := s.Clock.Now()
	b, err := s.Slots.Bookable(ctx, slotID)
	switch {
	case errors.Is(err, slot.ErrUnavailable), err == nil && !b.Start.After(now):
		return Appointment{}, s.refuse(ctx, c, auditevent.CodeCreate, ErrSlotUnavailable)
	case err != nil:
		return Appointment{}, fmt.Errorf("appointment: book: %w", err)
	}
	// Paid before the booking's transaction, refunded when the booking then fails (ADR 0015 "pkg/payments"). The key
	// is the patient's and the slot's until the payments epic gives a booking a request id of its own.
	receipt, err := s.purchase(ctx, c, b, "booking:"+c.PatientID+":"+b.ID)
	switch {
	case errors.Is(err, payments.ErrDeclined):
		return Appointment{}, s.refuse(ctx, c, auditevent.CodeCreate, ErrPaymentDeclined)
	case err != nil:
		return Appointment{}, fmt.Errorf("appointment: book: %w", err)
	}
	out, err := s.bookSlot(ctx, c, b, now)
	if err != nil {
		// Not cut short by the request's end: a charge must not stay without its booking.
		if rerr := s.refund(context.WithoutCancel(ctx), receipt); rerr != nil {
			return Appointment{}, errors.Join(err, fmt.Errorf("appointment: book: refund: %w", rerr))
		}
		return Appointment{}, err
	}
	return out, nil
}

// bookSlot books the slot b for c in one transaction, audited; each refusal is audited in a transaction of its own.
func (s *Service) bookSlot(ctx context.Context, c Caller, b slot.Bookable, now time.Time) (Appointment, error) {
	var out Appointment
	var refused error
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		var err error
		if out, refused, err = s.book(ctx, c, b, now); err != nil || refused != nil {
			return err
		}
		return s.record(ctx, c, auditevent.CodeCreate, "Appointment", out.ID, auditevent.OutcomeSuccess)
	})
	switch {
	case errors.Is(err, ErrSlotUnavailable):
		// The insert aborted the transaction: the refusal is audited in a transaction of its own.
		return Appointment{}, s.refuse(ctx, c, auditevent.CodeCreate, ErrSlotUnavailable)
	case err != nil:
		return Appointment{}, fmt.Errorf("appointment: book: %w", err)
	case refused != nil:
		return Appointment{}, s.refuse(ctx, c, auditevent.CodeCreate, refused)
	}
	return out, nil
}

// book checks and books the slot b in the transaction in ctx, or gives the refusal.
func (s *Service) book(ctx context.Context, c Caller, b slot.Bookable, now time.Time) (out Appointment, refused, err error) {
	if err := s.Repo.SetPatient(ctx, c.PatientID); err != nil {
		return Appointment{}, nil, err
	}
	if err := s.Repo.SetClinic(ctx, b.OrganizationID); err != nil {
		return Appointment{}, nil, err
	}
	clinic, err := s.Clinics.Read(ctx, organization.Caller{OrganizationID: b.OrganizationID}, b.OrganizationID)
	if err != nil {
		return Appointment{}, nil, err
	}
	if !clinic.Listed { // not on the directory's row-level security alone
		return Appointment{}, ErrSlotUnavailable, nil
	}
	consent, err := s.Consents.State(ctx, c.PatientID, b.OrganizationID)
	if err != nil {
		return Appointment{}, nil, err
	}
	if !consent.Active {
		return Appointment{}, ErrConsentRequired, nil
	}
	if _, err := s.Slots.Hold(ctx, b.OrganizationID, b.ID); errors.Is(err, slot.ErrUnavailable) {
		return Appointment{}, ErrSlotUnavailable, nil
	} else if err != nil {
		return Appointment{}, nil, err
	}
	out, err = s.Repo.Create(ctx, Appointment{
		PatientID: c.PatientID, PractitionerRoleID: b.PractitionerRoleID, OrganizationID: b.OrganizationID, SlotID: b.ID,
		Start: b.Start, End: b.End, Created: now, BookedVia: BookedDirect,
	})
	if err != nil {
		return Appointment{}, nil, err
	}
	if err := s.Jobs.EnqueueBooked(ctx, out.ID, out.OrganizationID); err != nil {
		return Appointment{}, nil, err
	}
	if err := s.Jobs.EnqueueQuestionnaireRequest(ctx, out.ID, out.OrganizationID); err != nil {
		return Appointment{}, nil, err
	}
	if err := s.Stats.EnqueueStatsRefresh(ctx, out.OrganizationID, out.Start); err != nil {
		return Appointment{}, nil, err
	}
	if err := s.scheduleReminders(ctx, out, now); err != nil {
		return Appointment{}, nil, err
	}
	return out, nil, nil
}

// refuse audits a refused interaction by c in a transaction of its own and returns refusal.
func (s *Service) refuse(ctx context.Context, c Caller, interaction string, refusal error) error {
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		return s.record(ctx, c, interaction, "", "", auditevent.OutcomeMinorFailure)
	})
	if err != nil {
		return fmt.Errorf("appointment: %s: %w", interaction, err)
	}
	return refusal
}

// record audits interaction by c on resource/id ("" when none was disclosed: no id from the request is
// stored). The purpose is the patient's own request for a patient and treatment for staff.
func (s *Service) record(ctx context.Context, c Caller, interaction, resource, id string, outcome auditevent.Outcome) error {
	purpose := auditevent.CodePATRQT
	if !c.patient() {
		purpose = auditevent.CodeTREAT
	}
	e := entry(interaction, resource, id, outcome, purpose)
	e.AgentAccountID, e.OrganizationID = c.AccountID, c.OrganizationID
	return s.Audit.Record(ctx, e)
}

// recordJob audits interaction on resource/id by the job agent (its kind) in the clinic org, for operations.
func (s *Service) recordJob(ctx context.Context, agent, org, interaction, resource, id string, outcome auditevent.Outcome) error {
	e := entry(interaction, resource, id, outcome, auditevent.CodeHOPERAT)
	e.AgentName, e.OrganizationID = agent, org
	return s.Audit.Record(ctx, e)
}

// entry is the audit event of a FHIR restful interaction on resource/id.
func entry(interaction, resource, id string, outcome auditevent.Outcome, purpose string) auditevent.AuditEvent {
	action := auditevent.ActionRead
	switch interaction {
	case auditevent.CodeCreate:
		action = auditevent.ActionCreate
	case auditevent.CodeUpdate:
		action = auditevent.ActionUpdate
	case auditevent.CodeSearchType:
		action = auditevent.ActionExecute
	}
	e := auditevent.AuditEvent{
		Type:    auditevent.Coding{System: auditevent.SystemAuditEventType, Code: auditevent.CodeRest},
		Subtype: auditevent.Coding{System: auditevent.SystemRestfulInteraction, Code: interaction},
		Purpose: auditevent.Coding{System: auditevent.SystemActReason, Code: purpose},
		Action:  action,
		Outcome: outcome,
	}
	if id != "" {
		e.EntityReference = resource + "/" + id
	}
	return e
}
