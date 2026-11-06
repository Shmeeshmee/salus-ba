package questionnaireresponse_test

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/questionnaireresponse"
)

// Read gives each viewer the level access.QuestionnaireResponse gives them, a Name view without answers, and
// audits the read; anyone at None gets ErrNotFound, audited without the entity.
func TestRead_GivesEachViewerTheirLevelAndAuditsIt(t *testing.T) {
	tests := []struct {
		setup  func(f *fixture)
		name   string
		status questionnaireresponse.Status
		caller appointment.Caller
		want   access.Level
	}{
		{name: "the patient, open", caller: me, status: questionnaireresponse.StatusInProgress, want: access.Full},
		{name: "the patient, submitted", caller: me, status: questionnaireresponse.StatusCompleted, want: access.Full},
		{name: "the practitioner, open", caller: pract, status: questionnaireresponse.StatusInProgress, want: access.Name},
		{name: "the practitioner, submitted", caller: pract, status: questionnaireresponse.StatusCompleted, want: access.Full},
		{name: "the practitioner, withdrawn", caller: pract, status: questionnaireresponse.StatusStopped, want: access.Name},
		{name: "the admin, submitted", caller: admin, status: questionnaireresponse.StatusCompleted, want: access.Name},
		{name: "the admin, open", caller: admin, status: questionnaireresponse.StatusInProgress, want: access.Name},
		{name: "another practitioner, submitted", caller: pract2, status: questionnaireresponse.StatusCompleted, want: access.None},
		{name: "another patient, submitted", caller: other, status: questionnaireresponse.StatusCompleted, want: access.None},
		{name: "a patient without a profile", caller: noProfile, status: questionnaireresponse.StatusCompleted, want: access.None},
		{name: "another clinic's admin", caller: adminB, status: questionnaireresponse.StatusCompleted, want: access.None},
		// D9: after a revocation, the submitted answers of a past appointment to its practitioner only.
		{name: "after a revocation, the practitioner, past and submitted", caller: pract, status: questionnaireresponse.StatusCompleted, want: access.Full,
			setup: func(f *fixture) { f.revoke(revokedAt.Add(-time.Hour)) }},
		{name: "after a revocation, the practitioner, past and open", caller: pract, status: questionnaireresponse.StatusInProgress, want: access.None,
			setup: func(f *fixture) { f.revoke(revokedAt.Add(-time.Hour)) }},
		{name: "after a revocation, the admin, past and submitted", caller: admin, status: questionnaireresponse.StatusCompleted, want: access.Name,
			setup: func(f *fixture) { f.revoke(revokedAt.Add(-time.Hour)) }},
		{name: "after a revocation, the practitioner, future and submitted", caller: pract, status: questionnaireresponse.StatusCompleted, want: access.None,
			setup: func(f *fixture) { f.revoke(start) }},
		{name: "after a revocation, the patient, future", caller: me, status: questionnaireresponse.StatusCompleted, want: access.Full,
			setup: func(f *fixture) { f.revoke(start) }},
	}
	for _, tt := range tests {
		f := newFixture(now)
		id := f.seed(apptA, tt.status)
		if tt.setup != nil {
			tt.setup(f)
		}

		got, err := f.svc.Read(t.Context(), tt.caller, id)

		stored := f.repo.responses[0]
		switch tt.want {
		case access.None:
			if !errors.Is(err, questionnaireresponse.ErrNotFound) {
				t.Errorf("%s: Read() = %+v, %v; want ErrNotFound", tt.name, got, err)
			}
		case access.Name:
			stored.Answers = nil
			fallthrough
		default:
			if err != nil || got.Level != tt.want || !equal(got.Response, stored) {
				t.Errorf("%s: Read() = %+v, %v;\nwant %v of %+v", tt.name, got, err, tt.want, stored)
			}
		}
		entity, outcome := id, auditevent.OutcomeSuccess
		if tt.want == access.None {
			entity, outcome = "", auditevent.OutcomeMinorFailure
		}
		if want := []auditevent.AuditEvent{event(tt.caller, auditevent.CodeRead, entity, outcome)}; !slices.Equal(f.audit.events, want) {
			t.Errorf("%s: audited %+v,\nwant %+v", tt.name, f.audit.events, want)
		}
	}
}

