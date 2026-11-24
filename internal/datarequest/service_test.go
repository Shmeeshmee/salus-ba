package datarequest_test

import (
	"errors"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/datarequest"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

// Until their packages (D2 and D3 of docs/clinic-cms/WAVE5.md) build them, their methods refuse and touch nothing.
func TestService_RefusesUntilItsPackageBuildsIt(t *testing.T) {
	t.Parallel()
	s := datarequest.NewService(datarequest.Ports{Clock: clock.Fixed{T: time.Date(2026, 11, 24, 9, 0, 0, 0, time.UTC)}})
	ctx := t.Context()
	patient := datarequest.Patient{AccountID: "00000000-0000-4000-8000-000000000001"}
	const id = "00000000-0000-4000-8000-000000000004"
	calls := map[string]func() error{
		"CloseAccount":  func() error { return s.CloseAccount(ctx, patient, "a password") },
		"ReopenAccount": func() error { return s.ReopenAccount(ctx, patient.AccountID) },
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
	if datarequest.EraseAfter != 30*24*time.Hour || datarequest.ExportWindow != 7*24*time.Hour {
		t.Errorf("erase after %v, export window %v, want 30 and 7 days", datarequest.EraseAfter, datarequest.ExportWindow)
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
