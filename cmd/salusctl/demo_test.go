package main

import (
	"bytes"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/api"
	"github.com/Shmeeshmee/salus/internal/config"
	"github.com/Shmeeshmee/salus/pkg/llm/llmtest"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
	"github.com/Shmeeshmee/salus/pkg/redis/redistest"
)

// The demo seed refuses outside a developer's stack, whatever the database holds: nothing is written.
func TestDemoSeed_IsRefusedOutsideDev(t *testing.T) {
	_, count := ownerDB(t)
	for _, env := range []string{"", "production", "staging"} {
		t.Setenv("SALUS_ENV", env)
		err := run(t.Context(), []string{"demo-seed", "-api", "http://127.0.0.1:1"}, &bytes.Buffer{})
		if !errors.Is(err, errDemoRefused) || !strings.Contains(err.Error(), "SALUS_ENV") {
			t.Errorf("SALUS_ENV=%q: error = %v, want the refusal naming SALUS_ENV", env, err)
		}
	}
	if n := count("SELECT count(*) FROM organization") + count("SELECT count(*) FROM subscription_plan"); n != 0 {
		t.Errorf("%d rows written by a refused seed, want none", n)
	}
}

// On a developer's stack it still refuses a database with a clinic that is not the demo's (every production or
// staging database), and one it stopped on partway; nothing is written.
func TestDemoSeed_RefusesADatabaseThatIsNotFresh(t *testing.T) {
	db, count := ownerDB(t)
	t.Setenv("SALUS_ENV", config.EnvDev)
	if _, err := db.Exec(t.Context(), `INSERT INTO organization (name) VALUES ('Synthetic clinic')`); err != nil {
		t.Fatal(err)
	}
	err := run(t.Context(), []string{"demo-seed", "-api", "http://127.0.0.1:1"}, &bytes.Buffer{})
	if !errors.Is(err, errDemoRefused) || !strings.Contains(err.Error(), "fresh stack") {
		t.Errorf("another clinic: error = %v, want the refusal", err)
	}
	if _, err := db.Exec(t.Context(), `UPDATE organization SET name = $1`, demoClinicName); err != nil {
		t.Fatal(err)
	}
	err = run(t.Context(), []string{"demo-seed", "-api", "http://127.0.0.1:1"}, &bytes.Buffer{})
	if !errors.Is(err, errDemoRefused) || !strings.Contains(err.Error(), "partway") {
		t.Errorf("a partial demo clinic: error = %v, want the refusal", err)
	}
	if _, err := db.Exec(t.Context(), `INSERT INTO organization (name) VALUES ('Synthetic clinic')`); err != nil {
		t.Fatal(err)
	}
	err = run(t.Context(), []string{"demo-seed", "-api", "http://127.0.0.1:1"}, &bytes.Buffer{})
	if !errors.Is(err, errDemoRefused) || !strings.Contains(err.Error(), "1 clinics besides") {
		t.Errorf("the demo clinic and another: error = %v, want the refusal", err)
	}
	for _, args := range [][]string{{"demo-seed", "-bogus"}, {"demo-seed", "extra"}} {
		if err := run(t.Context(), args, &bytes.Buffer{}); err == nil || errors.Is(err, errDemoRefused) {
			t.Errorf("run(%q): error = %v, want the usage", args, err)
		}
	}
	if n := count("SELECT count(*) FROM organization"); n != 2 {
		t.Errorf("%d clinics, want the 2 the test made", n)
	}
}

// demoAPI is the api on the test's database (as salus_app, row-level security on) behind an HTTP server.
func demoAPI(t *testing.T, ownerURL string) string {
	t.Helper()
	cfg := config.Config{
		DatabaseURL: pgtest.AppURL(t, ownerURL), RedisURL: redistest.URL(t),
		TOTPKey:      []byte("synthetic 32-byte TOTP key, test"),
		RateLimitKey: []byte("synthetic 32-byte rate-limit key"),
		WebhookKey:   []byte("synthetic 32-byte webhook key!!!"),
		FHIRBaseURL:  "https://api.example.test/fhir/R4",
	}
	files := config.ObjectStore{
		Endpoint: os.Getenv("TEST_S3_ENDPOINT"), Region: os.Getenv("TEST_S3_REGION"), Bucket: os.Getenv("TEST_S3_BUCKET"),
		AccessKeyID: os.Getenv("TEST_S3_ACCESS_KEY_ID"), SecretAccessKey: os.Getenv("TEST_S3_SECRET_ACCESS_KEY"),
		KEKID: "test-1", KEK: []byte("synthetic 32-byte KEK for tests!"),
	}
	if files.Endpoint == "" {
		t.Fatal("TEST_S3_* are not set; run the tests with `make test`, which points them at the compose object store")
	}
	lk := config.LiveKit{URL: "http://livekit.invalid:7880", PublicURL: "ws://localhost:7880", APIKey: "synthetic-key",
		APISecret: "synthetic-webhook-secret-of-enough-length"}
	app, err := api.New(t.Context(), cfg, lk, files, &llmtest.Drafter{DiscardCalls: true}, &llmtest.Summarizer{DiscardCalls: true},
		slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("api.New: %v", err)
	}
	t.Cleanup(app.Close)
	srv := httptest.NewServer(app.Handler)
	t.Cleanup(srv.Close)
	return srv.URL
}

