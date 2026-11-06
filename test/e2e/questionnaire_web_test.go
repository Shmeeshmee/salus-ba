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

// TestQuestionnaire_TheBuilderPublishesAQuestionnaire runs the web journey (test/e2e/web/questionnaire.spec.ts):
// the practitioner builds a three-item questionnaire in the builder by drag and drop, by keyboard and with "Move
// up", publishes it and finds it in the library. Then its FHIR read holds the three questions in the order the
// moves left them.
func TestQuestionnaire_TheBuilderPublishesAQuestionnaire(t *testing.T) {
	clinic := newBookableClinic(t)
	title := "Placeholder web questionnaire " + suffix(t)

	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "web/node_modules/.bin/playwright", "test", "--config", "web/playwright.config.ts", "questionnaire.spec.ts")
	cmd.Env = append(os.Environ(),
		"E2E_PRACTITIONER_EMAIL="+clinic.practitionerAddr,
		"E2E_PASSWORD="+password,
		"E2E_TOTP_SECRET="+hex.EncodeToString(clinic.totp.secret),
		fmt.Sprintf("E2E_TOTP_LAST_STEP=%d", clinic.totp.last),
		"E2E_QUESTIONNAIRE_TITLE="+title,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("the web journey failed: %v\nPlaywright:\n%s", err, out)
	}

	var found struct {
		Entry []struct {
			Resource struct {
				Title, Status string
				Item          []struct{ Text, Type string }
			}
		}
	}
	clinic.practitioner.must(http.StatusOK, http.MethodGet, "/fhir/R4/Questionnaire?"+url.Values{"status": {"active"}}.Encode(), nil, &found)
	if len(found.Entry) != 1 || found.Entry[0].Resource.Title != title {
		t.Fatalf("the clinic's active questionnaires are %d, want the one published in the web app", len(found.Entry))
	}
	got := fmt.Sprint(found.Entry[0].Resource.Item)
	want := "[{Placeholder question C date} {Placeholder question A boolean} {Placeholder question B string}]"
	if got != want {
		t.Errorf("the published items are %s, want %s", got, want)
	}
}