// The patient reads in their own scope, staff in their clinic's.
func TestRead_ScopesThePatientToThemselvesAndStaffToTheirClinic(t *testing.T) {
	for c, scope := range map[appointment.Caller][2]string{me: {patientA, ""}, pract: {"", clinicA}} {
		f := newFixture(now)
		id := f.seed(apptA, questionnaireresponse.StatusCompleted)
		if _, err := f.svc.Read(t.Context(), c, id); err != nil {
			t.Fatal(err)
		}
		if got := [2]string{f.repo.patient, f.repo.clinic}; got != scope {
			t.Errorf("as %s: scoped to %v, want %v", c.AccountID, got, scope)
		}
	}
}

func TestRead_IsNotFoundForAnUnknownResponseAndFailsWithTheBackend(t *testing.T) {
	f := newFixture(now)
	if _, err := f.svc.Read(t.Context(), pract, unknownID); !errors.Is(err, questionnaireresponse.ErrNotFound) {
		t.Errorf("Read(unknown) = %v, want ErrNotFound", err)
	}
	f = newFixture(now)
	id := f.seed(apptA, questionnaireresponse.StatusCompleted)
	f.appts.err = errBackend
	if _, err := f.svc.Read(t.Context(), pract, id); !errors.Is(err, errBackend) || len(f.audit.events) != 0 {
		t.Errorf("Read() = %v with %d events, want the backend's error and nothing audited", err, len(f.audit.events))
	}
}

// Search scopes each caller like Read: the patient only their own, whatever patient the filter names; staff
// their clinic's, each response at its level, those at None left out. Each response returned is audited; a
// search that returns none writes one event without an entity.
func TestSearch_GivesEachCallerTheResponsesTheyMaySeeAndAuditsEach(t *testing.T) {
	tests := []struct {
		name      string
		filter    questionnaireresponse.Filter
		want      []access.Level // per seeded response: A open, A submitted, B submitted
		wantScope questionnaireresponse.Filter
		caller    appointment.Caller
	}{
		{name: "the patient", caller: me, want: []access.Level{access.Full, access.Full, access.None},
			wantScope: questionnaireresponse.Filter{PatientID: patientA}},
		{name: "the patient naming another", caller: me, filter: questionnaireresponse.Filter{PatientID: patientB},
			want: []access.Level{access.Full, access.Full, access.None}, wantScope: questionnaireresponse.Filter{PatientID: patientA}},
		{name: "the practitioner", caller: pract, want: []access.Level{access.Name, access.Full, access.Full}},
		{name: "the practitioner by appointment and status", caller: pract,
			filter:    questionnaireresponse.Filter{AppointmentID: apptA, Status: questionnaireresponse.StatusCompleted},
			want:      []access.Level{access.None, access.Full, access.None},
			wantScope: questionnaireresponse.Filter{AppointmentID: apptA, Status: questionnaireresponse.StatusCompleted}},
		{name: "another practitioner", caller: pract2, want: []access.Level{access.None, access.None, access.None}},
		{name: "the admin", caller: admin, want: []access.Level{access.Name, access.Name, access.Name}},
		{name: "another clinic's admin", caller: adminB, want: []access.Level{access.None, access.None, access.None}},
	}
	for _, tt := range tests {
		f := newFixture(now)
		ids := []string{f.seed(apptA, questionnaireresponse.StatusInProgress), f.seed(apptA, questionnaireresponse.StatusCompleted),
			f.seed(apptB, questionnaireresponse.StatusCompleted)}

		got, err := f.svc.Search(t.Context(), tt.caller, tt.filter)

		if err != nil {
			t.Fatalf("%s: Search() error = %v", tt.name, err)
		}
		var wantIDs []string
		var wantEvents []auditevent.AuditEvent
		for i, level := range tt.want {
			if level != access.None {
				wantIDs = append(wantIDs, ids[i])
				wantEvents = append(wantEvents, event(tt.caller, auditevent.CodeSearchType, ids[i], auditevent.OutcomeSuccess))
			}
		}
		if len(wantEvents) == 0 {
			wantEvents = []auditevent.AuditEvent{event(tt.caller, auditevent.CodeSearchType, "", auditevent.OutcomeSuccess)}
		}
		var gotIDs []string
		for _, s := range got {
			gotIDs = append(gotIDs, s.Response.ID)
			i := slices.Index(ids, s.Response.ID)
			if i < 0 || s.Level != tt.want[i] || (s.Level == access.Name) != (s.Response.Answers == nil) {
				t.Errorf("%s: %s at %v with %d answers, want %v (a Name view has none)", tt.name, s.Response.ID, s.Level, len(s.Response.Answers), tt.want[max(i, 0)])
			}
		}
		if !slices.Equal(gotIDs, wantIDs) {
			t.Errorf("%s: Search() gave %v, want %v", tt.name, gotIDs, wantIDs)
		}
		if !slices.Equal(f.audit.events, wantEvents) {
			t.Errorf("%s: audited %+v,\nwant %+v", tt.name, f.audit.events, wantEvents)
		}
		if tt.caller.PatientID != "" && (len(f.repo.searched) != 1 || f.repo.searched[0] != tt.wantScope) {
			t.Errorf("%s: searched %+v, want %+v", tt.name, f.repo.searched, tt.wantScope)
		}
		if tt.wantScope.AppointmentID != "" && f.repo.searched[0] != tt.wantScope {
			t.Errorf("%s: searched %+v, want %+v", tt.name, f.repo.searched, tt.wantScope)
		}
	}
}