// The seed fills a fresh database through the api with everything the demo shows, and a second run changes nothing
// and does not fail.
func TestDemoSeed_FillsAFreshStackOnce(t *testing.T) {
	db, count := ownerDB(t)
	t.Setenv("SALUS_ENV", config.EnvDev)
	base := demoAPI(t, db.Config().ConnString())
	var out bytes.Buffer
	if err := run(t.Context(), []string{"demo-seed", "-api", base}, &out); err != nil {
		t.Fatalf("demo-seed: %v", err)
	}
	for _, c := range demoCounts {
		if n := count(c.query); n < c.min {
			t.Errorf("%s: %d, want at least %d", c.name, n, c.min)
		}
	}
	if !strings.Contains(out.String(), demoAdminEmail) || !strings.Contains(out.String(), "TOTP ") {
		t.Errorf("output %q: want the sign-in sheet with the secrets", out.String())
	}
	var secrets []string
	for _, line := range strings.Split(out.String(), "\n") {
		if _, secret, ok := strings.Cut(line, " TOTP "); ok && strings.HasPrefix(line, "  ") {
			secrets = append(secrets, secret)
		}
	}
	if len(secrets) != 6 {
		t.Errorf("%d TOTP secrets on the sheet, want 6", len(secrets))
	}
	before := demoSnapshot(t, count)
	out.Reset()
	if err := run(t.Context(), []string{"demo-seed", "-api", base}, &out); err != nil {
		t.Fatalf("demo-seed again: %v", err)
	}
	if after := demoSnapshot(t, count); after != before {
		t.Errorf("a second run changed the database: %v, then %v", before, after)
	}
	if !strings.Contains(out.String(), "nothing changed") || !strings.Contains(out.String(), demoAdminEmail) {
		t.Errorf("second output %q: want the sheet", out.String())
	}
	for _, secret := range secrets {
		if strings.Contains(out.String(), secret) {
			t.Errorf("the second run printed a TOTP secret")
		}
	}
}

// demoCounts is what the demo shows, each at least once (or the number SCRIPT.md promises).
var demoCounts = []struct {
	name, query string
	min         int
}{
	{"clinics", "SELECT count(*) FROM organization WHERE listed", 1},
	{"locations", "SELECT count(*) FROM location", 2},
	{"rooms", "SELECT count(*) FROM room", 5},
	{"devices", "SELECT count(*) FROM device", 1},
	{"practitioners", "SELECT count(*) FROM practitioner_role WHERE specialty_code IS NOT NULL", 4},
	{"receptionists", "SELECT count(*) FROM account WHERE role = 'reception'", 1},
	{"admins", "SELECT count(*) FROM account WHERE role = 'admin'", 1},
	{"services", "SELECT count(*) FROM charge_item_definition", 9},
	{"patients", "SELECT count(*) FROM patient", 60},
	{"app patients", "SELECT count(*) FROM account WHERE role = 'patient'", 5},
	{"visits behind", "SELECT count(*) FROM appointment WHERE lower(period) < now() - interval '7 days'", 40},
	{"visits ahead", "SELECT count(*) FROM appointment WHERE lower(period) > now() + interval '1 day'", 40},
	{"fulfilled", "SELECT count(*) FROM appointment WHERE status = 'fulfilled'", 20},
	{"no-shows", "SELECT count(*) FROM appointment WHERE status = 'noshow'", 1},
	{"cancelled behind", "SELECT count(*) FROM appointment WHERE status = 'cancelled' AND lower(period) < now()", 1},
	{"cancelled ahead", "SELECT count(*) FROM appointment WHERE status = 'cancelled' AND lower(period) > now()", 1},
	{"booked", "SELECT count(*) FROM appointment WHERE status = 'booked'", 20},
	{"arrived", "SELECT count(*) FROM appointment WHERE status = 'arrived'", 1},
	{"video visits", "SELECT count(*) FROM appointment WHERE visit_mode = 'video'", 5},
	{"receipts", "SELECT count(*) FROM invoice WHERE kind = 'receipt'", 20},
	{"a reversal", "SELECT count(*) FROM invoice WHERE reverses_id IS NOT NULL", 1},
	{"quotes", "SELECT count(*) FROM quote", 2},
	{"lab results released", "SELECT count(*) FROM diagnostic_report WHERE status = 'final'", 1},
	{"an active treatment plan", "SELECT count(*) FROM care_plan WHERE status = 'active'", 1},
	{"a questionnaire with branching", "SELECT count(*) FROM questionnaire WHERE status = 'active'", 1},
	{"a submitted questionnaire", "SELECT count(*) FROM questionnaire_response WHERE status = 'completed'", 1},
	{"chat messages", "SELECT count(*) FROM communication", 3},
	{"data requests", "SELECT count(*) FROM data_request", 2},
	{"the membership invoice", "SELECT count(*) FROM subscription_invoice", 1},
}

