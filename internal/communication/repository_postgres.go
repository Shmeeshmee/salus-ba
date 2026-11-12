package communication

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Shmeeshmee/salus/internal/communication/communicationdb"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

type postgresRepository struct {
	db postgres.Querier
}

// NewPostgresRepository returns the Repository on db, using the transaction in the context when there is one.
func NewPostgresRepository(db postgres.Querier) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) q(ctx context.Context) *communicationdb.Queries {
	return communicationdb.New(postgres.Conn(ctx, r.db))
}

func (r *postgresRepository) SetPatient(ctx context.Context, id string) error {
	return postgres.SetLocal(ctx, "app.patient_id", id)
}

func (r *postgresRepository) SetClinic(ctx context.Context, id string) error {
	return postgres.SetLocal(ctx, "app.organization_id", id)
}

func (r *postgresRepository) SetConversation(ctx context.Context, id string) error {
	return postgres.SetLocal(ctx, "app.conversation_id", id)
}

func (r *postgresRepository) SetPractitionerRole(ctx context.Context, id string) error {
	return postgres.SetLocal(ctx, "app.practitioner_role_id", id)
}

func (r *postgresRepository) CreateConversation(ctx context.Context, c Conversation, parts []Participant) (Conversation, bool, error) {
	q := r.q(ctx)
	row, err := q.CreateConversation(ctx, communicationdb.CreateConversationParams{
		OrganizationID: uid(c.OrganizationID), PatientID: uid(c.PatientID), PractitionerRoleID: uid(c.PractitionerRoleID),
		EncounterID: optUID(c.EncounterID), CreatedBy: uid(c.CreatedBy), CreatedAt: ts(c.Created),
	})
	created := err == nil
	if errors.Is(err, pgx.ErrNoRows) { // the one for this care exists
		row, err = q.CareConversation(ctx, communicationdb.CareConversationParams{
			PatientID: uid(c.PatientID), PractitionerRoleID: uid(c.PractitionerRoleID), EncounterID: optUID(c.EncounterID),
		})
	}
	if err != nil {
		return Conversation{}, false, fmt.Errorf("communication: create conversation: %w", err)
	}
	out := conversation(row)
	if created {
		for _, p := range parts {
			p.ConversationID = out.ID
			if _, err := r.InsertParticipant(ctx, p); err != nil {
				return Conversation{}, false, err
			}
		}
	}
	if out.Participants, err = r.Participants(ctx, out.ID); err != nil {
		return Conversation{}, false, err
	}
	return out, created, nil
}

