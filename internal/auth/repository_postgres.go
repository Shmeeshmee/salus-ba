package auth

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Shmeeshmee/salus/internal/auth/authdb"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

type postgresRepository struct {
	db postgres.Querier
}

// NewPostgresRepository returns the Repository on db, using the transaction in the context when there is one.
func NewPostgresRepository(db postgres.Querier) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) q(ctx context.Context) *authdb.Queries {
	return authdb.New(postgres.Conn(ctx, r.db))
}

func (r *postgresRepository) Create(ctx context.Context, a Account) (string, error) {
	id, err := r.q(ctx).CreateAccount(ctx, authdb.CreateAccountParams{
		Email: a.Email, PasswordHash: a.PasswordHash, Role: string(a.Role),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrEmailTaken
	}
	if err != nil {
		return "", err
	}
	return id.String(), nil
}

func (r *postgresRepository) ByEmail(ctx context.Context, email string) (Account, error) {
	row, err := r.q(ctx).AccountByEmail(ctx, email)
	return toAccount(row, err)
}

func (r *postgresRepository) ByID(ctx context.Context, id string) (Account, error) {
	var u pgtype.UUID
	if u.Scan(id) != nil {
		return Account{}, ErrNotFound
	}
	row, err := r.q(ctx).AccountByID(ctx, u)
	return toAccount(authdb.AccountByEmailRow(row), err)
}

func (r *postgresRepository) SetPasswordHash(ctx context.Context, id, hash string) error {
	var u pgtype.UUID
	if u.Scan(id) != nil {
		return ErrNotFound
	}
	n, err := r.q(ctx).SetPasswordHash(ctx, authdb.SetPasswordHashParams{ID: u, PasswordHash: hash})
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// toAccount maps a row; both account queries select the same columns.
func toAccount(row authdb.AccountByEmailRow, err error) (Account, error) {
	if errors.Is(err, pgx.ErrNoRows) {
		return Account{}, ErrNotFound
	}
	if err != nil {
		return Account{}, err
	}
	a := Account{
		ID:           row.ID.String(),
		Email:        row.Email,
		Role:         Role(row.Role),
		PasswordHash: row.PasswordHash,
	}
	if row.OrganizationID.Valid {
		a.OrganizationID = row.OrganizationID.String()
	}
	if row.MfaEnrolledAt.Valid {
		a.MFAEnrolledAt = row.MfaEnrolledAt.Time
	}
	return a, nil
}
