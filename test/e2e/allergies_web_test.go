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
	"testing"
	"time"
)

// TestAllergies_ThePractitionerSeesThePatientsAllergyOnTheCallPage (SAL-41, docs/plans/patient-history.md): the
// patient books with the clinic's care consent and adds an active allergy to their profile; the practitioner's FHIR
// search holds it, unconfirmed; the web journey (test/e2e/web/allergies.spec.ts) signs the practitioner in and finds
// the allergy warning with the substance, its reaction and severity on the appointment's call page.
func TestAllergies_ThePractitionerSeesThePatientsAllergyOnTheCallPage(t *testing.T) {
	clinic := newBookableClinic(t)
	p := newPatientWithProfile(t)
	appt := p.bookFirstTime(clinic)

	text, reaction := "Placeholder allergen "+suffix(t), "Placeholder reaction"
	var written struct{ ID, VersionID string }
	p.must(http.StatusCreated, http.MethodPost, "/me/allergies", map[string]string{
		"text": text, "reaction": reaction, "severity": "severe", "clinicalStatus": "active",
	}, &written)
	if written.ID == "" {
		t.Fatal("POST /me/allergies answered no id")
	}

	var found struct {
		Entry []struct {
			Resource struct {
				ID   string
				Code struct{ Text string }
			}
		}
	}
	clinic.practitioner.must(http.StatusOK, http.MethodGet, "/fhir/R4/AllergyIntolerance?"+url.Values{"patient": {p.patientID}}.Encode(), nil, &found)
	if len(found.Entry) != 1 || found.Entry[0].Resource.ID != written.ID || found.Entry[0].Resource.Code.Text != text {
		t.Fatalf("the practitioner's AllergyIntolerance search holds %d entries, want the patient's one", len(found.Entry))
	}

	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "web/node_modules/.bin/playwright", "test", "--config", "web/playwright.config.ts", "allergies.spec.ts")
	cmd.Env = append(os.Environ(),
		"E2E_PRACTITIONER_EMAIL="+clinic.practitionerAddr,
		"E2E_PASSWORD="+password,
		"E2E_TOTP_SECRET="+hex.EncodeToString(clinic.totp.secret),
		fmt.Sprintf("E2E_TOTP_LAST_STEP=%d", clinic.totp.last),
		"E2E_APPOINTMENT_ID="+appt.ID,
		"E2E_ALLERGY_TEXT="+text,
		"E2E_ALLERGY_REACTION="+reaction,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("the web journey failed: %v\nPlaywright:\n%s", err, out)
	}
}
