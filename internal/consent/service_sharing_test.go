package consent_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/consent"
)

// Synthetic visits of patient A at clinic B, which patient A may share with clinic A.
const (
	visit1    = "e1000000-0000-4000-8000-0000000000e1"
	visit2    = "e2000000-0000-4000-8000-0000000000e2"
	visit3    = "e3000000-0000-4000-8000-0000000000e3"
	historyAA = "f4000000-0000-4000-8000-0000000000f4" // patient A's history consent with clinic A, when a test adds it
)

// fakeVisits gives the visits patient A may share, as history.Sharing does after its rules; it records the clinics
// asked for and fails with err.
type fakeVisits struct {
	err    error
	asked  []string
	visits []consent.Candidate
}

func (v *fakeVisits) Shareable(ctx context.Context, c consent.Caller, organizationID string) ([]consent.Candidate, error) {
	if ctx.Value(txKey{}) == nil {
		return nil, errors.New("read outside a transaction")
	}
	v.asked = append(v.asked, organizationID)
	if v.err != nil {
		return nil, v.err
	}
	if organizationID != clinicA && organizationID != clinicB {
		return nil, consent.ErrClinicNotFound
	}
	if c.PatientID != patientA {
		return []consent.Candidate{}, nil
	}
	return slices.Clone(v.visits), nil
}

func candidate(id string, day int) consent.Candidate {
	return consent.Candidate{
		Date: time.Date(2026, 10, day, 9, 0, 0, 0, time.UTC), EncounterID: id, OrganizationID: clinicB,
		OrganizationName: "Synthetic clinic B", Specialty: "394579002", Title: "Placeholder findings",
	}
}

func sharingFixture() fixture {
	f := newFixture()
	f.visits.visits = []consent.Candidate{candidate(visit3, 22), candidate(visit2, 21), candidate(visit1, 20)}
	return f
}

// withHistory is the sharing fixture with patient A's active history consent with clinic A, listing visit 1, at
// version 2.
func withHistory() fixture {
	f := sharingFixture()
	f.repo.consents = append(f.repo.consents, consent.Consent{
		ID: historyAA, PatientID: patientA, OrganizationID: clinicA, Kind: consent.KindHistory, Status: consent.StatusActive,
		DateTime: granted, LastUpdated: granted, VersionID: 2, Shared: []consent.SharedVisit{{EncounterID: visit1, AddedAt: granted}},
	})
	return f
}

func (f fixture) history() consent.Consent {
	for _, c := range f.repo.consents {
		if c.Kind == consent.KindHistory && c.Status == consent.StatusActive {
			return c
		}
	}
	return consent.Consent{}
}

func shared(c consent.Consent) []string {
	var ids []string
	for _, v := range c.Shared {
		ids = append(ids, v.EncounterID)
	}
	return ids
}

func TestCandidates_GivesThePatientsVisitsAtOtherClinicsMarkedWhenSharedAndAuditsIt(t *testing.T) {
	f := withHistory()
	got, err := f.svc.Candidates(t.Context(), me, clinicA)
	if err != nil || len(got) != 3 {
		t.Fatalf("Candidates() = %+v, %v; want the three visits", got, err)
	}
	for _, c := range got {
		if c.Selected != (c.EncounterID == visit1) {
			t.Errorf("candidate %s selected %v, want only visit 1 selected", c.EncounterID, c.Selected)
		}
	}
	if !slices.Equal(f.visits.asked, []string{clinicA}) || f.repo.patient != patientA {
		t.Errorf("asked %v with patient %q, want clinic A in patient A's scope", f.visits.asked, f.repo.patient)
	}
	checkEvents(t, "Candidates", f.audit, event(me, auditevent.CodeSearchType, historyAA, auditevent.OutcomeSuccess))
}

func TestCandidates_WithNothingSharedSelectsNoneAndAuditsWithoutAConsent(t *testing.T) {
	f := sharingFixture()
	got, err := f.svc.Candidates(t.Context(), me, clinicA)
	if err != nil || len(got) != 3 || slices.ContainsFunc(got, func(c consent.Candidate) bool { return c.Selected }) {
		t.Fatalf("Candidates() = %+v, %v; want three, none selected", got, err)
	}
	checkEvents(t, "Candidates", f.audit, event(me, auditevent.CodeSearchType, "", auditevent.OutcomeSuccess))
	if got, err := f.svc.Candidates(t.Context(), noProf, clinicA); err != nil || got == nil || len(got) != 0 {
		t.Errorf("Candidates() without a profile = %#v, %v; want an empty list", got, err)
	}
}

