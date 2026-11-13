package appointment_test

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
)

// The appointments of the flag tests, in clinic A unless named: patient A's two no-shows 10 and 100 days ago and one
// 200 days ago (outside a 180-day window), another at clinic B, and their booked appointment with role A1 in the
// week; patient B's one no-show and their booked one in the week; patient A's booked one with role A2 in the week.
const (
	flagNoShow1  = "f1000000-0000-4000-8000-000000000001"
	flagNoShow2  = "f1000000-0000-4000-8000-000000000002"
	flagNoShowO  = "f1000000-0000-4000-8000-000000000003"
	flagNoShowB  = "f1000000-0000-4000-8000-000000000004"
	flagNoShowX  = "f1000000-0000-4000-8000-000000000005"
	flagBookedA  = "f1000000-0000-4000-8000-0000000000a1"
	flagBookedB  = "f1000000-0000-4000-8000-0000000000b1"
	flagBookedA2 = "f1000000-0000-4000-8000-0000000000a2"
)

var week = struct{ from, to time.Time }{now, now.Add(7 * 24 * time.Hour)}

func flagsFixture(threshold, days int) *fixture {
	day := 24 * time.Hour
	return withNoShows(newFixture(
		withStatus(booked(flagNoShow1, patientA, roleA1, clinicA, now.Add(-10*day)), appointment.StatusNoShow),
		withStatus(booked(flagNoShow2, patientA, roleA2, clinicA, now.Add(-100*day)), appointment.StatusNoShow),
		withStatus(booked(flagNoShowO, patientA, roleA1, clinicA, now.Add(-200*day)), appointment.StatusNoShow),
		withStatus(booked(flagNoShowX, patientA, roleB, clinicB, now.Add(-10*day)), appointment.StatusNoShow),
		withStatus(booked(flagNoShowB, patientB, roleA1, clinicA, now.Add(-10*day)), appointment.StatusNoShow),
		booked(flagBookedA, patientA, roleA1, clinicA, now.Add(day)),
		booked(flagBookedB, patientB, roleA1, clinicA, now.Add(day+time.Hour)),
		booked(flagBookedA2, patientA, roleA2, clinicA, now.Add(2*day)),
	), threshold, days)
}

func countCalls(calls []string, name string) int {
	n := 0
	for _, c := range calls {
		if c == name {
			n++
		}
	}
	return n
}

// Q7: with the clinic's threshold on, staff get "N no-shows in the last D days" beside each appointment of the range
// they see in full whose patient missed at least the threshold at the clinic in the window, counted on read; each
// flagged appointment's read is audited. Patient B (one no-show at the clinic) is under the threshold; the no-show
// at clinic B and the one before the window do not count.
func TestNoShowFlags_FlagsTheAppointmentsOfPatientsWhoMissedTheThreshold(t *testing.T) {
	f := flagsFixture(2, 180)
	f.consents.state[patientB+clinicA] = access.Consent{Active: true}
	got, err := f.svc.NoShowFlags(t.Context(), admin, week.from, week.to)
	want := []appointment.NoShowFlag{
		{AppointmentID: flagBookedA, NoShows: 2, WindowDays: 180},
		{AppointmentID: flagBookedA2, NoShows: 2, WindowDays: 180},
	}
	if err != nil || !slices.Equal(got, want) {
		t.Errorf("the admin's NoShowFlags() = %+v, %v; want %+v", got, err, want)
	}
	// Every appointment of the range seen in full is audited, flagged or not: that it is not flagged says something
	// about its patient too.
	checkEvents(t, "NoShowFlags", f.audit,
		event(admin, auditevent.CodeSearchType, "Appointment", flagBookedA, auditevent.OutcomeSuccess),
		event(admin, auditevent.CodeSearchType, "Appointment", flagBookedB, auditevent.OutcomeSuccess),
		event(admin, auditevent.CodeSearchType, "Appointment", flagBookedA2, auditevent.OutcomeSuccess))
	if countCalls(f.repo.calls, "Search") != 1 || countCalls(f.repo.calls, "NoShowsSince") != 1 {
		t.Errorf("calls %v; want one search of the range and the no-shows of its patients counted in one query", f.repo.calls)
	}

	// The practitioner of role A1 sees only their own appointments in full.
	f = flagsFixture(2, 180)
	got, err = f.svc.NoShowFlags(t.Context(), pract, week.from, week.to)
	if err != nil || !slices.Equal(got, want[:1]) {
		t.Errorf("the practitioner's NoShowFlags() = %+v, %v; want %+v", got, err, want[:1])
	}
	checkEvents(t, "the practitioner's NoShowFlags", f.audit, event(pract, auditevent.CodeSearchType, "Appointment", flagBookedA, auditevent.OutcomeSuccess))
	// A wider window counts the older no-show too; a threshold of 3 then flags.
	f = flagsFixture(3, 365)
	if got, err = f.svc.NoShowFlags(t.Context(), admin, week.from, week.to); err != nil || len(got) != 2 || got[0].NoShows != 3 || got[0].WindowDays != 365 {
		t.Errorf("a 365-day window: %+v, %v; want 3 no-shows", got, err)
	}
}

