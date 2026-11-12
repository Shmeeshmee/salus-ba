package binary_test

import (
	"bytes"
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/binary"
	"github.com/Shmeeshmee/salus/pkg/envelope"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// seededChat is seedDB's patient in a conversation with the clinic's practitioner role, both participants, and a
// message of the patient sent at messageAt; a second conversation of the other patient. Synthetic.
type seededChat struct {
	seededDB
	conv, patientPart, practPart, message string
	otherConv, otherPart                  string
}

var messageAt = time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)

func seedChatDB(t testing.TB, owner *pgxpool.Pool) seededChat {
	t.Helper()
	ctx := t.Context()
	c := seededChat{seededDB: seedDB(t, owner)}
	var role, account string
	if err := owner.QueryRow(ctx, `SELECT r.id::text, p.account_id::text FROM practitioner_role r, patient p WHERE r.organization_id = $1 AND p.id = $2`,
		c.clinic, c.patient).Scan(&role, &account); err != nil {
		t.Fatal(err)
	}
	conversation := func(patient string) (conv, part string) {
		if err := owner.QueryRow(ctx, `INSERT INTO conversation (organization_id, patient_id, practitioner_role_id, created_by)
			VALUES ($1, $2, $3, $4) RETURNING id::text`, c.clinic, patient, role, account).Scan(&conv); err != nil {
			t.Fatal(err)
		}
		if err := owner.QueryRow(ctx, `INSERT INTO conversation_participant (conversation_id, patient_id, role, added_by)
			VALUES ($1, $2, 'patient', $3) RETURNING id::text`, conv, patient, account).Scan(&part); err != nil {
			t.Fatal(err)
		}
		return conv, part
	}
	c.conv, c.patientPart = conversation(c.patient)
	c.otherConv, c.otherPart = conversation(c.other)
	if err := owner.QueryRow(ctx, `INSERT INTO conversation_participant (conversation_id, practitioner_role_id, role, added_by)
		VALUES ($1, $2, 'practitioner', $3) RETURNING id::text`, c.conv, role, account).Scan(&c.practPart); err != nil {
		t.Fatal(err)
	}
	if err := owner.QueryRow(ctx, `INSERT INTO communication (conversation_id, organization_id, patient_id, staff_only, seq, sent_at,
			sender_participant_id, sender_account_id, sender_role, client_message_id, kind, attachment_count)
		VALUES ($1, $2, $3, false, 1, $4, $5, $6, 'patient', gen_random_uuid(), 'text', 1) RETURNING id::text`,
		c.conv, c.clinic, c.patient, messageAt, c.patientPart, account).Scan(&c.message); err != nil {
		t.Fatal(err)
	}
	return c
}

// chatFile is a file of the conversation uploaded by the participant, as UploadChat builds it.
func (c seededChat) chatFile(part string, size int64) binary.Binary {
	return binary.Binary{
		ID: binaryID(), OrganizationID: c.clinic, PatientID: c.patient, ConversationID: c.conv, ParticipantID: part,
		ContentType: "image/png", Title: "placeholder.png", KEKID: "test-1", SHA1: bytes.Repeat([]byte{3}, 20),
		WrappedKey: bytes.Repeat([]byte{1}, 60), Size: size, SegmentSize: envelope.SegmentSize,
	}
}

// inConversation is the scope the chat sets before it reads or writes a conversation's files: the caller's, then the
// conversation (app.conversation_id, set by communication).
func inConversation(scope func(context.Context, binary.Repository) error, conv string) func(context.Context, binary.Repository) error {
	return func(ctx context.Context, r binary.Repository) error {
		if err := scope(ctx, r); err != nil {
			return err
		}
		return postgres.SetLocal(ctx, "app.conversation_id", conv)
	}
}

func asChatJob(conv, org string) func(context.Context, binary.Repository) error {
	return func(ctx context.Context, r binary.Repository) error { return r.SetChatJob(ctx, conv, org) }
}

