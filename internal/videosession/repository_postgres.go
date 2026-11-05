package videosession

import (
	"context"
	"errors"
	"time"

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

func (r *postgresRepository) Open(ctx context.Context, s Session) (Session, error) {
	var appt, org pgtype.UUID
	if appt.Scan(s.AppointmentID) != nil || org.Scan(s.OrganizationID) != nil {
		return Session{}, errors.New("videosession: an id is not a UUID")
	}
	err := r.q(ctx).CreateSession(ctx, videosessiondb.CreateSessionParams{
		AppointmentID: appt, OrganizationID: org, Room: s.Room, PatientIdentity: s.PatientIdentity,
		PractitionerIdentity: s.PractitionerIdentity, ClosesAt: ts(s.ClosesAt), CreatedAt: ts(s.CreatedAt),
	})
	if err != nil {
		return Session{}, err
	}
	row, err := r.q(ctx).SessionByAppointment(ctx, appt)
	if err != nil {
		return Session{}, err
	}
	return toSession(row), nil
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
		EndReason: EndReason(row.EndReason.String),
	}
}
