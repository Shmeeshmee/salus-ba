package api_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// SAL-29 through api.New: the patient's app registers its device (staff cannot), the admin sets the clinic's chat
// settings and the patient and the practitioner read its hours, the admin creates a webhook and sees its secret
// once; a message enqueues one notification job with ids only; the log holds no token, secret or text.
func TestNew_ServesTheChatsNotificationSettingsDevicesAndWebhooks(t *testing.T) {
	c := newConsultationApp(t)
	c.book(t, wallNow().Add(-48*time.Hour))

	devices := `{"platform":"android","token":"synthetic-device-token-77"}`
	if rec := c.d.patient.do(http.MethodPut, "/me/devices", devices); rec.Code != http.StatusNoContent {
		t.Errorf("the patient registers = %d %s", rec.Code, rec.Body)
	}
	if rec := c.d.practitioner.do(http.MethodPut, "/me/devices", devices); rec.Code != http.StatusForbidden {
		t.Errorf("staff register = %d, want 403", rec.Code)
	}

	settings := `{"hours":[{"day":"mon","start":"08:00","end":"16:00"}],"response_time_hours":8,"email_after_minutes":30}`
	if rec := c.d.admin.do(http.MethodPut, "/clinic/chat-settings", settings); rec.Code != http.StatusOK {
		t.Fatalf("PUT settings = %d %s", rec.Code, rec.Body)
	}
	expectHTTP(t, c.d.admin, http.MethodGet, "/clinic/chat-settings", http.StatusOK, `"email_after_minutes":30`)
	if rec := c.d.practitioner.do(http.MethodPut, "/clinic/chat-settings", settings); rec.Code != http.StatusForbidden {
		t.Errorf("the practitioner sets = %d, want 403", rec.Code)
	}
	hours := "/organizations/" + inviteClinic + "/chat-hours"
	expectHTTP(t, c.d.patient, http.MethodGet, hours, http.StatusOK, `"response_time_hours":8`, `"time_zone":"Europe/Sarajevo"`, `"open_now":`)
	expectHTTP(t, c.d.practitioner, http.MethodGet, hours, http.StatusOK, `"day":"mon"`)
	expectHTTP(t, c.stranger, http.MethodGet, hours, http.StatusNotFound)

	rec := c.d.admin.do(http.MethodPost, "/clinic/webhooks", `{"url":"https://hooks.example.test/salus","events":["message.created"]}`)
	var created struct{ ID, Secret string }
	if rec.Code != http.StatusCreated || json.Unmarshal(rec.Body.Bytes(), &created) != nil || !strings.HasPrefix(created.Secret, "whsec_") {
		t.Fatalf("POST webhook = %d %s", rec.Code, rec.Body)
	}
	if list := expectHTTP(t, c.d.admin, http.MethodGet, "/clinic/webhooks", http.StatusOK, created.ID); strings.Contains(list, created.Secret) {
		t.Errorf("the list shows the secret: %s", list)
	}
	expectHTTP(t, c.d.practitioner, http.MethodGet, "/clinic/webhooks", http.StatusForbidden)
	var sealed []byte
	if err := c.owner.QueryRow(t.Context(), `SELECT secret_sealed FROM webhook_endpoint WHERE id = $1`, created.ID).Scan(&sealed); err != nil ||
		strings.Contains(string(sealed), created.Secret) {
		t.Errorf("the stored secret is not sealed (%v)", err)
	}

	rec = c.d.patient.do(http.MethodPost, "/conversations", `{"organization_id":"`+inviteClinic+`","practitioner_role_id":"`+c.d.roleID+`"}`)
	var conv struct{ ID string }
	if rec.Code != http.StatusCreated || json.Unmarshal(rec.Body.Bytes(), &conv) != nil {
		t.Fatalf("open = %d %s", rec.Code, rec.Body)
	}
	const text = "Synthetic secret text 41be: never in a job"
	if rec := c.d.patient.do(http.MethodPost, "/conversations/"+conv.ID+"/messages", `{"client_message_id":"c1000000-0000-4000-8000-0000000000c1","body":"`+text+`"}`); rec.Code != http.StatusCreated {
		t.Fatalf("send = %d %s", rec.Code, rec.Body)
	}
	var kind, queue, args string
	if err := c.owner.QueryRow(t.Context(), `SELECT kind, queue, args::text FROM river_job WHERE kind = 'chat_message_sent'`).Scan(&kind, &queue, &args); err != nil {
		t.Fatalf("the notification job: %v", err)
	}
	if queue != "push" || strings.Contains(args, "41be") || !strings.Contains(args, conv.ID) {
		t.Errorf("job %s on %s with %s", kind, queue, args)
	}

	if rec := c.d.admin.do(http.MethodDelete, "/clinic/webhooks/"+created.ID, ""); rec.Code != http.StatusNoContent {
		t.Errorf("DELETE webhook = %d", rec.Code)
	}
	if rec := c.d.patient.do(http.MethodDelete, "/me/devices", devices); rec.Code != http.StatusNoContent {
		t.Errorf("the patient unregisters = %d", rec.Code)
	}
	for _, never := range []string{"synthetic-device-token-77", created.Secret, "41be", "hooks.example.test"} {
		if strings.Contains(c.log.String(), never) {
			t.Errorf("the log holds %q", never)
		}
	}
}

