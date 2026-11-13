package api_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"
)

// The AI summary through api.New (SAL-31): for the practitioner of a long conversation only, on the fake, in the
// asked language; the patient is refused, and a short conversation has nothing to summarise.
func TestNew_SummarisesALongChatForThePractitioner(t *testing.T) {
	c := newConsultationApp(t)
	c.book(t, wallNow().Add(-48*time.Hour))
	rec := c.d.patient.do(http.MethodPost, "/conversations", `{"organization_id":"`+inviteClinic+`","practitioner_role_id":"`+c.d.roleID+`"}`)
	var conv struct{ ID string }
	if rec.Code != http.StatusCreated || json.Unmarshal(rec.Body.Bytes(), &conv) != nil {
		t.Fatalf("open = %d %s", rec.Code, rec.Body)
	}
	path := "/conversations/" + conv.ID
	expectHTTP(t, c.d.practitioner, http.MethodPost, path+"/summary", http.StatusBadRequest, `"invalid_request"`)
	for i := range 50 {
		body := fmt.Sprintf(`{"client_message_id":"c%07d-0000-4000-8000-000000000000","body":"Synthetic placeholder"}`, i)
		if rec := c.d.patient.do(http.MethodPost, path+"/messages", body); rec.Code != http.StatusCreated {
			t.Fatalf("send %d = %d %s", i, rec.Code, rec.Body)
		}
	}
	expectHTTP(t, c.d.practitioner, http.MethodPost, path+"/summary", http.StatusOK, `"summary":"[bs] Placeholder summary of 50 messages"`, `"up_to_seq":50`, `"ai_generated":true`)
	expectHTTP(t, c.d.patient, http.MethodPost, path+"/summary", http.StatusForbidden)
}
