package migrations_test

import (
	"testing"

	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// clinic_appointment_stat counts the appointments by how they were booked too (00057, SAL-36): a direct and a
// waitlist booking of the same hour, role and status are two rows; a row without it is a direct booking (every day
// counted before the waitlist existed); no other way of booking.
func TestNoShowReports_CountsTheAppointmentsByHowTheyWereBooked(t *testing.T) {
	t.Parallel()
	owner := pgtest.NewDatabase(t)
	d := seedDirectory(t, owner)
	insert := `INSERT INTO clinic_appointment_stat (organization_id, practitioner_role_id, day, hour, specialty_code, status, booked_via, count)
		VALUES ($1, $2, '2026-11-02', 9, '', 'noshow', $3, 1)`
	for _, via := range []string{"direct", "waitlist"} {
		if _, err := owner.Exec(t.Context(), insert, clinicA, d[clinicA].role, via); err != nil {
			t.Errorf("booked_via %s: %v", via, err)
		}
	}
	if _, err := owner.Exec(t.Context(), insert, clinicA, d[clinicA].role, "phone"); sqlState(err) != "23514" {
		t.Errorf("booked_via phone: %v, want a check violation", err)
	}
	var via string
	err := owner.QueryRow(t.Context(), `INSERT INTO clinic_appointment_stat (organization_id, practitioner_role_id, day, hour, specialty_code, status, count)
		VALUES ($1, $2, '2026-11-02', 10, '', 'fulfilled', 2) RETURNING booked_via`, clinicA, d[clinicA].role).Scan(&via)
	if err != nil || via != "direct" {
		t.Errorf("a row without booked_via: %q, %v; want direct", via, err)
	}
}