func (r *postgresRepository) Conversation(ctx context.Context, id string, lock bool) (Conversation, error) {
	get := r.q(ctx).Conversation
	if lock {
		get = r.q(ctx).ConversationForUpdate
	}
	row, err := get(ctx, uid(id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Conversation{}, ErrNotFound
	}
	if err != nil {
		return Conversation{}, fmt.Errorf("communication: conversation: %w", err)
	}
	return conversation(row), nil
}

func (r *postgresRepository) Participants(ctx context.Context, conversationID string) ([]Participant, error) {
	rows, err := r.q(ctx).Participants(ctx, uid(conversationID))
	if err != nil {
		return nil, fmt.Errorf("communication: participants: %w", err)
	}
	out := make([]Participant, 0, len(rows))
	for _, p := range rows {
		out = append(out, participant(p))
	}
	return out, nil
}

func participant(p communicationdb.ParticipantsRow) Participant {
	return Participant{
		AddedAt: p.AddedAt.Time, ID: str(p.ID), ConversationID: str(p.ConversationID), PatientID: str(p.PatientID),
		PractitionerRoleID: str(p.PractitionerRoleID), OrganizationID: str(p.OrganizationID), AddedBy: str(p.AddedBy), Role: Role(p.Role),
		Status: ParticipantStatus(p.Status), Reason: Reason(p.Reason.String), HistoryFromSeq: p.HistoryFromSeq, ReadSeq: p.ReadSeq,
	}
}

func (r *postgresRepository) Conversations(ctx context.Context, f ListFilter) ([]Conversation, error) {
	arg := communicationdb.ConversationsOfParticipantParams{
		PatientID: optUID(f.PatientID), PractitionerRoleID: optUID(f.PractitionerRoleID), Lim: int32(f.Limit), //nolint:gosec // at most ListSize + 1
	}
	if f.Before != nil {
		arg.BeforeAt, arg.BeforeID = ts(f.Before.At), uid(f.Before.ID)
	}
	rows, err := r.q(ctx).ConversationsOfParticipant(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("communication: conversations: %w", err)
	}
	out := make([]Conversation, 0, len(rows))
	for _, row := range rows {
		out = append(out, withUnread(row))
	}
	return out, nil
}

// withUnread is the conversation of a participant's list, with their unread count.
func withUnread(row communicationdb.ConversationsOfParticipantRow) Conversation {
	c := conversation(communicationdb.Conversation{
		ID: row.ID, CreatedAt: row.CreatedAt, LastSentAt: row.LastSentAt, OrganizationID: row.OrganizationID, PatientID: row.PatientID,
		PractitionerRoleID: row.PractitionerRoleID, EncounterID: row.EncounterID, ParentID: row.ParentID, CreatedBy: row.CreatedBy,
		LastSeq: row.LastSeq, StaffOnly: row.StaffOnly,
	})
	c.Unread = max(c.LastSeq-row.ReadSeq, 0)
	return c
}

func (r *postgresRepository) Messages(ctx context.Context, conversationID string, before *Cursor, fromSeq int64, limit int) ([]Message, error) {
	var rows []communicationdb.Communication
	var err error
	lim := int32(limit) //nolint:gosec // at most MaxPageSize + 1
	if before == nil {
		rows, err = r.q(ctx).LatestMessages(ctx, communicationdb.LatestMessagesParams{ConversationID: uid(conversationID), FromSeq: fromSeq, Lim: lim})
	} else {
		rows, err = r.q(ctx).MessagesBefore(ctx, communicationdb.MessagesBeforeParams{
			ConversationID: uid(conversationID), FromSeq: fromSeq, SentAt: ts(before.SentAt), Seq: before.Seq, Lim: lim,
		})
	}
	if err != nil {
		return nil, fmt.Errorf("communication: messages: %w", err)
	}
	return messages(rows)
}

func (r *postgresRepository) After(ctx context.Context, conversationID string, after Cursor, fromSeq int64, limit int) ([]Message, error) {
	rows, err := r.q(ctx).MessagesAfter(ctx, communicationdb.MessagesAfterParams{
		ConversationID: uid(conversationID), FromSeq: fromSeq, SentAt: ts(after.SentAt), Seq: after.Seq, Lim: int32(limit), //nolint:gosec // ReplayPage
	})
	if err != nil {
		return nil, fmt.Errorf("communication: messages after: %w", err)
	}
	return messages(rows)
}

func (r *postgresRepository) Message(ctx context.Context, conversationID, id string) (Message, error) {
	row, err := r.q(ctx).Message(ctx, communicationdb.MessageParams{ConversationID: uid(conversationID), ID: uid(id)})
	return one(row, err)
}

func (r *postgresRepository) MessageByClientID(ctx context.Context, conversationID, participantID, clientMessageID string, since time.Time) (Message, error) {
	row, err := r.q(ctx).MessageByClientID(ctx, communicationdb.MessageByClientIDParams{
		ConversationID: uid(conversationID), ClientMessageID: uid(clientMessageID), SenderParticipantID: uid(participantID), Since: ts(since),
	})
	return one(row, err)
}

func (r *postgresRepository) InsertMessage(ctx context.Context, m Message) (Message, error) {
	var args []byte
	if m.SystemArgs != nil {
		args, _ = json.Marshal(m.SystemArgs) // a map of strings always encodes
	}
	id, err := r.q(ctx).InsertMessage(ctx, communicationdb.InsertMessageParams{
		SentAt: ts(m.SentAt), ConversationID: uid(m.ConversationID), OrganizationID: uid(m.OrganizationID), PatientID: uid(m.PatientID),
		SenderParticipantID: optUID(m.SenderParticipantID), SenderAccountID: optUID(m.SenderAccountID),
		ClientMessageID: uid(m.ClientMessageID), Seq: m.Seq, StaffOnly: m.StaffOnly, SenderRole: string(m.SenderRole),
		Kind: string(m.Kind), Body: text(m.Body), SystemCode: text(m.SystemCode), SystemArgs: args,
	})
	if err != nil {
		return Message{}, fmt.Errorf("communication: insert message: %w", err)
	}
	m.ID, m.VersionID = str(id), 1
	return m, nil
}

func (r *postgresRepository) Advance(ctx context.Context, conversationID string, seq int64, sentAt time.Time) error {
	if err := r.q(ctx).AdvanceConversation(ctx, communicationdb.AdvanceConversationParams{ID: uid(conversationID), LastSeq: seq, LastSentAt: ts(sentAt)}); err != nil {
		return fmt.Errorf("communication: advance: %w", err)
	}
	return nil
}

func (r *postgresRepository) MarkRead(ctx context.Context, participantID string, seq int64) error {
	if err := r.q(ctx).MarkRead(ctx, communicationdb.MarkReadParams{ID: uid(participantID), ReadSeq: seq}); err != nil {
		return fmt.Errorf("communication: mark read: %w", err)
	}
	return nil
}

func (r *postgresRepository) UpdateBody(ctx context.Context, m Message) error {
	n, err := r.q(ctx).UpdateBody(ctx, communicationdb.UpdateBodyParams{
		Body: text(m.Body), VersionID: m.VersionID, EditedAt: ts(m.EditedAt), ConversationID: uid(m.ConversationID), ID: uid(m.ID),
		PreviousVersion: m.VersionID - 1,
	})
	switch {
	case err != nil:
		return fmt.Errorf("communication: update body: %w", err)
	case n == 0:
		return ErrVersionConflict
	}
	return nil
}

func (r *postgresRepository) Revisions(ctx context.Context, conversationID, id string) ([]Revision, error) {
	rows, err := r.q(ctx).Revisions(ctx, communicationdb.RevisionsParams{ConversationID: uid(conversationID), CommunicationID: uid(id)})
	if err != nil {
		return nil, fmt.Errorf("communication: revisions: %w", err)
	}
	out := make([]Revision, 0, len(rows))
	for _, row := range rows {
		out = append(out, Revision{EditedAt: row.EditedAt.Time, Body: row.Body, VersionID: row.VersionID})
	}
	return out, nil
}

func (r *postgresRepository) AddPartitions(ctx context.Context, from time.Time, months int) error {
	if err := r.q(ctx).AddPartitions(ctx, communicationdb.AddPartitionsParams{
		FromDay: pgtype.Date{Time: from.UTC(), Valid: true}, Months: int32(months), //nolint:gosec // PartitionsAhead
	}); err != nil {
		return fmt.Errorf("communication: add partitions: %w", err)
	}
	return nil
}

func (r *postgresRepository) Participant(ctx context.Context, id string) (Participant, error) {
	row, err := r.q(ctx).ParticipantByID(ctx, uid(id))
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return Participant{}, ErrNotFound
	case err != nil:
		return Participant{}, fmt.Errorf("communication: participant: %w", err)
	}
	return participant(communicationdb.ParticipantsRow(row)), nil
}

