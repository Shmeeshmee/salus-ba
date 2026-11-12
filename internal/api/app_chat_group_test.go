package api_test

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/api"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// The group consult through api.New with row-level security on, as in production (docs/plans/chat.md, SAL-28): the
// treating practitioner invites a doctor of another, listed clinic, who sees nothing until the patient consents; then
// the doctor reads the history, writes and opens a doctor-only side thread the patient never sees; the patient's
// revocation ends the doctor's access to both. No message text reaches the log.
func TestNew_ServesTheGroupConsultAcrossClinicsUnderRowLevelSecurity(t *testing.T) {
	cfg := testConfig(t)
	owner := ownerPool(t, cfg)
	cfg.DatabaseURL = pgtest.AppURL(t, cfg.DatabaseURL)
	logs := &syncBuffer{}
	app, err := api.New(t.Context(), cfg, testLiveKit(), testObjectStore(t), testDrafter(), slog.New(slog.NewJSONHandler(logs, nil)))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(app.Close)
	d := newDirectory(t, app.Handler, owner)
	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := owner.Exec(t.Context(), q, args...); err != nil {
			t.Fatal(err)
		}
	}
	var profile struct{ ID string }
	decode(t, d.patient.do(http.MethodPut, "/me/patient", `{"givenName":"Zorana","familyName":"Syntheticova","birthDate":"1977-07-07"}`), &profile)
	if rec := d.patient.do(http.MethodPost, "/consents", `{"organizationId":"`+inviteClinic+`"}`); rec.Code != http.StatusCreated {
		t.Fatalf("the care consent = %d %s", rec.Code, rec.Body)
	}
	exec(`INSERT INTO appointment (patient_id, practitioner_role_id, organization_id, slot_id, period)
		VALUES ($1, $2, $3, gen_random_uuid(), tstzrange(now() - interval '2 days', now() - interval '2 days' + interval '30 minutes'))`,
		profile.ID, d.roleID, inviteClinic)

	// Clinic B, listed, with its admin and a practitioner: the guest.
	exec("INSERT INTO organization (id, name, listed) VALUES ($1, 'Other synthetic clinic', true)", otherClinic)
	exec("INSERT INTO invite (organization_id, expires_at, role, email) VALUES ($1, now() + interval '72 hours', 'admin', 'fay@example.test')", otherClinic)
	otherAdmin := &client{t: t, h: app.Handler}
	otherAdmin.bearer = accept(t, otherAdmin, issueToken(t, exec, "fay@example.test"))
	if rec := otherAdmin.do(http.MethodPost, "/invites", `{"email":"gus@example.test","role":"practitioner","givenName":"Gus","familyName":"Example"}`); rec.Code != http.StatusCreated {
		t.Fatalf("POST /invites in clinic B = %d %s", rec.Code, rec.Body)
	}
	guest := &client{t: t, h: app.Handler}
	guest.bearer = accept(t, guest, issueToken(t, exec, "gus@example.test"))
	var guestRole string
	if err := owner.QueryRow(t.Context(), "SELECT id::text FROM practitioner_role WHERE organization_id = $1", otherClinic).Scan(&guestRole); err != nil {
		t.Fatal(err)
	}

	var conv struct{ ID string }
	decode(t, d.patient.do(http.MethodPost, "/conversations", `{"organization_id":"`+inviteClinic+`","practitioner_role_id":"`+d.roleID+`"}`), &conv)
	path := "/conversations/" + conv.ID
	if rec := d.patient.do(http.MethodPost, path+"/messages", `{"client_message_id":"c1000000-0000-4000-8000-0000000000c1","body":"Synthetic placeholder question"}`); rec.Code != http.StatusCreated {
		t.Fatalf("send = %d %s", rec.Code, rec.Body)
	}
	rec := d.practitioner.do(http.MethodPost, path+"/participants",
		`{"practitioner_role_id":"`+guestRole+`","organization_id":"`+otherClinic+`","reason":"second_opinion","history":"full"}`)
	var invited struct{ ID, Status string }
	if rec.Code != http.StatusCreated || json.Unmarshal(rec.Body.Bytes(), &invited) != nil || invited.Status != "pending_consent" {
		t.Fatalf("invite = %d %s", rec.Code, rec.Body)
	}
	for _, p := range []string{path, path + "/messages", path + "/threads"} {
		expectHTTP(t, guest, http.MethodGet, p, http.StatusNotFound)
	}
	if body := expectHTTP(t, guest, http.MethodGet, "/conversations", http.StatusOK); strings.Contains(body, conv.ID) {
		t.Errorf("the guest's list before the consent: %s", body)
	}
	expectHTTP(t, d.patient, http.MethodGet, "/me/chat-invitations", http.StatusOK, `"id":"`+invited.ID+`"`, `"reason":"second_opinion"`)
	expectHTTP(t, d.practitioner, http.MethodGet, "/me/chat-invitations", http.StatusForbidden)

	if rec := d.patient.do(http.MethodPost, "/me/chat-invitations/"+invited.ID+"/decision", `{"decision":"grant"}`); rec.Code != http.StatusNoContent {
		t.Fatalf("grant = %d %s", rec.Code, rec.Body)
	}
	expectHTTP(t, guest, http.MethodGet, path+"/messages", http.StatusOK, "Synthetic placeholder question", `"code":"consent_granted"`, `"code":"participant_invited"`)
	if rec := guest.do(http.MethodPost, path+"/messages", `{"client_message_id":"c2000000-0000-4000-8000-0000000000c2","body":"Synthetic placeholder opinion"}`); rec.Code != http.StatusCreated {
		t.Errorf("the guest's send = %d %s", rec.Code, rec.Body)
	}
	var thread struct{ ID string }
	decode(t, guest.do(http.MethodPost, path+"/threads", ""), &thread)
	if thread.ID == "" {
		t.Fatal("no side thread")
	}
	threadPath := "/conversations/" + thread.ID
	if rec := guest.do(http.MethodPost, threadPath+"/messages", `{"client_message_id":"c3000000-0000-4000-8000-0000000000c3","body":"Synthetic placeholder between doctors"}`); rec.Code != http.StatusCreated {
		t.Errorf("the guest's send in the thread = %d %s", rec.Code, rec.Body)
	}
	expectHTTP(t, d.practitioner, http.MethodGet, threadPath+"/messages", http.StatusOK, "Synthetic placeholder between doctors")
	expectHTTP(t, d.practitioner, http.MethodGet, path+"/threads", http.StatusOK, `"id":"`+thread.ID+`"`)

	// The patient never sees the side thread.
	expectHTTP(t, d.patient, http.MethodGet, threadPath, http.StatusNotFound)
	expectHTTP(t, d.patient, http.MethodGet, threadPath+"/messages", http.StatusNotFound)
	expectHTTP(t, d.patient, http.MethodGet, path+"/threads", http.StatusForbidden)
	for _, p := range []string{path + "/messages", "/conversations"} {
		if body := expectHTTP(t, d.patient, http.MethodGet, p, http.StatusOK); strings.Contains(body, "between doctors") || strings.Contains(body, thread.ID) {
			t.Errorf("the patient's %s shows the side thread: %s", p, body)
		}
	}

	if rec := d.patient.do(http.MethodPost, "/me/chat-invitations/"+invited.ID+"/revoke", ""); rec.Code != http.StatusNoContent {
		t.Fatalf("revoke = %d %s", rec.Code, rec.Body)
	}
	for _, p := range []string{path, path + "/messages", threadPath, threadPath + "/messages"} {
		expectHTTP(t, guest, http.MethodGet, p, http.StatusNotFound)
	}
	expectHTTP(t, d.patient, http.MethodGet, path+"/messages", http.StatusOK, `"code":"participant_removed"`)
	var audited int
	if err := owner.QueryRow(t.Context(), `SELECT count(*) FROM audit_event WHERE entity_name = 'conversation_participant'`).Scan(&audited); err != nil || audited < 4 {
		t.Errorf("participant events audited: %d (%v), want the invitation, the list, the grant and the revocation", audited, err)
	}
	if strings.Contains(logs.String(), "Synthetic placeholder") {
		t.Error("the log holds a message's text")
	}
}
