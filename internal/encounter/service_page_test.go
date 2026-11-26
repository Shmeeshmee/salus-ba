package encounter_test

import (
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/encounter"
)

// A staff search not narrowed to one appointment is paged, DefaultCount a page unless the count asks for another; one
// appointment's (an encounter at most), a patient's own and the reads inside the app are not, unless a count or a
// position is given.
func TestSearchPage_PagesAStaffSearchNotNarrowedToOneAppointment(t *testing.T) {
	for _, tt := range []struct {
		name   string
		caller appointment.Caller
		filter encounter.Filter
		count  int
		limit  int // asked of the repository: the page and one more, 0 for every match
	}{
		{"the admin, no parameter", admin, encounter.Filter{}, 0, appointment.DefaultCount + 1},
		{"the admin, by patient", admin, encounter.Filter{PatientID: patientA}, 0, appointment.DefaultCount + 1},
		{"the admin, a count", admin, encounter.Filter{}, 3, 4},
		{"the admin, after a position", admin, encounter.Filter{AfterStart: start, AfterID: encounterOf(apptA)}, 0, appointment.DefaultCount + 1},
		{"the admin, one appointment", admin, encounter.Filter{AppointmentID: apptA}, 0, 0},
		{"the practitioner, their own", pract, encounter.Filter{PractitionerID: practA1}, 0, appointment.DefaultCount + 1},
		{"the practitioner, one appointment", pract, encounter.Filter{AppointmentID: apptA}, 0, 0},
		{"a patient, their own", me, encounter.Filter{}, 0, 0},
		{"a patient, with a count", me, encounter.Filter{}, 5, 6},
	} {
		f := started()
		if _, err := f.svc.SearchPage(t.Context(), tt.caller, tt.filter, tt.count); err != nil {
			t.Fatalf("%s: SearchPage() = %v", tt.name, err)
		}
		if len(f.repo.searched) == 0 || f.repo.searched[0].Limit != tt.limit {
			t.Errorf("%s: searched %+v, want a limit of %d", tt.name, f.repo.searched, tt.limit)
		}
	}
	f := started()
	if _, err := f.svc.Search(t.Context(), admin, encounter.Filter{}); err != nil || len(f.repo.searched) != 1 || f.repo.searched[0].Limit != 0 {
		t.Errorf("Search(the admin, no parameter) = %v after %+v, want one read of every match: the reads inside the app are not paged", err, f.repo.searched)
	}
}

// pagingFixture is started() with twelve more of clinic A's encounters, two at each start: patient A's (with the
// consent: the admin sees them, practitioner A1 those of role A1) and patient B's (without: nobody on staff sees them).
func pagingFixture() *fixture {
	f := started()
	for i := range 12 {
		appt := fmt.Sprintf("e%07d-0000-4000-8000-000000000000", 20-i)
		patient, role, practitioner, at := patientA, roleA1, practA1, start.Add(time.Duration(i/2-3)*time.Hour)
		if i%3 == 0 {
			patient = patientB
		}
		if i%4 == 1 {
			role, practitioner = roleA2, practA2
		}
		a := bookedAt(appt, patient, role, at)
		f.appts.appts[appt] = a
		if patient == patientA {
			f.appts.attendance[adminAcc+appt] = appointment.Attendance{Appointment: a, PractitionerID: practitioner, Level: access.Full}
			f.appts.attendance[practAcc+appt] = appointment.Attendance{Appointment: a, PractitionerID: practitioner, Level: access.Full, Practitioner: role == roleA1}
			f.appts.attendance[accountA+appt] = appointment.Attendance{Appointment: a, PractitionerID: practitioner, Level: access.Full, Patient: true}
		}
		f.repo.encounters = append(f.repo.encounters, encounter.Encounter{
			ID: encounterOf(appt), AppointmentID: appt, OrganizationID: clinicA, PatientID: patient, PractitionerRoleID: role,
			Status: encounter.StatusFinished, PeriodStart: at, LastUpdated: at, VersionID: 1,
		})
	}
	return f
}

