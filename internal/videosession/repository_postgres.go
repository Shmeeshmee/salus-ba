package videosession

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Shmeeshmee/salus/internal/videosession/videosessiondb"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

type postgresRepository struct {
	db postgres.Querier
}

// NewPostgresRepository returns the Repository on db, using the transaction in the context when there is one.
// On the salus_app pool row-level security shows only the scope set; on the retention role's, every clinic.
func NewPostgresRepository(db postgres.Querier) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) q(ctx context.Context) *videosessiondb.Queries {
	return videosessiondb.New(postgres.Conn(ctx, r.db))
}

func (r *postgresRepository) SetPatient(ctx context.Context, id string) error {
	return postgres.SetLocal(ctx, "app.patient_id", id)
}

func (r *postgresRepository) SetClinic(ctx context.Context, id string) error {
	return postgres.SetLocal(ctx, "app.organization_id", id)
}

func (r *postgresRepository) Open(ctx context.Context, s Session) (Session, bool, error) {
	var appt, org pgtype.UUID
	if appt.Scan(s.AppointmentID) != nil || org.Scan(s.OrganizationID) != nil {
		return Session{}, false, errors.New("videosession: an id is not a UUID")
	}
	n, err := r.q(ctx).CreateSession(ctx, videosessiondb.CreateSessionParams{
		AppointmentID: appt, OrganizationID: org, Room: s.Room, PatientIdentity: s.PatientIdentity,
		PractitionerIdentity: s.PractitionerIdentity, ClosesAt: ts(s.ClosesAt), CreatedAt: ts(s.CreatedAt),
	})
	if err != nil {
		return Session{}, false, err
	}
	row, err := r.q(ctx).SessionByAppointment(ctx, appt)
	if err != nil {
		return Session{}, false, err
	}
	return toSession(row), n == 1, nil
}

func (r *postgresRepository) ByID(ctx context.Context, id string) (Session, error) {
	var u pgtype.UUID
	if u.Scan(id) != nil {
		return Session{}, ErrNotFound
	}
	row, err := r.q(ctx).SessionByID(ctx, u)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return Session{}, ErrNotFound
	case err != nil:
		return Session{}, err
	}
	return toSession(row), nil
}

func (r *postgresRepository) SetRoom(ctx context.Context, room string) error {
	return postgres.SetLocal(ctx, "app.video_room", room)
}

func (r *postgresRepository) SetAppointment(ctx context.Context, id string) error {
	return postgres.SetLocal(ctx, "app.appointment_id", id)
}

func (r *postgresRepository) Removed(ctx context.Context, id string, reason Reason, at time.Time) error {
	var u pgtype.UUID
	if u.Scan(id) != nil {
		return nil
	}
	return r.q(ctx).RecordRemoval(ctx, videosessiondb.RecordRemovalParams{
		ID: u, At: ts(at), Reason: pgtype.Text{String: string(reason), Valid: true},
	})
}

func (r *postgresRepository) ByRoom(ctx context.Context, room string) (Session, error) {
	row, err := r.q(ctx).SessionByRoom(ctx, room)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return Session{}, ErrNotFound
	case err != nil:
		return Session{}, err
	}
	return toSession(row), nil
}

func (r *postgresRepository) Joined(ctx context.Context, id string, role Role, at, now time.Time) (Session, bool, error) {
	var u pgtype.UUID
	if u.Scan(id) != nil {
		return Session{}, false, nil
	}
	var row videosessiondb.VideoSession
	var err error
	if role == RolePractitioner {
		row, err = r.q(ctx).PractitionerJoined(ctx, videosessiondb.PractitionerJoinedParams{ID: u, At: ts(at), UpdatedAt: ts(now)})
	} else {
		row, err = r.q(ctx).PatientJoined(ctx, videosessiondb.PatientJoinedParams{ID: u, At: ts(at), UpdatedAt: ts(now)})
	}
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return Session{}, false, nil
	case err != nil:
		return Session{}, false, err
	}
	return toSession(row), true, nil
}

func (r *postgresRepository) End(ctx context.Context, id string, reason EndReason, at time.Time) (Session, bool, error) {
	var u pgtype.UUID
	if u.Scan(id) != nil {
		return Session{}, false, nil
	}
	row, err := r.q(ctx).EndSession(ctx, videosessiondb.EndSessionParams{
		ID: u, At: ts(at), Reason: pgtype.Text{String: string(reason), Valid: true},
	})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return Session{}, false, nil
	case err != nil:
		return Session{}, false, err
	}
	return toSession(row), true, nil
}

func (r *postgresRepository) DeleteEndedBefore(ctx context.Context, before time.Time) (int64, error) {
	return r.q(ctx).DeleteSessionsEndedBefore(ctx, ts(before))
}

func ts(t time.Time) pgtype.Timestamptz { return pgtype.Timestamptz{Time: t, Valid: true} }

func toSession(row videosessiondb.VideoSession) Session {
	return Session{
		CreatedAt: row.CreatedAt.Time, ClosesAt: row.ClosesAt.Time,
		PatientJoinedAt: row.PatientJoinedAt.Time, PractitionerJoinedAt: row.PractitionerJoinedAt.Time, EndedAt: row.EndedAt.Time,
		ID: row.ID.String(), AppointmentID: row.AppointmentID.String(), OrganizationID: row.OrganizationID.String(),
		Room: row.Room, PatientIdentity: row.PatientIdentity, PractitionerIdentity: row.PractitionerIdentity,
		EndReason: EndReason(row.EndReason.String), RemovalReason: Reason(row.RemovalReason.String),
	}
}

func (r *postgresRepository) ByAppointment(ctx context.Context, appointmentID string) (Session, error) {
	var u pgtype.UUID
	if u.Scan(appointmentID) != nil {
		return Session{}, ErrNotFound
	}
	row, err := r.q(ctx).SessionByAppointment(ctx, u)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return Session{}, ErrNotFound
	case err != nil:
		return Session{}, err
	}
	return toSession(row), nil
}

func (r *postgresRepository) ClosingAfter(ctx context.Context, org string, after time.Time) ([]Session, error) {
	var u pgtype.UUID
	if u.Scan(org) != nil {
		return nil, nil
	}
	rows, err := r.q(ctx).SessionsClosingAfter(ctx, videosessiondb.SessionsClosingAfterParams{OrganizationID: u, After: ts(after)})
	if err != nil {
		return nil, err
	}
	out := make([]Session, 0, len(rows))
	for _, row := range rows {
		out = append(out, toSession(row))
	}
	return out, nil
}
