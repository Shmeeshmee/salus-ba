package notification

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Shmeeshmee/salus/internal/notification/notificationdb"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/push"
)

type postgresRepository struct {
	db postgres.Querier
}

// NewPostgresRepository returns the Repository on db, using the transaction in the context when there is one. As
// salus_app, row-level security shows only the endpoints and dead letters of the clinic set by SetClinic.
func NewPostgresRepository(db postgres.Querier) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) q(ctx context.Context) *notificationdb.Queries {
	return notificationdb.New(postgres.Conn(ctx, r.db))
}

// uid parses id, or returns ErrNotFound for one that is not a UUID.
func uid(id string) (pgtype.UUID, error) {
	var u pgtype.UUID
	if u.Scan(id) != nil {
		return u, ErrNotFound
	}
	return u, nil
}

func (r *postgresRepository) RegisterDevice(ctx context.Context, d Device, at time.Time) error {
	account, err := uid(d.AccountID)
	if err != nil {
		return err
	}
	return r.q(ctx).RegisterDevice(ctx, notificationdb.RegisterDeviceParams{
		AccountID: account, Platform: string(d.Platform), Token: d.Token, UpdatedAt: pgtype.Timestamptz{Time: at, Valid: true},
	})
}

func (r *postgresRepository) UnregisterDevice(ctx context.Context, accountID, token string) error {
	account, err := uid(accountID)
	if err != nil {
		return err
	}
	return r.q(ctx).UnregisterDevice(ctx, notificationdb.UnregisterDeviceParams{AccountID: account, Token: token})
}

func (r *postgresRepository) Devices(ctx context.Context, accountID string) ([]Device, error) {
	account, err := uid(accountID)
	if err != nil {
		return nil, nil
	}
	rows, err := r.q(ctx).DevicesOfAccount(ctx, account)
	if err != nil {
		return nil, err
	}
	out := make([]Device, 0, len(rows))
	for _, row := range rows {
		out = append(out, Device{ID: row.ID.String(), AccountID: row.AccountID.String(), Token: row.Token, Platform: push.Platform(row.Platform)})
	}
	return out, nil
}

func (r *postgresRepository) Device(ctx context.Context, id string) (Device, error) {
	u, err := uid(id)
	if err != nil {
		return Device{}, err
	}
	row, err := r.q(ctx).DeviceByID(ctx, u)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return Device{}, ErrNotFound
	case err != nil:
		return Device{}, err
	}
	return Device{ID: row.ID.String(), AccountID: row.AccountID.String(), Token: row.Token, Platform: push.Platform(row.Platform)}, nil
}

func (r *postgresRepository) DeleteDevice(ctx context.Context, id, token string) error {
	u, err := uid(id)
	if err != nil {
		return nil
	}
	return r.q(ctx).DeleteDevice(ctx, notificationdb.DeleteDeviceParams{ID: u, Token: token})
}

func (r *postgresRepository) SetClinic(ctx context.Context, id string) error {
	return postgres.SetLocal(ctx, "app.organization_id", id)
}

func (r *postgresRepository) LockEndpoints(ctx context.Context, org string) error {
	return r.q(ctx).LockWebhookEndpoints(ctx, org)
}

func (r *postgresRepository) CreateEndpoint(ctx context.Context, e Endpoint) (Endpoint, error) {
	id, err := uid(e.ID)
	if err != nil {
		return Endpoint{}, err
	}
	org, err := uid(e.OrganizationID)
	if err != nil {
		return Endpoint{}, err
	}
	by, err := uid(e.CreatedBy)
	if err != nil {
		return Endpoint{}, err
	}
	at, err := r.q(ctx).CreateWebhookEndpoint(ctx, notificationdb.CreateWebhookEndpointParams{
		ID: id, OrganizationID: org, CreatedBy: by, IncludeClinical: e.IncludeClinical, Url: e.URL, Events: e.Events, SecretSealed: e.SecretSealed,
	})
	if err != nil {
		return Endpoint{}, err
	}
	e.CreatedAt = at.Time
	return e, nil
}

func (r *postgresRepository) Endpoints(ctx context.Context, org string) ([]Endpoint, error) {
	u, err := uid(org)
	if err != nil {
		return nil, nil
	}
	rows, err := r.q(ctx).WebhookEndpoints(ctx, u)
	if err != nil {
		return nil, err
	}
	out := make([]Endpoint, 0, len(rows))
	for _, row := range rows {
		out = append(out, toEndpoint(notificationdb.WebhookEndpointByIDRow(row)))
	}
	return out, nil
}

func (r *postgresRepository) Endpoint(ctx context.Context, id string) (Endpoint, error) {
	u, err := uid(id)
	if err != nil {
		return Endpoint{}, err
	}
	row, err := r.q(ctx).WebhookEndpointByID(ctx, u)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return Endpoint{}, ErrNotFound
	case err != nil:
		return Endpoint{}, err
	}
	return toEndpoint(row), nil
}

func toEndpoint(row notificationdb.WebhookEndpointByIDRow) Endpoint {
	return Endpoint{
		CreatedAt: row.CreatedAt.Time, ID: row.ID.String(), OrganizationID: row.OrganizationID.String(), URL: row.Url,
		Events: row.Events, SecretSealed: row.SecretSealed, IncludeClinical: row.IncludeClinical,
	}
}

func (r *postgresRepository) DeleteEndpoint(ctx context.Context, id string) error {
	u, err := uid(id)
	if err != nil {
		return err
	}
	n, err := r.q(ctx).DeleteWebhookEndpoint(ctx, u)
	switch {
	case err != nil:
		return err
	case n == 0:
		return ErrNotFound
	}
	return nil
}

func (r *postgresRepository) AddDeadLetter(ctx context.Context, d DeadLetter) error {
	var ids [5]pgtype.UUID
	for i, s := range []string{d.EndpointID, d.OrganizationID, d.EventID, d.ConversationID, d.MessageID} {
		var err error
		if ids[i], err = uid(s); err != nil {
			return err
		}
	}
	status := pgtype.Int4{Int32: int32(d.LastStatus), Valid: d.LastStatus != 0}
	return r.q(ctx).AddDeadLetter(ctx, notificationdb.AddDeadLetterParams{
		EndpointID: ids[0], OrganizationID: ids[1], EventID: ids[2], ConversationID: ids[3], MessageID: ids[4],
		Attempts: int32(d.Attempts), LastStatus: status, EventType: d.EventType,
	})
}

func (r *postgresRepository) DeleteDeadLettersBefore(ctx context.Context, before time.Time) (int64, error) {
	return r.q(ctx).DeleteDeadLettersBefore(ctx, pgtype.Timestamptz{Time: before, Valid: true})
}
