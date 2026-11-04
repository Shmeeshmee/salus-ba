package slot_test

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/slot"
)

func TestBookable_GivesAFreeSlotOfTheDirectoryWithItsRole(t *testing.T) {
	t.Parallel()
	f := newFixture()
	got, err := f.service(now).Bookable(t.Context(), slotA1)
	want := slot.Bookable{
		Slot:               slot.Slot{ID: slotA1, ScheduleID: scheduleA1, OrganizationID: clinicA, Status: slot.StatusFree, Period: slot.Period{Start: now, End: now.Add(time.Hour)}},
		PractitionerRoleID: roleA1,
	}
	if err != nil || got != want {
		t.Fatalf("Bookable() = %+v, %v;\nwant %+v", got, err, want)
	}
	if want := []string{"SetDirectory", "ByID:" + slotA1, "Read:" + scheduleA1}; !slices.Equal(f.calls, want) {
		t.Errorf("calls %v, want %v in one transaction in the directory", f.calls, want)
	}
}

func TestBookable_IsUnavailableForASlotTheDirectoryDoesNotShow(t *testing.T) {
	t.Parallel()
	f := newFixture()
	if _, err := f.service(now).Bookable(t.Context(), "a6000000-0000-4000-8000-0000000000ff"); !errors.Is(err, slot.ErrUnavailable) {
		t.Errorf("Bookable() of an unknown slot = %v, want ErrUnavailable", err)
	}
	g := newFixture()
	g.schedules.failing = scheduleA1 // the role is no longer active, or its clinic no longer listed
	if _, err := g.service(now).Bookable(t.Context(), slotA1); !errors.Is(err, slot.ErrUnavailable) {
		t.Errorf("Bookable() of a slot whose schedule the directory hides = %v, want ErrUnavailable", err)
	}
	h := newFixture()
	h.repo.err = errors.New("synthetic failure")
	if _, err := h.service(now).Bookable(t.Context(), slotA1); err == nil || errors.Is(err, slot.ErrUnavailable) {
		t.Errorf("Bookable() on a failing repository = %v, want its error", err)
	}
}
