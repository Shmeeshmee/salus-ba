package api_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"maps"
	"net/http"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/api"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// searchset is a searchset Bundle as the paging reads it: its links, its entries as sent, and its total.
type searchset struct {
	Total *int `json:"total"`
	Link  []struct {
		Relation string `json:"relation"`
		URL      string `json:"url"`
	} `json:"link"`
	Entry []json.RawMessage `json:"entry"`
}

// next is the path of the bundle's next link, empty on the last page.
func (b searchset) next(t *testing.T) string {
	t.Helper()
	for _, l := range b.Link {
		if l.Relation == "next" {
			u, err := url.Parse(l.URL)
			if err != nil {
				t.Fatal(err)
			}
			return u.RequestURI()
		}
	}
	return ""
}

func getBundle(t *testing.T, c *client, path string) searchset {
	t.Helper()
	rec := c.do(http.MethodGet, path, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET %s = %d %s", path, rec.Code, rec.Body)
	}
	var b searchset
	if err := json.Unmarshal(rec.Body.Bytes(), &b); err != nil {
		t.Fatal(err)
	}
	return b
}

// pagesOf follows the search's next links from path to the last page, and gives the entries of every page,
// concatenated, and how many pages there were. A page of several has no total; only a first page that is the whole
// answer has one.
func pagesOf(t *testing.T, c *client, path string) (entries []json.RawMessage, pages int) {
	t.Helper()
	for path != "" && pages < 100 {
		b := getBundle(t, c, path)
		pages++
		next := b.next(t)
		if b.Total != nil && (pages > 1 || next != "") {
			t.Errorf("GET %s: a total on page %d of several: %d", path, pages, *b.Total)
		}
		entries, path = append(entries, b.Entry...), next
	}
	return entries, pages
}

// searchEvents is how many search-type audit events name each entity.
func searchEvents(t *testing.T, owner *pgxpool.Pool) map[string]int {
	t.Helper()
	rows, err := owner.Query(t.Context(), "SELECT entity_reference FROM audit_event WHERE subtype_code = 'search-type' AND entity_reference IS NOT NULL")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var ref string
		if err := rows.Scan(&ref); err != nil {
			t.Fatal(err)
		}
		out[ref]++
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

// audited runs fn and gives the search-type events it wrote, by entity.
func audited(t *testing.T, owner *pgxpool.Pool, fn func()) map[string]int {
	t.Helper()
	before := searchEvents(t, owner)
	fn()
	out := map[string]int{}
	for ref, n := range searchEvents(t, owner) {
		if d := n - before[ref]; d > 0 {
			out[ref] = d
		}
	}
	return out
}

// sameEntries reports whether two searches gave the same entries, byte for byte, in the same order.
func sameEntries(a, b []json.RawMessage) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !bytes.Equal(a[i], b[i]) {
			return false
		}
	}
	return true
}

// seedHistory gives the matrix's patient six more past appointments with the practitioner, each with a finished
// encounter (started at the appointment's start) and an unanswered questionnaire response (due at it), and returns
// every appointment of the patient with an encounter, by start. Synthetic.
func seedHistory(t *testing.T, owner *pgxpool.Pool, m matrix) []string {
	t.Helper()
	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := owner.Exec(t.Context(), q, args...); err != nil {
			t.Fatal(err)
		}
	}
	for day := 1; day <= 6; day++ {
		exec(`INSERT INTO appointment (patient_id, practitioner_role_id, organization_id, slot_id, period, created_at)
			VALUES ($1, $2, $3, gen_random_uuid(), tstzrange(date_trunc('hour', now()) - make_interval(days => $4),
				date_trunc('hour', now()) - make_interval(days => $4) + interval '30 minutes'), now() - interval '30 days')`,
			m.patientID, m.roleID, inviteClinic, day)
	}
	exec(`INSERT INTO encounter (appointment_id, organization_id, patient_id, practitioner_role_id, status, period_start, period_end)
		SELECT id, organization_id, patient_id, practitioner_role_id, 'finished', lower(period), upper(period) FROM appointment WHERE lower(period) < now()`)
	exec(`INSERT INTO questionnaire (organization_id, series_id, version, status, title, published_at)
		VALUES ($1, gen_random_uuid(), 1, 'active', 'Placeholder questionnaire', now() - interval '60 days')`, inviteClinic)
	exec(`INSERT INTO questionnaire_response (organization_id, appointment_id, patient_id, practitioner_role_id, questionnaire_id, series_id, source, due_at, created_at)
		SELECT a.organization_id, a.id, a.patient_id, a.practitioner_role_id, q.id, q.series_id, 'added', lower(a.period), now() - interval '30 days'
		FROM appointment a, questionnaire q WHERE lower(a.period) < now()`)
	rows, err := owner.Query(t.Context(), "SELECT appointment_id::text FROM encounter ORDER BY period_start, id")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var appts []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		appts = append(appts, id)
	}
	if err := rows.Err(); err != nil || len(appts) != 7 {
		t.Fatalf("seeded %v, %v; want seven past appointments", appts, err)
	}
	return appts
}

