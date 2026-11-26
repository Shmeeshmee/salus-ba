package organization

import (
	"context"
	"errors"
	"fmt"
)

// Served gives the clinics whose public reads the signed-in caller keeps once their membership stops (Service.Served).
// A read of the directory calls it only when the directory no longer shows the clinic, so that the directory's reads
// cost what they did.
type Served func(ctx context.Context) ([]string, error)

// PatientClinics gives the clinics a patient has a care consent with, active or revoked (consent.Service).
type PatientClinics interface {
	CareClinics(ctx context.Context, patientID string) ([]string, error)
}

// WithPatients returns a copy of s whose Served knows a patient's clinics from p.
func (s *Service) WithPatients(p PatientClinics) *Service {
	out := *s
	out.patients = p
	return &out
}

// Served returns the listed clinics a clinic's suspended or cancelled membership (package I) does not take out of the
// directory for c (docs/clinic-cms/PROGRESS.md, "Readiness follow-ups: suspended clinics"): for its staff their own
// clinic, for a patient those they have a care consent with, active or revoked (every visit there was booked on one,
// and the clinic keeps their record); none for anyone else. Their locations, profile and schedules are then read in
// the clinic's own scope. Called outside the caller's transaction, as Names: each clinic is read in its own scope. Not
// audited: nothing in a clinic is personal data, and the patient's consents are read for their own access check, as
// consent.Service.State is.
func (s *Service) Served(ctx context.Context, c Caller) ([]string, error) {
	var ids []string
	switch {
	case c.OrganizationID != "":
		ids = []string{c.OrganizationID}
	case c.PatientID != "" && s.patients != nil:
		var err error
		if ids, err = s.patients.CareClinics(ctx, c.PatientID); err != nil {
			return nil, fmt.Errorf("organization: served: %w", err)
		}
	}
	var out []string
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		for _, id := range ids {
			o, err := s.byID(ctx, id)
			switch {
			case errors.Is(err, ErrNotFound):
				continue
			case err != nil:
				return err
			}
			if o.Listed {
				out = append(out, id)
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("organization: served: %w", err)
	}
	return out, nil
}
