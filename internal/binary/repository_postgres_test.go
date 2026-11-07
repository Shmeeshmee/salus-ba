package binary_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/binary"
	"github.com/Shmeeshmee/salus/migrations"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/envelope"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

func TestMain(m *testing.M) { pgtest.Main(m, migrations.FS) }

// seededDB is a clinic and a patient with an active consent, an appointment a day ahead and an in-progress
// questionnaire response to it; and a second patient with their own. Synthetic.
type seededDB struct {
	clinic, patient, appt, response string
	other, otherResponse            string
}

func seedDB(t testing.TB, owner *pgxpool.Pool) seededDB {
	t.Helper()
	ctx := t.Context()
	var s seededDB
	var role string
	err := owner.QueryRow(ctx, `WITH o AS (INSERT INTO organization (name, listed) VALUES ('Synthetic clinic', true) RETURNING id),
		p AS (INSERT INTO practitioner (organization_id, given_name, family_name) SELECT id, 'Cy', 'Example' FROM o RETURNING id, organization_id),
		r AS (INSERT INTO practitioner_role (organization_id, practitioner_id) SELECT organization_id, id FROM p RETURNING id, organization_id)
		SELECT organization_id::text, id::text FROM r`).Scan(&s.clinic, &role)
	if err != nil {
		t.Fatal(err)
	}
	var q, series string
	err = owner.QueryRow(ctx, `INSERT INTO questionnaire (organization_id, series_id, version, status, title, published_at)
		VALUES ($1, gen_random_uuid(), 1, 'active', 'Placeholder questionnaire', now()) RETURNING id::text, series_id::text`, s.clinic).Scan(&q, &series)
	if err != nil {
		t.Fatal(err)
	}
	patient := func(day int) (string, string) {
		var p, response string
		err := owner.QueryRow(ctx, `WITH a AS (INSERT INTO account (email, password_hash, role) VALUES (gen_random_uuid()::text || '@example.test', 'h', 'patient') RETURNING id),
			p AS (INSERT INTO patient (account_id, given_name, family_name, birth_date) SELECT id, 'Ada', 'Example', '1990-01-01' FROM a RETURNING id)
			INSERT INTO consent (patient_id, organization_id, status, date_time) SELECT id, $1, 'active', now() - interval '30 days' FROM p
			RETURNING patient_id::text`, s.clinic).Scan(&p)
		if err != nil {
			t.Fatal(err)
		}
		err = owner.QueryRow(ctx, `WITH a AS (INSERT INTO appointment (patient_id, practitioner_role_id, organization_id, slot_id, period)
				VALUES ($1, $2, $3, gen_random_uuid(), tstzrange(now() + $6 * interval '1 day', now() + $6 * interval '1 day' + interval '30 minutes')) RETURNING id, period)
			INSERT INTO questionnaire_response (organization_id, appointment_id, patient_id, practitioner_role_id, questionnaire_id, series_id, source, status, due_at)
			SELECT $3, a.id, $1, $2, $4, $5, 'added', 'in-progress', lower(a.period) FROM a RETURNING id::text`,
			p, role, s.clinic, q, series, day).Scan(&response)
		if err != nil {
			t.Fatal(err)
		}
		return p, response
	}
	s.patient, s.response = patient(1)
	s.other, s.otherResponse = patient(2)
	if err := owner.QueryRow(ctx, `SELECT appointment_id::text FROM questionnaire_response WHERE id = $1`, s.response).Scan(&s.appt); err != nil {
		t.Fatal(err)
	}
	return s
}

// file is a row of the response for the item, as the upload builds it.
func (s seededDB) file(link string, size int64, sum byte) binary.Binary {
	return binary.Binary{
		ID: binaryID(), OrganizationID: s.clinic, PatientID: s.patient, AppointmentID: s.appt, ResponseID: s.response, LinkID: link,
		ContentType: "application/pdf", Title: title, KEKID: "test-1", SHA1: bytes.Repeat([]byte{sum}, 20),
		WrappedKey: bytes.Repeat([]byte{1}, 60), Size: size, SegmentSize: envelope.SegmentSize,
	}
}

