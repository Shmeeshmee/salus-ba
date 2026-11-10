//go:build e2e

package e2e_test

import (
	"context"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"slices"
	"testing"
	"time"
)

// reviewEntry is the part of a TermReviewEntry the journey checks.
type reviewEntry struct {
	Translation *struct {
		Review    struct{ Status, ReviewedBy, ReviewedAt, Note string }
		VersionID int
	}
	Explanation *struct {
		Text   string
		Review struct{ Status, ReviewedBy, ReviewedAt, Note string }
	}
	Status string
	Term   struct {
		ID           string
		ReviewStatus string
		Labels       []foundLabels
	}
}

// firstCroatianEntry is the first entry of status in the Croatian review queue, as c.
func firstCroatianEntry(t *testing.T, c *api, status string) reviewEntry {
	t.Helper()
	var page struct{ Entries []reviewEntry }
	c.must(http.StatusOK, http.MethodGet, "/terminology/review?language=hr&limit=1&status="+status, nil, &page)
	if len(page.Entries) != 1 {
		t.Fatalf("the Croatian %s queue has %d entries, want 1", status, len(page.Entries))
	}
	return page.Entries[0]
}

// TestTerminology_AReviewerEditsApprovesAndRequestsChanges: the journey of the review panel (SAL-14,
// docs/terminology/SPEC.md §3 "Review"). A practitioner who is no reviewer reads the queue, where the synthetic term
// (ranked above every seed term) is the first unreviewed Croatian entry, and is refused an approval; a patient is
// refused the queue. The operator makes the practitioner a reviewer with salusctl; in the web app
// (test/e2e/web/term-review.spec.ts), with the keyboard, they edit the Croatian label, approve it, write an
// explanation, send it back with a note and edit the approved labels again. The stored review then has the edited
// labels unreviewed (editing an approved entry returns it to the queue) with the approval's reviewer and date kept,
// the explanation needing changes with the note, the term unreviewed in the cheatsheet, and every write audited.
func TestTerminology_AReviewerEditsApprovesAndRequestsChanges(t *testing.T) {
	practitioner, practitionerAddr, totp := newPractitioner(t)
	term := newSyntheticTerm(t, 1_000_000_000)
	me := practitioner.me()

	first := firstCroatianEntry(t, practitioner, "unreviewed")
	if first.Term.ID != term.id || first.Status != "unreviewed" || first.Translation == nil || first.Explanation != nil {
		t.Fatalf("the first Croatian entry = %+v; want the synthetic term, unreviewed, with its translation and no explanation", first)
	}
	approve := fmt.Sprintf("/terminology/terms/%s/translations/hr/review", term.id)
	header := http.Header{"If-Match": {fmt.Sprintf(`W/"%d"`, first.Translation.VersionID)}}
	practitioner.mustWith(header, http.StatusForbidden, http.MethodPost, approve, map[string]string{"status": "approved"}, nil)

	patient := newAPI(t)
	patientAddr := email(t, "patient")
	patient.must(http.StatusCreated, http.MethodPost, "/auth/register", map[string]string{"email": patientAddr, "password": password, "detectedLanguage": "en"}, nil)
	patient.bearer = patient.login(patientAddr).Token
	if status, code := patient.refusal(http.MethodGet, "/terminology/review?language=hr", nil); status != http.StatusForbidden || code != "forbidden" {
		t.Errorf("a patient's queue = %d %q, want 403 forbidden", status, code)
	}

	salusctl(t, "term-reviewer", "grant", me.ID)
	t.Cleanup(func() {
		cmd := exec.CommandContext(context.Background(), os.Getenv("E2E_SALUSCTL"), "term-reviewer", "revoke", me.ID)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Errorf("revoking the reviewer: %v: %s", err, out)
		}
	})

	edited, explanation := "Čirzynt uređen "+term.suffix, "Placeholder explanation of a synthetic term "+term.suffix+"."
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "web/node_modules/.bin/playwright", "test", "--config", "web/playwright.config.ts", "term-review.spec.ts")
	cmd.Env = append(os.Environ(),
		"E2E_PRACTITIONER_EMAIL="+practitionerAddr,
		"E2E_PASSWORD="+password,
		"E2E_TOTP_SECRET="+hex.EncodeToString(totp.secret),
		fmt.Sprintf("E2E_TOTP_LAST_STEP=%d", totp.last),
		"E2E_TERM_LATIN="+term.latin,
		"E2E_TERM_CROATIAN="+term.croatian,
		"E2E_TERM_EDITED="+edited,
		"E2E_EXPLANATION="+explanation,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("the web journey failed: %v\nPlaywright:\n%s", err, out)
	}

	got := firstCroatianEntry(t, practitioner, "needs_changes") // the explanation needs changes: the most common such entry
	wantLabels := foundLabels{Language: "hr", Preferred: edited, Synonyms: []string{"Sinonim e2e"}}
	if got.Term.ID != term.id || len(got.Term.Labels) == 0 || !slices.Equal(got.Term.Labels[0].Synonyms, wantLabels.Synonyms) ||
		got.Term.Labels[0].Preferred != wantLabels.Preferred || got.Term.Labels[0].Language != "hr" {
		t.Errorf("the entry's labels = %+v, want first %+v", got.Term.Labels, wantLabels)
	}
	if tr := got.Translation; tr == nil || tr.Review.Status != "unreviewed" || tr.Review.ReviewedBy != me.ID || tr.Review.ReviewedAt == "" {
		t.Errorf("the translation = %+v; want unreviewed again, the approval's reviewer and date kept", tr)
	}
	if ex := got.Explanation; ex == nil || ex.Text != explanation || ex.Review.Status != "needs_changes" || ex.Review.Note != "Placeholder note" || ex.Review.ReviewedBy != me.ID {
		t.Errorf("the explanation = %+v; want it needing changes with the note, by the reviewer", ex)
	}
	if got.Status != "needs_changes" || got.Term.ReviewStatus != "unreviewed" {
		t.Errorf("the entry is %s and the term %s; want needs_changes (its explanation) and unreviewed (its labels)", got.Status, got.Term.ReviewStatus)
	}
	if found, _ := search(t, practitioner, term.latin, "la,hr,en"); firstOf(found).ID != term.id || firstOf(found).ReviewStatus != "unreviewed" {
		t.Errorf("the cheatsheet finds %+v; want the synthetic term, still not reviewed", firstOf(found))
	}

	var writes, refused int
	if err := ownerConn(t).QueryRow(t.Context(), `SELECT count(*) FILTER (WHERE outcome = 0), count(*) FILTER (WHERE outcome = 4)
		FROM audit_event WHERE agent_account_id = $1 AND entity_name IN ('term_translation', 'term_explanation')`, me.ID).Scan(&writes, &refused); err != nil {
		t.Fatal(err)
	}
	if writes != 5 || refused != 1 {
		t.Errorf("audited %d review writes and %d refusals; want 5 (edit, approve, write, needs changes, edit) and 1", writes, refused)
	}
}
