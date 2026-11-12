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
	org, err := optionalUUID(a.OrganizationID)
	if err != nil {
		return "", err
	}
	id, err := r.q(ctx).CreateAccount(ctx, authdb.CreateAccountParams{
		Email: a.Email, PasswordHash: a.PasswordHash, Role: string(a.Role), OrganizationID: org,
		Language: nullLanguage(a.Language), DetectedLanguage: nullLanguage(a.DetectedLanguage),
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

func (r *postgresRepository) ByPractitioner(ctx context.Context, practitionerID string) (Account, error) {
	u, err := uuidOf(practitionerID)
	if err != nil {
		return Account{}, err
	}
	row, err := r.q(ctx).AccountByPractitioner(ctx, u)
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

func (r *postgresRepository) SetLanguage(ctx context.Context, id, language string) error {
	u, err := uuidOf(id)
	if err != nil {
		return err
	}
	n, err := r.q(ctx).SetAccountLanguage(ctx, authdb.SetAccountLanguageParams{ID: u, Language: nullLanguage(language)})
	return oneRow(n, err)
}

func (r *postgresRepository) SetDetectedLanguage(ctx context.Context, id, language string) error {
	u, err := uuidOf(id)
	if err != nil {
		return err
	}
	n, err := r.q(ctx).SetDetectedLanguage(ctx, authdb.SetDetectedLanguageParams{ID: u, DetectedLanguage: nullLanguage(language)})
	return oneRow(n, err)
}

// nullLanguage is a language tag as the column takes it: "" is null.
func nullLanguage(tag string) authdb.NullLanguage {
	return authdb.NullLanguage{Language: authdb.Language(tag), Valid: tag != ""}
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
		// An unset language scans as "": no choice, no match.
		Language:         string(row.Language.Language),
		DetectedLanguage: string(row.DetectedLanguage.Language),
		TermReviewer:     row.TermReviewer,
	}
	if row.OrganizationID.Valid {
		a.OrganizationID = row.OrganizationID.String()
	}
	if row.PractitionerID.Valid {
		a.PractitionerID = row.PractitionerID.String()
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

func (r *postgresRepository) RevokeSessions(ctx context.Context, accountID string, at time.Time) error {
	u, err := uuidOf(accountID)
	if err != nil {
		return err
	}
	return r.q(ctx).RevokeSessions(ctx, authdb.RevokeSessionsParams{AccountID: u, RevokedAt: timestamptz(at)})
}

func (r *postgresRepository) ResetMFA(ctx context.Context, id string) error {
	u, err := uuidOf(id)
	if err != nil {
		return err
	}
	n, err := r.q(ctx).ResetMFA(ctx, u)
	return oneRow(n, err)
}

func (r *postgresRepository) SetPractitioner(ctx context.Context, id, practitionerID string) error {
	u, err := uuidOf(id)
	if err != nil {
		return err
	}
	p, err := uuidOf(practitionerID)
	if err != nil {
		return err
	}
	n, err := r.q(ctx).SetPractitioner(ctx, authdb.SetPractitionerParams{ID: u, PractitionerID: p})
	return oneRow(n, err)
}

func (r *postgresRepository) SetTermReviewer(ctx context.Context, id string, reviewer bool) error {
	u, err := uuidOf(id)
	if err != nil {
		return err
	}
	if reviewer {
		return r.q(ctx).GrantTermReviewer(ctx, u)
	}
	return r.q(ctx).RevokeTermReviewer(ctx, u)
}

func (r *postgresRepository) SetOrganization(ctx context.Context, id string) error {
	return postgres.SetLocal(ctx, "app.organization_id", id)
}

func (r *postgresRepository) PutInvite(ctx context.Context, inv Invite) (string, error) {
	org, err := uuidOf(inv.OrganizationID)
	if err != nil {
		return "", errors.New("auth: the clinic id is not a UUID")
	}
	by, err := optionalUUID(inv.InvitedBy)
	if err != nil {
		return "", errors.New("auth: the inviter id is not a UUID")
	}
	id, err := r.q(ctx).PutInvite(ctx, authdb.PutInviteParams{
		CreatedAt: timestamptz(inv.CreatedAt), ExpiresAt: timestamptz(inv.ExpiresAt), OrganizationID: org, InvitedBy: by,
		Role: string(inv.Role), Email: inv.Email, GivenName: inv.GivenName, FamilyName: inv.FamilyName,
	})
	if err != nil {
		return "", err
	}
	return id.String(), nil
}

func (r *postgresRepository) OpenInvites(ctx context.Context, org string, now time.Time) ([]Invite, error) {
	u, err := uuidOf(org)
	if err != nil {
		return nil, nil
	}
	rows, err := r.q(ctx).OpenInvites(ctx, authdb.OpenInvitesParams{OrganizationID: u, Now: timestamptz(now)})
	if err != nil {
		return nil, err
	}
	invites := make([]Invite, 0, len(rows))
	for _, row := range rows {
		invites = append(invites, toInvite(authdb.InviteByIDRow(row)))
	}
	return invites, nil
}

func (r *postgresRepository) InviteByID(ctx context.Context, id string) (Invite, error) {
	u, err := uuidOf(id)
	if err != nil {
		return Invite{}, ErrInvalidInvite
	}
	row, err := r.q(ctx).InviteByID(ctx, u)
	return oneInvite(row, err)
}

func (r *postgresRepository) InviteByTokenHash(ctx context.Context, tokenHash []byte) (Invite, error) {
	row, err := r.q(ctx).InviteByTokenHash(ctx, tokenHash)
	return oneInvite(authdb.InviteByIDRow(row), err)
}

func (r *postgresRepository) SetInviteToken(ctx context.Context, id string, tokenHash []byte) error {
	u, err := uuidOf(id)
	if err != nil {
		return ErrInvalidInvite
	}
	n, err := r.q(ctx).SetInviteToken(ctx, authdb.SetInviteTokenParams{ID: u, TokenHash: tokenHash})
	if err == nil && n == 0 {
		return ErrInvalidInvite
	}
	return err
}

func (r *postgresRepository) InviteOrganization(ctx context.Context, tokenHash []byte) (string, error) {
	org, err := r.q(ctx).InviteOrganization(ctx, tokenHash)
	if err != nil {
		return "", err
	}
	if !org.Valid {
		return "", ErrInvalidInvite
	}
	return org.String(), nil
}

func (r *postgresRepository) AcceptInvite(ctx context.Context, id string, at time.Time) error {
	u, err := uuidOf(id)
	if err != nil {
		return ErrInvalidInvite
	}
	n, err := r.q(ctx).AcceptInvite(ctx, authdb.AcceptInviteParams{ID: u, AcceptedAt: timestamptz(at)})
	if err == nil && n == 0 {
		return ErrInvalidInvite
	}
	return err
}

func oneInvite(row authdb.InviteByIDRow, err error) (Invite, error) {
	if errors.Is(err, pgx.ErrNoRows) {
		return Invite{}, ErrInvalidInvite
	}
	if err != nil {
		return Invite{}, err
	}
	return toInvite(row), nil
}

// toInvite maps a row; every invite query selects the same columns.
func toInvite(row authdb.InviteByIDRow) Invite {
	inv := Invite{
		ID: row.ID.String(), CreatedAt: row.CreatedAt.Time, ExpiresAt: row.ExpiresAt.Time, OrganizationID: row.OrganizationID.String(),
		Role: Role(row.Role), Email: row.Email, GivenName: row.GivenName, FamilyName: row.FamilyName,
	}
	if row.AcceptedAt.Valid {
		inv.AcceptedAt = row.AcceptedAt.Time
	}
	if row.InvitedBy.Valid {
		inv.InvitedBy = row.InvitedBy.String()
	}
	return inv
}

func (r *postgresRepository) DeleteSessionsEndedBefore(ctx context.Context, before time.Time) (int64, error) {
	return r.q(ctx).DeleteSessionsEndedBefore(ctx, pgtype.Timestamptz{Time: before, Valid: true})
}

func (r *postgresRepository) DeleteInvitesEndedBefore(ctx context.Context, before time.Time) (int64, error) {
	return r.q(ctx).DeleteInvitesEndedBefore(ctx, pgtype.Timestamptz{Time: before, Valid: true})
}