var ids struct {
	sync.Mutex
	n int
}

// binaryID is a fresh synthetic UUID.
func binaryID() string {
	ids.Lock()
	defer ids.Unlock()
	ids.n++
	return fmt.Sprintf("b1000000-0000-4000-8000-%012d", ids.n)
}

// in runs fn on the repository in a transaction of its own, scoped by scope.
func in(t *testing.T, app *pgxpool.Pool, scope func(ctx context.Context, r binary.Repository) error, fn func(ctx context.Context, r binary.Repository) error) error {
	t.Helper()
	r := binary.NewPostgresRepository(app)
	return postgres.NewTransactor(app).WithTx(t.Context(), func(ctx context.Context) error {
		if err := scope(ctx, r); err != nil {
			return err
		}
		return fn(ctx, r)
	})
}

func asPatient(id string) func(context.Context, binary.Repository) error {
	return func(ctx context.Context, r binary.Repository) error { return r.SetPatient(ctx, id) }
}

func asClinic(id string) func(context.Context, binary.Repository) error {
	return func(ctx context.Context, r binary.Repository) error { return r.SetClinic(ctx, id) }
}

func asJob(response, org string) func(context.Context, binary.Repository) error {
	return func(ctx context.Context, r binary.Repository) error { return r.SetJob(ctx, response, org) }
}

