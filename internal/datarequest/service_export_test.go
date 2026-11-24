package datarequest_test

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"io"
	"slices"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/auth"
	"github.com/Shmeeshmee/salus/internal/datarequest"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/envelope"
)

const (
	accountB = "00000000-0000-4000-8000-0000000000b1"
	request1 = "00000000-0000-4000-8000-000000000101"
	request2 = "00000000-0000-4000-8000-000000000102"
)

// fakeExports keeps the exports in memory and shows only the scope's account, as row-level security does.
type fakeExports struct {
	rows      map[string]datarequest.StoredExport
	kinds     map[string]datarequest.Kind
	orgs      map[string]string
	closed    map[string]bool // requests withdrawn or answered
	scope     string
	fulfilled []string
}

func newFakeExports() *fakeExports {
	return &fakeExports{rows: map[string]datarequest.StoredExport{}, kinds: map[string]datarequest.Kind{}, orgs: map[string]string{},
		closed: map[string]bool{}}
}

func (r *fakeExports) SetAccount(_ context.Context, id string) error {
	r.scope = id
	return nil
}

func (r *fakeExports) InsertExport(_ context.Context, x datarequest.StoredExport) (bool, error) {
	if x.AccountID != r.scope {
		return false, errors.New("fakeExports: outside the scope")
	}
	for _, row := range r.rows {
		if row.AccountID == x.AccountID && row.Status == datarequest.ExportQueued {
			return false, nil
		}
	}
	x.Status, x.Kind, x.OrganizationID = datarequest.ExportQueued, r.kinds[x.RequestID], r.orgs[x.RequestID]
	r.rows[x.RequestID] = x
	return true, nil
}

func (r *fakeExports) ExportOf(_ context.Context, id string) (datarequest.StoredExport, error) {
	x, ok := r.rows[id]
	if !ok || x.AccountID != r.scope {
		return datarequest.StoredExport{}, datarequest.ErrNotFound
	}
	return x, nil
}

func (r *fakeExports) MarkExportReady(_ context.Context, x datarequest.StoredExport) (bool, error) {
	cur, ok := r.rows[x.RequestID]
	if !ok || cur.AccountID != r.scope || cur.Status != datarequest.ExportQueued || r.closed[x.RequestID] {
		return false, nil
	}
	x.Status = datarequest.ExportReady
	r.rows[x.RequestID] = x
	return true, nil
}

func (r *fakeExports) MarkExportFailed(_ context.Context, id string) (bool, error) {
	cur, ok := r.rows[id]
	if !ok || cur.AccountID != r.scope || cur.Status != datarequest.ExportQueued {
		return false, nil
	}
	cur.Status = datarequest.ExportFailed
	r.rows[id] = cur
	return true, nil
}

func (r *fakeExports) Fulfil(_ context.Context, id string) (bool, error) {
	if r.rows[id].Status != datarequest.ExportReady || slices.Contains(r.fulfilled, id) {
		return false, nil
	}
	r.fulfilled = append(r.fulfilled, id)
	return true, nil
}

func (r *fakeExports) ExportsDoneBefore(_ context.Context, before time.Time, _ int32) ([]string, error) {
	var out []string
	for id, x := range r.rows {
		if x.Status == datarequest.ExportReady && x.ReadyAt.Before(before) ||
			x.Status != datarequest.ExportReady && x.RequestedAt.Before(before) {
			out = append(out, id)
		}
	}
	slices.Sort(out)
	return out, nil
}

func (r *fakeExports) DeleteExport(_ context.Context, id string) (bool, error) {
	_, ok := r.rows[id]
	delete(r.rows, id)
	return ok, nil
}

type fakeStore struct{ objects map[string][]byte }

func (s *fakeStore) Put(_ context.Context, key string, r io.Reader, size int64) error {
	b, err := io.ReadAll(r)
	if err != nil || int64(len(b)) != size {
		return errors.New("fakeStore: short write")
	}
	s.objects[key] = b
	return nil
}

func (s *fakeStore) GetRange(_ context.Context, key string, from, to int64) (io.ReadCloser, error) {
	b, ok := s.objects[key]
	if !ok {
		return nil, errors.New("fakeStore: no object")
	}
	return io.NopCloser(bytes.NewReader(b[from : to+1])), nil
}

func (s *fakeStore) Delete(_ context.Context, key string) error {
	delete(s.objects, key)
	return nil
}

