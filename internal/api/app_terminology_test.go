package api_test

import (
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/api"
)

func TestNew_StaffSearchTheTerminologyInTheirLanguageAndNoSearchTextIsLogged(t *testing.T) {
	cfg := testConfig(t)
	var log syncBuffer
	app, err := api.New(t.Context(), cfg, testLiveKit(), testObjectStore(t), testDrafter(), testSummarizer(), slog.New(slog.NewTextHandler(&log, nil)))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(app.Close)
	owner := ownerPool(t, cfg)
	d := newDirectory(t, app.Handler, owner)
	_, err = owner.Exec(t.Context(), `
		INSERT INTO term (id, category, rank, wikidata_id, ta98) VALUES
			('00000000-0000-4000-8000-0000000000a1', 'anatomy', 100, 'Q9368', 'A05.8.01.001');
		INSERT INTO term_label (term_id, language, label, preferred) VALUES
			('00000000-0000-4000-8000-0000000000a1', 'la', 'hepar', true),
			('00000000-0000-4000-8000-0000000000a1', 'hr', 'jetra', true),
			('00000000-0000-4000-8000-0000000000a1', 'sr-Cyrl', 'јетра', true),
			('00000000-0000-4000-8000-0000000000a1', 'en', 'liver', true);
		UPDATE account SET language = 'sr-Cyrl' WHERE email = 'cy@example.test';
		UPDATE account SET language = NULL, detected_language = 'hr' WHERE email = 'dee@example.test'`)
	if err != nil {
		t.Fatal(err)
	}
	const secret = "Marko Marković" // a doctor could type a patient's name

	rec := d.practitioner.do(http.MethodGet, "/terminology/search?q=jetra", "")
	want := `"labels":[{"language":"la","preferred":"hepar","synonyms":[]},{"language":"sr-Cyrl","preferred":"јетра","synonyms":[]},` +
		`{"language":"en","preferred":"liver","synonyms":[]}]`
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), want) {
		t.Errorf("GET as the practitioner (sr-Cyrl) = %d %s, want 200 with Latin, Serbian Cyrillic and English", rec.Code, rec.Body)
	}
	rec = d.admin.do(http.MethodGet, "/terminology/search?q=hepar", "")
	want = `"labels":[{"language":"la","preferred":"hepar","synonyms":[]},{"language":"hr","preferred":"jetra","synonyms":[]},` +
		`{"language":"en","preferred":"liver","synonyms":[]}]`
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), want) {
		t.Errorf("GET as the admin (no choice, hr detected) = %d %s, want 200 with Latin, Croatian and English", rec.Code, rec.Body)
	}
	rec = d.admin.do(http.MethodGet, "/terminology/search?q=hepar&languages=hr", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"labels":[{"language":"hr","preferred":"jetra","synonyms":[]}]`) {
		t.Errorf("GET as the admin in Croatian = %d %s, want 200 with the Croatian label", rec.Code, rec.Body)
	}
	for _, c := range []struct {
		client *client
		name   string
		status int
	}{
		{d.practitioner, "the practitioner", http.StatusOK},
		{d.patient, "a patient", http.StatusForbidden},
		{&client{t: t, h: app.Handler}, "nobody", http.StatusUnauthorized},
	} {
		if rec := c.client.do(http.MethodGet, "/terminology/search?q="+url.QueryEscape(secret), ""); rec.Code != c.status {
			t.Errorf("GET a name as %s = %d %s, want %d", c.name, rec.Code, rec.Body, c.status)
		}
	}
	if rec := d.practitioner.do(http.MethodGet, "/terminology/search?q="+url.QueryEscape(secret)+"&limit=99", ""); rec.Code != http.StatusBadRequest {
		t.Errorf("GET a name with limit 99 = %d %s, want 400", rec.Code, rec.Body)
	}

	logs := log.String()
	if !strings.Contains(logs, "route=\"GET /terminology/search\"") {
		t.Errorf("the access log has no line for the search route:\n%s", logs)
	}
	for _, text := range []string{"jetra", "hepar", "Marko", "Markovi"} {
		if strings.Contains(logs, text) {
			t.Errorf("the log holds the search text %q:\n%s", text, logs)
		}
	}
}
