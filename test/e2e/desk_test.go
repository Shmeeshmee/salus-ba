//go:build e2e

package e2e_test

import (
	"context"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/pkg/clock"
)

// TestDesk_AReceptionistRegistersAWalkInBooksChecksInAndIssuesTheReceipt (docs/clinic-cms F1, F2, F3): the admin
// invites a receptionist; at the front desk the receptionist finds today's free in-person times through the clinic's
// own Schedule and Slot searches, registers a walk-in who has no account with the consent given in person, books them
// at once with a service, sees the visit on the desk's day with the service's name, marks the arrival and issues the
// receipt. The receptionist is refused a patient's questionnaire answers, documents, history and chat.
func TestDesk_AReceptionistRegistersAWalkInBooksChecksInAndIssuesTheReceipt(t *testing.T) {
	clinic := newBookableClinic(t)
	atTheClinic(t, clinic) // in-person hours, the clinic's address
	clinic.admin.must(http.StatusOK, http.MethodPut, "/clinic/billing-details", map[string]any{
		"legalName": "E2E synthetic clinic d.o.o.", "taxId": "4200000000001", "vatRegistered": false, "billingEmail": "billing@example.test",
	}, nil)
	var exam struct{ ID string }
	clinic.admin.must(http.StatusCreated, http.MethodPost, "/clinic/services", map[string]any{
		"name": "Placeholder examination", "priceMinor": 5000, "modes": []string{"in-person"}, "active": true, "translations": []any{},
	}, &exam)
	deskAddr := email(t, "reception")
	clinic.admin.must(http.StatusCreated, http.MethodPost, "/invites", map[string]string{"email": deskAddr, "role": "reception"}, nil)
	desk, deskTOTP := enrolFromInvite(t, deskAddr)

	slotID, start := deskFreeTime(t, desk, clinic)
	var walkIn struct{ ID string }
	desk.must(http.StatusCreated, http.MethodPost, "/clinic/patients", map[string]any{
		"givenName": "Vedrana", "familyName": "Walkinova", "birthDate": "1966-06-06", "phone": "+38761000666", "consentGivenInPerson": true,
	}, &walkIn)
	var visit struct{ ID, Status, ServiceID string }
	desk.must(http.StatusCreated, http.MethodPost, "/clinic/appointments", map[string]string{"patientId": walkIn.ID, "slotId": slotID, "serviceId": exam.ID}, &visit)
	if visit.ServiceID != exam.ID || visit.Status != "booked" {
		t.Errorf("the walk-in's booking = %+v, want booked with the examination", visit)
	}

	sarajevo, err := time.LoadLocation("Europe/Sarajevo")
	if err != nil {
		t.Fatal(err)
	}
	var day struct {
		Date   string
		Visits []struct {
			ServiceName string
			Appointment struct{ ID string }
			Patient     struct{ FamilyName string }
		}
	}
	desk.must(http.StatusOK, http.MethodGet, "/clinic/desk/today?date="+start.In(sarajevo).Format(time.DateOnly), nil, &day)
	found := false
	for _, v := range day.Visits {
		found = found || v.Appointment.ID == visit.ID && v.ServiceName == "Placeholder examination" && v.Patient.FamilyName == "Walkinova"
	}
	if !found {
		t.Errorf("the desk's day %+v does not hold the walk-in's visit with its service", day)
	}

	var arrived struct{ Status, ArrivedAt string }
	desk.must(http.StatusOK, http.MethodPost, "/clinic/appointments/"+visit.ID+"/arrive", nil, &arrived)
	if arrived.Status != "arrived" || arrived.ArrivedAt == "" {
		t.Errorf("the arrival = %+v, want arrived with its time", arrived)
	}
	var issued receipt
	desk.must(http.StatusCreated, http.MethodPost, "/clinic/receipts", map[string]any{
		"kind": "receipt", "appointmentId": visit.ID, "paymentMethod": "cash",
		"lines": []map[string]any{{"serviceId": exam.ID, "quantity": 1}},
	}, &issued)
	if issued.TotalMinor != 5000 || issued.PatientID != walkIn.ID {
		t.Errorf("the desk's receipt = %+v, want 50.00 BAM to the walk-in", issued)
	}
	if code := desk.do(http.MethodGet, "/clinic/receipts/"+issued.ID+"/pdf", nil, nil); code != http.StatusOK {
		t.Errorf("GET the receipt's PDF as the receptionist = %d, want 200", code)
	}
	deskListsReceipt(t, desk, visit.ID, issued, start.In(sarajevo).Format(time.DateOnly))
	// In the browser: the row shows the receipt with its PDF, and a second receipt is asked for first.
	runDeskJourney(t, deskAddr, deskTOTP, "desk.spec.ts", "E2E_DESK_PATIENT=Vedrana Walkinova", fmt.Sprintf("E2E_DESK_RECEIPT=%d", issued.Number))

	deskRefusesTheRecord(t, desk, walkIn.ID, visit.ID)
}