func TestCandidates_RefusesStaffAndAnUnknownClinicAndAuditsIt(t *testing.T) {
	for _, tt := range []struct {
		want   error
		caller consent.Caller
		name   string
		clinic string
	}{
		{name: "staff", caller: staff, clinic: clinicA, want: consent.ErrForbidden},
		{name: "an unknown clinic", caller: me, clinic: newID, want: consent.ErrClinicNotFound},
	} {
		f := sharingFixture()
		if _, err := f.svc.Candidates(t.Context(), tt.caller, tt.clinic); !errors.Is(err, tt.want) {
			t.Errorf("%s: Candidates() = %v, want %v", tt.name, err, tt.want)
		}
		checkEvents(t, tt.name, f.audit, event(tt.caller, auditevent.CodeSearchType, "", auditevent.OutcomeMinorFailure))
	}
}

func TestSetSharing_AFirstGrantListsExactlyTheVisitsSentAndAuditsIt(t *testing.T) {
	f := sharingFixture()
	got, err := f.svc.SetSharing(t.Context(), me, clinicA, consent.SetSharing{EncounterIDs: []string{visit3, visit1, visit3}}, 0)
	want := consent.Sharing{OrganizationID: clinicA, ConsentID: newID, VersionID: "1", Shared: []string{visit3, visit1}}
	if err != nil || !sameSharing(got, want) {
		t.Fatalf("SetSharing() = %+v, %v; want %+v", got, err, want)
	}
	h := f.history()
	if h.ID != newID || !h.DateTime.Equal(now) || !slices.Equal(shared(h), []string{visit3, visit1}) {
		t.Errorf("stored %+v, want an active history consent granted now listing visits 3 and 1", h)
	}
	checkEvents(t, "grant", f.audit, event(me, auditevent.CodeCreate, newID, auditevent.OutcomeSuccess))
}

func TestSetSharing_AChangeAddsAndRemovesAtTheVersionReadAndAuditsIt(t *testing.T) {
	f := withHistory()
	got, err := f.svc.SetSharing(t.Context(), me, clinicA, consent.SetSharing{EncounterIDs: []string{visit2}}, 2)
	want := consent.Sharing{OrganizationID: clinicA, ConsentID: historyAA, VersionID: "3", Shared: []string{visit2}}
	if err != nil || !sameSharing(got, want) {
		t.Fatalf("SetSharing() = %+v, %v; want %+v", got, err, want)
	}
	if want := []string{"ChangeHistory " + historyAA + " v2 +[" + visit2 + "] -[" + visit1 + "]"}; !slices.Equal(f.repo.changes, want) {
		t.Errorf("changes %v, want %v", f.repo.changes, want)
	}
	checkEvents(t, "change", f.audit, event(me, auditevent.CodeUpdate, historyAA, auditevent.OutcomeSuccess))
}

func TestSetSharing_TheSameSelectionChangesNothing(t *testing.T) {
	f := withHistory()
	got, err := f.svc.SetSharing(t.Context(), me, clinicA, consent.SetSharing{EncounterIDs: []string{visit1}}, 2)
	if want := (consent.Sharing{OrganizationID: clinicA, ConsentID: historyAA, VersionID: "2", Shared: []string{visit1}}); err != nil || !sameSharing(got, want) {
		t.Fatalf("SetSharing() = %+v, %v; want %+v", got, err, want)
	}
	if len(f.repo.changes) != 0 {
		t.Errorf("changes %v, want none", f.repo.changes)
	}
	checkEvents(t, "unchanged", f.audit, event(me, auditevent.CodeUpdate, historyAA, auditevent.OutcomeSuccess))
}

