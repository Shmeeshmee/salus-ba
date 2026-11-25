package careplan_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/careplan"
)

// A booked plan in the patient's export is a FHIR R4 CarePlan (intent plan): its period from its first day, and each
// step an activity whose detail is the visit: its service by name, its status from the visit's, its time, and the
// visit it was booked as.
func TestFHIR_WritesABookedPlanWithItsVisits(t *testing.T) {
	t.Parallel()
	day := time.Date(2030, 1, 7, 0, 0, 0, 0, time.UTC)
	booked := time.Date(2030, 1, 2, 9, 0, 0, 0, time.FixedZone("CET", 3600))
	start := time.Date(2030, 1, 7, 9, 0, 0, 0, time.UTC)
	end := start.Add(30 * time.Minute)
	p := careplan.Plan{
		ID: "cp1", OrganizationID: "o1", PatientID: "p1", Status: careplan.StatusActive, Title: "Synthetic plan", StartsOn: &day,
		CreatedAt: booked.Add(-time.Hour), BookedAt: &booked, VersionID: 2, Steps: []careplan.Step{
			{ID: "s1", ServiceID: "sv1", ServiceName: "Synthetic exam", VisitMode: "in-person", AppointmentID: "a1", Position: 1,
				VisitStart: &start, VisitEnd: &end, VisitStatus: "fulfilled"},
			{ID: "s2", ServiceID: "sv2", VisitMode: "video", AppointmentID: "a2", Position: 2, DayOffset: 14,
				VisitStart: &start, VisitEnd: &end, VisitStatus: "booked"},
		},
	}
	got, err := json.Marshal(careplan.FHIR(p))
	want := `{"resourceType":"CarePlan","id":"cp1","meta":{"lastUpdated":"2030-01-02T08:00:00Z","versionId":"2"},` +
		`"status":"active","intent":"plan","title":"Synthetic plan","subject":{"reference":"Patient/p1"},` +
		`"period":{"start":"2030-01-07"},"created":"2030-01-02T07:00:00Z","author":{"reference":"Organization/o1"},"activity":[` +
		`{"detail":{"scheduledPeriod":{"start":"2030-01-07T09:00:00Z","end":"2030-01-07T09:30:00Z"},` +
		`"kind":"Appointment","status":"completed","description":"Synthetic exam"},"outcomeReference":[{"reference":"Appointment/a1"}]},` +
		`{"detail":{"scheduledPeriod":{"start":"2030-01-07T09:00:00Z","end":"2030-01-07T09:30:00Z"},` +
		`"kind":"Appointment","status":"scheduled"},"outcomeReference":[{"reference":"Appointment/a2"}]}]}`
	if err != nil || string(got) != want {
		t.Errorf("%s, %v\nwant %s", got, err, want)
	}
}