// demoSnapshot is the row counts of the seeded tables and the audit trail's, which any write of a run would change.
func demoSnapshot(t *testing.T, count func(string, ...any) int) [2]int {
	t.Helper()
	total := 0
	for _, c := range demoCounts {
		total += count(c.query)
	}
	return [2]int{total, count("SELECT count(*) FROM audit_event")}
}

// A refused call's error names the method, the route and the api's code: never the query (a name or a birth date may
// be in it) or the rest of the body.
func TestDemoUser_ARefusalNamesTheRouteAndTheCodeOnly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"error":"version_conflict","patient":"Amra Primjerović"}`))
	}))
	t.Cleanup(srv.Close)
	u := &demoUser{http: srv.Client(), base: srv.URL}
	_, err := u.must(t.Context(), http.StatusOK, http.MethodGet, "/clinic/patients?name=Amra&birthDate=1945-01-01", nil, nil)
	if err == nil || err.Error() != "demo-seed: GET /clinic/patients = 409 version_conflict, want 200" {
		t.Errorf("error = %v", err)
	}
}

// The arrival is today's next in-person visit still to come; late in the day the last one that began; with none
// booked today, a spare slot of today's.
func TestArrival_IsTodaysNextInPersonVisit(t *testing.T) {
	loc, err := time.LoadLocation("Europe/Sarajevo")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 25, 12, 0, 0, 0, loc)
	at := func(h int, mode, status string) demoVisit {
		return demoVisit{start: time.Date(2026, 11, 25, h, 0, 0, 0, loc), mode: mode, status: status}
	}
	tomorrow := demoVisit{start: now.AddDate(0, 0, 1), mode: "in-person", status: "booked"}
	for _, tt := range []struct {
		name         string
		month, spare []demoVisit
		want         int // index of the arrival in the result, -1 for none
	}{
		{"the next one to come", []demoVisit{at(9, "in-person", "fulfilled"), at(15, "in-person", "booked"), at(13, "in-person", "booked")}, nil, 2},
		{"video and cancelled visits never arrive", []demoVisit{at(13, "video", "booked"), at(14, "in-person", "cancelled"), at(10, "in-person", "fulfilled")}, nil, 2},
		{"late in the day, the last that began", []demoVisit{at(8, "in-person", "fulfilled"), at(11, "in-person", "noshow"), tomorrow}, nil, 1},
		{"a spare slot of today", []demoVisit{tomorrow}, []demoVisit{at(16, "in-person", "booked")}, 1},
		{"no visit today", []demoVisit{tomorrow}, nil, -1},
	} {
		got := arrival(tt.month, tt.spare, now, loc)
		for i, v := range got {
			if (v.status == "arrived") != (i == tt.want) {
				t.Errorf("%s: visit %d is %s", tt.name, i, v.status)
			}
		}
	}
}

// The sheet lines up names with letters of two bytes.
func TestPad_CountsCharactersNotBytes(t *testing.T) {
	if got := pad("Testić", 8); got != "Testić  " {
		t.Errorf("pad = %q", got)
	}
	if got := pad("Primjerović", 4); got != "Primjerović" {
		t.Errorf("pad of a longer name = %q", got)
	}
}
