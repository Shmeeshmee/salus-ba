package terminology

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Shmeeshmee/salus/internal/terminology/terminologydb"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

type postgresRepository struct {
	db postgres.Querier
}

// NewPostgresRepository returns the Repository on db, using the transaction in the context when there is one.
// The terms need no scope: salus_app reads them all and writes none.
func NewPostgresRepository(db postgres.Querier) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) q(ctx context.Context) *terminologydb.Queries {
	return terminologydb.New(postgres.Conn(ctx, r.db))
}

// likeEscaper makes the search text a literal LIKE prefix; folding leaves these characters as they are.
var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

func (r *postgresRepository) Search(ctx context.Context, s Search) ([]Term, error) {
	params := terminologydb.SearchTermsParams{Text: s.Text, Prefix: likeEscaper.Replace(s.Text), MaxTerms: s.Limit}
	if s.Category != "" {
		params.Category = terminologydb.NullTermCategory{TermCategory: terminologydb.TermCategory(s.Category), Valid: true}
	}
	rows, err := r.q(ctx).SearchTerms(ctx, params)
	if err != nil || len(rows) == 0 {
		return []Term{}, err
	}
	terms := make([]Term, len(rows))
	index := make(map[[16]byte]int, len(rows))
	ids := make([]pgtype.UUID, len(rows))
	for i, row := range rows {
		terms[i] = Term{
			ReviewedAt: row.ReviewedAt.Time, ID: row.ID.String(), Category: string(row.Category),
			WikidataID: row.WikidataID.String, ICD10: row.Icd10.String, TA98: row.Ta98.String, ATC: row.Atc.String,
			PCS: row.Pcs.String, ReviewStatus: row.ReviewStatus, Rank: row.Rank,
		}
		index[row.ID.Bytes] = i
		ids[i] = row.ID
	}
	labels, err := r.q(ctx).TermLabels(ctx, terminologydb.TermLabelsParams{TermIds: ids, Languages: s.Languages})
	if err != nil {
		return nil, err
	}
	for _, l := range labels {
		t := &terms[index[l.TermID.Bytes]]
		t.Labels = append(t.Labels, Label{Language: string(l.Language), Text: l.Label, Preferred: l.Preferred})
	}
	return terms, nil
}
