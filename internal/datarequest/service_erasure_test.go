package datarequest_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Shmeeshmee/salus/internal/consent"
	"github.com/Shmeeshmee/salus/internal/datarequest"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

func (f *fixture) erasure() *datarequest.Erasure {
	return datarequest.NewErasure(datarequest.ErasurePorts{
		Tx: fakeTx{}, Repo: f.repo, Scope: f.scope, Accounts: f.accounts, Records: f.consents,
		Restrictions: f.restrictions, Devices: f.devices, Clock: clock.Fixed{T: now.Add(datarequest.EraseAfter)},
	})
}

// closed is a fixture whose account was closed and is now due for erasure.
func closed(t *testing.T, profile string, consents ...consent.Consent) *fixture {
	t.Helper()
	f := newFixture(profile, consents...)
	if _, err := f.closure().CloseAccount(t.Context(), datarequest.Patient{AccountID: accountID}, password); err != nil {
		t.Fatal(err)
	}
	f.accounts.due = []string{accountID}
	f.devices.removed = nil
	return f
}

func TestErasure_ErasesTheAccountRestrictsEachClinicsRecordAndFulfilsTheRequest(t *testing.T) {
	t.Parallel()
	f := closed(t, profileID,
		careAt("c1", clinicA, consent.KindCare, consent.StatusActive),
		careAt("c2", clinicB, consent.KindCare, consent.StatusInactive),
	)

	got, err := f.erasure().Run(t.Context())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	want := []datarequest.Erased{{AccountID: accountID, RequestID: f.repo.requests[0].ID, Kept: []string{clinicA, clinicB}}}
	if len(got) != 1 || got[0].AccountID != want[0].AccountID || got[0].RequestID != want[0].RequestID ||
		!slices.Equal(got[0].Kept, want[0].Kept) {
		t.Errorf("Run() = %+v, want %+v", got, want)
	}
	if !slices.Equal(f.accounts.erased, []string{accountID}) || !slices.Equal(f.devices.removed, []string{accountID}) {
		t.Errorf("erased %v, devices removed %v; want the account's", f.accounts.erased, f.devices.removed)
	}
	if !f.restrictions.restricted[profileID+"@"+clinicA] || !f.restrictions.restricted[profileID+"@"+clinicB] {
		t.Errorf("restricted %v, want the record at both clinics", f.restrictions.restricted)
	}
	r := f.repo.requests[0]
	if r.Status != datarequest.StatusFulfilled || r.AnswerCode != datarequest.AnswerDonePartlyKept ||
		!r.AnsweredAt.Equal(now.Add(datarequest.EraseAfter)) || r.Version != 3 {
		t.Errorf("request %+v, want fulfilled done_partly_kept at the run", r)
	}
	if last := f.repo.changes[len(f.repo.changes)-1]; last.Status != datarequest.StatusFulfilled || last.ChangedBy != "" {
		t.Errorf("last change %+v, want fulfilled by the job", last)
	}
}

func TestErasure_APatientWithNoRecordAnywhereIsErasedWhole(t *testing.T) {
	t.Parallel()
	f := closed(t, profileID)
	got, err := f.erasure().Run(t.Context())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(got) != 1 || len(got[0].Kept) != 0 || len(f.restrictions.restricted) != 0 {
		t.Errorf("Run() = %+v, restricted %v; want nothing kept", got, f.restrictions.restricted)
	}
	if r := f.repo.requests[0]; r.Status != datarequest.StatusFulfilled || r.AnswerCode != datarequest.AnswerDone {
		t.Errorf("request %+v, want fulfilled done", r)
	}
}

func TestErasure_ASecondRunErasesNothingMore(t *testing.T) {
	t.Parallel()
	f := closed(t, profileID, careAt("c1", clinicA, consent.KindCare, consent.StatusActive))
	if _, err := f.erasure().Run(t.Context()); err != nil {
		t.Fatal(err)
	}
	changes := len(f.repo.changes)
	got, err := f.erasure().Run(t.Context())
	if err != nil || len(got) != 0 {
		t.Errorf("second Run() = %+v, %v; want nothing", got, err)
	}
	if len(f.accounts.erased) != 1 || len(f.repo.changes) != changes {
		t.Errorf("erased %v, %d changes (was %d); want nothing more", f.accounts.erased, len(f.repo.changes), changes)
	}
}

// Every closure files its erasure request; an account due without one is not erased blind (its profile, and so the
// records to keep, would be unknown): the run fails and alerts.
func TestErasure_RefusesAnAccountWithoutItsErasureRequest(t *testing.T) {
	t.Parallel()
	f := newFixture(profileID)
	f.accounts.due = []string{accountID}
	if _, err := f.erasure().Run(t.Context()); !errors.Is(err, datarequest.ErrNotFound) {
		t.Fatalf("Run() error = %v, want ErrNotFound", err)
	}
	if len(f.accounts.erased) != 0 {
		t.Errorf("erased %v, want nothing", f.accounts.erased)
	}
}
