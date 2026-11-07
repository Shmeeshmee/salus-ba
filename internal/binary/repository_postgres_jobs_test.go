package binary_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/binary"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// The queries of the scan and discard jobs (the job scope) and of the purger (the retention role).

// insertAsOwner stores b as the owner, with its status and scan time as given.
func insertAsOwner(t *testing.T, owner *pgxpool.Pool, b binary.Binary) {
	t.Helper()
	var scanned any
	if !b.ScannedAt.IsZero() {
		scanned = b.ScannedAt
	}
	var key any
	if b.Status != binary.StatusRejected {
		key = b.WrappedKey
	}
	_, err := owner.Exec(t.Context(), `INSERT INTO "binary" (id, organization_id, patient_id, appointment_id, questionnaire_response_id,
			link_id, content_type, size, title, sha1, kek_id, wrapped_key, segment_size, status, scanned_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $16)`,
		b.ID, b.OrganizationID, b.PatientID, b.AppointmentID, b.ResponseID, b.LinkID, b.ContentType, b.Size, b.Title, b.SHA1,
		b.KEKID, key, b.SegmentSize, string(b.Status), scanned, now)
	if err != nil {
		t.Fatal(err)
	}
}

func TestPostgres_TheJobScopeMarksItsResponsesQuarantinedFileScannedOnce(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	s := seedDB(t, owner)
	clean, rejected := s.file("files", 10, 1), s.file("files", 20, 2)
	clean.Status, rejected.Status = binary.StatusQuarantined, binary.StatusQuarantined
	insertAsOwner(t, owner, clean)
	insertAsOwner(t, owner, rejected)
	at := now.Add(time.Minute)

	for _, c := range []struct {
		scope  func(context.Context, binary.Repository) error
		name   string
		status binary.Status
		id     string
		marks  bool
	}{
		{name: "another response's job scope", scope: asJob(s.otherResponse, s.clinic), id: clean.ID, status: binary.StatusClean},
		{name: "the patient", scope: asPatient(s.patient), id: clean.ID, status: binary.StatusClean},
		{name: "the job scope, clean", scope: asJob(s.response, s.clinic), id: clean.ID, status: binary.StatusClean, marks: true},
		{name: "the job scope, again", scope: asJob(s.response, s.clinic), id: clean.ID, status: binary.StatusRejected},
		{name: "the job scope, rejected", scope: asJob(s.response, s.clinic), id: rejected.ID, status: binary.StatusRejected, marks: true},
		{name: "not a UUID", scope: asJob(s.response, s.clinic), id: "not-a-uuid", status: binary.StatusClean},
	} {
		err := in(t, app, c.scope, func(ctx context.Context, r binary.Repository) error {
			marked, err := r.MarkScanned(ctx, c.id, c.status, at)
			if marked != c.marks || err != nil {
				t.Errorf("%s: MarkScanned() = %v, %v; want %v", c.name, marked, err, c.marks)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	var got []binary.Binary
	if err := in(t, app, asJob(s.response, s.clinic), func(ctx context.Context, r binary.Repository) error {
		var err error
		got, err = r.OfResponse(ctx, s.response)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	wantClean, wantRejected := clean, rejected
	wantClean.Status, wantClean.ScannedAt, wantClean.CreatedAt, wantClean.UpdatedAt = binary.StatusClean, at, now, at
	wantRejected.Status, wantRejected.ScannedAt, wantRejected.CreatedAt, wantRejected.UpdatedAt = binary.StatusRejected, at, now, at
	wantRejected.WrappedKey = nil
	if len(got) != 2 || !equalBinary(got[0], wantClean) || !equalBinary(got[1], wantRejected) {
		t.Errorf("OfResponse() = %+v,\nwant %+v and %+v", got, wantClean, wantRejected)
	}
}

func TestPostgres_OfResponseListsEveryFileOfTheJobScopesResponseOnly(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	s := seedDB(t, owner)
	var want []string
	for i, status := range []binary.Status{binary.StatusQuarantined, binary.StatusClean, binary.StatusRejected} {
		b := s.file("files", int64(10+i), byte(i))
		b.Status = status
		if status != binary.StatusQuarantined {
			b.ScannedAt = now
		}
		insertAsOwner(t, owner, b)
		want = append(want, b.ID)
	}
	other := s.file("files", 5, 9)
	other.PatientID, other.ResponseID, other.Status = s.other, s.otherResponse, binary.StatusQuarantined
	if err := owner.QueryRow(t.Context(), `SELECT appointment_id::text FROM questionnaire_response WHERE id = $1`, s.otherResponse).Scan(&other.AppointmentID); err != nil {
		t.Fatal(err)
	}
	insertAsOwner(t, owner, other)

	for name, c := range map[string]struct {
		scope func(context.Context, binary.Repository) error
		want  []string
	}{
		"the job scope":                {scope: asJob(s.response, s.clinic), want: want},
		"another response's job scope": {scope: asJob(s.otherResponse, s.clinic), want: nil},
	} {
		err := in(t, app, c.scope, func(ctx context.Context, r binary.Repository) error {
			files, err := r.OfResponse(ctx, s.response)
			var got []string
			for _, b := range files {
				got = append(got, b.ID)
			}
			if err != nil || !slices.Equal(got, c.want) {
				t.Errorf("%s: OfResponse() = %v, %v; want %v", name, got, err, c.want)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

// The purger's queries run as salus_retention, past row-level security, in every clinic.
func TestPostgres_TheRetentionRoleDeletesOldRejectedFilesAndTellsWhichFilesExist(t *testing.T) {
	t.Parallel()
	owner, ret := pgtest.NewRetentionDatabase(t)
	s := seedDB(t, owner)
	cut := now.Add(-24 * time.Hour)
	rows := map[string]binary.Binary{}
	for name, c := range map[string]struct {
		scanned time.Time
		status  binary.Status
	}{
		"old rejected":   {cut.Add(-time.Second), binary.StatusRejected},
		"fresh rejected": {cut.Add(time.Second), binary.StatusRejected},
		"old clean":      {cut.Add(-time.Hour), binary.StatusClean},
		"quarantined":    {status: binary.StatusQuarantined},
	} {
		b := s.file("files", 10, 1)
		b.Status, b.ScannedAt = c.status, c.scanned
		insertAsOwner(t, owner, b)
		rows[name] = b
	}
	repo := binary.NewPostgresRepository(ret)
	n, err := repo.DeleteRejectedBefore(t.Context(), cut)
	if err != nil || n != 1 {
		t.Errorf("DeleteRejectedBefore() = %d, %v; want 1", n, err)
	}
	keys := []string{rows["old rejected"].ID, rows["fresh rejected"].ID, rows["old clean"].ID, rows["quarantined"].ID, unknownID, "test/not-a-uuid"}
	got, err := repo.Existing(t.Context(), keys)
	slices.Sort(got)
	want := []string{rows["fresh rejected"].ID, rows["old clean"].ID, rows["quarantined"].ID}
	slices.Sort(want)
	if err != nil || !slices.Equal(got, want) {
		t.Errorf("Existing() = %v, %v; want %v", got, err, want)
	}
	if got, err := repo.Existing(t.Context(), []string{"test/not-a-uuid"}); err != nil || len(got) != 0 {
		t.Errorf("Existing() of no UUID = %v, %v; want none", got, err)
	}
}

// The app role, in any scope, deletes no rejected file by the purger's query: it is the retention role's.
func TestPostgres_TheAppRoleCannotPurge(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	s := seedDB(t, owner)
	b := s.file("files", 10, 1)
	b.Status, b.ScannedAt = binary.StatusRejected, now.Add(-48*time.Hour)
	insertAsOwner(t, owner, b)
	err := postgres.NewTransactor(app).WithTx(t.Context(), func(ctx context.Context) error {
		n, err := binary.NewPostgresRepository(app).DeleteRejectedBefore(ctx, now)
		if n != 0 {
			t.Errorf("the app role purged %d files outside any scope", n)
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}
