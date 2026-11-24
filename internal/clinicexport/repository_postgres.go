package clinicexport

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Shmeeshmee/salus/internal/clinicexport/clinicexportdb"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

type postgresRepository struct {
	db postgres.Querier
}

// NewPostgresRepository returns the Repository on db: salus_app for the API and the build job (row-level security
// shows the clinic set by SetOrganization), salus_retention for the purger.
func NewPostgresRepository(db postgres.Querier) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) q(ctx context.Context) *clinicexportdb.Queries {
	return clinicexportdb.New(postgres.Conn(ctx, r.db))
}

func (r *postgresRepository) SetOrganization(ctx context.Context, id string) error {
	return postgres.SetLocal(ctx, "app.organization_id", id)
}

func (r *postgresRepository) Insert(ctx context.Context, e Export) (Export, bool, error) {
	org, okOrg := uuid(e.OrganizationID)
	account, okAccount := uuid(e.RequestedBy)
	if !okOrg || !okAccount {
		return Export{}, false, ErrNotFound
	}
	row, err := r.q(ctx).InsertExport(ctx, clinicexportdb.InsertExportParams{
		OrganizationID: org, RequestedByAccountID: account, RequestedAt: instant(e.RequestedAt), ExpiresAt: instant(e.ExpiresAt),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Export{}, false, nil
	}
	if err != nil {
		return Export{}, false, err
	}
	return toExport(row), true, nil
}

func (r *postgresRepository) Queued(ctx context.Context, org string) (Export, error) {
	o, ok := uuid(org)
	if !ok {
		return Export{}, ErrNotFound
	}
	return one(r.q(ctx).QueuedExport(ctx, o))
}

func (r *postgresRepository) ByID(ctx context.Context, id string) (Export, error) {
	u, ok := uuid(id)
	if !ok {
		return Export{}, ErrNotFound
	}
	return one(r.q(ctx).ExportByID(ctx, u))
}

func (r *postgresRepository) MarkReady(ctx context.Context, e Export) (bool, error) {
	u, ok := uuid(e.ID)
	if !ok {
		return false, nil
	}
	n, err := r.q(ctx).MarkExportReady(ctx, clinicexportdb.MarkExportReadyParams{
		ID: u, ReadyAt: instant(e.ReadyAt), Size: pgtype.Int8{Int64: e.Size, Valid: true},
		KekID: pgtype.Text{String: e.KEKID, Valid: true}, WrappedKey: e.WrappedKey,
	})
	return n == 1, err
}

func (r *postgresRepository) MarkFailed(ctx context.Context, id string) (bool, error) {
	u, ok := uuid(id)
	if !ok {
		return false, nil
	}
	n, err := r.q(ctx).MarkExportFailed(ctx, u)
	return n == 1, err
}

func (r *postgresRepository) RequestedBefore(ctx context.Context, before time.Time, limit int32) ([]string, error) {
	rows, err := r.q(ctx).ExportsRequestedBefore(ctx, clinicexportdb.ExportsRequestedBeforeParams{Before: instant(before), Max: limit})
	if err != nil {
		return nil, err
	}
	out := make([]string, len(rows))
	for i, id := range rows {
		out[i] = id.String()
	}
	return out, nil
}

func (r *postgresRepository) Delete(ctx context.Context, id string) (bool, error) {
	u, ok := uuid(id)
	if !ok {
		return false, nil
	}
	n, err := r.q(ctx).DeleteExport(ctx, u)
	return n == 1, err
}

func one(row clinicexportdb.ClinicExport, err error) (Export, error) {
	if errors.Is(err, pgx.ErrNoRows) {
		return Export{}, ErrNotFound
	}
	if err != nil {
		return Export{}, err
	}
	return toExport(row), nil
}

func toExport(row clinicexportdb.ClinicExport) Export {
	return Export{
		ID: row.ID.String(), OrganizationID: row.OrganizationID.String(), RequestedBy: row.RequestedByAccountID.String(),
		RequestedAt: row.RequestedAt.Time, ExpiresAt: row.ExpiresAt.Time, ReadyAt: row.ReadyAt.Time, Status: Status(row.Status),
		Size: row.Size.Int64, KEKID: row.KekID.String, WrappedKey: row.WrappedKey,
	}
}

func uuid(s string) (pgtype.UUID, bool) {
	var u pgtype.UUID
	return u, u.Scan(s) == nil
}

func instant(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}
