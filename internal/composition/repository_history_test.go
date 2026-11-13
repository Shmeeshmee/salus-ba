package composition_test

import (
	"context"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/composition"
	"github.com/Shmeeshmee/salus/pkg/keyset"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// SignedOf pages the signed documents of the encounters asked, newest first by signature, strictly after the
// position and in the range; never a draft that is not signed, nor another patient's in the patient's scope.
func TestSignedOf_PagesTheSignedDocumentsOfTheEncounters(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	s, other := seedEncounter(t, owner), seedEncounter(t, owner)
	_, signed := openAndSigned(t, app, s)
	signedAt := at.Add(time.Hour)
	ref := "Composition/" + signed.ID
	for name, tc := range map[string]struct {
		p    keyset.Page
		encs []string
		want int
	}{
		"the signed one":       {p: keyset.Page{Limit: 10}, encs: []string{s.encounter}, want: 1},
		"the limit":            {p: keyset.Page{Limit: 0}, encs: []string{s.encounter}},
		"after it":             {p: keyset.Page{At: signedAt, Ref: ref, Limit: 10}, encs: []string{s.encounter}},
		"after a later ref":    {p: keyset.Page{At: signedAt, Ref: "Encounter/", Limit: 10}, encs: []string{s.encounter}, want: 1},
		"after an earlier ref": {p: keyset.Page{At: signedAt, Ref: "Binary/", Limit: 10}, encs: []string{s.encounter}},
		"from after it":        {p: keyset.Page{From: signedAt.Add(time.Second), Limit: 10}, encs: []string{s.encounter}},
		"to it":                {p: keyset.Page{To: signedAt, Limit: 10}, encs: []string{s.encounter}},
		"both encounters":      {p: keyset.Page{Limit: 10}, encs: []string{s.encounter, other.encounter}, want: 1},
		"no encounter":         {p: keyset.Page{Limit: 10}},
	} {
		var got []composition.Draft
		err := asPatient(t, app, s.patient, func(ctx context.Context, r composition.Repository) error {
			var err error
			got, err = r.SignedOf(ctx, tc.encs, tc.p)
			return err
		})
		if err != nil || len(got) != tc.want || (tc.want == 1 && got[0].ID != signed.ID) {
			t.Errorf("%s: SignedOf() = %+v, %v; want %d, the signed one", name, got, err, tc.want)
		}
	}
	err := asPatient(t, app, s.patient, func(ctx context.Context, r composition.Repository) error {
		_, err := r.SignedOf(ctx, []string{"x"}, keyset.Page{Limit: 10})
		return err
	})
	if err == nil {
		t.Error("SignedOf(an id that is not a UUID) = nil, want an error")
	}
	err = asPatient(t, app, other.patient, func(ctx context.Context, r composition.Repository) error {
		got, err := r.SignedOf(ctx, []string{s.encounter}, keyset.Page{Limit: 10})
		if len(got) != 0 {
			t.Errorf("another patient's scope: SignedOf() = %+v, want none", got)
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}
