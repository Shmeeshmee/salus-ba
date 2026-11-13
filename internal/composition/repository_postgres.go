package composition

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Shmeeshmee/salus/internal/composition/compositiondb"
	"github.com/Shmeeshmee/salus/pkg/keyset"
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

func (r *postgresRepository) q(ctx context.Context) *compositiondb.Queries {
	return compositiondb.New(postgres.Conn(ctx, r.db))
}

func (r *postgresRepository) SetClinic(ctx context.Context, id string) error {
	return postgres.SetLocal(ctx, "app.organization_id", id)
}

func (r *postgresRepository) SetPatient(ctx context.Context, id string) error {
	return postgres.SetLocal(ctx, "app.patient_id", id)
}

func (r *postgresRepository) Templates(ctx context.Context, org string) ([]Template, error) {
	var uorg pgtype.UUID
	if uorg.Scan(org) != nil {
		return nil, nil
	}
	rows, err := r.q(ctx).Templates(ctx, uorg)
	if err != nil {
		return nil, err
	}
	out := make([]Template, len(rows))
	for i, row := range rows {
		if out[i], err = templateOf(compositiondb.TemplateRow(row)); err != nil {
			return nil, err
		}
	}
	return out, r.translate(ctx, out)
}

func (r *postgresRepository) Template(ctx context.Context, id string, lock bool) (Template, error) {
	var uid pgtype.UUID
	if uid.Scan(id) != nil {
		return Template{}, ErrTemplateNotFound
	}
	var row compositiondb.TemplateRow
	var err error
	if lock {
		var locked compositiondb.TemplateForUpdateRow
		locked, err = r.q(ctx).TemplateForUpdate(ctx, uid)
		row = compositiondb.TemplateRow(locked)
	} else {
		row, err = r.q(ctx).Template(ctx, uid)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return Template{}, ErrTemplateNotFound
	}
	if err != nil {
		return Template{}, err
	}
	return r.withTranslations(ctx, row)
}

func (r *postgresRepository) InsertTemplate(ctx context.Context, org string, c TemplateContent, at time.Time) (Template, error) {
	var uorg pgtype.UUID
	if uorg.Scan(org) != nil {
		return Template{}, errors.New("composition: the clinic id is not a UUID")
	}
	fields, err := json.Marshal(c.Fields)
	if err != nil {
		return Template{}, err
	}
	row, err := r.q(ctx).InsertTemplate(ctx, compositiondb.InsertTemplateParams{
		OrganizationID: uorg, Title: c.Title, Language: compositiondb.Language(c.Language), Status: string(c.Status),
		TypeCode: c.TypeCode, Fields: fields, At: ts(at),
	})
	if err != nil {
		return Template{}, err
	}
	return r.replaceTranslations(ctx, compositiondb.TemplateRow(row), c.Translations)
}

