package questionnaireresponse_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/questionnaireresponse"
)

const agentBooked = "appointment_booked_email"

// noticeEvent is the audit event of the notifier's check: a search of appointment A's responses by the
// e-mail job, one per response read, or one without an entity when there is none.
func noticeEvent(id string) auditevent.AuditEvent {
	e := jobEvent(agentBooked, auditevent.CodeSearchType, id)
	e.Action = auditevent.ActionExecute
	if id == "" {
		e.EntityReference = ""
	}
	return e
}

func TestOutstanding_TellsWhetherTheAppointmentHasAQuestionnaireToFillIn(t *testing.T) {
	for _, tt := range []struct {
		setup func(f *fixture) []string // the responses the check reads
		name  string
		want  bool
	}{
		{name: "nothing requested, no default", want: false, setup: func(*fixture) []string { return nil }},
		{name: "a response in progress", want: true, setup: func(f *fixture) []string {
			return []string{f.seed(apptA, questionnaireresponse.StatusInProgress)}
		}},
		{name: "a submitted response", want: false, setup: func(f *fixture) []string {
			return []string{f.seed(apptA, questionnaireresponse.StatusCompleted)}
		}},
		{name: "a withdrawn response", want: false, setup: func(f *fixture) []string {
			return []string{f.seed(apptA, questionnaireresponse.StatusStopped)}
		}},
		{name: "the role's active default not yet requested", want: true, setup: func(f *fixture) []string {
			f.questionnaires.defaults[roleA1] = qActive
			return nil
		}},
		{name: "the role's default already submitted", want: false, setup: func(f *fixture) []string {
			f.questionnaires.defaults[roleA1] = qActive
			return []string{f.seed(apptA, questionnaireresponse.StatusCompleted)} // of the default's series
		}},
		{name: "the role's default retired", want: false, setup: func(f *fixture) []string {
			f.questionnaires.defaults[roleA1] = qRetired
			return nil
		}},
		{name: "a default of another clinic", want: false, setup: func(f *fixture) []string {
			f.questionnaires.defaults[roleA1] = qOther
			return nil
		}},
		{name: "another appointment's response in progress", want: false, setup: func(f *fixture) []string {
			f.seed(apptB, questionnaireresponse.StatusInProgress)
			return nil
		}},
	} {
		f := newFixture(now)
		read := tt.setup(f)
		got, err := f.svc.Outstanding(t.Context(), agentBooked, clinicA, apptA)
		if err != nil || got != tt.want {
			t.Errorf("%s: Outstanding() = %v, %v; want %v", tt.name, got, err, tt.want)
		}
		if f.repo.clinic != clinicA || f.repo.notice != apptA {
			t.Errorf("%s: in clinic %q with the notice %q, want clinic A's scope with the appointment's notice", tt.name, f.repo.clinic, f.repo.notice)
		}
		want := []auditevent.AuditEvent{noticeEvent("")}
		if len(read) > 0 {
			want = want[:0]
			for _, id := range read {
				want = append(want, noticeEvent(id))
			}
		}
		if got := own(f.audit.events); !slices.Equal(got, want) {
			t.Errorf("%s: audit events %+v,\nwant %+v", tt.name, got, want)
		}
		if len(f.questionnaires.agents) > 0 && !slices.Equal(f.questionnaires.agents, []string{agentBooked}) {
			t.Errorf("%s: the default asked by %v, want the job as agent", tt.name, f.questionnaires.agents)
		}
	}
}

// The check runs for an appointment the job knows: one unknown, or of another clinic, has nothing outstanding.
func TestOutstanding_IsFalseForAnAppointmentOutsideTheClinic(t *testing.T) {
	for _, tt := range []struct{ name, appt, org string }{
		{name: "an unknown appointment", appt: unknownID, org: clinicA},
		{name: "an appointment of another clinic", appt: apptA, org: clinicB},
	} {
		f := newFixture(now)
		f.questionnaires.defaults[roleA1] = qActive
		f.seed(apptA, questionnaireresponse.StatusInProgress)
		got, err := f.svc.Outstanding(t.Context(), agentBooked, tt.org, tt.appt)
		if err != nil || got {
			t.Errorf("%s: Outstanding() = %v, %v; want false", tt.name, got, err)
		}
	}
}

func TestOutstanding_FailsOnABackendErrorAndKeepsNoAudit(t *testing.T) {
	for name, breakIt := range map[string]func(f *fixture){
		"appointments":   func(f *fixture) { f.appts.err = errBackend },
		"questionnaires": func(f *fixture) { f.questionnaires.err = errBackend },
		"repository":     func(f *fixture) { f.repo.err = errBackend },
	} {
		f := newFixture(now)
		f.questionnaires.defaults[roleA1] = qActive
		breakIt(f)
		got, err := f.svc.Outstanding(t.Context(), agentBooked, clinicA, apptA)
		if !errors.Is(err, errBackend) || got || strings.Contains(err.Error(), apptA) {
			t.Errorf("%s: Outstanding() = %v, %v; want the backend's error without the id", name, got, err)
		}
		if len(f.audit.events) != 0 {
			t.Errorf("%s: events %+v after the rollback, want none", name, f.audit.events)
		}
	}
}
