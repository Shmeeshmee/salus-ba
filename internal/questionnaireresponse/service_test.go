package questionnaireresponse_test

import (
	"errors"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/questionnaire"
	"github.com/Shmeeshmee/salus/internal/questionnaireresponse"
)

// The appointment's practitioner asks the patient to fill in an active version of the clinic before a booked
// appointment that has not started: in progress, no answers, due at the start, with the appointment's patient,
// role and clinic; audited as a create, beside the questionnaire's own audited read.
func TestRequest_ByTheAppointmentsPractitionerStoresItInProgressAndAudits(t *testing.T) {
	f := newFixture(now)

	got, err := f.svc.Request(t.Context(), pract, apptA, qActive)

	want := questionnaireresponse.Response{
		ID: got.ID, OrganizationID: clinicA, AppointmentID: apptA, PatientID: patientA, PractitionerRoleID: roleA1,
		QuestionnaireID: qActive, SeriesID: seriesA, Source: questionnaireresponse.SourceAdded, Status: questionnaireresponse.StatusInProgress,
		DueAt: start, LastUpdated: now, VersionID: 1,
	}
	if err != nil || got.ID == "" || !equal(got, want) {
		t.Fatalf("Request() = %+v, %v;\nwant %+v", got, err, want)
	}
	if len(f.repo.responses) != 1 || !equal(f.repo.responses[0], want) {
		t.Errorf("stored %+v, want the response", f.repo.responses)
	}
	if f.repo.clinic != clinicA || f.repo.patient != "" {
		t.Errorf("scoped to clinic %q, patient %q; want clinic A only", f.repo.clinic, f.repo.patient)
	}
	if !slices.Equal(f.appts.locked, []string{apptA}) {
		t.Errorf("locked %v, want the appointment, against a concurrent cancellation", f.appts.locked)
	}
	if want := []questionnaire.Caller{{AccountID: practAcc, OrganizationID: clinicA, PractitionerID: practA1}}; !slices.Equal(f.questionnaires.callers, want) {
		t.Errorf("read the version as %+v, want %+v", f.questionnaires.callers, want)
	}
	if len(f.audit.events) != 2 || f.audit.events[0].EntityReference != "Questionnaire/"+qActive {
		t.Errorf("audit %+v, want the version's read, then the create", f.audit.events)
	}
	if got, want := own(f.audit.events), []auditevent.AuditEvent{event(pract, auditevent.CodeCreate, got.ID, auditevent.OutcomeSuccess)}; !slices.Equal(got, want) {
		t.Errorf("audited %+v,\nwant %+v", got, want)
	}
}

// equal compares two responses, their answers included.
func equal(a, b questionnaireresponse.Response) bool { return reflect.DeepEqual(a, b) }