func TestPostgres_ThePatientInsertsAndReadsTheirFilesAndEachScopeSeesWhatItMay(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	s := seedDB(t, owner)
	at := time.Date(2030, 1, 6, 9, 0, 0, 123000, time.UTC)
	b := s.file("files", 1234, 9)
	var stored binary.Binary
	err := in(t, app, asPatient(s.patient), func(ctx context.Context, r binary.Repository) error {
		var err error
		stored, err = r.Insert(ctx, b, at)
		return err
	})
	if err != nil {
		t.Fatalf("Insert() = %v", err)
	}
	want := b
	want.Status, want.CreatedAt, want.UpdatedAt = binary.StatusQuarantined, at, at
	if !equalBinary(stored, want) {
		t.Errorf("Insert() = %+v,\nwant %+v", stored, want)
	}
	for name, c := range map[string]struct {
		scope func(context.Context, binary.Repository) error
		sees  bool
	}{
		"the patient": {asPatient(s.patient), true}, "another patient": {asPatient(s.other), false},
		"the clinic": {asClinic(s.clinic), true}, "the job scope of the response": {asJob(s.response, s.clinic), true},
		"the job scope of another response": {asJob(s.otherResponse, s.clinic), false},
	} {
		err := in(t, app, c.scope, func(ctx context.Context, r binary.Repository) error {
			got, err := r.ByID(ctx, b.ID)
			switch {
			case c.sees && (err != nil || !equalBinary(got, want)):
				t.Errorf("%s: ByID() = %+v, %v; want the file", name, got, err)
			case !c.sees && !errors.Is(err, binary.ErrNotFound):
				t.Errorf("%s: ByID() = %v, want ErrNotFound", name, err)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	err = in(t, app, asPatient(s.patient), func(ctx context.Context, r binary.Repository) error {
		if _, err := r.ByID(ctx, "not-a-uuid"); !errors.Is(err, binary.ErrNotFound) {
			t.Errorf("ByID(not a UUID) = %v, want ErrNotFound", err)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func equalBinary(a, b binary.Binary) bool {
	return a.ID == b.ID && a.OrganizationID == b.OrganizationID && a.PatientID == b.PatientID && a.AppointmentID == b.AppointmentID &&
		a.ResponseID == b.ResponseID && a.LinkID == b.LinkID && a.ContentType == b.ContentType && a.Title == b.Title && a.KEKID == b.KEKID &&
		a.Status == b.Status && bytes.Equal(a.SHA1, b.SHA1) && bytes.Equal(a.WrappedKey, b.WrappedKey) && a.Size == b.Size &&
		a.SegmentSize == b.SegmentSize && a.CreatedAt.Equal(b.CreatedAt) && a.UpdatedAt.Equal(b.UpdatedAt) && a.ScannedAt.Equal(b.ScannedAt)
}

func TestPostgres_RowLevelSecurityRefusesAnInsertForAnotherPatient(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	s := seedDB(t, owner)
	err := in(t, app, asPatient(s.other), func(ctx context.Context, r binary.Repository) error {
		_, err := r.Insert(ctx, s.file("files", 10, 1), now)
		return err
	})
	if err == nil {
		t.Error("Insert() of another patient's file in the patient's scope = nil")
	}
}

func TestPostgres_OfItemAndResponseSizeCountEveryFileButTheRejected(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	s := seedDB(t, owner)
	at := time.Date(2030, 1, 6, 9, 0, 0, 0, time.UTC)
	first, second, other, rejected := s.file("files", 100, 1), s.file("files", 200, 2), s.file("other", 400, 3), s.file("files", 800, 4)
	err := in(t, app, asPatient(s.patient), func(ctx context.Context, r binary.Repository) error {
		for i, b := range []binary.Binary{second, first, other, rejected} {
			if _, err := r.Insert(ctx, b, at.Add(-time.Duration(i)*time.Minute)); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := owner.Exec(t.Context(), `UPDATE "binary" SET status = 'rejected', scanned_at = now(), wrapped_key = NULL WHERE id = $1`, rejected.ID); err != nil {
		t.Fatal(err)
	}
	err = in(t, app, asPatient(s.patient), func(ctx context.Context, r binary.Repository) error {
		files, err := r.OfItem(ctx, s.response, "files")
		if err != nil {
			return err
		}
		var got []string
		for _, f := range files {
			got = append(got, f.ID)
		}
		if want := []string{first.ID, second.ID}; !slices.Equal(got, want) {
			t.Errorf("OfItem() = %v, want %v (oldest first, the rejected left out)", got, want)
		}
		n, err := r.ResponseSize(ctx, s.response)
		if n != 700 || err != nil {
			t.Errorf("ResponseSize() = %d, %v; want 700", n, err)
		}
		if n, err := r.ResponseSize(ctx, s.otherResponse); n != 0 || err != nil {
			t.Errorf("ResponseSize() of a response without files = %d, %v; want 0", n, err)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestPostgres_OnlyThePatientDeletesTheirFile(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	s := seedDB(t, owner)
	b := s.file("files", 10, 1)
	if err := in(t, app, asPatient(s.patient), func(ctx context.Context, r binary.Repository) error {
		_, err := r.Insert(ctx, b, now)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	for name, c := range map[string]struct {
		scope   func(context.Context, binary.Repository) error
		deletes bool
	}{"the clinic": {asClinic(s.clinic), false}, "another patient": {asPatient(s.other), false}, "the patient": {asPatient(s.patient), true}} {
		err := in(t, app, c.scope, func(ctx context.Context, r binary.Repository) error {
			deleted, err := r.Delete(ctx, b.ID)
			if deleted != c.deletes || err != nil {
				t.Errorf("%s: Delete() = %v, %v; want %v", name, deleted, err, c.deletes)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := in(t, app, asPatient(s.patient), func(ctx context.Context, r binary.Repository) error {
		if deleted, err := r.Delete(ctx, "not-a-uuid"); deleted || err != nil {
			t.Errorf("Delete(not a UUID) = %v, %v", deleted, err)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestPostgres_TheJobScopeDeletesItsResponsesFile(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	s := seedDB(t, owner)
	b := s.file("files", 10, 1)
	if err := in(t, app, asPatient(s.patient), func(ctx context.Context, r binary.Repository) error {
		_, err := r.Insert(ctx, b, now)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		scope   func(context.Context, binary.Repository) error
		deletes bool
	}{{asJob(s.otherResponse, s.clinic), false}, {asJob(s.response, s.clinic), true}} {
		err := in(t, app, c.scope, func(ctx context.Context, r binary.Repository) error {
			if deleted, err := r.Delete(ctx, b.ID); deleted != c.deletes || err != nil {
				t.Errorf("Delete() in the job scope = %v, %v; want %v", deleted, err, c.deletes)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

// Two uploads to one response that would each fit but together pass the limit: the advisory lock makes the
// second commit see the first, so one of them is refused (real database and store, a fake holder).
func TestUpload_TwoConcurrentUploadsCannotTogetherPassTheLimits(t *testing.T) {
	for name, limit := range map[string]struct {
		want  error
		max   int
		taken int64 // bytes already on another item
	}{
		"the item's files":   {want: binary.ErrTooManyFiles, max: 1},
		"the response bytes": {want: binary.ErrResponseFull, max: 10, taken: binary.MaxResponseSize - 3*int64(len(pdf()))/2},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			owner, app := pgtest.NewAppDatabase(t)
			s := seedDB(t, owner)
			if limit.taken > 0 {
				if err := in(t, app, asPatient(s.patient), func(ctx context.Context, r binary.Repository) error {
					_, err := r.Insert(ctx, s.file("other", limit.taken, 7), now)
					return err
				}); err != nil {
					t.Fatal(err)
				}
			}
			f := newFixture(t)
			f.holder.target = binary.Target{OrganizationID: s.clinic, PatientID: s.patient, AppointmentID: s.appt, Accept: allKinds, MaxFiles: limit.max}
			gate := &gateHolder{fakeHolder: f.holder, response: s.response, prechecked: make(chan struct{}), committing: make(chan struct{})}
			tx := postgres.NewTransactor(app)
			svc := binary.NewService(binary.Ports{
				Repo: binary.NewPostgresRepository(app), Store: f.store, Holder: gate, Tx: tx, Audit: f.audit,
				Jobs: f.jobs, Clock: clock.Fixed{T: now}, KEK: f.kek,
			})
			c := binary.Caller{AccountID: accountA, PatientID: s.patient}
			errs := make(chan error, 2)
			for i := range 2 {
				body := pdf()
				body[len(body)-2] = byte('0' + i) // two different files of the same size
				go func() {
					_, _, err := svc.Upload(t.Context(), c, binary.Upload{Body: bytes.NewReader(body), ResponseID: s.response, LinkID: "files", Title: title, Size: int64(len(body))})
					errs <- err
				}()
			}
			var refused, ok int
			for range 2 {
				switch err := <-errs; {
				case err == nil:
					ok++
				case errors.Is(err, limit.want):
					refused++
				default:
					t.Errorf("Upload() = %v", err)
				}
			}
			var n int
			if err := owner.QueryRow(t.Context(), `SELECT count(*) FROM "binary" WHERE link_id = 'files'`).Scan(&n); err != nil {
				t.Fatal(err)
			}
			if ok != 1 || refused != 1 || n != 1 {
				t.Errorf("%d stored and %d refused, %d rows; want one of each and one row", ok, refused, n)
			}
		})
	}
}

// gateHolder holds both uploads at each of their target checks until both have reached it, so that both
// pass the first checks and their commits overlap.
type gateHolder struct {
	*fakeHolder
	prechecked chan struct{}
	committing chan struct{}
	response   string
	mu         sync.Mutex
	calls      int
}

func (g *gateHolder) UploadTarget(ctx context.Context, c binary.Caller, responseID, linkID string) (binary.Target, error) {
	g.mu.Lock()
	g.calls++
	wait := g.prechecked
	switch g.calls {
	case 2:
		close(g.prechecked)
	case 3:
		wait = g.committing
	case 4:
		close(g.committing)
		wait = g.committing
	}
	g.mu.Unlock()
	select {
	case <-wait:
	case <-time.After(10 * time.Second):
	}
	if responseID != g.response {
		return binary.Target{}, binary.ErrNotFound
	}
	return g.target, nil
}

func (g *gateHolder) Attach(ctx context.Context, responseID string, a binary.Attachment) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.fakeHolder.Attach(ctx, responseID, a)
}
