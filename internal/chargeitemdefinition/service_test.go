package chargeitemdefinition_test

import (
	"errors"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/chargeitemdefinition"
	"github.com/Shmeeshmee/salus/internal/schedule"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

// Step 0 is the contract only: every method refuses until package B builds it (docs/plans/clinic-cms.md).
func TestService_RefusesEveryMethodUntilPackageB(t *testing.T) {
	svc := chargeitemdefinition.NewService(nil, clock.Fixed{T: time.Date(2026, 11, 21, 9, 0, 0, 0, time.UTC)})
	ctx, c := t.Context(), chargeitemdefinition.Caller{AccountID: "a", OrganizationID: "o", Admin: true}
	_, list := svc.List(ctx, c)
	_, create := svc.Create(ctx, c, chargeitemdefinition.Definition{})
	_, update := svc.Update(ctx, c, chargeitemdefinition.Definition{})
	_, offered := svc.Offered(ctx, "o", schedule.ModeInPerson, "en")
	_, lines := svc.Lines(ctx, "o", []chargeitemdefinition.Pick{{DefinitionID: "d", Quantity: 1}})
	for name, err := range map[string]error{"List": list, "Create": create, "Update": update, "Offered": offered, "Lines": lines} {
		if !errors.Is(err, errors.ErrUnsupported) {
			t.Errorf("%s() = %v, want errors.ErrUnsupported", name, err)
		}
	}
}
