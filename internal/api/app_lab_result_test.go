package api_test

import (
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/api"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// labPDF is a synthetic lab report's PDF.
const labPDF = "%PDF-1.4\n% placeholder lab report\n%%EOF\n"

// A lab result's PDF on the real stack, row-level security on (docs/clinic-cms/WAVE6.md U5): the visit's practitioner
// attaches it at the report's version, encrypted and quarantined until its scan; the clinic's staff read it once clean;
// the patient only once the report is released, and from then on it is frozen. Each read and write is audited.
func TestLabResultPDF_StoredScannedAndReadByWhoMay(t *testing.T) {
	t.Parallel()
	cfg := testConfig(t)
	owner := ownerPool(t, cfg)
	count := openDB(t, cfg)
	cfg.DatabaseURL = pgtest.AppURL(t, cfg.DatabaseURL)
	app, err := api.New(t.Context(), cfg, testLiveKit(), testObjectStore(t), testDrafter(), testSummarizer(), slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(app.Close)
	d := newDirectory(t, app.Handler, owner)
	visit := bookedVisit(t, d, owner)
	rec := d.practitioner.do(http.MethodPost, "/clinic/appointments/"+visit+"/diagnostic-reports", `{"title":"Placeholder blood count","results":[]}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST diagnostic-reports = %d %s", rec.Code, rec.Body)
	}
	var report struct{ ID string }
	decode(t, rec, &report)
	pdf, mine := "/clinic/diagnostic-reports/"+report.ID+"/pdf", "/me/diagnostic-reports/"+report.ID+"/pdf"
	d.practitioner.ifMatch = `W/"1"`
	if rec := d.practitioner.do(http.MethodPut, pdf, "plain text, no PDF"); rec.Code != http.StatusBadRequest ||
		!strings.Contains(rec.Body.String(), "invalid_diagnostic_report") {
		t.Errorf("PUT a text = %d %s", rec.Code, rec.Body)
	}
	rec = d.practitioner.do(http.MethodPut, pdf, labPDF)
	if rec.Code != http.StatusOK || rec.Header().Get("ETag") != `W/"2"` || !strings.Contains(rec.Body.String(), `"hasPdf":true`) {
		t.Fatalf("PUT pdf = %d %s %s", rec.Code, rec.Header(), rec.Body)
	}
	d.practitioner.ifMatch = ""
	if rec := d.admin.do(http.MethodGet, pdf, ""); rec.Code != http.StatusNotFound {
		t.Errorf("GET pdf while it is scanned = %d %s, want 404", rec.Code, rec.Body)
	}
	if _, err := owner.Exec(t.Context(), `UPDATE "binary" SET status = 'clean', scanned_at = now() WHERE diagnostic_report_id = $1`, report.ID); err != nil {
		t.Fatal(err)
	}
	for name, c := range map[string]*client{"the admin": d.admin, "the practitioner": d.practitioner} {
		if rec := c.do(http.MethodGet, pdf, ""); rec.Code != http.StatusOK || rec.Body.String() != labPDF ||
			rec.Header().Get("Content-Type") != "application/pdf" {
			t.Errorf("GET pdf as %s = %d %q", name, rec.Code, rec.Body)
		}
	}
	if rec := d.patient.do(http.MethodGet, mine, ""); rec.Code != http.StatusNotFound {
		t.Errorf("the patient's GET of a preliminary result's PDF = %d %s, want 404", rec.Code, rec.Body)
	}
	d.practitioner.ifMatch = `W/"2"`
	if rec := d.practitioner.do(http.MethodPost, "/clinic/diagnostic-reports/"+report.ID+"/release", ""); rec.Code != http.StatusOK {
		t.Fatalf("release = %d %s", rec.Code, rec.Body)
	}
	d.practitioner.ifMatch = `W/"3"`
	if rec := d.practitioner.do(http.MethodPut, pdf, labPDF); rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "report_released") {
		t.Errorf("PUT pdf after the release = %d %s, want 409 report_released", rec.Code, rec.Body)
	}
	d.practitioner.ifMatch = ""
	if rec := d.patient.do(http.MethodGet, mine, ""); rec.Code != http.StatusOK || rec.Body.String() != labPDF {
		t.Errorf("the patient's GET of the released result's PDF = %d %q", rec.Code, rec.Body)
	}
	if n := count(`SELECT count(*) FROM "binary" WHERE diagnostic_report_id = $1`, report.ID); n != 1 {
		t.Errorf("%d files of the report, want its one PDF", n)
	}
	if n := count(`SELECT count(*) FROM audit_event WHERE entity_reference = $1 AND action = 'R' AND outcome = 0`,
		"DiagnosticReport/"+report.ID); n < 3 {
		t.Errorf("%d audited reads of the report, want the PDF's reads", n)
	}
}