// An empty selection revokes the history consent (an active one lists at least one visit): from its commit the clinic
// sees none of the visits.
func TestSetSharing_NoVisitsRevokesTheSharing(t *testing.T) {
	f := withHistory()
	got, err := f.svc.SetSharing(t.Context(), me, clinicA, consent.SetSharing{EncounterIDs: []string{}}, 2)
	if want := (consent.Sharing{OrganizationID: clinicA, Shared: []string{}}); err != nil || !sameSharing(got, want) {
		t.Fatalf("SetSharing([]) = %+v, %v; want %+v", got, err, want)
	}
	if h := f.history(); h.ID != "" {
		t.Errorf("an active history consent is left: %+v", h)
	}
	if st, _ := f.svc.State(t.Context(), patientA, clinicA); st.History || len(st.Shared) != 0 || !st.Active {
		t.Errorf("State() = %+v, want the care consent only", st)
	}
	checkEvents(t, "revoke", f.audit, event(me, auditevent.CodeUpdate, historyAA, auditevent.OutcomeSuccess))
	g := sharingFixture()
	if got, err := g.svc.SetSharing(t.Context(), me, clinicA, consent.SetSharing{}, 0); err != nil || len(got.Shared) != 0 || got.ConsentID != "" {
		t.Errorf("SetSharing([]) with nothing shared = %+v, %v; want nothing", got, err)
	}
}

func TestSetSharing_RefusesAndAuditsEveryCaseOfRefusal(t *testing.T) {
	tests := []struct {
		want    error
		fixture func() fixture
		caller  consent.Caller
		name    string
		clinic  string
		visits  []string
		version int32
	}{
		{name: "staff", fixture: sharingFixture, caller: staff, clinic: clinicA, visits: []string{visit1}, want: consent.ErrForbidden},
		{name: "without a profile", fixture: sharingFixture, caller: noProf, clinic: clinicA, visits: []string{visit1}, want: consent.ErrCareConsentRequired},
		{name: "without the care consent", fixture: sharingFixture, caller: me, clinic: clinicB, visits: []string{visit1}, want: consent.ErrCareConsentRequired},
		{name: "a visit that is not a candidate", fixture: sharingFixture, caller: me, clinic: clinicA, visits: []string{visit1, newID}, want: consent.ErrInvalidVisit},
		{name: "a version with nothing shared", fixture: sharingFixture, caller: me, clinic: clinicA, visits: []string{visit1}, version: 1, want: consent.ErrVersionConflict},
		{name: "no version with a sharing", fixture: withHistory, caller: me, clinic: clinicA, visits: []string{visit2}, want: consent.ErrVersionConflict},
		{name: "a stale version", fixture: withHistory, caller: me, clinic: clinicA, visits: []string{visit2}, version: 1, want: consent.ErrVersionConflict},
		{name: "a stale revocation", fixture: withHistory, caller: me, clinic: clinicA, visits: []string{}, version: 1, want: consent.ErrVersionConflict},
	}
	for _, tt := range tests {
		f := tt.fixture()
		before := f.history()
		if _, err := f.svc.SetSharing(t.Context(), tt.caller, tt.clinic, consent.SetSharing{EncounterIDs: tt.visits}, tt.version); !errors.Is(err, tt.want) {
			t.Errorf("%s: SetSharing() = %v, want %v", tt.name, err, tt.want)
		}
		if after := f.history(); after.ID != before.ID || after.VersionID != before.VersionID || len(f.repo.changes) != 0 {
			t.Errorf("%s: the sharing became %+v after %v, want it unchanged", tt.name, after, f.repo.changes)
		}
		checkEvents(t, tt.name, f.audit, event(tt.caller, auditevent.CodeUpdate, "", auditevent.OutcomeMinorFailure))
	}
}

// A concurrent change commits between the read and the write: the version no longer matches.
func TestSetSharing_LosingARaceIsAConflict(t *testing.T) {
	for _, tt := range []struct {
		fixture func() fixture
		name    string
		version int32
	}{
		{name: "a first grant", fixture: sharingFixture},
		{name: "a change", fixture: withHistory, version: 2},
	} {
		f := tt.fixture()
		f.repo.race = true
		if _, err := f.svc.SetSharing(t.Context(), me, clinicA, consent.SetSharing{EncounterIDs: []string{visit2}}, tt.version); !errors.Is(err, consent.ErrVersionConflict) {
			t.Errorf("%s: SetSharing() = %v, want ErrVersionConflict", tt.name, err)
		}
		checkEvents(t, tt.name, f.audit, event(me, auditevent.CodeUpdate, "", auditevent.OutcomeMinorFailure))
	}
}