func (r *postgresRepository) InsertParticipant(ctx context.Context, p Participant) (Participant, error) {
	row, err := r.q(ctx).InsertParticipant(ctx, communicationdb.InsertParticipantParams{
		ConversationID: uid(p.ConversationID), PatientID: optUID(p.PatientID), PractitionerRoleID: optUID(p.PractitionerRoleID),
		OrganizationID: optUID(p.OrganizationID), Role: string(p.Role), Status: string(p.Status), Reason: text(string(p.Reason)),
		HistoryFromSeq: p.HistoryFromSeq, AddedBy: uid(p.AddedBy), AddedAt: ts(p.AddedAt),
	})
	if err != nil {
		return Participant{}, fmt.Errorf("communication: add participant: %w", err)
	}
	return participant(communicationdb.ParticipantsRow(row)), nil
}

func (r *postgresRepository) ActivateParticipant(ctx context.Context, id string) error {
	if err := r.q(ctx).ActivateParticipant(ctx, uid(id)); err != nil {
		return fmt.Errorf("communication: activate participant: %w", err)
	}
	return nil
}

func (r *postgresRepository) RemoveParticipant(ctx context.Context, id string, at time.Time) error {
	if err := r.q(ctx).RemoveParticipant(ctx, communicationdb.RemoveParticipantParams{ID: uid(id), RemovedAt: ts(at)}); err != nil {
		return fmt.Errorf("communication: remove participant: %w", err)
	}
	return nil
}

