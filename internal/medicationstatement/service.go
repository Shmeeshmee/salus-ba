package medicationstatement

import (
	"context"
	"errors"
	"fmt"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/encounter"
	"github.com/Shmeeshmee/salus/internal/practitionerrole"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

// Transactor runs fn in a transaction carried by the context, joining one already there (postgres.Transactor).
type Transactor interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// Auditor records an AuditEvent inside the caller's transaction (auditevent.Service).
type Auditor interface {
	Record(ctx context.Context, e auditevent.AuditEvent) error
}

// Consents gives the state of the patient's consents with a clinic, the history consent's shared visits included, in
// the caller's scope (consent.Service).
type Consents interface {
	State(ctx context.Context, patientID, organizationID string) (access.Consent, error)
}

// Visits gives the patient's appointments in a clinic, in the caller's scope (appointment.Facts).
type Visits interface {
	Visits(ctx context.Context, patientID, organizationID string) ([]access.Visit, error)
}

// Roles gives a practitioner's role in their clinic (practitionerrole.Service).
type Roles interface {
	OfPractitioner(ctx context.Context, org, practitionerID string) (practitionerrole.PractitionerRole, error)
}

// Encounters gives an encounter with the caller's attendance of its appointment, in the caller's scope, unaudited
// (encounter.Service).
type Encounters interface {
	Access(ctx context.Context, c appointment.Caller, id string) (encounter.Encounter, appointment.Attendance, error)
}

// Terms gives the ATC code of each drug term among ids ("" for a drug without one); a term that is not a drug is
// absent (terminology.Service).
type Terms interface {
	DrugCodes(ctx context.Context, ids []string) (map[string]string, error)
}

// Ports are the Service's dependencies.
type Ports struct {
	Repo       Repository
	Consents   Consents
	Visits     Visits
	Roles      Roles
	Encounters Encounters
	Terms      Terms
	Audit      Auditor
	Tx         Transactor
	Clock      clock.Clock
}

// Service holds the medications' rules (docs/plans/patient-history.md, ADR 0014): who sees an entry
// (access.ProfileEntry) and its source (access.Attributed), who writes, changes, confirms and removes one, and an
// AuditEvent for every entry read and every write, refusals included.
type Service struct{ Ports }

// NewService returns a Service on its ports.
func NewService(p Ports) *Service { return &Service{Ports: p} }

// Caller is who asks (internal/api resolves it from the session).
type Caller = appointment.Caller

// Seen returns the patient's medications the caller may see, newest first, entered-in-error ones left out, each with
// whether the caller may see its source, and audits each one returned (history.Medications). A caller who sees none
// gets none, audited as one search.
func (s *Service) Seen(ctx context.Context, c Caller, patientID string) ([]Seen, error) {
	var out []Seen
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		v, err := s.identify(ctx, c)
		if err != nil {
			return err
		}
		f, err := s.facts(ctx, c, v, patientID)
		if err != nil {
			return err
		}
		all, err := s.Repo.ByPatient(ctx, patientID)
		if err != nil {
			return err
		}
		for _, a := range all {
			if seen, ok := see(v, f, a); ok {
				out = append(out, seen)
			}
		}
		if err := s.codes(ctx, out); err != nil {
			return err
		}
		if len(out) == 0 {
			return s.record(ctx, c, auditevent.CodeSearchType, "", auditevent.OutcomeSuccess)
		}
		for _, a := range out {
			if err := s.record(ctx, c, auditevent.CodeSearchType, a.ID, auditevent.OutcomeSuccess); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("medicationstatement: seen: %w", err)
	}
	return out, nil
}

// Get returns the medication id to a caller who may see it; ErrNotFound otherwise, and for a removed one. Audited.
func (s *Service) Get(ctx context.Context, c Caller, id string) (Seen, error) {
	var out Seen
	_, err := s.do(ctx, c, auditevent.CodeRead, func(ctx context.Context) (MedicationStatement, error) {
		v, f, a, err := s.visible(ctx, c, id)
		if err != nil {
			return MedicationStatement{}, err
		}
		seen, _ := see(v, f, a)
		all := []Seen{seen}
		if err := s.codes(ctx, all); err != nil {
			return MedicationStatement{}, err
		}
		out = all[0]
		return a, nil
	})
	return out, err
}

// identify scopes the transaction to the caller and returns them as the access rules see them: a patient, the
// clinic's admin, a practitioner with their role (none when they have no role there), or nobody.
func (s *Service) identify(ctx context.Context, c Caller) (access.Viewer, error) {
	if c.OrganizationID == "" {
		if c.PatientID == "" {
			return access.Viewer{}, nil
		}
		return access.Viewer{Role: access.RolePatient, Patient: c.PatientID}, s.Repo.SetPatient(ctx, c.PatientID)
	}
	if err := s.Repo.SetClinic(ctx, c.OrganizationID); err != nil {
		return access.Viewer{}, err
	}
	v := access.Viewer{Role: access.RolePractitioner, Clinic: c.OrganizationID}
	switch {
	case c.Admin:
		v.Role = access.RoleAdmin
		return v, nil
	case c.PractitionerID == "":
		return access.Viewer{}, nil
	}
	role, err := s.Roles.OfPractitioner(ctx, c.OrganizationID, c.PractitionerID)
	switch {
	case errors.Is(err, practitionerrole.ErrNotFound):
		return v, nil
	case err != nil:
		return access.Viewer{}, err
	}
	v.PractitionerRole = role.ID
	return v, nil
}

// facts gathers what access needs about the patient in the caller's clinic: for staff the consents and the
// appointments there; a patient needs none.
func (s *Service) facts(ctx context.Context, c Caller, v access.Viewer, patientID string) (access.Facts, error) {
	f := access.Facts{Patient: patientID, Clinic: c.OrganizationID}
	if v.Role != access.RolePractitioner && v.Role != access.RoleAdmin {
		return f, nil
	}
	var err error
	if f.Consent, err = s.Consents.State(ctx, patientID, c.OrganizationID); err != nil {
		return access.Facts{}, err
	}
	if f.Visits, err = s.Visits.Visits(ctx, patientID, c.OrganizationID); err != nil {
		return access.Facts{}, err
	}
	return f, nil
}

// see reports whether v sees a under the facts f (never a removed entry) and how.
func see(v access.Viewer, f access.Facts, a MedicationStatement) (Seen, bool) {
	if a.Status == StatusEnteredInError {
		return Seen{}, false
	}
	r := access.Record{Encounter: a.Source.EncounterID, Clinic: a.Source.OrganizationID, PractitionerRole: a.Source.RecorderRoleID}
	if !access.ProfileEntry(v, f, a.InProfile, r) {
		return Seen{}, false
	}
	return Seen{MedicationStatement: a, Attributed: access.Attributed(v, f, a.Source.OrganizationID)}, true
}

// visible reads the medication id in the caller's scope with the caller and the patient's facts: ErrNotFound when the
// caller does not see it.
func (s *Service) visible(ctx context.Context, c Caller, id string) (access.Viewer, access.Facts, MedicationStatement, error) {
	v, err := s.identify(ctx, c)
	if err != nil {
		return access.Viewer{}, access.Facts{}, MedicationStatement{}, err
	}
	if v.Role == 0 {
		return access.Viewer{}, access.Facts{}, MedicationStatement{}, ErrNotFound
	}
	a, err := s.Repo.ByID(ctx, id)
	if err != nil {
		return access.Viewer{}, access.Facts{}, MedicationStatement{}, err
	}
	f, err := s.facts(ctx, c, v, a.PatientID)
	if err != nil {
		return access.Viewer{}, access.Facts{}, MedicationStatement{}, err
	}
	if _, ok := see(v, f, a); !ok {
		return access.Viewer{}, access.Facts{}, MedicationStatement{}, ErrNotFound
	}
	return v, f, a, nil
}

// codes fills in the ATC code of each entry picked from the terminology.
func (s *Service) codes(ctx context.Context, seen []Seen) error {
	var terms []string
	for _, a := range seen {
		if a.Code.TermID != "" {
			terms = append(terms, a.Code.TermID)
		}
	}
	if len(terms) == 0 {
		return nil
	}
	atc, err := s.Terms.DrugCodes(ctx, terms)
	if err != nil {
		return err
	}
	for i := range seen {
		seen[i].Code.ATC = atc[seen[i].Code.TermID]
	}
	return nil
}

// refusals are the errors a caller is told; anything else is an internal error.
var refusals = []error{ErrNotFound, ErrForbidden, ErrInvalid, ErrVersionConflict}

func refused(err error) bool {
	for _, r := range refusals {
		if errors.Is(err, r) {
			return true
		}
	}
	return false
}

// do runs fn in a transaction and audits interaction on the entry it returns. A refusal rolls back what fn did and
// is audited in a transaction of its own, naming the entry fn returned with it (one the caller sees) or none.
func (s *Service) do(ctx context.Context, c Caller, interaction string, fn func(ctx context.Context) (MedicationStatement, error)) (MedicationStatement, error) {
	var out, named MedicationStatement
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		a, err := fn(ctx)
		if err != nil {
			named = a
			return err
		}
		out = a
		return s.record(ctx, c, interaction, a.ID, auditevent.OutcomeSuccess)
	})
	switch {
	case refused(err):
		if aerr := s.Tx.WithTx(ctx, func(ctx context.Context) error {
			return s.record(ctx, c, interaction, named.ID, auditevent.OutcomeMinorFailure)
		}); aerr != nil {
			return MedicationStatement{}, fmt.Errorf("medicationstatement: %s: %w", interaction, aerr)
		}
		return MedicationStatement{}, err
	case err != nil:
		return MedicationStatement{}, fmt.Errorf("medicationstatement: %s: %w", interaction, err)
	}
	return out, nil
}

// record audits interaction by c on MedicationStatement/id ("" when none is disclosed). The purpose is the patient's
// own request for a patient and treatment for staff; a search is an execute (FHIR AuditEvent.action).
func (s *Service) record(ctx context.Context, c Caller, interaction, id string, outcome auditevent.Outcome) error {
	action := auditevent.ActionRead
	switch interaction {
	case auditevent.CodeCreate:
		action = auditevent.ActionCreate
	case auditevent.CodeUpdate:
		action = auditevent.ActionUpdate
	case auditevent.CodeDelete:
		action = auditevent.ActionDelete
	case auditevent.CodeSearchType:
		action = auditevent.ActionExecute
	}
	purpose := auditevent.CodeTREAT
	if c.OrganizationID == "" {
		purpose = auditevent.CodePATRQT
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
		e.EntityReference = "MedicationStatement/" + id
	}
	return s.Audit.Record(ctx, e)
}
