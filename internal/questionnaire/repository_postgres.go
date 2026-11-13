package questionnaire

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Shmeeshmee/salus/internal/questionnaire/questionnairedb"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

type postgresRepository struct {
	db postgres.Querier
}

// NewPostgresRepository returns the Repository on db, using the transaction in the context when there is one.
// Its queries run as salus_app, so row-level security shows only the clinic set.
func NewPostgresRepository(db postgres.Querier) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) q(ctx context.Context) *questionnairedb.Queries {
	return questionnairedb.New(postgres.Conn(ctx, r.db))
}

func (r *postgresRepository) SetOrganization(ctx context.Context, id string) error {
	return postgres.SetLocal(ctx, "app.organization_id", id)
}

func (r *postgresRepository) SetPatient(ctx context.Context, id string) error {
	return postgres.SetLocal(ctx, "app.patient_id", id)
}

func (r *postgresRepository) ByID(ctx context.Context, id string) (Questionnaire, error) {
	var u pgtype.UUID
	if u.Scan(id) != nil {
		return Questionnaire{}, ErrNotFound
	}
	row, err := r.q(ctx).QuestionnaireByID(ctx, u)
	return toQuestionnaire(row, err, ErrNotFound)
}

func (r *postgresRepository) ByIDs(ctx context.Context, ids []string) ([]Questionnaire, error) {
	rows, err := r.q(ctx).QuestionnairesByIDs(ctx, uuidList(ids))
	if err != nil {
		return nil, err
	}
	out := make([]Questionnaire, 0, len(rows))
	for _, row := range rows {
		q, err := toQuestionnaire(questionnairedb.QuestionnaireByIDRow(row), nil, nil)
		if err != nil {
			return nil, err
		}
		out = append(out, q)
	}
	return out, nil
}

func (r *postgresRepository) Shared(ctx context.Context, practitionerID string, ids []string) ([]Questionnaire, error) {
	var p pgtype.UUID
	if p.Scan(practitionerID) != nil {
		return nil, nil
	}
	rows, err := r.q(ctx).SharedQuestionnaires(ctx, questionnairedb.SharedQuestionnairesParams{Ids: uuidList(ids), PractitionerID: p})
	if err != nil {
		return nil, err
	}
	out := make([]Questionnaire, 0, len(rows))
	for _, row := range rows {
		q, err := toQuestionnaire(questionnairedb.QuestionnaireByIDRow(row), nil, nil)
		if err != nil {
			return nil, err
		}
		out = append(out, q)
	}
	return out, nil
}

// uuidList is the ids that are UUIDs; the others match nothing.
func uuidList(ids []string) []pgtype.UUID {
	out := make([]pgtype.UUID, 0, len(ids))
	for _, id := range ids {
		var u pgtype.UUID
		if u.Scan(id) == nil {
			out = append(out, u)
		}
	}
	return out
}

func (r *postgresRepository) Lock(ctx context.Context, id string) (Questionnaire, error) {
	var u pgtype.UUID
	if u.Scan(id) != nil {
		return Questionnaire{}, ErrNotFound
	}
	row, err := r.q(ctx).LockQuestionnaire(ctx, u)
	return toQuestionnaire(questionnairedb.QuestionnaireByIDRow(row), err, ErrNotFound)
}

func (r *postgresRepository) Search(ctx context.Context, f Filter) ([]Questionnaire, error) {
	var p questionnairedb.SearchQuestionnairesParams
	if f.SeriesID != "" && p.SeriesID.Scan(f.SeriesID) != nil {
		return nil, nil // no series has that url
	}
	if f.Status != "" {
		p.Status = pgtype.Text{String: string(f.Status), Valid: true}
	}
	if f.Version != 0 {
		p.Version = pgtype.Int4{Int32: f.Version, Valid: true}
	}
	rows, err := r.q(ctx).SearchQuestionnaires(ctx, p)
	if err != nil {
		return nil, err
	}
	out := make([]Questionnaire, 0, len(rows))
	for _, row := range rows {
		q, err := toQuestionnaire(questionnairedb.QuestionnaireByIDRow(row), nil, nil)
		if err != nil {
			return nil, err
		}
		out = append(out, q)
	}
	return out, nil
}

