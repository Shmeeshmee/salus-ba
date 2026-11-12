package binary

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Shmeeshmee/salus/internal/binary/binarydb"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

type postgresRepository struct {
	db postgres.Querier
}

// NewPostgresRepository returns the Repository on db (the salus_app pool), using the transaction in the
// context when there is one; row-level security shows only the scope set.
func NewPostgresRepository(db postgres.Querier) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) q(ctx context.Context) *binarydb.Queries {
	return binarydb.New(postgres.Conn(ctx, r.db))
}

func (r *postgresRepository) SetPatient(ctx context.Context, id string) error {
	return postgres.SetLocal(ctx, "app.patient_id", id)
}

func (r *postgresRepository) SetClinic(ctx context.Context, id string) error {
	return postgres.SetLocal(ctx, "app.organization_id", id)
}

func (r *postgresRepository) SetJob(ctx context.Context, responseID, org string) error {
	if err := postgres.SetLocal(ctx, "app.questionnaire_response_id", responseID); err != nil {
		return err
	}
	return postgres.SetLocal(ctx, "app.organization_id", org)
}

func (r *postgresRepository) LockResponse(ctx context.Context, id string) error {
	return r.q(ctx).LockResponseBinaries(ctx, id)
}

func (r *postgresRepository) Insert(ctx context.Context, b Binary, at time.Time) (Binary, error) {
	p := binarydb.InsertBinaryParams{
		LinkID: pgtype.Text{String: b.LinkID, Valid: b.LinkID != ""}, ContentType: b.ContentType, Size: b.Size, Title: b.Title,
		Sha1: b.SHA1, KekID: b.KEKID, WrappedKey: b.WrappedKey, SegmentSize: b.SegmentSize, At: pgtype.Timestamptz{Time: at, Valid: true},
	}
	if p.ID.Scan(b.ID) != nil || p.OrganizationID.Scan(b.OrganizationID) != nil || p.PatientID.Scan(b.PatientID) != nil ||
		optional(&p.AppointmentID, b.AppointmentID) != nil || optional(&p.QuestionnaireResponseID, b.ResponseID) != nil ||
		optional(&p.ConversationID, b.ConversationID) != nil || optional(&p.ConversationParticipantID, b.ParticipantID) != nil {
		return Binary{}, errors.New("binary: an id is not a UUID")
	}
	row, err := r.q(ctx).InsertBinary(ctx, p)
	if err != nil {
		return Binary{}, err
	}
	return toBinary(row), nil
}

func (r *postgresRepository) ByID(ctx context.Context, id string) (Binary, error) {
	var u pgtype.UUID
	if u.Scan(id) != nil {
		return Binary{}, ErrNotFound
	}
	row, err := r.q(ctx).BinaryByID(ctx, u)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return Binary{}, ErrNotFound
	case err != nil:
		return Binary{}, err
	}
	return toBinary(row), nil
}

func (r *postgresRepository) OfItem(ctx context.Context, responseID, linkID string) ([]Binary, error) {
	var u pgtype.UUID
	if u.Scan(responseID) != nil {
		return nil, nil
	}
	rows, err := r.q(ctx).BinariesOfItem(ctx, binarydb.BinariesOfItemParams{QuestionnaireResponseID: u, LinkID: pgtype.Text{String: linkID, Valid: true}})
	if err != nil {
		return nil, err
	}
	out := make([]Binary, 0, len(rows))
	for _, row := range rows {
		out = append(out, toBinary(row))
	}
	return out, nil
}

func (r *postgresRepository) ResponseSize(ctx context.Context, responseID string) (int64, error) {
	var u pgtype.UUID
	if u.Scan(responseID) != nil {
		return 0, nil
	}
	return r.q(ctx).ResponseBinarySize(ctx, u)
}

func (r *postgresRepository) Delete(ctx context.Context, id string) (bool, error) {
	var u pgtype.UUID
	if u.Scan(id) != nil {
		return false, nil
	}
	n, err := r.q(ctx).DeleteBinary(ctx, u)
	return n == 1, err
}

func (r *postgresRepository) MarkScanned(ctx context.Context, id string, status Status, at time.Time) (bool, error) {
	var u pgtype.UUID
	if u.Scan(id) != nil {
		return false, nil
	}
	n, err := r.q(ctx).MarkBinaryScanned(ctx, binarydb.MarkBinaryScannedParams{Status: string(status), At: pgtype.Timestamptz{Time: at, Valid: true}, ID: u})
	return n == 1, err
}

func (r *postgresRepository) OfResponse(ctx context.Context, responseID string) ([]Binary, error) {
	var u pgtype.UUID
	if u.Scan(responseID) != nil {
		return nil, nil
	}
	rows, err := r.q(ctx).BinariesOfResponse(ctx, u)
	if err != nil {
		return nil, err
	}
	var out []Binary
	for _, row := range rows {
		out = append(out, toBinary(row))
	}
	return out, nil
}

func (r *postgresRepository) DeleteRejectedBefore(ctx context.Context, before time.Time) (int64, error) {
	return r.q(ctx).DeleteRejectedBinariesBefore(ctx, pgtype.Timestamptz{Time: before, Valid: true})
}

