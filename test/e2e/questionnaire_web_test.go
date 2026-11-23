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

	runWebJourney(t, clinic, "questionnaire.spec.ts", "E2E_QUESTIONNAIRE_TITLE="+title)

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

// TestQuestionnaire_TheBuilderBranchesTheGynaecologyExample runs the web journey test/e2e/web/branching.spec.ts
// (SAL-43): the example built with "Ask a question if the answer is Yes", both paths and the changed answer in the
// preview, then published. Its FHIR read holds the branch as FHIR R4 enableWhen arrays on the yes or no question,
// and the main question after it with none: where the branches join.
func TestQuestionnaire_TheBuilderBranchesTheGynaecologyExample(t *testing.T) {
	clinic := newBookableClinic(t)
	title := "Placeholder branching questionnaire " + suffix(t)
	runWebJourney(t, clinic, "branching.spec.ts", "E2E_QUESTIONNAIRE_TITLE="+title)

	var found struct {
		Entry []struct {
			Resource struct {
				Title string
				Item  []struct {
					LinkID, Text, Type, EnableBehavior string
					EnableWhen                         []map[string]any
				}
			}
		}
	}
	clinic.practitioner.must(http.StatusOK, http.MethodGet, "/fhir/R4/Questionnaire?"+url.Values{"status": {"active"}}.Encode(), nil, &found)
	if len(found.Entry) != 1 || found.Entry[0].Resource.Title != title || len(found.Entry[0].Resource.Item) != 4 {
		t.Fatalf("the clinic's active questionnaires are %+v, want the one published with four items", found.Entry)
	}
	items := found.Entry[0].Resource.Item
	had := items[0].LinkID
	for _, it := range items[1:3] {
		want := fmt.Sprint([]map[string]any{{"question": had, "operator": "=", "answerBoolean": true}})
		if it.Type != "integer" || fmt.Sprint(it.EnableWhen) != want || it.EnableBehavior != "" {
			t.Errorf("%q is %s shown when %v (%q), want an integer shown when %s", it.Text, it.Type, it.EnableWhen, it.EnableBehavior, want)
		}
	}
	if items[3].EnableWhen != nil {
		t.Errorf("the main question is shown when %v, want always (the join)", items[3].EnableWhen)
	}
}

// runWebJourney runs one spec of test/e2e/web as the clinic's practitioner, with the extra E2E_* variables.
func runWebJourney(t *testing.T, clinic *bookableClinic, spec string, env ...string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "web/node_modules/.bin/playwright", "test", "--config", "web/playwright.config.ts", spec)
	cmd.Env = append(append(os.Environ(),
		"E2E_PRACTITIONER_EMAIL="+clinic.practitionerAddr,
		"E2E_PASSWORD="+password,
		"E2E_TOTP_SECRET="+hex.EncodeToString(clinic.totp.secret),
		fmt.Sprintf("E2E_TOTP_LAST_STEP=%d", clinic.totp.last),
	), env...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("the web journey %s failed: %v\nPlaywright:\n%s", spec, err, out)
	}
}
