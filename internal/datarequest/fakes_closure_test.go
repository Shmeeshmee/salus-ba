package datarequest_test

import (
	"context"
	"slices"
	"time"

	"github.com/Shmeeshmee/salus/internal/auth"
	"github.com/Shmeeshmee/salus/internal/consent"
	"github.com/Shmeeshmee/salus/internal/datarequest"
	"github.com/Shmeeshmee/salus/internal/patient"
)

const (
	accountID = "00000000-0000-4000-8000-0000000000a1"
	profileID = "00000000-0000-4000-8000-0000000000b1"
	password  = "a long enough secret"
)

var closedNow = time.Date(2026, 11, 24, 9, 0, 0, 0, time.UTC)

type txKey struct{}

type closureTx struct{}

func (closureTx) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(context.WithValue(ctx, txKey{}, true))
}

func inTx(ctx context.Context) bool { return ctx.Value(txKey{}) != nil }

// fakeAccounts is the auth service: one patient account, its closure and erasure.
type fakeAccounts struct {
	erased    []string
	due       []string
	calls     []string
	closedFor time.Duration
	closed    bool
	reopened  bool
}

func (a *fakeAccounts) Close(ctx context.Context, id, pw string, eraseAfter time.Duration) error {
	a.calls = append(a.calls, "Close")
	switch {
	case !inTx(ctx):
		panic("Close outside the transaction")
	case id != accountID || a.closed:
		return auth.ErrNotFound
	case pw != password:
		return auth.ErrWrongCredentials
	}
	a.closed, a.closedFor = true, eraseAfter
	return nil
}

func (a *fakeAccounts) Reopen(_ context.Context, id string) error {
	if id != accountID || !a.closed {
		return auth.ErrNotFound
	}
	a.closed, a.reopened = false, true
	return nil
}

func (a *fakeAccounts) DueForErasure(context.Context) ([]string, error) {
	var out []string
	for _, id := range a.due {
		if !slices.Contains(a.erased, id) {
			out = append(out, id)
		}
	}
	return out, nil
}

func (a *fakeAccounts) Erase(ctx context.Context, id string) error {
	if !inTx(ctx) {
		panic("Erase outside the transaction")
	}
	if slices.Contains(a.erased, id) {
		return auth.ErrNotFound
	}
	a.erased = append(a.erased, id)
	return nil
}

type closureProfiles struct{ id string }

func (p closureProfiles) Me(_ context.Context, c patient.Caller) (patient.Patient, error) {
	if p.id == "" || c.AccountID != accountID {
		return patient.Patient{}, patient.ErrNotFound
	}
	return patient.Patient{ID: p.id}, nil
}

// closureConsents holds the patient's consents; Revoke of a care consent ends the others with the clinic, as consent's.
type closureConsents struct {
	all     []consent.Consent
	revoked []string
	callers []consent.Caller
}

func (f *closureConsents) Mine(_ context.Context, c consent.Caller) ([]consent.Consent, error) {
	f.callers = append(f.callers, c)
	return slices.Clone(f.all), nil
}

func (f *closureConsents) Revoke(ctx context.Context, c consent.Caller, id string) (consent.Consent, error) {
	if !inTx(ctx) {
		panic("Revoke outside the transaction")
	}
	f.callers = append(f.callers, c)
	for i, x := range f.all {
		if x.ID != id {
			continue
		}
		if x.Status != consent.StatusActive {
			return consent.Consent{}, consent.ErrInactive
		}
		f.all[i].Status = consent.StatusInactive
		f.revoked = append(f.revoked, id)
		if x.Kind == consent.KindCare {
			for j, y := range f.all {
				if y.OrganizationID == x.OrganizationID && y.Kind != consent.KindCare {
					f.all[j].Status = consent.StatusInactive
				}
			}
		}
		return f.all[i], nil
	}
	return consent.Consent{}, consent.ErrNotFound
}

func (f *closureConsents) ClinicsOf(_ context.Context, patientID string) ([]string, error) {
	var out []string
	for _, x := range f.all {
		if x.PatientID == patientID && !slices.Contains(out, x.OrganizationID) {
			out = append(out, x.OrganizationID)
		}
	}
	slices.Sort(out)
	return out, nil
}

