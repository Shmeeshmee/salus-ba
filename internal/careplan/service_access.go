package careplan

import (
	"context"
	"errors"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/practitionerrole"
	"github.com/Shmeeshmee/salus/pkg/fhir"
)

// works returns ErrNotFound unless staff c may work with the patient's plans (access.CarePlan: the clinic's admin
// for any patient of the clinic, a practitioner for a patient they have or had a visit with), as for a patient c
// cannot see. It reads in the transaction in ctx, in the clinic's scope.
func (s *Service) works(ctx context.Context, c Staff, patientID string) error {
	if !fhir.IsUUID(patientID) {
		return ErrNotFound
	}
	v := access.Viewer{Role: access.RoleAdmin, Clinic: c.OrganizationID}
	if !c.Admin {
		role, err := s.p.Roles.OfPractitioner(ctx, c.OrganizationID, c.PractitionerID)
		switch {
		case errors.Is(err, practitionerrole.ErrNotFound):
			return ErrNotFound
		case err != nil:
			return err
		}
		v = access.Viewer{Role: access.RolePractitioner, Clinic: c.OrganizationID, PractitionerRole: role.ID}
	}
	state, err := s.p.Consents.State(ctx, patientID, c.OrganizationID)
	if err != nil {
		return err
	}
	visits, err := s.p.Care.Visits(ctx, patientID, c.OrganizationID)
	if err != nil {
		return err
	}
	if !access.CarePlan(v, access.Facts{Patient: patientID, Clinic: c.OrganizationID, Consent: state, Visits: visits}) {
		return ErrNotFound
	}
	return nil
}

// seen reads plan id for staff c: ErrNotFound for one the clinic does not have or c may not work with (works).
func (s *Service) seen(ctx context.Context, c Staff, id string) (Plan, error) {
	p, err := s.p.Repo.ByID(ctx, id)
	if err != nil {
		return Plan{}, err
	}
	if err := s.works(ctx, c, p.PatientID); err != nil {
		return Plan{}, err
	}
	return p, nil
}

// named names each step of plans by its clinic's price list (a service gone from it keeps no name), each clinic's
// read once, in its own scope.
func (s *Service) named(ctx context.Context, plans []Plan) error {
	names := map[string]map[string]string{} // service names by clinic
	for _, p := range plans {
		if len(p.Steps) == 0 {
			continue
		}
		byID, ok := names[p.OrganizationID]
		if !ok {
			prices, err := s.p.Prices.OfClinic(ctx, p.OrganizationID)
			if err != nil {
				return err
			}
			byID = make(map[string]string, len(prices))
			for _, d := range prices {
				byID[d.ID] = d.Name
			}
			names[p.OrganizationID] = byID
		}
		for i := range p.Steps {
			p.Steps[i].ServiceName = byID[p.Steps[i].ServiceID]
		}
	}
	return nil
}
