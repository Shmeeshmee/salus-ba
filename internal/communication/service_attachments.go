package communication

import (
	"context"
	"errors"
	"io"
	"slices"
	"strings"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/binary"
)

// Files are a conversation's files through binary's upload and scan pipeline (binary.Service, ADR 0010; SAL-30).
// binary keeps the rules of a file (its type from its bytes, its size, its encryption, its scan, its thumbnail); the
// chat decides who uploads, attaches and reads one, and audits it, in the transactions binary joins.
type Files interface {
	UploadChat(ctx context.Context, in binary.ChatUpload, commit func(ctx context.Context, b binary.Binary) error) (binary.Binary, error)
	AttachChat(ctx context.Context, m binary.MessageFiles) ([]binary.Binary, error)
	ChatFiles(ctx context.Context, conversationIDs, messageIDs []string) ([]binary.Binary, error)
	ChatFile(ctx context.Context, conversationID, id string) (binary.Binary, error)
	OpenChat(ctx context.Context, b binary.Binary, thumbnail bool) (binary.File, error)
	Content(ctx context.Context, f binary.File, from, to int64) (io.ReadCloser, error)
}

// Upload is a file the caller uploads to a conversation before sending the message it goes with: exactly Size bytes
// of Body (-1 without a length), and the name it had (personal data, never logged).
type Upload struct {
	Body     io.Reader
	Filename string
	Size     int64
}

// entityBinary names the files' table in the audit of a refused upload, which names no file.
const entityBinary = "binary"

// Upload stores a file in the conversation for the caller to attach to their next message, through binary's scan
// pipeline (docs/plans/chat.md, SAL-30): who may write in the conversation may upload (enter, before a byte is
// read, and again in the transaction that stores the file, so a revocation meanwhile refuses it); the file is
// quarantined until its scan, and audited as created. binary's refusals (ErrLengthRequired, ErrTooLarge,
// ErrUnsupportedType, ErrShortBody) are audited too, as a refused creation in the patient's chat.
func (s *Service) Upload(ctx context.Context, c Caller, id string, in Upload) (Attachment, error) {
	var e entered
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		var err error
		e, err = s.enter(ctx, c, id, true, false)
		return err
	})
	if err != nil {
		return Attachment{}, wrap("upload", s.refusal(ctx, c, id, err))
	}
	b, err := s.Files.UploadChat(ctx, binary.ChatUpload{
		Body: in.Body, ConversationID: id, ParticipantID: e.part.ID, OrganizationID: e.conv.OrganizationID, PatientID: e.conv.PatientID,
		Title: in.Filename, Size: in.Size,
	}, func(ctx context.Context, b binary.Binary) error {
		again, err := s.enter(ctx, c, id, true, false)
		switch {
		case err != nil:
			return err
		case again.part.ID != e.part.ID:
			return ErrForbidden
		}
		return s.record(ctx, c, auditevent.ActionCreate, auditevent.CodeCreate, "Binary/"+b.ID, "", 0)
	})
	if fileRefusal(err) {
		return Attachment{}, s.refusedUpload(ctx, c, e.conv.PatientID, err)
	}
	if err != nil {
		return Attachment{}, wrap("upload", s.refusal(ctx, c, id, err))
	}
	return attachmentOf(b), nil
}

// fileRefusal is whether err is binary's refusal of an upload's bytes.
func fileRefusal(err error) bool {
	for _, r := range []error{binary.ErrLengthRequired, binary.ErrTooLarge, binary.ErrUnsupportedType, binary.ErrShortBody} {
		if errors.Is(err, r) {
			return true
		}
	}
	return false
}

// refusedUpload audits binary's refusal err of an upload into the patient's chat, in its own transaction, and
// returns err.
func (s *Service) refusedUpload(ctx context.Context, c Caller, patientID string, err error) error {
	ev := auditevent.AuditEvent{
		Type:            auditevent.Coding{System: auditevent.SystemAuditEventType, Code: auditevent.CodeRest},
		Subtype:         auditevent.Coding{System: auditevent.SystemRestfulInteraction, Code: auditevent.CodeCreate},
		Purpose:         auditevent.Coding{System: auditevent.SystemActReason, Code: auditevent.CodeTREAT},
		Action:          auditevent.ActionCreate,
		Outcome:         auditevent.OutcomeMinorFailure,
		AgentAccountID:  c.AccountID,
		OrganizationID:  c.OrganizationID,
		EntityReference: "Patient/" + patientID,
		EntityName:      entityBinary,
	}
	if c.OrganizationID == "" {
		ev.Purpose.Code = auditevent.CodePATRQT
	}
	ctx = context.WithoutCancel(ctx) // the client may have gone mid-body
	if aerr := s.Tx.WithTx(ctx, func(ctx context.Context) error { return s.Audit.Record(ctx, ev) }); aerr != nil {
		return wrap("upload", aerr)
	}
	return err
}