// Every refusal stores nothing and writes one create event without an entity, a minor failure, in the
// caller's name.
func TestRequest_RefusesAnyoneButThePractitionerAndAnythingButABookedFutureAppointmentAndAnActiveVersion(t *testing.T) {
	tests := []struct {
		setup  func(f *fixture)
		want   error
		name   string
		appt   string
		q      string
		at     time.Time
		caller appointment.Caller
	}{
		{name: "another practitioner of the clinic", caller: pract2, appt: apptA, q: qActive, want: questionnaireresponse.ErrNotFound},
		{name: "the clinic's admin", caller: admin, appt: apptA, q: qActive, want: questionnaireresponse.ErrNotFound},
		{name: "the patient", caller: me, appt: apptA, q: qActive, want: questionnaireresponse.ErrNotFound},
		{name: "a patient without a profile", caller: noProfile, appt: apptA, q: qActive, want: questionnaireresponse.ErrNotFound},
		{name: "another clinic's admin", caller: adminB, appt: apptA, q: qActive, want: questionnaireresponse.ErrNotFound},
		{name: "an unknown appointment", caller: pract, appt: unknownID, q: qActive, want: questionnaireresponse.ErrNotFound},
		{name: "a cancelled appointment", caller: pract, appt: apptA, q: qActive, want: questionnaireresponse.ErrAppointmentClosed,
			setup: func(f *fixture) { f.setStatus(apptA, appointment.StatusCancelled) }},
		{name: "a cancellation committed before the lock", caller: pract, appt: apptA, q: qActive, want: questionnaireresponse.ErrAppointmentClosed,
			setup: func(f *fixture) { f.appts.committed[apptA] = true }},
		{name: "a fulfilled appointment", caller: pract, appt: apptA, q: qActive, want: questionnaireresponse.ErrAppointmentClosed,
			setup: func(f *fixture) { f.setStatus(apptA, appointment.StatusFulfilled) }},
		{name: "an appointment at its start", caller: pract, appt: apptA, q: qActive, at: start, want: questionnaireresponse.ErrAppointmentStarted},
		{name: "a started appointment", caller: pract, appt: apptA, q: qActive, at: start.Add(time.Minute), want: questionnaireresponse.ErrAppointmentStarted},
		{name: "a past appointment after a revocation", caller: pract, appt: apptA, q: qActive, want: questionnaireresponse.ErrConsentRequired,
			setup: func(f *fixture) { f.revoke(revokedAt.Add(-time.Hour)) }},
		{name: "a future appointment after a revocation", caller: pract, appt: apptA, q: qActive, want: questionnaireresponse.ErrNotFound,
			setup: func(f *fixture) { f.revoke(start) }},
		{name: "a draft", caller: pract, appt: apptA, q: qDraft, want: questionnaireresponse.ErrNotActive},
		{name: "a retired version", caller: pract, appt: apptA, q: qRetired, want: questionnaireresponse.ErrNotActive},
		{name: "another clinic's version", caller: pract, appt: apptA, q: qOther, want: questionnaireresponse.ErrNotFound},
		{name: "an unknown version", caller: pract, appt: apptA, q: unknownID, want: questionnaireresponse.ErrNotFound},
		{name: "twice", caller: pract, appt: apptA, q: qActive, want: questionnaireresponse.ErrAlreadyRequested,
			setup: func(f *fixture) { f.seed(apptA, questionnaireresponse.StatusInProgress) }},
		{name: "after it was submitted", caller: pract, appt: apptA, q: qActive, want: questionnaireresponse.ErrAlreadyRequested,
			setup: func(f *fixture) { f.seed(apptA, questionnaireresponse.StatusCompleted) }},
	}
	for _, tt := range tests {
		at := now
		if !tt.at.IsZero() {
			at = tt.at
		}
		f := newFixture(at)
		if tt.setup != nil {
			tt.setup(f)
		}
		before := len(f.repo.responses)

		_, err := f.svc.Request(t.Context(), tt.caller, tt.appt, tt.q)

		if !errors.Is(err, tt.want) {
			t.Errorf("%s: Request() = %v, want %v", tt.name, err, tt.want)
		}
		if len(f.repo.responses) != before {
			t.Errorf("%s: stored %d responses, want none", tt.name, len(f.repo.responses)-before)
		}
		if got, want := own(f.audit.events), []auditevent.AuditEvent{event(tt.caller, auditevent.CodeCreate, "", auditevent.OutcomeMinorFailure)}; !slices.Equal(got, want) {
			t.Errorf("%s: audited %+v,\nwant %+v", tt.name, got, want)
		}
	}
}

func (f *fixture) setStatus(id string, status appointment.Status) {
	a := f.appts.appts[id]
	a.Status = status
	f.appts.appts[id] = a
}

// A withdrawn request may be requested again.
func TestRequest_AgainAfterAWithdrawal(t *testing.T) {
	f := newFixture(now)
	f.seed(apptA, questionnaireresponse.StatusStopped)

	got, err := f.svc.Request(t.Context(), pract, apptA, qActive)

	if err != nil || got.Status != questionnaireresponse.StatusInProgress || len(f.repo.responses) != 2 {
		t.Errorf("Request() = %+v, %v with %d stored; want a second, live request", got, err, len(f.repo.responses))
	}
}

func TestRequest_FailsWithTheBackendAndKeepsNothing(t *testing.T) {
	for name, setup := range map[string]func(f *fixture){
		"the repository":   func(f *fixture) { f.repo.err = errBackend },
		"the appointments": func(f *fixture) { f.appts.err = errBackend },
	} {
		f := newFixture(now)
		setup(f)
		if _, err := f.svc.Request(t.Context(), pract, apptA, qActive); !errors.Is(err, errBackend) {
			t.Errorf("%s: Request() = %v, want the backend's error", name, err)
		}
		if len(f.repo.responses) != 0 || len(f.audit.events) != 0 {
			t.Errorf("%s: kept %d responses and %d events, want none (rolled back)", name, len(f.repo.responses), len(f.audit.events))
		}
	}
}

// The appointment's practitioner withdraws a request while it is not submitted: stopped, its answers cleared
// at once, audited as an update.
func TestWithdraw_ByTheAppointmentsPractitionerStopsItAndClearsTheAnswers(t *testing.T) {
	f := newFixture(now)
	id := f.seed(apptA, questionnaireresponse.StatusInProgress)

	got, err := f.svc.Withdraw(t.Context(), pract, id)

	if err != nil || got.Status != questionnaireresponse.StatusStopped || len(got.Answers) != 0 || got.VersionID != 4 || !got.LastUpdated.Equal(now) {
		t.Fatalf("Withdraw() = %+v, %v; want it stopped at now, without answers, at version 4", got, err)
	}
	if stored := f.repo.responses[0]; stored.Status != questionnaireresponse.StatusStopped || len(stored.Answers) != 0 {
		t.Errorf("stored %+v, want it stopped without answers", stored)
	}
	if !slices.Equal(f.repo.locked, []string{id}) {
		t.Errorf("locked %v, want the response", f.repo.locked)
	}
	if want := []auditevent.AuditEvent{event(pract, auditevent.CodeUpdate, id, auditevent.OutcomeSuccess)}; !slices.Equal(f.audit.events, want) {
		t.Errorf("audited %+v,\nwant %+v", f.audit.events, want)
	}
}

