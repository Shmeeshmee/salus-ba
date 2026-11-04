package auth

import (
	"context"
	"errors"
	"time"

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
	u, err := uuidOf(id)
	if err != nil {
		return Account{}, err
	}
	row, err := r.q(ctx).AccountByID(ctx, u)
	return toAccount(authdb.AccountByEmailRow(row), err)
}

func (r *postgresRepository) SetPasswordHash(ctx context.Context, id, hash string) error {
	u, err := uuidOf(id)
	if err != nil {
		return err
	}
	n, err := r.q(ctx).SetPasswordHash(ctx, authdb.SetPasswordHashParams{ID: u, PasswordHash: hash})
	return oneRow(n, err)
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

// uuidOf parses id, or returns ErrNotFound: an id that is not a UUID names nothing.
func uuidOf(id string) (pgtype.UUID, error) {
	var u pgtype.UUID
	if u.Scan(id) != nil {
		return u, ErrNotFound
	}
	return u, nil
}

// optionalUUID is NULL for "".
func optionalUUID(id string) (pgtype.UUID, error) {
	if id == "" {
		return pgtype.UUID{}, nil
	}
	return uuidOf(id)
}

func timestamptz(t time.Time) pgtype.Timestamptz { return pgtype.Timestamptz{Time: t, Valid: true} }

func (r *postgresRepository) TOTPSecret(ctx context.Context, id string) ([]byte, error) {
	u, err := uuidOf(id)
	if err != nil {
		return nil, err
	}
	sealed, err := r.q(ctx).TOTPSecret(ctx, u)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return sealed, err
}

func (r *postgresRepository) SetTOTPSecret(ctx context.Context, id string, sealed []byte) error {
	u, err := uuidOf(id)
	if err != nil {
		return err
	}
	n, err := r.q(ctx).SetTOTPSecret(ctx, authdb.SetTOTPSecretParams{ID: u, TotpSecret: sealed})
	return oneRow(n, err)
}

func (r *postgresRepository) ConfirmMFA(ctx context.Context, id string, step int64, at time.Time) error {
	u, err := uuidOf(id)
	if err != nil {
		return err
	}
	n, err := r.q(ctx).ConfirmMFA(ctx, authdb.ConfirmMFAParams{ID: u, Step: step, EnrolledAt: timestamptz(at)})
	if err != nil || n == 1 {
		return err
	}
	if _, err := r.ByID(ctx, id); err != nil { // no row updated: unknown, or enrolled already
		return err
	}
	return ErrAlreadyEnrolled
}

func (r *postgresRepository) UseTOTPStep(ctx context.Context, id string, step int64) (bool, error) {
	u, err := uuidOf(id)
	if err != nil {
		return false, err
	}
	n, err := r.q(ctx).UseTOTPStep(ctx, authdb.UseTOTPStepParams{ID: u, Step: step})
	return n == 1, err
}

func (r *postgresRepository) CreateSession(ctx context.Context, s Session, tokenHash []byte, at time.Time) (string, error) {
	account, err := uuidOf(s.AccountID)
	if err != nil {
		return "", err
	}
	org, err := optionalUUID(s.OrganizationID)
	if err != nil {
		return "", err
	}
	id, err := r.q(ctx).CreateSession(ctx, authdb.CreateSessionParams{
		CreatedAt: timestamptz(at), ExpiresAt: timestamptz(s.ExpiresAt), AccountID: account, OrganizationID: org,
		MfaVerified: s.MFAVerified, TokenHash: tokenHash,
	})
	if err != nil {
		return "", err
	}
	return id.String(), nil
}

func (r *postgresRepository) SessionByTokenHash(ctx context.Context, tokenHash []byte, now time.Time) (Session, error) {
	row, err := r.q(ctx).SessionByTokenHash(ctx, authdb.SessionByTokenHashParams{TokenHash: tokenHash, Now: timestamptz(now)})
	if errors.Is(err, pgx.ErrNoRows) {
		return Session{}, ErrNotFound
	}
	if err != nil {
		return Session{}, err
	}
	s := Session{ID: row.ID.String(), AccountID: row.AccountID.String(), ExpiresAt: row.ExpiresAt.Time, MFAVerified: row.MfaVerified}
	if row.OrganizationID.Valid {
		s.OrganizationID = row.OrganizationID.String()
	}
	return s, nil
}

func (r *postgresRepository) RevokeSession(ctx context.Context, id string, at time.Time) error {
	u, err := uuidOf(id)
	if err != nil {
		return err
	}
	n, err := r.q(ctx).RevokeSession(ctx, authdb.RevokeSessionParams{ID: u, RevokedAt: timestamptz(at)})
	return oneRow(n, err)
}

// oneRow maps an update's row count: none means ErrNotFound.
func oneRow(n int64, err error) error {
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
