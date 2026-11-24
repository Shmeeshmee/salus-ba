package datarequest_test

import (
	"errors"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/datarequest"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

// Until their packages (D1, D2 of docs/clinic-cms/WAVE5.md) build them, every method refuses and touches nothing.
func TestService_RefusesUntilItsPackageBuildsIt(t *testing.T) {
	t.Parallel()
	s := datarequest.NewService(datarequest.Ports{Clock: clock.Fixed{T: time.Date(2026, 11, 24, 9, 0, 0, 0, time.UTC)}})
	ctx := t.Context()
	patient := datarequest.Patient{AccountID: "00000000-0000-4000-8000-000000000001"}
	staff := datarequest.Staff{AccountID: "00000000-0000-4000-8000-000000000002", OrganizationID: "00000000-0000-4000-8000-000000000003", Admin: true}
	const id = "00000000-0000-4000-8000-000000000004"
	calls := map[string]func() error{
		"File": func() error {
			_, err := s.File(ctx, patient, datarequest.Filing{Kind: datarequest.KindAccess, Target: datarequest.TargetAccount})
			return err
		},
		"Mine":     func() error { _, err := s.Mine(ctx, patient); return err },
		"Withdraw": func() error { _, err := s.Withdraw(ctx, patient, id, 1); return err },
		"FileOnBehalf": func() error {
			_, err := s.FileOnBehalf(ctx, staff, id, datarequest.KindErasure)
			return err
		},
		"ClinicRequests": func() error { _, err := s.ClinicRequests(ctx, staff); return err },
		"ClinicRequest":  func() error { _, err := s.ClinicRequest(ctx, staff, id); return err },
		"Answer": func() error {
			_, err := s.Answer(ctx, staff, id, datarequest.Answer{Status: datarequest.StatusFulfilled, Code: datarequest.AnswerDone, Version: 1})
			return err
		},
		"Extend": func() error {
			_, err := s.Extend(ctx, staff, id, datarequest.ExtensionComplex, 1)
			return err
		},
		"Export":        func() error { _, err := s.Export(ctx, patient, id); return err },
		"ExportArchive": func() error { _, _, err := s.ExportArchive(ctx, patient, id); return err },
	}
	for name, call := range calls {
		if err := call(); !errors.Is(err, errors.ErrUnsupported) {
			t.Errorf("%s: %v, want errors.ErrUnsupported", name, err)
		}
	}
}

// A request is due one month after it was received, and an extension adds at most two months more (S3).
func TestDeadlines_AreTheContractsConstants(t *testing.T) {
	t.Parallel()
	if datarequest.DueMonths != 1 || datarequest.ExtensionMonths != 2 {
		t.Errorf("due %d, extension %d months, want 1 and 2", datarequest.DueMonths, datarequest.ExtensionMonths)
	}
	if datarequest.EraseAfter != 21*24*time.Hour || datarequest.ExportWindow != 7*24*time.Hour {
		t.Errorf("erase after %v, export window %v, want 21 and 7 days", datarequest.EraseAfter, datarequest.ExportWindow)
	}
}

// A request without a clinic targets the patient's account; with one, that clinic (S1).
func TestRequest_TargetIsTheAccountWithoutAClinic(t *testing.T) {
	t.Parallel()
	if got := (datarequest.Request{}).Target(); got != datarequest.TargetAccount {
		t.Errorf("no clinic: %q, want %q", got, datarequest.TargetAccount)
	}
	if got := (datarequest.Request{OrganizationID: "00000000-0000-4000-8000-000000000003"}).Target(); got != datarequest.TargetClinic {
		t.Errorf("a clinic: %q, want %q", got, datarequest.TargetClinic)
	}
}
