package diagnosticreport_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/diagnosticreport"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

// Until LB1 of docs/clinic-cms/WAVE6.md builds them, every method refuses and touches nothing.
func TestService_RefusesUntilItsPackageBuildsIt(t *testing.T) {
	t.Parallel()
	s := diagnosticreport.NewService(diagnosticreport.Ports{Clock: clock.Fixed{T: time.Date(2026, 11, 25, 9, 0, 0, 0, time.UTC)}})
	ctx := t.Context()
	staff := diagnosticreport.Staff{AccountID: "00000000-0000-4000-8000-000000000001", OrganizationID: "00000000-0000-4000-8000-000000000002", Practitioner: true}
	patient := diagnosticreport.Patient{AccountID: "00000000-0000-4000-8000-000000000003", PatientID: "00000000-0000-4000-8000-000000000004"}
	const id = "00000000-0000-4000-8000-000000000005"
	draft := diagnosticreport.Draft{Title: "Complete blood count"}
	calls := map[string]func() error{
		"Attach": func() error { _, err := s.Attach(ctx, staff, id, draft); return err },
		"Change": func() error { _, err := s.Change(ctx, staff, id, draft); return err },
		"AttachPDF": func() error {
			_, err := s.AttachPDF(ctx, staff, id, strings.NewReader("%PDF-"), 5)
			return err
		},
		"Release":  func() error { _, err := s.Release(ctx, staff, id, 1); return err },
		"ForVisit": func() error { _, err := s.ForVisit(ctx, staff, id); return err },
		"Read":     func() error { _, err := s.Read(ctx, staff, id); return err },
		"PDF":      func() error { _, _, err := s.PDF(ctx, staff, id); return err },
		"Mine":     func() error { _, err := s.Mine(ctx, patient); return err },
		"MyReport": func() error { _, err := s.MyReport(ctx, patient, id); return err },
		"MyPDF":    func() error { _, _, err := s.MyPDF(ctx, patient, id); return err },
	}
	for name, call := range calls {
		if err := call(); !errors.Is(err, errors.ErrUnsupported) {
			t.Errorf("%s: %v, want errors.ErrUnsupported", name, err)
		}
	}
}
