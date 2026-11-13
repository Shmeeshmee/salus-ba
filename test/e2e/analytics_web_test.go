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

// TestAnalytics_ThePractitionerSeesTheBookingOnTheirClinicsChart (SAL-22, docs/plans/analytics.md §9 ANA-4): a
// patient books with the clinic; the clinic's counts hold the booking (refreshed within seconds of the change); the
// web journey (test/e2e/web/analytics.spec.ts) signs the practitioner in, opens Analytics, finds the booking on
// My clinic's appointments chart from the generated client, with no specialty or practitioner filter (the API refuses
// them to a practitioner who is not an admin), and the All of Salus tab (on in make e2e's stack) without an error.
func TestAnalytics_ThePractitionerSeesTheBookingOnTheirClinicsChart(t *testing.T) {
	clinic := newBookableClinic(t)
	p := newPatientWithProfile(t)
	appt := p.bookFirstTime(clinic)
	start, err := time.Parse(time.RFC3339, appt.Start)
	if err != nil {
		t.Fatalf("the appointment's start %q: %v", appt.Start, err)
	}

	month := ""
	eventually(t, "the clinic's counts hold the booking", func() bool {
		var stats struct {
			Series []struct {
				Period string
				Booked int
			}
		}
		q := url.Values{"from": {start.AddDate(0, 0, -2).Format(time.DateOnly)}, "to": {start.AddDate(0, 0, 2).Format(time.DateOnly)}}
		clinic.practitioner.must(http.StatusOK, http.MethodGet, "/analytics/clinic/appointments?"+q.Encode(), nil, &stats)
		for _, s := range stats.Series {
			if s.Booked == 1 {
				month = s.Period
			}
		}
		return month != ""
	})

	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "web/node_modules/.bin/playwright", "test", "--config", "web/playwright.config.ts", "analytics.spec.ts")
	cmd.Env = append(os.Environ(),
		"E2E_PRACTITIONER_EMAIL="+clinic.practitionerAddr,
		"E2E_PASSWORD="+password,
		"E2E_TOTP_SECRET="+hex.EncodeToString(clinic.totp.secret),
		fmt.Sprintf("E2E_TOTP_LAST_STEP=%d", clinic.totp.last),
		"E2E_BOOKED_MONTH="+month,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("the web journey failed: %v\nPlaywright:\n%s", err, out)
	}
}
