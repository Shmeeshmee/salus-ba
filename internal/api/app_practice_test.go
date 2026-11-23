package api_test

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/api"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

// practiceRow is one row of the practice report as the test compares it.
type practiceRow struct {
	mode                     string
	offered, visits, receipt int64
}

// TestNew_ThePracticeReportCountsOfferedHeldAndReceiptedWorkByMode is package E of docs/plans/clinic-cms.md wired
// through the api on Postgres: over a full week of the clinic's own days, the minutes today's working hours offer in
// each visit mode, the visit held (fulfilled) and the amount of its receipt, a reversed receipt left out; only the
// admin reads it, and a visit corrected to a no-show counts for nothing.
func TestNew_ThePracticeReportCountsOfferedHeldAndReceiptedWorkByMode(t *testing.T) {
	cfg := testConfig(t)
	app, err := api.New(t.Context(), cfg, testLiveKit(), testObjectStore(t), testDrafter(), testSummarizer(), slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(app.Close)
	owner := ownerPool(t, cfg)
	d := newDirectory(t, app.Handler, owner)
	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := owner.Exec(t.Context(), q, args...); err != nil {
			t.Fatal(err)
		}
	}
	visit := bookedVisit(t, d, owner) // in person, tomorrow
	exec(`UPDATE organization SET legal_name = 'Synthetic clinic d.o.o.', tax_id = '4200000000001',
		region_id = 'BA-09', city_id = 'Q1270190', location_since = now(), address_line = 'Synthetic street 1' WHERE id = $1`, inviteClinic)
	exam := saveService(t, d, `{"name":"Synthetic exam","priceMinor":5000,"modes":["video","in-person"],"active":true,"translations":[]}`)
	everyDay(t, d)

	// The visit held and receipted twice: one receipt stands, the other is reversed.
	exec(`UPDATE appointment SET status = 'fulfilled' WHERE id = $1`, visit)
	issueReceipt(t, d, visit, exam)
	reversed := issueReceipt(t, d, visit, exam)
	if rec := d.admin.do(http.MethodPost, "/clinic/receipts/"+reversed+"/reverse", `{"paymentMethod":"cash"}`); rec.Code != http.StatusCreated {
		t.Fatalf("POST /clinic/receipts/%s/reverse = %d %s", reversed, rec.Code, rec.Body)
	}

	sarajevo, err := time.LoadLocation("Europe/Sarajevo")
	if err != nil {
		t.Fatal(err)
	}
	today := clock.System{}.Now().In(sarajevo)
	from := time.Date(today.Year(), today.Month(), today.Day()-2, 0, 0, 0, 0, sarajevo)
	report := "/clinic/reports/practice?" + url.Values{"from": {from.Format(time.RFC3339)}, "to": {from.AddDate(0, 0, 7).Format(time.RFC3339)}}.Encode()

	if rec := d.practitioner.do(http.MethodGet, report, ""); rec.Code != http.StatusForbidden {
		t.Errorf("GET the practice report as a practitioner = %d, want 403", rec.Code)
	}
	got := readPractice(t, d, report)
	want := map[string]practiceRow{
		"in-person": {"in-person", 7 * 120, 1, 5000},
		"video":     {"video", 7 * 60, 0, 0},
	}
	if len(got) != 2 || got["in-person"] != want["in-person"] || got["video"] != want["video"] {
		t.Errorf("the report is %+v, want %+v", got, want)
	}

	exec(`UPDATE appointment SET status = 'noshow' WHERE id = $1`, visit)
	for _, r := range readPractice(t, d, report) {
		if r.visits != 0 || r.receipt != 0 {
			t.Errorf("after the visit was corrected to a no-show the report holds %+v, want no visit and no amount", r)
		}
	}
}

// everyDay sets the directory's role to work every day two hours at the clinic and one by video, as its admin.
func everyDay(t *testing.T, d directory) {
	t.Helper()
	path := "/practitioner-roles/" + d.roleID + "/availability"
	d.admin.ifMatch = d.admin.do(http.MethodGet, path, "").Header().Get("ETag")
	defer func() { d.admin.ifMatch = "" }()
	var rules []map[string]any
	for day := 1; day <= 7; day++ {
		rules = append(rules,
			map[string]any{"day": day, "start": "09:00", "end": "11:00", "mode": "in-person"},
			map[string]any{"day": day, "start": "20:00", "end": "21:00", "mode": "video"})
	}
	body, err := json.Marshal(map[string]any{"slotMinutes": 30, "availability": rules})
	if err != nil {
		t.Fatal(err)
	}
	if rec := d.admin.do(http.MethodPut, path, string(body)); rec.Code != http.StatusOK {
		t.Fatalf("PUT %s = %d %s", path, rec.Code, rec.Body)
	}
}

// issueReceipt issues the visit's receipt of one service as the admin and returns its id.
func issueReceipt(t *testing.T, d directory, visit, service string) string {
	t.Helper()
	rec := d.admin.do(http.MethodPost, "/clinic/receipts", `{"kind":"receipt","appointmentId":"`+visit+`","paymentMethod":"cash","lines":[{"serviceId":"`+service+`","quantity":1}]}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /clinic/receipts = %d %s", rec.Code, rec.Body)
	}
	var r struct{ ID string }
	if err := json.Unmarshal(rec.Body.Bytes(), &r); err != nil {
		t.Fatal(err)
	}
	return r.ID
}

// readPractice reads the report as the admin, by visit mode; every row must be the directory's role, in BAM.
func readPractice(t *testing.T, d directory, report string) map[string]practiceRow {
	t.Helper()
	rec := d.admin.do(http.MethodGet, report, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET %s = %d %s", report, rec.Code, rec.Body)
	}
	var got struct {
		Currency string
		Rows     []struct {
			PractitionerRole, VisitMode            string
			MinutesOffered, Visits, ReceiptedMinor int64
		}
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Currency != "BAM" {
		t.Errorf("the report's currency is %q, want BAM", got.Currency)
	}
	out := map[string]practiceRow{}
	for _, r := range got.Rows {
		if r.PractitionerRole != d.roleID {
			t.Errorf("a row of another role %s", r.PractitionerRole)
		}
		out[r.VisitMode] = practiceRow{r.VisitMode, r.MinutesOffered, r.Visits, r.ReceiptedMinor}
	}
	return out
}
