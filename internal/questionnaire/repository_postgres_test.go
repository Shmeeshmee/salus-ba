package questionnaire_test

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/questionnaire"
	"github.com/Shmeeshmee/salus/migrations"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

func TestMain(m *testing.M) { pgtest.Main(m, migrations.FS) }

// seed creates, as the owner, clinics A and B with roles A1, A2 and B, series A in clinic A (version 1
// active, version 2 a draft) and series B in clinic B (version 1 active).
func seed(t testing.TB, owner *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	if _, err := owner.Exec(ctx, `INSERT INTO organization (id, name) VALUES ($1, 'Synthetic clinic A'), ($2, 'Synthetic clinic B')`, clinicA, clinicB); err != nil {
		t.Fatal(err)
	}
	for _, r := range []struct{ id, org, practitioner string }{{roleA1, clinicA, practitioner1}, {roleA2, clinicA, practitioner2}, {roleB, clinicB, "b3000000-0000-4000-8000-0000000000b1"}} {
		if _, err := owner.Exec(ctx, `WITH p AS (INSERT INTO practitioner (id, organization_id, given_name, family_name)
			VALUES ($3, $2, 'Ada', 'Example') RETURNING id, organization_id)
			INSERT INTO practitioner_role (id, organization_id, practitioner_id) SELECT $1, organization_id, id FROM p`, r.id, r.org, r.practitioner); err != nil {
			t.Fatal(err)
		}
	}
	const stored = `[{"linkId": "q1", "text": "Placeholder question", "type": "boolean", "required": true}]`
	if _, err := owner.Exec(ctx, `INSERT INTO questionnaire (id, series_id, organization_id, version, status, title, items, published_at, updated_at, created_at)
		VALUES ($1, $2, $3, 1, 'active', 'Placeholder A', $7, '2026-11-01T08:00:00Z', '2026-11-01T08:00:00Z', '2026-11-01T07:00:00Z'),
		       ($4, $2, $3, 2, 'draft', 'Placeholder A', $7, NULL, '2026-11-02T08:00:00Z', '2026-11-02T07:00:00Z'),
		       ($5, $6, $8, 1, 'active', 'Placeholder B', '[]', '2026-11-01T08:00:00Z', '2026-11-01T08:00:00Z', '2026-11-01T07:00:00Z')`,
		activeA, seriesA, clinicA, draftA, activeB, seriesB, stored, clinicB); err != nil {
		t.Fatal(err)
	}
}

// in runs fn in a transaction on app in clinic org's scope.
func in(t testing.TB, app *pgxpool.Pool, repo questionnaire.Repository, org string, fn func(ctx context.Context) error) error {
	t.Helper()
	return postgres.NewTransactor(app).WithTx(context.Background(), func(ctx context.Context) error {
		if err := repo.SetOrganization(ctx, org); err != nil {
			return err
		}
		return fn(ctx)
	})
}

var activeARow = questionnaire.Questionnaire{
	ID: activeA, SeriesID: seriesA, OrganizationID: clinicA, Version: 1, VersionID: 1, Status: questionnaire.StatusActive,
	Title: "Placeholder A", Items: []questionnaire.Item{{LinkID: "q1", Text: "Placeholder question", Type: questionnaire.ItemBoolean, Required: true}},
	LastUpdated: time.Date(2026, 11, 1, 8, 0, 0, 0, time.UTC), PublishedAt: time.Date(2026, 11, 1, 8, 0, 0, 0, time.UTC),
}

func equal(a, b questionnaire.Questionnaire) bool {
	return a.ID == b.ID && a.SeriesID == b.SeriesID && a.OrganizationID == b.OrganizationID && a.Version == b.Version &&
		a.VersionID == b.VersionID && a.Status == b.Status && a.Title == b.Title && a.Description == b.Description &&
		slices.EqualFunc(a.Items, b.Items, equalItem) && a.LastUpdated.Equal(b.LastUpdated) &&
		a.PublishedAt.Equal(b.PublishedAt) && a.RetiredAt.Equal(b.RetiredAt)
}