// pages follows the search's pages from the first to the last, as a client follows the next links, and gives what
// they returned, concatenated, with the audit events of each page.
func pages(t *testing.T, f *fixture, c appointment.Caller, filter encounter.Filter, count int) (all []encounter.Encounter, events [][]auditevent.AuditEvent) {
	t.Helper()
	for range 100 {
		before := len(f.audit.events)
		p, err := f.svc.SearchPage(t.Context(), c, filter, count)
		if err != nil {
			t.Fatalf("SearchPage(%+v, %d) = %v", filter, count, err)
		}
		all, events = append(all, p.Encounters...), append(events, slices.Clone(f.audit.events[before:]))
		if !p.More {
			return all, events
		}
		last := p.Encounters[len(p.Encounters)-1]
		filter.AfterStart, filter.AfterID = last.PeriodStart, last.ID
	}
	t.Fatal("the pages do not end")
	return nil, nil
}

// matchEvents is the events of the pages, concatenated, without those of an empty page (one event without an entity,
// as a search that finds none writes): the events of the matches.
func matchEvents(t *testing.T, events [][]auditevent.AuditEvent) []auditevent.AuditEvent {
	t.Helper()
	var out []auditevent.AuditEvent
	for i, page := range events {
		if len(page) == 1 && page[0].EntityReference == "" && page[0].Outcome == auditevent.OutcomeSuccess {
			continue
		}
		for _, e := range page {
			if e.EntityReference == "" {
				t.Errorf("page %d: an event without an entity beside others: %+v", i, page)
			}
		}
		out = append(out, page...)
	}
	return out
}

// The pages of a search, concatenated, are what the search without pages returns, in the same order, each encounter
// audited as it returns it; an empty page writes the one event of a search that finds none.
func TestSearchPage_ConcatenatedPagesAreTheUnpagedAnswer(t *testing.T) {
	for _, tt := range []struct {
		caller appointment.Caller
		filter encounter.Filter
	}{
		{admin, encounter.Filter{}}, {pract, encounter.Filter{}}, {admin, encounter.Filter{PractitionerID: practA2}},
		{admin, encounter.Filter{PatientID: patientB}}, {me, encounter.Filter{PractitionerID: practA1}},
	} {
		f := pagingFixture()
		whole, err := f.svc.Search(t.Context(), tt.caller, tt.filter)
		if err != nil {
			t.Fatal(err)
		}
		var wholeEvents []auditevent.AuditEvent // a search that finds none writes one event without an entity
		for _, e := range f.audit.events {
			if e.EntityReference != "" {
				wholeEvents = append(wholeEvents, e)
			}
		}
		for count := 1; count <= len(whole)+1; count++ {
			f := pagingFixture()
			got, events := pages(t, f, tt.caller, tt.filter, count)
			if !slices.Equal(got, whole) {
				t.Errorf("%s %+v by %d:\n got %v\nwant %v", tt.caller.AccountID, tt.filter, count, ids(got), ids(whole))
			}
			if matched := matchEvents(t, events); !slices.Equal(matched, wholeEvents) {
				t.Errorf("%s %+v by %d: the pages' events\n %+v\nwant %+v", tt.caller.AccountID, tt.filter, count, matched, wholeEvents)
			}
		}
	}
}

// Paging keeps the refusals: a practitioner naming another is refused and audited as before.
func TestSearchPage_RefusesAsTheSearchDoes(t *testing.T) {
	f := started()
	if _, err := f.svc.SearchPage(t.Context(), pract, encounter.Filter{PractitionerID: practA2}, 0); !errors.Is(err, encounter.ErrForbidden) {
		t.Errorf("SearchPage(another practitioner) = %v, want ErrForbidden", err)
	}
	checkEvents(t, "refused", f.audit, event(pract, auditevent.CodeSearchType, "", auditevent.OutcomeMinorFailure))
}