// The flag is off by default (threshold 0): nothing is read or flagged, and nothing of a patient audited.
func TestNoShowFlags_NoneWhenTheRuleIsOff(t *testing.T) {
	f := flagsFixture(0, 180)
	got, err := f.svc.NoShowFlags(t.Context(), admin, week.from, week.to)
	if err != nil || len(got) != 0 || slices.Contains(f.repo.calls, "Search") || len(f.audit.events) != 0 {
		t.Errorf("NoShowFlags() = %+v, %v; calls %v, events %v; want none", got, err, f.repo.calls, f.audit.events)
	}
}

// After a revocation the clinic no longer sees the patient's future appointments, so none is flagged; a search
// that flags none is audited once without an entity.
func TestNoShowFlags_NoneOfAPatientWhoRevoked(t *testing.T) {
	f := flagsFixture(1, 180)
	f.consents.state[patientA+clinicA] = access.Consent{RevokedAt: now.Add(-time.Hour)}
	got, err := f.svc.NoShowFlags(t.Context(), admin, week.from, week.to)
	if err != nil || len(got) != 0 {
		t.Errorf("NoShowFlags() = %+v, %v; want none", got, err)
	}
	checkEvents(t, "NoShowFlags", f.audit, event(admin, auditevent.CodeSearchType, "", "", auditevent.OutcomeSuccess))
}

// Only staff: a patient never sees the flag (Q7), refused as not found and audited; a range missing, reversed or
// longer than 62 days is invalid before anything is read.
func TestNoShowFlags_RefusesPatientsAndInvalidRanges(t *testing.T) {
	f := flagsFixture(1, 180)
	if _, err := f.svc.NoShowFlags(t.Context(), me, week.from, week.to); !errors.Is(err, appointment.ErrNotFound) {
		t.Errorf("a patient: %v, want ErrNotFound", err)
	}
	checkEvents(t, "a patient", f.audit, event(me, auditevent.CodeSearchType, "", "", auditevent.OutcomeMinorFailure))
	for name, r := range map[string][2]time.Time{
		"missing":  {{}, week.to},
		"reversed": {week.to, week.from},
		"63 days":  {now, now.Add(63 * 24 * time.Hour)},
	} {
		f := flagsFixture(1, 180)
		if _, err := f.svc.NoShowFlags(t.Context(), admin, r[0], r[1]); !errors.Is(err, appointment.ErrInvalidRange) || len(f.repo.calls) != 0 {
			t.Errorf("%s: %v, calls %v; want ErrInvalidRange", name, err, f.repo.calls)
		}
	}
}
