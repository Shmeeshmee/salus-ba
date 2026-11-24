package appointment_test

import (
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/organization"
)

// docs/clinic-cms R3: a visit the desk books for a patient without an account schedules its reminders as any booking
// does (one per offset of the clinic still ahead, each with its dispatch job); the notifier decides at the dispatch
// whether an SMS goes.
func TestBookForPatient_SchedulesTheRemindersOfAPatientWithoutAnAccount(t *testing.T) {
	t.Parallel()
	f := staffFixture()
	if _, err := f.svc.BookForPatient(t.Context(), desk, appointment.StaffBooking{PatientID: patientC, SlotID: slotFree}); err != nil {
		t.Fatal(err)
	}
	offsets := organization.DefaultAppointmentSettings().ReminderOffsetsMinutes
	if len(offsets) == 0 || len(f.reminders.reminders) != len(offsets) || len(f.jobs.enqueued) != len(offsets) {
		t.Fatalf("reminders %v, jobs %v; want one per offset %v", f.reminders.reminders, f.jobs.enqueued, offsets)
	}
	for _, r := range f.reminders.reminders {
		if r.PatientID != patientC || r.OrganizationID != clinicA || !r.StartAt.Equal(start) {
			t.Errorf("reminder %+v, want patient C's at clinic A", r)
		}
	}
}

// The desk's visit says whether a reminder will go: always to a patient with an account (by their own channels); to
// one without, only while the clinic offers SMS and the profile has a phone.
func TestDeskDay_SaysWhetherAReminderWillGo(t *testing.T) {
	t.Parallel()
	for _, sms := range []bool{false, true} {
		app := booked("ab000000-0000-4000-8000-0000000000ab", patientA, roleA1, clinicA, start)
		phone := booked("ac000000-0000-4000-8000-0000000000ac", patientC, roleA1, clinicA, start.Add(30*time.Minute))
		none := booked("ad000000-0000-4000-8000-0000000000ad", patientD, roleA1, clinicA, start.Add(time.Hour))
		f := staffFixture(app, phone, none)
		f.consents.state[patientD+clinicA] = access.Consent{Active: true} // given in person
		settings := organization.DefaultAppointmentSettings()
		settings.SMSReminders = sms
		f.rules.settings = map[string]organization.AppointmentSettings{clinicA: settings}
		out, err := f.svc.DeskDay(t.Context(), desk, start)
		if err != nil || len(out.Visits) != 3 {
			t.Fatalf("DeskDay = %+v, %v; want the three visits", out, err)
		}
		got := map[string]bool{}
		for _, v := range out.Visits {
			got[v.Patient.ID] = v.Reminder
		}
		if want := map[string]bool{patientA: true, patientC: sms, patientD: false}; got[patientA] != want[patientA] || got[patientC] != want[patientC] || got[patientD] != want[patientD] {
			t.Errorf("SMS option %v: reminders %v, want %v", sms, got, want)
		}
	}
}
