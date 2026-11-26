//go:build e2e

package e2e_test

import (
	"context"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/pkg/clock"
)

// TestAccessibility_EveryStaffScreenAndDialog (docs/clinic-cms/WAVE7.md RD3): a clinic with a row on every staff screen,
// then the web journey test/e2e/web/a11y.spec.ts once as each of its staff roles (the admin, the practitioner, the
// receptionist). On every screen and dialog of the role, at 1280 and 390 px: the axe engine (WCAG 2.2 A and AA), the
// keyboard (every control reached by Tab in order, its focus visible, the focus back on the opener when a dialog
// closes), touch targets of at least 44 px on a touch screen, and errors announced.
func TestAccessibility_EveryStaffScreenAndDialog(t *testing.T) {
	clinic := newBookableClinic(t)
	atTheClinic(t, clinic)
	clinic.admin.must(http.StatusOK, http.MethodPut, "/clinic/billing-details", map[string]any{
		"legalName": "E2E synthetic clinic d.o.o.", "taxId": "4200000000001", "vatRegistered": false, "billingEmail": "billing@example.test",
	}, nil)
	var exam struct{ ID string }
	clinic.admin.must(http.StatusCreated, http.MethodPost, "/clinic/services", map[string]any{
		"name": "Placeholder examination", "priceMinor": 5000, "modes": []string{"in-person", "video"}, "active": true, "translations": []any{},
	}, &exam)
	var annex struct{ ID string }
	clinic.admin.must(http.StatusCreated, http.MethodPost, "/clinic/locations", map[string]any{
		"name": "Synthetic annex", "addressLine": "Synthetic annex 2", "postalCode": "88000", "active": true,
	}, &annex)
	clinic.admin.must(http.StatusCreated, http.MethodPost, "/clinic/locations/"+annex.ID+"/rooms", map[string]any{"name": "Room 1", "active": true}, nil)
	clinic.admin.must(http.StatusCreated, http.MethodPost, "/clinic/locations/"+annex.ID+"/devices", map[string]any{"name": "Ultrasound", "active": true}, nil)

	deskAddr := email(t, "reception")
	clinic.admin.must(http.StatusCreated, http.MethodPost, "/invites", map[string]string{"email": deskAddr, "role": "reception"}, nil)
	desk, deskTOTP := enrolFromInvite(t, deskAddr)

	// The desk's day: a walk-in who arrived and has a receipt, and a second visit still to come.
	var walkIn, booked struct{ ID string }
	desk.must(http.StatusCreated, http.MethodPost, "/clinic/patients", map[string]any{
		"givenName": "Vesna", "familyName": "Pristupna", "birthDate": "1966-06-06", "phone": "+38761000611", "consentGivenInPerson": true,
	}, &walkIn)
	desk.must(http.StatusCreated, http.MethodPost, "/clinic/patients", map[string]any{
		"givenName": "Tarik", "familyName": "Tastaturov", "birthDate": "1980-01-02", "consentGivenInPerson": true,
	}, &booked)
	var visit struct{ ID string }
	first, start := deskFreeTime(t, desk, clinic)
	desk.must(http.StatusCreated, http.MethodPost, "/clinic/appointments", map[string]string{"patientId": walkIn.ID, "slotId": first, "serviceId": exam.ID}, &visit)
	second, _ := deskFreeTime(t, desk, clinic)
	desk.must(http.StatusCreated, http.MethodPost, "/clinic/appointments", map[string]string{"patientId": booked.ID, "slotId": second, "serviceId": exam.ID}, nil)
	desk.must(http.StatusOK, http.MethodPost, "/clinic/appointments/"+visit.ID+"/arrive", nil, nil)
	desk.must(http.StatusCreated, http.MethodPost, "/clinic/receipts", map[string]any{
		"kind": "receipt", "appointmentId": visit.ID, "paymentMethod": "cash", "lines": []map[string]any{{"serviceId": exam.ID, "quantity": 1}},
	}, nil)
	desk.must(http.StatusCreated, http.MethodPost, "/clinic/patients/"+walkIn.ID+"/data-requests", map[string]string{"kind": "rectification"}, nil)

	// A patient of the app with a visit ahead, its lab result and a draft treatment plan, and a conversation.
	p := newPatientWithProfile(t)
	p.must(http.StatusCreated, http.MethodPost, "/consents", map[string]string{"organizationId": clinic.id}, nil)
	var appVisit struct{ ID string }
	p.must(http.StatusCreated, http.MethodPost, "/appointments", map[string]string{"slotId": freeSlots(t, p.api, clinic.scheduleID)[0]}, &appVisit)
	clinic.practitioner.must(http.StatusCreated, http.MethodPost, "/clinic/appointments/"+appVisit.ID+"/diagnostic-reports", map[string]any{
		"title": "Synthetic blood count", "effectiveDate": clock.System{}.Now().Format(time.DateOnly),
		"results": []map[string]any{{"test": "Synthetic haemoglobin", "value": 140, "unit": "g/L", "referenceRange": "120-160", "interpretation": "N"}},
	}, nil)
	clinic.newPlan(t, p.patientID, exam.ID)
	openChat(t, p.api, clinic.id, publicRole(t, clinic.id))

	env := []string{"E2E_A11Y_VISIT=" + appVisit.ID, "E2E_A11Y_PATIENT=" + p.name, "E2E_A11Y_DESK_PATIENT=Vesna Pristupna"}
	// The desk shows today's visits: after the day's last free time (23:30 in Sarajevo) the visits are tomorrow's, and the
	// rows' own dialogs (move, receipt) are checked on the visit page and the receipts page only.
	sarajevo := sarajevoZone(t)
	if start.In(sarajevo).Format(time.DateOnly) == (clock.System{}).Now().In(sarajevo).Format(time.DateOnly) {
		env = append(env, "E2E_A11Y_DESK_ROWS=yes")
	} else {
		t.Log("the desk's visits are tomorrow's: its rows' dialogs are not checked in this run")
	}
	t.Run("admin", func(t *testing.T) {
		runA11yJourney(t, clinic.adminAddr, clinic.adminTOTP, append(env, "E2E_A11Y_ROLE=admin")...)
	})
	t.Run("practitioner", func(t *testing.T) {
		runA11yJourney(t, clinic.practitionerAddr, clinic.totp, append(env, "E2E_A11Y_ROLE=practitioner")...)
	})
	t.Run("reception", func(t *testing.T) {
		runA11yJourney(t, deskAddr, deskTOTP, append(env, "E2E_A11Y_ROLE=reception")...)
	})
}

// runA11yJourney runs a11y.spec.ts signed in as addr. A role's pass walks every one of its screens twice, so it has
// more time than a journey.
func runA11yJourney(t *testing.T, addr string, totp *authenticator, env ...string) {
	t.Helper()
	clearLoginLimits(t)
	ctx, cancel := context.WithTimeout(t.Context(), 12*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "web/node_modules/.bin/playwright", "test", "--config", "web/playwright.config.ts", "a11y.spec.ts")
	cmd.Env = append(append(os.Environ(),
		"E2E_PRACTITIONER_EMAIL="+addr, "E2E_PASSWORD="+password,
		"E2E_TOTP_SECRET="+hex.EncodeToString(totp.secret), fmt.Sprintf("E2E_TOTP_LAST_STEP=%d", totp.last),
	), env...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("the web journey a11y.spec.ts failed: %v\nPlaywright:\n%s", err, out)
	}
}
