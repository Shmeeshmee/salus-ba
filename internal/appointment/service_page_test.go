package appointment_test

import (
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
)

// A staff search without a closed date range is paged, DefaultCount a page unless the count asks for another; a
// closed range, a patient's own search and the reads inside the app are not, unless a count or a position is given.
func TestSearchPage_PagesAStaffSearchWithoutAClosedDateRange(t *testing.T) {
	week := appointment.Filter{From: start, To: start.Add(7 * 24 * time.Hour)}
	for _, tt := range []struct {
		name   string
		caller appointment.Caller
		filter appointment.Filter
		count  int
		limit  int // asked of the repository: the page and one more, 0 for every match
	}{
		{"the admin, no parameter", admin, appointment.Filter{}, 0, appointment.DefaultCount + 1},
		{"the admin, by patient", admin, appointment.Filter{PatientID: patientA}, 0, appointment.DefaultCount + 1},
		{"the admin, from a date on: one bound is no range", admin, appointment.Filter{From: start}, 0, appointment.DefaultCount + 1},
		{"the admin, up to a date", admin, appointment.Filter{To: start}, 0, appointment.DefaultCount + 1},
		{"the admin, a week", admin, week, 0, 0},
		{"the admin, a week with a count", admin, week, 10, 11},
		{"the admin, a week after a position", admin, appointment.Filter{From: week.From, To: week.To, AfterStart: start, AfterID: apptA}, 0, appointment.DefaultCount + 1},
		{"the admin, a count", admin, appointment.Filter{}, 7, 8},
		{"the practitioner, their own", pract, appointment.Filter{PractitionerID: practA1}, 0, appointment.DefaultCount + 1},
		{"the practitioner, their week", pract, appointment.Filter{PractitionerID: practA1, From: week.From, To: week.To}, 0, 0},
		{"a patient, their own", me, appointment.Filter{}, 0, 0},
		{"a patient, with a count", me, appointment.Filter{}, 5, 6},
	} {
		f := cancelFixture(start)
		if _, err := f.svc.SearchPage(t.Context(), tt.caller, tt.filter, tt.count); err != nil {
			t.Fatalf("%s: SearchPage() = %v", tt.name, err)
		}
		if len(f.repo.searched) == 0 || f.repo.searched[0].Limit != tt.limit {
			t.Errorf("%s: searched %+v, want a limit of %d", tt.name, f.repo.searched, tt.limit)
		}
	}
	f := cancelFixture(start)
	if _, err := f.svc.Search(t.Context(), admin, appointment.Filter{}); err != nil || len(f.repo.searched) != 1 || f.repo.searched[0].Limit != 0 {
		t.Errorf("Search(the admin, no parameter) = %v after %+v, want one read of every match: the reads inside the app are not paged", err, f.repo.searched)
	}
}

// pagingFixture is clinic A's appointments of patient A (active consent) and patient B (revoked: the past ones at the
// name level, the future ones left out), some at the same start, and one of clinic B, out of the admin's scope.
func pagingFixture() *fixture {
	var appts []appointment.Appointment
	for i := range 14 {
		patient, at := patientA, now.Add(time.Duration(i/2-3)*time.Hour) // two at each start, three hours of the past
		if i%3 == 0 {
			patient = patientB
		}
		appts = append(appts, booked(fmt.Sprintf("e%07d-0000-4000-8000-000000000000", 14-i), patient, roleA1, clinicA, at))
	}
	appts = append(appts, booked(apptX, patientA, roleB, clinicB, now))
	f := newFixture(appts...)
	f.consents.state[patientB+clinicA] = access.Consent{RevokedAt: now.Add(-30 * time.Minute)}
	return f
}

// pages follows the search's pages from the first to the last, as a client follows the next links, and gives what
// they returned, concatenated, with the audit events of each page.
func pages(t *testing.T, f *fixture, c appointment.Caller, filter appointment.Filter, count int) (all []appointment.Seen, events [][]auditevent.AuditEvent) {
	t.Helper()
	for range 100 {
		before := len(f.audit.events)
		p, err := f.svc.SearchPage(t.Context(), c, filter, count)
		if err != nil {
			t.Fatalf("SearchPage(%+v, %d) = %v", filter, count, err)
		}
		all, events = append(all, p.Seen...), append(events, slices.Clone(f.audit.events[before:]))
		if !p.More {
			return all, events
		}
		if len(p.Seen) == 0 {
			t.Fatalf("a page with more after it has no match to start the next from")
		}
		last := p.Seen[len(p.Seen)-1].Appointment
		filter.AfterStart, filter.AfterID = last.Start, last.ID
	}
	t.Fatal("the pages do not end")
	return nil, nil
}

