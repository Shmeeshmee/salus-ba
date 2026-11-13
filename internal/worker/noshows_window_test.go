package worker_test

import (
	"testing"

	"github.com/riverqueue/river"

	"github.com/Shmeeshmee/salus/internal/videosession"
)

// SAL-36: the window's close is where the system marks a no-show. The practitioner joined and the patient never did:
// the appointment becomes noshow with one status_set by the system (the job as agent) in its history; a retry
// writes no second one.
func TestConsultationWindowClose_MarksTheNoShowInTheHistory(t *testing.T) {
	t.Parallel()
	db := newLogins(t)
	b := seedBooking(t, db.owner)
	c := seedConsultation(t, db, b, true, false)
	for run := range 2 {
		ended, logs := runWorkerJob(t, db, videosession.WindowCloseArgs{SessionID: c.session, OrganizationID: clinicID})
		if ended != river.EventKindJobCompleted {
			t.Fatalf("run %d: the job ended %s, want completed:\n%s", run, ended, logs)
		}
	}
	var appt string
	if err := db.owner.QueryRow(t.Context(), `SELECT status FROM appointment WHERE id = $1`, b.past).Scan(&appt); err != nil {
		t.Fatal(err)
	}
	n := countWhere(t, db, `SELECT count(*) FROM appointment_change WHERE appointment_id = $1 AND kind = 'status_set' AND via = 'system'
		AND agent = 'consultation_window_close' AND by_account_id IS NULL AND from_status = 'booked' AND to_status = 'noshow'`, b.past)
	if appt != "noshow" || n != 1 || countWhere(t, db, `SELECT count(*) FROM appointment_change WHERE appointment_id = $1`, b.past) != 1 {
		t.Errorf("appointment %s, %d system no-shows in its history; want noshow and exactly one", appt, n)
	}
}
