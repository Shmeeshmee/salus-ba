package migrations_test

import (
	"testing"

	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// The retention role (00028, docs/attachments/SPEC.md §5 "Retention"): binary's purger reads the files of
// every clinic, past row-level security, to tell which stored objects still have a row, and deletes the
// rejected rows past their 24 hours. It cannot insert, update or truncate one.
func TestBinary_TheRetentionRoleReadsAndDeletesFilesOfEveryClinicAndChangesNone(t *testing.T) {
	t.Parallel()
	owner, ret := pgtest.NewRetentionDatabase(t)
	_, a, r, f := seedFiles(t, owner)
	rejected := f[r.open[a.bPast]]
	if _, err := owner.Exec(t.Context(), `UPDATE "binary" SET status = 'rejected', scanned_at = now() - interval '25 hours',
		wrapped_key = NULL WHERE id = $1`, rejected); err != nil {
		t.Fatal(err)
	}

	var n int
	if err := ret.QueryRow(t.Context(), `SELECT count(*) FROM "binary"`).Scan(&n); err != nil || n != len(f) {
		t.Errorf("%s reads %d files, %v; want every clinic's %d", pgtest.RetentionRole, n, err, len(f))
	}
	tag, err := ret.Exec(t.Context(), `DELETE FROM "binary" WHERE status = 'rejected' AND scanned_at < now() - interval '24 hours'`)
	if err != nil || tag.RowsAffected() != 1 {
		t.Errorf("purge: %d rows, %v; want the rejected one", tag.RowsAffected(), err)
	}
	if got := idsWith(t, owner, nil, `SELECT id::text FROM "binary" WHERE id = $1`, rejected); len(got) != 0 {
		t.Errorf("the rejected file is left: %v", got)
	}
	for _, stmt := range []string{
		`UPDATE "binary" SET status = 'clean'`,
		`INSERT INTO "binary" SELECT * FROM "binary"`,
		`TRUNCATE "binary"`,
	} {
		if _, err := ret.Exec(t.Context(), stmt); sqlState(err) != "42501" {
			t.Errorf("%s as %s: %v, want insufficient_privilege", stmt, pgtest.RetentionRole, err)
		}
	}
}

// The purger finds the rejected rows past their period by an index, not by reading every file.
func TestBinary_RejectedFilesAreIndexedByTheirScanTime(t *testing.T) {
	t.Parallel()
	db := pgtest.NewDatabase(t)
	var def string
	if err := db.QueryRow(t.Context(), `SELECT indexdef FROM pg_indexes WHERE tablename = 'binary' AND indexname = 'binary_rejected_scanned'`).Scan(&def); err != nil {
		t.Fatalf("no index binary_rejected_scanned: %v", err)
	}
	if want := `CREATE INDEX binary_rejected_scanned ON public."binary" USING btree (scanned_at) WHERE (status = 'rejected'::text)`; def != want {
		t.Errorf("index %s, want %s", def, want)
	}
}
