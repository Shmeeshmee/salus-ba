package careplan_test

import (
	"errors"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/careplan"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

// Until TP1 of docs/clinic-cms/WAVE6.md builds them, every method refuses and touches nothing.
func TestService_RefusesUntilItsPackageBuildsIt(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 11, 25, 9, 0, 0, 0, time.UTC)
	s := careplan.NewService(careplan.Ports{Clock: clock.Fixed{T: now}})
	ctx := t.Context()
	staff := careplan.Staff{AccountID: "00000000-0000-4000-8000-000000000001", OrganizationID: "00000000-0000-4000-8000-000000000002", Practitioner: true}
	patient := careplan.Patient{AccountID: "00000000-0000-4000-8000-000000000003", PatientID: "00000000-0000-4000-8000-000000000004"}
	const id = "00000000-0000-4000-8000-000000000005"
	draft := careplan.Draft{PatientID: patient.PatientID, Title: "Synthetic plan"}
	calls := map[string]func() error{
		"Write":  func() error { _, err := s.Write(ctx, staff, draft); return err },
		"Change": func() error { _, err := s.Change(ctx, staff, id, draft); return err },
		"Book": func() error {
			_, err := s.Book(ctx, staff, id, careplan.Booking{StartsOn: now, Version: 1})
			return err
		},
		"Cancel":     func() error { _, err := s.Cancel(ctx, staff, id, 1); return err },
		"Read":       func() error { _, err := s.Read(ctx, staff, id); return err },
		"ForPatient": func() error { _, err := s.ForPatient(ctx, staff, patient.PatientID); return err },
		"Mine":       func() error { _, err := s.Mine(ctx, patient); return err },
	}
	for name, call := range calls {
		if err := call(); !errors.Is(err, errors.ErrUnsupported) {
			t.Errorf("%s: %v, want errors.ErrUnsupported", name, err)
		}
	}
}
