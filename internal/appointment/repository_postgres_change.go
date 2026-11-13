package appointment

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Shmeeshmee/salus/internal/appointment/appointmentdb"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

func (r *postgresRepository) AddChange(ctx context.Context, c Change) (string, error) {
	var id pgtype.UUID
	if id.Scan(c.AppointmentID) != nil {
		return "", ErrNotFound
	}
	p := appointmentdb.InsertAppointmentChangeParams{
		At: ts(c.At), AppointmentID: id, Kind: string(c.Kind), Via: string(c.Via),
		Agent: text(c.Agent), FromStatus: text(string(c.FromStatus)), ToStatus: text(string(c.ToStatus)),
		FromStart: optionalTS(c.FromStart), FromEnd: optionalTS(c.FromEnd), ToStart: optionalTS(c.ToStart), ToEnd: optionalTS(c.ToEnd),
	}
	for _, u := range []struct {
		dst *pgtype.UUID
		v   string
	}{{&p.ByAccountID, c.ByAccountID}, {&p.FromSlotID, c.FromSlotID}, {&p.ToSlotID, c.ToSlotID}} {
		if u.v != "" && u.dst.Scan(u.v) != nil {
			return "", errors.New("appointment: an id of the change is not a UUID")
		}
	}
	out, err := r.q(ctx).InsertAppointmentChange(ctx, p)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return "", ErrNotFound
	case err != nil:
		return "", err
	}
	return out.String(), nil
}

func (r *postgresRepository) Changes(ctx context.Context, id string) ([]Change, error) {
	var u pgtype.UUID
	if u.Scan(id) != nil {
		return nil, nil
	}
	rows, err := r.q(ctx).ListAppointmentChanges(ctx, u)
	if err != nil {
		return nil, err
	}
	out := make([]Change, 0, len(rows))
	for _, row := range rows {
		c := Change{
			At: row.At.Time, ID: row.ID.String(), AppointmentID: row.AppointmentID.String(), Kind: ChangeKind(row.Kind), Via: Via(row.Via),
			Agent: row.Agent.String, FromStatus: Status(row.FromStatus.String), ToStatus: Status(row.ToStatus.String),
			FromStart: row.FromStart.Time, FromEnd: row.FromEnd.Time, ToStart: row.ToStart.Time, ToEnd: row.ToEnd.Time,
		}
		for _, u := range []struct {
			dst *string
			v   pgtype.UUID
		}{{&c.ByAccountID, row.ByAccountID}, {&c.FromSlotID, row.FromSlotID}, {&c.ToSlotID, row.ToSlotID}} {
			if u.v.Valid {
				*u.dst = u.v.String()
			}
		}
		out = append(out, c)
	}
	return out, nil
}

// NewPostgresTokenRepository returns the TokenRepository on db, using the transaction in the context when there is
// one.
func NewPostgresTokenRepository(db postgres.Querier) TokenRepository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) InsertToken(ctx context.Context, t Token) error {
	p := appointmentdb.InsertActionTokenParams{
		CreatedAt: ts(t.CreatedAt), ExpiresAt: ts(t.ExpiresAt), TokenHash: t.Hash[:], Channel: string(t.Channel),
		Language: appointmentdb.Language(t.Language),
	}
	for _, u := range []struct {
		dst *pgtype.UUID
		v   string
	}{{&p.AppointmentID, t.AppointmentID}, {&p.OrganizationID, t.OrganizationID}, {&p.PatientID, t.PatientID}, {&p.AccountID, t.AccountID}, {&p.ReminderID, t.ReminderID}} {
		if u.v != "" && u.dst.Scan(u.v) != nil {
			return errors.New("appointment: an id of the link is not a UUID")
		}
	}
	return r.q(ctx).InsertActionToken(ctx, p)
}

func (r *postgresRepository) DeleteTokensExpiredBefore(ctx context.Context, before time.Time) (int64, error) {
	return r.q(ctx).DeleteActionTokensExpiredBefore(ctx, ts(before))
}

func text(s string) pgtype.Text { return pgtype.Text{String: s, Valid: s != ""} }

func optionalTS(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: !t.IsZero()}
}
