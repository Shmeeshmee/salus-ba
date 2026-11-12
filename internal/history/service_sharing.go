package history

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/composition"
	"github.com/Shmeeshmee/salus/internal/consent"
	"github.com/Shmeeshmee/salus/internal/encounter"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/internal/practitionerrole"
)

// SharingEncounters lists the calling patient's own encounters, each read audited (encounter.Service.Search).
type SharingEncounters interface {
	Search(ctx context.Context, c appointment.Caller, f encounter.Filter) ([]encounter.Encounter, error)
}

// SignedDocuments gives an encounter's signed documents, in the order signed, to its patient, audited
// (composition.Service.Signed).
type SignedDocuments interface {
	Signed(ctx context.Context, c appointment.Caller, encounterID string) ([]composition.Document, error)
}

// SharingRoles gives a clinic's practitioner role, unaudited: a role holds no personal data
// (practitionerrole.Service.Lookup).
type SharingRoles interface {
	Lookup(ctx context.Context, org, id string) (practitionerrole.PractitionerRole, error)
}

// SharingClinics gives a clinic's name, unaudited (organization.Service.Name).
type SharingClinics interface {
	Name(ctx context.Context, organizationID string) (string, error)
}

// SharingPorts are what Sharing reads.
type SharingPorts struct {
	Encounters SharingEncounters
	Documents  SignedDocuments
	Roles      SharingRoles
	Clinics    SharingClinics
}

// Sharing gives the visits a patient may share with a clinic (SAL-42, consent.Visits): the preview at booking and in
// settings, built from the owning domains' services, each applying its own rule for the patient and auditing.
type Sharing struct {
	p SharingPorts
}

// NewSharing returns Sharing on its ports.
func NewSharing(p SharingPorts) *Sharing { return &Sharing{p: p} }

// Shareable returns the calling patient's finished visits at every clinic but organizationID, newest first, each as
// the clinic would see it marked once shared (ADR 0014; docs/plans/patient-history.md "The visit's title in the
// preview"): its date, its clinic's name, its practitioner role's specialty and the title of its first signed
// document (none without one). consent.ErrClinicNotFound for an unknown clinic. It joins the transaction in ctx.
func (s *Sharing) Shareable(ctx context.Context, c consent.Caller, organizationID string) ([]consent.Candidate, error) {
	if _, err := s.p.Clinics.Name(ctx, organizationID); errors.Is(err, organization.ErrNotFound) {
		return nil, consent.ErrClinicNotFound
	} else if err != nil {
		return nil, fmt.Errorf("history: shareable: %w", err)
	}
	patient := appointment.Caller{AccountID: c.AccountID, PatientID: c.PatientID}
	all, err := s.p.Encounters.Search(ctx, patient, encounter.Filter{PatientID: c.PatientID})
	if err != nil {
		return nil, fmt.Errorf("history: shareable: %w", err)
	}
	names, specialties := map[string]string{}, map[string]string{}
	out := []consent.Candidate{}
	for _, e := range all {
		if e.Status != encounter.StatusFinished || e.OrganizationID == organizationID || e.PatientID != c.PatientID {
			continue
		}
		v, err := s.candidate(ctx, patient, e, names, specialties)
		if err != nil {
			return nil, fmt.Errorf("history: shareable: %w", err)
		}
		out = append(out, v)
	}
	slices.SortFunc(out, func(a, b consent.Candidate) int {
		return cmp.Or(b.Date.Compare(a.Date), cmp.Compare(a.EncounterID, b.EncounterID))
	})
	return out, nil
}

// candidate is the visit e as the preview shows it; names and specialties keep the clinics' names and the roles'
// specialties already read.
func (s *Sharing) candidate(ctx context.Context, patient appointment.Caller, e encounter.Encounter, names, specialties map[string]string) (consent.Candidate, error) {
	name, ok := names[e.OrganizationID]
	if !ok {
		var err error
		if name, err = s.p.Clinics.Name(ctx, e.OrganizationID); err != nil {
			return consent.Candidate{}, err
		}
		names[e.OrganizationID] = name
	}
	specialty, ok := specialties[e.PractitionerRoleID]
	if !ok {
		role, err := s.p.Roles.Lookup(ctx, e.OrganizationID, e.PractitionerRoleID)
		if err != nil && !errors.Is(err, practitionerrole.ErrNotFound) {
			return consent.Candidate{}, err
		}
		specialty = role.SpecialtyCode
		specialties[e.PractitionerRoleID] = specialty
	}
	docs, err := s.p.Documents.Signed(ctx, patient, e.ID)
	if err != nil {
		return consent.Candidate{}, err
	}
	v := consent.Candidate{
		Date: e.PeriodStart, EncounterID: e.ID, OrganizationID: e.OrganizationID, OrganizationName: name, Specialty: specialty,
	}
	if len(docs) > 0 {
		v.Title = docs[0].Draft.Title
	}
	return v, nil
}
