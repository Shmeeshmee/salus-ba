//go:build e2e

package e2e_test

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"testing"
	"time"
)

// TestDesk_AnImportedPatientGivesTheConsentAtTheDeskAndIsBooked (docs/clinic-cms wave 4, V5): the admin imports a
// patient from a file; the receptionist finds them at the desk, without a consent; a booking is refused until the
// receptionist records the consent the patient gives in person, which another clinic cannot do; then the visit is
// booked, and the desk's day says no reminder will go (the clinic sends no SMS).
func TestDesk_AnImportedPatientGivesTheConsentAtTheDeskAndIsBooked(t *testing.T) {
	clinic := newBookableClinic(t)
	atTheClinic(t, clinic)
	deskAddr := email(t, "reception")
	clinic.admin.must(http.StatusCreated, http.MethodPost, "/invites", map[string]string{"email": deskAddr, "role": "reception"}, nil)
	desk, _ := enrolFromInvite(t, deskAddr)

	file := "given name,family name,birth date,phone\nIda,Importovana,05.05.1975,061 000 004\n"
	var imported struct {
		Rows  []struct{ Errors []string }
		Saved bool
	}
	if code := clinic.admin.upload("/clinic/patients/import?save=true", "patients.csv", []byte(file), &imported); code != http.StatusOK || !imported.Saved || len(imported.Rows) != 1 {
		t.Fatalf("the import = %d %+v, want one row saved", code, imported)
	}
	if code := desk.upload("/clinic/patients/import?save=true", "patients.csv", []byte(file), nil); code != http.StatusForbidden {
		t.Errorf("the receptionist imports: %d, want 403", code)
	}

	var found []struct {
		ID                                string
		Phone                             string
		RegisteredByClinic, ConsentActive bool
	}
	desk.must(http.StatusOK, http.MethodGet, "/clinic/patients?name=Importovana", nil, &found)
	if len(found) != 1 || !found[0].RegisteredByClinic || found[0].ConsentActive || found[0].Phone != "+38761000004" {
		t.Fatalf("the desk finds %+v, want the imported patient without a consent", found)
	}
	ida := found[0].ID
	slotID, start := deskFreeTime(t, desk, clinic)
	if code, why := desk.refusal(http.MethodPost, "/clinic/appointments", map[string]string{"patientId": ida, "slotId": slotID}); why != "consent_required" {
		t.Errorf("a booking without the consent = %d %s, want consent_required", code, why)
	}

	other := newBookableClinic(t)
	if code := other.admin.do(http.MethodPost, "/clinic/patients/"+ida+"/consent", nil, nil); code != http.StatusNotFound {
		t.Errorf("another clinic records the consent: %d, want 404", code)
	}
	desk.must(http.StatusNoContent, http.MethodPost, "/clinic/patients/"+ida+"/consent", nil, nil)
	if code, why := desk.refusal(http.MethodPost, "/clinic/patients/"+ida+"/consent", nil); code != http.StatusConflict || why != "consent_active" {
		t.Errorf("a second consent = %d %s, want 409 consent_active", code, why)
	}
	desk.must(http.StatusOK, http.MethodGet, "/clinic/patients?name=Importovana", nil, &found)
	if len(found) != 1 || !found[0].ConsentActive {
		t.Errorf("after the consent the desk finds %+v, want it active", found)
	}

	var visit struct{ ID string }
	desk.must(http.StatusCreated, http.MethodPost, "/clinic/appointments", map[string]string{"patientId": ida, "slotId": slotID}, &visit)
	sarajevo := sarajevoZone(t)
	var day struct {
		Visits []struct {
			Appointment struct{ ID string }
			ReminderOff string
			Patient     struct{ ConsentActive bool }
			Reminder    bool
		}
	}
	desk.must(http.StatusOK, http.MethodGet, "/clinic/desk/today?date="+start.In(sarajevo).Format(time.DateOnly), nil, &day)
	seen := false
	for _, v := range day.Visits {
		if v.Appointment.ID == visit.ID {
			seen = true
			if v.Reminder || v.ReminderOff != "sms_off" || !v.Patient.ConsentActive {
				t.Errorf("the desk's row = %+v, want no reminder (sms_off) and the consent active", v)
			}
		}
	}
	if !seen {
		t.Errorf("the visit %s is not on the desk's day", visit.ID)
	}
}

// upload posts data as the multipart part "file" named name, as the import screens do, and decodes a JSON answer
// into out when out is not nil. It returns the status.
func (a *api) upload(path, name string, data []byte, out any) int {
	a.t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	part, err := mw.CreateFormFile("file", name)
	if err == nil {
		_, err = part.Write(data)
	}
	if err == nil {
		err = mw.Close()
	}
	if err != nil {
		a.t.Fatal(err)
	}
	req, err := http.NewRequestWithContext(a.t.Context(), http.MethodPost, a.base+path, &body)
	if err != nil {
		a.t.Fatal(err)
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+a.bearer)
	resp, err := a.http.Do(req)
	if err != nil {
		a.t.Fatalf("POST %s: %v", path, err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		a.t.Fatal(err)
	}
	if out != nil && resp.StatusCode < 300 {
		if err := json.Unmarshal(raw, out); err != nil {
			a.t.Fatalf("POST %s = %d: decoding %s: %v", path, resp.StatusCode, raw, err)
		}
	}
	return resp.StatusCode
}

// sarajevoZone is the test clinics' time zone.
func sarajevoZone(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("Europe/Sarajevo")
	if err != nil {
		t.Fatal(err)
	}
	return loc
}
