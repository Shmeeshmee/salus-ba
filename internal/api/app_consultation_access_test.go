package api_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/pkg/clock"
)

// The end of access during a call through api.New (docs/consultation, package 5): the token refusal takes
// effect when the cancellation or the revocation commits, before the job that removes the patient has run (the
// api works no jobs, so it is still queued).

// queued returns the args of the jobs of kind still waiting on the default queue.
func (c consultationApp) queued(t *testing.T, kind string) []string {
	t.Helper()
	rows, err := c.owner.Query(t.Context(), `SELECT args::text FROM river_job WHERE kind = $1 AND queue = 'default' AND state = 'available'
		ORDER BY id`, kind)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var args string
		if err := rows.Scan(&args); err != nil {
			t.Fatal(err)
		}
		out = append(out, args)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

// Both joined; the practitioner cancels the appointment. The next token request of either is refused at once,
// and the cancellation queued consultation_access_end with the appointment's and the clinic's ids only.
func TestNew_RefusesTheTokenAtOnceAfterACancellation(t *testing.T) {
	c := newConsultationApp(t)
	appt := c.book(t, clock.System{}.Now().Add(5*time.Minute))
	join := "/appointments/" + appt + "/consultation/join"
	for _, cl := range []*client{c.d.patient, c.d.practitioner} {
		expectHTTP(t, cl, http.MethodPost, join, http.StatusOK, `"token"`)
	}

	expectHTTP(t, c.d.practitioner, http.MethodPost, "/appointments/"+appt+"/cancel", http.StatusOK, `"status":"cancelled"`)
	for _, cl := range []*client{c.d.patient, c.d.practitioner} {
		expectHTTP(t, cl, http.MethodPost, join, http.StatusConflict, `"consultation_ended"`)
	}
	want := `{"appointment_id": "` + appt + `", "organization_id": "` + inviteClinic + `"}`
	if got := c.queued(t, "consultation_access_end"); len(got) != 1 || got[0] != want {
		t.Errorf("queued consultation_access_end %v, want one with %s", got, want)
	}
}

// Both joined; the patient revokes their consent. The next token request of either is refused at once, while
// the consent_revoked job, which removes the patient from the room, is still queued.
func TestNew_RefusesTheTokenAtOnceAfterARevocation(t *testing.T) {
	c := newConsultationApp(t)
	appt := c.book(t, clock.System{}.Now().Add(5*time.Minute))
	join := "/appointments/" + appt + "/consultation/join"
	for _, cl := range []*client{c.d.patient, c.d.practitioner} {
		expectHTTP(t, cl, http.MethodPost, join, http.StatusOK, `"token"`)
	}

	expectHTTP(t, c.d.patient, http.MethodPost, "/consents/"+c.consent+"/revoke", http.StatusOK)
	expectHTTP(t, c.d.patient, http.MethodPost, join, http.StatusConflict, `"consent_required"`)
	// The practitioner no longer sees an appointment that had not started (SPEC.md §2 "Who may join").
	expectHTTP(t, c.d.practitioner, http.MethodPost, join, http.StatusNotFound)
	if got := c.queued(t, "consent_revoked"); len(got) != 1 {
		t.Errorf("queued consent_revoked %v, want the one job, not yet worked", got)
	}
}
