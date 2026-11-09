package suggestion

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Shmeeshmee/salus/internal/suggestion/suggestiondb"
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

var errNotUUID = errors.New("suggestion: an id is not a UUID")

func (r *postgresRepository) Insert(ctx context.Context, e Event) (bool, error) {
	var id, enc pgtype.UUID
	if id.Scan(e.ID) != nil || enc.Scan(e.EncounterID) != nil {
		return false, errNotUUID
	}
	sources := make([]pgtype.UUID, len(e.SourceSegmentIDs))
	for i, s := range e.SourceSegmentIDs {
		if sources[i].Scan(s) != nil {
			return false, errNotUUID
		}
	}
	n, err := suggestiondb.New(postgres.Conn(ctx, r.db)).InsertSuggestion(ctx, suggestiondb.InsertSuggestionParams{
		ID: id, EncounterID: enc, Type: string(e.Type), Text: e.Text, Confidence: e.Confidence, SourceSegmentIds: sources,
		Status: string(e.Status),
	})
	return n == 1, err
}

func (r *postgresRepository) SetStatus(ctx context.Context, encounterID, id string, status Status) (Event, bool, error) {
	var sid, enc pgtype.UUID
	if sid.Scan(id) != nil || enc.Scan(encounterID) != nil {
		return Event{}, false, ErrNotFound
	}
	row, err := suggestiondb.New(postgres.Conn(ctx, r.db)).UpdateSuggestionStatus(ctx, suggestiondb.UpdateSuggestionStatusParams{
		ID: sid, EncounterID: enc, Status: string(status),
	})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return Event{}, false, ErrNotFound
	case err != nil:
		return Event{}, false, err
	}
	return event(row.ID, row.EncounterID, row.Type, row.Text, string(status), row.Confidence, row.SourceSegmentIds), row.Changed, nil
}

func (r *postgresRepository) ByEncounter(ctx context.Context, encounterID string) ([]Event, error) {
	var enc pgtype.UUID
	if enc.Scan(encounterID) != nil {
		return nil, errNotUUID
	}
	rows, err := suggestiondb.New(postgres.Conn(ctx, r.db)).SuggestionsByEncounter(ctx, enc)
	if err != nil {
		return nil, err
	}
	out := make([]Event, len(rows))
	for i, row := range rows {
		out[i] = event(row.ID, row.EncounterID, row.Type, row.Text, row.Status, row.Confidence, row.SourceSegmentIds)
	}
	return out, nil
}

// event maps a row to its event.
func event(id, enc pgtype.UUID, typ, text, status string, confidence float32, sources []pgtype.UUID) Event {
	e := Event{
		ID: id.String(), EncounterID: enc.String(), Type: Type(typ), Text: text, Status: Status(status), Origin: OriginAI,
		Confidence: confidence, SourceSegmentIDs: make([]string, len(sources)),
	}
	for i, s := range sources {
		e.SourceSegmentIDs[i] = s.String()
	}
	return e
}
