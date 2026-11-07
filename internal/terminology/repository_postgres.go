package terminology

import (
	"context"
	"strings"

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
	params := terminologydb.SearchTermsParams{Text: s.Text, Prefix: likeEscaper.Replace(s.Text), MaxTerms: s.Limit, Languages: s.Languages}
	if s.Category != "" {
		params.Category = terminologydb.NullTermCategory{TermCategory: terminologydb.TermCategory(s.Category), Valid: true}
	}
	rows, err := r.q(ctx).SearchTerms(ctx, params)
	if err != nil {
		return nil, err
	}
	terms := make([]Term, len(rows))
	for i, row := range rows {
		terms[i] = Term{
			ReviewedAt: row.ReviewedAt.Time, ID: row.ID.String(), Category: string(row.Category),
			WikidataID: row.WikidataID.String, ICD10: row.Icd10.String, TA98: row.Ta98.String, ATC: row.Atc.String,
			PCS: row.Pcs.String, ReviewStatus: row.ReviewStatus, Rank: row.Rank,
		}
		for j, text := range row.Labels {
			terms[i].Labels = append(terms[i].Labels, Label{Language: row.LabelLanguages[j], Text: text, Preferred: row.LabelPreferred[j]})
		}
	}
	return terms, nil
}
