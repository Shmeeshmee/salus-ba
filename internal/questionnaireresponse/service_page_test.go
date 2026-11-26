package questionnaireresponse_test

import (
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/questionnaireresponse"
)

const apptC = "e5000000-0000-4000-8000-0000000000c1" // patient A with role A2, at appointment A's start

// A staff search not narrowed to one appointment (a few responses at most) is paged, DefaultCount a page unless the
// count asks for another; a patient's own and the reads inside the app are not, unless a count or a position is given.
func TestSearchPage_PagesAStaffSearchNotNarrowedToOneAppointment(t *testing.T) {
	for _, tt := range []struct {
		name   string
		caller appointment.Caller
		filter questionnaireresponse.Filter
		count  int
		limit  int // asked of the repository: the page and one more, 0 for every match
	}{
		{"the admin, no parameter", admin, questionnaireresponse.Filter{}, 0, appointment.DefaultCount + 1},
		{"the admin, by patient and status", admin, questionnaireresponse.Filter{PatientID: patientA, Status: questionnaireresponse.StatusCompleted}, 0, appointment.DefaultCount + 1},
		{"the admin, a count", admin, questionnaireresponse.Filter{}, 3, 4},
		{"the admin, one appointment", admin, questionnaireresponse.Filter{AppointmentID: apptA}, 0, 0},
		{"the admin, one appointment with a count", admin, questionnaireresponse.Filter{AppointmentID: apptA}, 2, 3},
		{"the practitioner, no parameter", pract, questionnaireresponse.Filter{}, 0, appointment.DefaultCount + 1},
		{"a patient, their own", me, questionnaireresponse.Filter{}, 0, 0},
		{"a patient, with a count", me, questionnaireresponse.Filter{}, 5, 6},
	} {
		f := pagingFixture()
		if _, err := f.svc.SearchPage(t.Context(), tt.caller, tt.filter, tt.count); err != nil {
			t.Fatalf("%s: SearchPage() = %v", tt.name, err)
		}
		if len(f.repo.searched) == 0 || f.repo.searched[0].Limit != tt.limit {
			t.Errorf("%s: searched %+v, want a limit of %d", tt.name, f.repo.searched, tt.limit)
		}
	}
	f := pagingFixture()
	if _, err := f.svc.Search(t.Context(), admin, questionnaireresponse.Filter{}); err != nil || len(f.repo.searched) != 1 || f.repo.searched[0].Limit != 0 {
		t.Errorf("Search(the admin, no parameter) = %v after %+v, want one read of every match: the reads inside the app are not paged", err, f.repo.searched)
	}
}

// pagingFixture is clinic A's twelve responses of appointment A (patient A, role A1), B (patient B, role A1) and C
// (patient A, role A2), some due at one time, in turn in progress and completed: the practitioner of role A1 sees A's
// and B's, the admin each at the name level, a patient their own.
func pagingFixture() *fixture {
	f := newFixture(now)
	f.appts.appts[apptC] = appointment.Appointment{ID: apptC, PatientID: patientA, PractitionerRoleID: roleA2, OrganizationID: clinicA,
		Status: appointment.StatusBooked, Start: start, End: start.Add(30 * time.Minute)}
	for i := range 12 {
		appt := []string{apptA, apptB, apptC}[i%3]
		status := []questionnaireresponse.Status{questionnaireresponse.StatusInProgress, questionnaireresponse.StatusCompleted}[i%2]
		f.seed(appt, status)
		f.repo.responses[i].ID = fmt.Sprintf("d%07d-0000-4000-8000-000000000000", 20-i)
	}
	return f
}

// pages follows the search's pages from the first to the last, as a client follows the next links, and gives what
// they returned, concatenated, and the events of their matches.
func pages(t *testing.T, f *fixture, c appointment.Caller, filter questionnaireresponse.Filter, count int) (all []questionnaireresponse.Seen, events []auditevent.AuditEvent) {
	t.Helper()
	for range 100 {
		before := len(f.audit.events)
		p, err := f.svc.SearchPage(t.Context(), c, filter, count)
		if err != nil {
			t.Fatalf("SearchPage(%+v, %d) = %v", filter, count, err)
		}
		page := f.audit.events[before:]
		if len(page) == 1 && page[0].EntityReference == "" && page[0].Outcome == auditevent.OutcomeSuccess {
			page = nil // an empty page, audited as a search that found none
		}
		all, events = append(all, p.Seen...), append(events, page...)
		if !p.More {
			return all, events
		}
		filter.AfterID = p.Seen[len(p.Seen)-1].Response.ID
	}
	t.Fatal("the pages do not end")
	return nil, nil
}

// The pages of a search, concatenated, are what the search without pages returns, in the same order at the same
// levels, each response audited as it returns it.
func TestSearchPage_ConcatenatedPagesAreTheUnpagedAnswer(t *testing.T) {
	for _, tt := range []struct {
		caller appointment.Caller
		filter questionnaireresponse.Filter
	}{
		{admin, questionnaireresponse.Filter{}}, {pract, questionnaireresponse.Filter{}}, {pract2, questionnaireresponse.Filter{}},
		{admin, questionnaireresponse.Filter{Status: questionnaireresponse.StatusCompleted}}, {me, questionnaireresponse.Filter{}},
	} {
		f := pagingFixture()
		whole, err := f.svc.Search(t.Context(), tt.caller, tt.filter)
		if err != nil || len(whole) == 0 {
			t.Fatalf("Search(%s, %+v) = %d, %v; want some", tt.caller.AccountID, tt.filter, len(whole), err)
		}
		wholeEvents := own(f.audit.events)
		for count := 1; count <= len(whole)+1; count++ {
			f := pagingFixture()
			got, events := pages(t, f, tt.caller, tt.filter, count)
			if !slices.EqualFunc(got, whole, func(x, y questionnaireresponse.Seen) bool { return fmt.Sprint(x) == fmt.Sprint(y) }) {
				t.Errorf("%s %+v by %d:\n got %+v\nwant %+v", tt.caller.AccountID, tt.filter, count, got, whole)
			}
			if events = own(events); !slices.Equal(events, wholeEvents) {
				t.Errorf("%s %+v by %d: the pages' events\n %+v\nwant %+v", tt.caller.AccountID, tt.filter, count, events, wholeEvents)
			}
		}
	}
}

// A position the scope does not show (gone since the page before, or never the caller's) is refused: the search is
// run again from the start rather than skipping or repeating.
func TestSearchPage_RefusesAPositionTheScopeDoesNotShow(t *testing.T) {
	f := pagingFixture()
	if _, err := f.svc.SearchPage(t.Context(), admin, questionnaireresponse.Filter{AfterID: "d9999999-0000-4000-8000-000000000000"}, 2); !errors.Is(err, questionnaireresponse.ErrPosition) {
		t.Errorf("SearchPage(after an unknown response) = %v, want ErrPosition", err)
	}
}