func (r *postgresRepository) Existing(ctx context.Context, ids []string) ([]string, error) {
	keys := uuids(ids)
	if len(keys) == 0 {
		return nil, nil
	}
	rows, err := r.q(ctx).ExistingBinaries(ctx, keys)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(rows))
	for _, u := range rows {
		out = append(out, u.String())
	}
	return out, nil
}

func (r *postgresRepository) SetChatJob(ctx context.Context, conversationID, org string) error {
	if err := postgres.SetLocal(ctx, "app.binary_conversation_id", conversationID); err != nil {
		return err
	}
	return postgres.SetLocal(ctx, "app.organization_id", org)
}

func (r *postgresRepository) AttachToMessage(ctx context.Context, m MessageFiles, at time.Time) ([]Binary, error) {
	p := binarydb.AttachChatBinariesParams{
		CommunicationSentAt: pgtype.Timestamptz{Time: m.SentAt, Valid: true}, At: pgtype.Timestamptz{Time: at, Valid: true}, Ids: uuids(m.IDs),
	}
	if p.CommunicationID.Scan(m.MessageID) != nil || p.ConversationID.Scan(m.ConversationID) != nil || p.ConversationParticipantID.Scan(m.ParticipantID) != nil {
		return nil, errors.New("binary: an id is not a UUID")
	}
	rows, err := r.q(ctx).AttachChatBinaries(ctx, p)
	if err != nil {
		return nil, err
	}
	return toBinaries(rows), nil
}

func (r *postgresRepository) PendingChat(ctx context.Context, conversationID, participantID string) (int64, error) {
	var conv, part pgtype.UUID
	if conv.Scan(conversationID) != nil || part.Scan(participantID) != nil {
		return 0, nil
	}
	q := r.q(ctx)
	if err := q.LockChatUploads(ctx, participantID); err != nil {
		return 0, err
	}
	return q.PendingChatBinaries(ctx, binarydb.PendingChatBinariesParams{ConversationID: conv, ConversationParticipantID: part})
}

func (r *postgresRepository) OfMessages(ctx context.Context, conversationIDs, messageIDs []string) ([]Binary, error) {
	rows, err := r.q(ctx).ChatBinariesOfMessages(ctx, binarydb.ChatBinariesOfMessagesParams{
		ConversationIds: uuids(conversationIDs), CommunicationIds: uuids(messageIDs),
	})
	if err != nil {
		return nil, err
	}
	return toBinaries(rows), nil
}

func (r *postgresRepository) SetThumbnail(ctx context.Context, id string, t Thumbnail, at time.Time) (bool, error) {
	p := binarydb.SetBinaryThumbnailParams{
		ThumbnailSize: pgtype.Int8{Int64: t.Size, Valid: true}, ThumbnailWrappedKey: t.Key, At: pgtype.Timestamptz{Time: at, Valid: true},
	}
	if p.ID.Scan(id) != nil || p.ThumbnailID.Scan(t.ID) != nil {
		return false, nil
	}
	n, err := r.q(ctx).SetBinaryThumbnail(ctx, p)
	return n == 1, err
}

func (r *postgresRepository) DeleteUnattachedBefore(ctx context.Context, before time.Time) (int64, error) {
	return r.q(ctx).DeleteUnattachedChatBinariesBefore(ctx, pgtype.Timestamptz{Time: before, Valid: true})
}

// optional scans the id into u, leaving it null for "".
func optional(u *pgtype.UUID, id string) error {
	if id == "" {
		return nil
	}
	return u.Scan(id)
}

// uuids are the ids that are UUIDs; another matches no row.
func uuids(ids []string) []pgtype.UUID {
	out := make([]pgtype.UUID, 0, len(ids))
	for _, id := range ids {
		var u pgtype.UUID
		if u.Scan(id) == nil {
			out = append(out, u)
		}
	}
	return out
}

func toBinaries(rows []binarydb.Binary) []Binary {
	out := make([]Binary, 0, len(rows))
	for _, row := range rows {
		out = append(out, toBinary(row))
	}
	return out
}

// toBinary maps a row to the model; a null id is "".
func toBinary(row binarydb.Binary) Binary {
	b := Binary{
		CreatedAt: row.CreatedAt.Time, UpdatedAt: row.UpdatedAt.Time, ScannedAt: row.ScannedAt.Time, MessageSentAt: row.CommunicationSentAt.Time,
		ID: row.ID.String(), OrganizationID: row.OrganizationID.String(), PatientID: row.PatientID.String(),
		AppointmentID: idOf(row.AppointmentID), ResponseID: idOf(row.QuestionnaireResponseID), LinkID: row.LinkID.String,
		ConversationID: idOf(row.ConversationID), ParticipantID: idOf(row.ConversationParticipantID), MessageID: idOf(row.CommunicationID),
		ContentType: row.ContentType, Title: row.Title, KEKID: row.KekID, Status: Status(row.Status), SHA1: row.Sha1,
		WrappedKey: row.WrappedKey, Size: row.Size, SegmentSize: row.SegmentSize,
	}
	if row.ThumbnailID.Valid {
		b.Thumbnail = Thumbnail{ID: row.ThumbnailID.String(), Key: row.ThumbnailWrappedKey, Size: row.ThumbnailSize.Int64}
	}
	return b
}

func idOf(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	return u.String()
}
