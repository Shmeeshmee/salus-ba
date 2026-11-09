package api_test

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/api"
	"github.com/Shmeeshmee/salus/internal/config"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

// The consultation's JSON API and the FHIR Encounter through api.New, on the real database and the compose
// LiveKit (docs/consultation, package 4).

// composeLiveKitConfig is the compose LiveKit (TEST_LIVEKIT_*) as the api's configuration; without it the
// test fails.
func composeLiveKitConfig(t *testing.T) config.LiveKit {
	t.Helper()
	lk := config.LiveKit{
		URL: os.Getenv("TEST_LIVEKIT_URL"), APIKey: os.Getenv("TEST_LIVEKIT_API_KEY"), APISecret: os.Getenv("TEST_LIVEKIT_API_SECRET"),
		PublicURL: "ws://localhost:7880",
	}
	if lk.URL == "" || lk.APIKey == "" || lk.APISecret == "" {
		t.Fatal("TEST_LIVEKIT_URL, TEST_LIVEKIT_API_KEY and TEST_LIVEKIT_API_SECRET are not set; run the tests with `make test`")
	}
	return lk
}

// consultationApp is the api on the compose LiveKit with the directory's admin, practitioner and patient (with
// a profile and an active consent), a second patient with no appointment, and the owner's pool.
type consultationApp struct {
	log       *syncBuffer
	owner     *pgxpool.Pool
	stranger  *client
	h         http.Handler
	d         directory
	lk        config.LiveKit
	consent   string // the patient's consent with the clinic
	patientID string // the patient's profile
	redisURL  string // the api's Redis, which carries the encounters' streams
}

func newConsultationApp(t *testing.T) consultationApp {
	t.Helper()
	cfg := testConfig(t)
	c := consultationApp{log: &syncBuffer{}, lk: composeLiveKitConfig(t), redisURL: cfg.RedisURL}
	app, err := api.New(t.Context(), cfg, c.lk, testObjectStore(t), slog.New(slog.NewJSONHandler(c.log, nil)))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(app.Close)
	c.h, c.owner = app.Handler, ownerPool(t, cfg)
	c.d = newDirectory(t, app.Handler, c.owner)
	var profile, consent struct{ ID string }
	decode(t, c.d.patient.do(http.MethodPut, "/me/patient", `{"givenName":"Zorana","familyName":"Syntheticova","birthDate":"1977-07-07","phone":"+38760000777"}`), &profile)
	decode(t, c.d.patient.do(http.MethodPost, "/consents", `{"organizationId":"`+inviteClinic+`"}`), &consent)
	if profile.ID == "" || consent.ID == "" {
		t.Fatalf("the patient's profile %q and consent %q, want both", profile.ID, consent.ID)
	}
	c.patientID, c.consent = profile.ID, consent.ID

	c.stranger = &client{t: t, h: app.Handler}
	c.stranger.do(http.MethodPost, "/auth/register", `{"email":"eve@example.test","password":"`+testPassword+`"}`)
	var login struct{ Token string }
	decode(t, c.stranger.do(http.MethodPost, "/auth/login", `{"email":"eve@example.test","password":"`+testPassword+`"}`), &login)
	c.stranger.bearer = login.Token
	return c
}

// book stores, as the owner, the patient's appointment with the role booked from start for 30 minutes.
func (c consultationApp) book(t *testing.T, start time.Time) string {
	t.Helper()
	var id string
	if err := c.owner.QueryRow(t.Context(), `INSERT INTO appointment (patient_id, practitioner_role_id, organization_id, slot_id, period)
		VALUES ($1, $2, $3, gen_random_uuid(), tstzrange($4::timestamptz, $4::timestamptz + interval '30 minutes')) RETURNING id::text`,
		c.patientID, c.d.roleID, inviteClinic, start).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

// joined sends LiveKit's signed participant_joined of the role's identity in the appointment's room, at.
func (c consultationApp) joined(t *testing.T, appointmentID, role string, at time.Time) {
	t.Helper()
	var room, identity string
	if err := c.owner.QueryRow(t.Context(), `SELECT room, CASE WHEN $2 = 'practitioner' THEN practitioner_identity ELSE patient_identity END
		FROM video_session WHERE appointment_id = $1`, appointmentID, role).Scan(&room, &identity); err != nil {
		t.Fatal(err)
	}
	body := fmt.Sprintf(`{"event":"participant_joined","room":{"name":%q},"participant":{"identity":%q,"joinedAt":"%d"},"createdAt":"%d"}`,
		room, identity, at.Unix(), at.Unix())
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/webhooks/livekit", strings.NewReader(body))
	req.Header.Set("Authorization", livekitSignature(t, c.lk, []byte(body)))
	rec := httptest.NewRecorder()
	c.h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("the %s's participant_joined = %d %s, want 200", role, rec.Code, rec.Body)
	}
}