// Row-level security refuses a visit the service passed (a visit finished elsewhere meanwhile is not one): the
// insert aborted the transaction, so the refusal is audited in a transaction of its own.
func TestSetSharing_AVisitRowLevelSecurityRefusesIsInvalid(t *testing.T) {
	f := sharingFixture()
	f.repo.refuseVisits = true
	if _, err := f.svc.SetSharing(t.Context(), me, clinicA, consent.SetSharing{EncounterIDs: []string{visit2}}, 0); !errors.Is(err, consent.ErrInvalidVisit) {
		t.Errorf("SetSharing() = %v, want ErrInvalidVisit", err)
	}
	checkEvents(t, "refused visit", f.audit, event(me, auditevent.CodeUpdate, "", auditevent.OutcomeMinorFailure))
}

func TestStopSharing_RevokesTheHistoryConsentAndAuditsIt(t *testing.T) {
	f := withHistory()
	if err := f.svc.StopSharing(t.Context(), me, clinicA); err != nil {
		t.Fatal(err)
	}
	if h := f.history(); h.ID != "" {
		t.Errorf("an active history consent is left: %+v", h)
	}
	checkEvents(t, "stop", f.audit, event(me, auditevent.CodeUpdate, historyAA, auditevent.OutcomeSuccess))
	g := sharingFixture()
	if err := g.svc.StopSharing(t.Context(), me, clinicA); err != nil {
		t.Errorf("StopSharing() with nothing shared = %v, want nil", err)
	}
	checkEvents(t, "nothing to stop", g.audit, event(me, auditevent.CodeUpdate, "", auditevent.OutcomeSuccess))
	h := sharingFixture()
	if err := h.svc.StopSharing(t.Context(), staff, clinicA); !errors.Is(err, consent.ErrForbidden) {
		t.Errorf("StopSharing() as staff = %v, want ErrForbidden", err)
	}
	checkEvents(t, "staff", h.audit, event(staff, auditevent.CodeUpdate, "", auditevent.OutcomeMinorFailure))
}

func TestMySharing_ListsTheActiveHistoryConsentsAndAuditsEach(t *testing.T) {
	f := withHistory()
	got, err := f.svc.MySharing(t.Context(), me)
	want := []consent.Sharing{{OrganizationID: clinicA, ConsentID: historyAA, VersionID: "2", Shared: []string{visit1}}}
	if err != nil || len(got) != 1 || !sameSharing(got[0], want[0]) {
		t.Fatalf("MySharing() = %+v, %v; want %+v", got, err, want)
	}
	checkEvents(t, "mine", f.audit, event(me, auditevent.CodeSearchType, historyAA, auditevent.OutcomeSuccess))
	g := sharingFixture()
	if got, err := g.svc.MySharing(t.Context(), me); err != nil || got == nil || len(got) != 0 {
		t.Errorf("MySharing() with nothing shared = %#v, %v; want an empty list", got, err)
	}
	checkEvents(t, "none", g.audit, event(me, auditevent.CodeSearchType, "", auditevent.OutcomeSuccess))
	if _, err := g.svc.MySharing(t.Context(), staff); !errors.Is(err, consent.ErrForbidden) {
		t.Errorf("MySharing() as staff = %v, want ErrForbidden", err)
	}
}

// Revoking the care consent revokes the history consent with the clinic in the same transaction, audited.
func TestRevoke_TheCareConsentRevokesTheHistoryConsentToo(t *testing.T) {
	f := withHistory()
	if _, err := f.svc.Revoke(t.Context(), me, consentAA); err != nil {
		t.Fatal(err)
	}
	if h := f.history(); h.ID != "" {
		t.Errorf("an active history consent is left: %+v", h)
	}
	checkEvents(t, "Revoke", f.audit,
		event(me, auditevent.CodeUpdate, historyAA, auditevent.OutcomeSuccess),
		event(me, auditevent.CodeUpdate, consentAA, auditevent.OutcomeSuccess))
}