// A patient without a profile has no responses: nothing is searched, one event is written.
func TestSearch_GivesAPatientWithoutAProfileNothing(t *testing.T) {
	f := newFixture(now)
	f.seed(apptA, questionnaireresponse.StatusCompleted)
	got, err := f.svc.Search(t.Context(), noProfile, questionnaireresponse.Filter{})
	if err != nil || len(got) != 0 || len(f.repo.searched) != 0 {
		t.Errorf("Search() = %v, %v after %d searches; want nothing and no search", got, err, len(f.repo.searched))
	}
	if want := []auditevent.AuditEvent{event(noProfile, auditevent.CodeSearchType, "", auditevent.OutcomeSuccess)}; !slices.Equal(f.audit.events, want) {
		t.Errorf("audited %+v, want %+v", f.audit.events, want)
	}
}

// The appointment's attendance is asked once per appointment, however many of its responses a search finds.
func TestSearch_AsksEachAppointmentsAttendanceOnce(t *testing.T) {
	f := newFixture(now)
	f.seed(apptA, questionnaireresponse.StatusInProgress)
	f.seed(apptA, questionnaireresponse.StatusCompleted)
	f.seed(apptB, questionnaireresponse.StatusCompleted)
	if _, err := f.svc.Search(t.Context(), admin, questionnaireresponse.Filter{}); err != nil {
		t.Fatal(err)
	}
	if want := []string{apptA, apptB}; !slices.Equal(f.appts.asked, want) {
		t.Errorf("asked the attendance of %v, want %v", f.appts.asked, want)
	}
}

func TestSearch_FailsWithTheBackendAndAuditsNothing(t *testing.T) {
	for name, setup := range map[string]func(f *fixture){
		"the repository":   func(f *fixture) { f.repo.err = errBackend },
		"the appointments": func(f *fixture) { f.appts.err = errBackend },
	} {
		f := newFixture(now)
		f.seed(apptA, questionnaireresponse.StatusCompleted)
		setup(f)
		if _, err := f.svc.Search(t.Context(), pract, questionnaireresponse.Filter{}); !errors.Is(err, errBackend) || len(f.audit.events) != 0 {
			t.Errorf("%s: Search() = %v with %d events, want the backend's error and nothing audited", name, err, len(f.audit.events))
		}
	}
}
