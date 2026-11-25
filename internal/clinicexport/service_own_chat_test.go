package clinicexport_test

import (
	"bytes"
	"context"
	"io"
	"slices"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/binary"
	"github.com/Shmeeshmee/salus/internal/communication"
)

// ownChat is the patient's own chat as communication.Service gives it to them: their conversations, the messages of
// each (two to a page, so the export pages through them) and the files they may open.
type ownChat struct{ o *ownSources }

func (c ownChat) List(_ context.Context, cl appointment.Caller, before *communication.ListCursor) ([]communication.Conversation, *communication.ListCursor, error) {
	c.o.call("conversations " + cl.PatientID)
	if before != nil {
		return nil, nil, nil
	}
	return c.o.convs, nil, nil
}

func (c ownChat) Messages(_ context.Context, cl appointment.Caller, id string, before *communication.Cursor, limit int) (communication.Page, error) {
	c.o.call("messages " + id)
	all := c.o.msgs[id] // newest first, as the service gives them
	from := 0
	if before != nil {
		from = int(before.Seq)
	}
	n := min(limit, 2, len(all)-from)
	page := communication.Page{Messages: all[from : from+n]}
	if from+n < len(all) {
		page.Next = &communication.Cursor{Seq: int64(from + n)}
	}
	return page, nil
}

func (c ownChat) Attachment(_ context.Context, _ appointment.Caller, id, attachmentID string, thumbnail bool) (binary.File, error) {
	c.o.call("chat file " + attachmentID)
	body, ok := c.o.chatFiles[attachmentID]
	if !ok || thumbnail {
		return binary.File{}, communication.ErrNotFound
	}
	return binary.File{Binary: binary.Binary{ID: attachmentID, Size: int64(len(body))}}, nil
}

func (c ownChat) Content(_ context.Context, f binary.File, from, to int64) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader(c.o.chatFiles[f.ID][from : to+1])), nil
}

// withChat gives the patient a conversation with clinic B: three messages, one with a clean file and one with a file
// the app cannot open.
func withChat(o *ownSources) {
	o.convs = []communication.Conversation{{ID: "k1", OrganizationID: clinicB, PatientID: "pt1"}}
	o.msgs = map[string][]communication.Message{"k1": {
		{ID: "m3", ConversationID: "k1", Seq: 3, SenderRole: communication.RolePractitioner, Kind: communication.KindText, Body: "Synthetic placeholder answer",
			Attachments: []communication.Attachment{{ID: "f1", ContentType: "image/png", Status: communication.AttachmentClean}}},
		{ID: "m2", ConversationID: "k1", Seq: 2, SenderRole: communication.RolePatient, Kind: communication.KindText, Body: "",
			Attachments: []communication.Attachment{{ID: "f2", ContentType: "image/png"}}},
		{ID: "m1", ConversationID: "k1", Seq: 1, SenderRole: communication.RolePatient, Kind: communication.KindText, Body: "Synthetic placeholder question"},
	}}
	o.chatFiles = map[string]string{"f1": "chat png"}
}

// The chat is in the export (S4): each clinic's part holds the patient's conversations with it, every message oldest
// first with its text and files, read as the patient through the chat's own reads (each audits it); a file the app
// cannot open is named but left out.
func TestOwnRecords_TheChatOfEachClinic(t *testing.T) {
	t.Parallel()
	o, own := newOwn()
	withChat(o)
	x, err := own.Export(t.Context(), "acc1", "")
	if err != nil {
		t.Fatal(err)
	}
	b := x.Parts[1]
	chat := file(t, b, "chat.json")
	first, last := bytes.Index(chat, []byte("Synthetic placeholder question")), bytes.Index(chat, []byte("Synthetic placeholder answer"))
	if first < 0 || last < first || !bytes.Contains(chat, []byte(`"conversation": "k1"`)) || !bytes.Contains(chat, []byte(`"file": "files/f1.png"`)) {
		t.Errorf("chat.json does not hold the conversation's messages oldest first with their file:\n%s", chat)
	}
	if !bytes.Equal(file(t, b, "files/f1.png"), []byte("chat png")) || slices.Contains(names(b), "files/f2.png") {
		t.Errorf("clinic B's files %v, want the clean chat file only", names(b))
	}
	if !slices.Contains(b.Holds, "1 chat conversation (chat.json)") || !slices.Contains(b.Holds, "1 file of the chat (in files/)") {
		t.Errorf("clinic B holds %q", b.Holds)
	}
	if slices.Contains(names(x.Parts[0]), "chat.json") {
		t.Error("clinic A's part holds a chat it does not have")
	}
}

// A clinic target reads the messages of that clinic's conversations only.
func TestOwnRecords_AClinicTargetReadsThatClinicsChat(t *testing.T) {
	t.Parallel()
	o, own := newOwn()
	withChat(o)
	if _, err := own.Export(t.Context(), "acc1", clinicA); err != nil {
		t.Fatal(err)
	}
	if slices.Contains(o.calls, "messages k1") {
		t.Errorf("read clinic B's chat for clinic A: %v", o.calls)
	}
}