// Every refusal changes nothing and is audited as an update that failed: with the entity when the caller may
// see the response, without it when its existence is not disclosed.
func TestWithdraw_RefusesAnyoneButThePractitionerAndASubmittedOrWithdrawnResponse(t *testing.T) {
	tests := []struct {
		want    error
		name    string
		status  questionnaireresponse.Status
		caller  appointment.Caller
		unknown bool
		entity  bool
	}{
		{name: "submitted", caller: pract, status: questionnaireresponse.StatusCompleted, want: questionnaireresponse.ErrSubmitted, entity: true},
		{name: "withdrawn", caller: pract, status: questionnaireresponse.StatusStopped, want: questionnaireresponse.ErrWithdrawn, entity: true},
		{name: "the clinic's admin", caller: admin, status: questionnaireresponse.StatusInProgress, want: questionnaireresponse.ErrForbidden, entity: true},
		{name: "the patient", caller: me, status: questionnaireresponse.StatusInProgress, want: questionnaireresponse.ErrForbidden, entity: true},
		{name: "another practitioner", caller: pract2, status: questionnaireresponse.StatusInProgress, want: questionnaireresponse.ErrNotFound},
		{name: "another patient", caller: other, status: questionnaireresponse.StatusInProgress, want: questionnaireresponse.ErrNotFound},
		{name: "a patient without a profile", caller: noProfile, status: questionnaireresponse.StatusInProgress, want: questionnaireresponse.ErrNotFound},
		{name: "another clinic's admin", caller: adminB, status: questionnaireresponse.StatusInProgress, want: questionnaireresponse.ErrNotFound},
		{name: "an unknown response", caller: pract, status: questionnaireresponse.StatusInProgress, unknown: true, want: questionnaireresponse.ErrNotFound},
	}
	for _, tt := range tests {
		f := newFixture(now)
		id := f.seed(apptA, tt.status)
		before := f.repo.responses[0]
		if tt.unknown {
			id = unknownID
		}

		_, err := f.svc.Withdraw(t.Context(), tt.caller, id)

		if !errors.Is(err, tt.want) {
			t.Errorf("%s: Withdraw() = %v, want %v", tt.name, err, tt.want)
		}
		if !equal(f.repo.responses[0], before) {
			t.Errorf("%s: stored %+v, want it unchanged", tt.name, f.repo.responses[0])
		}
		entity := ""
		if tt.entity {
			entity = id
		}
		if want := []auditevent.AuditEvent{event(tt.caller, auditevent.CodeUpdate, entity, auditevent.OutcomeMinorFailure)}; !slices.Equal(f.audit.events, want) {
			t.Errorf("%s: audited %+v,\nwant %+v", tt.name, f.audit.events, want)
		}
	}
}

// After a revocation the clinic still sees the submitted response of a past appointment (D9) but may no
// longer change it: a withdrawal is forbidden, audited with the entity, and not a 404.
func TestWithdraw_AfterARevocationIsForbiddenToTheClinicThatSeesIt(t *testing.T) {
	for _, c := range []appointment.Caller{pract, admin} {
		f := newFixture(now)
		f.revoke(revokedAt.Add(-time.Hour))
		id := f.seed(apptA, questionnaireresponse.StatusCompleted)
		before := f.repo.responses[0]

		_, err := f.svc.Withdraw(t.Context(), c, id)

		if !errors.Is(err, questionnaireresponse.ErrForbidden) {
			t.Errorf("Withdraw() by %s after a revocation = %v, want ErrForbidden", c.AccountID, err)
		}
		if !equal(f.repo.responses[0], before) || len(f.repo.locked) != 0 {
			t.Errorf("stored %+v, locked %v; want it unchanged and unlocked", f.repo.responses[0], f.repo.locked)
		}
		if want := []auditevent.AuditEvent{event(c, auditevent.CodeUpdate, id, auditevent.OutcomeMinorFailure)}; !slices.Equal(f.audit.events, want) {
			t.Errorf("audited %+v,\nwant %+v", f.audit.events, want)
		}
	}
}

func TestWithdraw_FailsWithTheBackend(t *testing.T) {
	f := newFixture(now)
	id := f.seed(apptA, questionnaireresponse.StatusInProgress)
	f.appts.err = errBackend
	if _, err := f.svc.Withdraw(t.Context(), pract, id); !errors.Is(err, errBackend) {
		t.Errorf("Withdraw() = %v, want the backend's error", err)
	}
	if len(f.audit.events) != 0 || f.repo.responses[0].Status != questionnaireresponse.StatusInProgress {
		t.Errorf("events %v, response %+v; want nothing changed", f.audit.events, f.repo.responses[0])
	}
}