// expectHTTP fails unless cl's method path answers status with a body holding each of want.
func expectHTTP(t *testing.T, cl *client, method, path string, status int, want ...string) string {
	t.Helper()
	rec := cl.do(method, path, "")
	for _, w := range want {
		if !strings.Contains(rec.Body.String(), w) {
			rec.Code = -rec.Code
		}
	}
	if rec.Code != status {
		t.Errorf("%s %s = %d %s; want %d with %q", method, path, rec.Code, rec.Body, status, want)
	}
	return rec.Body.String()
}

// The patient and the practitioner join; the status follows the session for both and the admin; only the
// practitioner ends it, once they joined, and again with the same answer; the Encounter is read and searched
// by those who see the appointment; every read is audited and the log holds no token or name.
func TestNew_ServesTheConsultationsJoinStatusAndEnd(t *testing.T) {
	c := newConsultationApp(t)
	now := clock.System{}.Now()
	appt := c.book(t, now.Add(5*time.Minute))
	consultation := "/appointments/" + appt + "/consultation"

	for _, cl := range []*client{c.d.patient, c.d.practitioner, c.d.admin} {
		expectHTTP(t, cl, http.MethodGet, consultation, http.StatusOK, `{"state":"not_started"}`)
	}
	expectHTTP(t, c.stranger, http.MethodGet, consultation, http.StatusNotFound, `"not_found"`)
	expectHTTP(t, c.stranger, http.MethodPost, consultation+"/join", http.StatusNotFound, `"not_found"`)
	expectHTTP(t, c.d.admin, http.MethodPost, consultation+"/join", http.StatusNotFound, `"not_found"`)
	expectHTTP(t, c.d.practitioner, http.MethodPost, consultation+"/end", http.StatusConflict, `"consultation_not_started"`)

	tokens := []string{}
	for _, tc := range []struct {
		cl         *client
		role, peer string
	}{
		{cl: c.d.patient, role: "patient", peer: "Cy Example"},
		{cl: c.d.practitioner, role: "practitioner", peer: "Zorana Syntheticova"},
	} {
		var j struct{ URL, Token, ExpiresAt, PeerName, Role string }
		rec := tc.cl.do(http.MethodPost, consultation+"/join", "")
		decode(t, rec, &j)
		if rec.Code != http.StatusOK || j.URL != "ws://localhost:7880" || j.Token == "" || j.PeerName != tc.peer || j.Role != tc.role ||
			rec.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("the %s's join = %d %s, want 200 with a token, %q and the role", tc.role, rec.Code, rec.Body, tc.peer)
		}
		tokens = append(tokens, j.Token)
	}
	expectHTTP(t, c.d.admin, http.MethodGet, consultation, http.StatusOK, `{"state":"open"}`)

	joinedAt := now.Add(-2 * time.Minute).UTC().Truncate(time.Second) // inside the window, before the end
	c.joined(t, appt, "practitioner", joinedAt)
	c.joined(t, appt, "patient", joinedAt.Add(time.Minute))
	at := joinedAt.Format(time.RFC3339)
	expectHTTP(t, c.d.admin, http.MethodGet, consultation, http.StatusOK, `"state":"open"`, `"practitionerJoinedAt":"`+at+`"`, `"patientJoinedAt":"`)
	checkEncounterHTTP(t, c, appt, "in-progress")

	expectHTTP(t, c.d.patient, http.MethodPost, consultation+"/end", http.StatusNotFound, `"not_found"`)
	expectHTTP(t, c.d.admin, http.MethodPost, consultation+"/end", http.StatusNotFound, `"not_found"`)
	ended := expectHTTP(t, c.d.practitioner, http.MethodPost, consultation+"/end", http.StatusOK, `"state":"ended"`, `"endReason":"practitioner"`)
	if again := expectHTTP(t, c.d.practitioner, http.MethodPost, consultation+"/end", http.StatusOK); again != ended {
		t.Errorf("ending again = %s, want %s", again, ended)
	}
	checkEncounterHTTP(t, c, appt, "finished")
	expectHTTP(t, c.d.patient, http.MethodGet, "/fhir/R4/Appointment/"+appt, http.StatusOK, `"status":"fulfilled"`)
	expectHTTP(t, c.d.patient, http.MethodPost, consultation+"/join", http.StatusConflict, `"consultation_ended"`)

	count := func(q string, args ...any) (n int) {
		if err := c.owner.QueryRow(t.Context(), q, args...).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	// Five status reads that succeeded and the patient's FHIR read of the appointment; the stranger's refusal
	// has no entity.
	if n := count(`SELECT count(*) FROM audit_event WHERE subtype_code = 'read' AND entity_reference = $1 AND outcome = 0`, "Appointment/"+appt); n != 6 {
		t.Errorf("%d audited reads of the appointment, want 6", n)
	}
	if n := count(`SELECT count(*) FROM audit_event WHERE subtype_code = 'operation' AND entity_reference = $1 AND agent_name = 'consultation_end'`, "Appointment/"+appt); n != 1 {
		t.Errorf("%d ends by consultation_end audited, want 1", n)
	}
	log := c.log.String()
	for _, secret := range append(tokens, "Zorana", "Syntheticova", "Cy Example") {
		if strings.Contains(log, secret) {
			t.Errorf("the log holds %q", secret)
		}
	}
}