// A participant's upload is stored in its conversation's scope with its conversation and uploader, and read back
// there and in the scan job's scope of the conversation; not by another patient, the clinic without the conversation
// set, or another conversation's job.
func TestPostgres_AChatFileIsStoredInItsConversationAndSeenWhereItsMessagesAre(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	c := seedChatDB(t, owner)
	at := time.Date(2030, 1, 6, 9, 0, 0, 0, time.UTC)
	b := c.chatFile(c.patientPart, 500)
	var stored binary.Binary
	err := in(t, app, inConversation(asPatient(c.patient), c.conv), func(ctx context.Context, r binary.Repository) error {
		var err error
		stored, err = r.Insert(ctx, b, at)
		return err
	})
	if err != nil {
		t.Fatalf("Insert() = %v", err)
	}
	if stored.ConversationID != c.conv || stored.ParticipantID != c.patientPart || stored.ResponseID != "" || stored.AppointmentID != "" ||
		stored.LinkID != "" || stored.MessageID != "" || stored.Status != binary.StatusQuarantined {
		t.Errorf("Insert() = %+v, want the conversation's quarantined file", stored)
	}
	for name, sc := range map[string]struct {
		scope func(context.Context, binary.Repository) error
		sees  bool
	}{
		"the patient":                         {asPatient(c.patient), true},
		"another patient":                     {asPatient(c.other), false},
		"the clinic":                          {asClinic(c.clinic), false},
		"the clinic in the conversation":      {inConversation(asClinic(c.clinic), c.conv), true},
		"the clinic in another conversation":  {inConversation(asClinic(c.clinic), c.otherConv), false},
		"the scan job of the conversation":    {asChatJob(c.conv, c.clinic), true},
		"the scan job of another":             {asChatJob(c.otherConv, c.clinic), false},
		"a questionnaire job of the response": {asJob(c.response, c.clinic), false},
	} {
		_ = in(t, app, sc.scope, func(ctx context.Context, r binary.Repository) error {
			got, err := r.ByID(ctx, b.ID)
			switch {
			case sc.sees && (err != nil || got.ConversationID != c.conv):
				t.Errorf("%s: ByID() = %+v, %v; want the file", name, got, err)
			case !sc.sees && !errors.Is(err, binary.ErrNotFound):
				t.Errorf("%s: ByID() = %v, want ErrNotFound", name, err)
			}
			return nil
		})
	}
	err = in(t, app, inConversation(asPatient(c.other), c.otherConv), func(ctx context.Context, r binary.Repository) error {
		_, err := r.Insert(ctx, c.chatFile(c.patientPart, 10), at) // patient A's file into patient B's scope
		return err
	})
	if err == nil {
		t.Error("Insert() of another conversation's file = nil, want row-level security's refusal")
	}
}

