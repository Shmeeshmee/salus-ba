//go:build e2e && loadtest

package e2e_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Shmeeshmee/salus/pkg/clock"
)

// The time from a revocation's commit to the patient's removal from the call, through the running stack
// (`make bench-removal`, docs/consultation/PROGRESS.md, package 5): the api commits the revocation and enqueues
// consent_revoked, the worker picks it up (River's notify), cancels what had not started and removes the
// patient through LiveKit, which disconnects them. Not part of `make e2e`.
const removalRuns = 3

// TestRemovalLatency measures, removalRuns times each, a consultation that started before the revocation (the
// patient is removed, the practitioner stays) and one that had not (the appointment is cancelled and the room
// deleted). The commit lies between the consent's revoked_at, set just before it, and the revoke's response, so
// the time to the patient's disconnection is bracketed by the two.
func TestRemovalLatency(t *testing.T) {
	for _, tc := range []struct {
		name    string
		startIn time.Duration
		stays   bool // the practitioner stays in the room
	}{{"started", -time.Minute, true}, {"not started", 5 * time.Minute, false}} {
		var low, high []time.Duration
		for run := range removalRuns {
			lo, hi := removal(t, tc.startIn, tc.stays)
			t.Logf("%s, run %d: from the response %v, from revoked_at %v", tc.name, run, lo.Round(time.Millisecond), hi.Round(time.Millisecond))
			low, high = append(low, lo), append(high, hi)
		}
		t.Logf("%-12s from the response: median %v max %v; from revoked_at: median %v max %v", tc.name,
			pct(low, 50), pct(low, 100), pct(high, 50), pct(high, 100))
	}
}

// removal books a synthetic appointment starting startIn from now, connects the patient and the practitioner
// with the tokens the API gives them, revokes the patient's consent through the API and waits for the patient's
// disconnection. It returns the time from the revoke's response and from revoked_at to it.
func removal(t *testing.T, startIn time.Duration, practitionerStays bool) (fromResponse, fromRevokedAt time.Duration) {
	t.Helper()
	clinic := newBookableClinic(t)
	p := newPatientWithProfile(t)
	appt := p.bookFirstTime(clinic)
	startAt(t, appt.ID, startIn)

	patient := connect(t, p.api, appt.ID, false)
	practitioner := connect(t, clinic.practitioner, appt.ID, false)
	eventually(t, "both in the room", func() bool { return len(practitioner.room.GetRemoteParticipants()) == 1 })

	var consents struct{ Consents []struct{ ID string } }
	p.must(http.StatusOK, http.MethodGet, "/me/consents", nil, &consents)
	p.must(http.StatusOK, http.MethodPost, "/consents/"+consents.Consents[0].ID+"/revoke", nil, nil)
	responded := clock.System{}.Now()

	var gone time.Time
	select {
	case gone = <-patient.gone:
	case <-time.After(30 * time.Second):
		t.Fatal("the patient was not removed within 30s")
	}
	if code := p.do(http.MethodPost, "/appointments/"+appt.ID+"/consultation/join", nil, nil); code != http.StatusConflict {
		t.Errorf("the patient's next join = %d, want 409", code)
	}
	if practitionerStays {
		select {
		case <-practitioner.gone:
			t.Error("the practitioner left the consultation that started before the revocation")
		default:
		}
	} else {
		eventually(t, "the cancelled consultation's room deleted", func() bool { return len(practitioner.gone) == 1 })
	}
	return gone.Sub(responded), gone.Sub(revokedAt(t, p.patientID))
}

// revokedAt is when the patient's consent was revoked, set in the revocation's transaction before its commit.
func revokedAt(t *testing.T, patientID string) time.Time {
	t.Helper()
	var at []time.Time
	rows, err := ownerConn(t).Query(t.Context(), "SELECT revoked_at FROM consent WHERE patient_id = $1 AND revoked_at IS NOT NULL", patientID)
	if err != nil {
		t.Fatal(err)
	}
	if at, err = pgx.CollectRows(rows, pgx.RowTo[time.Time]); err != nil || len(at) != 1 {
		t.Fatalf("revoked_at %v, %v; want one", at, err)
	}
	return at[0]
}
