package worker

import (
	"slices"
	"testing"

	"github.com/Shmeeshmee/salus/internal/clinicstats"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

// historyOf gives, as the owner, every final appointment of the clinic its history as the services write it
// (SAL-36, docs/plans/appointments.md): a status_set by the system (the consultation's end) for each fulfilled or
// no-show, a status_set by staff for each cancellation; then one in seven of the fulfilled and no-show ones is
// corrected by staff to the other status (status_corrected), and one in six of all is marked booked through the
// waitlist.
func historyOf(t *testing.T, p statsPools, org string) {
	t.Helper()
	ctx := t.Context()
	var staff string
	if err := p.owner.QueryRow(ctx, `INSERT INTO account (email, password_hash, role, organization_id)
		VALUES (gen_random_uuid()::text || '@example.test', 'h', 'admin', $1) RETURNING id`, org).Scan(&staff); err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{
		`INSERT INTO appointment_change (at, appointment_id, organization_id, patient_id, by_account_id, kind, via, agent, from_status, to_status)
			SELECT upper(period), id, organization_id, patient_id, CASE WHEN status = 'cancelled' THEN $2::uuid END, 'status_set',
				CASE WHEN status = 'cancelled' THEN 'staff' ELSE 'system' END,
				CASE WHEN status = 'cancelled' THEN NULL ELSE 'consultation_window_close' END, 'booked', status
			FROM appointment WHERE organization_id = $1 AND status <> 'booked'`,
		`WITH c AS (UPDATE appointment SET status = CASE status WHEN 'noshow' THEN 'fulfilled' ELSE 'noshow' END
				WHERE organization_id = $1 AND status IN ('fulfilled', 'noshow') AND abs(hashtext(id::text)) % 7 = 0
				RETURNING id, organization_id, patient_id, period, status)
			INSERT INTO appointment_change (at, appointment_id, organization_id, patient_id, by_account_id, kind, via, from_status, to_status)
			SELECT upper(period) + interval '1 day', id, organization_id, patient_id, $2, 'status_corrected', 'staff',
				CASE status WHEN 'noshow' THEN 'fulfilled' ELSE 'noshow' END, status FROM c`,
		`UPDATE appointment SET booked_via = 'waitlist' WHERE organization_id = $1 AND abs(hashtext(id::text || $2)) % 6 = 0`,
	} {
		if _, err := p.owner.Exec(ctx, q, org, staff); err != nil {
			t.Fatal(err)
		}
	}
}

// The no-show reports equal the raw appointments and their history: a synthetic clinic of three roles (one a general
// practitioner) over October and November 2026 (every status, corrections both ways, waitlist bookings, and the
// autumn clock change of 25 October), refreshed day by day and read back as its admin. The no-shows of each month,
// practitioner and specialty are the appointments whose latest status in their history is noshow; the refilled slots
// the waitlist bookings of each month.
func TestNoShowReports_EqualTheAppointmentsAndTheirHistory(t *testing.T) {
	t.Parallel()
	p := newStatsPools(t)
	org, roles := seedStatsRoles(t, p.owner, 3)
	if _, err := p.owner.Exec(t.Context(), `UPDATE practitioner_role SET specialty_code = '394814009' WHERE id = $1`, roles[0]); err != nil {
		t.Fatal(err)
	}
	first, last := date(2026, 10, 1), date(2026, 11, 30)
	synthetic(t, p.owner, org, roles, first, last, 4)
	historyOf(t, p, org)
	b := newClinicStats(p.stats, clock.Fixed{T: localAt(2026, 12, 1, 9, 0)})
	for d := first; !d.After(last); d = d.AddDate(0, 0, 1) {
		if err := b.Refresh(t.Context(), org, localAt(d.Year(), d.Month(), d.Day(), 12, 0)); err != nil {
			t.Fatal(err)
		}
	}
	svc := clinicstats.NewService(clinicstats.Ports{Repo: clinicstats.NewPostgresRepository(p.app), Audit: recordNothing{}, Tx: postgres.NewTransactor(p.app)})
	got, err := svc.Appointments(t.Context(), clinicstats.Caller{AccountID: "acc", OrganizationID: org, Admin: true},
		clinicstats.AppointmentQuery{From: first, To: last})
	if err != nil {
		t.Fatal(err)
	}
	// Each appointment's status by its history: the latest change, booked without one.
	const final = `(SELECT a.*, coalesce((SELECT c.to_status FROM appointment_change c WHERE c.appointment_id = a.id
			ORDER BY c.at DESC LIMIT 1), 'booked') AS history_status FROM appointment a WHERE a.organization_id = $1) x`
	var series []clinicstats.Period
	rawQuery(t, p.owner, `SELECT to_char(`+localStart+`::date, 'YYYY-MM'), count(*), count(*) FILTER (WHERE history_status = 'fulfilled'),
			count(*) FILTER (WHERE history_status = 'noshow'), count(*) FILTER (WHERE history_status = 'cancelled'),
			count(*) FILTER (WHERE booked_via = 'waitlist')
		FROM `+final+` WHERE `+localStart+`::date BETWEEN $2 AND $3 GROUP BY 1 ORDER BY 1`,
		func(r interface{ Scan(...any) error }) error {
			var x clinicstats.Period
			err := r.Scan(&x.Period, &x.Booked, &x.Fulfilled, &x.NoShow, &x.Cancelled, &x.Refilled)
			series = append(series, x)
			return err
		}, org, first, last)
	if len(series) != 2 || series[0].Refilled == 0 || series[0].NoShow == 0 || !slices.Equal(got.Series, series) {
		t.Errorf("series\n got %v\nwant %v", got.Series, series)
	}
	var totals []clinicstats.PractitionerTotal
	rawQuery(t, p.owner, `SELECT practitioner_role_id::text, count(*) FILTER (WHERE history_status = 'fulfilled'),
			count(*) FILTER (WHERE history_status = 'noshow'), count(*)
		FROM `+final+` WHERE `+localStart+`::date BETWEEN $2 AND $3 GROUP BY 1 ORDER BY 2 DESC, 1`,
		func(r interface{ Scan(...any) error }) error {
			var x clinicstats.PractitionerTotal
			err := r.Scan(&x.PractitionerRoleID, &x.Fulfilled, &x.NoShow, &x.Booked)
			totals = append(totals, x)
			return err
		}, org, first, last)
	if len(totals) != 3 || !slices.Equal(got.Practitioners, totals) {
		t.Errorf("practitioners\n got %v\nwant %v", got.Practitioners, totals)
	}
	var specialties []clinicstats.SpecialtyTotal
	rawQuery(t, p.owner, `SELECT coalesce(r.specialty_code, ''), count(*), count(*) FILTER (WHERE history_status = 'fulfilled'),
			count(*) FILTER (WHERE history_status = 'noshow'), count(*) FILTER (WHERE history_status = 'cancelled')
		FROM `+final+` JOIN practitioner_role r ON r.id = x.practitioner_role_id
		WHERE `+localStart+`::date BETWEEN $2 AND $3 GROUP BY 1 ORDER BY 2 DESC, 1`,
		func(r interface{ Scan(...any) error }) error {
			var x clinicstats.SpecialtyTotal
			err := r.Scan(&x.Specialty, &x.Booked, &x.Fulfilled, &x.NoShow, &x.Cancelled)
			specialties = append(specialties, x)
			return err
		}, org, first, last)
	if len(specialties) != 2 || !slices.Equal(got.Specialties, specialties) {
		t.Errorf("specialties\n got %v\nwant %v", got.Specialties, specialties)
	}

	// The autumn clock change on its own: 25 hours of local day, both midnights excluded.
	dst := date(2026, 10, 25)
	day, err := svc.Appointments(t.Context(), clinicstats.Caller{AccountID: "acc", OrganizationID: org, Admin: true},
		clinicstats.AppointmentQuery{From: dst, To: dst})
	var n, refilled int64
	if err := p.owner.QueryRow(t.Context(), `SELECT count(*), count(*) FILTER (WHERE booked_via = 'waitlist') FROM appointment
		WHERE organization_id = $1 AND lower(period) >= '2026-10-24T22:00:00Z' AND lower(period) < '2026-10-25T23:00:00Z'`, org).Scan(&n, &refilled); err != nil {
		t.Fatal(err)
	}
	if err != nil || len(day.Series) != 1 || day.Series[0].Booked != n || day.Series[0].Refilled != refilled || n == 0 {
		t.Errorf("25 October: %v, %v; want %d appointments, %d refilled", day.Series, err, n, refilled)
	}
}
