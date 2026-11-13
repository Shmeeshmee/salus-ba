package api_test

import (
	"maps"
	"slices"
	"testing"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/organization"
)

// The appointments epic's shared schemas (docs/plans/appointments.md "Contract") are what the Go types encode, and
// their enums the Go constants.
func TestAppointmentsContract_SchemasMatchTheGoTypes(t *testing.T) {
	t.Parallel()
	s := schemas(t)
	all, required := jsonFields(organization.AppointmentSettings{})
	if got := slices.Sorted(maps.Keys(s["AppointmentSettings"].Properties)); !slices.Equal(got, all) {
		t.Errorf("AppointmentSettings: properties %v, Go encodes %v", got, all)
	}
	if got := slices.Sorted(slices.Values(s["AppointmentSettings"].Required)); !slices.Equal(got, required) {
		t.Errorf("AppointmentSettings: required %v, want %v", got, required)
	}
	for _, f := range []string{"confirmedAt", "confirmedVia", "bookedVia", "rescheduleCount"} {
		if _, ok := s["Appointment"].Properties[f]; !ok {
			t.Errorf("Appointment has no %s", f)
		}
	}
	for name, want := range map[string][]string{
		"AppointmentChangeKind": {string(appointment.ChangeConfirmed), string(appointment.ChangeRescheduled), string(appointment.ChangeStatusSet), string(appointment.ChangeStatusCorrected)},
		"AppointmentChangeVia":  {string(appointment.ViaApp), string(appointment.ViaLink), string(appointment.ViaStaff), string(appointment.ViaSystem)},
	} {
		if got := s[name].Enum; !slices.Equal(got, want) {
			t.Errorf("%s: enum %v, want %v", name, got, want)
		}
	}
	if got := s["Appointment"].Properties["bookedVia"].Enum; !slices.Equal(got, []string{string(appointment.BookedDirect), string(appointment.BookedWaitlist)}) {
		t.Errorf("bookedVia: enum %v", got)
	}
	if got := s["Appointment"].Properties["confirmedVia"].Enum; !slices.Equal(got, []string{string(appointment.ViaApp), string(appointment.ViaLink)}) {
		t.Errorf("confirmedVia: enum %v", got)
	}
	if got := s["AppointmentLinkAction"].Properties["action"].Enum; !slices.Equal(got, []string{"confirm", "cancel"}) {
		t.Errorf("link action enum %v", got)
	}
}
