package location_test

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/location"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

const (
	lcClinicA = "a0000000-0000-4000-8000-00000000000a"
	lcClinicB = "b0000000-0000-4000-8000-00000000000b"
)

var (
	lcNow    = time.Date(2026, 11, 25, 9, 0, 0, 0, time.UTC)
	lcAdmin  = location.Staff{AccountID: "a1000000-0000-4000-8000-0000000000a1", OrganizationID: lcClinicA, Admin: true}
	lcStaff  = location.Staff{AccountID: "a2000000-0000-4000-8000-0000000000a2", OrganizationID: lcClinicA}
	lcNobody = location.Staff{AccountID: "c1000000-0000-4000-8000-0000000000c1"}
)

type lcTxKey struct{}

type lcTx struct{}

func (lcTx) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(context.WithValue(ctx, lcTxKey{}, true))
}

type lcAuditor struct{ events []auditevent.AuditEvent }

func (a *lcAuditor) Record(_ context.Context, e auditevent.AuditEvent) error {
	a.events = append(a.events, e)
	return nil
}

type lcClinics struct{ listed []string }

func (c lcClinics) ListListed(context.Context) ([]organization.Organization, error) {
	var out []organization.Organization
	for _, id := range c.listed {
		out = append(out, organization.Organization{ID: id})
	}
	return out, nil
}

// lcRepo holds locations and shows what the scope shows, as row-level security does: the clinic's own, or in the
// directory the active ones. The rooms and devices are RM1's.
type lcRepo struct {
	location.Repository // RM1's methods: not called here
	clinic              string
	rows                []location.Location
	directory, outside  bool
}

func (r *lcRepo) seen(ctx context.Context) {
	if ctx.Value(lcTxKey{}) == nil {
		r.outside = true
	}
}

func (r *lcRepo) SetOrganization(ctx context.Context, id string) error {
	r.seen(ctx)
	r.clinic, r.directory = id, false
	return nil
}

func (r *lcRepo) SetDirectory(ctx context.Context) error {
	r.seen(ctx)
	r.clinic, r.directory = "", true
	return nil
}

func (r *lcRepo) Locations(ctx context.Context, org string) ([]location.Location, error) {
	r.seen(ctx)
	var out []location.Location
	for _, l := range r.rows {
		if l.OrganizationID == org && l.OrganizationID == r.clinic {
			out = append(out, l)
		}
	}
	return out, nil
}

func (r *lcRepo) PublicLocations(ctx context.Context, org string) ([]location.Location, error) {
	r.seen(ctx)
	var out []location.Location
	for _, l := range r.rows {
		if r.directory && l.OrganizationID == org && l.Active {
			out = append(out, l)
		}
	}
	return out, nil
}

func (r *lcRepo) SaveLocation(ctx context.Context, l location.Location) (location.Location, bool, error) {
	r.seen(ctx)
	if l.ID == "" {
		l.ID, l.VersionID = "l"+strconv.Itoa(len(r.rows)+1), 1
		r.rows = append(r.rows, l)
		return l, true, nil
	}
	i := slices.IndexFunc(r.rows, func(x location.Location) bool { return x.ID == l.ID && x.OrganizationID == r.clinic })
	switch {
	case i < 0:
		return location.Location{}, false, location.ErrNotFound
	case r.rows[i].VersionID != l.VersionID-1:
		return location.Location{}, false, nil
	}
	r.rows[i] = l
	return l, true, nil
}

func newLC(repo *lcRepo, audit *lcAuditor, listed ...string) *location.Service {
	return location.NewService(location.Ports{Repo: repo, Clock: clock.Fixed{T: lcNow}, Audit: audit, Tx: lcTx{}, Clinics: lcClinics{listed: listed}})
}

func TestLocations_TheClinicsStaffSeeItsOwn(t *testing.T) {
	t.Parallel()
	repo := &lcRepo{rows: []location.Location{
		{ID: "a1", OrganizationID: lcClinicA, Name: "Centre", Active: true}, {ID: "b1", OrganizationID: lcClinicB, Name: "Other"},
	}}
	got, err := newLC(repo, &lcAuditor{}).Locations(t.Context(), lcStaff)
	if err != nil || len(got) != 1 || got[0].ID != "a1" || repo.outside {
		t.Errorf("Locations = %+v, %v (outside a transaction: %v); want clinic A's one", got, err, repo.outside)
	}
	if _, err := newLC(repo, &lcAuditor{}).Locations(t.Context(), lcNobody); !errors.Is(err, location.ErrNotFound) {
		t.Errorf("Locations of no clinic = %v, want ErrNotFound", err)
	}
}

