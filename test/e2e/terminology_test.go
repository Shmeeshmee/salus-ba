//go:build e2e

package e2e_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

// The journeys of docs/terminology (IMPLEMENTATION_PLAN.md 3.1): a practitioner's search, in the API and in the web
// app's cheatsheet, a patient refused, and the curator's spreadsheet round trip with salusctl. Each journey adds its
// own synthetic term with salusctl's seed import and deletes it at the end, so the terms of a loaded seed (the dev
// database's) are never changed and a stack without the seed (CI) runs them the same.

// TestTerminology_APractitionerFindsATermAndAPatientIsRefused: the practitioner finds the synthetic term by its Latin
// label with its Croatian and English labels, by its Croatian label typed without the háček, and in the web app's
// cheatsheet (test/e2e/web/terminology.spec.ts); a patient is refused with 403 and nobody signed in with 401.
func TestTerminology_APractitionerFindsATermAndAPatientIsRefused(t *testing.T) {
	practitioner, practitionerAddr, totp := newPractitioner(t)
	term := newSyntheticTerm(t)

	got, header := search(t, practitioner, term.latin, "la,hr,en")
	if header.Get("Cache-Control") != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store (the URL holds the search text)", header.Get("Cache-Control"))
	}
	want := foundTerm{ID: term.id, Category: "symptom", ReviewStatus: "draft", Labels: []foundLabels{
		{Language: "la", Preferred: term.latin, Synonyms: []string{}},
		{Language: "hr", Preferred: term.croatian, Synonyms: []string{}},
		{Language: "en", Preferred: term.english, Synonyms: []string{}},
	}}
	if first := firstOf(got); !first.equal(want) {
		t.Errorf("searching the Latin label: first term %+v, want %+v", first, want)
	}
	if accentless, _ := search(t, practitioner, strings.Replace(term.croatian, "Č", "c", 1), "la,hr,en"); firstOf(accentless).ID != term.id {
		t.Errorf("searching the Croatian label without its háček: first term %q, want %q", firstOf(accentless).ID, term.id)
	}

	patient := newAPI(t)
	patientAddr := email(t, "patient")
	patient.must(http.StatusCreated, http.MethodPost, "/auth/register", map[string]string{"email": patientAddr, "password": password, "detectedLanguage": "en"}, nil)
	patient.bearer = patient.login(patientAddr).Token
	path := "/terminology/search?" + url.Values{"q": {term.latin}}.Encode()
	if status, code := patient.refusal(http.MethodGet, path, nil); status != http.StatusForbidden || code != "forbidden" {
		t.Errorf("a patient's search = %d %q, want 403 forbidden", status, code)
	}
	if status := newAPI(t).do(http.MethodGet, path, nil, nil); status != http.StatusUnauthorized {
		t.Errorf("a search with nobody signed in = %d, want 401", status)
	}

	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "web/node_modules/.bin/playwright", "test", "--config", "web/playwright.config.ts", "terminology.spec.ts")
	cmd.Env = append(os.Environ(),
		"E2E_PRACTITIONER_EMAIL="+practitionerAddr,
		"E2E_PASSWORD="+password,
		"E2E_TOTP_SECRET="+hex.EncodeToString(totp.secret),
		fmt.Sprintf("E2E_TOTP_LAST_STEP=%d", totp.last),
		"E2E_TERM_LATIN="+term.latin,
		"E2E_TERM_ENGLISH="+term.english,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("the web journey failed: %v\nPlaywright:\n%s", err, out)
	}
}