// pageEvents is the events of the pages, concatenated, without those of an empty page (one event without an entity,
// as a search that finds none writes): the events of the matches.
func pageEvents(t *testing.T, events [][]auditevent.AuditEvent) []auditevent.AuditEvent {
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

// The pages of a search, concatenated, are what the search without pages returns, each audited as it returned it:
// the same appointments at the same levels with the same names, in the same order, and the same events.
func TestSearchPage_ConcatenatedPagesAreTheUnpagedAnswer(t *testing.T) {
	for _, filter := range []appointment.Filter{{}, {PatientID: patientB}, {Status: appointment.StatusBooked, From: now.Add(-2 * time.Hour)}} {
		f := pagingFixture()
		whole, err := f.svc.Search(t.Context(), admin, filter)
		if err != nil || len(whole) < 2 {
			t.Fatalf("Search(%+v) = %d, %v; want several", filter, len(whole), err)
		}
		levels := map[access.Level]int{}
		for _, s := range whole {
			levels[s.Level]++
		}
		if filter.PatientID == "" && (levels[access.Full] == 0 || levels[access.Name] == 0) {
			t.Fatalf("the fixture's levels %v, want both Full and Name", levels)
		}
		wholeEvents := f.audit.events
		for count := 1; count <= len(whole)+1; count++ {
			f := pagingFixture()
			got, events := pages(t, f, admin, filter, count)
			if !slices.Equal(got, whole) {
				t.Errorf("%+v by %d:\n got %+v\nwant %+v", filter, count, got, whole)
			}
			if entities := pageEvents(t, events); !slices.Equal(entities, wholeEvents) {
				t.Errorf("%+v by %d: the pages' events\n %+v\nwant %+v", filter, count, entities, wholeEvents)
			}
		}
	}
}

// A page holds count matches at most; the appointments the search leaves out (here a patient's of another
// practitioner) are read past, not counted, and never the page's end, from which the next link starts.
func TestSearchPage_ReadsPastWhatTheSearchLeavesOut(t *testing.T) {
	f := cancelFixture(start)
	p, err := f.svc.SearchPage(t.Context(), me, appointment.Filter{PractitionerID: practA2}, 1)
	if err != nil || len(p.Seen) != 1 || p.Seen[0].Appointment.ID != apptA2 || p.More {
		t.Fatalf("SearchPage(practitioner A2, 1) = %+v, %v; want apptA2, the last", p, err)
	}
	if len(f.repo.searched) != 3 || f.repo.searched[1].AfterID == "" {
		t.Errorf("searched %+v, want three reads of two, each after the last read: patient A's other two read past", f.repo.searched)
	}
	checkEvents(t, "the page", f.audit,
		event(me, auditevent.CodeSearchType, "Appointment", apptA2, auditevent.OutcomeSuccess),
		event(me, auditevent.CodeSearchType, "Patient", patientA, auditevent.OutcomeSuccess))

	for _, pract := range []string{practA1, practA2, practB} {
		f := cancelFixture(start)
		whole, err := f.svc.Search(t.Context(), me, appointment.Filter{PractitionerID: pract})
		if err != nil {
			t.Fatal(err)
		}
		wholeEvents := f.audit.events
		f = cancelFixture(start)
		got, events := pages(t, f, me, appointment.Filter{PractitionerID: pract}, 1)
		if !slices.Equal(got, whole) || !slices.Equal(pageEvents(t, events), wholeEvents) {
			t.Errorf("practitioner %s by 1: %+v and %+v, want %+v and %+v", pract, got, events, whole, wholeEvents)
		}
	}
}

// Paging keeps the refusals: a practitioner searching another's appointments is refused and audited as before.
func TestSearchPage_RefusesAsTheSearchDoes(t *testing.T) {
	f := cancelFixture(start)
	if _, err := f.svc.SearchPage(t.Context(), pract, appointment.Filter{PractitionerID: practA2}, 0); !errors.Is(err, appointment.ErrForbidden) {
		t.Errorf("SearchPage(another practitioner) = %v, want ErrForbidden", err)
	}
	checkEvents(t, "refused", f.audit, event(pract, auditevent.CodeSearchType, "", "", auditevent.OutcomeMinorFailure))
}
