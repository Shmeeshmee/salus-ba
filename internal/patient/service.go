package patient

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

const (
	maxName  = 100 // characters, after trimming
	minPhone = 8   // digits after the +
	maxPhone = 15
)

// earliestBirthDate is the first birth date accepted (SPEC.md "Profile").
var earliestBirthDate = time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC)

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

// Appointments gives the patient's appointments in a clinic, in the caller's transaction and scope (the
// appointment domain, docs/booking package 6; none before it).
type Appointments interface {
	Visits(ctx context.Context, patientID, organizationID string) ([]access.Visit, error)
}

// Service holds the profile rules: a patient reads and saves only their own profile; staff read one only
// when access gives them all of it (SPEC.md "Access rule"); every read and write is audited, also when it is
// refused.
type Service struct {
	repo         Repository
	audit        Auditor
	tx           Transactor
	clock        clock.Clock
	consents     Consents
	appointments Appointments
}

// NewService returns a Service on its ports.
func NewService(repo Repository, audit Auditor, tx Transactor, clk clock.Clock, consents Consents, appointments Appointments) *Service {
	return &Service{repo: repo, audit: audit, tx: tx, clock: clk, consents: consents, appointments: appointments}
}

// Me returns the calling patient's profile, ErrNotFound before their first save, or ErrForbidden to staff.
func (s *Service) Me(ctx context.Context, c Caller) (Patient, error) {
	if !c.patient() {
		if err := s.record(ctx, c, auditevent.CodeRead, "", auditevent.OutcomeMinorFailure); err != nil {
			return Patient{}, fmt.Errorf("patient: me: %w", err)
		}
		return Patient{}, ErrForbidden
	}
	var p Patient
	var missing bool
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.repo.SetAccount(ctx, c.AccountID); err != nil {
			return err
		}
		var err error
		p, err = s.repo.ByAccount(ctx, c.AccountID)
		if missing = errors.Is(err, ErrNotFound); missing {
			return s.record(ctx, c, auditevent.CodeRead, "", auditevent.OutcomeMinorFailure)
		}
		if err != nil {
			return err
		}
		return s.record(ctx, c, auditevent.CodeRead, p.ID, auditevent.OutcomeSuccess)
	})
	switch {
	case err != nil:
		return Patient{}, fmt.Errorf("patient: me: %w", err)
	case missing:
		return Patient{}, ErrNotFound
	}
	return p, nil
}

// SaveMe saves the calling patient's profile from in, trimmed: it creates it when in.VersionID is 0 and
// updates the version in.VersionID otherwise. ErrConflict when that is not the current state (a profile
// exists already, or the version is stale), ErrForbidden to staff, ErrInvalid… for a field that breaks
// SPEC.md "Profile" (before any access).
func (s *Service) SaveMe(ctx context.Context, c Caller, in Patient) (Patient, error) {
	interaction := auditevent.CodeCreate
	if in.VersionID != 0 {
		interaction = auditevent.CodeUpdate
	}
	if !c.patient() {
		if err := s.record(ctx, c, interaction, "", auditevent.OutcomeMinorFailure); err != nil {
			return Patient{}, fmt.Errorf("patient: save: %w", err)
		}
		return Patient{}, ErrForbidden
	}
	in, err := s.valid(in)
	if err != nil {
		return Patient{}, err
	}
	in.AccountID = c.AccountID
	var p Patient
	var conflict bool
	err = s.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.repo.SetAccount(ctx, c.AccountID); err != nil {
			return err
		}
		var err error
		if interaction == auditevent.CodeCreate {
			p, err = s.repo.Create(ctx, in, s.clock.Now())
		} else {
			p, err = s.repo.Update(ctx, in, s.clock.Now())
		}
		if conflict = errors.Is(err, ErrConflict); conflict {
			return s.record(ctx, c, interaction, "", auditevent.OutcomeMinorFailure)
		}
		if err != nil {
			return err
		}
		return s.record(ctx, c, interaction, p.ID, auditevent.OutcomeSuccess)
	})
	switch {
	case err != nil:
		return Patient{}, fmt.Errorf("patient: save: %w", err)
	case conflict:
		return Patient{}, ErrConflict
	}
	return p, nil
}