func TestSaveLocation_TheAdminAddsAndChangesAudited(t *testing.T) {
	t.Parallel()
	repo, audit := &lcRepo{}, &lcAuditor{}
	s := newLC(repo, audit)
	added, err := s.SaveLocation(t.Context(), lcAdmin, location.Location{
		Name: "  Second site ", AddressLine: "Synthetic street 3", Phone: "+38733000000", TimeZone: "Europe/Sarajevo", Active: true,
	})
	if err != nil || added.ID == "" || added.Name != "Second site" || added.OrganizationID != lcClinicA || added.VersionID != 1 {
		t.Fatalf("add = %+v, %v", added, err)
	}
	added.Name, added.Active = "Second site, renamed", false
	changed, err := s.SaveLocation(t.Context(), lcAdmin, added)
	if err != nil || changed.Name != "Second site, renamed" || changed.Active || changed.VersionID != 2 {
		t.Fatalf("change = %+v, %v", changed, err)
	}
	want := []auditevent.AuditEvent{
		lcEvent(auditevent.CodeCreate, auditevent.ActionCreate, "Location/"+added.ID, auditevent.OutcomeSuccess),
		lcEvent(auditevent.CodeUpdate, auditevent.ActionUpdate, "Location/"+added.ID, auditevent.OutcomeSuccess),
	}
	if !slices.Equal(audit.events, want) || repo.outside {
		t.Errorf("audit = %+v (outside a transaction: %v), want %+v", audit.events, repo.outside, want)
	}
}

func TestSaveLocation_RefusesAndAuditsTheRefusal(t *testing.T) {
	t.Parallel()
	ok := location.Location{Name: "Site", Active: true}
	stale := location.Location{ID: "l1", Name: "Site", VersionID: 5, Active: true}
	other := location.Location{ID: "b1", Name: "Site", VersionID: 2, Active: true}
	for name, tc := range map[string]struct {
		want    error
		staff   location.Staff
		in      location.Location
		audited bool
	}{
		"a practitioner":            {location.ErrForbidden, lcStaff, ok, true},
		"no clinic":                 {location.ErrNotFound, lcNobody, ok, true},
		"another version":           {location.ErrVersionConflict, lcAdmin, stale, true},
		"another clinic's location": {location.ErrNotFound, lcAdmin, other, true},
		"an empty name":             {location.ErrInvalid, lcAdmin, location.Location{Name: "  "}, false},
		"a malformed phone":         {location.ErrInvalid, lcAdmin, location.Location{Name: "Site", Phone: "033 000 000"}, false},
		"an unknown time zone":      {location.ErrInvalid, lcAdmin, location.Location{Name: "Site", TimeZone: "Europe/Atlantis"}, false},
		"Local as a time zone":      {location.ErrInvalid, lcAdmin, location.Location{Name: "Site", TimeZone: "Local"}, false},
		"a long postal code":        {location.ErrInvalid, lcAdmin, location.Location{Name: "Site", PostalCode: "123456789012345678901"}, false},
	} {
		repo := &lcRepo{rows: []location.Location{
			{ID: "l1", OrganizationID: lcClinicA, Name: "Site", VersionID: 1}, {ID: "b1", OrganizationID: lcClinicB, Name: "B", VersionID: 1},
		}}
		audit := &lcAuditor{}
		_, err := newLC(repo, audit).SaveLocation(t.Context(), tc.staff, tc.in)
		if !errors.Is(err, tc.want) {
			t.Errorf("%s: %v, want %v", name, err, tc.want)
		}
		if refused := len(audit.events) == 1 && audit.events[0].Outcome == auditevent.OutcomeMinorFailure; refused != tc.audited {
			t.Errorf("%s: audit %+v, want a refusal audited: %v", name, audit.events, tc.audited)
		}
	}
}

func TestPublicLocations_TheActiveLocationsOfAListedClinic(t *testing.T) {
	t.Parallel()
	repo := &lcRepo{rows: []location.Location{
		{ID: "a1", OrganizationID: lcClinicA, Name: "Centre", Active: true}, {ID: "a2", OrganizationID: lcClinicA, Name: "Closed"},
		{ID: "b1", OrganizationID: lcClinicB, Name: "Unlisted", Active: true},
	}}
	s := newLC(repo, &lcAuditor{}, lcClinicA)
	got, err := s.PublicLocations(t.Context(), lcClinicA)
	if err != nil || len(got) != 1 || got[0].ID != "a1" || repo.outside {
		t.Errorf("PublicLocations(A) = %+v, %v; want its active one", got, err)
	}
	if _, err := s.PublicLocations(t.Context(), lcClinicB); !errors.Is(err, location.ErrNotFound) {
		t.Errorf("PublicLocations(an unlisted clinic) = %v, want ErrNotFound", err)
	}
}

func lcEvent(code string, action auditevent.Action, entity string, outcome auditevent.Outcome) auditevent.AuditEvent {
	return auditevent.AuditEvent{
		Type:            auditevent.Coding{System: auditevent.SystemAuditEventType, Code: auditevent.CodeRest},
		Subtype:         auditevent.Coding{System: auditevent.SystemRestfulInteraction, Code: code},
		Purpose:         auditevent.Coding{System: auditevent.SystemActReason, Code: auditevent.CodeHOPERAT},
		Action:          action,
		Outcome:         outcome,
		AgentAccountID:  lcAdmin.AccountID,
		OrganizationID:  lcClinicA,
		EntityReference: entity,
	}
}
