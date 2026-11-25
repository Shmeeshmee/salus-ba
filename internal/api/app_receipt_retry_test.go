package api_test

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"testing"

	"github.com/Shmeeshmee/salus/internal/api"
)

// Package H's retry on Postgres: a receipt whose report job gave up is reported again once however often the admin
// asks while that report is still queued, so the tax authority is not sent the same receipt once per click.
func TestNew_RetryingAFailedReceiptQueuesOneReport(t *testing.T) {
	t.Parallel()
	cfg := testConfig(t)
	app, err := api.New(t.Context(), cfg, testLiveKit(), testObjectStore(t), testDrafter(), testSummarizer(), slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(app.Close)
	owner := ownerPool(t, cfg)
	d := newDirectory(t, app.Handler, owner)
	visit := bookedVisit(t, d, owner)
	if _, err := owner.Exec(t.Context(), `UPDATE organization SET legal_name = 'Synthetic clinic d.o.o.', tax_id = '4200000000001',
		region_id = 'BA-09', city_id = 'Q1270190', location_since = now(), address_line = 'Synthetic street 1' WHERE id = $1`, inviteClinic); err != nil {
		t.Fatal(err)
	}
	exam := saveService(t, d, `{"name":"Synthetic exam","priceMinor":5000,"modes":["video","in-person"],"active":true,"translations":[]}`)
	id := issueReceipt(t, d, visit, exam)

	// The report job gave up: River discarded it and the receipt is failed.
	for _, q := range []string{
		`UPDATE river_job SET state = 'discarded', finalized_at = now() WHERE kind = 'invoice_fiscalize' AND args->>'invoice_id' = $1`,
		`UPDATE invoice SET fiscal_status = 'failed' WHERE id = $1`,
	} {
		if _, err := owner.Exec(t.Context(), q, id); err != nil {
			t.Fatal(err)
		}
	}
	for range 2 {
		if rec := d.admin.do(http.MethodPost, "/clinic/receipts/"+id+"/retry", ""); rec.Code != http.StatusAccepted {
			t.Fatalf("POST /clinic/receipts/%s/retry = %d %s, want 202", id, rec.Code, rec.Body)
		}
	}
	var queued int
	if err := owner.QueryRow(t.Context(), `SELECT count(*) FROM river_job WHERE kind = 'invoice_fiscalize' AND args->>'invoice_id' = $1
		AND state NOT IN ('discarded', 'cancelled', 'completed')`, id).Scan(&queued); err != nil {
		t.Fatal(err)
	}
	if queued != 1 {
		t.Errorf("%d report jobs queued after two retries, want 1", queued)
	}
	var r struct{ FiscalStatus string }
	rec := d.admin.do(http.MethodGet, "/clinic/receipts/"+id, "")
	if err := json.Unmarshal(rec.Body.Bytes(), &r); err != nil || r.FiscalStatus != "failed" {
		t.Errorf("the receipt is %q (%v), want still failed until its report runs", r.FiscalStatus, err)
	}
}
