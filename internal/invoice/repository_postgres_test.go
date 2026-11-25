package invoice_test

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"

	"github.com/Shmeeshmee/salus/internal/invoice"
	"github.com/Shmeeshmee/salus/migrations"
	"github.com/Shmeeshmee/salus/pkg/fiscal"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

func TestMain(m *testing.M) { pgtest.Main(m, migrations.FS) }

// Synthetic rows: two clinics, each with an admin's account and a service, and two patients.
const (
	clinicA  = "a1000000-0000-4000-8000-0000000000a1"
	clinicB  = "b1000000-0000-4000-8000-0000000000b1"
	staffA   = "a2000000-0000-4000-8000-0000000000a2"
	staffB   = "b2000000-0000-4000-8000-0000000000b2"
	serviceA = "a3000000-0000-4000-8000-0000000000a3"
	serviceB = "b3000000-0000-4000-8000-0000000000b3"
	patAcctA = "c1000000-0000-4000-8000-0000000000c1"
	patAcctB = "c2000000-0000-4000-8000-0000000000c2"
	patA     = "d1000000-0000-4000-8000-0000000000d1"
	patB     = "d2000000-0000-4000-8000-0000000000d2"
)

func seed(t *testing.T, owner *pgxpool.Pool) {
	t.Helper()
	for _, s := range []string{
		`INSERT INTO organization (id, name) VALUES ('` + clinicA + `', 'Synthetic clinic A'), ('` + clinicB + `', 'Synthetic clinic B')`,
		`INSERT INTO account (id, email, password_hash, role, organization_id, mfa_enrolled_at) VALUES
			('` + staffA + `', 'admin.a@example.test', 'h', 'admin', '` + clinicA + `', now()),
			('` + staffB + `', 'admin.b@example.test', 'h', 'admin', '` + clinicB + `', now())`,
		`INSERT INTO account (id, email, password_hash, role) VALUES
			('` + patAcctA + `', 'pat.a@example.test', 'h', 'patient'), ('` + patAcctB + `', 'pat.b@example.test', 'h', 'patient')`,
		`INSERT INTO patient (id, account_id, given_name, family_name, birth_date) VALUES
			('` + patA + `', '` + patAcctA + `', 'Ada', 'Example', '1990-01-01'), ('` + patB + `', '` + patAcctB + `', 'Cy', 'Example', '1991-01-01')`,
		`INSERT INTO charge_item_definition (id, organization_id, name, price_minor) VALUES
			('` + serviceA + `', '` + clinicA + `', 'Placeholder examination', 5000), ('` + serviceB + `', '` + clinicB + `', 'Placeholder examination', 4000)`,
	} {
		if _, err := owner.Exec(t.Context(), s); err != nil {
			t.Fatal(err)
		}
	}
}

// receipt is a cash receipt of clinic A to the patient: two examinations with a discount and an item off the price
// list with VAT.
func receipt(patientID string) invoice.Invoice {
	vat := int32(1700)
	return invoice.Invoice{
		IssuedAt: at, OrganizationID: clinicA, SellerLegalName: theSeller.LegalName, SellerTaxID: theSeller.TaxID, SellerAddress: theSeller.Address,
		PatientID: patientID, IssuedByAccountID: staffA, Currency: "BAM", Kind: invoice.KindReceipt, PaymentMethod: invoice.PaymentCash,
		TotalMinor: 10670,
		Lines: []invoice.Line{
			{ChargeItemDefinitionID: serviceA, Name: "Placeholder examination", Quantity: 2, UnitPriceMinor: 5000, DiscountMinor: 500, TotalMinor: 9500},
			{Name: "Placeholder item", Quantity: 1, UnitPriceMinor: 1170, TotalMinor: 1170, VATRateBP: &vat},
		},
	}
}

// issue stores inv with the clinic's next number, in a transaction in its clinic's scope.
func issue(t *testing.T, tr *postgres.Transactor, repo invoice.Repository, inv invoice.Invoice) (invoice.Invoice, error) {
	t.Helper()
	var out invoice.Invoice
	err := tr.WithTx(t.Context(), func(ctx context.Context) error {
		if err := repo.SetOrganization(ctx, inv.OrganizationID); err != nil {
			return err
		}
		var err error
		if inv.Number, err = repo.NextNumber(ctx, inv.OrganizationID); err != nil {
			return err
		}
		out, err = repo.Insert(ctx, inv)
		return err
	})
	return out, err
}

