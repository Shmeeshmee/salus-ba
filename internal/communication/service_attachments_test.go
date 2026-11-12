package communication_test

import (
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/binary"
	"github.com/Shmeeshmee/salus/internal/communication"
)

const (
	pdfBytes = "%PDF-1.4 synthetic"
	pngBytes = "\x89PNG synthetic"
)

func (f *fixture) upload(t *testing.T, c communication.Caller, conv, body string) communication.Attachment {
	t.Helper()
	a, err := f.svc.Upload(t.Context(), c, conv, communication.Upload{Body: strings.NewReader(body), Filename: "Synthetic lab result.pdf", Size: int64(len(body))})
	if err != nil {
		t.Fatal(err)
	}
	return a
}

// A participant who may write uploads a file to the conversation (SAL-30): through binary, as their participant, in
// the conversation's clinic and patient; it is scanning until its scan; the upload is audited as created.
func TestUpload_StoresTheWritersFileScanningAndAuditsIt(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	c := f.open(t)
	for _, who := range []communication.Caller{me, pract} {
		a := f.upload(t, who, c.ID, pdfBytes)
		if a.Status != communication.AttachmentScanning || a.ContentType != "application/pdf" || a.Filename != "Synthetic lab result.pdf" || a.Thumbnail {
			t.Errorf("%s: Upload() = %+v", who.AccountID, a)
		}
		b := f.sal30.files.rows[a.ID]
		part, _ := participantOf(c, who)
		if b.ConversationID != c.ID || b.ParticipantID != part || b.OrganizationID != clinicA || b.PatientID != patientA {
			t.Errorf("%s: stored %+v, want the caller's participant in the conversation", who.AccountID, b)
		}
		if e := f.audit.last(t); e.Action != auditevent.ActionCreate || e.EntityReference != "Binary/"+a.ID || e.AgentAccountID != who.AccountID ||
			e.Outcome != auditevent.OutcomeSuccess {
			t.Errorf("%s: audit = %+v", who.AccountID, e)
		}
	}
}

// participantOf is the caller's participant row of the conversation.
func participantOf(c communication.Conversation, who communication.Caller) (string, bool) {
	for _, p := range c.Participants {
		if (who.PatientID != "" && p.PatientID == who.PatientID) || (who.OrganizationID != "" && p.Role == communication.RolePractitioner && p.PractitionerRoleID == roleA1) {
			return p.ID, true
		}
	}
	return "", false
}

// Who may not write uploads nothing, before a byte is stored: another patient (not found), a practitioner who takes
// no part (forbidden), the patient after their revocation (consent required, audited); binary's refusal of the
// bytes is audited as a refused creation without a file.
func TestUpload_RefusesWhoMayNotWriteAndAuditsTheRefusals(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	c := f.open(t)
	for who, want := range map[*communication.Caller]error{&other: communication.ErrNotFound, &pract2: communication.ErrForbidden} {
		if _, err := f.svc.Upload(t.Context(), *who, c.ID, communication.Upload{Body: strings.NewReader(pdfBytes), Size: 3}); !errors.Is(err, want) {
			t.Errorf("%s: %v, want %v", who.AccountID, err, want)
		}
	}
	if _, err := f.svc.Upload(t.Context(), me, c.ID, communication.Upload{Body: strings.NewReader("plain text"), Size: 10}); !errors.Is(err, binary.ErrUnsupportedType) {
		t.Fatalf("a text file: %v, want binary.ErrUnsupportedType", err)
	}
	if e := f.audit.last(t); e.Outcome != auditevent.OutcomeMinorFailure || e.Action != auditevent.ActionCreate || e.EntityName != "binary" ||
		e.EntityReference != "Patient/"+patientA {
		t.Errorf("the refusal's audit = %+v", e)
	}
	f.revoke()
	if _, err := f.svc.Upload(t.Context(), me, c.ID, communication.Upload{Body: strings.NewReader(pdfBytes), Size: 3}); !errors.Is(err, communication.ErrConsentRequired) {
		t.Errorf("after the revocation: %v, want ErrConsentRequired", err)
	}
	if e := f.audit.last(t); e.Outcome != auditevent.OutcomeMinorFailure {
		t.Errorf("the revocation's refusal: %+v", e)
	}
	if len(f.sal30.files.rows) != 0 {
		t.Errorf("%d files stored", len(f.sal30.files.rows))
	}
}

// A send carries the sender's uploads: the body may then be empty; the message lists them, as stored, in its answer,
// its live frame and its page; a repeat answers the first with them.
func TestSend_AttachesTheSendersUploadsToTheMessage(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	c := f.open(t)
	a, b := f.upload(t, me, c.ID, pdfBytes), f.upload(t, me, c.ID, pngBytes)
	m, created, err := f.svc.Send(t.Context(), me, c.ID, clientID1, "   ", a.ID, b.ID)
	if err != nil || !created || m.Body != "" || m.AttachmentCount != 2 || len(m.Attachments) != 2 || m.Attachments[0].ID != a.ID {
		t.Fatalf("Send() = %+v, %v", m, err)
	}
	var live struct {
		Body        *string                    `json:"body"`
		Attachments []communication.Attachment `json:"attachments"`
	}
	if frames := f.pub.published(); len(frames) == 0 || json.Unmarshal(frames[len(frames)-1].Payload, &live) != nil || len(live.Attachments) != 2 {
		t.Errorf("live frame = %+v", frames)
	}
	f.sal30.files.set(b.ID, binary.StatusClean, true)
	page, err := f.svc.Messages(t.Context(), pract, c.ID, nil, 0)
	if err != nil || len(page.Messages) != 1 || len(page.Messages[0].Attachments) != 2 {
		t.Fatalf("page = %+v, %v", page, err)
	}
	if got := page.Messages[0].Attachments[1]; got.Status != communication.AttachmentClean || !got.Thumbnail {
		t.Errorf("the scanned image on the page: %+v", got)
	}
	again, created, err := f.svc.Send(t.Context(), me, c.ID, clientID1, "", a.ID, b.ID)
	if err != nil || created || again.ID != m.ID || len(again.Attachments) != 2 {
		t.Errorf("repeat = %+v, %v, %v", again, created, err)
	}
}

