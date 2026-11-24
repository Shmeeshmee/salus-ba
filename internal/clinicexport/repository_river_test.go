package clinicexport_test

import (
	"context"
	"errors"
	"testing"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"

	"github.com/Shmeeshmee/salus/internal/clinicexport"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

func TestRiverJobs_EnqueuesTheBuildInTheTransactionWithOnlyIDs(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	client, err := river.NewClient(riverpgxv5.New(app), &river.Config{})
	if err != nil {
		t.Fatal(err)
	}
	jobs := clinicexport.NewRiverJobs(client)
	args := clinicexport.BuildArgs{ExportID: adminA, OrganizationID: clinicA}
	if err := postgres.NewTransactor(app).WithTx(t.Context(), func(ctx context.Context) error { return jobs.EnqueueBuild(ctx, args) }); err != nil {
		t.Fatalf("EnqueueBuild() = %v", err)
	}
	var kind, got string
	var attempts int
	if err := owner.QueryRow(t.Context(), "SELECT kind, args::text, max_attempts FROM river_job").Scan(&kind, &got, &attempts); err != nil {
		t.Fatal(err)
	}
	want := `{"export_id": "` + adminA + `", "organization_id": "` + clinicA + `"}`
	if kind != "clinic_export_build" || got != want || attempts != clinicexport.BuildAttempts {
		t.Errorf("job %s with %s, %d attempts; want clinic_export_build with %s, %d", kind, got, attempts, want, clinicexport.BuildAttempts)
	}
	if err := jobs.EnqueueBuild(t.Context(), args); !errors.Is(err, postgres.ErrNoTx) {
		t.Errorf("EnqueueBuild() outside a transaction = %v, want postgres.ErrNoTx", err)
	}
}
