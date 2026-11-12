package organization_test

import (
	"context"
	"errors"
	"maps"
	"testing"

	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

// SAL-39: Names gives the name of each clinic of the patient's own timeline, listed or not, each read in its own
// scope, unaudited (a clinic holds no personal data); an unknown clinic is left out.
func TestNames_GivesEachClinicsNameUnaudited(t *testing.T) {
	repo, audit := newFakeRepository(), &fakeAuditor{}
	got, err := newService(repo, audit).Names(t.Context(), []string{clinicA, clinicB, clinicA, "d0000000-0000-4000-8000-00000000000d"})
	want := map[string]string{clinicA: "Synthetic clinic A", clinicB: "Synthetic clinic B"}
	if err != nil || !maps.Equal(got, want) {
		t.Errorf("Names() = %v, %v; want %v", got, err, want)
	}
	if len(repo.calls) != 6 || len(audit.events) != 0 {
		t.Errorf("calls %v, events %d; want each clinic once in its scope, no event", repo.calls, len(audit.events))
	}
	repo.err = errors.New("the database is down")
	if _, err := newService(repo, audit).Names(t.Context(), []string{clinicA}); !errors.Is(err, repo.err) {
		t.Errorf("Names() = %v, want the backend's error", err)
	}
}

// countingTx counts its transactions; a nested one joins the outer, as postgres.Transactor's does.
type countingTx struct{ n int }

func (c *countingTx) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if ctx.Value(txKey{}) != nil {
		return fn(ctx)
	}
	c.n++
	return fn(context.WithValue(ctx, txKey{}, true))
}

// Names reads every clinic in one transaction, not one each (the review of SAL-39, finding 5).
func TestNames_ReadsEveryClinicInOneTransaction(t *testing.T) {
	repo, tx := newFakeRepository(), &countingTx{}
	got, err := organization.NewService(repo, &fakeAuditor{}, tx, clock.Fixed{T: now}).Names(t.Context(), []string{clinicA, clinicB})
	if err != nil || len(got) != 2 || tx.n != 1 {
		t.Errorf("Names() = %v, %v in %d transactions; want both in one", got, err, tx.n)
	}
}
