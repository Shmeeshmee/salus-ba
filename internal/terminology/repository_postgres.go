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
	if err := r.insertLabels(ctx, id, t.Labels); err != nil {
		return false, err
	}
	return true, r.q(ctx).SyncTranslations(ctx, id)
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
	reviewer, ok := optionalUUID(t.ReviewedBy)
	if !ok {
		return errInvalidReviewer
	}
	n, err := r.q(ctx).UpdateTerm(ctx, terminologydb.UpdateTermParams{
		Category: terminologydb.TermCategory(t.Category), Rank: t.Rank, WikidataID: text(t.WikidataID),
		Icd10: text(t.ICD10), Ta98: text(t.TA98), Atc: text(t.ATC), Pcs: text(t.PCS), ReviewStatus: t.ReviewStatus,
		ReviewedAt: pgtype.Timestamptz{Time: t.ReviewedAt, Valid: !t.ReviewedAt.IsZero()}, ReviewedBy: reviewer,
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
	if err := r.insertLabels(ctx, id, t.Labels); err != nil {
		return err
	}
	return r.q(ctx).SyncTranslations(ctx, id)
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

// errInvalidReviewer is returned for a reviewer that is no account id; the value is not repeated.
var errInvalidReviewer = errors.New("terminology: reviewed_by is no UUID")

// optionalUUID is id as a nullable uuid, null when empty; false when it is no UUID.
func optionalUUID(id string) (pgtype.UUID, bool) {
	var u pgtype.UUID
	if id == "" {
		return u, true
	}
	return u, u.Scan(id) == nil
}

// NewPostgresReviewRepository returns the ReviewRepository on db (the app role), using the transaction in the
// context when there is one.
func NewPostgresReviewRepository(db postgres.Querier) ReviewRepository {
	return &postgresRepository{db: db}
}

// statusOrder is the place of a status in the queue: unreviewed first (ReviewQueue's status_order).
var statusOrder = map[string]int32{ReviewUnreviewed: 0, ReviewNeedsChanges: 1, ReviewApproved: 2}

// orderStatus is the status at a place of the queue.
var orderStatus = []string{ReviewUnreviewed, ReviewNeedsChanges, ReviewApproved}

func (r *postgresRepository) Queue(ctx context.Context, q QueueFilter) ([]QueueRow, error) {
	p := terminologydb.ReviewQueueParams{Language: terminologydb.TermLanguage(q.Language), MaxEntries: q.Limit}
	if q.Status != "" {
		p.StatusOrder = pgtype.Int4{Int32: statusOrder[q.Status], Valid: true}
	}
	if q.AfterID != "" {
		if err := p.AfterID.Scan(q.AfterID); err != nil {
			return nil, nil // no term has that id: nothing comes after it
		}
		p.AfterOrder, p.AfterRank = statusOrder[q.AfterStatus], q.AfterRank
	}
	rows, err := r.q(ctx).ReviewQueue(ctx, p)
	if err != nil {
		return nil, err
	}
	out := make([]QueueRow, len(rows))
	for i, row := range rows {
		id := row.ID.String()
		out[i] = QueueRow{
			Status: orderStatus[row.StatusOrder],
			Term: Term{
				ReviewedAt: row.ReviewedAt.Time.UTC(), ID: id, Category: string(row.Category), WikidataID: row.WikidataID.String,
				ICD10: row.Icd10.String, TA98: row.Ta98.String, ATC: row.Atc.String, PCS: row.Pcs.String,
				ReviewStatus: row.ReviewStatus, Rank: row.Rank,
			},
		}
		for j, text := range row.Labels {
			out[i].Term.Labels = append(out[i].Term.Labels, Label{Language: row.LabelLanguages[j], Text: text, Preferred: row.LabelPreferred[j]})
		}
		if row.HasTranslation {
			out[i].Translation = &Translation{
				UpdatedAt: row.TrUpdatedAt.Time.UTC(), TermID: id, Language: q.Language, VersionID: row.TrVersionID.Int32,
				Review: review(row.TrStatus, row.TrReviewedAt, row.TrReviewedBy, row.TrNote),
			}
		}
		if row.HasExplanation {
			out[i].Explanation = &Explanation{
				UpdatedAt: row.ExUpdatedAt.Time.UTC(), TermID: id, Language: q.Language, Text: row.ExText.String,
				VersionID: row.ExVersionID.Int32, Review: review(row.ExStatus, row.ExReviewedAt, row.ExReviewedBy, row.ExNote),
			}
		}
	}
	return out, nil
}

func (r *postgresRepository) Progress(ctx context.Context) ([]LanguageProgress, error) {
	rows, err := r.q(ctx).ReviewProgress(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]LanguageProgress, len(rows))
	for i, row := range rows {
		out[i] = LanguageProgress(row)
	}
	return out, nil
}

func (r *postgresRepository) LockEntry(ctx context.Context, termID, language string) (EntryState, error) {
	var id pgtype.UUID
	if err := id.Scan(termID); err != nil {
		return EntryState{}, ErrTermNotFound
	}
	q := r.q(ctx)
	lang := terminologydb.TermLanguage(language)
	term, err := q.LockTerm(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return EntryState{}, ErrTermNotFound
	}
	if err != nil {
		return EntryState{}, err
	}
	s := EntryState{TermReview: review(term.ReviewStatus, term.ReviewedAt, term.ReviewedBy, pgtype.Text{})}
	labels, err := q.TermLabelsIn(ctx, terminologydb.TermLabelsInParams{TermID: id, Language: lang})
	if err != nil {
		return EntryState{}, err
	}
	for _, l := range labels {
		s.Labels = append(s.Labels, Label{Language: language, Text: l.Label, Preferred: l.Preferred})
	}
	translations, err := q.TermTranslations(ctx, id)
	if err != nil {
		return EntryState{}, err
	}
	for _, t := range translations {
		s.Translations = append(s.Translations, Translation{
			UpdatedAt: t.UpdatedAt.Time.UTC(), TermID: termID, Language: t.Language, VersionID: t.VersionID,
			Review: review(t.ReviewStatus, t.ReviewedAt, t.ReviewedBy, t.Note),
		})
	}
	explanations, err := q.TermExplanationIn(ctx, terminologydb.TermExplanationInParams{TermID: id, Language: lang})
	if err != nil {
		return EntryState{}, err
	}
	for _, e := range explanations {
		s.Explanation = &Explanation{
			UpdatedAt: e.UpdatedAt.Time.UTC(), TermID: termID, Language: language, Text: e.Text, VersionID: e.VersionID,
			Review: review(e.ReviewStatus, e.ReviewedAt, e.ReviewedBy, e.Note),
		}
	}
	return s, nil
}

func (r *postgresRepository) ReplaceLabels(ctx context.Context, termID, language string, labels []Label) error {
	var id pgtype.UUID
	if err := id.Scan(termID); err != nil {
		return ErrTermNotFound
	}
	if err := r.q(ctx).DeleteLabelsIn(ctx, terminologydb.DeleteLabelsInParams{TermID: id, Language: terminologydb.TermLanguage(language)}); err != nil {
		return err
	}
	return r.insertLabels(ctx, id, labels)
}

func (r *postgresRepository) PutTranslation(ctx context.Context, t Translation) error {
	var id pgtype.UUID
	if err := id.Scan(t.TermID); err != nil {
		return ErrTermNotFound
	}
	by, ok := optionalUUID(t.Review.ReviewedBy)
	if !ok {
		return errInvalidReviewer
	}
	return r.q(ctx).PutTranslation(ctx, terminologydb.PutTranslationParams{
		TermID: id, Language: terminologydb.TermLanguage(t.Language), UpdatedAt: instant(t.UpdatedAt),
		ReviewedAt: instant(t.Review.ReviewedAt), ReviewedBy: by, VersionID: t.VersionID, ReviewStatus: t.Review.Status,
		Note: text(t.Review.Note),
	})
}

func (r *postgresRepository) PutExplanation(ctx context.Context, e Explanation) error {
	var id pgtype.UUID
	if err := id.Scan(e.TermID); err != nil {
		return ErrTermNotFound
	}
	by, ok := optionalUUID(e.Review.ReviewedBy)
	if !ok {
		return errInvalidReviewer
	}
	return r.q(ctx).PutExplanation(ctx, terminologydb.PutExplanationParams{
		TermID: id, Language: terminologydb.TermLanguage(e.Language), UpdatedAt: instant(e.UpdatedAt),
		ReviewedAt: instant(e.Review.ReviewedAt), ReviewedBy: by, VersionID: e.VersionID, ReviewStatus: e.Review.Status,
		Note: text(e.Review.Note), Text: e.Text,
	})
}

func (r *postgresRepository) SetTermReview(ctx context.Context, termID string, rv Review, at time.Time) error {
	var id pgtype.UUID
	if err := id.Scan(termID); err != nil {
		return ErrTermNotFound
	}
	by, ok := optionalUUID(rv.ReviewedBy)
	if !ok {
		return errInvalidReviewer
	}
	return r.q(ctx).SetTermReview(ctx, terminologydb.SetTermReviewParams{
		ReviewStatus: rv.Status, ReviewedAt: instant(rv.ReviewedAt), ReviewedBy: by, UpdatedAt: instant(at), ID: id,
	})
}

func (r *postgresRepository) SetTranslationsStatus(ctx context.Context, termID, status string, at time.Time) error {
	var id pgtype.UUID
	if err := id.Scan(termID); err != nil {
		return ErrTermNotFound
	}
	return r.q(ctx).SetTranslationsStatus(ctx, terminologydb.SetTranslationsStatusParams{ReviewStatus: status, At: instant(at), TermID: id})
}

// review maps a row's review columns; the instants in UTC, as every instant this repository reads.
func review(status string, at pgtype.Timestamptz, by pgtype.UUID, note pgtype.Text) Review {
	r := Review{Status: status, ReviewedAt: at.Time.UTC(), Note: note.String}
	if by.Valid {
		r.ReviewedBy = by.String()
	}
	return r
}

// instant is t as a nullable timestamptz: NULL when zero.
func instant(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: !t.IsZero()}
}