// in runs fn in a transaction scoped to the clinic, or to the patient when clinic is empty.
func in(t *testing.T, tr *postgres.Transactor, repo invoice.Repository, clinic, patient string, fn func(ctx context.Context) error) error {
	t.Helper()
	return tr.WithTx(t.Context(), func(ctx context.Context) error {
		var err error
		if clinic != "" {
			err = repo.SetOrganization(ctx, clinic)
		} else {
			err = repo.SetPatient(ctx, patient)
		}
		if err != nil {
			return err
		}
		return fn(ctx)
	})
}

// A receipt is stored with its lines and read back as issued, by its clinic and by its patient, and by no other
// clinic or patient.
func TestRepository_StoresAndShowsAReceiptToItsClinicAndItsPatient(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seed(t, owner)
	repo, tr := invoice.NewPostgresRepository(app), postgres.NewTransactor(app)
	stored, err := issue(t, tr, repo, receipt(patA))
	if err != nil {
		t.Fatal(err)
	}
	if stored.ID == "" || stored.Number != 1 || stored.FiscalStatus != invoice.FiscalPending || stored.Lines[0].ID == "" {
		t.Fatalf("Insert() = %+v", stored)
	}
	for name, tt := range map[string]struct {
		clinic, patient string
		seen            bool
	}{
		"its clinic": {clinic: clinicA, seen: true}, "another clinic": {clinic: clinicB},
		"its patient": {patient: patA, seen: true}, "another patient": {patient: patB},
	} {
		var got invoice.Invoice
		var mine []invoice.Invoice
		err := in(t, tr, repo, tt.clinic, tt.patient, func(ctx context.Context) error {
			var err error
			if mine, err = repo.OfPatient(ctx, patA); err != nil {
				return err
			}
			got, err = repo.ByID(ctx, stored.ID)
			return err
		})
		if tt.seen != (err == nil) || (!tt.seen && !errors.Is(err, invoice.ErrNotFound)) || len(mine) != map[bool]int{true: 1}[tt.seen] {
			t.Errorf("%s: ByID() = %v, OfPatient() = %d receipts; want seen %v", name, err, len(mine), tt.seen)
		}
		if !tt.seen {
			continue
		}
		if got.Number != 1 || got.PatientID != patA || got.SellerTaxID != theSeller.TaxID || got.TotalMinor != 10670 || !got.IssuedAt.Equal(at) ||
			got.IssuedByAccountID != staffA || got.Kind != invoice.KindReceipt || got.PaymentMethod != invoice.PaymentCash || len(got.Lines) != 2 {
			t.Errorf("%s: ByID() = %+v", name, got)
			continue
		}
		exam, item := got.Lines[0], got.Lines[1] // by name
		if exam.ChargeItemDefinitionID != serviceA || exam.Quantity != 2 || exam.DiscountMinor != 500 || exam.TotalMinor != 9500 || exam.VATRateBP != nil ||
			item.ChargeItemDefinitionID != "" || item.VATRateBP == nil || *item.VATRateBP != 1700 || item.UnitPriceMinor != 1170 {
			t.Errorf("%s: lines = %+v", name, got.Lines)
		}
	}
	if err := in(t, tr, repo, clinicA, "", func(ctx context.Context) error {
		_, err := repo.ByID(ctx, "not-a-uuid")
		return err
	}); !errors.Is(err, invoice.ErrNotFound) {
		t.Errorf("ByID(no UUID) = %v, want ErrNotFound", err)
	}
	other := receipt("")
	other.Lines[0].ChargeItemDefinitionID = serviceB // another clinic's service
	if _, err := issue(t, tr, repo, other); !errors.Is(err, invoice.ErrInvalid) {
		t.Errorf("a line from another clinic's price list = %v, want ErrInvalid", err)
	}
}

