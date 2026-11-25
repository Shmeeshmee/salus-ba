package schedule_test

import (
	"bytes"
	"net/http"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/schedule"
)

func TestHTTPHandler_TakesAndGivesARulesRoom(t *testing.T) {
	t.Parallel()
	out := savedA1
	out.Rules = []schedule.Rule{{Day: 1, Start: "09:00", End: "12:00", Mode: schedule.ModeInPerson, RoomID: room1}}
	svc := &stubAvailability{out: out}
	body := `{"slotMinutes":30,"availability":[{"day":1,"start":"09:00","end":"12:00","mode":"in-person","roomId":"` + room1 + `"}]}`
	rec := serve(t, svc, true, &bytes.Buffer{}, http.MethodPut, `W/"1"`, body)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"roomId":"`+room1+`"`) || len(svc.in.Rules) != 1 || svc.in.Rules[0].RoomID != room1 {
		t.Fatalf("PUT = %d %s, asked %+v; want the room both ways", rec.Code, rec.Body, svc.in)
	}
	if rec := serve(t, &stubAvailability{err: schedule.ErrResourceBusy}, true, &bytes.Buffer{}, http.MethodPut, `W/"1"`, body); rec.Code != http.StatusConflict ||
		!strings.Contains(rec.Body.String(), `"resource_busy"`) {
		t.Errorf("a busy room: %d %s, want 409 resource_busy", rec.Code, rec.Body)
	}
	rec = serve(t, &stubAvailability{out: savedA1}, true, &bytes.Buffer{}, http.MethodPut, `W/"1"`, mondays)
	if strings.Contains(rec.Body.String(), "roomId") {
		t.Errorf("a rule without a room: %s, want no roomId", rec.Body)
	}
}
