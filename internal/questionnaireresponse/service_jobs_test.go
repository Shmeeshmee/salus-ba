package questionnaireresponse_test

import (
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/questionnaireresponse"
)

const (
	agentRequest = "questionnaire_default_request"
	agentDiscard = "questionnaire_responses_discard"
	apptPast     = "e5000000-0000-4000-8000-0000000000a5" // patient A with role A1, started an hour ago
)

// jobEvent is the audit event a job writes: interaction on QuestionnaireResponse/id, the job as agent, for
// operations in clinic A.
func jobEvent(agent, interaction, id string) auditevent.AuditEvent {
	action := map[string]auditevent.Action{auditevent.CodeCreate: auditevent.ActionCreate, auditevent.CodeDelete: auditevent.ActionDelete}[interaction]
	return auditevent.AuditEvent{
		Type:            auditevent.Coding{System: auditevent.SystemAuditEventType, Code: auditevent.CodeRest},
		Subtype:         auditevent.Coding{System: auditevent.SystemRestfulInteraction, Code: interaction},
		Purpose:         auditevent.Coding{System: auditevent.SystemActReason, Code: auditevent.CodeHOPERAT},
		Action:          action,
		Outcome:         auditevent.OutcomeSuccess,
		AgentName:       agent,
		OrganizationID:  clinicA,
		EntityReference: "QuestionnaireResponse/" + id,
	}
}

func TestRequestDefault_RequestsTheActiveVersionOfTheRolesDefaultOnce(t *testing.T) {
	f := newFixture(now)
	f.questionnaires.defaults[roleA1] = qActive

	for run := range 2 { // the second run is River's retry: one row
		if err := f.svc.RequestDefault(t.Context(), apptA, clinicA); err != nil {
			t.Fatalf("run %d: RequestDefault() error = %v", run, err)
		}
	}
	if len(f.repo.responses) != 1 {
		t.Fatalf("%d responses after two runs, want one: %+v", len(f.repo.responses), f.repo.responses)
	}
	got := f.repo.responses[0]
	want := questionnaireresponse.Response{
		ID: got.ID, OrganizationID: clinicA, AppointmentID: apptA, PatientID: patientA, PractitionerRoleID: roleA1,
		QuestionnaireID: qActive, SeriesID: seriesA, Source: questionnaireresponse.SourceDefault, Status: questionnaireresponse.StatusInProgress,
		DueAt: start, LastUpdated: now, VersionID: 1,
	}
	if !equalResponse(got, want) {
		t.Errorf("the request = %+v,\nwant %+v", got, want)
	}
	if f.repo.clinic != clinicA || !slices.Equal(f.questionnaires.agents, []string{agentRequest, agentRequest}) {
		t.Errorf("in clinic %q, the default asked by %v; want clinic A's scope, the job as agent", f.repo.clinic, f.questionnaires.agents)
	}
	if got := own(f.audit.events); !slices.Equal(got, []auditevent.AuditEvent{jobEvent(agentRequest, auditevent.CodeCreate, want.ID)}) {
		t.Errorf("audit events %+v, want the one create by the job", got)
	}
}

func TestRequestDefault_DoesNothingWhenThereIsNothingToRequest(t *testing.T) {
	for _, tt := range []struct {
		at    time.Time
		setup func(f *fixture)
		name  string
		appt  string
		org   string
	}{
		{name: "a role without a default", appt: apptA, org: clinicA, at: now, setup: func(f *fixture) { delete(f.questionnaires.defaults, roleA1) }},
		{name: "the default's series retired", appt: apptA, org: clinicA, at: now, setup: func(f *fixture) { f.questionnaires.defaults[roleA1] = qRetired }},
		{name: "a draft only", appt: apptA, org: clinicA, at: now, setup: func(f *fixture) { f.questionnaires.defaults[roleA1] = qDraft }},
		{name: "a default of another clinic", appt: apptA, org: clinicA, at: now, setup: func(f *fixture) { f.questionnaires.defaults[roleA1] = qOther }},
		{name: "a cancelled appointment", appt: apptA, org: clinicA, at: now, setup: func(f *fixture) {
			a := f.appts.appts[apptA]
			a.Status = appointment.StatusCancelled
			f.appts.appts[apptA] = a
		}},
		{name: "an appointment that has started", appt: apptA, org: clinicA, at: start},
		{name: "an unknown appointment", appt: unknownID, org: clinicA, at: now},
		{name: "an appointment of another clinic", appt: apptA, org: clinicB, at: now},
		{name: "a consent revoked since the booking", appt: apptA, org: clinicA, at: now, setup: func(f *fixture) {
			f.appts.consents[patientA+clinicA] = access.Consent{RevokedAt: revokedAt}
		}},
	} {
		f := newFixture(tt.at)
		f.questionnaires.defaults[roleA1] = qActive
		if tt.setup != nil {
			tt.setup(f)
		}
		if err := f.svc.RequestDefault(t.Context(), tt.appt, tt.org); err != nil {
			t.Errorf("%s: RequestDefault() error = %v, want none", tt.name, err)
		}
		if len(f.repo.responses) != 0 || len(own(f.audit.events)) != 0 {
			t.Errorf("%s: responses %+v, events %+v; want neither", tt.name, f.repo.responses, own(f.audit.events))
		}
	}
}

