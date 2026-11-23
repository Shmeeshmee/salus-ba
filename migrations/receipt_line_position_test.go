package migrations_test

import (
	"context"
	"testing"

	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// receiptLinePosition is the version of 00063 (clinic-cms package H).
const receiptLinePosition = 63

// 00063 numbers the lines of receipts issued before it by name, from 0 within each receipt, so they keep the order
// they were read in; every line after it needs a position, unique within its receipt, and a receipt from before has
// no idempotency key.
func TestReceiptLinePosition_TheBackfillNumbersExistingLinesByName(t *testing.T) {
	t.Parallel()
	dbURL := pgtest.NewEmptyDatabaseURL(t)
	provider := gooseProvider(t, dbURL)
	if _, err := provider.UpTo(t.Context(), receiptLinePosition-1); err != nil {
		t.Fatal(err)
	}
	owner := open(t, dbURL)
	ctx := t.Context()
	var account, service string
	if err := owner.QueryRow(ctx, `WITH o AS (INSERT INTO organization (id, name) VALUES ($1, 'Synthetic clinic A') RETURNING id)
		INSERT INTO account (email, password_hash, role, organization_id) SELECT 'admin@example.test', 'h', 'admin', id FROM o RETURNING id`, clinicA).Scan(&account); err != nil {
		t.Fatal(err)
	}
	if err := owner.QueryRow(ctx, `INSERT INTO charge_item_definition (organization_id, name, price_minor, modes)
		VALUES ($1, 'Placeholder exam', 1000, '{in-person}') RETURNING id`, clinicA).Scan(&service); err != nil {
		t.Fatal(err)
	}
	receipt := func(number int, names ...string) string {
		t.Helper()
		var id string
		err := postgres.NewTransactor(owner).WithTx(ctx, func(ctx context.Context) error {
			db := postgres.Conn(ctx, owner)
			if err := db.QueryRow(ctx, `INSERT INTO invoice (seller_legal_name, seller_tax_id, seller_address, organization_id, company_name,
					company_tax_id, kind, number, issued_at, issued_by_account_id, payment_method, total_minor)
				VALUES ('Synthetic clinic d.o.o.', '4200000000001', 'Placeholder street 1, 71000', $1, 'Synthetic company', '4200000000000',
					'invoice', $2, now(), $3, 'transfer', $4) RETURNING id`, clinicA, number, account, 1000*len(names)).Scan(&id); err != nil {
				return err
			}
			for _, name := range names {
				if _, err := db.Exec(ctx, `INSERT INTO invoice_line (invoice_id, charge_item_definition_id, name, quantity, unit_price_minor, total_minor)
					VALUES ($1, $2, $3, 1, 1000, 1000)`, id, service, name); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	three := receipt(1, "Placeholder C", "Placeholder A", "Placeholder B")
	one := receipt(2, "Placeholder only")

	if _, err := provider.UpTo(ctx, receiptLinePosition); err != nil {
		t.Fatal(err)
	}
	rows, err := owner.Query(ctx, `SELECT invoice_id::text || ' ' || position || ' ' || name FROM invoice_line ORDER BY invoice_id = $1 DESC, position`, three)
	if err != nil {
		t.Fatal(err)
	}
	got, err := collectStrings(rows)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{three + " 0 Placeholder A", three + " 1 Placeholder B", three + " 2 Placeholder C", one + " 0 Placeholder only"}
	if len(got) != len(want) {
		t.Fatalf("the lines are %q, want %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d is %q, want %q", i, got[i], want[i])
		}
	}
	var keys int
	if err := owner.QueryRow(ctx, `SELECT count(*) FROM invoice WHERE idempotency_key IS NOT NULL`).Scan(&keys); err != nil || keys != 0 {
		t.Errorf("receipts from before with an idempotency key: %d (%v), want 0", keys, err)
	}
	var nullable string
	if err := owner.QueryRow(ctx, `SELECT is_nullable FROM information_schema.columns
		WHERE table_name = 'invoice_line' AND column_name = 'position'`).Scan(&nullable); err != nil || nullable != "NO" {
		t.Errorf("invoice_line.position is nullable %q (%v), want NO after the backfill", nullable, err)
	}
}
