package composition

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/encounter"
	"github.com/Shmeeshmee/salus/internal/transcript"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/envelope"
	"github.com/Shmeeshmee/salus/pkg/llm"
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

// Encounters gives a draft its encounter with the caller's attendance of its appointment, read in the caller's
// scope, unaudited (encounter.Service): composition applies its own rule and audits its decision.
type Encounters interface {
	Access(ctx context.Context, c appointment.Caller, id string) (encounter.Encounter, appointment.Attendance, error)
}

// Transcripts gives the encounter's stored transcript to its practitioner (transcript.Service), auditing the
// read itself.
type Transcripts interface {
	Export(ctx context.Context, c appointment.Caller, encounterID string) ([]transcript.Event, error)
}

// Ports are the Service's dependencies.
type Ports struct {
	Repo        Repository
	Encounters  Encounters
	Transcripts Transcripts
	Drafter     llm.Drafter
	Store       Store
	Publisher   Publisher
	Jobs        Jobs
	Audit       Auditor
	Tx          Transactor
	Clock       clock.Clock
	KEK         *envelope.KEK // wraps each PDF's data key
	Log         *slog.Logger  // a failed publish: ids and the error only
}

// Service holds the rules of the templates, the drafting and the signing (docs/plans/consult-intelligence.md,
// "E (SAL-8)"): staff of a clinic keep its templates; only the encounter's practitioner, with the patient's
// active consent (access.ConsultationRecord), drafts, signs and reads its documents; the encounter's patient
// reads its signed documents, never a draft (access.SignedDocument, docs/consultation/SPEC.md "Patient
// documents"). Every write, read and refusal of a draft or document is audited.
type Service struct{ Ports }

// NewService returns a Service on its ports.
func NewService(p Ports) *Service {
	return &Service{Ports: p}
}

// errRefused wraps a refusal, which is audited and committed, unlike a failure.
type errRefused struct{ err error }

func (e errRefused) Error() string { return e.err.Error() }
func (e errRefused) Unwrap() error { return e.err }

// refusal is the refusal in err, or nil when err is none.
func refusal(err error) error {
	var r errRefused
	if errors.As(err, &r) {
		return r.err
	}
	return nil
}

// allowed checks that the caller may draft, sign and read the documents of the encounter, or, with signed, read
// its signed documents, in the transaction in ctx (scoped to the caller by it), and returns the encounter. A
// refusal is an errRefused; the encounter's id is returned with it as far as the caller may know it ("" when the
// appointment does not show it).
func (s *Service) allowed(ctx context.Context, c appointment.Caller, encounterID string, signed bool) (encounter.Encounter, error) {
	e, _, err := s.allowedAttendance(ctx, c, encounterID, signed)
	return e, err
}

// allowedDrafting is allowed for a draft, which the model writes from the transcript: it needs the patient's
// transcription consent too (SAL-12), else ErrTranscriptionConsentRequired.
func (s *Service) allowedDrafting(ctx context.Context, c appointment.Caller, encounterID string) (encounter.Encounter, error) {
	e, att, err := s.allowedAttendance(ctx, c, encounterID, false)
	if err == nil && !att.TranscriptionConsent {
		return encounter.Encounter{ID: e.ID}, errRefused{ErrTranscriptionConsentRequired}
	}
	return e, err
}

