package api_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/videosession"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// LiveKit's webhook on the real database, through the services as New builds them (docs/consultation,
// package 3).

// names are an open session's room and identities, as LiveKit knows them.
type names struct{ room, patient, practitioner string }

// openSession opens the appointment's session with the patient's join and returns its names.
func openSession(t *testing.T, owner *pgxpool.Pool, svc *videosession.Service, s consultationSeed) names {
	t.Helper()
	pat, _ := callers(s)
	if _, err := svc.Join(t.Context(), pat, s.appointment); err != nil {
		t.Fatal(err)
	}
	var n names
	if err := owner.QueryRow(t.Context(), `SELECT room, patient_identity, practitioner_identity FROM video_session WHERE appointment_id = $1`,
		s.appointment).Scan(&n.room, &n.patient, &n.practitioner); err != nil {
		t.Fatal(err)
	}
	return n
}

// deliverEvent hands e to the service as a signed webhook.
func deliverEvent(t *testing.T, svc *videosession.Service, e videosession.Event) error {
	t.Helper()
	body, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.HandleWebhook(t.Context(), body, "signed")
	return err
}

// consultationState is what the webhook may change of the appointment's consultation: the session's joins
// and end, the Encounter, the appointment's status and how many audit events the webhook wrote in its clinic.
func consultationState(t *testing.T, owner *pgxpool.Pool, s consultationSeed) string {
	t.Helper()
	var state string
	err := owner.QueryRow(t.Context(), `SELECT concat_ws(' | ',
			coalesce(v.patient_joined_at::text, '-'), coalesce(v.practitioner_joined_at::text, '-'),
			coalesce(v.ended_at::text, '-'), coalesce(v.end_reason, '-'),
			coalesce(e.status, '-'), coalesce(e.period_start::text, '-'), coalesce(e.period_end::text, '-'), a.status,
			(SELECT count(*) FROM audit_event ae WHERE ae.agent_name = 'livekit_webhook' AND ae.organization_id = a.organization_id))
		FROM video_session v JOIN appointment a ON a.id = v.appointment_id LEFT JOIN encounter e ON e.appointment_id = a.id
		WHERE v.appointment_id = $1`, s.appointment).Scan(&state)
	if err != nil {
		t.Fatal(err)
	}
	return state
}

// permutations returns every order of the indexes 0 to n-1.
func permutations(n int) [][]int {
	if n == 0 {
		return [][]int{{}}
	}
	var out [][]int
	for _, p := range permutations(n - 1) {
		for i := range len(p) + 1 {
			out = append(out, slices.Insert(slices.Clone(p), i, n-1))
		}
	}
	return out
}

// During the window, every order of the practitioner's and the patient's joins and leaves and the room
// finishing, each delivered twice, reaches the same state: both joins at LiveKit's times, the Encounter in
// progress from the practitioner's join, the appointment booked, and three events by the webhook (the two
// joins and the Encounter's create).
func TestHandleWebhook_EveryOrderReachesTheSameState(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	at := time.Date(2030, 1, 7, 9, 0, 0, 0, time.UTC)
	svc := realService(t, app, at.Add(10*time.Minute), &recordingProvider{})
	practitionerAt, patientAt := at.Add(-2*time.Minute+250*time.Millisecond), at.Add(-time.Minute)

	var want string
	orders := permutations(5)
	if len(orders) != 120 {
		t.Fatalf("%d orders, want 120", len(orders))
	}
	for i, order := range orders {
		s := seedConsultation(t, owner, at)
		n := openSession(t, owner, svc, s)
		events := []videosession.Event{
			{Kind: videosession.EventParticipantJoined, Room: n.room, Identity: n.practitioner, At: practitionerAt},
			{Kind: videosession.EventParticipantJoined, Room: n.room, Identity: n.patient, At: patientAt},
			{Kind: videosession.EventParticipantLeft, Room: n.room, Identity: n.practitioner, At: at},
			{Kind: videosession.EventParticipantLeft, Room: n.room, Identity: n.patient, At: at},
			{Kind: videosession.EventRoomFinished, Room: n.room, At: at},
		}
		ordered := make([]videosession.Event, 0, len(order))
		for _, j := range order {
			ordered = append(ordered, events[j])
		}
		for _, e := range slices.Concat(ordered, ordered) {
			if err := deliverEvent(t, svc, e); err != nil {
				t.Fatalf("order %d, %s: %v", i, e.Kind, err)
			}
		}
		got := consultationState(t, owner, s)
		if i == 0 {
			want = got
			continue
		}
		if got != want {
			t.Errorf("order %d (%v) reached %s, want %s", i, kinds(ordered, n), got, want)
		}
	}
	wantState := fmt.Sprintf("%s | %s | - | - | in-progress | %s | - | booked | 3",
		pgTime(patientAt), pgTime(practitionerAt), pgTime(practitionerAt))
	if want != wantState {
		t.Errorf("every order reached %s, want %s", want, wantState)
	}
}