// Receipts issued at the same instant get consecutive numbers, never the same one and never a gap, and a receipt
// that is rolled back gives its number back; each clinic has its own sequence.
func TestRepository_NumbersWithoutGapOrRepeatUnderConcurrency(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seed(t, owner)
	repo, tr := invoice.NewPostgresRepository(app), postgres.NewTransactor(app)
	const n = 24
	errRollBack := errors.New("synthetic failure after the number was taken")
	var wg sync.WaitGroup
	numbers, errs := make([]int64, n), make([]error, n)
	for i := range n {
		wg.Go(func() {
			errs[i] = tr.WithTx(context.Background(), func(ctx context.Context) error {
				if err := repo.SetOrganization(ctx, clinicA); err != nil {
					return err
				}
				inv := receipt("")
				var err error
				if inv.Number, err = repo.NextNumber(ctx, clinicA); err != nil {
					return err
				}
				if _, err := repo.Insert(ctx, inv); err != nil {
					return err
				}
				if i%3 == 0 { // a third of them fail after taking a number
					return errRollBack
				}
				numbers[i] = inv.Number
				return nil
			})
		})
	}
	wg.Wait()
	var got []int64
	for i, err := range errs {
		switch {
		case i%3 == 0 && !errors.Is(err, errRollBack), i%3 != 0 && err != nil:
			t.Fatalf("receipt %d: %v", i, err)
		case err == nil:
			got = append(got, numbers[i])
		}
	}
	slices.Sort(got)
	want := make([]int64, n-n/3)
	for i := range want {
		want[i] = int64(i + 1)
	}
	if !slices.Equal(got, want) {
		t.Errorf("numbers = %v, want 1 to %d without gap or repeat", got, len(want))
	}
	stored := storedNumbers(t, owner, clinicA)
	if !slices.Equal(stored, want) {
		t.Errorf("stored numbers = %v, want %v", stored, want)
	}
	b := receipt("")
	b.OrganizationID, b.IssuedByAccountID, b.Lines = clinicB, staffB, b.Lines[1:]
	b.TotalMinor = 1170
	if first, err := issue(t, tr, repo, b); err != nil || first.Number != 1 {
		t.Errorf("another clinic's first receipt = number %d, %v; want 1", first.Number, err)
	}
}

// storedNumbers are the clinic's receipt numbers in the database, in order.
func storedNumbers(t *testing.T, owner *pgxpool.Pool, clinic string) []int64 {
	t.Helper()
	rows, err := owner.Query(t.Context(), `SELECT number FROM invoice WHERE organization_id = $1 ORDER BY number`, clinic)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var stored []int64
	for rows.Next() {
		var x int64
		if err := rows.Scan(&x); err != nil {
			t.Fatal(err)
		}
		stored = append(stored, x)
	}
	return stored
}

