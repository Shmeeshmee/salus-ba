package organization

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Shmeeshmee/salus/internal/organization/organizationdb"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

type postgresRepository struct {
	db postgres.Querier
}

// NewPostgresRepository returns the Repository on db, using the transaction in the context when there is one.
// Its queries run as salus_app, so row-level security shows only the clinic set by SetOrganization.
func NewPostgresRepository(db postgres.Querier) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) q(ctx context.Context) *organizationdb.Queries {
	return organizationdb.New(postgres.Conn(ctx, r.db))
}

func (r *postgresRepository) SetOrganization(ctx context.Context, id string) error {
	return postgres.SetLocal(ctx, "app.organization_id", id)
}

func (r *postgresRepository) SetDirectory(ctx context.Context) error {
	return postgres.SetLocal(ctx, "app.directory", "public")
}

func (r *postgresRepository) Listed(ctx context.Context) ([]Organization, error) {
	rows, err := r.q(ctx).ListedOrganizations(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Organization, 0, len(rows))
	for _, row := range rows {
		out = append(out, toOrganization(organizationdb.OrganizationByIDRow(row)))
	}
	return out, nil
}

func (r *postgresRepository) ByID(ctx context.Context, id string) (Organization, error) {
	var u pgtype.UUID
	if u.Scan(id) != nil {
		return Organization{}, ErrNotFound
	}
	row, err := r.q(ctx).OrganizationByID(ctx, u)
	if errors.Is(err, pgx.ErrNoRows) {
		return Organization{}, ErrNotFound
	}
	if err != nil {
		return Organization{}, err
	}
	return toOrganization(row), nil
}

func (r *postgresRepository) Update(ctx context.Context, o Organization, at time.Time) (Organization, error) {
	var u pgtype.UUID
	if u.Scan(o.ID) != nil {
		return Organization{}, ErrConflict
	}
	row, err := r.q(ctx).UpdateOrganization(ctx, organizationdb.UpdateOrganizationParams{
		ID: u, Name: o.Name, Listed: o.Listed, TimeZone: o.TimeZone, DefaultLanguage: organizationdb.Language(o.DefaultLanguage), VersionID: o.VersionID,
		PatientTranscriptCopy: o.PatientTranscriptCopy, UpdatedAt: pgtype.Timestamptz{Time: at, Valid: true},
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Organization{}, ErrConflict
	}
	if err != nil {
		return Organization{}, err
	}
	return toOrganization(organizationdb.OrganizationByIDRow(row)), nil
}

func (r *postgresRepository) Create(ctx context.Context, name string) (Organization, error) {
	row, err := r.q(ctx).CreateOrganization(ctx, name)
	if err != nil {
		return Organization{}, err
	}
	return toOrganization(organizationdb.OrganizationByIDRow(row)), nil
}

func toOrganization(row organizationdb.OrganizationByIDRow) Organization {
	return Organization{
		ID: row.ID.String(), Name: row.Name, TimeZone: row.TimeZone, VersionID: row.VersionID, Active: row.Active,
		Listed: row.Listed, DefaultLanguage: string(row.DefaultLanguage), LastUpdated: row.UpdatedAt.Time,
		PatientTranscriptCopy: row.PatientTranscriptCopy,
	}
}

// NewPostgresChatRepository returns the ChatRepository on db, as salus_app: row-level security shows only the
// clinic set by SetOrganization.
func NewPostgresChatRepository(db postgres.Querier) ChatRepository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) ChatSettings(ctx context.Context, org string) (ChatSettings, bool, error) {
	var u pgtype.UUID
	if u.Scan(org) != nil {
		return ChatSettings{}, false, nil
	}
	row, err := r.q(ctx).ChatSettings(ctx, u)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return ChatSettings{}, false, nil
	case err != nil:
		return ChatSettings{}, false, err
	}
	s := ChatSettings{ResponseTimeHours: int(row.ResponseTimeHours), EmailAfterMinutes: int(row.EmailAfterMinutes)}
	if err := json.Unmarshal(row.Hours, &s.Hours); err != nil {
		return ChatSettings{}, false, fmt.Errorf("the stored chat hours: %w", err)
	}
	return s, true, nil
}

func (r *postgresRepository) PutChatSettings(ctx context.Context, org string, s ChatSettings, at time.Time) error {
	var u pgtype.UUID
	if u.Scan(org) != nil {
		return ErrNotFound
	}
	if s.Hours == nil {
		s.Hours = []ChatHour{}
	}
	hours, err := json.Marshal(s.Hours)
	if err != nil {
		return err
	}
	return r.q(ctx).PutChatSettings(ctx, organizationdb.PutChatSettingsParams{
		OrganizationID: u, Hours: hours, ResponseTimeHours: int32(s.ResponseTimeHours), EmailAfterMinutes: int32(s.EmailAfterMinutes),
		UpdatedAt: pgtype.Timestamptz{Time: at, Valid: true},
	})
}

type postgresStatsRepository struct {
	db postgres.Querier
}

// NewPostgresStatsRepository returns the StatsRepository on db, the salus_stats pool.
func NewPostgresStatsRepository(db postgres.Querier) StatsRepository {
	return &postgresStatsRepository{db: db}
}

func (r *postgresStatsRepository) Locations(ctx context.Context) ([]Location, error) {
	rows, err := organizationdb.New(postgres.Conn(ctx, r.db)).ListLocationsForStats(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Location, len(rows))
	for i, row := range rows {
		out[i] = Location{ID: row.ID.String(), TimeZone: row.TimeZone}
	}
	return out, nil
}

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
