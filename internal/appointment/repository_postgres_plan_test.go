package appointment_test

import (
	"context"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// The staff's reads of a range of the clinic's visits (the desk's day, the calendar's week, the practice report) read
// that range's rows only, under row-level security: a busy clinic's year is never scanned for one day
// (docs/clinic-cms/load/REPORT.md, RD2). The clinic here has 4 visits a day for 300 days.
func TestStaffRangeReads_ReadOnlyTheRangesRows(t *testing.T) {
	t.Parallel()
	url := pgtest.NewDatabaseURL(t)
	owner := pgtest.TracedPool(t, url, nil)
	s := seedDB(t, owner)
	if _, err := owner.Exec(t.Context(), `INSERT INTO appointment (patient_id, practitioner_role_id, organization_id, slot_id, period)
		SELECT $1, $2, $3, gen_random_uuid(), tstzrange($4::timestamptz + d * interval '1 day' + k * interval '1 hour',
			$4::timestamptz + d * interval '1 day' + k * interval '1 hour' + interval '30 minutes')
		FROM generate_series(-150, 149) d, generate_series(0, 3) k`, s.patientA, s.roleA1, clinicA, start); err != nil {
		t.Fatal(err)
	}
	if _, err := owner.Exec(t.Context(), "ANALYZE appointment"); err != nil {
		t.Fatal(err)
	}
	traced := &pgtest.LastQuery{Match: "FROM appointment"}
	app := pgtest.TracedPool(t, pgtest.AppURL(t, url), traced)
	day := start.Add(-time.Hour)
	for name, read := range map[string]func(ctx context.Context, r appointment.Repository) (int, error){
		"the desk's day": func(ctx context.Context, r appointment.Repository) (int, error) {
			all, err := r.Desk(ctx, clinicA, day, day.Add(24*time.Hour))
			return len(all), err
		},
		"the calendar's week": func(ctx context.Context, r appointment.Repository) (int, error) {
			all, err := r.Search(ctx, appointment.Filter{OrganizationID: clinicA, PractitionerRoleID: s.roleA1, From: day, To: day.Add(7 * 24 * time.Hour)})
			return len(all), err
		},
	} {
		err := inScope(t, app, asClinic, func(ctx context.Context, r appointment.Repository) error {
			n, err := read(ctx, r)
			if err != nil {
				return err
			}
			passed := pgtest.Explain(t, ctx, app, traced).Dropped
			if n == 0 || passed > 40 {
				t.Errorf("%s: %d rows, %v read and dropped; want the range's rows and few others", name, n, passed)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
}