// Read returns the profile id to its own patient and to staff whose access to it is Full (a practitioner
// with an active consent and an appointment with the patient), and ErrNotFound to anyone else: the name-only
// level is given with an appointment, not here (SPEC.md "FHIR reads and searches").
func (s *Service) Read(ctx context.Context, c Caller, id string) (Patient, error) {
	var p Patient
	var missing bool
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		var err error
		if c.patient() {
			p, missing, err = s.readOwn(ctx, c, id)
		} else {
			p, missing, err = s.readAsStaff(ctx, c, id)
		}
		if err != nil {
			return err
		}
		if missing {
			return s.record(ctx, c, auditevent.CodeRead, "", auditevent.OutcomeMinorFailure)
		}
		return s.record(ctx, c, auditevent.CodeRead, p.ID, auditevent.OutcomeSuccess)
	})
	switch {
	case err != nil:
		return Patient{}, fmt.Errorf("patient: read: %w", err)
	case missing:
		return Patient{}, ErrNotFound
	}
	return p, nil
}

// readOwn reads the profile id in the calling patient's scope and reports it missing unless it is theirs.
func (s *Service) readOwn(ctx context.Context, c Caller, id string) (Patient, bool, error) {
	if err := s.repo.SetAccount(ctx, c.AccountID); err != nil {
		return Patient{}, false, err
	}
	p, err := s.repo.ByID(ctx, id)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return Patient{}, false, err
	}
	// The owner check does not rest on row-level security alone.
	return p, err != nil || p.AccountID != c.AccountID, nil
}

// readAsStaff gathers the facts of the patient id in the caller's clinic, in the clinic's scope, and reads
// the profile only when access gives the caller all of it; otherwise it reports it missing.
func (s *Service) readAsStaff(ctx context.Context, c Caller, id string) (Patient, bool, error) {
	if err := s.repo.SetClinic(ctx, c.OrganizationID); err != nil {
		return Patient{}, false, err
	}
	state, err := s.consents.State(ctx, id, c.OrganizationID)
	if err != nil {
		return Patient{}, false, err
	}
	visits, err := s.appointments.Visits(ctx, id, c.OrganizationID)
	if err != nil {
		return Patient{}, false, err
	}
	viewer := access.Viewer{Role: access.RolePractitioner, Clinic: c.OrganizationID, PractitionerRole: c.PractitionerRoleID}
	if c.Admin {
		viewer = access.Viewer{Role: access.RoleAdmin, Clinic: c.OrganizationID}
	}
	facts := access.Facts{Patient: id, Clinic: c.OrganizationID, Consent: state, Visits: visits}
	if access.Profile(viewer, facts) != access.Full {
		return Patient{}, true, nil
	}
	p, err := s.repo.ByID(ctx, id)
	if errors.Is(err, ErrNotFound) {
		return Patient{}, true, nil
	}
	return p, false, err
}

// valid returns in with its names trimmed, or the error of the first field that breaks SPEC.md "Profile".
// A birth date must be before today (UTC) and not before 1900-01-01.
func (s *Service) valid(in Patient) (Patient, error) {
	in.GivenName, in.FamilyName = strings.TrimSpace(in.GivenName), strings.TrimSpace(in.FamilyName)
	for _, name := range []string{in.GivenName, in.FamilyName} {
		if name == "" || utf8.RuneCountInString(name) > maxName {
			return Patient{}, ErrInvalidName
		}
	}
	y, m, d := s.clock.Now().UTC().Date()
	today := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	if in.BirthDate.Before(earliestBirthDate) || !in.BirthDate.Before(today) {
		return Patient{}, ErrInvalidBirthDate
	}
	if in.Phone != "" && !e164(in.Phone) {
		return Patient{}, ErrInvalidPhone
	}
	return in, nil
}

// e164 reports whether phone is + and 8 to 15 ASCII digits.
func e164(phone string) bool {
	digits, ok := strings.CutPrefix(phone, "+")
	if !ok || len(digits) < minPhone || len(digits) > maxPhone {
		return false
	}
	for _, r := range digits {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// record audits interaction by c on Patient/id ("" when none was disclosed: no id from the request is
// stored). The purpose is the patient's own request for a patient and treatment for staff.
func (s *Service) record(ctx context.Context, c Caller, interaction, id string, outcome auditevent.Outcome) error {
	action := auditevent.ActionRead
	switch interaction {
	case auditevent.CodeCreate:
		action = auditevent.ActionCreate
	case auditevent.CodeUpdate:
		action = auditevent.ActionUpdate
	}
	purpose := auditevent.CodePATRQT
	if !c.patient() {
		purpose = auditevent.CodeTREAT
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
		e.EntityReference = "Patient/" + id
	}
	return s.audit.Record(ctx, e)
}
