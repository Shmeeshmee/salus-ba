package suggestion

import (
	"context"
	"errors"

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

func (r *postgresRepository) Save(ctx context.Context, e Event) (Saved, error) {
	var id, enc pgtype.UUID
	if id.Scan(e.ID) != nil || enc.Scan(e.EncounterID) != nil {
		return SavedUnchanged, errors.New("suggestion: an id is not a UUID")
	}
	sources := make([]pgtype.UUID, len(e.SourceSegmentIDs))
	for i, s := range e.SourceSegmentIDs {
		if sources[i].Scan(s) != nil {
			return SavedUnchanged, errors.New("suggestion: a source segment id is not a UUID")
		}
	}
	q := suggestiondb.New(postgres.Conn(ctx, r.db))
	n, err := q.InsertSuggestion(ctx, suggestiondb.InsertSuggestionParams{
		ID: id, EncounterID: enc, Type: string(e.Type), Text: e.Text, Confidence: e.Confidence, SourceSegmentIds: sources,
		Status: string(e.Status),
	})
	switch {
	case err != nil:
		return SavedUnchanged, err
	case n == 1:
		return SavedCreated, nil
	}
	n, err = q.UpdateSuggestionStatus(ctx, suggestiondb.UpdateSuggestionStatusParams{Status: string(e.Status), ID: id, EncounterID: enc})
	if err != nil || n == 0 {
		return SavedUnchanged, err
	}
	return SavedUpdated, nil
}