type fakeDevices struct{ removed []string }

func (d *fakeDevices) RemoveDevices(_ context.Context, id string) (int64, error) {
	d.removed = append(d.removed, id)
	return 1, nil
}

type fakeRestrictions struct{ restricted map[string]bool }

func (r *fakeRestrictions) RestrictForErasure(ctx context.Context, patientID, org string) error {
	if !inTx(ctx) {
		panic("RestrictForErasure outside the transaction")
	}
	r.restricted[patientID+"@"+org] = true
	return nil
}

// fakeScope records the scope the repository calls were made in.
type fakeScope struct{ erasure bool }

func (s *fakeScope) SetErasure(ctx context.Context) error {
	if !inTx(ctx) {
		panic("SetErasure outside the transaction")
	}
	s.erasure = true
	return nil
}

// closureRepo is the requests' store. Every call must come in the erasure scope.
type closureRepo struct {
	scope    *fakeScope
	requests []datarequest.Request
	changes  []datarequest.Change
}

func (r *closureRepo) must() {
	if !r.scope.erasure {
		panic("a request read or written outside the erasure scope")
	}
}

func (r *closureRepo) SetAccount(context.Context, string) error      { panic("not in the closure") }
func (r *closureRepo) SetOrganization(context.Context, string) error { panic("not in the closure") }

func (r *closureRepo) Insert(_ context.Context, x datarequest.Request) (datarequest.Request, error) {
	r.must()
	x.ID = "00000000-0000-4000-8000-0000000000d1"
	r.requests = append(r.requests, x)
	r.changes = append(r.changes, datarequest.Change{RequestID: x.ID, Status: x.Status, ChangedBy: x.FiledBy})
	return x, nil
}

func (r *closureRepo) ByID(context.Context, string) (datarequest.Request, error) {
	panic("not in the closure")
}

func (r *closureRepo) ForAccount(_ context.Context, id string) ([]datarequest.Request, error) {
	r.must()
	var out []datarequest.Request
	for _, x := range r.requests {
		if x.AccountID == id {
			out = append(out, x)
		}
	}
	return out, nil
}

func (r *closureRepo) ForClinic(context.Context, string) ([]datarequest.Request, error) {
	panic("not in the closure")
}

func (r *closureRepo) ForOperator(context.Context) ([]datarequest.Request, error) {
	panic("not in the closure")
}

func (r *closureRepo) Update(_ context.Context, x datarequest.Request, by string) (bool, error) {
	r.must()
	for i, old := range r.requests {
		if old.ID == x.ID {
			if old.Version != x.Version-1 {
				return false, nil
			}
			if old.Status != x.Status {
				r.changes = append(r.changes, datarequest.Change{RequestID: x.ID, Status: x.Status, ChangedBy: by})
			}
			r.requests[i] = x
			return true, nil
		}
	}
	return false, nil
}

func (r *closureRepo) History(context.Context, string) ([]datarequest.Change, error) {
	panic("not in the closure")
}

type closureFixture struct {
	accounts     *fakeAccounts
	consents     *closureConsents
	devices      *fakeDevices
	restrictions *fakeRestrictions
	scope        *fakeScope
	repo         *closureRepo
	profile      string
}

func newClosureFixture(profile string, consents ...consent.Consent) *closureFixture {
	scope := &fakeScope{}
	return &closureFixture{
		accounts: &fakeAccounts{}, consents: &closureConsents{all: consents}, devices: &fakeDevices{},
		restrictions: &fakeRestrictions{restricted: map[string]bool{}}, scope: scope, repo: &closureRepo{scope: scope},
		profile: profile,
	}
}

func (r *closureRepo) SetDeadlines(context.Context) error { panic("not in the closure") }

func (r *closureRepo) OpenDeadlines(context.Context, time.Time) ([]datarequest.Deadline, error) {
	panic("not in the closure")
}

func (r *closureRepo) MarkWarned(context.Context, string, time.Time, time.Time) (bool, error) {
	panic("not in the closure")
}
