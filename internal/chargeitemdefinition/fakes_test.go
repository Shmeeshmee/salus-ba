package chargeitemdefinition_test

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"time"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/chargeitemdefinition"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

const (
	clinicA = "a0000000-0000-4000-8000-00000000000a"
	clinicB = "b0000000-0000-4000-8000-00000000000b"
	adminA  = "a1000000-0000-4000-8000-0000000000a1"
)

var (
	now       = time.Date(2026, 11, 21, 9, 0, 0, 0, time.UTC)
	admin     = chargeitemdefinition.Caller{AccountID: adminA, OrganizationID: clinicA, Admin: true}
	staff     = chargeitemdefinition.Caller{AccountID: "a2000000-0000-4000-8000-0000000000a2", OrganizationID: clinicA}
	noClinic  = chargeitemdefinition.Caller{AccountID: "c1000000-0000-4000-8000-0000000000c1"}
	errBroken = errors.New("fake: broken")
)

type txKey struct{}

type fakeTransactor struct{}

func (fakeTransactor) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(context.WithValue(ctx, txKey{}, true))
}

// fakeRepository holds services and shows what the scope set on the transaction shows, as row-level security does:
// the clinic's own, or in the directory every clinic's active ones.
type fakeRepository struct {
	writtenAt time.Time
	err       error
	clinic    string
	rows      []chargeitemdefinition.Definition
	directory bool
	outside   bool // a call was made outside a transaction
}

func (r *fakeRepository) seen(ctx context.Context) {
	if ctx.Value(txKey{}) == nil {
		r.outside = true
	}
}

func (r *fakeRepository) SetOrganization(ctx context.Context, id string) error {
	r.seen(ctx)
	r.clinic, r.directory = id, false
	return r.err
}

func (r *fakeRepository) SetDirectory(ctx context.Context) error {
	r.seen(ctx)
	r.clinic, r.directory = "", true
	return r.err
}

func (r *fakeRepository) shows(d chargeitemdefinition.Definition) bool {
	return d.OrganizationID == r.clinic || r.directory && d.Active
}

func (r *fakeRepository) List(ctx context.Context, org string) ([]chargeitemdefinition.Definition, error) {
	r.seen(ctx)
	var out []chargeitemdefinition.Definition
	for _, d := range r.rows {
		if d.OrganizationID == org && r.shows(d) {
			out = append(out, d)
		}
	}
	return out, r.err
}

func (r *fakeRepository) Create(ctx context.Context, d chargeitemdefinition.Definition, at time.Time) (chargeitemdefinition.Definition, error) {
	r.seen(ctx)
	if d.OrganizationID != r.clinic {
		return chargeitemdefinition.Definition{}, errors.New("fake: outside the scope")
	}
	d.ID, d.VersionID, d.LastUpdated = "d0000000-0000-4000-8000-00000000000"+strconv.Itoa(len(r.rows)+1), 1, at
	r.rows, r.writtenAt = append(r.rows, d), at
	return d, r.err
}

func (r *fakeRepository) Update(ctx context.Context, d chargeitemdefinition.Definition, at time.Time) (chargeitemdefinition.Definition, error) {
	r.seen(ctx)
	i := slices.IndexFunc(r.rows, func(x chargeitemdefinition.Definition) bool { return x.ID == d.ID && x.OrganizationID == r.clinic })
	switch {
	case i < 0:
		return chargeitemdefinition.Definition{}, chargeitemdefinition.ErrNotFound
	case r.rows[i].VersionID != d.VersionID:
		return chargeitemdefinition.Definition{}, chargeitemdefinition.ErrConflict
	}
	d.OrganizationID, d.VersionID, d.LastUpdated = r.clinic, d.VersionID+1, at
	r.rows[i], r.writtenAt = d, at
	return d, r.err
}

type fakeAuditor struct {
	err    error
	events []auditevent.AuditEvent
}

func (a *fakeAuditor) Record(_ context.Context, e auditevent.AuditEvent) error {
	if a.err != nil {
		return a.err
	}
	a.events = append(a.events, e)
	return nil
}

// fakeClinics is the public directory: the listed clinics.
type fakeClinics struct {
	err    error
	listed []string
}

func (c fakeClinics) ListListed(context.Context) ([]organization.Organization, error) {
	out := make([]organization.Organization, 0, len(c.listed))
	for _, id := range c.listed {
		out = append(out, organization.Organization{ID: id, Listed: true})
	}
	return out, c.err
}

func newService(repo *fakeRepository, audit *fakeAuditor) *chargeitemdefinition.Service {
	return chargeitemdefinition.NewService(chargeitemdefinition.Ports{
		Repo: repo, Audit: audit, Tx: fakeTransactor{}, Clock: clock.Fixed{T: now}, Clinics: fakeClinics{listed: []string{clinicA}},
	})
}
