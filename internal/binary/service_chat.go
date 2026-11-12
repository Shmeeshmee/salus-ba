package binary

import (
	"context"
	"fmt"
	"io"
	"slices"
)

// The chat's files (docs/plans/chat.md, SAL-30): a file's second kind of parent is a conversation. The chat
// (communication) decides who uploads, attaches and reads a conversation's files and audits each call, in the
// transactions binary joins; binary keeps the rules of the file itself: its type from its bytes, its size, its
// encryption, its scan before anyone opens it, its thumbnail.

// chatKinds are what the chat takes: photos (lab results photographed among them) and PDFs. No video.
var chatKinds = []Kind{KindPhoto, KindDocument}

// MaxPendingChat is how many files a participant may have waiting for a message in a conversation: uploaded, neither
// sent nor rejected (a message carries at most 10). It bounds what one account stores and has scanned for nothing; a
// file never sent is deleted after 24 hours.
const MaxPendingChat = 10

// ChatUpload is one file a participant uploads to a conversation before sending the message it goes with: exactly
// Size bytes of Body (-1 when the request gave no length). OrganizationID and PatientID are the conversation's.
type ChatUpload struct {
	Body           io.Reader
	ConversationID string
	ParticipantID  string
	OrganizationID string
	PatientID      string
	Title          string
	Size           int64
}

// UploadChat stores one file of a conversation (as Upload does a questionnaire file's): its type from the first
// bytes, a photo or a PDF of at most MaxPhotoSize bytes, streamed sealed into the store outside any transaction;
// then one transaction runs commit first (the chat's checks and its audit, which name the file), stores the row
// quarantined in the conversation and enqueues its scan. ErrLengthRequired, ErrTooLarge, ErrUnsupportedType and
// ErrShortBody refuse the bytes before commit runs; commit's error is returned as it is; ErrTooManyFiles, after it,
// a participant who already has MaxPendingChat files waiting (their uploads counted one after the other). After any failure once
// stored, the object is deleted. Nothing here is audited: the chat audits the call it serves.
func (s *Service) UploadChat(ctx context.Context, in ChatUpload, commit func(ctx context.Context, b Binary) error) (Binary, error) {
	switch {
	case in.Size < 0:
		return Binary{}, ErrLengthRequired
	case in.Size > MaxPhotoSize:
		return Binary{}, ErrTooLarge
	}
	b, err := s.sealFile(ctx, in.Body, in.Size, chatKinds, Binary{
		OrganizationID: in.OrganizationID, PatientID: in.PatientID, ConversationID: in.ConversationID, ParticipantID: in.ParticipantID,
		Title: in.Title,
	})
	if err != nil {
		return Binary{}, err
	}
	// Once the bytes are read, the commit runs even when the client has gone (as Upload's).
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), afterBody)
	defer cancel()
	var out Binary
	err = s.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := commit(ctx, b); err != nil {
			return err
		}
		pending, err := s.Repo.PendingChat(ctx, b.ConversationID, b.ParticipantID)
		switch {
		case err != nil:
			return err
		case pending >= MaxPendingChat:
			return ErrTooManyFiles
		}
		if out, err = s.Repo.Insert(ctx, b, s.Clock.Now()); err != nil {
			return err
		}
		return s.Jobs.EnqueueScan(ctx, ScanArgs{BinaryID: out.ID, ConversationID: out.ConversationID, OrganizationID: out.OrganizationID})
	})
	if err != nil {
		_ = s.Store.Delete(ctx, b.ID) // the purger removes an object left behind (SPEC.md §5)
		return Binary{}, err
	}
	return out, nil
}

// AttachChat attaches the files m names to its message, in the chat's transaction and scope (the send's, under the
// conversation's lock): every one must be the sender's upload to the conversation, neither attached already nor
// rejected, else ErrNotFound and the send fails as a whole.
func (s *Service) AttachChat(ctx context.Context, m MessageFiles) ([]Binary, error) {
	ids := slices.Compact(slices.Sorted(slices.Values(m.IDs)))
	if len(ids) != len(m.IDs) {
		return nil, ErrNotFound
	}
	out, err := s.Repo.AttachToMessage(ctx, m, s.Clock.Now())
	switch {
	case err != nil:
		return nil, fmt.Errorf("binary: attach: %w", err)
	case len(out) != len(ids):
		return nil, ErrNotFound
	}
	return out, nil
}

// ChatFiles returns the files of the messages of the conversations, as the chat's scope shows them, oldest first.
func (s *Service) ChatFiles(ctx context.Context, conversationIDs, messageIDs []string) ([]Binary, error) {
	if len(messageIDs) == 0 {
		return nil, nil
	}
	out, err := s.Repo.OfMessages(ctx, conversationIDs, messageIDs)
	if err != nil {
		return nil, fmt.Errorf("binary: chat files: %w", err)
	}
	return out, nil
}

// ChatFile returns the file id of the conversation, as the chat's scope shows it; ErrNotFound otherwise.
func (s *Service) ChatFile(ctx context.Context, conversationID, id string) (Binary, error) {
	b, err := s.Repo.ByID(ctx, id)
	switch {
	case err != nil:
		return Binary{}, err
	case b.ConversationID != conversationID || conversationID == "":
		return Binary{}, ErrNotFound
	}
	return b, nil
}

// OpenChat opens a chat file the chat has let its caller read, once its scan says it is clean (ErrScanning while
// quarantined, ErrNotFound when rejected), or its thumbnail when thumbnail is set (ErrNotFound for a file without
// one); Content then streams it.
func (s *Service) OpenChat(_ context.Context, b Binary, thumbnail bool) (File, error) {
	switch {
	case b.Status == StatusQuarantined:
		return File{}, ErrScanning
	case b.Status != StatusClean, thumbnail && b.Thumbnail.ID == "":
		return File{}, ErrNotFound
	case b.KEKID != s.KEK.ID():
		return File{}, fmt.Errorf("binary: open: the data key is wrapped by another key-encryption key")
	}
	key := b.WrappedKey
	if thumbnail {
		key = b.Thumbnail.Key
		b = Binary{ID: b.Thumbnail.ID, ConversationID: b.ConversationID, ContentType: thumbnailType, Title: b.Title, Size: b.Thumbnail.Size, Status: b.Status}
	}
	dek, err := s.KEK.Unwrap(key, b.ID)
	if err != nil {
		return File{}, fmt.Errorf("binary: open: %w", err)
	}
	return File{Binary: b, dek: dek}, nil
}