// fakeSources are the patient's own record and account, and who read them.
type fakeSources struct {
	bundle   []byte
	orgs     []string
	accounts []string
}

func (f *fakeSources) Bundle(_ context.Context, account, org string) ([]byte, error) {
	f.accounts = append(f.accounts, account)
	f.orgs = append(f.orgs, org)
	return f.bundle, nil
}

func (f *fakeSources) Me(_ context.Context, id string) (auth.Account, error) {
	f.accounts = append(f.accounts, id)
	return auth.Account{ID: id, Email: "patient@example.test", Language: "bs", PasswordHash: "secret-hash"}, nil
}

type exportEnv struct {
	svc     *datarequest.Service
	repo    *fakeExports
	store   *fakeStore
	jobs    *fakeJobs
	audit   *fakeAudit
	sources *fakeSources
	clk     *clock.Fixed
}

var now = time.Date(2026, 11, 24, 9, 0, 0, 0, time.UTC)

func newExportEnv(t *testing.T) *exportEnv {
	t.Helper()
	kek, err := envelope.NewKEK("test", bytes.Repeat([]byte{7}, 32))
	if err != nil {
		t.Fatal(err)
	}
	e := &exportEnv{repo: newFakeExports(), store: &fakeStore{objects: map[string][]byte{}}, jobs: &fakeJobs{},
		audit: &fakeAudit{}, sources: &fakeSources{bundle: []byte(`{"resourceType":"Bundle"}` + "\n")}, clk: &clock.Fixed{T: now}}
	e.svc = datarequest.NewService(datarequest.Ports{Clock: e.clk, Export: datarequest.ExportPorts{
		Repo: e.repo, Audit: e.audit, Tx: fakeTx{}, Jobs: e.jobs, Store: e.store, KEK: kek, Records: e.sources, Accounts: e.sources,
	}})
	e.repo.kinds[request1], e.repo.kinds[request2] = datarequest.KindAccess, datarequest.KindPortability
	return e
}

func access(id, account string) datarequest.Request {
	return datarequest.Request{ID: id, AccountID: account, FiledBy: account, Kind: datarequest.KindAccess}
}

// Filing an access or portability request in the app queues its export and the build, audited; one export in
// progress per patient; nothing for another kind or for a request the clinic filed for a patient it registered (S4,
// S10).
func TestQueueExport_QueuesOnePerPatientForAccessAndPortability(t *testing.T) {
	t.Parallel()
	e := newExportEnv(t)
	ctx := t.Context()
	if err := e.svc.QueueExport(ctx, access(request1, accountA)); err != nil {
		t.Fatal(err)
	}
	if len(e.jobs.queued) != 1 || e.jobs.queued[0] != (datarequest.ExportArgs{RequestID: request1, AccountID: accountA}) {
		t.Fatalf("queued %+v, want the build of request1 with ids only", e.jobs.queued)
	}
	if ev := e.audit.events; len(ev) != 1 || ev[0].Action != auditevent.ActionCreate || ev[0].AgentAccountID != accountA {
		t.Errorf("audited %+v, want the patient's create", ev)
	}
	second := access(request2, accountA)
	second.Kind = datarequest.KindPortability
	if err := e.svc.QueueExport(ctx, second); !errors.Is(err, datarequest.ErrExportInProgress) {
		t.Errorf("a second while one is queued: %v, want ErrExportInProgress", err)
	}
	other := []datarequest.Request{
		{ID: request2, AccountID: accountA, Kind: datarequest.KindErasure},
		{ID: request2, PatientID: "p", OrganizationID: clinicA, Kind: datarequest.KindAccess}, // S10: no account
	}
	for _, r := range other {
		if err := e.svc.QueueExport(ctx, r); err != nil {
			t.Errorf("QueueExport(%+v) = %v, want nothing", r, err)
		}
	}
	if len(e.jobs.queued) != 1 {
		t.Errorf("queued %d builds, want 1", len(e.jobs.queued))
	}
}

// files are the names and contents of a zip.
func files(t *testing.T, r io.Reader) map[string]string {
	t.Helper()
	b, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]string{}
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(rc)
		_ = rc.Close()
		out[f.Name] = string(body)
	}
	return out
}

