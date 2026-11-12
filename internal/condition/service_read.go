package condition

import (
	"context"
	"errors"
	"fmt"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/encounter"
)

// Transactor runs fn in a transaction carried by the context, joining one already there (postgres.Transactor).
type Transactor interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// Auditor records an AuditEvent inside the caller's transaction (auditevent.Service).
type Auditor interface {
	Record(ctx context.Context, e auditevent.AuditEvent) error
}

// Encounters gives a Condition's encounter with the caller's attendance of its appointment, read in the caller's
// scope, unaudited (encounter.Service).
type Encounters interface {
	Access(ctx context.Context, c appointment.Caller, id string) (encounter.Encounter, appointment.Attendance, error)
}

// ReadPorts are what Reads is built on.
type ReadPorts struct {
	Repo       ReadRepository
	Encounters Encounters
	Audit      Auditor
	Tx         Transactor
}

// Reads serves the FHIR read of a Condition (api/fhir/CapabilityStatement.json). A Condition is part of the signed
// document it came from, so it has that document's rule (access.SignedDocument): the encounter's practitioner while
// the patient's consent is active, and the encounter's patient whatever the consent (their own record). Every read
// and refusal is audited.
type Reads struct {
	p ReadPorts
}

// NewReads returns Reads on p.
func NewReads(p ReadPorts) *Reads { return &Reads{p: p} }

// Get returns the Condition id to a caller allowed to read it: ErrNotFound, ErrForbidden or ErrConsentRequired
// otherwise, each audited and committed.
func (s *Reads) Get(ctx context.Context, c appointment.Caller, id string) (Condition, error) {
	var out Condition
	var refused error
	err := s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		cond, org, err := s.allowed(ctx, c, id)
		e := auditevent.AuditEvent{
			Type:           auditevent.Coding{System: auditevent.SystemAuditEventType, Code: auditevent.CodeRest},
			Subtype:        auditevent.Coding{System: auditevent.SystemRestfulInteraction, Code: auditevent.CodeRead},
			Purpose:        auditevent.Coding{System: auditevent.SystemActReason, Code: auditevent.CodeTREAT},
			Action:         auditevent.ActionRead,
			Outcome:        auditevent.OutcomeSuccess,
			AgentAccountID: c.AccountID,
			OrganizationID: org,
		}
		if c.OrganizationID == "" {
			e.Purpose.Code = auditevent.CodePATRQT
		}
		switch {
		case errors.Is(err, ErrNotFound), errors.Is(err, ErrForbidden), errors.Is(err, ErrConsentRequired):
			refused, e.Outcome = err, auditevent.OutcomeMinorFailure
		case err != nil:
			return err
		default:
			out, e.EntityReference = cond, "Condition/"+cond.ID
		}
		return s.p.Audit.Record(ctx, e)
	})
	if err != nil {
		return Condition{}, fmt.Errorf("condition: read: %w", err)
	}
	if refused != nil {
		return Condition{}, refused
	}
	return out, nil
}

// allowed reads the Condition in the caller's scope and applies the signed document's rule; it returns the
// clinic to audit under.
func (s *Reads) allowed(ctx context.Context, c appointment.Caller, id string) (Condition, string, error) {
	var err error
	switch {
	case c.OrganizationID != "":
		err = s.p.Repo.SetClinic(ctx, c.OrganizationID)
	case c.PatientID != "":
		err = s.p.Repo.SetPatient(ctx, c.PatientID)
	default:
		return Condition{}, "", ErrNotFound
	}
	if err != nil {
		return Condition{}, "", err
	}
	cond, err := s.p.Repo.ByID(ctx, id)
	if err != nil {
		return Condition{}, c.OrganizationID, err
	}
	_, att, err := s.p.Encounters.Access(ctx, c, cond.EncounterID)
	switch {
	case errors.Is(err, encounter.ErrNotFound):
		return Condition{}, c.OrganizationID, ErrNotFound
	case err != nil:
		return Condition{}, "", err
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
	case access.SignedDocument(v, f, visit, true):
		return cond, cond.OrganizationID, nil
	case att.Practitioner && !att.ConsentActive:
		return Condition{}, cond.OrganizationID, ErrConsentRequired
	}
	return Condition{}, cond.OrganizationID, ErrForbidden
}
