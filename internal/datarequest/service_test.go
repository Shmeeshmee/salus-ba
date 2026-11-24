package datarequest_test

import (
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/datarequest"
)

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