// The job builds the archive (the record and, for the account target, the account data, never its credentials),
// stores it encrypted, makes it ready for 7 days and fulfils the request; the patient reads its status and downloads
// it, both audited; once expired it is not served.
func TestBuildExport_StoresItEncryptedFulfilsAndServesItForSevenDays(t *testing.T) {
	t.Parallel()
	e := newExportEnv(t)
	ctx := t.Context()
	if err := e.svc.QueueExport(ctx, access(request1, accountA)); err != nil {
		t.Fatal(err)
	}
	if got, err := e.svc.Export(ctx, datarequest.Patient{AccountID: accountA}, request1); err != nil || got.Status != datarequest.ExportQueued {
		t.Fatalf("Export before the build = %+v, %v; want queued", got, err)
	}
	if _, _, err := e.svc.ExportArchive(ctx, datarequest.Patient{AccountID: accountA}, request1); !errors.Is(err, datarequest.ErrNotFound) {
		t.Errorf("download before the build: %v, want ErrNotFound", err)
	}
	if err := e.svc.BuildExport(ctx, e.jobs.queued[0]); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(e.repo.fulfilled, []string{request1}) {
		t.Errorf("fulfilled %v, want request1", e.repo.fulfilled)
	}
	if obj := e.store.objects[datarequest.ExportObjectKey(request1)]; bytes.Contains(obj, []byte("patient@example.test")) || len(obj) == 0 {
		t.Error("the stored archive is not encrypted")
	}
	got, err := e.svc.Export(ctx, datarequest.Patient{AccountID: accountA}, request1)
	if err != nil || got.Status != datarequest.ExportReady || !got.ExpiresAt.Equal(now.Add(datarequest.ExportWindow)) {
		t.Fatalf("Export = %+v, %v; want ready until 7 days on", got, err)
	}
	body, size, err := e.svc.ExportArchive(ctx, datarequest.Patient{AccountID: accountA}, request1)
	if err != nil || size == 0 {
		t.Fatalf("ExportArchive = %d bytes, %v", size, err)
	}
	fs := files(t, body)
	_ = body.Close()
	if fs["record.json"] != string(e.sources.bundle) || fs["manifest.txt"] == "" {
		t.Errorf("archive %v, want the record and the manifest", fs)
	}
	if a := fs["account.json"]; !bytes.Contains([]byte(a), []byte("patient@example.test")) || bytes.Contains([]byte(a), []byte("secret-hash")) {
		t.Errorf("account.json %q, want the e-mail and no credential", a)
	}
	if !slices.Equal(e.sources.orgs, []string{""}) {
		t.Errorf("record read for %q, want every clinic", e.sources.orgs)
	}
	e.clk.T = now.Add(datarequest.ExportWindow)
	if got, _ := e.svc.Export(ctx, datarequest.Patient{AccountID: accountA}, request1); got.Status != datarequest.ExportExpired {
		t.Errorf("after 7 days: %q, want expired", got.Status)
	}
	if _, _, err := e.svc.ExportArchive(ctx, datarequest.Patient{AccountID: accountA}, request1); !errors.Is(err, datarequest.ErrNotFound) {
		t.Errorf("download after 7 days: %v, want ErrNotFound", err)
	}
	var reads, jobs int
	for _, ev := range e.audit.events {
		switch {
		case ev.Action == auditevent.ActionRead && ev.AgentAccountID == accountA && ev.Purpose.Code == auditevent.CodePATRQT:
			reads++
		case ev.AgentName == datarequest.AgentExport && ev.Outcome == auditevent.OutcomeSuccess:
			jobs++
		}
	}
	if reads != 4 || jobs != 1 {
		t.Errorf("audited %d reads of the patient's and %d of the job's, want 4 and 1", reads, jobs)
	}
}

// A clinic target's archive is that clinic's record only, without the account data; a build of an export that is not
// queued does nothing.
func TestBuildExport_ClinicTargetAndNotQueued(t *testing.T) {
	t.Parallel()
	e := newExportEnv(t)
	ctx := t.Context()
	e.repo.orgs[request1] = clinicA
	if err := e.svc.QueueExport(ctx, access(request1, accountA)); err != nil {
		t.Fatal(err)
	}
	args := e.jobs.queued[0]
	if err := e.svc.BuildExport(ctx, args); err != nil {
		t.Fatal(err)
	}
	body, _, err := e.svc.ExportArchive(ctx, datarequest.Patient{AccountID: accountA}, request1)
	if err != nil {
		t.Fatal(err)
	}
	fs := files(t, body)
	_ = body.Close()
	if _, ok := fs["account.json"]; ok || !slices.Equal(e.sources.orgs, []string{clinicA}) {
		t.Errorf("archive %v read for %v, want the clinic's record only", fs, e.sources.orgs)
	}
	if err := e.svc.BuildExport(ctx, args); err != nil || len(e.sources.orgs) != 1 {
		t.Errorf("a second build: %v, read %d times; want nothing", err, len(e.sources.orgs))
	}
	if err := e.svc.BuildExport(ctx, datarequest.ExportArgs{RequestID: request2, AccountID: accountA}); err != nil {
		t.Errorf("a build of no export: %v", err)
	}
}