func (r *postgresRepository) CreateThread(ctx context.Context, c Conversation, parts []Participant) (Conversation, error) {
	row, err := r.q(ctx).InsertThread(ctx, communicationdb.InsertThreadParams{
		OrganizationID: uid(c.OrganizationID), PatientID: uid(c.PatientID), PractitionerRoleID: uid(c.PractitionerRoleID),
		EncounterID: optUID(c.EncounterID), ParentID: uid(c.ParentID), CreatedBy: uid(c.CreatedBy), CreatedAt: ts(c.Created),
	})
	if err != nil {
		return Conversation{}, fmt.Errorf("communication: open thread: %w", err)
	}
	out := conversation(row)
	for _, p := range parts {
		p.ConversationID = out.ID
		added, err := r.InsertParticipant(ctx, p)
		if err != nil {
			return Conversation{}, err
		}
		out.Participants = append(out.Participants, added)
	}
	return out, nil
}

func (r *postgresRepository) Threads(ctx context.Context, parentID, roleID string) ([]Conversation, error) {
	rows, err := r.q(ctx).ThreadsOfParticipant(ctx, communicationdb.ThreadsOfParticipantParams{ParentID: uid(parentID), PractitionerRoleID: uid(roleID)})
	if err != nil {
		return nil, fmt.Errorf("communication: threads: %w", err)
	}
	out := make([]Conversation, 0, len(rows))
	for _, row := range rows {
		out = append(out, withUnread(communicationdb.ConversationsOfParticipantRow(row)))
	}
	return out, nil
}

func (r *postgresRepository) Invitations(ctx context.Context, patientID string) ([]Invitation, error) {
	rows, err := r.q(ctx).Invitations(ctx, uid(patientID))
	if err != nil {
		return nil, fmt.Errorf("communication: invitations: %w", err)
	}
	out := make([]Invitation, 0, len(rows))
	for _, row := range rows {
		out = append(out, Invitation{
			InvitedAt: row.AddedAt.Time, ID: str(row.ID), ConversationID: str(row.ConversationID), OrganizationID: str(row.OrganizationID),
			PractitionerRoleID: str(row.PractitionerRoleID), Reason: Reason(row.Reason.String),
		})
	}
	return out, nil
}

func conversation(c communicationdb.Conversation) Conversation {
	return Conversation{
		Created: c.CreatedAt.Time, LastSentAt: c.LastSentAt.Time, ID: str(c.ID), OrganizationID: str(c.OrganizationID),
		PatientID: str(c.PatientID), PractitionerRoleID: str(c.PractitionerRoleID), EncounterID: str(c.EncounterID),
		ParentID: str(c.ParentID), CreatedBy: str(c.CreatedBy), LastSeq: c.LastSeq, StaffOnly: c.StaffOnly,
	}
}

func one(row communicationdb.Communication, err error) (Message, error) {
	if errors.Is(err, pgx.ErrNoRows) {
		return Message{}, ErrNotFound
	}
	if err != nil {
		return Message{}, fmt.Errorf("communication: message: %w", err)
	}
	return message(row)
}

func messages(rows []communicationdb.Communication) ([]Message, error) {
	out := make([]Message, 0, len(rows))
	for _, row := range rows {
		m, err := message(row)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, nil
}

func message(c communicationdb.Communication) (Message, error) {
	m := Message{
		SentAt: c.SentAt.Time, EditedAt: c.EditedAt.Time, ID: str(c.ID), ConversationID: str(c.ConversationID),
		OrganizationID: str(c.OrganizationID), PatientID: str(c.PatientID), SenderParticipantID: str(c.SenderParticipantID),
		SenderAccountID: str(c.SenderAccountID), ClientMessageID: str(c.ClientMessageID), Body: c.Body.String,
		SystemCode: c.SystemCode.String, SenderRole: Role(c.SenderRole), Kind: Kind(c.Kind), Seq: c.Seq, VersionID: c.VersionID,
		StaffOnly: c.StaffOnly,
	}
	if len(c.SystemArgs) > 0 {
		if err := json.Unmarshal(c.SystemArgs, &m.SystemArgs); err != nil {
			return Message{}, fmt.Errorf("communication: system args: %w", err)
		}
	}
	return m, nil
}

// uid is the UUID of a validated id (the service checks ids before they reach the repository); an invalid one is
// null, which matches no row.
func uid(s string) pgtype.UUID {
	var u pgtype.UUID
	_ = u.Scan(s)
	return u
}

// optUID is null for "".
func optUID(s string) pgtype.UUID {
	if s == "" {
		return pgtype.UUID{}
	}
	return uid(s)
}

func str(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	return u.String()
}

func text(s string) pgtype.Text { return pgtype.Text{String: s, Valid: s != ""} }

func ts(t time.Time) pgtype.Timestamptz { return pgtype.Timestamptz{Time: t, Valid: true} }