// The practitioner added the default's questionnaire before the job ran: the job adds no second request.
func TestRequestDefault_AddsNothingWhenTheSeriesIsAlreadyRequested(t *testing.T) {
	f := newFixture(now)
	f.questionnaires.defaults[roleA1] = qActive
	id := f.seed(apptA, questionnaireresponse.StatusInProgress)
	if err := f.svc.RequestDefault(t.Context(), apptA, clinicA); err != nil {
		t.Fatal(err)
	}
	if len(f.repo.responses) != 1 || f.repo.responses[0].ID != id || len(own(f.audit.events)) != 0 {
		t.Errorf("responses %+v, events %+v; want the practitioner's only, nothing audited", f.repo.responses, own(f.audit.events))
	}
}

func TestRequestDefault_FailsOnABackendErrorAndKeepsNothing(t *testing.T) {
	for name, breakIt := range map[string]func(f *fixture){
		"appointments":   func(f *fixture) { f.appts.err = errBackend },
		"questionnaires": func(f *fixture) { f.questionnaires.err = errBackend },
		"repository":     func(f *fixture) { f.repo.err = errBackend },
	} {
		f := newFixture(now)
		f.questionnaires.defaults[roleA1] = qActive
		breakIt(f)
		err := f.svc.RequestDefault(t.Context(), apptA, clinicA)
		if !errors.Is(err, errBackend) || strings.Contains(err.Error(), apptA) {
			t.Errorf("%s: RequestDefault() error = %v, want the backend's without the id", name, err)
		}
		if len(f.repo.responses) != 0 || len(f.audit.events) != 0 {
			t.Errorf("%s: responses %+v, events %+v after the rollback; want none", name, f.repo.responses, f.audit.events)
		}
	}
}

// cancel cancels the appointment id in the fake.
func (f *fixture) cancel(id string) {
	a := f.appts.appts[id]
	a.Status = appointment.StatusCancelled
	f.appts.appts[id] = a
}

func TestDiscard_DeletesEveryResponseOfTheCancelledAppointmentOnce(t *testing.T) {
	f := newFixture(now)
	ids := []string{
		f.seed(apptA, questionnaireresponse.StatusInProgress), f.seed(apptA, questionnaireresponse.StatusCompleted),
		f.seed(apptA, questionnaireresponse.StatusStopped),
	}
	kept := f.seed(apptB, questionnaireresponse.StatusInProgress)
	f.cancel(apptA)

	for run := range 2 { // the retry finds nothing left
		if err := f.svc.Discard(t.Context(), apptA, clinicA); err != nil {
			t.Fatalf("run %d: Discard() error = %v", run, err)
		}
	}
	if len(f.repo.responses) != 1 || f.repo.responses[0].ID != kept {
		t.Errorf("left %+v, want only appointment B's", f.repo.responses)
	}
	if f.repo.clinic != clinicA || f.repo.notice != apptA {
		t.Errorf("in clinic %q with the notice %q, want clinic A's scope with the appointment's notice", f.repo.clinic, f.repo.notice)
	}
	want := make([]auditevent.AuditEvent, 0, len(ids))
	for _, id := range ids {
		want = append(want, jobEvent(agentDiscard, auditevent.CodeDelete, id))
	}
	if !slices.Equal(f.audit.events, want) {
		t.Errorf("audit events %+v,\nwant %+v", f.audit.events, want)
	}
}