func (r *postgresRepository) UpdateTemplate(ctx context.Context, id string, c TemplateContent, at time.Time) (Template, error) {
	var uid pgtype.UUID
	if uid.Scan(id) != nil {
		return Template{}, ErrTemplateNotFound
	}
	fields, err := json.Marshal(c.Fields)
	if err != nil {
		return Template{}, err
	}
	row, err := r.q(ctx).UpdateTemplate(ctx, compositiondb.UpdateTemplateParams{
		ID: uid, Title: c.Title, Language: compositiondb.Language(c.Language), Status: string(c.Status),
		TypeCode: c.TypeCode, Fields: fields, At: ts(at),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Template{}, ErrTemplateNotFound
	}
	if err != nil {
		return Template{}, err
	}
	if err := r.q(ctx).DeleteTranslations(ctx, uid); err != nil {
		return Template{}, err
	}
	return r.replaceTranslations(ctx, compositiondb.TemplateRow(row), c.Translations)
}

// replaceTranslations stores the translations of the template row (which has none) and returns the template.
func (r *postgresRepository) replaceTranslations(ctx context.Context, row compositiondb.TemplateRow, ts []Translation) (Template, error) {
	for _, tr := range ts {
		texts, err := json.Marshal(tr)
		if err != nil {
			return Template{}, err
		}
		err = r.q(ctx).InsertTranslation(ctx, compositiondb.InsertTranslationParams{TemplateID: row.ID, Language: compositiondb.Language(tr.Language), Texts: texts})
		if err != nil {
			return Template{}, err
		}
	}
	return r.withTranslations(ctx, row)
}

func (r *postgresRepository) withTranslations(ctx context.Context, row compositiondb.TemplateRow) (Template, error) {
	t, err := templateOf(row)
	if err != nil {
		return Template{}, err
	}
	out := []Template{t}
	if err := r.translate(ctx, out); err != nil {
		return Template{}, err
	}
	return out[0], nil
}

// translate sets each template's translations, in the enum's order.
func (r *postgresRepository) translate(ctx context.Context, ts []Template) error {
	if len(ts) == 0 {
		return nil
	}
	ids := make([]pgtype.UUID, len(ts))
	index := make(map[string]int, len(ts))
	for i, t := range ts {
		_ = ids[i].Scan(t.ID)
		index[t.ID] = i
	}
	rows, err := r.q(ctx).Translations(ctx, ids)
	if err != nil {
		return err
	}
	for _, row := range rows {
		var tr Translation
		if err := json.Unmarshal(row.Texts, &tr); err != nil {
			return err
		}
		tr.Language = string(row.Language)
		i := index[row.TemplateID.String()]
		ts[i].Translations = append(ts[i].Translations, tr)
	}
	return nil
}

func templateOf(row compositiondb.TemplateRow) (Template, error) {
	t := Template{
		ID: row.ID.String(), OrganizationID: row.OrganizationID.String(), Title: row.Title, Language: string(row.Language),
		Status: TemplateStatus(row.Status), TypeCode: row.TypeCode, VersionID: row.VersionID, LastUpdated: row.UpdatedAt.Time.UTC(),
	}
	return t, json.Unmarshal(row.Fields, &t.Fields)
}

func (r *postgresRepository) SaveDraft(ctx context.Context, d Draft, at time.Time) (Draft, bool, error) {
	var enc, tpl pgtype.UUID
	if enc.Scan(d.EncounterID) != nil || tpl.Scan(d.TemplateID) != nil {
		return Draft{}, false, errors.New("composition: an id is not a UUID")
	}
	fields, err := json.Marshal(d.Fields)
	if err != nil {
		return Draft{}, false, err
	}
	row, err := r.q(ctx).SaveDraft(ctx, compositiondb.SaveDraftParams{
		EncounterID: enc, TemplateID: tpl, Language: compositiondb.Language(d.Language), Title: d.Title, TypeCode: d.TypeCode,
		Fields: fields, At: ts(at),
	})
	if err != nil {
		return Draft{}, false, err
	}
	out, err := r.Draft(ctx, row.ID.String(), false)
	return out, row.Inserted, err
}

func (r *postgresRepository) Draft(ctx context.Context, id string, lock bool) (Draft, error) {
	return r.draft(ctx, id, func(q *compositiondb.Queries, uid pgtype.UUID) (compositiondb.DraftRow, error) {
		if lock {
			row, err := q.DraftForUpdate(ctx, uid)
			return compositiondb.DraftRow(row), err
		}
		return q.Draft(ctx, uid)
	})
}

func (r *postgresRepository) ByDocumentReference(ctx context.Context, id string) (Draft, error) {
	return r.draft(ctx, id, func(q *compositiondb.Queries, uid pgtype.UUID) (compositiondb.DraftRow, error) {
		row, err := q.ByDocumentReference(ctx, uid)
		return compositiondb.DraftRow(row), err
	})
}

func (r *postgresRepository) ByProvenance(ctx context.Context, id string) (Draft, error) {
	return r.draft(ctx, id, func(q *compositiondb.Queries, uid pgtype.UUID) (compositiondb.DraftRow, error) {
		row, err := q.ByProvenance(ctx, uid)
		return compositiondb.DraftRow(row), err
	})
}

func (r *postgresRepository) draft(ctx context.Context, id string, get func(*compositiondb.Queries, pgtype.UUID) (compositiondb.DraftRow, error)) (Draft, error) {
	var uid pgtype.UUID
	if uid.Scan(id) != nil {
		return Draft{}, ErrNotFound
	}
	row, err := get(r.q(ctx), uid)
	if errors.Is(err, pgx.ErrNoRows) {
		return Draft{}, ErrNotFound
	}
	if err != nil {
		return Draft{}, err
	}
	return draftOf(row)
}

func (r *postgresRepository) Drafts(ctx context.Context, encounterID string) ([]Draft, error) {
	var enc pgtype.UUID
	if enc.Scan(encounterID) != nil {
		return nil, nil
	}
	rows, err := r.q(ctx).Drafts(ctx, enc)
	if err != nil {
		return nil, err
	}
	out := make([]Draft, len(rows))
	for i, row := range rows {
		if out[i], err = draftOf(compositiondb.DraftRow(row)); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (r *postgresRepository) Signed(ctx context.Context, encounterID string) ([]Draft, error) {
	var enc pgtype.UUID
	if enc.Scan(encounterID) != nil {
		return nil, nil
	}
	rows, err := r.q(ctx).Signed(ctx, enc)
	if err != nil {
		return nil, err
	}
	out := make([]Draft, len(rows))
	for i, row := range rows {
		if out[i], err = draftOf(compositiondb.DraftRow(row)); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (r *postgresRepository) SignedOf(ctx context.Context, encounterIDs []string, p keyset.Page) ([]Draft, error) {
	arg := compositiondb.SignedOfParams{
		EncounterIds: make([]pgtype.UUID, len(encounterIDs)), FromAt: optTS(p.From), ToAt: optTS(p.To), AfterAt: optTS(p.At),
		AfterRef: p.Ref, Lim: int32(min(p.Limit, 1000)), //nolint:gosec // bounded
	}
	for i, id := range encounterIDs {
		if arg.EncounterIds[i].Scan(id) != nil {
			return nil, errors.New("composition: an encounter id is not a UUID")
		}
	}
	rows, err := r.q(ctx).SignedOf(ctx, arg)
	if err != nil {
		return nil, err
	}
	out := make([]Draft, len(rows))
	for i, row := range rows {
		if out[i], err = draftOf(compositiondb.DraftRow(row)); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (r *postgresRepository) SignedOfVisits(ctx context.Context, encounterIDs []string) ([]Draft, error) {
	ids := make([]pgtype.UUID, 0, len(encounterIDs))
	for _, id := range encounterIDs {
		var u pgtype.UUID
		if u.Scan(id) == nil {
			ids = append(ids, u)
		}
	}
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := r.q(ctx).SignedOfEncounters(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]Draft, len(rows))
	for i, row := range rows {
		if out[i], err = draftOf(compositiondb.DraftRow(row)); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// optTS is t, or null when it is zero.
func optTS(t time.Time) pgtype.Timestamptz { return pgtype.Timestamptz{Time: t, Valid: !t.IsZero()} }

func (r *postgresRepository) Sign(ctx context.Context, d Draft, at time.Time) (Draft, error) {
	var id, role, ref, prov pgtype.UUID
	if id.Scan(d.ID) != nil || role.Scan(d.SignerRoleID) != nil || ref.Scan(d.DocumentReferenceID) != nil || prov.Scan(d.ProvenanceID) != nil {
		return Draft{}, errors.New("composition: an id is not a UUID")
	}
	fields, err := json.Marshal(d.Fields)
	if err != nil {
		return Draft{}, err
	}
	_, err = r.q(ctx).Sign(ctx, compositiondb.SignParams{
		ID: id, VersionID: d.VersionID, Fields: fields, At: ts(at), SignerRoleID: role, DocumentReferenceID: ref, ProvenanceID: prov,
		PdfSize: pgtype.Int8{Int64: d.PDF.Size, Valid: true}, PdfKekID: pgtype.Text{String: d.PDF.KEKID, Valid: true},
		PdfSha1: d.PDF.SHA1, PdfWrappedKey: d.PDF.WrappedKey,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Draft{}, ErrVersionMismatch
	}
	if err != nil {
		return Draft{}, err
	}
	return r.Draft(ctx, d.ID, false)
}

func draftOf(row compositiondb.DraftRow) (Draft, error) {
	d := Draft{
		ID: row.ID.String(), EncounterID: row.EncounterID.String(), TemplateID: row.TemplateID.String(), Status: Status(row.Status),
		Language: string(row.Language), Title: row.Title, TypeCode: row.TypeCode, VersionID: row.VersionID, UpdatedAt: row.UpdatedAt.Time.UTC(),
	}
	if row.SignedAt.Valid {
		d.SignedAt, d.SignerRoleID = row.SignedAt.Time.UTC(), row.SignerRoleID.String()
		d.DocumentReferenceID, d.ProvenanceID = row.DocumentReferenceID.String(), row.ProvenanceID.String()
		d.PDF = PDF{KEKID: row.PdfKekID.String, SHA1: row.PdfSha1, WrappedKey: row.PdfWrappedKey, Size: row.PdfSize.Int64}
	}
	return d, json.Unmarshal(row.Fields, &d.Fields)
}

func ts(t time.Time) pgtype.Timestamptz { return pgtype.Timestamptz{Time: t, Valid: true} }
