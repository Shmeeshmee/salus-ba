package communication_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/communication"
)

// long opens patient A's conversation and writes n messages in it, alternately the patient's and the doctor's.
func (f *fixture) long(t *testing.T, n int) communication.Conversation {
	t.Helper()
	c := f.open(t)
	for i := range n {
		who := me
		if i%2 == 1 {
			who = pract
		}
		f.send(t, who, c.ID, "Synthetic placeholder")
	}
	return c
}

func TestSummarize_SummarisesTheLatestMessagesForTheDoctorAndAuditsTheRowsRead(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	c := f.long(t, communication.SummaryMinMessages)
	events := f.audit.count()
	got, err := f.svc.Summarize(t.Context(), pract, c.ID, "hr")
	if err != nil {
		t.Fatal(err)
	}
	want := "[hr] Placeholder summary of 50 messages"
	if got.Summary != want || !got.AIGenerated || got.UpToSeq != 50 || !got.GeneratedAt.Equal(start) {
		t.Errorf("Summarize() = %+v, want %q up to 50, AI generated, at the clock's time", got, want)
	}
	calls := f.sum.Calls()
	if len(calls) != 1 || calls[0].Lang != "hr" || len(calls[0].Messages) != 50 {
		t.Fatalf("the model was asked %+v", calls)
	}
	first, last := calls[0].Messages[0], calls[0].Messages[49]
	if first.Speaker != "patient" || last.Speaker != "practitioner" || first.Text != "Synthetic placeholder" || first.ID == "" {
		t.Errorf("the messages go oldest first with their roles: first %+v, last %+v", first, last)
	}
	if f.audit.count() != events+1 {
		t.Fatalf("%d audit events, want one", f.audit.count()-events)
	}
	if e := f.audit.last(t); e.Action != auditevent.ActionRead || e.EntityName != "communication" || e.EntityCount != 50 ||
		e.EntityReference != "Patient/"+patientA || e.Purpose.Code != auditevent.CodeTREAT {
		t.Errorf("audit = %+v", e)
	}
	if _, err := f.svc.Summarize(t.Context(), pract, c.ID, ""); err != nil || f.sum.Calls()[1].Lang != "bs" {
		t.Errorf("no language: %v, want the clinic's default, bs", err)
	}
}

// Only the latest SummaryMaxMessages messages are read and sent to the model, so a call's cost is bounded.
func TestSummarize_ReadsAtMostTheLatestMaximum(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	c := f.long(t, communication.SummaryMaxMessages+3)
	got, err := f.svc.Summarize(t.Context(), pract, c.ID, "en")
	if err != nil {
		t.Fatal(err)
	}
	msgs := f.sum.Calls()[0].Messages
	if len(msgs) != communication.SummaryMaxMessages || got.UpToSeq != communication.SummaryMaxMessages+3 || f.audit.last(t).EntityCount != communication.SummaryMaxMessages {
		t.Errorf("%d messages up to %d, audited %d", len(msgs), got.UpToSeq, f.audit.last(t).EntityCount)
	}
}

// A doctor added later summarises only from their point, and below the threshold there is nothing to summarise.
func TestSummarize_CountsOnlyWhatTheCallerReads(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	c := f.long(t, communication.SummaryMinMessages+1)
	f.repo.addPractitioner(c.ID, roleA2, 2)
	if _, err := f.svc.Summarize(t.Context(), pract2, c.ID, "en"); !errors.Is(err, communication.ErrInvalid) {
		t.Errorf("a doctor who reads fewer than the threshold: %v, want invalid", err)
	}
	if len(f.sum.Calls()) != 0 {
		t.Error("the model was asked")
	}
}

func TestSummarize_RefusesThePatientOthersAndWhatTheContractRefuses(t *testing.T) {
	t.Parallel()
	tests := []struct {
		want   error
		name   string
		lang   string
		revoke bool
		short  bool
		other  bool
	}{
		{want: communication.ErrForbidden, name: "the patient", other: false},
		{want: communication.ErrForbidden, name: "a doctor who does not take part", other: true},
		{want: communication.ErrNotFound, name: "the doctor after the revocation", revoke: true},
		{want: communication.ErrInvalid, name: "a short conversation", short: true},
		{want: communication.ErrInvalid, name: "a language the staff do not use", lang: "de"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newFixture(t)
			n := communication.SummaryMinMessages
			if tt.short {
				n--
			}
			c := f.long(t, n)
			if tt.revoke {
				f.revoke()
			}
			caller := pract
			switch {
			case tt.name == "the patient":
				caller = me
			case tt.other:
				caller = pract2
			}
			lang := tt.lang
			if lang == "" {
				lang = "en"
			}
			if _, err := f.svc.Summarize(t.Context(), caller, c.ID, lang); !errors.Is(err, tt.want) {
				t.Errorf("Summarize() = %v, want %v", err, tt.want)
			}
			if len(f.sum.Calls()) != 0 {
				t.Error("the model was asked")
			}
		})
	}
}

// A failure of the model is an unexpected error (a 500 to the client), and the read stays audited.
func TestSummarize_AFailingModelIsAnUnexpectedErrorAndTheReadStaysAudited(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	c := f.long(t, communication.SummaryMinMessages)
	f.sum.Err = errBackend
	_, err := f.svc.Summarize(t.Context(), pract, c.ID, "en")
	if !errors.Is(err, errBackend) {
		t.Fatalf("Summarize() = %v, want an unexpected error", err)
	}
	if !strings.Contains(err.Error(), "summary") {
		t.Errorf("error %q does not name the operation", err)
	}
	if e := f.audit.last(t); e.EntityCount != 50 {
		t.Errorf("the read is not audited: %+v", e)
	}
}
