package organization

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Shmeeshmee/salus/internal/organization/organizationdb"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

// NewPostgresAppointmentsRepository returns the AppointmentsRepository on db, as salus_app: row-level security shows
// only the clinic set by SetOrganization.
func NewPostgresAppointmentsRepository(db postgres.Querier) AppointmentsRepository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) AppointmentSettings(ctx context.Context, org string) (AppointmentSettings, bool, error) {
	var u pgtype.UUID
	if u.Scan(org) != nil {
		return AppointmentSettings{}, false, nil
	}
	row, err := r.q(ctx).AppointmentSettings(ctx, u)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return AppointmentSettings{}, false, nil
	case err != nil:
		return AppointmentSettings{}, false, err
	}
	offsets := make([]int, len(row.ReminderOffsetsMinutes))
	for i, m := range row.ReminderOffsetsMinutes {
		offsets[i] = int(m)
	}
	return AppointmentSettings{
		ReminderOffsetsMinutes: offsets, SMSReminders: row.SmsReminders, RescheduleNoticeHours: int(row.RescheduleNoticeHours),
		RescheduleMax: int(row.RescheduleMax), WaitlistEnabled: row.WaitlistEnabled, WaitlistHoldMinutes: int(row.WaitlistHoldMinutes),
		NoShowFlagThreshold: int(row.NoshowFlagThreshold), NoShowFlagWindowDays: int(row.NoshowFlagWindowDays),
	}, true, nil
}

func (r *postgresRepository) PutAppointmentSettings(ctx context.Context, org string, s AppointmentSettings, at time.Time) error {
	var u pgtype.UUID
	if u.Scan(org) != nil {
		return ErrNotFound
	}
	offsets := make([]int32, len(s.ReminderOffsetsMinutes))
	for i, m := range s.ReminderOffsetsMinutes {
		offsets[i] = int32(m)
	}
	return r.q(ctx).PutAppointmentSettings(ctx, organizationdb.PutAppointmentSettingsParams{
		OrganizationID: u, ReminderOffsetsMinutes: offsets, SmsReminders: s.SMSReminders,
		RescheduleNoticeHours: int32(s.RescheduleNoticeHours), RescheduleMax: int32(s.RescheduleMax),
		WaitlistEnabled: s.WaitlistEnabled, WaitlistHoldMinutes: int32(s.WaitlistHoldMinutes),
		NoshowFlagThreshold: int32(s.NoShowFlagThreshold), NoshowFlagWindowDays: int32(s.NoShowFlagWindowDays),
		UpdatedAt: pgtype.Timestamptz{Time: at, Valid: true},
	})
}