// A send attaches exactly the sender's files of the conversation that are neither attached nor rejected, once; the
// page's files are listed by their messages.
func TestPostgres_AttachToMessageAttachesTheSendersFreeFilesOnce(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	c := seedChatDB(t, owner)
	mine, theirs, rejected := c.chatFile(c.patientPart, 10), c.chatFile(c.practPart, 20), c.chatFile(c.patientPart, 30)
	scope := inConversation(asPatient(c.patient), c.conv)
	err := in(t, app, scope, func(ctx context.Context, r binary.Repository) error {
		for _, b := range []binary.Binary{mine, theirs, rejected} {
			if _, err := r.Insert(ctx, b, now); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := owner.Exec(t.Context(), `UPDATE "binary" SET status = 'rejected', scanned_at = now(), wrapped_key = NULL WHERE id = $1`, rejected.ID); err != nil {
		t.Fatal(err)
	}
	attach := binary.MessageFiles{ConversationID: c.conv, ParticipantID: c.patientPart, MessageID: c.message, SentAt: messageAt,
		IDs: []string{mine.ID, theirs.ID, rejected.ID, unknownID}}
	for i, want := range [][]string{{mine.ID}, nil} {
		err := in(t, app, scope, func(ctx context.Context, r binary.Repository) error {
			got, err := r.AttachToMessage(ctx, attach, now.Add(time.Minute))
			var ids []string
			for _, b := range got {
				ids = append(ids, b.ID)
				if b.MessageID != c.message || !b.MessageSentAt.Equal(messageAt) {
					t.Errorf("AttachToMessage() #%d: %+v, want it attached to the message", i+1, b)
				}
			}
			if !slices.Equal(ids, want) {
				t.Errorf("AttachToMessage() #%d = %v, want %v", i+1, ids, want)
			}
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	for name, sc := range map[string]func(context.Context, binary.Repository) error{
		"the patient":                    asPatient(c.patient),
		"the clinic in the conversation": inConversation(asClinic(c.clinic), c.conv),
	} {
		_ = in(t, app, sc, func(ctx context.Context, r binary.Repository) error {
			got, err := r.OfMessages(ctx, []string{c.conv}, []string{c.message, unknownID})
			if err != nil || len(got) != 1 || got[0].ID != mine.ID {
				t.Errorf("%s: OfMessages() = %+v, %v; want the attached file", name, got, err)
			}
			return nil
		})
	}
}

// The scan job of a conversation marks its file and stores an image's thumbnail once it is clean, once.
func TestPostgres_TheChatJobMarksAFileScannedAndStoresItsThumbnailOnce(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	c := seedChatDB(t, owner)
	b := c.chatFile(c.patientPart, 10)
	if err := in(t, app, inConversation(asPatient(c.patient), c.conv), func(ctx context.Context, r binary.Repository) error {
		_, err := r.Insert(ctx, b, now)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	thumb := binary.Thumbnail{ID: binaryID(), Key: bytes.Repeat([]byte{5}, 60), Size: 99}
	err := in(t, app, asChatJob(c.conv, c.clinic), func(ctx context.Context, r binary.Repository) error {
		if ok, err := r.SetThumbnail(ctx, b.ID, thumb, now); err != nil || ok {
			t.Errorf("SetThumbnail() while quarantined = %v, %v; want false", ok, err)
		}
		if ok, err := r.MarkScanned(ctx, b.ID, binary.StatusClean, now); err != nil || !ok {
			t.Errorf("MarkScanned() = %v, %v; want true", ok, err)
		}
		for i, want := range []bool{true, false} {
			if ok, err := r.SetThumbnail(ctx, b.ID, thumb, now); err != nil || ok != want {
				t.Errorf("SetThumbnail() #%d = %v, %v; want %v", i+1, ok, err, want)
			}
		}
		got, err := r.ByID(ctx, b.ID)
		if err != nil || got.Thumbnail.ID != thumb.ID || got.Thumbnail.Size != 99 || !bytes.Equal(got.Thumbnail.Key, thumb.Key) {
			t.Errorf("ByID() = %+v, %v; want the thumbnail", got.Thumbnail, err)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// The purger deletes chat files never attached 24 h after their upload, and keeps the attached ones; a thumbnail's
// object exists while its file's row does.
func TestPostgres_TheRetentionRoleDeletesOldUnattachedChatFilesAndKnowsThumbnails(t *testing.T) {
	t.Parallel()
	owner, ret := pgtest.NewRetentionDatabase(t)
	c := seedChatDB(t, owner)
	cut := now.Add(-24 * time.Hour)
	old, fresh, attached := c.chatFile(c.patientPart, 10), c.chatFile(c.patientPart, 20), c.chatFile(c.patientPart, 30)
	thumb := binaryID()
	for _, f := range []struct {
		message any
		at      time.Time
		b       binary.Binary
	}{{nil, cut.Add(-time.Second), old}, {nil, cut.Add(time.Second), fresh}, {c.message, cut.Add(-time.Hour), attached}} {
		if _, err := owner.Exec(t.Context(), `INSERT INTO "binary" (id, organization_id, patient_id, conversation_id, conversation_participant_id,
				content_type, size, title, sha1, kek_id, wrapped_key, segment_size, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $13)`, f.b.ID, f.b.OrganizationID, f.b.PatientID, f.b.ConversationID,
			f.b.ParticipantID, f.b.ContentType, f.b.Size, f.b.Title, f.b.SHA1, f.b.KEKID, f.b.WrappedKey, f.b.SegmentSize, f.at); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := owner.Exec(t.Context(), `UPDATE "binary" SET communication_id = $2, communication_sent_at = $3, status = 'clean', scanned_at = now(),
		thumbnail_id = $4, thumbnail_size = 9, thumbnail_wrapped_key = '\x01' WHERE id = $1`, attached.ID, c.message, messageAt, thumb); err != nil {
		t.Fatal(err)
	}
	repo := binary.NewPostgresRepository(ret)
	if n, err := repo.DeleteUnattachedBefore(t.Context(), cut); err != nil || n != 1 {
		t.Errorf("DeleteUnattachedBefore() = %d, %v; want 1", n, err)
	}
	got, err := repo.Existing(t.Context(), []string{old.ID, fresh.ID, attached.ID, thumb})
	slices.Sort(got)
	want := []string{fresh.ID, attached.ID, thumb}
	slices.Sort(want)
	if err != nil || !slices.Equal(got, want) {
		t.Errorf("Existing() = %v, %v; want %v", got, err, want)
	}
}

// PendingChat counts the participant's files of the conversation that wait for a message: neither attached nor
// rejected; another participant's, an attached and a rejected one do not count.
func TestPostgres_PendingChatCountsTheParticipantsUnsentFiles(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	c := seedChatDB(t, owner)
	scope := inConversation(asPatient(c.patient), c.conv)
	free, attached, rejected, theirs := c.chatFile(c.patientPart, 10), c.chatFile(c.patientPart, 20), c.chatFile(c.patientPart, 30),
		c.chatFile(c.practPart, 40)
	err := in(t, app, scope, func(ctx context.Context, r binary.Repository) error {
		for _, b := range []binary.Binary{free, attached, rejected, theirs} {
			if _, err := r.Insert(ctx, b, now); err != nil {
				return err
			}
		}
		_, err := r.AttachToMessage(ctx, binary.MessageFiles{ConversationID: c.conv, ParticipantID: c.patientPart, MessageID: c.message,
			SentAt: messageAt, IDs: []string{attached.ID}}, now)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := owner.Exec(t.Context(), `UPDATE "binary" SET status = 'rejected', scanned_at = now(), wrapped_key = NULL WHERE id = $1`, rejected.ID); err != nil {
		t.Fatal(err)
	}
	for part, want := range map[string]int64{c.patientPart: 1, c.practPart: 1, c.otherPart: 0} {
		_ = in(t, app, scope, func(ctx context.Context, r binary.Repository) error {
			if n, err := r.PendingChat(ctx, c.conv, part); err != nil || n != want {
				t.Errorf("PendingChat(%s) = %d, %v; want %d", part, n, err, want)
			}
			return nil
		})
	}
}
