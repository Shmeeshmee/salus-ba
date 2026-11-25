package chargeitemdefinition_test

import (
	"net/http"
	"strings"
	"testing"
)

// A service's device goes in and out as deviceId, absent for none (U4 of docs/clinic-cms/WAVE6.md, RM1).
func TestHandler_AServicesDevice(t *testing.T) {
	t.Parallel()
	const device = "a8000000-0000-4000-8000-0000000000a1"
	svc := &stubService{}
	w := serve(t, svc, false, http.MethodPost, "/clinic/services", strings.Replace(minimalBody, `"active":false`, `"active":false,"deviceId":"`+device+`"`, 1))
	if w.Code != http.StatusCreated || svc.saved.DeviceID != device || !strings.Contains(w.Body.String(), `"deviceId":"`+device+`"`) {
		t.Fatalf("POST = %d %s, saved %+v; want the device both ways", w.Code, w.Body, svc.saved)
	}
	w = serve(t, svc, false, http.MethodPost, "/clinic/services", minimalBody)
	if w.Code != http.StatusCreated || svc.saved.DeviceID != "" || strings.Contains(w.Body.String(), "deviceId") {
		t.Errorf("POST without a device = %d %s, saved %q; want none", w.Code, w.Body, svc.saved.DeviceID)
	}
}
