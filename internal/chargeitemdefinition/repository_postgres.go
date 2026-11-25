package chargeitemdefinition

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Shmeeshmee/salus/internal/chargeitemdefinition/chargeitemdefinitiondb"
	"github.com/Shmeeshmee/salus/internal/schedule"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

type postgresRepository struct {
	db postgres.Querier
}

// NewPostgresRepository returns the Repository on db, as salus_app: row-level security shows only the clinic set by
// SetOrganization, or the active services of listed clinics after SetDirectory.
func NewPostgresRepository(db postgres.Querier) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) q(ctx context.Context) *chargeitemdefinitiondb.Queries {
	return chargeitemdefinitiondb.New(postgres.Conn(ctx, r.db))
}

func (r *postgresRepository) SetOrganization(ctx context.Context, id string) error {
	return postgres.SetLocal(ctx, "app.organization_id", id)
}

func (r *postgresRepository) SetDirectory(ctx context.Context) error {
	return postgres.SetLocal(ctx, "app.directory", "public")
}

func (r *postgresRepository) List(ctx context.Context, org string) ([]Definition, error) {
	var u pgtype.UUID
	if u.Scan(org) != nil {
		return nil, nil
	}
	rows, err := r.q(ctx).Definitions(ctx, u)
	if err != nil {
		return nil, err
	}
	out := make([]Definition, len(rows))
	ids := make([]pgtype.UUID, len(rows))
	at := make(map[pgtype.UUID]int, len(rows))
	for i, row := range rows {
		modes := make([]schedule.VisitMode, len(row.Modes))
		for j, m := range row.Modes {
			modes[j] = schedule.VisitMode(m)
		}
		out[i] = Definition{
			LastUpdated: row.UpdatedAt.Time, ID: row.ID.String(), OrganizationID: row.OrganizationID.String(), Name: row.Name,
			Code: row.Code.String, Currency: row.Currency, Modes: modes, PriceMinor: row.PriceMinor,
			DurationMinutes: row.DurationMinutes.Int32, VersionID: row.VersionID, Active: row.Active,
		}
		if row.DeviceID.Valid {
			out[i].DeviceID = row.DeviceID.String()
		}
		if row.VatRateBp.Valid {
			out[i].VATRateBP = &row.VatRateBp.Int32
		}
		ids[i], at[row.ID] = row.ID, i
	}
	translations, err := r.q(ctx).DefinitionTranslations(ctx, ids)
	if err != nil {
		return nil, err
	}
	for _, t := range translations {
		d := &out[at[t.ChargeItemDefinitionID]]
		d.Translations = append(d.Translations, Translation{Language: string(t.Language), Name: t.Name})
	}
	return out, nil
}

func (r *postgresRepository) Create(ctx context.Context, d Definition, at time.Time) (Definition, error) {
	var org pgtype.UUID
	if err := org.Scan(d.OrganizationID); err != nil {
		return Definition{}, err
	}
	row, err := r.q(ctx).InsertDefinition(ctx, chargeitemdefinitiondb.InsertDefinitionParams{
		At: pgtype.Timestamptz{Time: at, Valid: true}, OrganizationID: org, Name: d.Name, Code: text(d.Code),
		PriceMinor: d.PriceMinor, Currency: d.Currency, DurationMinutes: duration(d), VatRateBp: vat(d), Modes: modes(d),
		Active: d.Active,
	})
	if err != nil {
		return Definition{}, err
	}
	d.ID, d.VersionID, d.LastUpdated = row.ID.String(), row.VersionID, at
	if err := r.setDevice(ctx, row.ID, d.DeviceID); err != nil {
		return Definition{}, err
	}
	return d, r.putTranslations(ctx, row.ID, d.Translations)
}

func (r *postgresRepository) Update(ctx context.Context, d Definition, at time.Time) (Definition, error) {
	var id pgtype.UUID
	if id.Scan(d.ID) != nil {
		return Definition{}, ErrNotFound
	}
	version, err := r.q(ctx).UpdateDefinition(ctx, chargeitemdefinitiondb.UpdateDefinitionParams{
		Name: d.Name, Code: text(d.Code), PriceMinor: d.PriceMinor, Currency: d.Currency, DurationMinutes: duration(d),
		VatRateBp: vat(d), Modes: modes(d), Active: d.Active, At: pgtype.Timestamptz{Time: at, Valid: true}, ID: id,
		VersionID: d.VersionID,
	})
	if errors.Is(err, pgx.ErrNoRows) { // not the scope's, or at another version
		exists, err := r.q(ctx).DefinitionExists(ctx, id)
		switch {
		case err != nil:
			return Definition{}, err
		case exists:
			return Definition{}, ErrConflict
		}
		return Definition{}, ErrNotFound
	}
	if err != nil {
		return Definition{}, err
	}
	d.VersionID, d.LastUpdated = version, at
	if err := r.setDevice(ctx, id, d.DeviceID); err != nil {
		return Definition{}, err
	}
	if err := r.q(ctx).DeleteDefinitionTranslations(ctx, id); err != nil {
		return Definition{}, err
	}
	return d, r.putTranslations(ctx, id, d.Translations)
}

func (r *postgresRepository) putTranslations(ctx context.Context, id pgtype.UUID, ts []Translation) error {
	for _, t := range ts {
		err := r.q(ctx).InsertDefinitionTranslation(ctx, chargeitemdefinitiondb.InsertDefinitionTranslationParams{
			ChargeItemDefinitionID: id, Language: chargeitemdefinitiondb.Language(t.Language), Name: t.Name,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func text(s string) pgtype.Text { return pgtype.Text{String: s, Valid: s != ""} }

func duration(d Definition) pgtype.Int4 {
	return pgtype.Int4{Int32: d.DurationMinutes, Valid: d.DurationMinutes != 0}
}

func vat(d Definition) pgtype.Int4 {
	if d.VATRateBP == nil {
		return pgtype.Int4{}
	}
	return pgtype.Int4{Int32: *d.VATRateBP, Valid: true}
}

func modes(d Definition) []string {
	out := make([]string, len(d.Modes))
	for i, m := range d.Modes {
		out[i] = string(m)
	}
	return out
}

// The device a service needs (RM1, docs/clinic-cms/WAVE6.md U4).

// foreignKeyViolation is Postgres's answer to a device that is not the service's clinic's.
const foreignKeyViolation = "23503"

// setDevice sets the device the service id needs, none for "" (RM1); ErrInvalid for an id that is not a device of
// the service's clinic.
func (r *postgresRepository) setDevice(ctx context.Context, id pgtype.UUID, device string) error {
	var u pgtype.UUID
	if device != "" && u.Scan(device) != nil {
		return ErrInvalid
	}
	err := r.q(ctx).SetDefinitionDevice(ctx, chargeitemdefinitiondb.SetDefinitionDeviceParams{ID: id, DeviceID: u})
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == foreignKeyViolation {
		return ErrInvalid
	}
	return err
}
