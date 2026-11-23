package signup

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Shmeeshmee/salus/internal/signup/signupdb"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

type postgresRepository struct {
	db postgres.Querier
}

// NewPostgresRepository returns the Repository on db: salus_app for the API and the notifier (clinic_signup has no
// row-level security: a signup belongs to no clinic yet; clinic_setup_step shows the clinic set by SetOrganization),
// the owner for salusctl's list and approval.
func NewPostgresRepository(db postgres.Querier) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) q(ctx context.Context) *signupdb.Queries {
	return signupdb.New(postgres.Conn(ctx, r.db))
}

func (r *postgresRepository) Insert(ctx context.Context, in Request, tokenHash []byte) (string, error) {
	row, err := r.q(ctx).InsertSignup(ctx, signupdb.InsertSignupParams{
		Email: in.Email, ClinicName: in.ClinicName, Language: signupdb.Language(in.Language), TokenHash: tokenHash,
	})
	if err != nil {
		return "", err
	}
	return row.ID.String(), nil
}

func (r *postgresRepository) ByID(ctx context.Context, id string) (Signup, error) {
	u, ok := uuid(id)
	if !ok {
		return Signup{}, ErrNotFound
	}
	return one(r.q(ctx).SignupByID(ctx, u))
}

func (r *postgresRepository) ByTokenHash(ctx context.Context, hash []byte) (Signup, error) {
	return one(r.q(ctx).SignupByTokenHash(ctx, hash))
}

func (r *postgresRepository) SetToken(ctx context.Context, id string, tokenHash []byte) (bool, error) {
	u, ok := uuid(id)
	if !ok {
		return false, nil
	}
	n, err := r.q(ctx).SetSignupToken(ctx, signupdb.SetSignupTokenParams{ID: u, TokenHash: tokenHash})
	return n == 1, err
}

func (r *postgresRepository) MarkVerified(ctx context.Context, id, organizationID string, at time.Time) (bool, error) {
	u, ok := uuid(id)
	o, okOrg := uuid(organizationID)
	if !ok || !okOrg {
		return false, nil
	}
	n, err := r.q(ctx).VerifySignup(ctx, signupdb.VerifySignupParams{ID: u, OrganizationID: o, VerifiedAt: instant(at)})
	return n == 1, err
}

func (r *postgresRepository) Verified(ctx context.Context) ([]Signup, error) {
	rows, err := r.q(ctx).ListSignups(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Signup, 0, len(rows))
	for _, row := range rows {
		out = append(out, toSignup(row))
	}
	return out, nil
}

func (r *postgresRepository) Approve(ctx context.Context, organizationID string, at time.Time) (bool, error) {
	o, ok := uuid(organizationID)
	if !ok {
		return false, nil
	}
	_, err := r.q(ctx).ApproveSignup(ctx, signupdb.ApproveSignupParams{OrganizationID: o, ApprovedAt: instant(at)})
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

func (r *postgresRepository) SetOrganization(ctx context.Context, organizationID string) error {
	return postgres.SetLocal(ctx, "app.organization_id", organizationID)
}

func (r *postgresRepository) Steps(ctx context.Context, organizationID string) (map[Step]StepState, error) {
	o, ok := uuid(organizationID)
	if !ok {
		return nil, ErrNotFound
	}
	rows, err := r.q(ctx).SetupSteps(ctx, o)
	if err != nil {
		return nil, err
	}
	out := make(map[Step]StepState, len(rows))
	for _, row := range rows {
		out[Step(row.Step)] = StepState(row.State)
	}
	return out, nil
}

func (r *postgresRepository) PutStep(ctx context.Context, organizationID string, step Step, state StepState, at time.Time) error {
	o, ok := uuid(organizationID)
	if !ok {
		return ErrNotFound
	}
	if state == StepOpen {
		return r.q(ctx).DeleteSetupStep(ctx, signupdb.DeleteSetupStepParams{OrganizationID: o, Step: string(step)})
	}
	return r.q(ctx).PutSetupStep(ctx, signupdb.PutSetupStepParams{
		OrganizationID: o, Step: string(step), State: string(state), UpdatedAt: instant(at),
	})
}

func one(row signupdb.ClinicSignup, err error) (Signup, error) {
	if errors.Is(err, pgx.ErrNoRows) {
		return Signup{}, ErrNotFound
	}
	if err != nil {
		return Signup{}, err
	}
	return toSignup(row), nil
}

func toSignup(row signupdb.ClinicSignup) Signup {
	s := Signup{
		ID: row.ID.String(), CreatedAt: row.CreatedAt.Time, VerifiedAt: row.VerifiedAt.Time, ApprovedAt: row.ApprovedAt.Time,
		Email: row.Email, ClinicName: row.ClinicName, Language: string(row.Language),
	}
	if row.OrganizationID.Valid {
		s.OrganizationID = row.OrganizationID.String()
	}
	return s
}

func uuid(s string) (pgtype.UUID, bool) {
	var u pgtype.UUID
	return u, u.Scan(s) == nil
}

func instant(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

func (r *postgresRepository) DeleteUnverifiedBefore(ctx context.Context, before time.Time) (int64, error) {
	return r.q(ctx).DeleteUnverifiedSignupsBefore(ctx, instant(before))
}
