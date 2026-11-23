package slot_test

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/slot"
)

// The clinic's own staff see its free times in its own scope, listed or not (the front desk, docs/clinic-cms F3); a
// schedule of another clinic is searched as the public directory shows it.
func TestSearchFor_GivesStaffTheirClinicsFreeSlotsInItsScope(t *testing.T) {
	t.Parallel()
	f := newFixture()
	got, err := f.service(now).SearchFor(t.Context(), clinicA, scheduleA1, now.Add(-time.Hour), now.Add(24*time.Hour))
	if err != nil || len(got) != 1 || !f.repo.from.Equal(now) {
		t.Errorf("SearchFor() = %v, %v from %v; want the clinic's free slots from now on", got, err, f.repo.from)
	}
	if want := []string{"Lookup:" + scheduleA1, "SetOrganization:" + clinicA, "Search:" + scheduleA1}; !slices.Equal(f.calls, want) {
		t.Errorf("calls = %v, want %v", f.calls, want)
	}
	g := newFixture()
	if _, err := g.service(now).SearchFor(t.Context(), clinicB, scheduleA1, now, now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if want := []string{"Lookup:" + scheduleA1, "SetDirectory", "Search:" + scheduleA1}; !slices.Equal(g.calls, want) {
		t.Errorf("another clinic's schedule: calls = %v, want the directory's search %v", g.calls, want)
	}
	h := newFixture()
	if _, err := h.service(now).SearchFor(t.Context(), "", scheduleA1, now, now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if want := []string{"SetDirectory", "Search:" + scheduleA1}; !slices.Equal(h.calls, want) {
		t.Errorf("no clinic: calls = %v, want the directory's search %v", h.calls, want)
	}
}

// A staff booking resolves its slot in the clinic's own scope, listed or not: a free slot of one of its schedules,
// with its role; a busy slot, another clinic's or an unknown one is unavailable.
func TestBookableInClinic_GivesAFreeSlotOfTheClinicWithItsRole(t *testing.T) {
	t.Parallel()
	f := newFixture()
	got, err := f.service(now).BookableInClinic(t.Context(), clinicA, slotA1)
	if err != nil || got.ID != slotA1 || got.PractitionerRoleID != roleA1 {
		t.Fatalf("BookableInClinic() = %+v, %v; want the slot with its role", got, err)
	}
	if want := []string{"SetOrganization:" + clinicA, "ByID:" + slotA1, "Lookup:" + scheduleA1}; !slices.Equal(f.calls, want) {
		t.Errorf("calls %v, want %v in one transaction in the clinic", f.calls, want)
	}
	for name, tt := range map[string]struct {
		org, id string
		busy    bool
	}{
		"another clinic's": {org: clinicB, id: slotA1},
		"an unknown slot":  {org: clinicA, id: "a6000000-0000-4000-8000-0000000000ff"},
		"a busy slot":      {org: clinicA, id: slotA1, busy: true},
	} {
		g := newFixture()
		g.repo.busy = tt.busy
		if _, err := g.service(now).BookableInClinic(t.Context(), tt.org, tt.id); !errors.Is(err, slot.ErrUnavailable) {
			t.Errorf("%s: BookableInClinic() = %v, want ErrUnavailable", name, err)
		}
	}
}
