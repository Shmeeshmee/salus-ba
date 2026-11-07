package questionnaireresponse_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/binary"
	"github.com/Shmeeshmee/salus/internal/questionnaire"
	"github.com/Shmeeshmee/salus/internal/questionnaireresponse"
)

// asBinary is c as binary sees the caller.
func asBinary(c appointment.Caller) binary.Caller {
	return binary.Caller{AccountID: c.AccountID, OrganizationID: c.OrganizationID, PractitionerID: c.PractitionerID, PatientID: c.PatientID, Admin: c.Admin}
}

// withFiles gives the active version an attachment item "files" (photos and documents, two files) in a section.
func withFiles(f *fixture) {
	q := f.questionnaires.versions[qActive]
	q.Items = append(slices.Clone(q.Items), questionnaire.Item{LinkID: "s1", Text: "Placeholder section", Type: questionnaire.ItemGroup, Items: []questionnaire.Item{
		{LinkID: "files", Text: "Placeholder upload", Type: questionnaire.ItemAttachment,
			Accept: []questionnaire.FileKind{questionnaire.FilePhoto, questionnaire.FileDocument}, MaxFiles: 2},
	}})
	f.questionnaires.versions[qActive] = q
}

// binaryRefusal is the binary error the holder answers for the response's own.
var binaryRefusal = map[error]error{
	questionnaireresponse.ErrNotFound: binary.ErrNotFound, questionnaireresponse.ErrForbidden: binary.ErrForbidden,
	questionnaireresponse.ErrSubmitted: binary.ErrSubmitted, questionnaireresponse.ErrWithdrawn: binary.ErrWithdrawn,
	questionnaireresponse.ErrPastDue: binary.ErrPastDue, questionnaireresponse.ErrAppointmentClosed: binary.ErrAppointmentClosed,
	questionnaireresponse.ErrConsentRequired: binary.ErrConsentRequired,
}

func TestUploadTarget_IsThePatientsOpenResponseAndTheItemsSettings(t *testing.T) {
	f := newFixture(now)
	withFiles(f)
	id := f.seed(apptA, questionnaireresponse.StatusInProgress)
	got, err := f.svc.UploadTarget(t.Context(), asBinary(me), id, "files")
	want := binary.Target{OrganizationID: clinicA, PatientID: patientA, AppointmentID: apptA, Accept: []binary.Kind{binary.KindPhoto, binary.KindDocument}, MaxFiles: 2}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("UploadTarget() = %+v, %v; want %+v", got, err, want)
	}
	if !slices.Equal(f.appts.locked, []string{apptA}) {
		t.Errorf("locked %v, want the appointment against a cancellation", f.appts.locked)
	}
	if events := own(f.audit.events); len(events) != 0 {
		t.Errorf("the holder audited %+v; binary audits the upload", events)
	}
}

func TestUploadTargetAndEditable_RefuseAsAChangeOfTheAnswersIs(t *testing.T) {
	for _, tt := range answerRefusals {
		if tt.version != 0 { // no version: the files are not the typed answers
			continue
		}
		at := now
		if !tt.at.IsZero() {
			at = tt.at
		}
		f := newFixture(at)
		withFiles(f)
		status := tt.status
		if status == "" {
			status = questionnaireresponse.StatusInProgress
		}
		id := f.seed(apptA, status)
		if tt.setup != nil {
			tt.setup(f)
		}
		if tt.unknown {
			id = unknownID
		}
		want := binaryRefusal[tt.want]
		if _, err := f.svc.UploadTarget(t.Context(), asBinary(tt.caller), id, "files"); !errors.Is(err, want) {
			t.Errorf("%s: UploadTarget() = %v, want %v", tt.name, err, want)
		}
		if err := f.svc.Editable(t.Context(), asBinary(tt.caller), id); !errors.Is(err, want) {
			t.Errorf("%s: Editable() = %v, want %v", tt.name, err, want)
		}
		if events := own(f.audit.events); len(events) != 0 {
			t.Errorf("%s: the holder audited %+v", tt.name, events)
		}
	}
}

func TestUploadTarget_RefusesAnItemThatTakesNoFiles(t *testing.T) {
	f := newFixture(now)
	withFiles(f)
	id := f.seed(apptA, questionnaireresponse.StatusInProgress)
	for _, link := range []string{"q1", "s1", "nothing"} {
		if _, err := f.svc.UploadTarget(t.Context(), asBinary(me), id, link); !errors.Is(err, binary.ErrNotAttachmentItem) {
			t.Errorf("UploadTarget(%s) = %v, want ErrNotAttachmentItem", link, err)
		}
	}
	if err := f.svc.Editable(t.Context(), asBinary(me), id); err != nil {
		t.Errorf("Editable() = %v", err)
	}
}

