package consent_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/consent"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// States gives in one statement what State gives each patient (docs/clinic-cms/load/REPORT.md): the care consent, its
// latest revocation, the transcription and history consents and the history's selection, in the scope of ctx; a patient
// without any consent there has the zero state, an id that is not a UUID none.
func TestStates_GivesEachPatientWhatStateGives(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seed(t, owner)
	seedVisits(t, owner) // patient A's history consent at clinic A, with a selection
	repo := consent.NewPostgresRepository(app)
	for _, clinic := range []string{clinicA, clinicB} {
		set := func(ctx context.Context) error { return repo.SetClinic(ctx, clinic) }
		var got map[string]access.Consent
		want := map[string]access.Consent{}
		err := in(t, app, set, func(ctx context.Context) error {
			for _, p := range []string{patientA, patientB} {
				st, err := repo.State(ctx, p, clinic)
				if err != nil {
					return err
				}
				want[p] = st
			}
			var err error
			got, err = repo.States(ctx, []string{patientA, patientB, "not-a-uuid"}, clinic)
			return err
		})
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Errorf("clinic %s: States() = %+v, %v;\nwant %+v", clinic, got, err, want)
		}
	}
}