// A staff search's pages, followed by their next links through the api, concatenated, are byte for byte the entries
// the same matches get without pages (the calendar's closed range for Appointment, one appointment at a time for
// Encounter and QuestionnaireResponse), each page's matches audited as the search without pages audits them; the
// default page holds them all here, with its total, as the searches did before paging.
func TestNew_PagesTheStaffSearchesWithoutChangingTheirEntries(t *testing.T) {
	t.Parallel()
	cfg := testConfig(t)
	owner := ownerPool(t, cfg)
	cfg.DatabaseURL = pgtest.AppURL(t, cfg.DatabaseURL)
	app, err := api.New(t.Context(), cfg, testLiveKit(), testObjectStore(t), testDrafter(), testSummarizer(), slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(app.Close)
	m := newMatrix(t, app.Handler, owner, cfg.FHIRBaseURL)
	appts := seedHistory(t, owner, m)

	now := clock.System{}.Now()
	week := url.Values{"date": {"ge" + now.Add(-30*24*time.Hour).UTC().Format(time.RFC3339), "lt" + now.Add(30*24*time.Hour).UTC().Format(time.RFC3339)}}
	byPractitioner := "practitioner=Practitioner/" + m.practitionerID
	for _, tt := range []struct {
		caller      *client
		whole       func() []json.RawMessage // the same matches without pages
		name, paged string
		matches     int
	}{
		{m.admin, func() []json.RawMessage {
			return getBundle(t, m.admin, "/fhir/R4/Appointment?"+week.Encode()).Entry
		}, "the admin's appointments", "/fhir/R4/Appointment?", 8},
		{m.practitioner, func() []json.RawMessage {
			return getBundle(t, m.practitioner, "/fhir/R4/Appointment?"+byPractitioner+"&"+week.Encode()).Entry
		}, "the practitioner's appointments", "/fhir/R4/Appointment?" + byPractitioner + "&", 8},
		{m.admin, func() []json.RawMessage {
			var all []json.RawMessage
			for _, a := range appts {
				all = append(all, getBundle(t, m.admin, "/fhir/R4/Encounter?appointment="+a).Entry...)
			}
			return all
		}, "the admin's encounters", "/fhir/R4/Encounter?", 7},
		{m.practitioner, func() []json.RawMessage {
			var all []json.RawMessage
			for _, a := range appts {
				all = append(all, getBundle(t, m.practitioner, "/fhir/R4/QuestionnaireResponse?_include=QuestionnaireResponse:questionnaire&appointment="+a).Entry...)
			}
			return all
		}, "the practitioner's questionnaire responses with their versions", "/fhir/R4/QuestionnaireResponse?_include=QuestionnaireResponse:questionnaire&", 14}, // each response with its version
	} {
		var whole []json.RawMessage
		wholeEvents := audited(t, owner, func() { whole = tt.whole() })
		if len(whole) != tt.matches {
			t.Fatalf("%s: %d entries without pages, want %d", tt.name, len(whole), tt.matches)
		}
		var first searchset
		if events := audited(t, owner, func() { first = getBundle(t, tt.caller, tt.paged) }); !maps.Equal(events, wholeEvents) {
			t.Errorf("%s: the default page audited %v, want %v", tt.name, events, wholeEvents)
		}
		normal := func(entries []json.RawMessage) []json.RawMessage { return entries }
		if tt.name == "the practitioner's questionnaire responses with their versions" {
			normal = func(entries []json.RawMessage) []json.RawMessage { return withOneInclude(t, entries) } // a page includes each version once
		}
		if first.next(t) != "" || first.Total == nil || !sameEntries(normal(first.Entry), whole) {
			t.Errorf("%s: the default page = %d entries, next %q, total %v; want them all with their total", tt.name, len(first.Entry), first.next(t), first.Total)
		}
		for _, count := range []int{1, 2, 3} {
			var got []json.RawMessage
			var pages int
			events := audited(t, owner, func() { got, pages = pagesOf(t, tt.caller, tt.paged+"_count="+strconv.Itoa(count)) })
			if !sameEntries(normal(got), whole) {
				t.Errorf("%s by %d: the pages' %d entries differ from the %d without pages", tt.name, count, len(got), len(whole))
			}
			if pages < 3 {
				t.Errorf("%s by %d: %d pages, want several", tt.name, count, pages)
			}
			if !maps.Equal(events, wholeEvents) {
				t.Errorf("%s by %d: the pages audited %v, want %v", tt.name, count, events, wholeEvents)
			}
		}
	}
	for _, path := range []string{"/fhir/R4/Appointment?_count=0", "/fhir/R4/Encounter?_count=x", "/fhir/R4/QuestionnaireResponse?_cursor=x"} {
		if rec := m.admin.do(http.MethodGet, path, ""); rec.Code != http.StatusBadRequest {
			t.Errorf("GET %s = %d %s, want 400", path, rec.Code, rec.Body)
		}
	}
}

// withOneInclude gives the QuestionnaireResponse pages' entries as one search without pages lists them when each
// response answers the same version: each match followed by the version, the way one appointment's search gives
// its one response and the version.
func withOneInclude(t *testing.T, entries []json.RawMessage) []json.RawMessage {
	t.Helper()
	var matches []json.RawMessage
	var include json.RawMessage
	for _, e := range entries {
		var mode struct {
			Search struct{ Mode string } `json:"search"`
		}
		if err := json.Unmarshal(e, &mode); err != nil {
			t.Fatal(err)
		}
		switch {
		case mode.Search.Mode == "match":
			matches = append(matches, e)
		case include == nil:
			include = e
		case !bytes.Equal(include, e):
			t.Errorf("two pages include different versions: %s and %s", include, e)
		}
	}
	var out []json.RawMessage
	for _, e := range matches {
		out = append(out, e, include)
	}
	return out
}
