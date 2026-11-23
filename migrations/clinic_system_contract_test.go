package migrations_test

import (
	"context"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// clinicSystemContract is the migration of the clinic system's shared contract (docs/plans/clinic-cms.md, step 0).
const clinicSystemContract = 59

// Every schedule rule, slot and appointment that existed before the migration is a video one: the rules gain the mode
// key, the columns their default.
func TestClinicSystemContract_EverythingThatExistedIsVideo(t *testing.T) {
	t.Parallel()
	dbURL := pgtest.NewEmptyDatabaseURL(t)
	provider := gooseProvider(t, dbURL)
	if _, err := provider.UpTo(t.Context(), clinicSystemContract-1); err != nil {
		t.Fatal(err)
	}
	owner := open(t, dbURL)
	dir := seedDirectory(t, owner)
	var schedule string
	if err := owner.QueryRow(t.Context(), `INSERT INTO schedule (organization_id, practitioner_role_id, slot_minutes, availability)
		VALUES ($1, $2, 30, '[{"day": 1, "start": "08:00", "end": "12:00"}, {"day": 3, "start": "14:00", "end": "16:00"}]')
		RETURNING id`, clinicA, dir[clinicA].role).Scan(&schedule); err != nil {
		t.Fatal(err)
	}
	if _, err := owner.Exec(t.Context(), `INSERT INTO slot (schedule_id, organization_id, period)
		VALUES ($1, $2, tstzrange('2030-01-07T08:00:00Z', '2030-01-07T08:30:00Z'))`, schedule, clinicA); err != nil {
		t.Fatal(err)
	}
	if _, err := provider.UpTo(t.Context(), clinicSystemContract); err != nil {
		t.Fatal(err)
	}

	var modes []string
	if err := owner.QueryRow(t.Context(), `SELECT array_agg(r->>'mode' ORDER BY r->>'day') FROM schedule, jsonb_array_elements(availability) r
		WHERE id = $1`, schedule).Scan(&modes); err != nil || !slices.Equal(modes, []string{"video", "video"}) {
		t.Errorf("rules' modes = %v, %v; want video for each", modes, err)
	}
	var slotMode string
	if err := owner.QueryRow(t.Context(), `SELECT visit_mode FROM slot WHERE schedule_id = $1`, schedule).Scan(&slotMode); err != nil || slotMode != "video" {
		t.Errorf("slot's mode = %q, %v; want video", slotMode, err)
	}
}

// A slot and an appointment are in-person or video and nothing else; an appointment's mode never changes in the app.
func TestClinicSystemContract_VisitModeIsInPersonOrVideoAndAnAppointmentKeepsIt(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	p, _, a := seedAppointments(t, owner)
	for _, stmt := range []string{
		`UPDATE appointment SET visit_mode = 'phone' WHERE id = $1`,
		`UPDATE appointment SET visit_mode = NULL WHERE id = $1`,
	} {
		if _, err := owner.Exec(t.Context(), stmt, a.aFuture); sqlState(err) != "23514" && sqlState(err) != "23502" {
			t.Errorf("%s = %v, want a constraint violation", stmt, err)
		}
	}
	if _, err := owner.Exec(t.Context(), `UPDATE appointment SET visit_mode = 'in-person' WHERE id = $1`, a.aFuture); err != nil {
		t.Errorf("the owner sets in-person: %v", err)
	}
	err := asApp(t, app, map[string]string{"app.patient_id": p[0].patient}, func(ctx context.Context, db postgres.Querier) error {
		_, err := db.Exec(ctx, `UPDATE appointment SET visit_mode = 'video' WHERE id = $1`, a.aFuture)
		return err
	})
	if sqlState(err) != "42501" {
		t.Errorf("the app changes an appointment's mode: %v, want permission denied", err)
	}
}

// billingRows are one clinic's price-list entry and two receipts: one to patient A, one to a company.
type billingRows struct {
	service, patientReceipt, companyReceipt string
}

// seedBilling creates, as the owner, a service of clinic A with a translation and two of its receipts, and a service
// of clinic B.
func seedBilling(t *testing.T, owner *pgxpool.Pool) ([2]patientRow, billingRows) {
	t.Helper()
	ctx := t.Context()
	p, _, a := seedAppointments(t, owner)
	var b billingRows
	if err := owner.QueryRow(ctx, `INSERT INTO charge_item_definition (organization_id, name, price_minor, modes)
		VALUES ($1, 'Placeholder exam', 5000, '{in-person}') RETURNING id`, clinicA).Scan(&b.service); err != nil {
		t.Fatal(err)
	}
	if _, err := owner.Exec(ctx, `INSERT INTO charge_item_definition (organization_id, name, price_minor) VALUES ($1, 'Other', 100)`, clinicB); err != nil {
		t.Fatal(err)
	}
	if _, err := owner.Exec(ctx, `INSERT INTO charge_item_definition_translation (charge_item_definition_id, language, name) VALUES ($1, 'en', 'Placeholder exam')`, b.service); err != nil {
		t.Fatal(err)
	}
	b.patientReceipt = issue(t, owner, `INSERT INTO invoice (seller_legal_name, seller_tax_id, seller_address, organization_id, patient_id, appointment_id, kind, number, issued_at,
			issued_by_account_id, payment_method, total_minor)
		VALUES ('Synthetic clinic d.o.o.', '4200000000001', 'Placeholder street 1, 71000', $1, $2, $3, 'receipt', 1, now(), $4, 'cash', 5000) RETURNING id`, b.service, clinicA, p[0].patient, a.aPast, p[0].account)
	b.companyReceipt = issue(t, owner, `INSERT INTO invoice (seller_legal_name, seller_tax_id, seller_address, organization_id, company_name, company_tax_id, kind, number, issued_at,
			issued_by_account_id, payment_method, total_minor)
		VALUES ('Synthetic clinic d.o.o.', '4200000000001', 'Placeholder street 1, 71000', $1, 'Synthetic company', '4200000000000', 'invoice', 2, now(), $2, 'transfer', 5000) RETURNING id`, b.service, clinicA, p[0].account)
	return p, b
}

// addLine is the insert of the first line (position 0, migration 00063) of the service to a receipt.
const addLine = `INSERT INTO invoice_line (invoice_id, position, charge_item_definition_id, name, quantity, unit_price_minor, total_minor)
	VALUES ($1, 0, $2, 'Placeholder exam', 1, 5000, 5000)`

// issue runs the receipt's insert and adds its line of the service in the same transaction, as the owner, and returns
// the receipt's id.
func issue(t *testing.T, owner *pgxpool.Pool, insert, service string, args ...any) string {
	t.Helper()
	var id string
	err := postgres.NewTransactor(owner).WithTx(t.Context(), func(ctx context.Context) error {
		db := postgres.Conn(ctx, owner)
		if err := db.QueryRow(ctx, insert, args...).Scan(&id); err != nil {
			return err
		}
		_, err := db.Exec(ctx, addLine, id, service)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return id
}

// The price list is the clinic's; the public directory sees the active services of listed clinics only.
func TestChargeItemDefinition_IsTheClinicsAndListedClinicsShowTheirActiveOnes(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	_, b := seedBilling(t, owner)
	if got := idsIn(t, app, map[string]string{"app.organization_id": clinicA}, `SELECT id::text FROM charge_item_definition`); !slices.Equal(got, []string{b.service}) {
		t.Errorf("clinic A sees %v, want its one service", got)
	}
	if got := idsIn(t, app, map[string]string{"app.organization_id": clinicA}, `SELECT charge_item_definition_id::text FROM charge_item_definition_translation`); !slices.Equal(got, []string{b.service}) {
		t.Errorf("clinic A sees the translations %v, want its service's", got)
	}
	if got := idsIn(t, app, nil, `SELECT id::text FROM charge_item_definition`); len(got) != 0 {
		t.Errorf("no scope sees %v, want none", got)
	}
	directory := map[string]string{"app.directory": "public"}
	if got := idsIn(t, app, directory, `SELECT id::text FROM charge_item_definition`); len(got) != 0 {
		t.Errorf("the directory sees %v before any clinic is listed, want none", got)
	}
	if _, err := owner.Exec(t.Context(), `UPDATE organization SET listed = true WHERE id = $1`, clinicA); err != nil {
		t.Fatal(err)
	}
	if got := idsIn(t, app, directory, `SELECT charge_item_definition_id::text FROM charge_item_definition_translation`); !slices.Equal(got, []string{b.service}) {
		t.Errorf("the directory sees the translations %v, want the listed clinic's", got)
	}
	if _, err := owner.Exec(t.Context(), `UPDATE charge_item_definition SET active = false WHERE id = $1`, b.service); err != nil {
		t.Fatal(err)
	}
	if got := idsIn(t, app, directory, `SELECT id::text FROM charge_item_definition`); len(got) != 0 {
		t.Errorf("the directory sees the inactive %v, want none", got)
	}
}

// A service is offered in at least one known mode, at a price of zero or more in an ISO 4217 currency, with a VAT
// rate in basis points or none (exempt).
func TestChargeItemDefinition_RefusesAnInvalidService(t *testing.T) {
	t.Parallel()
	owner := pgtest.NewDatabase(t)
	seedDirectory(t, owner)
	for _, values := range []string{
		`'x', -1, 'BAM', '{video}', NULL`,
		`'x', 100, 'KM', '{video}', NULL`,
		`'x', 100, 'BAM', '{}', NULL`,
		`'x', 100, 'BAM', '{video,phone}', NULL`,
		`'x', 100, 'BAM', '{video}', 10001`,
		`'', 100, 'BAM', '{video}', NULL`,
	} {
		_, err := owner.Exec(t.Context(), `INSERT INTO charge_item_definition (organization_id, name, price_minor, currency, modes, vat_rate_bp)
			VALUES ($1, `+values+`)`, clinicA)
		if sqlState(err) != "23514" {
			t.Errorf("(%s) = %v, want a check violation", values, err)
		}
	}
}

// A receipt is seen by its clinic and by its patient, never by another clinic or patient; a company's invoice only by
// its clinic. Its lines follow it.
func TestInvoice_IsSeenByItsClinicAndItsPatient(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	p, b := seedBilling(t, owner)
	for _, tt := range []struct {
		scope map[string]string
		name  string
		want  []string
	}{
		{name: "no scope"},
		{name: "clinic A", scope: map[string]string{"app.organization_id": clinicA}, want: sorted(b.patientReceipt, b.companyReceipt)},
		{name: "clinic B", scope: map[string]string{"app.organization_id": clinicB}},
		{name: "patient A", scope: map[string]string{"app.patient_id": p[0].patient}, want: []string{b.patientReceipt}},
		{name: "patient B", scope: map[string]string{"app.patient_id": p[1].patient}},
	} {
		if got := idsIn(t, app, tt.scope, `SELECT id::text FROM invoice`); !slices.Equal(got, tt.want) {
			t.Errorf("%s: sees the invoices %v, want %v", tt.name, got, tt.want)
		}
		if got := idsIn(t, app, tt.scope, `SELECT invoice_id::text FROM invoice_line`); !slices.Equal(got, tt.want) {
			t.Errorf("%s: sees the lines of %v, want %v", tt.name, got, tt.want)
		}
	}
}

// An issued receipt never changes and is never removed; only its fiscal columns move, from pending once, and from
// failed to fiscalized when the report is retried. A receipt is reversed at most once.
func TestInvoice_IsAppendOnlyExceptItsFiscalStatus(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	p, b := seedBilling(t, owner)
	clinic := map[string]string{"app.organization_id": clinicA}
	exec := func(stmt string, args ...any) error {
		return asApp(t, app, clinic, func(ctx context.Context, db postgres.Querier) error {
			_, err := db.Exec(ctx, stmt, args...)
			return err
		})
	}
	for _, stmt := range []string{
		`UPDATE invoice SET total_minor = 1 WHERE id = $1`,
		`DELETE FROM invoice WHERE id = $1`,
		`UPDATE invoice_line SET quantity = 2 WHERE invoice_id = $1`,
		`DELETE FROM invoice_line WHERE invoice_id = $1`,
	} {
		if err := exec(stmt, b.patientReceipt); sqlState(err) != "42501" {
			t.Errorf("the app: %s = %v, want permission denied", stmt, err)
		}
	}
	if _, err := owner.Exec(t.Context(), `UPDATE invoice SET total_minor = 1 WHERE id = $1`, b.patientReceipt); err == nil {
		t.Error("the owner changed an issued receipt's total")
	}
	if _, err := owner.Exec(t.Context(), `UPDATE invoice_line SET quantity = 2 WHERE invoice_id = $1`, b.patientReceipt); err == nil {
		t.Error("the owner changed an issued receipt's line")
	}

	fiscalize := `UPDATE invoice SET fiscal_status = 'fiscalized', fiscal_id = 'F-1', fiscal_code = 'C-1', fiscal_qr = 'Q-1',
		fiscalized_at = now() WHERE id = $1`
	if err := exec(`UPDATE invoice SET fiscal_status = 'failed' WHERE id = $1`, b.companyReceipt); err != nil {
		t.Errorf("pending to failed: %v", err)
	}
	if err := exec(fiscalize, b.companyReceipt); err != nil {
		t.Errorf("failed to fiscalized (the retried report): %v", err)
	}
	if err := exec(`UPDATE invoice SET fiscal_status = 'fiscalized' WHERE id = $1`, b.patientReceipt); sqlState(err) != "23514" {
		t.Errorf("fiscalized without an identifier and a code = %v, want a check violation", err)
	}
	if err := exec(`UPDATE invoice SET fiscal_status = 'fiscalized', fiscal_id = 'F-1', fiscal_code = 'C-1', fiscalized_at = now()
		WHERE id = $1`, b.patientReceipt); sqlState(err) != "23514" {
		t.Errorf("fiscalized without the QR = %v, want a check violation", err)
	}
	if err := exec(fiscalize, b.patientReceipt); err != nil {
		t.Errorf("pending to fiscalized: %v", err)
	}
	for _, stmt := range []string{
		`UPDATE invoice SET fiscal_status = 'pending' WHERE id = $1`,
		`UPDATE invoice SET fiscal_code = 'C-2' WHERE id = $1`,
		`UPDATE invoice SET fiscal_qr = 'Q-2' WHERE id = $1`,
		`UPDATE invoice SET fiscal_status = 'failed' WHERE id = $1`,
	} {
		if err := exec(stmt, b.patientReceipt); err == nil {
			t.Errorf("a fiscalized receipt changed: %s", stmt)
		}
	}

	reverse := `INSERT INTO invoice (seller_legal_name, seller_tax_id, seller_address, organization_id, patient_id, kind, reverses_id, number, issued_at, issued_by_account_id,
		payment_method, total_minor) VALUES ('Synthetic clinic d.o.o.', '4200000000001', 'Placeholder street 1, 71000', $1, $2, 'receipt', $3, $4, now(), $5, 'cash', -5000)`
	if err := exec(reverse, clinicA, p[0].patient, b.patientReceipt, 3, p[0].account); err != nil {
		t.Errorf("the reversal: %v", err)
	}
	if err := exec(reverse, clinicA, p[0].patient, b.patientReceipt, 4, p[0].account); sqlState(err) != "23505" {
		t.Errorf("a second reversal = %v, want a unique violation", err)
	}
}

// A receipt names a patient, a company or neither (an anonymous cash sale), never both; numbers are unique per clinic.
func TestInvoice_RefusesAPatientAndACompanyTogetherAndARepeatedNumber(t *testing.T) {
	t.Parallel()
	owner := pgtest.NewDatabase(t)
	p, _ := seedBilling(t, owner)
	insert := `INSERT INTO invoice (seller_legal_name, seller_tax_id, seller_address, organization_id, patient_id, company_name, kind, number, issued_at, issued_by_account_id,
		payment_method, total_minor) VALUES ('Synthetic clinic d.o.o.', '4200000000001', 'Placeholder street 1, 71000', $1, $2, $3, 'receipt', $4, now(), $5, 'card', 100)`
	if _, err := owner.Exec(t.Context(), insert, clinicA, p[0].patient, "Synthetic company", 10, p[0].account); sqlState(err) != "23514" {
		t.Errorf("patient and company = %v, want a check violation", err)
	}
	if _, err := owner.Exec(t.Context(), insert, clinicA, nil, nil, 11, p[0].account); err != nil {
		t.Errorf("an anonymous cash sale: %v", err)
	}
	if _, err := owner.Exec(t.Context(), insert, clinicA, nil, nil, 11, p[0].account); sqlState(err) != "23505" {
		t.Errorf("a repeated number = %v, want a unique violation", err)
	}
	if _, err := owner.Exec(t.Context(), insert, clinicB, nil, nil, 11, p[0].account); err != nil {
		t.Errorf("another clinic's number 11: %v", err)
	}
}

// The clinic's receipt sequence and its subscription are its own; plans are read by the app, written by the owner.
func TestSubscription_IsTheClinicsAndPlansAreReadOnlyForTheApp(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seedDirectory(t, owner)
	ctx := t.Context()
	if _, err := owner.Exec(ctx, `INSERT INTO subscription_plan (code, name, price_per_seat_minor, features) VALUES ('basic', 'Basic', 0, '{}')`); err != nil {
		t.Fatal(err)
	}
	for _, org := range []string{clinicA, clinicB} {
		if _, err := owner.Exec(ctx, `INSERT INTO clinic_subscription (organization_id, plan_code, status, trial_ends_at)
			VALUES ($1, 'basic', 'trial', now() + interval '30 days')`, org); err != nil {
			t.Fatal(err)
		}
		if _, err := owner.Exec(ctx, `INSERT INTO subscription_invoice (organization_id, number, period_start, period_end, seats,
			amount_minor, issued_at, due_at) VALUES ($1, $2, '2026-12-01', '2027-01-01', 1, 0, now(), now() + interval '14 days')`,
			org, map[string]int{clinicA: 1, clinicB: 2}[org]); err != nil {
			t.Fatal(err)
		}
		if _, err := owner.Exec(ctx, `INSERT INTO invoice_sequence (organization_id) VALUES ($1)`, org); err != nil {
			t.Fatal(err)
		}
	}
	scope := map[string]string{"app.organization_id": clinicA}
	for _, q := range []string{
		`SELECT organization_id::text FROM clinic_subscription`,
		`SELECT organization_id::text FROM subscription_invoice`,
		`SELECT organization_id::text FROM invoice_sequence`,
	} {
		if got := idsIn(t, app, scope, q); !slices.Equal(got, []string{clinicA}) {
			t.Errorf("clinic A: %s = %v, want its own", q, got)
		}
	}
	if got := idsIn(t, app, nil, `SELECT code FROM subscription_plan`); !slices.Equal(got, []string{"basic"}) {
		t.Errorf("the app reads the plans %v, want basic", got)
	}
	err := asApp(t, app, scope, func(ctx context.Context, db postgres.Querier) error {
		_, err := db.Exec(ctx, `UPDATE subscription_plan SET price_per_seat_minor = 1`)
		return err
	})
	if sqlState(err) != "42501" {
		t.Errorf("the app changes a plan: %v, want permission denied", err)
	}
	if _, err := owner.Exec(ctx, `INSERT INTO subscription_invoice (organization_id, number, period_start, period_end, seats,
		amount_minor, issued_at, due_at) VALUES ($1, 3, '2026-12-01', '2027-01-01', 1, 0, now(), now())`, clinicA); sqlState(err) != "23505" {
		t.Errorf("a second invoice of one clinic and month = %v, want a unique violation", err)
	}
}

// The clinic's address and billing details are columns of organization the clinic's admin sets.
func TestOrganization_HasTheAddressAndBillingDetails(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seedDirectory(t, owner)
	err := asApp(t, app, map[string]string{"app.organization_id": clinicA}, func(ctx context.Context, db postgres.Querier) error {
		_, err := db.Exec(ctx, `UPDATE organization SET address_line = 'Placeholder street 1', postal_code = '71000',
			legal_name = 'Synthetic clinic d.o.o.', tax_id = '4200000000000', vat_registered = true,
			billing_email = 'billing@example.test' WHERE id = $1`, clinicA)
		return err
	})
	if err != nil {
		t.Errorf("the clinic sets its address and billing details: %v", err)
	}
	var vat bool
	if err := owner.QueryRow(t.Context(), `SELECT vat_registered FROM organization WHERE id = $1`, clinicB).Scan(&vat); err != nil || vat {
		t.Errorf("a clinic's vat_registered = %v, %v; want false by default", vat, err)
	}
}

// A receipt's lines are added only in the transaction that issues it, and only of its own clinic's price list; a visit's
// receipt is of the visit's clinic even when it names no patient.
func TestInvoice_RefusesLaterLinesAnotherClinicsServiceAndAnotherClinicsVisit(t *testing.T) {
	t.Parallel()
	owner := pgtest.NewDatabase(t)
	p, b := seedBilling(t, owner)
	if _, err := owner.Exec(t.Context(), addLine, b.patientReceipt, b.service); sqlState(err) != "23514" {
		t.Errorf("a line added later = %v, want a check violation", err)
	}
	var otherService string
	if err := owner.QueryRow(t.Context(), `SELECT id FROM charge_item_definition WHERE organization_id = $1`, clinicB).Scan(&otherService); err != nil {
		t.Fatal(err)
	}
	err := postgres.NewTransactor(owner).WithTx(t.Context(), func(ctx context.Context) error {
		db := postgres.Conn(ctx, owner)
		var id string
		if err := db.QueryRow(ctx, `INSERT INTO invoice (seller_legal_name, seller_tax_id, seller_address, organization_id, kind, number, issued_at, issued_by_account_id, payment_method,
			total_minor) VALUES ('Synthetic clinic d.o.o.', '4200000000001', 'Placeholder street 1, 71000', $1, 'receipt', 20, now(), $2, 'cash', 5000) RETURNING id`, clinicA, p[0].account).Scan(&id); err != nil {
			return err
		}
		_, err := db.Exec(ctx, addLine, id, otherService)
		return err
	})
	if sqlState(err) != "23514" {
		t.Errorf("a line of clinic B's service on clinic A's receipt = %v, want a check violation", err)
	}
	var bVisit string
	if err := owner.QueryRow(t.Context(), `SELECT id FROM appointment WHERE organization_id = $1 LIMIT 1`, clinicB).Scan(&bVisit); err != nil {
		t.Fatal(err)
	}
	_, err = owner.Exec(t.Context(), `INSERT INTO invoice (seller_legal_name, seller_tax_id, seller_address, organization_id, company_name, appointment_id, kind, number, issued_at,
		issued_by_account_id, payment_method, total_minor) VALUES ('Synthetic clinic d.o.o.', '4200000000001', 'Placeholder street 1, 71000', $1, 'Synthetic company', $2, 'invoice', 21, now(), $3, 'transfer', 0)`,
		clinicA, bVisit, p[0].account)
	if sqlState(err) != "23503" {
		t.Errorf("clinic A's company invoice of clinic B's visit = %v, want a foreign key violation", err)
	}
}

// Salus's invoices are marked paid by the owner (salusctl), never by the app.
func TestSubscriptionInvoice_IsNotMarkedPaidByTheApp(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seedDirectory(t, owner)
	if _, err := owner.Exec(t.Context(), `INSERT INTO subscription_invoice (organization_id, number, period_start, period_end, seats,
		amount_minor, issued_at, due_at) VALUES ($1, 1, '2026-12-01', '2027-01-01', 1, 0, now(), now())`, clinicA); err != nil {
		t.Fatal(err)
	}
	err := asApp(t, app, map[string]string{"app.organization_id": clinicA}, func(ctx context.Context, db postgres.Querier) error {
		_, err := db.Exec(ctx, `UPDATE subscription_invoice SET status = 'paid', paid_at = now()`)
		return err
	})
	if sqlState(err) != "42501" {
		t.Errorf("the app marks its invoice paid: %v, want permission denied", err)
	}
}

// A receipt carries its seller as it was at issue (the clinic's legal name, tax id and address may change later), and
// never reverses itself.
func TestInvoice_KeepsItsSellerAndNeverReversesItself(t *testing.T) {
	t.Parallel()
	owner := pgtest.NewDatabase(t)
	p, b := seedBilling(t, owner)
	_, err := owner.Exec(t.Context(), `INSERT INTO invoice (organization_id, kind, number, issued_at, issued_by_account_id,
		payment_method, total_minor) VALUES ($1, 'receipt', 30, now(), $2, 'cash', 0)`, clinicA, p[0].account)
	if sqlState(err) != "23502" {
		t.Errorf("a receipt without its seller = %v, want a not-null violation", err)
	}
	if _, err := owner.Exec(t.Context(), `UPDATE invoice SET seller_tax_id = '4200000000002' WHERE id = $1`, b.patientReceipt); err == nil {
		t.Error("the owner changed an issued receipt's seller")
	}
	_, err = owner.Exec(t.Context(), `INSERT INTO invoice (id, reverses_id, seller_legal_name, seller_tax_id, seller_address,
		organization_id, kind, number, issued_at, issued_by_account_id, payment_method, total_minor)
		VALUES ('00000000-0000-4000-8000-000000000301', '00000000-0000-4000-8000-000000000301', 'Synthetic clinic d.o.o.',
		'4200000000001', 'Placeholder street 1, 71000', $1, 'receipt', 31, now(), $2, 'cash', 0)`, clinicA, p[0].account)
	if sqlState(err) != "23514" {
		t.Errorf("a receipt that reverses itself = %v, want a check violation", err)
	}
}