// allowedAttendance is allowed, with the caller's attendance of the encounter's appointment.
func (s *Service) allowedAttendance(ctx context.Context, c appointment.Caller, encounterID string, signed bool) (encounter.Encounter, appointment.Attendance, error) {
	e, att, err := s.Encounters.Access(ctx, c, encounterID)
	switch {
	case errors.Is(err, encounter.ErrNotFound):
		return encounter.Encounter{}, att, errRefused{ErrNotFound}
	case err != nil:
		return encounter.Encounter{}, att, err
	}
	a := att.Appointment
	visit := access.Visit{PractitionerRole: a.PractitionerRoleID, Start: a.Start}
	f := access.Facts{
		Patient: a.PatientID, Clinic: a.OrganizationID, Visits: []access.Visit{visit},
		Consent: access.Consent{Active: att.ConsentActive, RevokedAt: att.RevokedAt},
	}
	v := access.Viewer{Role: access.RolePractitioner, Clinic: c.OrganizationID}
	switch {
	case att.Patient:
		v = access.Viewer{Role: access.RolePatient, Patient: c.PatientID}
	case c.Admin:
		v.Role = access.RoleAdmin
	case att.Practitioner:
		v.PractitionerRole = a.PractitionerRoleID
	}
	switch {
	case access.SignedDocument(v, f, visit, signed):
		return e, att, nil
	case att.Practitioner && !att.ConsentActive:
		return encounter.Encounter{ID: e.ID}, att, errRefused{ErrConsentRequired}
	}
	return encounter.Encounter{ID: e.ID}, att, errRefused{ErrForbidden}
}

// entry is one audit event of the drafting: who, the interaction on which resource ("" when none is
// disclosed) or, for a template (no FHIR resource), on the document_template table, and the outcome.
type entry struct {
	org         string
	interaction string
	reference   string
	c           appointment.Caller
	outcome     auditevent.Outcome
	template    bool
}

// record audits e, with treatment as the purpose for staff and the patient's own request for a patient.
func (s *Service) record(ctx context.Context, e entry) error {
	action := map[string]auditevent.Action{
		auditevent.CodeCreate: auditevent.ActionCreate, auditevent.CodeUpdate: auditevent.ActionUpdate,
		auditevent.CodeRead: auditevent.ActionRead, auditevent.CodeSearchType: auditevent.ActionRead,
	}[e.interaction]
	org := e.org
	if org == "" {
		org = e.c.OrganizationID
	}
	a := auditevent.AuditEvent{
		Type:            auditevent.Coding{System: auditevent.SystemAuditEventType, Code: auditevent.CodeRest},
		Subtype:         auditevent.Coding{System: auditevent.SystemRestfulInteraction, Code: e.interaction},
		Purpose:         auditevent.Coding{System: auditevent.SystemActReason, Code: auditevent.CodeTREAT},
		Action:          action,
		Outcome:         e.outcome,
		AgentAccountID:  e.c.AccountID,
		OrganizationID:  org,
		EntityReference: e.reference,
	}
	if e.c.OrganizationID == "" {
		a.Purpose.Code = auditevent.CodePATRQT
	}
	if e.template {
		a.Purpose.Code = auditevent.CodeHOPERAT
		a.EntityName, a.EntityCount = "document_template", 1
		if e.outcome != auditevent.OutcomeSuccess {
			a.EntityCount = 0
		}
	}
	return s.Audit.Record(ctx, a)
}

// encounterRef is the reference of the encounter id, or "" for none.
func encounterRef(id string) string {
	if id == "" {
		return ""
	}
	return "Encounter/" + id
}

// staff checks the caller is staff of a clinic and scopes the transaction in ctx to it.
func (s *Service) staff(ctx context.Context, c appointment.Caller) error {
	if c.OrganizationID == "" {
		return errRefused{ErrForbidden}
	}
	return s.Repo.SetClinic(ctx, c.OrganizationID)
}

// wrapErr returns the refusal in err, or err wrapped as the failure of op.
func wrapErr(op string, err, refused error) error {
	switch {
	case err != nil:
		return fmt.Errorf("composition: %s: %w", op, err)
	case refused != nil:
		return refused
	}
	return nil
}

// newID is a random (version 4) UUID in its canonical lowercase form.
func newID() string {
	var u [16]byte
	_, _ = rand.Read(u[:]) // crypto/rand never fails
	u[6] = u[6]&0x0f | 0x40
	u[8] = u[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", u[0:4], u[4:6], u[6:8], u[8:10], u[10:])
}