func equalItem(a, b questionnaire.Item) bool {
	return a.LinkID == b.LinkID && a.Text == b.Text && a.Type == b.Type && a.Help == b.Help && a.Required == b.Required &&
		a.Repeats == b.Repeats && slices.Equal(a.Options, b.Options) && slices.EqualFunc(a.Items, b.Items, equalItem) &&
		slices.Equal(a.Accept, b.Accept) && a.MaxFiles == b.MaxFiles &&
		((a.EnableWhen == nil) == (b.EnableWhen == nil)) &&
		(a.EnableWhen == nil || (a.EnableWhen.Question == b.EnableWhen.Question && a.EnableWhen.String == b.EnableWhen.String &&
			(a.EnableWhen.Boolean == nil) == (b.EnableWhen.Boolean == nil) && (a.EnableWhen.Boolean == nil || *a.EnableWhen.Boolean == *b.EnableWhen.Boolean)))
}

func TestPostgres_ReadsSeeOnlyTheClinicSet(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seed(t, owner)
	repo := questionnaire.NewPostgresRepository(app)

	for name, read := range map[string]func(ctx context.Context, id string) (questionnaire.Questionnaire, error){"ByID": repo.ByID, "Lock": repo.Lock} {
		var got questionnaire.Questionnaire
		err := in(t, app, repo, clinicA, func(ctx context.Context) error {
			var err error
			got, err = read(ctx, activeA)
			return err
		})
		if err != nil || !equal(got, activeARow) {
			t.Errorf("%s in clinic A = %+v, %v;\nwant %+v", name, got, err, activeARow)
		}
		for _, id := range []string{activeB, missing, "x"} {
			err := in(t, app, repo, clinicA, func(ctx context.Context) error { _, err := read(ctx, id); return err })
			if !errors.Is(err, questionnaire.ErrNotFound) {
				t.Errorf("%s(%s) in clinic A = %v, want ErrNotFound", name, id, err)
			}
		}
	}
}

func TestPostgres_SearchesByFilterInTheClinicSet(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seed(t, owner)
	repo := questionnaire.NewPostgresRepository(app)

	for _, tt := range []struct {
		f    questionnaire.Filter
		want []string
	}{
		{f: questionnaire.Filter{}, want: []string{activeA, draftA}},
		{f: questionnaire.Filter{SeriesID: seriesA}, want: []string{activeA, draftA}},
		{f: questionnaire.Filter{Status: questionnaire.StatusDraft}, want: []string{draftA}},
		{f: questionnaire.Filter{SeriesID: seriesA, Version: 1}, want: []string{activeA}},
		{f: questionnaire.Filter{SeriesID: seriesB}, want: nil},
		{f: questionnaire.Filter{SeriesID: "x"}, want: nil},
	} {
		var got []questionnaire.Questionnaire
		err := in(t, app, repo, clinicA, func(ctx context.Context) error {
			var err error
			got, err = repo.Search(ctx, tt.f)
			return err
		})
		ids := make([]string, 0, len(got))
		for _, q := range got {
			ids = append(ids, q.ID)
		}
		if err != nil || !slices.Equal(ids, tt.want) {
			t.Errorf("Search(%+v) = %v, %v; want %v", tt.f, ids, err, tt.want)
		}
	}
}

