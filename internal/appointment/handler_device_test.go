package appointment_test

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/Shmeeshmee/salus/internal/appointment"
)

// A busy device is 409 resource_busy on the patient's and the desk's routes (U4 of docs/clinic-cms/WAVE6.md, RM1).
func TestHandler_ABusyDeviceIsResourceBusy(t *testing.T) {
	t.Parallel()
	rec := serveAppt(t, &stubAppointments{err: appointment.ErrResourceBusy}, as(me), &bytes.Buffer{}, http.MethodPost, "/appointments", `{"slotId":"`+slotFree+`"}`)
	if rec.Code != http.StatusConflict || rec.Body.String() != `{"error":"resource_busy"}`+"\n" {
		t.Errorf("the patient's booking: %d %s, want 409 resource_busy", rec.Code, rec.Body)
	}
	rec = serveStaff(t, &stubStaff{err: appointment.ErrResourceBusy}, http.MethodPost, "/clinic/appointments", `{"patientId":"x","slotId":"y"}`, nil)
	if rec.Code != http.StatusConflict || rec.Body.String() != `{"error":"resource_busy"}`+"\n" {
		t.Errorf("the desk's booking: %d %s, want 409 resource_busy", rec.Code, rec.Body)
	}
}
