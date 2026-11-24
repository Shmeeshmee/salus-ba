package datarequest_test

import (
	"context"
	"slices"
	"strconv"
	"time"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/consent"
	"github.com/Shmeeshmee/salus/internal/datarequest"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/internal/patient"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

// The synthetic ids of the requests' tests.
const (
	accountA   = "00000000-0000-4000-8000-0000000000a1"
	profileA   = "00000000-0000-4000-8000-0000000000b1"
	clinicA    = "00000000-0000-4000-8000-0000000000c1"
	clinicB    = "00000000-0000-4000-8000-0000000000c2"
	adminA     = "00000000-0000-4000-8000-0000000000d1"
	deskA      = "00000000-0000-4000-8000-0000000000d2"
	registered = "00000000-0000-4000-8000-0000000000e1"
)

// fakeRepo is the Repository in memory. Its scope is the last one set; it shows a request as row-level security does.
type fakeRepo struct {
	insertErr error
	requests  map[string]datarequest.Request
	warned    map[string]time.Time
	account   string
	org       string
	changes   []datarequest.Change
	next      int
	deadlines bool
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{requests: map[string]datarequest.Request{}, warned: map[string]time.Time{}}
}

func (r *fakeRepo) SetAccount(_ context.Context, id string) error {
	r.account, r.org, r.deadlines = id, "", false
	return nil
}

func (r *fakeRepo) SetOrganization(_ context.Context, id string) error {
	r.account, r.org, r.deadlines = "", id, false
	return nil
}

func (r *fakeRepo) SetDeadlines(context.Context) error {
	r.account, r.org, r.deadlines = "", "", true
	return nil
}

func (r *fakeRepo) visible(x datarequest.Request) bool {
	switch {
	case r.account != "":
		return x.AccountID == r.account
	case r.org != "":
		return x.OrganizationID == r.org
	case r.deadlines:
		return x.Status == datarequest.StatusReceived || x.Status == datarequest.StatusInProgress
	}
	return false
}

func (r *fakeRepo) Insert(_ context.Context, x datarequest.Request) (datarequest.Request, error) {
	if r.insertErr != nil {
		return datarequest.Request{}, r.insertErr
	}
	r.next++
	x.ID = "00000000-0000-4000-8000-" + leftPad(r.next)
	r.requests[x.ID] = x
	r.changes = append(r.changes, datarequest.Change{RequestID: x.ID, ChangedBy: x.FiledBy, Status: x.Status, ChangedAt: x.ReceivedAt})
	return x, nil
}

func leftPad(n int) string {
	s := strconv.Itoa(n)
	for len(s) < 12 {
		s = "0" + s
	}
	return s
}

// put stores x as the owner would, with its first history entry.
func (r *fakeRepo) put(x datarequest.Request) {
	r.requests[x.ID] = x
	r.changes = append(r.changes, datarequest.Change{RequestID: x.ID, Status: datarequest.StatusReceived, ChangedAt: x.ReceivedAt})
}

func (r *fakeRepo) ByID(_ context.Context, id string) (datarequest.Request, error) {
	x, ok := r.requests[id]
	if !ok || !r.visible(x) {
		return datarequest.Request{}, datarequest.ErrNotFound
	}
	return x, nil
}

func (r *fakeRepo) ForAccount(_ context.Context, id string) ([]datarequest.Request, error) {
	var out []datarequest.Request
	for _, x := range r.requests {
		if r.visible(x) && x.AccountID == id {
			out = append(out, x)
		}
	}
	slices.SortFunc(out, func(a, b datarequest.Request) int { return b.ReceivedAt.Compare(a.ReceivedAt) })
	return out, nil
}

func (r *fakeRepo) ForClinic(_ context.Context, id string) ([]datarequest.Request, error) {
	var out []datarequest.Request
	for _, x := range r.requests {
		if r.visible(x) && x.OrganizationID == id {
			out = append(out, x)
		}
	}
	slices.SortFunc(out, func(a, b datarequest.Request) int { return a.Deadline().Compare(b.Deadline()) })
	return out, nil
}

func (r *fakeRepo) Update(_ context.Context, x datarequest.Request, changedBy string) (bool, error) {
	old, ok := r.requests[x.ID]
	if !ok || !r.visible(old) || old.Version != x.Version-1 {
		return false, nil
	}
	r.requests[x.ID] = x
	if old.Status != x.Status {
		r.changes = append(r.changes, datarequest.Change{RequestID: x.ID, ChangedBy: changedBy, Status: x.Status, ChangedAt: x.AnsweredAt})
	}
	return true, nil
}

func (r *fakeRepo) History(_ context.Context, id string) ([]datarequest.Change, error) {
	var out []datarequest.Change
	for _, c := range r.changes {
		if c.RequestID == id {
			out = append(out, c)
		}
	}
	return out, nil
}

func (r *fakeRepo) OpenDeadlines(_ context.Context, before time.Time) ([]datarequest.Deadline, error) {
	var out []datarequest.Deadline
	for _, x := range r.requests {
		if r.visible(x) && x.Deadline().Before(before) {
			out = append(out, datarequest.Deadline{Request: x, WarnedAt: r.warned[x.ID]})
		}
	}
	slices.SortFunc(out, func(a, b datarequest.Deadline) int { return a.Request.Deadline().Compare(b.Request.Deadline()) })
	return out, nil
}

func (r *fakeRepo) MarkWarned(_ context.Context, id string, seen, at time.Time) (bool, error) {
	x, ok := r.requests[id]
	if !ok || !r.visible(x) || !r.warned[id].Equal(seen) {
		return false, nil
	}
	r.warned[id] = at
	return true, nil
}

type fakeAudit struct{ events []auditevent.AuditEvent }

func (a *fakeAudit) Record(_ context.Context, e auditevent.AuditEvent) error {
	a.events = append(a.events, e)
	return nil
}

type fakeTx struct{}

func (fakeTx) WithTx(ctx context.Context, fn func(ctx context.Context) error) error { return fn(ctx) }

// fakeProfiles has patient A's profile.
type fakeProfiles struct{}

func (fakeProfiles) Me(_ context.Context, c patient.Caller) (patient.Patient, error) {
	if c.AccountID == accountA {
		return patient.Patient{ID: profileA}, nil
	}
	return patient.Patient{}, patient.ErrNotFound
}

// fakeConsents: patient A has a (revoked) consent at clinic A, none at clinic B.
type fakeConsents struct{}

func (fakeConsents) Mine(_ context.Context, c consent.Caller) ([]consent.Consent, error) {
	if c.PatientID != profileA {
		return nil, nil
	}
	return []consent.Consent{{PatientID: profileA, OrganizationID: clinicA, Status: consent.StatusInactive}}, nil
}

// fakeClinics: clinic A is in Sarajevo, clinic B in Tokyo.
type fakeClinics struct{}

func (fakeClinics) Clinic(_ context.Context, id string) (organization.Organization, error) {
	switch id {
	case clinicA:
		return organization.Organization{ID: id, TimeZone: "Europe/Sarajevo"}, nil
	case clinicB:
		return organization.Organization{ID: id, TimeZone: "Asia/Tokyo"}, nil
	}
	return organization.Organization{}, organization.ErrNotFound
}

// fakeJobs keeps the jobs queued: D1's warnings and D2's exports.
type fakeJobs struct {
	warnings []datarequest.WarningArgs
	queued   []datarequest.ExportArgs
}

func (j *fakeJobs) EnqueueExport(_ context.Context, args datarequest.ExportArgs) error {
	j.queued = append(j.queued, args)
	return nil
}

func (j *fakeJobs) EnqueueWarning(_ context.Context, a datarequest.WarningArgs) error {
	j.warnings = append(j.warnings, a)
	return nil
}

// fixture is a Service on the fakes at now.
type fixture struct {
	svc     *datarequest.Service
	repo    *fakeRepo
	audit   *fakeAudit
	jobs    *fakeJobs
	exports *fakeExports
}

func newFixture(now time.Time) fixture { return newFixtureOn(newFakeRepo(), now) }

// newFixtureOn is a Service on the fakes at now, with repo's requests.
func newFixtureOn(repo *fakeRepo, now time.Time) fixture {
	f := fixture{repo: repo, audit: &fakeAudit{}, jobs: &fakeJobs{}, exports: newFakeExports()}
	f.svc = datarequest.NewService(datarequest.Ports{
		Repo: f.repo, Clock: clock.Fixed{T: now}, Tx: fakeTx{}, Audit: f.audit,
		Profiles: fakeProfiles{}, Consents: fakeConsents{}, Clinics: fakeClinics{}, Jobs: f.jobs,
		Export: datarequest.ExportPorts{Repo: f.exports, Audit: f.audit, Tx: fakeTx{}, Jobs: f.jobs},
	})
	return f
}

var (
	patientA = datarequest.Patient{AccountID: accountA}
	admin    = datarequest.Staff{AccountID: adminA, OrganizationID: clinicA, Admin: true}
	desk     = datarequest.Staff{AccountID: deskA, OrganizationID: clinicA, Reception: true}
)