// pgTime is t as Postgres prints a timestamptz in UTC.
func pgTime(t time.Time) string {
	return t.UTC().Format("2006-01-02 15:04:05.999999") + "+00"
}

// kinds names the events of an order, for a failure.
func kinds(events []videosession.Event, n names) []string {
	who := map[string]string{n.practitioner: "practitioner", n.patient: "patient", "": "room"}
	out := make([]string, 0, len(events))
	for _, e := range events {
		out = append(out, who[e.Identity]+" "+string(e.Kind))
	}
	return out
}

// After the window has closed, the last one leaving ends the consultation: the session ends
// window_closed, the Encounter finishes, the appointment is fulfilled, all by the webhook; a repeat of every
// event changes nothing.
func TestHandleWebhook_TheLastOneLeavingAfterTheWindowEndsTheConsultation(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	at := time.Date(2030, 1, 7, 9, 0, 0, 0, time.UTC)
	closes := at.Add(time.Hour)
	s := seedConsultation(t, owner, at)
	n := openSession(t, owner, realService(t, app, at.Add(-5*time.Minute), &recordingProvider{}), s)
	svc := realService(t, app, closes, &recordingProvider{})
	events := []videosession.Event{
		{Kind: videosession.EventParticipantJoined, Room: n.room, Identity: n.practitioner, At: at},
		{Kind: videosession.EventParticipantJoined, Room: n.room, Identity: n.patient, At: at},
		{Kind: videosession.EventParticipantLeft, Room: n.room, Identity: n.patient, At: closes},
		{Kind: videosession.EventParticipantLeft, Room: n.room, Identity: n.practitioner, At: closes},
		{Kind: videosession.EventRoomFinished, Room: n.room, At: closes},
	}
	var states []string
	for range 2 {
		for _, e := range events {
			if err := deliverEvent(t, svc, e); err != nil {
				t.Fatal(err)
			}
		}
		states = append(states, consultationState(t, owner, s))
	}
	want := fmt.Sprintf("%s | %s | %s | window_closed | finished | %s | %s | fulfilled | 6",
		pgTime(at), pgTime(at), pgTime(closes), pgTime(at), pgTime(closes))
	if states[0] != want || states[1] != want {
		t.Errorf("states %v, want %s twice", states, want)
	}
}

// A patient who joins after their consent was revoked (with a token issued just before) is removed from the
// room; their join is recorded. An event for a room we do not know changes nothing.
func TestHandleWebhook_RemovesAPatientJoiningAfterARevocation(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	at := time.Date(2030, 1, 7, 9, 0, 0, 0, time.UTC)
	s := seedConsultation(t, owner, at)
	p := &recordingProvider{}
	svc := realService(t, app, at.Add(-5*time.Minute), p)
	n := openSession(t, owner, svc, s)
	if _, err := owner.Exec(t.Context(), `UPDATE consent SET status = 'inactive', revoked_at = $2 WHERE patient_id = $1`,
		s.patient, at.Add(-6*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := deliverEvent(t, svc, videosession.Event{Kind: videosession.EventParticipantJoined, Room: n.room, Identity: n.patient, At: at.Add(-4 * time.Minute)}); err != nil {
		t.Fatal(err)
	}
	if want := []string{n.room + " " + n.patient}; !slices.Equal(p.removed, want) {
		t.Errorf("removed %v, want the patient", p.removed)
	}
	var joined bool
	if err := owner.QueryRow(t.Context(), `SELECT patient_joined_at IS NOT NULL FROM video_session`).Scan(&joined); err != nil || !joined {
		t.Errorf("patient joined %t, %v; want the join recorded", joined, err)
	}
	before := consultationState(t, owner, s)
	err := deliverEvent(t, svc, videosession.Event{Kind: videosession.EventParticipantJoined, Room: "r_unknown", Identity: n.practitioner, At: at})
	if !errors.Is(err, videosession.ErrUnknownRoom) || consultationState(t, owner, s) != before {
		t.Errorf("an unknown room: %v; want ErrUnknownRoom and nothing changed", err)
	}
}
