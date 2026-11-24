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

// The desk's visit says whether a reminder will go, and why not: always to a patient with an account (by their own
// channels); to one without, only while the clinic offers SMS (else "sms_off") and the profile has a phone (else
// "no_phone"). It says too whether the patient has an active care consent with the clinic.
func TestDeskDay_SaysWhetherAReminderWillGoAndWhyNot(t *testing.T) {
	t.Parallel()
	for _, sms := range []bool{false, true} {
		app := booked("ab000000-0000-4000-8000-0000000000ab", patientA, roleA1, clinicA, start)
		phone := booked("ac000000-0000-4000-8000-0000000000ac", patientC, roleA1, clinicA, start.Add(30*time.Minute))
		none := booked("ad000000-0000-4000-8000-0000000000ad", patientD, roleA1, clinicA, start.Add(time.Hour))
		f := staffFixture(app, phone, none)
		f.consents.state[patientD+clinicA] = access.Consent{Active: true}                        // given in person
		f.consents.state[patientC+clinicA] = access.Consent{RevokedAt: start.Add(2 * time.Hour)} // withdrawn after the visit
		settings := organization.DefaultAppointmentSettings()
		settings.SMSReminders = sms
		f.rules.settings = map[string]organization.AppointmentSettings{clinicA: settings}
		out, err := f.svc.DeskDay(t.Context(), desk, start)
		if err != nil || len(out.Visits) != 3 {
			t.Fatalf("DeskDay = %+v, %v; want the three visits", out, err)
		}
		type seen struct {
			off      string
			reminder bool
			consent  bool
		}
		got := map[string]seen{}
		for _, v := range out.Visits {
			got[v.Patient.ID] = seen{reminder: v.Reminder, off: v.ReminderOff, consent: v.Patient.ConsentActive}
		}
		want := map[string]seen{
			patientA: {reminder: true, consent: true},
			patientC: {reminder: sms, off: map[bool]string{false: appointment.ReminderOffSMS}[sms]},
			patientD: {off: map[bool]string{false: appointment.ReminderOffSMS, true: appointment.ReminderOffNoPhone}[sms], consent: true},
		}
		for id, w := range want {
			if got[id] != w {
				t.Errorf("SMS option %v: patient %s = %+v, want %+v", sms, id, got[id], w)
			}
		}
	}
}
