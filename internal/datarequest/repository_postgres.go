package datarequest

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Shmeeshmee/salus/internal/datarequest/datarequestdb"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

type postgresRepository struct {
	db postgres.Querier
}

// NewPostgresRepository returns the Repository on db (salus_app): row-level security shows the requests of the scope
// set on the transaction.
func NewPostgresRepository(db postgres.Querier) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) q(ctx context.Context) *datarequestdb.Queries {
	return datarequestdb.New(postgres.Conn(ctx, r.db))
}

func (r *postgresRepository) SetAccount(ctx context.Context, accountID string) error {
	return postgres.SetLocal(ctx, "app.account_id", accountID)
}

func (r *postgresRepository) SetOrganization(ctx context.Context, organizationID string) error {
	return postgres.SetLocal(ctx, "app.organization_id", organizationID)
}

func (r *postgresRepository) SetDeadlines(ctx context.Context) error {
	return postgres.SetLocal(ctx, "app.data_request_deadlines", "all")
}

// Insert stores x and its first history entry. A profile or clinic the scope may not file for (row-level security), or
// one that does not exist, is ErrNotFound.
func (r *postgresRepository) Insert(ctx context.Context, x Request) (Request, error) {
	filedBy, ok := uuid(x.FiledBy)
	if !ok {
		return Request{}, ErrNotFound
	}
	account, okAccount := optionalUUID(x.AccountID)
	patient, okPatient := optionalUUID(x.PatientID)
	org, okOrg := optionalUUID(x.OrganizationID)
	if !okAccount || !okPatient || !okOrg {
		return Request{}, ErrNotFound
	}
	row, err := r.q(ctx).InsertRequest(ctx, datarequestdb.InsertRequestParams{
		AccountID: account, PatientID: patient, OrganizationID: org, FiledByAccountID: filedBy, Kind: string(x.Kind),
		ReceivedAt: instant(x.ReceivedAt), DueAt: instant(x.DueAt),
	})
	if err != nil {
		return Request{}, refused(err)
	}
	if err := r.q(ctx).InsertChange(ctx, datarequestdb.InsertChangeParams{
		DataRequestID: row.ID, ChangedByAccountID: filedBy, Status: row.Status, ChangedAt: row.ReceivedAt,
	}); err != nil {
		return Request{}, err
	}
	return toRequest(row), nil
}

// refused maps a refusal of row-level security or a missing reference to ErrNotFound.
func refused(err error) error {
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && (pgErr.Code == "42501" || pgErr.Code == "23503") {
		return ErrNotFound
	}
	return err
}

func (r *postgresRepository) ByID(ctx context.Context, id string) (Request, error) {
	u, ok := uuid(id)
	if !ok {
		return Request{}, ErrNotFound
	}
	row, err := r.q(ctx).RequestByID(ctx, u)
	if errors.Is(err, pgx.ErrNoRows) {
		return Request{}, ErrNotFound
	}
	if err != nil {
		return Request{}, err
	}
	return toRequest(row), nil
}

func (r *postgresRepository) ForAccount(ctx context.Context, accountID string) ([]Request, error) {
	u, ok := uuid(accountID)
	if !ok {
		return nil, nil
	}
	return many(r.q(ctx).RequestsForAccount(ctx, u))
}

func (r *postgresRepository) ForClinic(ctx context.Context, organizationID string) ([]Request, error) {
	u, ok := uuid(organizationID)
	if !ok {
		return nil, nil
	}
	return many(r.q(ctx).RequestsForClinic(ctx, u))
}