// checkEncounterHTTP: the patient, the practitioner and the admin find the appointment's Encounter in status
// and read it, audited; the stranger reads 404 and finds none; an unknown parameter is 400.
func checkEncounterHTTP(t *testing.T, c consultationApp, appt, status string) {
	t.Helper()
	var id string
	if err := c.owner.QueryRow(t.Context(), `SELECT id::text FROM encounter WHERE appointment_id = $1`, appt).Scan(&id); err != nil {
		t.Fatal(err)
	}
	before := 0
	_ = c.owner.QueryRow(t.Context(), `SELECT count(*) FROM audit_event WHERE entity_reference = $1`, "Encounter/"+id).Scan(&before)
	for _, cl := range []*client{c.d.patient, c.d.practitioner, c.d.admin} {
		expectHTTP(t, cl, http.MethodGet, "/fhir/R4/Encounter?appointment=Appointment/"+appt, http.StatusOK, `"total":1`, `"status":"`+status+`"`)
		expectHTTP(t, cl, http.MethodGet, "/fhir/R4/Encounter/"+id, http.StatusOK, `"resourceType":"Encounter"`, `"status":"`+status+`"`,
			`"reference":"Appointment/`+appt+`"`, `"code":"VR"`)
	}
	expectHTTP(t, c.stranger, http.MethodGet, "/fhir/R4/Encounter/"+id, http.StatusNotFound, `"not-found"`)
	expectHTTP(t, c.stranger, http.MethodGet, "/fhir/R4/Encounter?appointment="+appt, http.StatusOK, `"total":0`)
	expectHTTP(t, c.d.admin, http.MethodGet, "/fhir/R4/Encounter?status="+status, http.StatusBadRequest, `"not-supported"`)
	var after int
	if err := c.owner.QueryRow(t.Context(), `SELECT count(*) FROM audit_event WHERE entity_reference = $1`, "Encounter/"+id).Scan(&after); err != nil || after-before != 6 {
		t.Errorf("%d new audit events of the Encounter, %v; want 6 (three searches, three reads)", after-before, err)
	}
}

// A join is refused with its code: before the window opens, on a session that ended without the
// practitioner (whose status the clinic reads, with the patient's join), and after the revocation.
func TestNew_RefusesAConsultationJoinWithItsCode(t *testing.T) {
	c := newConsultationApp(t)
	now := clock.System{}.Now().UTC().Truncate(time.Second)
	later, past, soon := c.book(t, now.Add(2*time.Hour)), c.book(t, now.Add(-2*time.Hour)), c.book(t, now.Add(5*time.Minute))
	expectHTTP(t, c.d.patient, http.MethodPost, "/appointments/"+later+"/consultation/join", http.StatusConflict, `"join_window_not_open"`)

	closes := now.Add(-time.Hour)
	if _, err := c.owner.Exec(t.Context(), `INSERT INTO video_session (appointment_id, organization_id, room, patient_identity,
		practitioner_identity, closes_at, created_at, updated_at, patient_joined_at, ended_at, end_reason)
		VALUES ($1, $2, 'r_absent', 'p_absent_patient', 'p_absent_practitioner', $3, $4, $3, $4, $3, 'practitioner_absent')`,
		past, inviteClinic, closes, now.Add(-2*time.Hour)); err != nil {
		t.Fatal(err)
	}
	for _, cl := range []*client{c.d.patient, c.d.practitioner} {
		expectHTTP(t, cl, http.MethodPost, "/appointments/"+past+"/consultation/join", http.StatusConflict, `"practitioner_did_not_join"`)
	}
	expectHTTP(t, c.d.admin, http.MethodGet, "/appointments/"+past+"/consultation", http.StatusOK, `"state":"ended"`,
		`"patientJoinedAt":"`+now.Add(-2*time.Hour).Format(time.RFC3339)+`"`, `"endReason":"practitioner_absent"`)

	expectHTTP(t, c.d.patient, http.MethodPost, "/consents/"+c.consent+"/revoke", http.StatusOK)
	expectHTTP(t, c.d.patient, http.MethodPost, "/appointments/"+soon+"/consultation/join", http.StatusConflict, `"consent_required"`)
}