// The revocation cancels the future appointment, whose responses go; the past one's completed response stays
// (D9), and a discard of an appointment not cancelled does nothing.
func TestDiscard_AfterARevocationDeletesTheFutureAppointmentsResponsesAndKeepsThePastOnes(t *testing.T) {
	f := newFixture(now)
	f.appts.appts[apptPast] = appointment.Appointment{ID: apptPast, PatientID: patientA, PractitionerRoleID: roleA1, OrganizationID: clinicA,
		Status: appointment.StatusBooked, Start: revokedAt.Add(-time.Hour), End: revokedAt.Add(-30 * time.Minute)}
	future := f.seed(apptA, questionnaireresponse.StatusInProgress)
	past := f.seed(apptPast, questionnaireresponse.StatusCompleted)
	f.appts.consents[patientA+clinicA] = access.Consent{RevokedAt: revokedAt}
	f.cancel(apptA)

	for _, appt := range []string{apptA, apptPast} {
		if err := f.svc.Discard(t.Context(), appt, clinicA); err != nil {
			t.Fatalf("Discard(%s) error = %v", appt, err)
		}
	}
	if len(f.repo.responses) != 1 || f.repo.responses[0].ID != past {
		t.Errorf("left %+v, want only the past appointment's completed response", f.repo.responses)
	}
	if want := []auditevent.AuditEvent{jobEvent(agentDiscard, auditevent.CodeDelete, future)}; !slices.Equal(f.audit.events, want) {
		t.Errorf("audit events %+v, want %+v", f.audit.events, want)
	}
}

func TestDiscard_DoesNothingForAnAppointmentNotCancelled(t *testing.T) {
	for _, tt := range []struct{ name, appt, org string }{
		{name: "a booked appointment", appt: apptA, org: clinicA},
		{name: "an unknown appointment", appt: unknownID, org: clinicA},
		{name: "an appointment of another clinic", appt: apptA, org: clinicB},
	} {
		f := newFixture(now)
		f.seed(apptA, questionnaireresponse.StatusInProgress)
		if tt.org == clinicB {
			f.cancel(apptA)
		}
		if err := f.svc.Discard(t.Context(), tt.appt, tt.org); err != nil {
			t.Errorf("%s: Discard() error = %v", tt.name, err)
		}
		if len(f.repo.responses) != 1 || len(f.audit.events) != 0 {
			t.Errorf("%s: left %d responses, events %+v; want the response kept, nothing audited", tt.name, len(f.repo.responses), f.audit.events)
		}
	}
}

func TestDiscard_FailsOnABackendErrorAndKeepsEverything(t *testing.T) {
	for name, breakIt := range map[string]func(f *fixture){
		"appointments": func(f *fixture) { f.appts.err = errBackend },
		"repository":   func(f *fixture) { f.repo.err = errBackend },
	} {
		f := newFixture(now)
		f.seed(apptA, questionnaireresponse.StatusInProgress)
		f.cancel(apptA)
		breakIt(f)
		err := f.svc.Discard(t.Context(), apptA, clinicA)
		if !errors.Is(err, errBackend) || strings.Contains(err.Error(), apptA) {
			t.Errorf("%s: Discard() error = %v, want the backend's without the id", name, err)
		}
		if len(f.repo.responses) != 1 || len(f.audit.events) != 0 {
			t.Errorf("%s: left %d responses, events %+v; want everything kept", name, len(f.repo.responses), f.audit.events)
		}
	}
}

// equalResponse compares two responses, times by instant.
func equalResponse(a, b questionnaireresponse.Response) bool {
	if !a.DueAt.Equal(b.DueAt) || !a.LastUpdated.Equal(b.LastUpdated) || !a.AuthoredAt.Equal(b.AuthoredAt) {
		return false
	}
	a.DueAt, a.LastUpdated, a.AuthoredAt = b.DueAt, b.LastUpdated, b.AuthoredAt
	return a.ID == b.ID && a.OrganizationID == b.OrganizationID && a.AppointmentID == b.AppointmentID && a.PatientID == b.PatientID &&
		a.PractitionerRoleID == b.PractitionerRoleID && a.QuestionnaireID == b.QuestionnaireID && a.SeriesID == b.SeriesID &&
		a.Source == b.Source && a.Status == b.Status && len(a.Answers) == len(b.Answers) && a.VersionID == b.VersionID
}