// Update stores x if its version is still x.Version-1, with a history entry when its status changed: at its answer, or
// at the transaction's time for a withdrawal.
func (r *postgresRepository) Update(ctx context.Context, x Request, changedBy string) (bool, error) {
	old, err := r.ByID(ctx, x.ID)
	if errors.Is(err, ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	n, err := r.q(ctx).UpdateRequest(ctx, datarequestdb.UpdateRequestParams{
		ID: mustUUID(x.ID), Version: int32(x.Version - 1), Status: string(x.Status), //nolint:gosec // a version is small
		ExtendedUntil: optionalInstant(x.ExtendedUntil), ExtensionReason: text(string(x.ExtensionReason)),
		AnsweredAt: optionalInstant(x.AnsweredAt), AnswerCode: text(string(x.AnswerCode)),
	})
	if err != nil || n != 1 {
		return false, err
	}
	if old.Status == x.Status {
		return true, nil
	}
	by, _ := optionalUUID(changedBy)
	return true, r.q(ctx).InsertChange(ctx, datarequestdb.InsertChangeParams{
		DataRequestID: mustUUID(x.ID), ChangedByAccountID: by, Status: string(x.Status), ChangedAt: optionalInstant(x.AnsweredAt),
	})
}

func (r *postgresRepository) History(ctx context.Context, id string) ([]Change, error) {
	u, ok := uuid(id)
	if !ok {
		return nil, nil
	}
	rows, err := r.q(ctx).RequestHistory(ctx, u)
	if err != nil {
		return nil, err
	}
	out := make([]Change, len(rows))
	for i, c := range rows {
		out[i] = Change{
			ID: c.ID.String(), RequestID: c.DataRequestID.String(), ChangedBy: optionalString(c.ChangedByAccountID),
			Status: Status(c.Status), ChangedAt: c.ChangedAt.Time,
		}
	}
	return out, nil
}

func (r *postgresRepository) OpenDeadlines(ctx context.Context, before time.Time) ([]Deadline, error) {
	rows, err := r.q(ctx).OpenDeadlines(ctx, instant(before))
	if err != nil {
		return nil, err
	}
	out := make([]Deadline, len(rows))
	for i, row := range rows {
		out[i] = Deadline{Request: toRequest(row), WarnedAt: row.DeadlineWarnedAt.Time}
	}
	return out, nil
}

func (r *postgresRepository) MarkWarned(ctx context.Context, id string, seen, at time.Time) (bool, error) {
	u, ok := uuid(id)
	if !ok {
		return false, nil
	}
	n, err := r.q(ctx).MarkWarned(ctx, datarequestdb.MarkWarnedParams{ID: u, WarnedAt: instant(at), Seen: optionalInstant(seen)})
	return n == 1, err
}

func many(rows []datarequestdb.DataRequest, err error) ([]Request, error) {
	if err != nil {
		return nil, err
	}
	out := make([]Request, len(rows))
	for i, row := range rows {
		out[i] = toRequest(row)
	}
	return out, nil
}

func toRequest(row datarequestdb.DataRequest) Request {
	return Request{
		ID: row.ID.String(), AccountID: optionalString(row.AccountID), PatientID: optionalString(row.PatientID),
		OrganizationID: optionalString(row.OrganizationID), FiledBy: row.FiledByAccountID.String(),
		Kind: Kind(row.Kind), Status: Status(row.Status), ReceivedAt: row.ReceivedAt.Time, DueAt: row.DueAt.Time,
		ExtendedUntil: row.ExtendedUntil.Time, ExtensionReason: ExtensionReason(row.ExtensionReason.String),
		AnsweredAt: row.AnsweredAt.Time, AnswerCode: AnswerCode(row.AnswerCode.String), Version: int(row.Version),
	}
}

func uuid(s string) (pgtype.UUID, bool) {
	var u pgtype.UUID
	return u, u.Scan(s) == nil
}

func mustUUID(s string) pgtype.UUID {
	u, _ := uuid(s)
	return u
}

// optionalUUID is null for "", and false for a string that is no UUID.
func optionalUUID(s string) (pgtype.UUID, bool) {
	if s == "" {
		return pgtype.UUID{}, true
	}
	return uuid(s)
}

func optionalString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	return u.String()
}

func instant(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

func optionalInstant(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: !t.IsZero()}
}

func text(s string) pgtype.Text {
	return pgtype.Text{String: s, Valid: s != ""}
}
