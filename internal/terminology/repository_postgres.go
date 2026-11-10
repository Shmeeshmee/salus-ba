package terminology

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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

// NewPostgresCurationRepository returns the CurationRepository on db, which must connect as the owner: salus_app
// may only read the terms.
func NewPostgresCurationRepository(db postgres.Querier) CurationRepository {
	return &postgresRepository{db: db}
}

// likeEscaper makes the search text a literal LIKE prefix; folding leaves these characters as they are.
var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

func (r *postgresRepository) Search(ctx context.Context, s Search) ([]Term, error) {
	params := terminologydb.SearchTermsParams{
		Text: s.Text, Prefix: likeEscaper.Replace(s.Text), UiLanguage: s.UILanguage, MaxTerms: s.Limit, Languages: s.Languages,
	}
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

func (r *postgresRepository) Glossary(ctx context.Context, q GlossaryQuery) ([]GlossaryEntry, error) {
	rows, err := r.q(ctx).GlossaryTerms(ctx, terminologydb.GlossaryTermsParams{
		MaxTerms: q.Limit, Candidates: q.Candidates,
		SourceLanguage: terminologydb.TermLanguage(q.From), TargetLanguage: terminologydb.TermLanguage(q.To),
	})
	if err != nil {
		return nil, err
	}
	entries := make([]GlossaryEntry, len(rows))
	for i, row := range rows {
		entries[i] = GlossaryEntry{Source: row.Source, Target: row.Target}
	}
	return entries, nil
}

func (r *postgresRepository) CreateFromSeed(ctx context.Context, t Term) (bool, error) {
	id, err := r.q(ctx).CreateSeedTerm(ctx, terminologydb.CreateSeedTermParams{
		Category: terminologydb.TermCategory(t.Category), Rank: t.Rank, WikidataID: text(t.WikidataID),
		Icd10: text(t.ICD10), Ta98: text(t.TA98), Atc: text(t.ATC), Pcs: text(t.PCS),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, r.insertLabels(ctx, id, t.Labels)
}

func (r *postgresRepository) insertLabels(ctx context.Context, id pgtype.UUID, labels []Label) error {
	p := terminologydb.InsertLabelsParams{TermID: id}
	for _, l := range labels {
		p.Languages = append(p.Languages, l.Language)
		p.Labels = append(p.Labels, l.Text)
		p.Preferred = append(p.Preferred, l.Preferred)
	}
	return r.q(ctx).InsertLabels(ctx, p)
}

func (r *postgresRepository) Term(ctx context.Context, id string) (Term, error) {
	var uid pgtype.UUID
	if err := uid.Scan(id); err != nil {
		return Term{}, ErrTermNotFound
	}
	rows, err := r.q(ctx).ListTerms(ctx, terminologydb.ListTermsParams{ID: uid})
	if err != nil {
		return Term{}, err
	}
	if len(rows) == 0 {
		return Term{}, ErrTermNotFound
	}
	return listedTerm(rows[0]), nil
}

func (r *postgresRepository) UpdateTerm(ctx context.Context, t Term, at time.Time) error {
	var id pgtype.UUID
	if err := id.Scan(t.ID); err != nil {
		return ErrTermNotFound
	}
	n, err := r.q(ctx).UpdateTerm(ctx, terminologydb.UpdateTermParams{
		Category: terminologydb.TermCategory(t.Category), Rank: t.Rank, WikidataID: text(t.WikidataID),
		Icd10: text(t.ICD10), Ta98: text(t.TA98), Atc: text(t.ATC), Pcs: text(t.PCS), ReviewStatus: t.ReviewStatus,
		ReviewedAt: pgtype.Timestamptz{Time: t.ReviewedAt, Valid: !t.ReviewedAt.IsZero()}, ReviewedBy: uuidOrNull(t.ReviewedBy),
		Note:      text(t.Note),
		UpdatedAt: pgtype.Timestamptz{Time: at, Valid: true}, ID: id,
	})
	if pgErr := (*pgconn.PgError)(nil); errors.As(err, &pgErr) && pgErr.ConstraintName == "term_wikidata_id_key" {
		return ErrDuplicateWikidata
	}
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrTermNotFound
	}
	if err := r.q(ctx).DeleteLabels(ctx, id); err != nil {
		return err
	}
	return r.insertLabels(ctx, id, t.Labels)
}

func (r *postgresRepository) Terms(ctx context.Context, status string) ([]Term, error) {
	rows, err := r.q(ctx).ListTerms(ctx, terminologydb.ListTermsParams{ReviewStatus: text(status)})
	if err != nil {
		return nil, err
	}
	terms := make([]Term, len(rows))
	for i, row := range rows {
		terms[i] = listedTerm(row)
	}
	return terms, nil
}

func (r *postgresRepository) CountByCategory(ctx context.Context) ([]CategoryCount, error) {
	rows, err := r.q(ctx).CountTermsByCategory(ctx)
	if err != nil {
		return nil, err
	}
	counts := make([]CategoryCount, len(rows))
	for i, row := range rows {
		counts[i] = CategoryCount{Category: string(row.Category), Terms: int(row.Terms)}
	}
	return counts, nil
}

// listedTerm maps a ListTerms row.
func listedTerm(row terminologydb.ListTermsRow) Term {
	t := Term{
		ReviewedAt: row.ReviewedAt.Time, ID: row.ID.String(), Category: string(row.Category),
		WikidataID: row.WikidataID.String, ICD10: row.Icd10.String, TA98: row.Ta98.String, ATC: row.Atc.String,
		PCS: row.Pcs.String, ReviewStatus: row.ReviewStatus, Note: row.Note.String, Rank: row.Rank,
	}
	if row.ReviewedBy.Valid {
		t.ReviewedBy = row.ReviewedBy.String()
	}
	for j, label := range row.Labels {
		t.Labels = append(t.Labels, Label{Language: row.LabelLanguages[j], Text: label, Preferred: row.LabelPreferred[j]})
	}
	return t
}

// text is s as a nullable column: NULL when empty.
func text(s string) pgtype.Text {
	return pgtype.Text{String: s, Valid: s != ""}
}

// uuidOrNull is id as a nullable uuid: null when empty or not a UUID.
func uuidOrNull(id string) pgtype.UUID {
	var u pgtype.UUID
	if u.Scan(id) != nil {
		return pgtype.UUID{}
	}
	return u
}