func (r *postgresRepository) Create(ctx context.Context, q Questionnaire, at time.Time) (Questionnaire, error) {
	var org pgtype.UUID
	if org.Scan(q.OrganizationID) != nil {
		return Questionnaire{}, errors.New("questionnaire: the clinic id is not a UUID")
	}
	items, err := itemsJSON(q.Items)
	if err != nil {
		return Questionnaire{}, err
	}
	row, err := r.q(ctx).CreateQuestionnaire(ctx, questionnairedb.CreateQuestionnaireParams{
		OrganizationID: org, Title: q.Title, Description: q.Description, Items: items, Language: nullLanguage(q.Language), UpdatedAt: timestamp(at),
	})
	return toQuestionnaire(questionnairedb.QuestionnaireByIDRow(row), err, nil)
}

func (r *postgresRepository) CreateVersion(ctx context.Context, from Questionnaire, at time.Time) (Questionnaire, error) {
	var u pgtype.UUID
	if u.Scan(from.ID) != nil {
		return Questionnaire{}, ErrNotFound
	}
	row, err := r.q(ctx).CreateQuestionnaireVersion(ctx, questionnairedb.CreateQuestionnaireVersionParams{FromID: u, UpdatedAt: timestamp(at)})
	next, err := toQuestionnaire(questionnairedb.QuestionnaireByIDRow(row), err, ErrDraftExists)
	if err != nil {
		return Questionnaire{}, err
	}
	err = r.q(ctx).CopyQuestionnaireTranslations(ctx, questionnairedb.CopyQuestionnaireTranslationsParams{FromID: u, ToID: row.ID, At: timestamp(at)})
	if err != nil {
		return Questionnaire{}, err
	}
	return next, nil
}

func (r *postgresRepository) UpdateDraft(ctx context.Context, q Questionnaire, at time.Time) (Questionnaire, error) {
	var u pgtype.UUID
	if u.Scan(q.ID) != nil {
		return Questionnaire{}, ErrConflict
	}
	items, err := itemsJSON(q.Items)
	if err != nil {
		return Questionnaire{}, err
	}
	row, err := r.q(ctx).UpdateQuestionnaireDraft(ctx, questionnairedb.UpdateQuestionnaireDraftParams{
		ID: u, Title: q.Title, Description: q.Description, Items: items, Language: nullLanguage(q.Language), VersionID: q.VersionID,
		UpdatedAt: timestamp(at),
	})
	return toQuestionnaire(questionnairedb.QuestionnaireByIDRow(row), err, ErrConflict)
}

func (r *postgresRepository) DeleteDraft(ctx context.Context, id string) error {
	var u pgtype.UUID
	if u.Scan(id) != nil {
		return ErrNotFound
	}
	n, err := r.q(ctx).DeleteQuestionnaireDraft(ctx, u)
	if err == nil && n == 0 {
		return ErrNotFound
	}
	return err
}

func (r *postgresRepository) Publish(ctx context.Context, id string, at time.Time) (Questionnaire, error) {
	var u pgtype.UUID
	if u.Scan(id) != nil {
		return Questionnaire{}, ErrConflict
	}
	row, err := r.q(ctx).PublishQuestionnaire(ctx, questionnairedb.PublishQuestionnaireParams{ID: u, At: timestamp(at)})
	return toQuestionnaire(questionnairedb.QuestionnaireByIDRow(row), err, ErrConflict)
}

func (r *postgresRepository) Retire(ctx context.Context, id string, at time.Time) (Questionnaire, error) {
	var u pgtype.UUID
	if u.Scan(id) != nil {
		return Questionnaire{}, ErrConflict
	}
	row, err := r.q(ctx).RetireQuestionnaire(ctx, questionnairedb.RetireQuestionnaireParams{ID: u, At: timestamp(at)})
	return toQuestionnaire(questionnairedb.QuestionnaireByIDRow(row), err, ErrConflict)
}

func (r *postgresRepository) RetireActive(ctx context.Context, seriesID string, at time.Time) error {
	var u pgtype.UUID
	if u.Scan(seriesID) != nil {
		return nil // no series has that id
	}
	return r.q(ctx).RetireActiveQuestionnaire(ctx, questionnairedb.RetireActiveQuestionnaireParams{SeriesID: u, At: timestamp(at)})
}

func (r *postgresRepository) Translations(ctx context.Context, id string) ([]Translation, error) {
	var u pgtype.UUID
	if u.Scan(id) != nil {
		return nil, nil // no version has that id
	}
	rows, err := r.q(ctx).QuestionnaireTranslations(ctx, u)
	if err != nil {
		return nil, err
	}
	out := make([]Translation, 0, len(rows))
	for _, row := range rows {
		var t Translation
		if err := json.Unmarshal(row.Texts, &t); err != nil {
			return nil, err
		}
		t.Language = string(row.Language)
		out = append(out, t)
	}
	return out, nil
}