// Another patient sees nothing of the export; a failed build marks it failed, audited, and fulfils nothing.
func TestExport_OtherPatientNotFoundAndFailure(t *testing.T) {
	t.Parallel()
	e := newExportEnv(t)
	ctx := t.Context()
	if err := e.svc.QueueExport(ctx, access(request1, accountA)); err != nil {
		t.Fatal(err)
	}
	for _, p := range []datarequest.Patient{{AccountID: accountB}, {}} {
		if _, err := e.svc.Export(ctx, p, request1); !errors.Is(err, datarequest.ErrNotFound) {
			t.Errorf("Export as %+v: %v, want ErrNotFound", p, err)
		}
	}
	if err := e.svc.FailExport(ctx, e.jobs.queued[0]); err != nil {
		t.Fatal(err)
	}
	if got, _ := e.svc.Export(ctx, datarequest.Patient{AccountID: accountA}, request1); got.Status != datarequest.ExportFailed {
		t.Errorf("after the failure: %q, want failed", got.Status)
	}
	if len(e.repo.fulfilled) != 0 {
		t.Error("a failed export fulfilled its request")
	}
	last := e.audit.events[len(e.audit.events)-2] // before the status read's
	if last.AgentName != datarequest.AgentExport || last.Outcome != auditevent.OutcomeSeriousFailure {
		t.Errorf("failure audited as %+v", last)
	}
}

// The retention rule deletes the exports ready or failed before the cut-off, their archives first.
func TestExportPurger_DeletesTheArchiveAndTheRow(t *testing.T) {
	t.Parallel()
	e := newExportEnv(t)
	ctx := t.Context()
	if err := e.svc.QueueExport(ctx, access(request1, accountA)); err != nil {
		t.Fatal(err)
	}
	if err := e.svc.BuildExport(ctx, e.jobs.queued[0]); err != nil {
		t.Fatal(err)
	}
	purger := datarequest.NewExportPurger(e.repo, e.store)
	if n, err := purger.Purge(ctx, now); err != nil || n != 0 {
		t.Fatalf("Purge before it was ready = %d, %v; want 0", n, err)
	}
	if n, err := purger.Purge(ctx, now.Add(time.Second)); err != nil || n != 1 {
		t.Fatalf("Purge = %d, %v; want 1", n, err)
	}
	if len(e.store.objects)+len(e.repo.rows) != 0 {
		t.Errorf("left %d objects and %d rows", len(e.store.objects), len(e.repo.rows))
	}
}

// A request withdrawn while its archive was built is never served: the export is failed, nothing fulfilled, and the
// purger deletes its archive with a build that never finished.
func TestBuildExport_WithdrawnMeanwhileIsNeverServed(t *testing.T) {
	t.Parallel()
	e := newExportEnv(t)
	ctx := t.Context()
	if err := e.svc.QueueExport(ctx, access(request1, accountA)); err != nil {
		t.Fatal(err)
	}
	e.repo.closed[request1] = true
	if err := e.svc.BuildExport(ctx, e.jobs.queued[0]); err != nil {
		t.Fatal(err)
	}
	if got, _ := e.svc.Export(ctx, datarequest.Patient{AccountID: accountA}, request1); got.Status != datarequest.ExportFailed || len(e.repo.fulfilled) != 0 {
		t.Errorf("export %q, fulfilled %v; want failed and nothing", got.Status, e.repo.fulfilled)
	}
	if n, err := datarequest.NewExportPurger(e.repo, e.store).Purge(ctx, now.Add(time.Second)); err != nil || n != 1 || len(e.store.objects) != 0 {
		t.Errorf("Purge = %d, %v, %d objects left; want the archive gone", n, err, len(e.store.objects))
	}
}
