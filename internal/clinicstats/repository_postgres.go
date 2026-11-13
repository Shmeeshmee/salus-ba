package clinicstats

import (
	"cmp"
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Shmeeshmee/salus/internal/clinicstats/clinicstatsdb"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

type postgresRepository struct {
	db postgres.Querier
}

// NewPostgresRepository returns the Repository on db (salus_app), using the transaction in the context when there
// is one: row-level security shows only the clinic SetClinic set.
func NewPostgresRepository(db postgres.Querier) Repository {
	return &postgresRepository{db: db}
}

// NewPostgresBuildRepository returns the BuildRepository on db (salus_stats), using the transaction in the
// context when there is one.
func NewPostgresBuildRepository(db postgres.Querier) BuildRepository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) q(ctx context.Context) *clinicstatsdb.Queries {
	return clinicstatsdb.New(postgres.Conn(ctx, r.db))
}

func (r *postgresRepository) Lock(ctx context.Context, orgID string) error {
	org, err := uuid(orgID)
	if err != nil {
		return err
	}
	return r.q(ctx).LockClinic(ctx, org.String())
}

func (r *postgresRepository) Replace(ctx context.Context, orgID string, first, last time.Time, appts []AppointmentCount, diags []DiagnosisCount) error {
	org, err := uuid(orgID)
	if err != nil {
		return err
	}
	q := r.q(ctx)
	days := clinicstatsdb.DeleteAppointmentDaysParams{OrganizationID: org, FirstDay: pgDate(first), LastDay: pgDate(last)}
	if err := q.DeleteAppointmentDays(ctx, days); err != nil {
		return err
	}
	if err := q.DeleteDiagnosisDays(ctx, clinicstatsdb.DeleteDiagnosisDaysParams(days)); err != nil {
		return err
	}
	a := clinicstatsdb.InsertAppointmentCountsParams{OrganizationID: org}
	for _, c := range appts {
		role, err := uuid(c.PractitionerRoleID)
		if err != nil {
			return err
		}
		a.Roles, a.Days, a.Hours = append(a.Roles, role), append(a.Days, pgDate(c.Day)), append(a.Hours, int16(c.Hour))
		a.Specialties, a.Statuses, a.Counts = append(a.Specialties, c.SpecialtyCode), append(a.Statuses, c.Status), append(a.Counts, c.Count)
		a.BookedVias = append(a.BookedVias, cmp.Or(c.BookedVia, "direct")) // a count without a way of booking is direct
	}
	if err := q.InsertAppointmentCounts(ctx, a); err != nil {
		return err
	}
	d := clinicstatsdb.InsertDiagnosisCountsParams{OrganizationID: org}
	for _, c := range diags {
		role, err := uuid(c.PractitionerRoleID)
		if err != nil {
			return err
		}
		d.Roles, d.Days, d.Hours = append(d.Roles, role), append(d.Days, pgDate(c.Day)), append(d.Hours, int16(c.Hour))
		d.Specialties, d.GroupCodes, d.SubgroupCodes = append(d.Specialties, c.SpecialtyCode), append(d.GroupCodes, c.Group), append(d.SubgroupCodes, c.Subgroup)
		d.Codes, d.Counts = append(d.Codes, c.Code), append(d.Counts, c.Count)
	}
	return q.InsertDiagnosisCounts(ctx, d)
}

func (r *postgresRepository) SetClinic(ctx context.Context, orgID string) error {
	return postgres.SetLocal(ctx, "app.organization_id", orgID)
}

// filter is the shared part of the reads' parameters.
type filter struct {
	from, to pgtype.Date
	org      pgtype.UUID
	role     pgtype.UUID
}

func newFilter(orgID string, from, to time.Time, roleID string) (filter, error) {
	org, err := uuid(orgID)
	if err != nil {
		return filter{}, err
	}
	f := filter{org: org, from: pgDate(from), to: pgDate(to)}
	if roleID != "" {
		// A role that is no UUID is no role of the clinic: it matches nothing, like an unknown one.
		if f.role, err = uuid(roleID); err != nil {
			f.role = pgtype.UUID{Valid: true}
		}
	}
	return f, nil
}

