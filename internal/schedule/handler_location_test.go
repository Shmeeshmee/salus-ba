package schedule_test

import (
	"bytes"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/schedule"
)

// An in-person rule's location travels in both directions as locationId (docs/clinic-cms/WAVE6.md LC1).
func TestHTTPHandler_TakesAndGivesAnInPersonRulesLocation(t *testing.T) {
	t.Parallel()
	out := schedule.Schedule{ID: scheduleA2, PractitionerRoleID: roleA1, SlotMinutes: 30, VersionID: 2, Rules: []schedule.Rule{
		{Day: 1, Start: "09:00", End: "12:00", Mode: schedule.ModeInPerson, LocationID: siteA1},
		{Day: 2, Start: "09:00", End: "12:00", Mode: schedule.ModeVideo},
	}}
	svc := &stubAvailability{out: out}
	body := `{"slotMinutes":30,"availability":[{"day":1,"start":"09:00","end":"12:00","mode":"in-person","locationId":"` + siteA1 + `"}]}`
	rec := serve(t, svc, true, &bytes.Buffer{}, http.MethodPut, `W/"1"`, body)
	want := []schedule.Rule{{Day: 1, Start: "09:00", End: "12:00", Mode: schedule.ModeInPerson, LocationID: siteA1}}
	if rec.Code != http.StatusOK || !slices.Equal(svc.in.Rules, want) {
		t.Fatalf("status %d, rules %+v; want 200 and %+v", rec.Code, svc.in.Rules, want)
	}
	got := rec.Body.String()
	if !strings.Contains(got, `"mode":"in-person","locationId":"`+siteA1+`"`) || strings.Contains(got, "location_id") ||
		!strings.Contains(got, `{"start":"09:00","end":"12:00","mode":"video","day":2}`) {
		t.Errorf("body %s; want locationId on the in-person rule only", got)
	}
}

func TestHTTPHandler_ARuleAtNoLocationOfTheClinicIsInvalidLocation(t *testing.T) {
	t.Parallel()
	svc := &stubAvailability{err: schedule.ErrInvalidLocation}
	rec := serve(t, svc, true, &bytes.Buffer{}, http.MethodPut, `W/"1"`, `{"slotMinutes":30,"availability":[]}`)
	if rec.Code != http.StatusBadRequest || strings.TrimSpace(rec.Body.String()) != `{"error":"invalid_location"}` {
		t.Errorf("%d %s, want 400 invalid_location", rec.Code, rec.Body)
	}
}
