package subscription_test

import (
	"errors"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/subscription"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

// Step 0 is the contract only: every method refuses until package D builds it (docs/plans/clinic-cms.md).
func TestService_RefusesEveryMethodUntilPackageD(t *testing.T) {
	at := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
	svc := subscription.NewService(nil, clock.Fixed{T: at})
	ctx, c := t.Context(), subscription.Caller{AccountID: "a", OrganizationID: "o", Admin: true}
	_, current := svc.Current(ctx, c)
	_, invoices := svc.Invoices(ctx, c)
	_, trial := svc.StartTrial(ctx, "o")
	_, monthly := svc.IssueMonthly(ctx, "o", at)
	_, paid := svc.MarkPaid(ctx, 1, at)
	for name, err := range map[string]error{"Current": current, "Invoices": invoices, "StartTrial": trial, "IssueMonthly": monthly, "MarkPaid": paid} {
		if !errors.Is(err, errors.ErrUnsupported) {
			t.Errorf("%s() = %v, want errors.ErrUnsupported", name, err)
		}
	}
}

// The monthly job carries opaque ids only.
func TestMonthlyArgs_IsTheSubscriptionMonthlyJob(t *testing.T) {
	if got := (subscription.MonthlyArgs{OrganizationID: "o"}).Kind(); got != "subscription_monthly" {
		t.Errorf("Kind() = %q, want subscription_monthly", got)
	}
}