func (r *postgresRepository) Series(ctx context.Context, orgID string, q AppointmentQuery) ([]Period, error) {
	f, err := newFilter(orgID, q.From, q.To, q.PractitionerRoleID)
	if err != nil {
		return nil, err
	}
	rows, err := r.q(ctx).AppointmentSeries(ctx, clinicstatsdb.AppointmentSeriesParams{
		OrganizationID: f.org, FromDay: f.from, ToDay: f.to, PractitionerRoleID: f.role, Specialty: q.Specialty,
	})
	if err != nil {
		return nil, err
	}
	out := make([]Period, len(rows))
	for i, row := range rows {
		out[i] = Period{Period: row.Period, Booked: row.Booked, Fulfilled: row.Fulfilled, NoShow: row.Noshow, Cancelled: row.Cancelled, Refilled: row.Refilled}
	}
	return out, nil
}

func (r *postgresRepository) Heatmap(ctx context.Context, orgID string, q AppointmentQuery) ([]HeatCell, error) {
	f, err := newFilter(orgID, q.From, q.To, q.PractitionerRoleID)
	if err != nil {
		return nil, err
	}
	rows, err := r.q(ctx).AppointmentHeatmap(ctx, clinicstatsdb.AppointmentHeatmapParams{
		OrganizationID: f.org, FromDay: f.from, ToDay: f.to, PractitionerRoleID: f.role, Specialty: q.Specialty,
	})
	if err != nil {
		return nil, err
	}
	out := make([]HeatCell, len(rows))
	for i, row := range rows {
		out[i] = HeatCell{Weekday: row.Weekday, Hour: row.Hour, Count: row.Count}
	}
	return out, nil
}

func (r *postgresRepository) Practitioners(ctx context.Context, orgID string, q AppointmentQuery) ([]PractitionerTotal, error) {
	f, err := newFilter(orgID, q.From, q.To, q.PractitionerRoleID)
	if err != nil {
		return nil, err
	}
	rows, err := r.q(ctx).AppointmentPractitioners(ctx, clinicstatsdb.AppointmentPractitionersParams{
		OrganizationID: f.org, FromDay: f.from, ToDay: f.to, PractitionerRoleID: f.role, Specialty: q.Specialty,
	})
	if err != nil {
		return nil, err
	}
	out := make([]PractitionerTotal, len(rows))
	for i, row := range rows {
		out[i] = PractitionerTotal{PractitionerRoleID: row.PractitionerRoleID.String(), Fulfilled: row.Fulfilled, NoShow: row.Noshow, Booked: row.Booked}
	}
	return out, nil
}

func (r *postgresRepository) Specialties(ctx context.Context, orgID string, q AppointmentQuery) ([]SpecialtyTotal, error) {
	f, err := newFilter(orgID, q.From, q.To, q.PractitionerRoleID)
	if err != nil {
		return nil, err
	}
	rows, err := r.q(ctx).AppointmentSpecialties(ctx, clinicstatsdb.AppointmentSpecialtiesParams{
		OrganizationID: f.org, FromDay: f.from, ToDay: f.to, PractitionerRoleID: f.role, Specialty: q.Specialty,
	})
	if err != nil {
		return nil, err
	}
	out := make([]SpecialtyTotal, len(rows))
	for i, row := range rows {
		out[i] = SpecialtyTotal{Specialty: row.Specialty, Booked: row.Booked, Fulfilled: row.Fulfilled, NoShow: row.Noshow, Cancelled: row.Cancelled}
	}
	return out, nil
}

func (r *postgresRepository) Diagnoses(ctx context.Context, orgID string, q DiagnosisQuery) ([]Count, error) {
	f, err := newFilter(orgID, q.From, q.To, q.PractitionerRoleID)
	if err != nil {
		return nil, err
	}
	rows, err := r.q(ctx).DiagnosisCounts(ctx, clinicstatsdb.DiagnosisCountsParams{
		Breakdown: string(q.By), OrganizationID: f.org, FromDay: f.from, ToDay: f.to, PractitionerRoleID: f.role,
		Specialty: q.Specialty, GroupCode: q.Group, SubgroupCode: q.Subgroup,
	})
	if err != nil {
		return nil, err
	}
	out := make([]Count, len(rows))
	for i, row := range rows {
		out[i] = Count{Key: row.Key, Count: row.Count}
	}
	return out, nil
}

func uuid(s string) (pgtype.UUID, error) {
	var u pgtype.UUID
	err := u.Scan(s)
	return u, err
}

func pgDate(t time.Time) pgtype.Date { return pgtype.Date{Time: t, Valid: true} }
