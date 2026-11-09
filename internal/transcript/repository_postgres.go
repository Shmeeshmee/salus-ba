package transcript

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Shmeeshmee/salus/internal/transcript/transcriptdb"
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

func (r *postgresRepository) q(ctx context.Context) *transcriptdb.Queries {
	return transcriptdb.New(postgres.Conn(ctx, r.db))
}

func (r *postgresRepository) Insert(ctx context.Context, e Event) (bool, error) {
	var id, enc pgtype.UUID
	if id.Scan(e.SegmentID) != nil || enc.Scan(e.EncounterID) != nil {
		return false, errors.New("transcript: an id is not a UUID")
	}
	n, err := r.q(ctx).InsertSegment(ctx, transcriptdb.InsertSegmentParams{
		ID: id, EncounterID: enc, Speaker: string(e.Speaker), Lang: e.Lang, Original: e.Original, Translation: e.Translation,
		TranslationFailed: e.TranslationFailed,
		TsStart:           ts(e.TsStart), TsEnd: ts(e.TsEnd),
	})
	return n == 1, err
}

func (r *postgresRepository) ByEncounter(ctx context.Context, encounterID string) ([]Event, error) {
	var enc pgtype.UUID
	if enc.Scan(encounterID) != nil {
		return nil, nil // an id that is not a UUID matches nothing
	}
	rows, err := r.q(ctx).SegmentsByEncounter(ctx, enc)
	if err != nil {
		return nil, err
	}
	out := make([]Event, 0, len(rows))
	for _, row := range rows {
		out = append(out, Event{
			EncounterID: row.EncounterID.String(), SegmentID: row.ID.String(), Speaker: Speaker(row.Speaker), Lang: row.Lang,
			Original: row.Original, Translation: row.Translation, TranslationFailed: row.TranslationFailed, IsFinal: true, TsStart: row.TsStart.Time.UTC(), TsEnd: row.TsEnd.Time.UTC(),
		})
	}
	return out, nil
}

func ts(t time.Time) pgtype.Timestamptz { return pgtype.Timestamptz{Time: t, Valid: true} }
