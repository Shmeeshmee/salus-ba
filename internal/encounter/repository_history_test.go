package encounter_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/encounter"
	"github.com/Shmeeshmee/salus/pkg/keyset"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// PatientHistory pages the patient's encounters newest first by start, the reference breaking a tie byte by byte,
// strictly after the position, in the range and at the clinic asked.
func TestPatientHistory_PagesThePatientsEncountersNewestFirst(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	at := time.Date(2030, 1, 7, 9, 2, 0, 0, time.UTC)
	s := seedDB(t, owner, at)
	var es [2]encounter.Encounter
	for i := range es {
		e, _, err := startIn(t, app, s, s.encounter(i, at)) // the same start: the reference decides
		if err != nil {
			t.Fatal(err)
		}
		es[i] = e
	}
	first, second := es[0], es[1]
	if "Encounter/"+first.ID < "Encounter/"+second.ID {
		first, second = second, first
	}
	page := func(org string, p keyset.Page) []string {
		t.Helper()
		var ids []string
		err := inScope(t, app, "", s.patient, func(ctx context.Context, r encounter.Repository) error {
			got, err := r.PatientHistory(ctx, s.patient, org, p)
			for _, e := range got {
				ids = append(ids, e.ID)
			}
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
		return ids
	}
	for name, tc := range map[string]struct {
		org  string
		p    keyset.Page
		want []string
	}{
		"every one":           {p: keyset.Page{Limit: 10}, want: []string{first.ID, second.ID}},
		"the limit":           {p: keyset.Page{Limit: 1}, want: []string{first.ID}},
		"after the first":     {p: keyset.Page{At: at, Ref: "Encounter/" + first.ID, Limit: 10}, want: []string{second.ID}},
		"after the second":    {p: keyset.Page{At: at, Ref: "Encounter/" + second.ID, Limit: 10}},
		"after a later start": {p: keyset.Page{At: at.Add(time.Second), Ref: "Encounter/", Limit: 10}, want: []string{first.ID, second.ID}},
		"after another type":  {p: keyset.Page{At: at, Ref: "Composition/" + first.ID, Limit: 10}},
		"from after":          {p: keyset.Page{From: at.Add(time.Second), Limit: 10}},
		"to it":               {p: keyset.Page{To: at, Limit: 10}},
		"in range":            {p: keyset.Page{From: at, To: at.Add(time.Second), Limit: 10}, want: []string{first.ID, second.ID}},
		"its clinic":          {org: s.clinic, p: keyset.Page{Limit: 10}, want: []string{first.ID, second.ID}},
		"another clinic":      {org: "00000000-0000-4000-8000-000000000000", p: keyset.Page{Limit: 10}},
		"a clinic not a UUID": {org: "x", p: keyset.Page{Limit: 10}},
	} {
		if got := page(tc.org, tc.p); !slices.Equal(got, tc.want) {
			t.Errorf("%s: PatientHistory() = %v, want %v", name, got, tc.want)
		}
	}
	other := seedDB(t, owner, at)
	err := inScope(t, app, "", other.patient, func(ctx context.Context, r encounter.Repository) error {
		got, err := r.PatientHistory(ctx, s.patient, "", keyset.Page{Limit: 10})
		if len(got) != 0 {
			t.Errorf("another patient's scope: PatientHistory() = %+v, want none", got)
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

// OfPatient gives the patient's encounters the scope shows, oldest first: their own in their scope, none in
// another patient's.
func TestOfPatient_GivesThePatientsEncounters(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	at := time.Date(2030, 1, 7, 9, 2, 0, 0, time.UTC)
	s, other := seedDB(t, owner, at), seedDB(t, owner, at)
	first, _, err := startIn(t, app, s, s.encounter(0, at))
	if err != nil {
		t.Fatal(err)
	}
	second, _, err := startIn(t, app, s, s.encounter(1, at.Add(time.Hour)))
	if err != nil {
		t.Fatal(err)
	}
	for scope, want := range map[string][]string{s.patient: {first.ID, second.ID}, other.patient: nil} {
		var ids []string
		err := inScope(t, app, "", scope, func(ctx context.Context, r encounter.Repository) error {
			got, err := r.OfPatient(ctx, s.patient)
			for _, e := range got {
				ids = append(ids, e.ID)
			}
			return err
		})
		if err != nil || !slices.Equal(ids, want) {
			t.Errorf("OfPatient() in %s's scope = %v, %v; want %v", scope, ids, err, want)
		}
	}
}
