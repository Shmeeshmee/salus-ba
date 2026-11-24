package datarequest

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Shmeeshmee/salus/internal/datarequest/datarequestexportdb"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

type exportRepository struct {
	db postgres.Querier
}

// NewExportRepository returns the ExportRepository on db: salus_app for the API and the build job (row-level security
// shows the account set by SetAccount), salus_retention for the purger.
func NewExportRepository(db postgres.Querier) ExportRepository {
	return &exportRepository{db: db}
}

func (r *exportRepository) q(ctx context.Context) *datarequestexportdb.Queries {
	return datarequestexportdb.New(postgres.Conn(ctx, r.db))
}

func (r *exportRepository) SetAccount(ctx context.Context, accountID string) error {
	return postgres.SetLocal(ctx, "app.account_id", accountID)
}

func (r *exportRepository) InsertExport(ctx context.Context, x StoredExport) (bool, error) {
	id, okID := exportUUID(x.RequestID)
	account, okAccount := exportUUID(x.AccountID)
	if !okID || !okAccount {
		return false, ErrNotFound
	}
	_, err := r.q(ctx).InsertExport(ctx, datarequestexportdb.InsertExportParams{
		DataRequestID: id, AccountID: account, RequestedAt: exportInstant(x.RequestedAt),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

func (r *exportRepository) ExportOf(ctx context.Context, requestID string) (StoredExport, error) {
	id, ok := exportUUID(requestID)
	if !ok {
		return StoredExport{}, ErrNotFound
	}
	row, err := r.q(ctx).ExportOf(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return StoredExport{}, ErrNotFound
	}
	if err != nil {
		return StoredExport{}, err
	}
	out := StoredExport{
		RequestedAt: row.RequestedAt.Time, ReadyAt: row.ReadyAt.Time, ExpiresAt: row.ExpiresAt.Time,
		RequestID: row.DataRequestID.String(), AccountID: row.AccountID.String(), Status: ExportStatus(row.Status),
		Kind: Kind(row.Kind), KEKID: row.KekID.String, ObjectKey: row.ObjectKey.String, WrappedKey: row.WrappedKey,
		Size: row.Size.Int64,
	}
	if row.OrganizationID.Valid {
		out.OrganizationID = row.OrganizationID.String()
	}
	return out, nil
}

func (r *exportRepository) ClaimExport(ctx context.Context, requestID, key string) (string, bool, error) {
	id, ok := exportUUID(requestID)
	if !ok {
		return "", false, nil
	}
	previous, err := r.q(ctx).ClaimExport(ctx, datarequestexportdb.ClaimExportParams{
		DataRequestID: id, ObjectKey: pgtype.Text{String: key, Valid: true},
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	return previous.String, err == nil, err
}

func (r *exportRepository) MarkExportReady(ctx context.Context, x StoredExport) (bool, error) {
	id, ok := exportUUID(x.RequestID)
	if !ok {
		return false, nil
	}
	n, err := r.q(ctx).MarkExportReady(ctx, datarequestexportdb.MarkExportReadyParams{
		DataRequestID: id, ReadyAt: exportInstant(x.ReadyAt), ExpiresAt: exportInstant(x.ExpiresAt),
		Size: pgtype.Int8{Int64: x.Size, Valid: true}, KekID: pgtype.Text{String: x.KEKID, Valid: true}, WrappedKey: x.WrappedKey,
		ObjectKey: pgtype.Text{String: x.ObjectKey, Valid: true},
	})
	return n == 1, err
}

func (r *exportRepository) MarkClaimFailed(ctx context.Context, x StoredExport) (bool, error) {
	id, ok := exportUUID(x.RequestID)
	if !ok {
		return false, nil
	}
	n, err := r.q(ctx).MarkClaimFailed(ctx, datarequestexportdb.MarkClaimFailedParams{
		DataRequestID: id, ObjectKey: pgtype.Text{String: x.ObjectKey, Valid: true},
	})
	return n == 1, err
}

func (r *exportRepository) MarkExportFailed(ctx context.Context, requestID string) (bool, error) {
	id, ok := exportUUID(requestID)
	if !ok {
		return false, nil
	}
	n, err := r.q(ctx).MarkExportFailed(ctx, id)
	return n == 1, err
}

func (r *exportRepository) Fulfil(ctx context.Context, requestID string) (bool, error) {
	id, ok := exportUUID(requestID)
	if !ok {
		return false, nil
	}
	return r.q(ctx).FulfilExportRequest(ctx, id)
}

func (r *exportRepository) ExportsDoneBefore(ctx context.Context, before time.Time, limit int32) ([]StoredExport, error) {
	rows, err := r.q(ctx).ExportsDoneBefore(ctx, datarequestexportdb.ExportsDoneBeforeParams{Before: exportInstant(before), Max: limit})
	if err != nil {
		return nil, err
	}
	out := make([]StoredExport, len(rows))
	for i, row := range rows {
		out[i] = StoredExport{RequestID: row.DataRequestID.String(), ObjectKey: row.ObjectKey.String}
	}
	return out, nil
}

func (r *exportRepository) DeleteExport(ctx context.Context, requestID string) (bool, error) {
	id, ok := exportUUID(requestID)
	if !ok {
		return false, nil
	}
	n, err := r.q(ctx).DeleteExport(ctx, id)
	return n == 1, err
}

func exportUUID(s string) (pgtype.UUID, bool) {
	var u pgtype.UUID
	return u, u.Scan(s) == nil
}

func exportInstant(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}
