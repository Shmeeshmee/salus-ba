package questionnaireresponse_test

import (
	"errors"
	"testing"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/binary"
	"github.com/Shmeeshmee/salus/internal/questionnaireresponse"
	"github.com/Shmeeshmee/salus/pkg/keyset"
)

const (
	fileClean   = "b1000000-0000-4000-8000-0000000000b1"
	filePending = "b2000000-0000-4000-8000-0000000000b2"
)

// historyFixture is patient A's completed response (with a clean and a pending file), one in progress, one
// withdrawn, and patient B's completed one.
func historyFixture() (*fixture, string) {
	f := newFixture(now)
	done := f.seed(apptA, questionnaireresponse.StatusCompleted)
	f.seed(apptA, questionnaireresponse.StatusInProgress)
	f.seed(apptA, questionnaireresponse.StatusStopped)
	f.seed(apptB, questionnaireresponse.StatusCompleted)
	f.withFiles(done,
		binary.Attachment{LinkID: "q2", BinaryID: fileClean, ContentType: "image/png", Title: "Placeholder.png", Size: 10},
		binary.Attachment{LinkID: "q2", BinaryID: filePending, ContentType: "image/png", Title: "Pending.png", Size: 10, Pending: true})
	return f, done
}

// SAL-39: the patient's timeline lists their own completed responses, never one in progress or withdrawn, after
// the page's position, in their own scope, without their answers, each audited as a search of their own request.
func TestPatientHistory_GivesThePatientTheirCompletedResponses(t *testing.T) {
	f, done := historyFixture()
	p := keyset.Page{At: now, Ref: "QuestionnaireResponse/x", Limit: 5}
	got, err := f.svc.PatientHistory(t.Context(), me, clinicA, p)
	if err != nil || len(got) != 1 || got[0].ID != done || got[0].Answers != nil {
		t.Fatalf("PatientHistory() = %+v, %v; want the completed one without its answers", got, err)
	}
	if f.repo.patient != patientA || f.repo.history.page != p || f.repo.history.org != clinicA {
		t.Errorf("read in patient %q, asked %+v; want patient A's scope and the page", f.repo.patient, f.repo.history)
	}
	if len(f.audit.events) != 0 {
		t.Errorf("audited %+v, want nothing: history.Timeline audits the page it returns", f.audit.events)
	}
}

// The files of the timeline are the clean files of the patient's completed responses, each with its response,
// unaudited; a pending one is not listed.
func TestPatientAttachments_GivesTheCleanFilesOfCompletedResponses(t *testing.T) {
	f, done := historyFixture()
	got, err := f.svc.PatientAttachments(t.Context(), me, "", keyset.Page{Limit: 5})
	if err != nil || len(got) != 1 || got[0].File.BinaryID != fileClean || got[0].Response.ID != done || got[0].Response.Answers != nil {
		t.Fatalf("PatientAttachments() = %+v, %v; want the clean file with its response", got, err)
	}
	if len(f.audit.events) != 0 {
		t.Errorf("audited %+v, want nothing", f.audit.events)
	}
}

// None is none, unaudited; a patient without a profile has none; staff get ErrForbidden.
func TestPatientHistory_NoneAndStaff(t *testing.T) {
	for name, list := range map[string]func(*fixture, appointment.Caller) error{
		"responses": func(f *fixture, c appointment.Caller) error {
			_, err := f.svc.PatientHistory(t.Context(), c, "", keyset.Page{Limit: 5})
			return err
		},
		"attachments": func(f *fixture, c appointment.Caller) error {
			_, err := f.svc.PatientAttachments(t.Context(), c, "", keyset.Page{Limit: 5})
			return err
		},
	} {
		for _, c := range []appointment.Caller{noProfile, me} {
			f := newFixture(now)
			if err := list(f, c); err != nil {
				t.Errorf("%s: %v, want none", name, err)
			}
			if len(f.audit.events) != 0 {
				t.Errorf("%s: audited %+v, want nothing", name, f.audit.events)
			}
		}
		f, _ := historyFixture()
		if err := list(f, pract); !errors.Is(err, questionnaireresponse.ErrForbidden) {
			t.Errorf("%s as staff: %v, want ErrForbidden", name, err)
		}
		if len(f.audit.events) != 0 {
			t.Errorf("%s as staff: audited %+v, want nothing", name, f.audit.events)
		}
		f, _ = historyFixture()
		f.repo.err = errBackend
		if err := list(f, me); !errors.Is(err, errBackend) {
			t.Errorf("%s: %v, want the backend's error", name, err)
		}
	}
}
