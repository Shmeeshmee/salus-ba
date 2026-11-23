package invoice_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"

	"github.com/Shmeeshmee/salus/internal/invoice"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// aQuote is clinic A's quote to the patient: an examination of its price list with a discount and an item off it.
func aQuote(patientID string) invoice.Quote {
	vat := int32(1700)
	return invoice.Quote{
		IssuedAt: at, ValidUntil: at.Add(30 * 24 * time.Hour), OrganizationID: clinicA, PatientID: patientID,
		CreatedByAccountID: staffA, Note: "Placeholder note", Currency: "BAM", TotalMinor: 10670,
		Lines: []invoice.Line{
			{ChargeItemDefinitionID: serviceA, Name: "Placeholder examination", Quantity: 2, UnitPriceMinor: 5000, DiscountMinor: 500, TotalMinor: 9500},
			{Name: "Placeholder item", Quantity: 1, UnitPriceMinor: 1170, TotalMinor: 1170, VATRateBP: &vat},
		},
	}
}

// A quote is stored with its lines in their order and read back by its clinic and its patient only. It needs the
// clinic's active care consent of the patient.
func TestRepository_StoresAQuoteForItsClinicAndItsPatient(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seed(t, owner)
	if _, err := owner.Exec(t.Context(), `INSERT INTO consent (patient_id, organization_id, status, date_time) VALUES ($1, $2, 'active', now())`,
		patA, clinicA); err != nil {
		t.Fatal(err)
	}
	repo, tr := invoice.NewPostgresRepository(app), postgres.NewTransactor(app)
	var stored invoice.Quote
	if err := in(t, tr, repo, clinicA, "", func(ctx context.Context) error {
		var err error
		stored, err = repo.InsertQuote(ctx, aQuote(patA))
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if stored.ID == "" || stored.Lines[0].ID == "" {
		t.Fatalf("InsertQuote() = %+v", stored)
	}
	err := in(t, tr, repo, clinicA, "", func(ctx context.Context) error {
		_, err := repo.InsertQuote(ctx, aQuote(patB)) // patient B gave clinic A no care consent
		return err
	})
	if !errors.Is(err, invoice.ErrNotFound) {
		t.Errorf("a quote to a patient without consent: %v, want ErrNotFound", err)
	}
	for _, scope := range []struct{ clinic, patient string }{{clinicA, ""}, {"", patA}} {
		var got invoice.Quote
		var list []invoice.Quote
		if err := in(t, tr, repo, scope.clinic, scope.patient, func(ctx context.Context) error {
			var err error
			if got, err = repo.QuoteByID(ctx, stored.ID); err != nil {
				return err
			}
			if scope.clinic != "" {
				list, err = repo.QuotesOf(ctx, clinicA)
			} else {
				list, err = repo.QuotesOfPatient(ctx, patA)
			}
			return err
		}); err != nil {
			t.Fatalf("%+v: %v", scope, err)
		}
		if got.Note != "Placeholder note" || got.PatientID != patA || !got.ValidUntil.Equal(stored.ValidUntil) || len(got.Lines) != 2 ||
			got.Lines[0].Name != "Placeholder examination" || got.Lines[1].VATRateBP == nil || got.TotalMinor != 10670 {
			t.Errorf("%+v: QuoteByID() = %+v", scope, got)
		}
		if len(list) != 1 || list[0].ID != stored.ID || len(list[0].Lines) != 2 {
			t.Errorf("%+v: list = %+v", scope, list)
		}
	}
	for _, scope := range []struct{ clinic, patient string }{{clinicB, ""}, {"", patB}} {
		err := in(t, tr, repo, scope.clinic, scope.patient, func(ctx context.Context) error {
			_, err := repo.QuoteByID(ctx, stored.ID)
			return err
		})
		if !errors.Is(err, invoice.ErrNotFound) {
			t.Errorf("%+v sees the quote: %v", scope, err)
		}
	}
}

// A quote takes no number from the clinic's receipt sequence and enqueues no fiscal report: its transaction leaves
// invoice_sequence and invoice untouched, and its one job is the patient's e-mail, with the two ids only.
func TestRepository_AQuoteTakesNoReceiptNumberAndNoFiscalJob(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seed(t, owner)
	ctx := t.Context()
	if _, err := owner.Exec(ctx, `INSERT INTO consent (patient_id, organization_id, status, date_time) VALUES ($1, $2, 'active', now())`,
		patA, clinicA); err != nil {
		t.Fatal(err)
	}
	client, err := river.NewClient(riverpgxv5.New(app), &river.Config{})
	if err != nil {
		t.Fatal(err)
	}
	repo, jobs, tr := invoice.NewPostgresRepository(app), invoice.NewRiverJobs(client), postgres.NewTransactor(app)
	const appointment = "e1000000-0000-4000-8000-0000000000e1" // an opaque id: the job names it, nothing reads it here
	if err := in(t, tr, repo, clinicA, "", func(ctx context.Context) error {
		if _, err := repo.InsertQuote(ctx, aQuote(patA)); err != nil {
			return err
		}
		return jobs.EnqueueQuoteEmail(ctx, appointment, clinicA)
	}); err != nil {
		t.Fatal(err)
	}
	var sequences, receipts int
	if err := owner.QueryRow(ctx, `SELECT (SELECT count(*) FROM invoice_sequence), (SELECT count(*) FROM invoice)`).Scan(&sequences, &receipts); err != nil {
		t.Fatal(err)
	}
	if sequences != 0 || receipts != 0 {
		t.Errorf("after a quote: %d receipt sequences, %d receipts; want none", sequences, receipts)
	}
	var kinds []string
	rows, err := owner.Query(ctx, `SELECT kind || ' ' || queue || ' ' || args::text FROM river_job ORDER BY id`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			t.Fatal(err)
		}
		kinds = append(kinds, k)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	want := []string{`quote_email email {"appointment_id": "` + appointment + `", "organization_id": "` + clinicA + `"}`}
	if !slices.Equal(kinds, want) {
		t.Errorf("jobs = %q, want only %q (no invoice_fiscalize)", kinds, want)
	}
	if err := jobs.EnqueueQuoteEmail(ctx, appointment, clinicA); !errors.Is(err, postgres.ErrNoTx) {
		t.Errorf("EnqueueQuoteEmail() outside a transaction = %v, want ErrNoTx", err)
	}
}