func TestReadable_IsTheResponsesAccessRuleAtFull(t *testing.T) {
	for _, tt := range []struct {
		setup   func(*fixture)
		name    string
		status  questionnaireresponse.Status
		purpose string // "" for refused
		caller  appointment.Caller
	}{
		{name: "the patient, open", caller: me, status: questionnaireresponse.StatusInProgress, purpose: auditevent.CodePATRQT},
		{name: "the patient, submitted", caller: me, status: questionnaireresponse.StatusCompleted, purpose: auditevent.CodePATRQT},
		{name: "the practitioner, submitted", caller: pract, status: questionnaireresponse.StatusCompleted, purpose: auditevent.CodeTREAT},
		{name: "the practitioner, open", caller: pract, status: questionnaireresponse.StatusInProgress},
		{name: "the admin, submitted", caller: admin, status: questionnaireresponse.StatusCompleted},
		{name: "another practitioner", caller: pract2, status: questionnaireresponse.StatusCompleted},
		{name: "another patient", caller: other, status: questionnaireresponse.StatusCompleted},
		{name: "D9: after a revocation, past and submitted", caller: pract, status: questionnaireresponse.StatusCompleted, purpose: auditevent.CodeTREAT,
			setup: func(f *fixture) { f.revoke(revokedAt.Add(-time.Hour)) }},
		{name: "after a revocation, future and submitted", caller: pract, status: questionnaireresponse.StatusCompleted,
			setup: func(f *fixture) { f.revoke(start) }},
	} {
		f := newFixture(now)
		id := f.seed(apptA, tt.status)
		if tt.setup != nil {
			tt.setup(f)
		}
		g, err := f.svc.Readable(t.Context(), asBinary(tt.caller), id)
		switch {
		case tt.purpose == "" && !errors.Is(err, binary.ErrNotFound):
			t.Errorf("%s: Readable() = %+v, %v; want ErrNotFound", tt.name, g, err)
		case tt.purpose != "" && (err != nil || g.Purpose != tt.purpose):
			t.Errorf("%s: Readable() = %+v, %v; want %s", tt.name, g, err, tt.purpose)
		}
		if len(f.audit.events) != 0 {
			t.Errorf("%s: the holder audited %+v", tt.name, f.audit.events)
		}
	}
}

func TestHolder_KeepsTheAnswersFilesInStep(t *testing.T) {
	f := newFixture(now)
	id := f.seed(apptA, questionnaireresponse.StatusInProgress)
	before := f.repo.responses[0]
	ctx := t.Context()
	a, b := binary.Attachment{LinkID: "files", BinaryID: "b1", Pending: true}, binary.Attachment{LinkID: "files", BinaryID: "b2", Pending: true}
	for _, x := range []binary.Attachment{a, b} {
		if err := f.svc.Attach(ctx, id, x); !errors.Is(err, questionnaireresponse.ErrNotFound) {
			t.Fatalf("Attach() outside a scope = %v, want ErrNotFound (the caller's scope decides)", err)
		}
	}
	if err := f.svc.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := f.repo.SetPatient(ctx, patientA); err != nil {
			return err
		}
		for _, x := range []binary.Attachment{a, b} {
			if err := f.svc.Attach(ctx, id, x); err != nil {
				return err
			}
		}
		if err := f.svc.Cleared(ctx, id, "b2"); err != nil {
			return err
		}
		return f.svc.Detach(ctx, id, "b1")
	}); err != nil {
		t.Fatal(err)
	}
	b.Pending = false
	if got := f.repo.attachments[id]; !reflect.DeepEqual(got, []binary.Attachment{b}) {
		t.Errorf("attachments %+v, want %+v", got, []binary.Attachment{b})
	}
	got, err := f.svc.Attached(ctx, id)
	if err != nil || !slices.Equal(got, []string{"b2"}) {
		t.Errorf("Attached() = %v, %v; want [b2]", got, err)
	}
	if after := f.repo.responses[0]; after.VersionID != before.VersionID {
		t.Errorf("the files moved the version from %d to %d", before.VersionID, after.VersionID)
	}
}