func TestPostgres_RunsTheLifecycle(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seed(t, owner)
	repo := questionnaire.NewPostgresRepository(app)
	at := time.Date(2026, 11, 6, 10, 0, 0, 0, time.UTC)
	no := false
	tree := []questionnaire.Item{
		{LinkID: "b", Text: "Placeholder", Type: questionnaire.ItemBoolean},
		{LinkID: "s", Text: "Placeholder section", Type: questionnaire.ItemGroup, Help: "Placeholder help",
			EnableWhen: &questionnaire.Condition{Question: "b", Boolean: &no},
			Items: []questionnaire.Item{{LinkID: "c", Text: "Placeholder", Type: questionnaire.ItemChoice, Repeats: true, Required: true,
				Options: []questionnaire.Option{{Value: "One"}, {Value: "Two"}}}}},
		{LinkID: "a", Text: "Placeholder file upload", Type: questionnaire.ItemAttachment, Required: true,
			Accept: []questionnaire.FileKind{questionnaire.FileDocument, questionnaire.FilePhoto}, MaxFiles: 10},
	}

	var created, updated, published, revised, retired questionnaire.Questionnaire
	err := in(t, app, repo, clinicA, func(ctx context.Context) error {
		var err error
		if created, err = repo.Create(ctx, questionnaire.Questionnaire{OrganizationID: clinicA, Title: "Placeholder", Items: tree}, at); err != nil {
			return err
		}
		created.VersionID = 1
		if updated, err = repo.UpdateDraft(ctx, questionnaire.Questionnaire{ID: created.ID, VersionID: 1, Title: "Placeholder 2", Description: "D", Items: tree}, at.Add(time.Minute)); err != nil {
			return err
		}
		if published, err = repo.Publish(ctx, created.ID, at.Add(2*time.Minute)); err != nil {
			return err
		}
		if revised, err = repo.CreateVersion(ctx, published, at.Add(3*time.Minute)); err != nil {
			return err
		}
		if err = repo.RetireActive(ctx, created.SeriesID, at.Add(4*time.Minute)); err != nil {
			return err
		}
		if published, err = repo.Publish(ctx, revised.ID, at.Add(4*time.Minute)); err != nil {
			return err
		}
		retired, err = repo.Retire(ctx, published.ID, at.Add(5*time.Minute))
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.ID == "" || created.SeriesID == "" || created.SeriesID == seriesA || created.Version != 1 || created.Status != questionnaire.StatusDraft ||
		!created.LastUpdated.Equal(at) || !slices.EqualFunc(created.Items, tree, equalItem) {
		t.Errorf("Create() = %+v; want version 1 draft of a new series with the tree", created)
	}
	if updated.VersionID != 2 || updated.Title != "Placeholder 2" || updated.Description != "D" || !updated.LastUpdated.Equal(at.Add(time.Minute)) {
		t.Errorf("UpdateDraft() = %+v; want the content at version 2", updated)
	}
	if revised.SeriesID != created.SeriesID || revised.Version != 2 || revised.Status != questionnaire.StatusDraft || revised.Title != "Placeholder 2" ||
		!slices.EqualFunc(revised.Items, tree, equalItem) || revised.VersionID != 1 {
		t.Errorf("CreateVersion() = %+v; want version 2, a draft copy", revised)
	}
	if retired.ID != revised.ID || retired.Status != questionnaire.StatusRetired || !retired.PublishedAt.Equal(at.Add(4*time.Minute)) ||
		!retired.RetiredAt.Equal(at.Add(5*time.Minute)) || retired.VersionID != 3 {
		t.Errorf("Retire() = %+v; want version 2 retired, published and retired at their times, at version 3", retired)
	}
	var v1 questionnaire.Questionnaire
	if err := in(t, app, repo, clinicA, func(ctx context.Context) error { var err error; v1, err = repo.ByID(ctx, created.ID); return err }); err != nil ||
		v1.Status != questionnaire.StatusRetired || !v1.RetiredAt.Equal(at.Add(4*time.Minute)) {
		t.Errorf("version 1 after RetireActive = %+v, %v; want retired", v1, err)
	}
}

func TestPostgres_RefusesWritesOutsideTheirState(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seed(t, owner)
	repo := questionnaire.NewPostgresRepository(app)
	at := time.Date(2026, 11, 6, 10, 0, 0, 0, time.UTC)

	for name, tt := range map[string]struct {
		write func(ctx context.Context) error
		want  error
	}{
		"a stale draft update": {want: questionnaire.ErrConflict, write: func(ctx context.Context) error {
			_, err := repo.UpdateDraft(ctx, questionnaire.Questionnaire{ID: draftA, VersionID: 9, Title: "x"}, at)
			return err
		}},
		"an update of an active version": {want: questionnaire.ErrConflict, write: func(ctx context.Context) error {
			_, err := repo.UpdateDraft(ctx, questionnaire.Questionnaire{ID: activeA, VersionID: 1, Title: "x"}, at)
			return err
		}},
		"an update of another clinic's": {want: questionnaire.ErrConflict, write: func(ctx context.Context) error {
			_, err := repo.UpdateDraft(ctx, questionnaire.Questionnaire{ID: activeB, VersionID: 1, Title: "x"}, at)
			return err
		}},
		"an update of not a UUID": {want: questionnaire.ErrConflict, write: func(ctx context.Context) error {
			_, err := repo.UpdateDraft(ctx, questionnaire.Questionnaire{ID: "x", VersionID: 1, Title: "x"}, at)
			return err
		}},
		"a second draft": {want: questionnaire.ErrDraftExists, write: func(ctx context.Context) error {
			_, err := repo.CreateVersion(ctx, activeARow, at)
			return err
		}},
		"publish an active version": {want: questionnaire.ErrConflict, write: func(ctx context.Context) error {
			_, err := repo.Publish(ctx, activeA, at)
			return err
		}},
		"retire a draft": {want: questionnaire.ErrConflict, write: func(ctx context.Context) error {
			_, err := repo.Retire(ctx, draftA, at)
			return err
		}},
		"retire not a UUID": {want: questionnaire.ErrConflict, write: func(ctx context.Context) error {
			_, err := repo.Retire(ctx, "x", at)
			return err
		}},
		"delete an active version": {want: questionnaire.ErrNotFound, write: func(ctx context.Context) error { return repo.DeleteDraft(ctx, activeA) }},
		"delete another clinic's":  {want: questionnaire.ErrNotFound, write: func(ctx context.Context) error { return repo.DeleteDraft(ctx, activeB) }},
		"delete not a UUID":        {want: questionnaire.ErrNotFound, write: func(ctx context.Context) error { return repo.DeleteDraft(ctx, "x") }},
	} {
		if err := in(t, app, repo, clinicA, tt.write); !errors.Is(err, tt.want) {
			t.Errorf("%s = %v, want %v", name, err, tt.want)
		}
	}
	if err := in(t, app, repo, clinicA, func(ctx context.Context) error { return repo.DeleteDraft(ctx, draftA) }); err != nil {
		t.Errorf("DeleteDraft() of the draft = %v", err)
	}
}

func TestPostgres_StoresOneDefaultPerRole(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seed(t, owner)
	repo := questionnaire.NewPostgresRepository(app)
	at := time.Date(2026, 11, 6, 10, 0, 0, 0, time.UTC)

	var first, second, read questionnaire.Default
	err := in(t, app, repo, clinicA, func(ctx context.Context) error {
		if _, err := repo.Default(ctx, roleA1); !errors.Is(err, questionnaire.ErrNotFound) {
			return errors.New("a default before any was set")
		}
		var err error
		if first, err = repo.SetDefault(ctx, questionnaire.Default{PractitionerRoleID: roleA1, OrganizationID: clinicA, SeriesID: seriesA}, at); err != nil {
			return err
		}
		if second, err = repo.SetDefault(ctx, questionnaire.Default{PractitionerRoleID: roleA1, OrganizationID: clinicA, SeriesID: seriesB}, at.Add(time.Minute)); err != nil {
			return err
		}
		read, err = repo.Default(ctx, roleA1)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if first.SeriesID != seriesA || first.VersionID != 1 || second.SeriesID != seriesB || second.VersionID != 2 ||
		read != second || !read.LastUpdated.Equal(at.Add(time.Minute)) || read.OrganizationID != clinicA {
		t.Errorf("SetDefault twice = %+v, %+v, read %+v; want the second series at version 2", first, second, read)
	}
	err = in(t, app, repo, clinicA, func(ctx context.Context) error {
		if err := repo.ClearDefault(ctx, roleA1); err != nil {
			return err
		}
		if err := repo.ClearDefault(ctx, "x"); err != nil {
			return err
		}
		_, err := repo.Default(ctx, roleA1)
		return err
	})
	if !errors.Is(err, questionnaire.ErrNotFound) {
		t.Errorf("Default() after ClearDefault = %v, want ErrNotFound", err)
	}
	if err := in(t, app, repo, clinicA, func(ctx context.Context) error {
		_, err := repo.SetDefault(ctx, questionnaire.Default{PractitionerRoleID: roleB, OrganizationID: clinicB, SeriesID: seriesB}, at)
		return err
	}); err == nil {
		t.Error("SetDefault() of clinic B's role in clinic A's scope = nil, want a row-level security error")
	}
	if err := in(t, app, repo, clinicA, func(ctx context.Context) error { _, err := repo.Default(ctx, "x"); return err }); !errors.Is(err, questionnaire.ErrNotFound) {
		t.Errorf("Default(not a UUID) = %v, want ErrNotFound", err)
	}
}

// countingAuditor counts the events recorded inside a database transaction; safe for concurrent use.
type countingAuditor struct {
	events []auditevent.AuditEvent
	mu     sync.Mutex
}

func (a *countingAuditor) Record(ctx context.Context, e auditevent.AuditEvent) error {
	if _, ok := postgres.Tx(ctx); !ok {
		return errors.New("audit outside a transaction")
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.events = append(a.events, e)
	return nil
}

// realService is the Service on the database as salus_app (the audit counted, not stored: auditevent's own
// tests cover the storage).
func realService(app *pgxpool.Pool) (*questionnaire.Service, *countingAuditor) {
	audit := &countingAuditor{}
	return questionnaire.NewService(questionnaire.Ports{
		Repo: questionnaire.NewPostgresRepository(app), Tx: postgres.NewTransactor(app), Clock: clock.Fixed{T: now},
		Audit: audit, Roles: fakeRoles{}, Clinics: &fakeClinics{},
	}), audit
}

func statuses(t *testing.T, owner *pgxpool.Pool, series string) map[string]string {
	t.Helper()
	rows, err := owner.Query(t.Context(), "SELECT id::text, status FROM questionnaire WHERE series_id = $1", series)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var id, status string
		if err := rows.Scan(&id, &status); err != nil {
			t.Fatal(err)
		}
		out[id] = status
	}
	return out
}

func TestPublish_ARollbackLeavesTheActiveVersionActive(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seed(t, owner)
	svc, _ := realService(app)
	failure := errors.New("later step failed")

	err := postgres.NewTransactor(app).WithTx(t.Context(), func(ctx context.Context) error {
		if _, err := svc.Publish(ctx, adminA, draftA, 1); err != nil {
			return err
		}
		return failure
	})
	if !errors.Is(err, failure) {
		t.Fatalf("the transaction = %v, want the failure", err)
	}
	if got := statuses(t, owner, seriesA); got[activeA] != "active" || got[draftA] != "draft" {
		t.Errorf("after the rollback: %v; want version 1 active and version 2 a draft", got)
	}
}

func TestPublish_TwoConcurrentPublishesLeaveOneActiveVersion(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seed(t, owner)
	svc, audit := realService(app)

	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i := range errs {
		wg.Go(func() { _, errs[i] = svc.Publish(context.Background(), adminA, draftA, 1) })
	}
	wg.Wait()
	ok := 0
	for _, err := range errs {
		switch {
		case err == nil:
			ok++
		case !errors.Is(err, questionnaire.ErrNotDraft):
			t.Errorf("the losing publish = %v, want ErrNotDraft", err)
		}
	}
	if ok != 1 {
		t.Errorf("%d publishes succeeded (%v), want 1", ok, errs)
	}
	if got := statuses(t, owner, seriesA); got[activeA] != "retired" || got[draftA] != "active" {
		t.Errorf("after both publishes: %v; want version 1 retired and version 2 active", got)
	}
	if n := len(audit.events); n != 2 {
		t.Errorf("audit events of the publishes: %d, want 2", n)
	}
}

// In a patient's scope a version is shown only while one of the patient's own responses references it.
func TestPostgres_ThePatientScopeShowsOnlyTheVersionsTheirResponsesReference(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seed(t, owner)
	var patient, other string
	for _, id := range []*string{&patient, &other} {
		if err := owner.QueryRow(t.Context(), `WITH a AS (INSERT INTO account (email, password_hash, role) VALUES (gen_random_uuid()::text || '@example.test', 'h', 'patient') RETURNING id)
			INSERT INTO patient (account_id, given_name, family_name, birth_date) SELECT id, 'Ada', 'Example', '1990-01-01' FROM a RETURNING id::text`).Scan(id); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := owner.Exec(t.Context(), `WITH a AS (INSERT INTO appointment (patient_id, practitioner_role_id, organization_id, slot_id, period)
			VALUES ($1, $2, $3, gen_random_uuid(), tstzrange('2030-01-07T09:00:00Z', '2030-01-07T09:30:00Z')) RETURNING id)
		INSERT INTO questionnaire_response (organization_id, appointment_id, patient_id, practitioner_role_id, questionnaire_id, series_id, source, status, due_at)
		SELECT $3, a.id, $1, $2, $4, $5, 'added', 'in-progress', '2030-01-07T09:00:00Z' FROM a`, patient, roleA1, clinicA, activeA, seriesA); err != nil {
		t.Fatal(err)
	}
	repo := questionnaire.NewPostgresRepository(app)
	read := func(patient, id string) (questionnaire.Questionnaire, error) {
		var got questionnaire.Questionnaire
		err := postgres.NewTransactor(app).WithTx(t.Context(), func(ctx context.Context) error {
			if err := repo.SetPatient(ctx, patient); err != nil {
				return err
			}
			var err error
			got, err = repo.ByID(ctx, id)
			return err
		})
		return got, err
	}
	if got, err := read(patient, activeA); err != nil || !equal(got, activeARow) {
		t.Errorf("ByID() of the referenced version = %+v, %v;\nwant %+v", got, err, activeARow)
	}
	for _, c := range []struct{ name, patient, id string }{
		{"another version of the series", patient, draftA}, {"another clinic's", patient, activeB}, {"another patient", other, activeA},
	} {
		if _, err := read(c.patient, c.id); !errors.Is(err, questionnaire.ErrNotFound) {
			t.Errorf("%s: ByID() = %v, want ErrNotFound", c.name, err)
		}
	}
}