// deskListsReceipt: the desk's day of date lists the receipt on the visit, for its row to show and print
// (docs/clinic-cms/walk/wave3, findings 1 and 2).
func deskListsReceipt(t *testing.T, desk *api, visit string, issued receipt, date string) {
	t.Helper()
	var day struct {
		Visits []struct {
			Appointment struct{ ID string }
			Receipts    []struct {
				ID       string
				Number   int64
				Reversed bool
			}
		}
	}
	desk.must(http.StatusOK, http.MethodGet, "/clinic/desk/today?date="+date, nil, &day)
	for _, v := range day.Visits {
		for _, r := range v.Receipts {
			if v.Appointment.ID == visit && r.ID == issued.ID && r.Number == issued.Number && !r.Reversed {
				return
			}
		}
	}
	t.Errorf("the desk's day %+v does not list receipt no. %d on the visit", day, issued.Number)
}

// runDeskJourney runs a spec of test/e2e/web signed in as the receptionist addr (the web journeys' sign-in variables).
func runDeskJourney(t *testing.T, addr string, totp *authenticator, spec string, env ...string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "web/node_modules/.bin/playwright", "test", "--config", "web/playwright.config.ts", spec)
	cmd.Env = append(append(os.Environ(),
		"E2E_PRACTITIONER_EMAIL="+addr, "E2E_PASSWORD="+password,
		"E2E_TOTP_SECRET="+hex.EncodeToString(totp.secret), fmt.Sprintf("E2E_TOTP_LAST_STEP=%d", totp.last),
	), env...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("the web journey %s failed: %v\nPlaywright:\n%s", spec, err, out)
	}
}

// deskFreeTime is a free in-person slot of the clinic's schedule today (or, after the day's last one, the next), as the
// receptionist's own Schedule and Slot searches give it.
func deskFreeTime(t *testing.T, desk *api, clinic *bookableClinic) (string, time.Time) {
	t.Helper()
	var schedules bundle
	desk.must(http.StatusOK, http.MethodGet, "/fhir/R4/Schedule?"+url.Values{"actor": {"PractitionerRole/" + publicRole(t, clinic.id)}}.Encode(), nil, &schedules)
	if !schedules.has("Schedule", clinic.scheduleID) {
		t.Fatalf("the receptionist's Schedule search does not hold the clinic's schedule %s", clinic.scheduleID)
	}
	now := clock.System{}.Now().UTC()
	q := url.Values{
		"schedule": {"Schedule/" + clinic.scheduleID}, "status": {"free"},
		"start": {"ge" + now.Add(10*time.Minute).Format(time.RFC3339), "lt" + now.Add(48*time.Hour).Format(time.RFC3339)},
	}
	var slots struct {
		Entry []struct {
			Resource struct {
				ID, Start string
				Extension []struct{ URL, ValueCode string }
			}
		}
	}
	desk.must(http.StatusOK, http.MethodGet, "/fhir/R4/Slot?"+q.Encode(), nil, &slots)
	for _, e := range slots.Entry {
		for _, x := range e.Resource.Extension {
			if x.URL == visitModeURL && x.ValueCode == "in-person" {
				start, err := time.Parse(time.RFC3339, e.Resource.Start)
				if err != nil {
					t.Fatal(err)
				}
				return e.Resource.ID, start
			}
		}
	}
	t.Fatal("the receptionist's Slot search holds no free in-person slot")
	return "", time.Time{}
}

// deskRefusesTheRecord: the receptionist is refused the patient's questionnaire answers, documents, history and chat
// (W2): 403 or 404, or an empty search.
func deskRefusesTheRecord(t *testing.T, desk *api, patientID, visitID string) {
	t.Helper()
	for _, path := range []string{
		"/fhir/R4/QuestionnaireResponse?patient=Patient/" + patientID,
		"/fhir/R4/QuestionnaireResponse?appointment=Appointment/" + visitID,
		"/fhir/R4/Encounter?appointment=Appointment/" + visitID,
		"/fhir/R4/Patient/" + patientID,
		"/patients/" + patientID + "/history",
		"/patients/" + patientID + "/history/summary",
		"/conversations",
	} {
		code, raw := desk.send(nil, http.MethodGet, path, nil)
		body := string(raw)
		empty := code == http.StatusOK && strings.Contains(body, `"resourceType":"Bundle"`) && !strings.Contains(body, `"entry"`)
		if code != http.StatusForbidden && code != http.StatusNotFound && !empty {
			t.Errorf("GET %s as the receptionist = %d %s, want a refusal", path, code, body)
		}
		if strings.Contains(body, "Walkinova") || strings.Contains(body, "1966-06-06") {
			t.Errorf("GET %s as the receptionist: the body holds the patient's data", path)
		}
	}
}
