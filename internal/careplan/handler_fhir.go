package careplan

import (
	"time"

	"github.com/Shmeeshmee/salus/pkg/fhir"
)

// A plan is not served over FHIR; it is written as a FHIR R4 CarePlan only inside the patient's own export
// (record.json, docs/clinic-cms/WAVE5.md D7). Each step is an activity whose detail is its visit (kind Appointment):
// the service by its price-list name, the visit's time and status, and the visit as the activity's outcome (an
// activity holds a reference or a detail, not both: cpl-3).

type fhirPlan struct {
	ResourceType string         `json:"resourceType"`
	ID           string         `json:"id"`
	Meta         fhir.Meta      `json:"meta"`
	Status       Status         `json:"status"`
	Intent       string         `json:"intent"`
	Title        string         `json:"title"`
	Subject      fhir.Reference `json:"subject"`
	Period       *fhirPeriod    `json:"period,omitempty"`
	Created      time.Time      `json:"created"`
	Author       fhir.Reference `json:"author"`
	Activity     []fhirActivity `json:"activity,omitempty"`
}

type fhirPeriod struct {
	Start string `json:"start,omitempty"`
	End   string `json:"end,omitempty"`
}

type fhirActivity struct {
	Detail           fhirDetail       `json:"detail"`
	OutcomeReference []fhir.Reference `json:"outcomeReference,omitempty"`
}

type fhirDetail struct {
	ScheduledPeriod *fhirPeriod `json:"scheduledPeriod,omitempty"`
	Kind            string      `json:"kind"`
	Status          string      `json:"status"`
	Description     string      `json:"description,omitempty"`
}

// FHIR is the plan p as a FHIR R4 CarePlan, last updated at its latest change (Plan.LastUpdated).
func FHIR(p Plan) any {
	out := fhirPlan{
		ResourceType: "CarePlan", ID: p.ID, Meta: fhir.NewMeta(int32(p.VersionID), p.LastUpdated()), Status: p.Status, Intent: "plan", //nolint:gosec // a version is small
		Title: p.Title, Subject: fhir.Reference{Reference: "Patient/" + p.PatientID}, Created: p.CreatedAt.UTC(),
		Author: fhir.Reference{Reference: "Organization/" + p.OrganizationID},
	}
	if p.StartsOn != nil {
		out.Period = &fhirPeriod{Start: p.StartsOn.Format(time.DateOnly)}
	}
	for _, s := range p.Steps {
		a := fhirActivity{Detail: fhirDetail{Kind: "Appointment", Status: activityStatus(s), Description: s.ServiceName}}
		if s.AppointmentID != "" {
			a.OutcomeReference = []fhir.Reference{{Reference: "Appointment/" + s.AppointmentID}}
		}
		if s.VisitStart != nil && s.VisitEnd != nil {
			a.Detail.ScheduledPeriod = &fhirPeriod{Start: s.VisitStart.UTC().Format(time.RFC3339), End: s.VisitEnd.UTC().Format(time.RFC3339)}
		}
		out.Activity = append(out.Activity, a)
	}
	return out
}

// activityStatus is the step's CarePlan.activity.detail.status (http://hl7.org/fhir/care-plan-activity-status) from
// its visit's status: not-started before it is booked.
func activityStatus(s Step) string {
	switch s.VisitStatus {
	case "":
		return "not-started"
	case "proposed", "pending", "booked":
		return "scheduled"
	case "arrived", "checked-in":
		return "in-progress"
	case "fulfilled":
		return "completed"
	case "cancelled":
		return "cancelled"
	case "noshow":
		return "stopped"
	}
	return "unknown"
}