// A history consent is revoked through /consents too, and enqueues nothing: the clinic's access ends at the commit.
func TestRevoke_AHistoryConsentEndsItAlone(t *testing.T) {
	f := withHistory()
	got, err := f.svc.Revoke(t.Context(), me, historyAA)
	if err != nil || got.Kind != consent.KindHistory || got.Status != consent.StatusInactive {
		t.Fatalf("Revoke() = %+v, %v; want the history consent inactive", got, err)
	}
	if len(f.jobs.enqueued) != 0 || f.repo.consents[0].Status != consent.StatusActive {
		t.Errorf("enqueued %v, care consent %+v; want nothing enqueued and the care consent active", f.jobs.enqueued, f.repo.consents[0])
	}
}

// A history consent is granted with its visits (SetSharing), never empty through /consents.
func TestGrant_RefusesTheHistoryKind(t *testing.T) {
	f := newFixture()
	if !consent.KindHistory.Valid() {
		t.Error("KindHistory.Valid() = false")
	}
	if _, err := f.svc.Grant(t.Context(), me, clinicA, consent.KindHistory); !errors.Is(err, consent.ErrKind) || len(f.repo.calls) != 0 {
		t.Errorf("Grant(history) = %v after %v, want ErrKind before the repository", err, f.repo.calls)
	}
	checkEvents(t, "history", f.audit, event(me, auditevent.CodeCreate, "", auditevent.OutcomeMinorFailure))
}

func sameSharing(a, b consent.Sharing) bool {
	return a.OrganizationID == b.OrganizationID && a.ConsentID == b.ConsentID && a.VersionID == b.VersionID &&
		slices.Equal(a.Shared, b.Shared) && a.Shared != nil
}

func (r *fakeRepository) ActiveHistory(ctx context.Context, patientID, organizationID string) (consent.Consent, error) {
	r.seen(ctx, "ActiveHistory")
	if r.err != nil {
		return consent.Consent{}, r.err
	}
	for _, c := range r.consents {
		if c.Kind == consent.KindHistory && c.Status == consent.StatusActive && c.PatientID == patientID && c.OrganizationID == organizationID && r.visible(c) {
			return c, nil
		}
	}
	return consent.Consent{}, consent.ErrNotFound
}

func (r *fakeRepository) GrantHistory(ctx context.Context, patientID, organizationID string, encounterIDs []string, at time.Time) (consent.Consent, error) {
	r.seen(ctx, "GrantHistory")
	switch {
	case r.err != nil:
		return consent.Consent{}, r.err
	case r.race:
		return consent.Consent{}, consent.ErrActive
	case r.refuseVisits:
		return consent.Consent{}, consent.ErrInvalidVisit
	}
	c := consent.Consent{ID: newID, PatientID: patientID, OrganizationID: organizationID, Kind: consent.KindHistory, Status: consent.StatusActive, DateTime: at, LastUpdated: at, VersionID: 1}
	for _, id := range encounterIDs {
		c.Shared = append(c.Shared, consent.SharedVisit{EncounterID: id, AddedAt: at})
	}
	r.consents = append(r.consents, c)
	return c, nil
}

func (r *fakeRepository) ChangeHistory(ctx context.Context, id string, version int32, add, remove []string, at time.Time) (consent.Consent, error) {
	r.seen(ctx, "ChangeHistory")
	if r.err != nil {
		return consent.Consent{}, r.err
	}
	for i, c := range r.consents {
		if c.ID != id || c.Status != consent.StatusActive || c.VersionID != version || r.race || !r.visible(c) {
			continue
		}
		r.changes = append(r.changes, fmt.Sprintf("ChangeHistory %s v%d +%v -%v", id, version, add, remove))
		kept := slices.DeleteFunc(slices.Clone(c.Shared), func(v consent.SharedVisit) bool { return slices.Contains(remove, v.EncounterID) })
		for _, e := range add {
			kept = append(kept, consent.SharedVisit{EncounterID: e, AddedAt: at})
		}
		c.Shared, c.VersionID, c.LastUpdated = kept, c.VersionID+1, at
		r.consents[i] = c
		return c, nil
	}
	return consent.Consent{}, consent.ErrVersionConflict
}

func (r *fakeRepository) RevokeHistory(ctx context.Context, patientID, organizationID string, at time.Time) (consent.Consent, error) {
	r.seen(ctx, "RevokeHistory")
	c, err := r.revoke(func(c consent.Consent) bool {
		return c.PatientID == patientID && c.OrganizationID == organizationID && c.Kind == consent.KindHistory
	}, at)
	return c, err
}
