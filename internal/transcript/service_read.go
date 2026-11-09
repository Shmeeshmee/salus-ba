package transcript

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/encounter"
)

// RecheckEvery bounds how long Next waits for the stream before it checks the caller's access again, so a
// subscription that sees nothing new still ends this soon after a revocation; one that sees entries is checked
// before each page leaves.
const RecheckEvery = time.Second

// Export returns the encounter's stored transcript, in the order it was said, to its practitioner with the
// patient's active consent (access.ConsultationRecord), read in the clinic's scope and audited with the number
// of segments. ErrNotFound to a caller the appointment does not show (audited without the entity),
// ErrConsentRequired to its practitioner after a revocation and ErrForbidden to anyone else who sees it (both
// audited). The patient's own copy comes with the export feature.
func (s *Service) Export(ctx context.Context, c appointment.Caller, encounterID string) ([]Event, error) {
	var out []Event
	var refused error
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		id, err := s.allowed(ctx, c, encounterID)
		if refused = refusal(err); refused != nil {
			return s.record(ctx, c, auditevent.CodeRead, id, auditevent.OutcomeMinorFailure, 0)
		}
		if err != nil {
			return err
		}
		if out, err = s.Repo.ByEncounter(ctx, id); err != nil {
			return err
		}
		return s.record(ctx, c, auditevent.CodeRead, id, auditevent.OutcomeSuccess, int64(len(out)))
	})
	switch {
	case err != nil:
		return nil, fmt.Errorf("transcript: export: %w", err)
	case refused != nil:
		return nil, refused
	}
	if out == nil {
		out = []Event{}
	}
	return out, nil
}

// Subscribe lets the caller follow the encounter's stream (transcript, suggestion and draft events alike),
// under Export's rule, from after the entry id after ("" or "0": every entry still on the stream), and returns
// the cursor to read from with Next. The subscription is audited as an operation on the encounter, also when
// refused, with Export's errors; ErrInvalidCursor, unaudited, for a cursor that is no entry id.
func (s *Service) Subscribe(ctx context.Context, c appointment.Caller, encounterID, after string) (string, error) {
	if after == "" {
		after = "0"
	}
	if !validCursor(after) {
		return "", ErrInvalidCursor
	}
	var refused error
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		id, err := s.allowed(ctx, c, encounterID)
		if refused = refusal(err); refused != nil {
			return s.record(ctx, c, auditevent.CodeOperation, id, auditevent.OutcomeMinorFailure, 0)
		}
		if err != nil {
			return err
		}
		return s.record(ctx, c, auditevent.CodeOperation, id, auditevent.OutcomeSuccess, 0)
	})
	switch {
	case err != nil:
		return "", fmt.Errorf("transcript: subscribe: %w", err)
	case refused != nil:
		return "", refused
	}
	return after, nil
}

// Next returns the entries of the encounter's stream strictly after the cursor after, waiting up to
// RecheckEvery for one, once the caller's access has been checked again after the read: once it has ended (a
// revocation, a cancellation that hides the appointment) not one more entry is given, with Export's errors.
// Unaudited: the subscription was. The entries carry their envelopes as they are on the stream.
func (s *Service) Next(ctx context.Context, c appointment.Caller, encounterID, after string) ([]Entry, error) {
	entries, err := s.Stream.Read(ctx, encounterID, after, RecheckEvery)
	if err != nil {
		return nil, fmt.Errorf("transcript: next: %w", err)
	}
	_, err = s.allowed(ctx, c, encounterID)
	if refused := refusal(err); refused != nil {
		return nil, refused
	}
	if err != nil {
		return nil, fmt.Errorf("transcript: next: %w", err)
	}
	return entries, nil
}

// Allowed checks Export's rule for another domain acting on what the stream shows, in the transaction in ctx
// (joining it, scoped to the caller), and returns the encounter's id as far as the caller may know it ("" when
// the appointment does not show it), with Export's refusals: ErrNotFound, ErrConsentRequired or ErrForbidden.
// Unaudited: the caller audits its own action, refusals included.
func (s *Service) Allowed(ctx context.Context, c appointment.Caller, encounterID string) (string, error) {
	var id string
	var refused error
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		var err error
		id, err = s.allowed(ctx, c, encounterID)
		refused = refusal(err)
		if refused != nil {
			return nil
		}
		return err
	})
	switch {
	case err != nil:
		return "", fmt.Errorf("transcript: allowed: %w", err)
	case refused != nil:
		return id, refused
	}
	return id, nil
}

// errRefused wraps a refusal with the encounter it discloses ("" for none).
type errRefused struct {
	err error
}

func (e errRefused) Error() string { return e.err.Error() }
func (e errRefused) Unwrap() error { return e.err }

// refusal is the refusal in err, or nil when err is none (another error, or nil).
func refusal(err error) error {
	var r errRefused
	if errors.As(err, &r) {
		return r.err
	}
	return nil
}

// allowed checks that the caller may follow and read what was said in the encounter, in the transaction in ctx
// (scoped to the caller by it), and returns the encounter's id as far as the caller may know it: "" when the
// appointment does not show it. A refusal is an errRefused.
func (s *Service) allowed(ctx context.Context, c appointment.Caller, encounterID string) (string, error) {
	e, att, err := s.Encounters.Access(ctx, c, encounterID)
	switch {
	case errors.Is(err, encounter.ErrNotFound):
		return "", errRefused{ErrNotFound}
	case err != nil:
		return "", err
	}
	v, f, visit := rule(c, att)
	switch {
	case access.ConsultationRecord(v, f, visit):
		return e.ID, nil
	case att.Practitioner && !att.ConsentActive:
		return e.ID, errRefused{ErrConsentRequired}
	}
	return e.ID, errRefused{ErrForbidden}
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

// record audits interaction by c on Encounter/id ("" when none was disclosed): a read of count segments of
// its transcript, or the subscription to its stream (an operation). The purpose is the patient's own request
// for a patient and treatment for staff.
func (s *Service) record(ctx context.Context, c appointment.Caller, interaction, id string, outcome auditevent.Outcome, count int64) error {
	e := auditevent.AuditEvent{
		Type:           auditevent.Coding{System: auditevent.SystemAuditEventType, Code: auditevent.CodeRest},
		Subtype:        auditevent.Coding{System: auditevent.SystemRestfulInteraction, Code: interaction},
		Purpose:        auditevent.Coding{System: auditevent.SystemActReason, Code: auditevent.CodeTREAT},
		Action:         auditevent.ActionRead,
		Outcome:        outcome,
		AgentAccountID: c.AccountID,
		OrganizationID: c.OrganizationID,
	}
	if c.OrganizationID == "" {
		e.Purpose.Code = auditevent.CodePATRQT
	}
	if interaction == auditevent.CodeOperation {
		e.Action = auditevent.ActionExecute
	}
	if id != "" {
		e.EntityReference = "Encounter/" + id
	}
	if interaction == auditevent.CodeRead && outcome == auditevent.OutcomeSuccess {
		e.EntityName, e.EntityCount = entityName, count
	}
	return s.Audit.Record(ctx, e)
}

// validCursor reports whether after is a stream entry id: "0", or "<milliseconds>-<sequence>", both 64-bit
// unsigned decimals.
func validCursor(after string) bool {
	if after == "0" {
		return true
	}
	ms, seq, ok := strings.Cut(after, "-")
	return ok && decimal(ms) && decimal(seq)
}

func decimal(s string) bool {
	if s == "" || strings.Trim(s, "0123456789") != "" {
		return false
	}
	_, err := strconv.ParseUint(s, 10, 64)
	return err == nil
}
