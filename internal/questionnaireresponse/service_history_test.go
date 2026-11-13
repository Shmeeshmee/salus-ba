package questionnaireresponse_test

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/binary"
	"github.com/Shmeeshmee/salus/internal/encounter"
	"github.com/Shmeeshmee/salus/internal/questionnaireresponse"
	"github.com/Shmeeshmee/salus/pkg/keyset"
)

const (
	apptShared = "e5000000-0000-4000-8000-0000000000b5" // patient A's finished visit at clinic B, shared with clinic A
	sharedResp = "d5000000-0000-4000-8000-0000000000b5"
	openResp   = "d6000000-0000-4000-8000-0000000000b6"
	encShared  = "c5000000-0000-4000-8000-0000000000b5"
)

// withShared adds clinic B's visit of patient A shared with clinic A's practitioner: a completed response with a
// clean and a pending file, and one in progress the scope does not show.
func withShared(f *fixture) encounter.Visit {
	enc := encounter.Encounter{ID: encShared, AppointmentID: apptShared, OrganizationID: clinicB, PatientID: patientA, Status: encounter.StatusFinished}
	f.encounters.shared = map[string]encounter.Encounter{practAcc + apptShared: enc}
	for id, status := range map[string]questionnaireresponse.Status{sharedResp: questionnaireresponse.StatusCompleted, openResp: questionnaireresponse.StatusInProgress} {
		f.repo.responses = append(f.repo.responses, questionnaireresponse.Response{
			ID: id, OrganizationID: clinicB, AppointmentID: apptShared, PatientID: patientA, QuestionnaireID: qOther, SeriesID: seriesB,
			Status: status, Answers: answers, AuthoredAt: now.Add(-72 * time.Hour), VersionID: 3,
		})
	}
	f.repo.shared = []string{sharedResp, openResp}
	f.repo.attachments = map[string][]binary.Attachment{sharedResp: {
		{LinkID: "q3", BinaryID: "b1000000-0000-4000-8000-000000000001", Title: "Placeholder file"},
		{LinkID: "q3", BinaryID: "b1000000-0000-4000-8000-000000000002", Pending: true},
	}}
	return encounter.Visit{Encounter: enc, Shared: true}
}

func responseIDs(rs []questionnaireresponse.Response) []string {
	out := make([]string, len(rs))
	for i, r := range rs {
		out[i] = r.ID
	}
	return out
}

// History gives the completed responses of the visits the history shows, with their scanned files only, in the
// caller's clinic scope; never one in progress, nothing of a visit not given. Unaudited: the history audits what it
// shows (Disclosed).
func TestHistory_GivesTheCompletedResponsesOfTheVisits(t *testing.T) {
	t.Parallel()
	f := newFixture(now)
	own := f.seed(apptA, questionnaireresponse.StatusCompleted)
	f.seed(apptB, questionnaireresponse.StatusCompleted) // patient B's: not a visit given
	f.seed(apptA, questionnaireresponse.StatusInProgress)
	shared := withShared(f)
	visits := []encounter.Visit{{Encounter: encounter.Encounter{ID: "c1000000-0000-4000-8000-0000000000a1", AppointmentID: apptA, PatientID: patientA}}, shared}

	got, err := f.svc.History(t.Context(), pract, visits)
	if err != nil || !slices.Equal(responseIDs(got), []string{own, sharedResp}) {
		t.Fatalf("History() = %v, %v; want %s and the shared one", responseIDs(got), err, own)
	}
	if files := got[1].Attachments; len(files) != 1 || files[0].Pending {
		t.Errorf("History() files = %+v, want the scanned one only", files)
	}
	if len(f.audit.events) != 0 {
		t.Errorf("audit = %+v, want none", f.audit.events)
	}
	if _, err := f.svc.History(t.Context(), admin, visits); !errors.Is(err, questionnaireresponse.ErrNotFound) {
		t.Errorf("History(admin) = %v, want ErrNotFound", err)
	}
	f.repo.err = errBackend
	if _, err := f.svc.History(t.Context(), pract, visits); !errors.Is(err, errBackend) {
		t.Errorf("History(down) = %v, want the failure", err)
	}
}

// Disclosed audits each response the history shows (with its files) as a search of it.
func TestDisclosed_AuditsEachResponse(t *testing.T) {
	t.Parallel()
	f := newFixture(now)
	if err := f.svc.Disclosed(t.Context(), pract, []string{sharedResp}); err != nil {
		t.Fatal(err)
	}
	if want := []auditevent.AuditEvent{event(pract, auditevent.CodeSearchType, sharedResp, auditevent.OutcomeSuccess)}; !slices.Equal(f.audit.events, want) {
		t.Errorf("audit = %+v, want %+v", f.audit.events, want)
	}
}

// DisclosedFiles audits each file the history shows the caller, as a search of the Binary.
func TestDisclosedFiles_AuditsEachFileAsItsBinary(t *testing.T) {
	t.Parallel()
	f := newFixture(now)
	file := "f9000000-0000-4000-8000-0000000000f9"
	if err := f.svc.DisclosedFiles(t.Context(), pract, []string{file}); err != nil {
		t.Fatal(err)
	}
	want := event(pract, auditevent.CodeSearchType, "", auditevent.OutcomeSuccess)
	want.EntityReference = "Binary/" + file
	if !slices.Equal(f.audit.events, []auditevent.AuditEvent{want}) {
		t.Errorf("audit = %+v, want %+v", f.audit.events, want)
	}
}

// A completed response of another clinic's visit the patient shared is read in full, and its files are readable,
// by the practitioner the history shows the visit to; nobody else, and never once the visit is no longer shared.
func TestRead_ASharedVisitsCompletedResponse(t *testing.T) {
	t.Parallel()
	f := newFixture(now)
	withShared(f)
	got, err := f.svc.Read(t.Context(), pract, sharedResp)
	if err != nil || got.Level != access.Full || len(got.Response.Answers) != 1 {
		t.Fatalf("Read(shared) = %+v, %v; want it in full", got, err)
	}
	grant, err := f.svc.Readable(t.Context(), binary.Caller{AccountID: practAcc, OrganizationID: clinicA, PractitionerID: practA1}, sharedResp)
	if err != nil || grant.Purpose != auditevent.CodeTREAT {
		t.Errorf("Readable(shared) = %+v, %v; want a grant for treatment", grant, err)
	}
	if want := []auditevent.AuditEvent{event(pract, auditevent.CodeRead, sharedResp, auditevent.OutcomeSuccess)}; !slices.Equal(f.audit.events, want) {
		t.Errorf("audit = %+v, want %+v", f.audit.events, want)
	}
	for name, c := range map[string]appointment.Caller{"the admin": admin, "another practitioner": pract2} {
		if _, err := f.svc.Read(t.Context(), c, sharedResp); !errors.Is(err, questionnaireresponse.ErrNotFound) {
			t.Errorf("Read by %s = %v, want ErrNotFound", name, err)
		}
	}
	if _, err := f.svc.Read(t.Context(), pract, openResp); !errors.Is(err, questionnaireresponse.ErrNotFound) {
		t.Errorf("Read(in progress) = %v, want ErrNotFound", err)
	}
	f.encounters.shared = nil
	if _, err := f.svc.Read(t.Context(), pract, sharedResp); !errors.Is(err, questionnaireresponse.ErrNotFound) {
		t.Errorf("Read once unshared = %v, want ErrNotFound", err)
	}
}

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