// A receipt is reversed once; its fiscal status moves from pending to failed, then to fiscalized with the
// authority's answer, and a fiscalized receipt never changes.
func TestRepository_ReversesOnceAndKeepsTheFiscalAnswer(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seed(t, owner)
	repo, tr := invoice.NewPostgresRepository(app), postgres.NewTransactor(app)
	orig, err := issue(t, tr, repo, receipt(patA))
	if err != nil {
		t.Fatal(err)
	}
	rev := receipt(patA)
	rev.ReversesID, rev.TotalMinor = orig.ID, -10670
	for i := range rev.Lines {
		rev.Lines[i].Quantity, rev.Lines[i].TotalMinor = -rev.Lines[i].Quantity, -rev.Lines[i].TotalMinor
	}
	stored, err := issue(t, tr, repo, rev)
	if err != nil || stored.Number != 2 {
		t.Fatalf("the reversal = number %d, %v", stored.Number, err)
	}
	if _, err := issue(t, tr, repo, rev); !errors.Is(err, invoice.ErrAlreadyReversed) {
		t.Errorf("a second reversal = %v, want ErrAlreadyReversed", err)
	}
	answer := fiscal.Fiscalized{At: at, ID: "TEST-1", Code: "code-1", QR: "TEST:qr-1"}
	var got invoice.Invoice
	var listed []invoice.Invoice
	err = in(t, tr, repo, clinicA, "", func(ctx context.Context) error {
		locked, err := repo.Locked(ctx, orig.ID)
		if err != nil || len(locked.Lines) != 2 {
			return errors.Join(err, errors.New("Locked() without its lines"))
		}
		if err := repo.SetFailed(ctx, orig.ID); err != nil {
			return err
		}
		if err := repo.SetFiscalized(ctx, orig.ID, answer); err != nil { // failed may still become fiscalized
			return err
		}
		if err := repo.SetFailed(ctx, orig.ID); err != nil { // a fiscalized receipt is left as it is
			return err
		}
		if got, err = repo.ByID(ctx, orig.ID); err != nil {
			return err
		}
		listed, err = repo.Issued(ctx, clinicA, invoice.Period{From: at, To: at.Add(time12h)})
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.FiscalStatus != invoice.FiscalFiscalized || got.FiscalID != "TEST-1" || got.FiscalCode != "code-1" || got.FiscalQR != "TEST:qr-1" ||
		!got.FiscalizedAt.Equal(at) || got.ReversedByID != stored.ID {
		t.Errorf("after the report = %+v", got)
	}
	if len(listed) != 2 || listed[0].ID != stored.ID || listed[0].ReversesID != orig.ID || len(listed[0].Lines) != 2 || listed[1].ReversedByID != stored.ID {
		t.Errorf("Issued() = %+v, want the reversal then the receipt", listed)
	}
	err = in(t, tr, repo, clinicA, "", func(ctx context.Context) error { return repo.SetFiscalized(ctx, orig.ID, answer) })
	if !errors.Is(err, invoice.ErrNotFound) {
		t.Errorf("fiscalizing a fiscalized receipt = %v, want ErrNotFound (nothing to change)", err)
	}
}

// The fiscal report is enqueued in the receipt's transaction with the two ids only: a receipt that is rolled back
// has no job.
func TestRiverJobs_EnqueueTheReportInTheTransactionWithOnlyTheTwoIDs(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	client, err := river.NewClient(riverpgxv5.New(app), &river.Config{})
	if err != nil {
		t.Fatal(err)
	}
	jobs, tr, ctx := invoice.NewRiverJobs(client), postgres.NewTransactor(app), t.Context()
	errRollBack := errors.New("the receipt failed")
	err = tr.WithTx(ctx, func(ctx context.Context) error {
		if err := jobs.EnqueueFiscalize(ctx, serviceA, clinicA); err != nil {
			return err
		}
		return errRollBack
	})
	var n int
	if qerr := owner.QueryRow(ctx, "SELECT count(*) FROM river_job").Scan(&n); !errors.Is(err, errRollBack) || qerr != nil || n != 0 {
		t.Fatalf("after a rolled-back receipt: %d jobs (%v, %v), want none", n, err, qerr)
	}
	if err := tr.WithTx(ctx, func(ctx context.Context) error { return jobs.EnqueueFiscalize(ctx, serviceA, clinicA) }); err != nil {
		t.Fatal(err)
	}
	var kind, args string
	if err := owner.QueryRow(ctx, "SELECT kind, args::text FROM river_job").Scan(&kind, &args); err != nil {
		t.Fatal(err)
	}
	if want := `{"invoice_id": "` + serviceA + `", "organization_id": "` + clinicA + `"}`; kind != "invoice_fiscalize" || args != want {
		t.Errorf("job = %s with %s; want invoice_fiscalize with %s", kind, args, want)
	}
	if err := jobs.EnqueueFiscalize(ctx, serviceA, clinicA); !errors.Is(err, postgres.ErrNoTx) {
		t.Errorf("EnqueueFiscalize() outside a transaction = %v, want ErrNoTx", err)
	}
}

// The fiscal job carries opaque ids only.
func TestFiscalizeArgs_IsTheInvoiceFiscalizeJob(t *testing.T) {
	if got := (invoice.FiscalizeArgs{InvoiceID: "i", OrganizationID: "o"}).Kind(); got != "invoice_fiscalize" {
		t.Errorf("Kind() = %q, want invoice_fiscalize", got)
	}
}

// Lines come back in the order they were entered, not by name, however the receipt is read.
func TestRepository_KeepsTheLinesInTheirOrder(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seed(t, owner)
	repo, tr := invoice.NewPostgresRepository(app), postgres.NewTransactor(app)
	inv := receipt(patA)
	inv.Lines[0], inv.Lines[1] = inv.Lines[1], inv.Lines[0] // "Placeholder item" before "Placeholder examination"
	inv.Lines = append(inv.Lines, invoice.Line{Name: "A placeholder last", Quantity: 1, UnitPriceMinor: 100, TotalMinor: 100})
	inv.TotalMinor += 100
	stored, err := issue(t, tr, repo, inv)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"Placeholder item", "Placeholder examination", "A placeholder last"}
	names := func(list []invoice.Invoice) []string {
		var out []string
		for _, l := range list[0].Lines {
			out = append(out, l.Name)
		}
		return out
	}
	err = in(t, tr, repo, clinicA, "", func(ctx context.Context) error {
		one, err := repo.ByID(ctx, stored.ID)
		if err != nil {
			return err
		}
		locked, err := repo.Locked(ctx, stored.ID)
		if err != nil {
			return err
		}
		issued, err := repo.Issued(ctx, clinicA, invoice.Period{From: at.Add(-time.Hour), To: at.Add(time.Hour)})
		if err != nil {
			return err
		}
		for name, got := range map[string][]string{
			"ByID": names([]invoice.Invoice{one}), "Locked": names([]invoice.Invoice{locked}), "Issued": names(issued),
		} {
			if !slices.Equal(got, want) {
				t.Errorf("%s lines = %v, want %v", name, got, want)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// A receipt issued with an idempotency key is found by it in its own clinic only; a second receipt with the same key
// in the same clinic is refused by the schema.
func TestRepository_FindsAReceiptByItsIdempotencyKey(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seed(t, owner)
	repo, tr := invoice.NewPostgresRepository(app), postgres.NewTransactor(app)
	const key = "e1000000-0000-4000-8000-0000000000e1"
	inv := receipt(patA)
	inv.IdempotencyKey = key
	stored, err := issue(t, tr, repo, inv)
	if err != nil {
		t.Fatal(err)
	}
	var got invoice.Invoice
	if err := in(t, tr, repo, clinicA, "", func(ctx context.Context) error {
		got, err = repo.ByIdempotencyKey(ctx, clinicA, key)
		return err
	}); err != nil || got.ID != stored.ID || len(got.Lines) != 2 {
		t.Errorf("ByIdempotencyKey() = %+v, %v; want the stored receipt", got, err)
	}
	for name, k := range map[string]string{"another key": "e2000000-0000-4000-8000-0000000000e2", "no UUID": "x"} {
		if err := in(t, tr, repo, clinicA, "", func(ctx context.Context) error {
			_, err := repo.ByIdempotencyKey(ctx, clinicA, k)
			return err
		}); !errors.Is(err, invoice.ErrNotFound) {
			t.Errorf("ByIdempotencyKey(%s) = %v, want ErrNotFound", name, err)
		}
	}
	if err := in(t, tr, repo, clinicB, "", func(ctx context.Context) error {
		_, err := repo.ByIdempotencyKey(ctx, clinicA, key)
		return err
	}); !errors.Is(err, invoice.ErrNotFound) {
		t.Errorf("ByIdempotencyKey() from another clinic = %v, want ErrNotFound", err)
	}
	if _, err := issue(t, tr, repo, inv); err == nil {
		t.Error("a second receipt with the same key was stored")
	}
}

// A receipt keeps the place of service it was issued with (docs/clinic-cms/WAVE6.md U2), and never changes it.
func TestRepository_KeepsThePlaceOfService(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seed(t, owner)
	repo, tr := invoice.NewPostgresRepository(app), postgres.NewTransactor(app)
	inv := receipt(patA)
	inv.PlaceOfService = "Synthetic annex, Synthetic annex 2, 88000 Mostar"
	stored, err := issue(t, tr, repo, inv)
	if err != nil {
		t.Fatal(err)
	}
	var got invoice.Invoice
	if err := in(t, tr, repo, clinicA, "", func(ctx context.Context) (err error) { got, err = repo.ByID(ctx, stored.ID); return err }); err != nil ||
		got.PlaceOfService != inv.PlaceOfService {
		t.Errorf("ByID() place %q, %v; want %q", got.PlaceOfService, err, inv.PlaceOfService)
	}
	if _, err := owner.Exec(t.Context(), `UPDATE invoice SET place_of_service = 'Elsewhere' WHERE id = $1`, stored.ID); err == nil {
		t.Error("the place of service of an issued receipt changed")
	}
}
