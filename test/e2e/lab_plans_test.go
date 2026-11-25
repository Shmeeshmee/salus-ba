//go:build e2e

package e2e_test

import (
	"bytes"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/pkg/clock"
)

// syntheticPDF is a minimal PDF of synthetic content: what the file type check and the scan need to see.
const syntheticPDF = "%PDF-1.4\n1 0 obj << /Type /Catalog >> endobj\ntrailer << /Root 1 0 R >>\n%%EOF\n"

// TestLabResults_ReleasedToThePatientAndNeverToTheDesk (docs/clinic-cms/WAVE6.md U5): the visit's practitioner attaches
// a result with typed-in values and the lab's PDF; before release the patient sees nothing; once released they read
// the values and the PDF; the clinic's receptionist is refused the result and its PDF.
func TestLabResults_ReleasedToThePatientAndNeverToTheDesk(t *testing.T) {
	clinic := newBookableClinic(t)
	p := newPatientWithProfile(t)
	p.must(http.StatusCreated, http.MethodPost, "/consents", map[string]string{"organizationId": clinic.id}, nil)
	var visit struct{ ID string }
	p.must(http.StatusCreated, http.MethodPost, "/appointments", map[string]string{"slotId": freeSlots(t, p.api, clinic.scheduleID)[0]}, &visit)
	deskAddr := email(t, "reception")
	clinic.admin.must(http.StatusCreated, http.MethodPost, "/invites", map[string]string{"email": deskAddr, "role": "reception"}, nil)
	desk, _ := enrolFromInvite(t, deskAddr)

	var report struct {
		ID, Status string
		VersionID  int
	}
	clinic.practitioner.must(http.StatusCreated, http.MethodPost, "/clinic/appointments/"+visit.ID+"/diagnostic-reports", map[string]any{
		"title": "Synthetic blood count", "effectiveDate": clock.System{}.Now().Format(time.DateOnly),
		"results": []map[string]any{{"test": "Synthetic haemoglobin", "value": 140, "unit": "g/L", "referenceRange": "120-160", "interpretation": "N"}},
	}, &report)
	if code, body := clinic.practitioner.rawPut("/clinic/diagnostic-reports/"+report.ID+"/pdf", report.VersionID, "application/pdf", []byte(syntheticPDF)); code != http.StatusOK {
		t.Fatalf("PUT the lab's PDF = %d %s, want 200", code, body)
	}
	var mine []struct{ ID string }
	p.must(http.StatusOK, http.MethodGet, "/me/diagnostic-reports", nil, &mine)
	if len(mine) != 0 {
		t.Errorf("before release the patient sees %+v, want nothing", mine)
	}
	if code := p.do(http.MethodGet, "/me/diagnostic-reports/"+report.ID+"/pdf", nil, nil); code != http.StatusNotFound {
		t.Errorf("the patient's PDF before release = %d, want 404", code)
	}

	clinic.practitioner.must(http.StatusOK, http.MethodGet, "/clinic/diagnostic-reports/"+report.ID, nil, &report)
	clinic.practitioner.mustWith(http.Header{"If-Match": {`W/"` + strconv.Itoa(report.VersionID) + `"`}}, http.StatusOK, http.MethodPost,
		"/clinic/diagnostic-reports/"+report.ID+"/release", nil, &report)
	if report.Status != "final" {
		t.Errorf("released: status %q, want final", report.Status)
	}
	var read struct {
		Status  string
		Results []struct {
			Test  string
			Value float64
		}
	}
	p.must(http.StatusOK, http.MethodGet, "/me/diagnostic-reports/"+report.ID, nil, &read)
	if read.Status != "final" || len(read.Results) != 1 || read.Results[0].Test != "Synthetic haemoglobin" || read.Results[0].Value != 140 {
		t.Errorf("the patient reads %+v, want the released value", read)
	}
	eventually(t, "the scan clears the lab's PDF and the patient opens it", func() bool {
		code, body := p.rawGet("/me/diagnostic-reports/" + report.ID + "/pdf")
		return code == http.StatusOK && bytes.HasPrefix(body, []byte("%PDF-"))
	})

	for _, path := range []string{"/clinic/diagnostic-reports/" + report.ID, "/clinic/diagnostic-reports/" + report.ID + "/pdf", "/clinic/appointments/" + visit.ID + "/diagnostic-reports"} {
		if code := desk.do(http.MethodGet, path, nil, nil); code != http.StatusNotFound && code != http.StatusForbidden {
			t.Errorf("the receptionist GET %s = %d, want a refusal", path, code)
		}
	}
}

