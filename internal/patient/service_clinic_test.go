package patient_test

import (
	"errors"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/patient"
)

// The clinic's own patients (search, register a patient without an account) are the wave 3 contract
// (docs/clinic-cms, 3.0): refused until package F2 builds them.
func TestClinicPatients_RefusesUntilPackageF2(t *testing.T) {
	t.Parallel()
	var s patient.Service
	c := patient.Caller{AccountID: "a", OrganizationID: "o", Admin: true}
	birth := time.Date(1990, 1, 2, 0, 0, 0, 0, time.UTC)
	if _, err := s.SearchClinic(t.Context(), c, patient.ClinicSearch{Name: "x", BirthDate: birth}); !errors.Is(err, errors.ErrUnsupported) {
		t.Errorf("SearchClinic = %v, want errors.ErrUnsupported", err)
	}
	if _, err := s.Register(t.Context(), c, patient.Patient{GivenName: "x", FamilyName: "y", BirthDate: birth}); !errors.Is(err, errors.ErrUnsupported) {
		t.Errorf("Register = %v, want errors.ErrUnsupported", err)
	}
}