// TestTerminology_TheCuratorsRoundTrip: the drafts' export holds the synthetic term; the curator corrects its Latin
// label, writes a note and sets it reviewed; the import applies that row and refuses a row whose id does not exist
// (exiting non-zero after applying the rest); the search shows the new label and no draft mark; the reviewed export
// holds it; importing the same row again changes nothing.
func TestTerminology_TheCuratorsRoundTrip(t *testing.T) {
	practitioner, _, _ := newPractitioner(t)
	term := newSyntheticTerm(t)
	dir := t.TempDir()

	header, row := exportedRow(t, filepath.Join(dir, "drafts.csv"), "draft", term.id)
	col := func(name string) int {
		i := slices.Index(header, name)
		if i < 0 {
			t.Fatalf("the export has no %q column: %v", name, header)
		}
		return i
	}
	if row[col("la")] != term.latin || row[col("hr")] != term.croatian || row[col("review_status")] != "draft" {
		t.Fatalf("the exported row: la %q, hr %q, review_status %q; want the synthetic term as a draft", row[col("la")], row[col("hr")], row[col("review_status")])
	}

	corrected := "Zyntheticum correctum " + term.suffix
	row[col("la")], row[col("review_status")], row[col("note")] = corrected, "reviewed", "Placeholder curator note"
	unknown := slices.Clone(row)
	unknown[col("id")] = "00000000-0000-4000-8000-" + term.suffix + "0000"
	edited := filepath.Join(dir, "edited.csv")
	writeCSV(t, edited, header, row, unknown)
	out, err := salusctlResult(t, "terminology", "import", edited)
	if err == nil || !strings.HasPrefix(out, "created 0, updated 1, unchanged 0, refused 1\n") || !strings.Contains(out, "(id "+unknown[col("id")]+")") {
		t.Fatalf("importing the edited row and an unknown id: %v, %q; want updated 1, refused 1 naming the unknown id, and a non-zero exit", err, out)
	}

	found, _ := search(t, practitioner, corrected, "la,en")
	got := firstOf(found)
	if got.ID != term.id || got.ReviewStatus != "reviewed" || len(got.Labels) == 0 || got.Labels[0].Preferred != corrected {
		t.Errorf("searching the corrected label: %+v, want the term reviewed with the corrected Latin label", got)
	}
	if _, reviewed := exportedRow(t, filepath.Join(dir, "reviewed.csv"), "reviewed", term.id); reviewed[col("note")] != "Placeholder curator note" {
		t.Errorf("the reviewed export's note = %q, want the curator's", reviewed[col("note")])
	}

	again := filepath.Join(dir, "again.csv")
	writeCSV(t, again, header, row)
	if out := salusctl(t, "terminology", "import", again); !strings.HasPrefix(out, "created 0, updated 0, unchanged 1, refused 0\n") {
		t.Errorf("importing the same row again: %q, want unchanged 1", out)
	}
}

// syntheticTerm is a symptom added with salusctl's seed import under a Wikidata id far beyond Wikidata's own, with
// made-up labels that match no seed term; the test's cleanup deletes it (its labels go with it).
type syntheticTerm struct {
	id, suffix, latin, croatian, english string
}

func newSyntheticTerm(t *testing.T) syntheticTerm {
	t.Helper()
	n, err := rand.Int(rand.Reader, big.NewInt(1e15))
	if err != nil {
		t.Fatal(err)
	}
	s := suffix(t)
	term := syntheticTerm{suffix: s, latin: "Zyntheticum " + s, croatian: "Čirzynt " + s, english: "E2E synthetic term " + s}
	wikidata := fmt.Sprintf("Q9%015d", n)
	seed := filepath.Join(t.TempDir(), "seed.jsonl")
	line := fmt.Sprintf(`{"labels":{"la":[%q],"hr":[%q],"en":[%q]},"wikidata":%q,"category":"symptom","rank":0}`+"\n",
		term.latin, term.croatian, term.english, wikidata)
	if err := os.WriteFile(seed, []byte(line), 0o600); err != nil {
		t.Fatal(err)
	}
	if out := salusctl(t, "terminology", "import", seed); !strings.HasPrefix(out, "created 1, updated 0, unchanged 0, refused 0\n") {
		t.Fatalf("salusctl terminology import of the synthetic term: %q", out)
	}
	if err := ownerConn(t).QueryRow(t.Context(), "SELECT id FROM term WHERE wikidata_id = $1", wikidata).Scan(&term.id); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		conn, err := pgx.Connect(context.Background(), env(t, "DATABASE_URL"))
		if err != nil {
			t.Errorf("deleting the synthetic term: %v", err)
			return
		}
		defer func() { _ = conn.Close(context.Background()) }()
		if _, err := conn.Exec(context.Background(), "DELETE FROM term WHERE id = $1", term.id); err != nil {
			t.Errorf("deleting the synthetic term: %v", err)
		}
	})
	return term
}