// Attachment opens a file of the conversation, or its thumbnail, for a caller who may read its message (enter;
// after their history_from_seq), or, before it is sent, for its uploader only, once its scan says it is clean:
// binary.ErrScanning while it is scanned, ErrNotFound when it was rejected, is not the conversation's, or has no
// thumbnail. The read is audited and committed before the first byte; Content then streams it.
func (s *Service) Attachment(ctx context.Context, c Caller, id, attachmentID string, thumbnail bool) (binary.File, error) {
	if !validUUID(attachmentID) {
		return binary.File{}, ErrNotFound
	}
	var f binary.File
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		e, err := s.enter(ctx, c, id, false, false)
		if err != nil {
			return err
		}
		b, err := s.Files.ChatFile(ctx, id, attachmentID)
		if err == nil {
			err = s.readable(ctx, e, b)
		}
		if err == nil {
			f, err = s.Files.OpenChat(ctx, b, thumbnail)
		}
		if errors.Is(err, binary.ErrNotFound) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		return s.record(ctx, c, auditevent.ActionRead, auditevent.CodeRead, "Binary/"+b.ID, "", 0)
	})
	if err != nil {
		return binary.File{}, wrap("attachment", s.refusal(ctx, c, id, err))
	}
	return f, nil
}

// readable is nil when the caller who entered may read the file b: its uploader before it is sent; once attached,
// whoever reads its message (a later participant from their history_from_seq: the message is read only then).
func (s *Service) readable(ctx context.Context, e entered, b binary.Binary) error {
	switch {
	case b.MessageID == "" && b.ParticipantID != e.part.ID:
		return ErrNotFound
	case b.MessageID == "", e.fromSeq() == 0:
		return nil
	}
	m, err := s.Repo.Message(ctx, b.ConversationID, b.MessageID)
	switch {
	case err != nil:
		return err
	case m.Seq <= e.fromSeq():
		return ErrNotFound
	}
	return nil
}

// Content streams the bytes from to to (inclusive) of a file Attachment opened (binary.Service.Content).
func (s *Service) Content(ctx context.Context, f binary.File, from, to int64) (io.ReadCloser, error) {
	return s.Files.Content(ctx, f, from, to)
}

// normalizeSend checks a send's body and files: the body as normalizeBody has it, or empty with files; at most
// MaxAttachments ids, each a UUID.
func normalizeSend(body string, attachmentIDs []string) (string, error) {
	if len(attachmentIDs) > MaxAttachments {
		return "", invalid("attachment_ids")
	}
	for _, id := range attachmentIDs {
		if !validUUID(id) {
			return "", invalid("attachment_ids")
		}
	}
	if len(attachmentIDs) > 0 && strings.TrimSpace(body) == "" {
		return "", nil
	}
	return normalizeBody(body)
}

// attach attaches the sender's files to the message m they just sent, in the send's transaction: every one must be
// their upload to the conversation, neither attached nor rejected (ErrInvalid otherwise, and the send fails).
func (s *Service) attach(ctx context.Context, m *Message, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	if s.Files == nil {
		return errors.New("communication: no files")
	}
	files, err := s.Files.AttachChat(ctx, binary.MessageFiles{
		SentAt: m.SentAt, ConversationID: m.ConversationID, ParticipantID: m.SenderParticipantID, MessageID: m.ID, IDs: ids,
	})
	if errors.Is(err, binary.ErrNotFound) {
		return invalid("attachment_ids")
	}
	if err != nil {
		return err
	}
	m.Attachments = attachmentsOf(files)
	return nil
}

// withAttachmentsOf sets the files of the one message m.
func (s *Service) withAttachmentsOf(ctx context.Context, m *Message) error {
	one := []Message{*m}
	err := s.withAttachments(ctx, one)
	*m = one[0]
	return err
}

// withAttachments sets the files of the messages that have any, in one query, in the transaction's scope.
func (s *Service) withAttachments(ctx context.Context, msgs []Message) error {
	var convs, ids []string
	for _, m := range msgs {
		if m.AttachmentCount > 0 {
			ids = append(ids, m.ID)
			if !slices.Contains(convs, m.ConversationID) {
				convs = append(convs, m.ConversationID)
			}
		}
	}
	if len(ids) == 0 || s.Files == nil {
		return nil
	}
	files, err := s.Files.ChatFiles(ctx, convs, ids)
	if err != nil {
		return err
	}
	byMessage := map[string][]binary.Binary{}
	for _, b := range files {
		byMessage[b.MessageID] = append(byMessage[b.MessageID], b)
	}
	for i := range msgs {
		msgs[i].Attachments = attachmentsOf(byMessage[msgs[i].ID])
	}
	return nil
}

func attachmentsOf(files []binary.Binary) []Attachment {
	if len(files) == 0 {
		return nil
	}
	out := make([]Attachment, 0, len(files))
	for _, b := range files {
		out = append(out, attachmentOf(b))
	}
	return out
}

// attachmentOf is a file as a message shows it: quarantined is scanning.
func attachmentOf(b binary.Binary) Attachment {
	status := AttachmentScanning
	switch b.Status {
	case binary.StatusClean:
		status = AttachmentClean
	case binary.StatusRejected:
		status = AttachmentRejected
	}
	return Attachment{
		ID: b.ID, ContentType: b.ContentType, Filename: b.Title, Status: status, Size: b.Size, Thumbnail: b.Thumbnail.ID != "",
	}
}