// A send's files must each be the sender's upload to the conversation, not yet attached: else, or past ten, or
// not UUIDs, nothing is sent. A body is still needed without files; a message of files alone has no body to edit.
func TestSend_RefusesFilesThatAreNotTheSendersToAttach(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	c := f.open(t)
	theirs := f.upload(t, pract, c.ID, pdfBytes)
	mine := f.upload(t, me, c.ID, pdfBytes)
	eleven := make([]string, 11)
	for i := range eleven {
		eleven[i] = unknownID
	}
	for name, ids := range map[string][]string{
		"someone else's": {theirs.ID}, "unknown": {unknownID}, "not a UUID": {"x"}, "eleven": eleven, "none, without a body": nil,
	} {
		if _, _, err := f.svc.Send(t.Context(), me, c.ID, clientID2, "", ids...); !errors.Is(err, communication.ErrInvalid) {
			t.Errorf("%s: %v, want ErrInvalid", name, err)
		}
	}
	if n := len(f.repo.msgs[c.ID]); n != 0 {
		t.Fatalf("%d messages stored", n)
	}
	m, _, err := f.svc.Send(t.Context(), me, c.ID, clientID1, "", mine.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := f.svc.Send(t.Context(), me, c.ID, clientID2, "Synthetic", mine.ID); !errors.Is(err, communication.ErrInvalid) {
		t.Errorf("attached twice: %v, want ErrInvalid", err)
	}
	if _, err := f.svc.Edit(t.Context(), me, c.ID, m.ID, "Synthetic caption", 1); !errors.Is(err, communication.ErrForbidden) {
		t.Errorf("editing a message of files alone: %v, want ErrForbidden", err)
	}
}

// A file opens for whoever reads its message (a practitioner who joined later only after their history_from_seq),
// before it is sent for its uploader only, once clean (scanning otherwise), its thumbnail when it has one; each open
// audited as a read of the file. Staff after the revocation get nothing.
func TestAttachment_OpensForWhoeverReadsItsMessageOnceClean(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	c := f.open(t)
	sent := f.upload(t, me, c.ID, pngBytes)
	f.send(t, pract, c.ID, "Synthetic first")
	if _, _, err := f.svc.Send(t.Context(), me, c.ID, clientID1, "", sent.ID); err != nil {
		t.Fatal(err)
	}
	draft := f.upload(t, me, c.ID, pdfBytes)
	if _, err := f.svc.Attachment(t.Context(), pract, c.ID, sent.ID, false); !errors.Is(err, binary.ErrScanning) {
		t.Errorf("while scanning: %v, want binary.ErrScanning", err)
	}
	f.sal30.files.set(sent.ID, binary.StatusClean, true)
	f.sal30.files.set(draft.ID, binary.StatusClean, false)
	f.repo.addPractitioner(c.ID, roleA2, 1) // joined after message 1: reads message 2, the file's
	for _, tt := range []struct {
		want      error
		id        string
		who       communication.Caller
		thumbnail bool
	}{
		{nil, sent.ID, pract, false}, {nil, sent.ID, pract, true}, {nil, sent.ID, me, true}, {nil, sent.ID, pract2, false},
		{nil, draft.ID, me, false}, {communication.ErrNotFound, draft.ID, pract, false}, {communication.ErrNotFound, draft.ID, me, true},
		{communication.ErrNotFound, sent.ID, other, false}, {communication.ErrNotFound, unknownID, me, false}, {communication.ErrNotFound, "x", me, false},
	} {
		file, err := f.svc.Attachment(t.Context(), tt.who, c.ID, tt.id, tt.thumbnail)
		if !errors.Is(err, tt.want) || (err == nil && file.ID != tt.id) {
			t.Errorf("%s opens %s (thumbnail %v): %v, want %v", tt.who.AccountID, tt.id, tt.thumbnail, err, tt.want)
		}
		if e := f.audit.last(t); tt.want == nil && (e.EntityReference != "Binary/"+tt.id || e.Action != auditevent.ActionRead || e.AgentAccountID != tt.who.AccountID) {
			t.Errorf("%s opens %s: audit %+v", tt.who.AccountID, tt.id, e)
		}
	}
	f.repo.addPractitioner(c.ID, roleA2, 2) // a later join after the file's message: replaces the earlier row below
	f.repo.mu.Lock()
	f.repo.parts[c.ID] = slices.DeleteFunc(f.repo.parts[c.ID], func(p communication.Participant) bool { return p.PractitionerRoleID == roleA2 && p.HistoryFromSeq == 1 })
	f.repo.mu.Unlock()
	if _, err := f.svc.Attachment(t.Context(), pract2, c.ID, sent.ID, false); !errors.Is(err, communication.ErrNotFound) {
		t.Errorf("a practitioner who joined after the file's message: %v, want ErrNotFound", err)
	}
	f.revoke()
	if _, err := f.svc.Attachment(t.Context(), pract, c.ID, sent.ID, false); !errors.Is(err, communication.ErrNotFound) {
		t.Errorf("staff after the revocation: %v, want ErrNotFound", err)
	}
}
