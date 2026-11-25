package schedule_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/location"
	"github.com/Shmeeshmee/salus/internal/schedule"
)

const (
	siteA1 = "a5000000-0000-4000-8000-0000000000a1"
	siteA2 = "a5000000-0000-4000-8000-0000000000a2"
	siteB1 = "b5000000-0000-4000-8000-0000000000b1"
)

// fakeSites holds the clinics' locations: clinic A's one active, and B's.
type fakeSites struct {
	org  string
	rows []location.Location
}

func newFakeSites() *fakeSites {
	return &fakeSites{rows: []location.Location{
		{ID: siteA1, OrganizationID: clinicA, Name: "Centre", Active: true},
		{ID: siteB1, OrganizationID: clinicB, Name: "Other", Active: true},
	}}
}

func (s *fakeSites) OfClinic(_ context.Context, org string) ([]location.Location, error) {
	s.org = org
	var out []location.Location
	for _, l := range s.rows {
		if l.OrganizationID == org {
			out = append(out, l)
		}
	}
	return out, nil
}

// An in-person rule is at a location of the clinic: the one it names, or the clinic's only active one (U2).
func TestSaveAvailability_AnInPersonRuleIsAtALocationOfTheClinic(t *testing.T) {
	t.Parallel()
	inPerson := func(loc string) schedule.Availability {
		return inRules(schedule.Rule{Day: 1, Start: "08:00", End: "12:00", Mode: schedule.ModeInPerson, LocationID: loc},
			schedule.Rule{Day: 2, Start: "08:00", End: "12:00", Mode: schedule.ModeVideo})
	}
	for name, tc := range map[string]struct {
		want   string
		in     schedule.Availability
		second bool
	}{
		"named":                       {siteA1, inPerson(siteA1), false},
		"named, of two":               {siteA2, inPerson(siteA2), true},
		"none, the clinic's only one": {siteA1, inPerson(""), false},
	} {
		f := newFixture()
		if tc.second {
			f.sites.rows = append(f.sites.rows, location.Location{ID: siteA2, OrganizationID: clinicA, Name: "Annex", Active: true})
		}
		got, err := f.svc.SaveAvailability(t.Context(), adminA, roleA1, tc.in, 0)
		if err != nil || len(got.Rules) != 2 || got.Rules[0].LocationID != tc.want || got.Rules[1].LocationID != "" || f.sites.org != clinicA {
			t.Errorf("%s: %+v, %v; want the in-person rule at %s and the video one nowhere", name, got.Rules, err, tc.want)
		}
	}
}

func TestSaveAvailability_RefusesARuleAtNoLocationOfTheClinic(t *testing.T) {
	t.Parallel()
	inPerson := func(loc string) schedule.Availability {
		return inRules(schedule.Rule{Day: 1, Start: "08:00", End: "12:00", Mode: schedule.ModeInPerson, LocationID: loc})
	}
	for name, tc := range map[string]struct {
		want   error
		edit   func(*fakeSites)
		in     schedule.Availability
		access bool
	}{
		"another clinic's": {schedule.ErrInvalidLocation, nil, inPerson(siteB1), true},
		"an unknown one":   {schedule.ErrInvalidLocation, nil, inPerson("a5000000-0000-4000-8000-0000000000ff"), true},
		"an inactive one":  {schedule.ErrInvalidLocation, func(s *fakeSites) { s.rows[0].Active = false }, inPerson(siteA1), true},
		"none, of two": {schedule.ErrInvalidLocation, func(s *fakeSites) {
			s.rows = append(s.rows, location.Location{ID: siteA2, OrganizationID: clinicA, Active: true})
		}, inPerson(""), true},
		"none, the clinic has none": {schedule.ErrInvalidLocation, func(s *fakeSites) { s.rows = s.rows[1:] }, inPerson(""), true},
		"a video rule at one":       {schedule.ErrInvalidAvailability, nil, inRules(schedule.Rule{Day: 1, Start: "08:00", End: "12:00", Mode: schedule.ModeVideo, LocationID: siteA1}), false},
	} {
		f := newFixture()
		if tc.edit != nil {
			tc.edit(f.sites)
		}
		_, err := f.svc.SaveAvailability(t.Context(), adminA, roleA1, tc.in, 0)
		if !errors.Is(err, tc.want) || len(f.repo.schedules) != 2 {
			t.Errorf("%s: %v (%d schedules); want %v and nothing saved", name, err, len(f.repo.schedules), tc.want)
		}
		refused := len(f.audit.events) == 1 && f.audit.events[0].Outcome == auditevent.OutcomeMinorFailure
		if refused != tc.access {
			t.Errorf("%s: audit %+v, want a refusal audited: %v", name, f.audit.events, tc.access)
		}
	}
}

// inRules is an edit of 30-minute slots with rs.
func inRules(rs ...schedule.Rule) schedule.Availability {
	return schedule.Availability{SlotMinutes: 30, Rules: rs}
}
