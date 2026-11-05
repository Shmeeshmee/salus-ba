package auditevent

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Shmeeshmee/salus/internal/auditevent/auditeventdb"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

type postgresRepository struct {
	db postgres.Querier
}

// NewPostgresRepository returns the Repository on db, using the transaction in the context when there is one.
func NewPostgresRepository(db postgres.Querier) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) Insert(ctx context.Context, e AuditEvent) error {
	agent, err := optionalUUID(e.AgentAccountID)
	if err != nil {
		return err
	}
	org, err := optionalUUID(e.OrganizationID)
	if err != nil {
		return err
	}
	return auditeventdb.New(postgres.Conn(ctx, r.db)).InsertAuditEvent(ctx, auditeventdb.InsertAuditEventParams{
		Recorded:        pgtype.Timestamptz{Time: e.Recorded, Valid: true},
		AgentAccountID:  agent,
		AgentName:       optionalText(e.AgentName),
		OrganizationID:  org,
		Outcome:         int16(e.Outcome),
		Action:          string(e.Action),
		TypeSystem:      e.Type.System,
		TypeCode:        e.Type.Code,
		SubtypeSystem:   e.Subtype.System,
		SubtypeCode:     e.Subtype.Code,
		PurposeSystem:   e.Purpose.System,
		PurposeCode:     e.Purpose.Code,
		EntityReference: optionalText(e.EntityReference),
		EntityName:      optionalText(e.EntityName),
		EntityCount:     pgtype.Int8{Int64: e.EntityCount, Valid: e.EntityName != ""},
		RequestID:       optionalText(e.RequestID),
	})
}

func (r *postgresRepository) CountRecordedBefore(ctx context.Context, before time.Time) (int64, error) {
	return auditeventdb.New(postgres.Conn(ctx, r.db)).CountRecordedBefore(ctx, pgtype.Timestamptz{Time: before, Valid: true})
}

// optionalUUID maps "" to NULL. The error never echoes the value.
func optionalUUID(s string) (pgtype.UUID, error) {
	var u pgtype.UUID
	if s == "" {
		return u, nil
	}
	if err := u.Scan(s); err != nil {
		return u, errors.New("auditevent: an id is not a UUID")
	}
	return u, nil
}

func optionalText(s string) pgtype.Text {
	return pgtype.Text{String: s, Valid: s != ""}
}