// TestCarePlans_BookedWholeMovedAndCancelled (docs/clinic-cms/WAVE6.md U6): the admin lays out a three-step plan and
// books it in one go; one step is moved and the plan shows it; cancelling the plan cancels its future visits. A plan
// whose second step's time is taken meanwhile books nothing.
func TestCarePlans_BookedWholeMovedAndCancelled(t *testing.T) {
	clinic := newBookableClinic(t)
	var service struct{ ID string }
	clinic.admin.must(http.StatusCreated, http.MethodPost, "/clinic/services", map[string]any{
		"name": "Synthetic physiotherapy session", "priceMinor": 4000, "modes": []string{"video"}, "active": true, "translations": []any{},
	}, &service)
	p := newPatientWithProfile(t)
	p.must(http.StatusCreated, http.MethodPost, "/consents", map[string]string{"organizationId": clinic.id}, nil)
	slots := freeSlots(t, clinic.admin, clinic.scheduleID)
	if len(slots) < 8 {
		t.Fatalf("the schedule has %d free slots, want 8", len(slots))
	}

	plan := clinic.newPlan(t, p.patientID, service.ID)
	clinic.book(t, plan, http.StatusOK, slots[0], slots[2], slots[4])
	got := clinic.readPlan(t, plan.ID)
	if got.Status != "active" || len(got.Steps) != 3 || got.Steps[0].AppointmentID == "" || got.Steps[2].AppointmentID == "" {
		t.Fatalf("the booked plan %+v, want active with three visits", got)
	}
	moved := got.Steps[0].AppointmentID
	var current struct{ Meta struct{ VersionID string } }
	clinic.admin.must(http.StatusOK, http.MethodGet, "/fhir/R4/Appointment/"+moved, nil, &current)
	clinic.admin.mustWith(http.Header{"If-Match": {`W/"` + current.Meta.VersionID + `"`}}, http.StatusOK, http.MethodPost,
		"/clinic/appointments/"+moved+"/reschedule", map[string]string{"slotId": slots[6]}, nil)
	got = clinic.readPlan(t, plan.ID)
	if last := got.Steps[2]; last.AppointmentID != moved || last.VisitStart == "" {
		t.Errorf("after the move the steps are %+v, want the moved visit last, at its new time", got.Steps)
	}
	clinic.admin.mustWith(http.Header{"If-Match": {`W/"` + strconv.Itoa(got.VersionID) + `"`}}, http.StatusOK, http.MethodPost, "/clinic/care-plans/"+plan.ID+"/cancel", nil, nil)
	got = clinic.readPlan(t, plan.ID)
	for _, s := range got.Steps {
		var a struct{ Status string }
		p.must(http.StatusOK, http.MethodGet, "/fhir/R4/Appointment/"+s.AppointmentID, nil, &a)
		if a.Status != "cancelled" {
			t.Errorf("after the plan's cancellation visit %s is %q, want cancelled", s.AppointmentID, a.Status)
		}
	}
	if got.Status != "revoked" {
		t.Errorf("the cancelled plan's status %q, want revoked", got.Status)
	}

	// The second step's time is taken by another patient: the plan books nothing.
	other := newPatientWithProfile(t)
	other.must(http.StatusCreated, http.MethodPost, "/consents", map[string]string{"organizationId": clinic.id}, nil)
	other.must(http.StatusCreated, http.MethodPost, "/appointments", map[string]string{"slotId": slots[3]}, nil)
	second := clinic.newPlan(t, p.patientID, service.ID)
	clinic.book(t, second, http.StatusConflict, slots[1], slots[3], slots[5])
	if got := clinic.readPlan(t, second.ID); got.Status != "draft" || got.Steps[0].AppointmentID != "" {
		t.Errorf("the refused plan %+v, want a draft without visits", got)
	}
	free := freeSlots(t, clinic.admin, clinic.scheduleID)
	for _, s := range []string{slots[1], slots[5]} {
		if !strings.Contains(strings.Join(free, " "), s) {
			t.Errorf("slot %s is not free after the refused plan: its booking was not rolled back", s)
		}
	}
}

type carePlan struct {
	ID, Status string
	Steps      []struct{ AppointmentID, VisitStart string }
	VersionID  int
}

func (c *bookableClinic) newPlan(t *testing.T, patientID, service string) carePlan {
	t.Helper()
	steps := make([]map[string]any, 0, 3)
	for day := range 3 {
		steps = append(steps, map[string]any{"serviceId": service, "dayOffset": day, "visitMode": "video"})
	}
	var plan carePlan
	c.admin.must(http.StatusCreated, http.MethodPost, "/clinic/care-plans", map[string]any{"patientId": patientID, "title": "Synthetic plan", "steps": steps}, &plan)
	return plan
}

func (c *bookableClinic) book(t *testing.T, plan carePlan, want int, slots ...string) {
	t.Helper()
	code, body := c.admin.send(http.Header{"If-Match": {`W/"` + strconv.Itoa(plan.VersionID) + `"`}}, http.MethodPost, "/clinic/care-plans/"+plan.ID+"/book",
		map[string]any{"startsOn": clock.System{}.Now().Format(time.DateOnly), "slotIds": slots})
	if code != want {
		t.Fatalf("POST book = %d %s, want %d", code, body, want)
	}
}

func (c *bookableClinic) readPlan(t *testing.T, id string) carePlan {
	t.Helper()
	var plan carePlan
	c.admin.must(http.StatusOK, http.MethodGet, "/clinic/care-plans/"+id, nil, &plan)
	return plan
}

// rawPut sends data as the body with its content type at the version, as the lab result's PDF upload does.
func (a *api) rawPut(path string, version int, contentType string, data []byte) (int, []byte) {
	a.t.Helper()
	req, err := http.NewRequestWithContext(a.t.Context(), http.MethodPut, a.base+path, bytes.NewReader(data))
	if err != nil {
		a.t.Fatal(err)
	}
	req.ContentLength = int64(len(data))
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("If-Match", `W/"`+strconv.Itoa(version)+`"`)
	return a.raw(req)
}

// rawGet reads path's bytes, whatever their type.
func (a *api) rawGet(path string) (int, []byte) {
	a.t.Helper()
	req, err := http.NewRequestWithContext(a.t.Context(), http.MethodGet, a.base+path, nil)
	if err != nil {
		a.t.Fatal(err)
	}
	return a.raw(req)
}

func (a *api) raw(req *http.Request) (int, []byte) {
	a.t.Helper()
	if a.bearer != "" {
		req.Header.Set("Authorization", "Bearer "+a.bearer)
	}
	resp, err := a.http.Do(req)
	if err != nil {
		a.t.Fatalf("%s %s: %v", req.Method, req.URL.Path, err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		a.t.Fatal(err)
	}
	return resp.StatusCode, body
}