func (r *postgresRepository) PutTranslations(ctx context.Context, id string, ts []Translation, at time.Time) error {
	var u pgtype.UUID
	if u.Scan(id) != nil {
		return ErrNotFound
	}
	if err := r.q(ctx).DeleteQuestionnaireTranslations(ctx, u); err != nil {
		return err
	}
	for _, t := range ts {
		if t.Items == nil {
			t.Items = map[string]ItemTranslation{}
		}
		texts, err := json.Marshal(t)
		if err != nil {
			return err
		}
		err = r.q(ctx).InsertQuestionnaireTranslation(ctx, questionnairedb.InsertQuestionnaireTranslationParams{
			QuestionnaireID: u, Language: questionnairedb.Language(t.Language), Texts: texts, At: timestamp(at),
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *postgresRepository) Default(ctx context.Context, roleID string) (Default, error) {
	var u pgtype.UUID
	if u.Scan(roleID) != nil {
		return Default{}, ErrNotFound
	}
	row, err := r.q(ctx).QuestionnaireDefault(ctx, u)
	return toDefault(row, err)
}

func (r *postgresRepository) SetDefault(ctx context.Context, d Default, at time.Time) (Default, error) {
	var role, org, series pgtype.UUID
	if role.Scan(d.PractitionerRoleID) != nil || org.Scan(d.OrganizationID) != nil || series.Scan(d.SeriesID) != nil {
		return Default{}, errors.New("questionnaire: an id of the default is not a UUID")
	}
	row, err := r.q(ctx).SetQuestionnaireDefault(ctx, questionnairedb.SetQuestionnaireDefaultParams{
		PractitionerRoleID: role, OrganizationID: org, SeriesID: series, UpdatedAt: timestamp(at),
	})
	return toDefault(questionnairedb.QuestionnaireDefaultRow(row), err)
}

func (r *postgresRepository) ClearDefault(ctx context.Context, roleID string) error {
	var u pgtype.UUID
	if u.Scan(roleID) != nil {
		return nil // no role has that id
	}
	return r.q(ctx).ClearQuestionnaireDefault(ctx, u)
}

// nullLanguage is a language tag as a query takes it: "" is null, for the column's default or the draft's own.
func nullLanguage(tag string) questionnairedb.NullLanguage {
	return questionnairedb.NullLanguage{Language: questionnairedb.Language(tag), Valid: tag != ""}
}

func timestamp(t time.Time) pgtype.Timestamptz { return pgtype.Timestamptz{Time: t, Valid: true} }

// itemsJSON is the item tree as stored in questionnaire.items: always a list.
func itemsJSON(items []Item) ([]byte, error) {
	if items == nil {
		items = []Item{}
	}
	return json.Marshal(items)
}

// toQuestionnaire maps a row, or the query's error: noRows for no row. Every questionnaire query selects the
// same columns.
func toQuestionnaire(row questionnairedb.QuestionnaireByIDRow, err, noRows error) (Questionnaire, error) {
	if errors.Is(err, pgx.ErrNoRows) && noRows != nil {
		return Questionnaire{}, noRows
	}
	if err != nil {
		return Questionnaire{}, err
	}
	var items []Item
	if err := json.Unmarshal(row.Items, &items); err != nil {
		return Questionnaire{}, err
	}
	return Questionnaire{
		ID: row.ID.String(), SeriesID: row.SeriesID.String(), OrganizationID: row.OrganizationID.String(),
		Version: row.Version, VersionID: row.VersionID, Status: Status(row.Status), Title: row.Title, Description: row.Description,
		Language: string(row.Language),
		Items:    items, LastUpdated: row.UpdatedAt.Time, PublishedAt: row.PublishedAt.Time, RetiredAt: row.RetiredAt.Time,
	}, nil
}

func toDefault(row questionnairedb.QuestionnaireDefaultRow, err error) (Default, error) {
	if errors.Is(err, pgx.ErrNoRows) {
		return Default{}, ErrNotFound
	}
	if err != nil {
		return Default{}, err
	}
	return Default{
		PractitionerRoleID: row.PractitionerRoleID.String(), OrganizationID: row.OrganizationID.String(),
		SeriesID: row.SeriesID.String(), VersionID: row.VersionID, LastUpdated: row.UpdatedAt.Time,
	}, nil
}