// The clinic's appointment settings through api.New (docs/plans/appointments.md, step 0): the admin reads the defaults
// and replaces them, a practitioner may not, and the paid SMS option is refused while the SMS engine is the fake.
func TestNew_ServesTheClinicsAppointmentSettings(t *testing.T) {
	c := newConsultationApp(t)
	rules := `{"reminder_offsets_minutes":[1440],"sms_reminders":false,"reschedule_notice_hours":12,"reschedule_max":1,` +
		`"waitlist_enabled":true,"waitlist_hold_minutes":20,"noshow_flag_threshold":2,"noshow_flag_window_days":90}`
	expectHTTP(t, c.d.admin, http.MethodGet, "/clinic/appointment-settings", http.StatusOK, `"reminder_offsets_minutes":[1440,60]`)
	if rec := c.d.admin.do(http.MethodPut, "/clinic/appointment-settings", rules); rec.Code != http.StatusBadRequest {
		t.Errorf("PUT without If-Match = %d %s, want 400", rec.Code, rec.Body)
	}
	c.d.admin.ifMatch = `W/"0"`
	if rec := c.d.admin.do(http.MethodPut, "/clinic/appointment-settings", rules); rec.Code != http.StatusOK || rec.Header().Get("ETag") != `W/"1"` {
		t.Errorf("PUT appointment settings = %d %s, ETag %q", rec.Code, rec.Body, rec.Header().Get("ETag"))
	}
	c.d.admin.ifMatch = ""
	expectHTTP(t, c.d.admin, http.MethodGet, "/clinic/appointment-settings", http.StatusOK, `"reschedule_notice_hours":12`)
	if rec := c.d.admin.do(http.MethodGet, "/clinic/appointment-settings", ""); rec.Header().Get("ETag") != `W/"1"` {
		t.Errorf("GET ETag = %q, want W/\"1\"", rec.Header().Get("ETag"))
	}
	c.d.admin.ifMatch = `W/"1"`
	if rec := c.d.admin.do(http.MethodPut, "/clinic/appointment-settings", strings.Replace(rules, `"sms_reminders":false`, `"sms_reminders":true`, 1)); rec.Code != http.StatusConflict ||
		!strings.Contains(rec.Body.String(), "sms_unavailable") {
		t.Errorf("PUT with SMS on, the fake engine = %d %s, want 409 sms_unavailable", rec.Code, rec.Body)
	}
	if rec := c.d.practitioner.do(http.MethodGet, "/clinic/appointment-settings", ""); rec.Code != http.StatusForbidden {
		t.Errorf("the practitioner reads the appointment settings = %d, want 403", rec.Code)
	}
}

// Two admins' saves of the settings read at the same version, sent at the same time: exactly one is stored (200) and
// the other is refused with 412 version_conflict; the stored settings are the winner's.
func TestNew_ConcurrentAppointmentSettingsSavesLeaveOneWinner(t *testing.T) {
	c := newConsultationApp(t)
	body := func(hours int) string {
		return fmt.Sprintf(`{"reminder_offsets_minutes":[1440],"sms_reminders":false,"reschedule_notice_hours":%d,"reschedule_max":1,`+
			`"waitlist_enabled":false,"waitlist_hold_minutes":20,"noshow_flag_threshold":0,"noshow_flag_window_days":90}`, hours)
	}
	for round, version := range []string{`W/"0"`, `W/"1"`} {
		recs := make([]*httptest.ResponseRecorder, 2)
		start := make(chan struct{})
		var wg sync.WaitGroup
		for i := range recs {
			admin := *c.d.admin
			admin.ifMatch = version
			wg.Go(func() {
				<-start
				recs[i] = admin.do(http.MethodPut, "/clinic/appointment-settings", body(10*round+i+1))
			})
		}
		close(start)
		wg.Wait()
		var winner string
		codes := map[int]int{}
		for i, rec := range recs {
			codes[rec.Code]++
			switch rec.Code {
			case http.StatusOK:
				winner = fmt.Sprintf(`"reschedule_notice_hours":%d`, 10*round+i+1)
			case http.StatusPreconditionFailed:
				if !strings.Contains(rec.Body.String(), `"version_conflict"`) {
					t.Errorf("412 body %s, want version_conflict", rec.Body)
				}
			}
		}
		if codes[http.StatusOK] != 1 || codes[http.StatusPreconditionFailed] != 1 {
			t.Fatalf("saves at %s = %v, want one 200 and one 412", version, codes)
		}
		expectHTTP(t, c.d.admin, http.MethodGet, "/clinic/appointment-settings", http.StatusOK, winner)
	}
}