// newPractitioner makes a clinic with salusctl whose admin invites a practitioner; it returns the practitioner on
// the session the enrolment opens, the address and the authenticator (for the web app's sign-in).
func newPractitioner(t *testing.T) (*api, string, *authenticator) {
	t.Helper()
	waitReady(t)
	adminAddr, practitionerAddr := email(t, "admin"), email(t, "practitioner")
	salusctl(t, "create-clinic", "-name", "E2E synthetic clinic "+suffix(t), "-admin-email", adminAddr)
	admin, _ := enrolFromInvite(t, adminAddr)
	admin.must(http.StatusCreated, http.MethodPost, "/invites", map[string]string{
		"email": practitionerAddr, "role": "practitioner", "givenName": "Te", "familyName": "Syntheticum",
	}, nil)
	practitioner, totp := enrolFromInvite(t, practitionerAddr)
	return practitioner, practitionerAddr, totp
}

type foundLabels struct {
	Language, Preferred string
	Synonyms            []string
}

type foundTerm struct {
	ID, Category, ReviewStatus string
	Labels                     []foundLabels
}

func (f foundTerm) equal(o foundTerm) bool {
	return f.ID == o.ID && f.Category == o.Category && f.ReviewStatus == o.ReviewStatus &&
		slices.EqualFunc(f.Labels, o.Labels, func(a, b foundLabels) bool {
			return a.Language == b.Language && a.Preferred == b.Preferred && slices.Equal(a.Synonyms, b.Synonyms)
		})
}

// search is GET /terminology/search for text in languages, as c; it fails the test on any status but 200.
func search(t *testing.T, c *api, text, languages string) ([]foundTerm, http.Header) {
	t.Helper()
	var body struct{ Terms []foundTerm }
	path := "/terminology/search?" + url.Values{"q": {text}, "languages": {languages}}.Encode()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, c.base+path, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+c.bearer)
	resp, err := c.http.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /terminology/search = %d, want 200", resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	return body.Terms, resp.Header
}

func firstOf(terms []foundTerm) foundTerm {
	if len(terms) == 0 {
		return foundTerm{}
	}
	return terms[0]
}

// exportedRow runs salusctl's export of the terms with status into path and returns its header and the row of id.
func exportedRow(t *testing.T, path, status, id string) (header, row []string) {
	t.Helper()
	salusctl(t, "terminology", "export", "-o", path, "-status", status)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	records, err := csv.NewReader(bytes.NewReader(bytes.TrimPrefix(raw, []byte("\xef\xbb\xbf")))).ReadAll()
	if err != nil || len(records) == 0 {
		t.Fatalf("reading the export: %d records, %v", len(records), err)
	}
	for _, r := range records[1:] {
		if r[0] == id {
			return records[0], r
		}
	}
	t.Fatalf("the %s export has no row for the synthetic term", status)
	return nil, nil
}

func writeCSV(t *testing.T, path string, records ...[]string) {
	t.Helper()
	var b bytes.Buffer
	w := csv.NewWriter(&b)
	if err := w.WriteAll(records); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
}

// salusctlResult is salusctl that returns its output and its error instead of failing the test.
func salusctlResult(t *testing.T, args ...string) (string, error) {
	t.Helper()
	out, err := exec.CommandContext(t.Context(), env(t, "E2E_SALUSCTL"), args...).Output()
	return string(out), err
}
