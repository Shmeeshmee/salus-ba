package organization_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

type fakeBillingRepository struct {
	err     error
	details map[string]organization.BillingDetails
	clinic  string
}

func (r *fakeBillingRepository) SetOrganization(_ context.Context, id string) error {
	r.clinic = id
	return nil
}

func (r *fakeBillingRepository) BillingDetails(_ context.Context, org string) (organization.BillingDetails, error) {
	d, ok := r.details[org]
	if r.err != nil || !ok || org != r.clinic {
		return organization.BillingDetails{}, errors.Join(r.err, organization.ErrNotFound)
	}
	return d, nil
}

func (r *fakeBillingRepository) SetBillingDetails(_ context.Context, org string, d organization.BillingDetails) error {
	if r.err != nil {
		return r.err
	}
	if _, ok := r.details[org]; !ok || org != r.clinic {
		return organization.ErrNotFound
	}
	r.details[org] = d
	return nil
}

var syntheticDetails = organization.BillingDetails{LegalName: "Synthetic clinic d.o.o.", TaxID: "4200000000001", BillingEmail: "billing@example.test", VATRegistered: true}

// Only the clinic's admin reads and replaces its billing details; a change is audited, also when refused.
func TestBilling_OnlyTheAdminReadsAndReplacesTheClinicsDetails(t *testing.T) {
	repo := &fakeBillingRepository{details: map[string]organization.BillingDetails{clinicA: {}}}
	audit := &fakeAuditor{}
	svc := organization.NewBilling(organization.BillingPorts{Repo: repo, Audit: audit, Tx: fakeTransactor{}})
	if d, err := svc.Details(t.Context(), admin); err != nil || d != (organization.BillingDetails{}) {
		t.Errorf("Details() = %+v, %v, want none stated", d, err)
	}
	in := syntheticDetails
	in.LegalName = "  " + in.LegalName + " "
	got, err := svc.SetDetails(t.Context(), admin, in)
	if err != nil || got != syntheticDetails || repo.details[clinicA] != syntheticDetails {
		t.Fatalf("SetDetails() = %+v, %v, stored %+v", got, err, repo.details[clinicA])
	}
	if len(audit.events) != 1 || audit.events[0].Outcome != auditevent.OutcomeSuccess || audit.events[0].EntityReference != "Organization/"+clinicA {
		t.Errorf("audit %+v, want one successful update of the clinic", audit.events)
	}
	for name, c := range map[string]struct {
		want   error
		caller organization.Caller
	}{
		"practitioner": {caller: practitioner, want: organization.ErrForbidden},
		"patient":      {caller: organization.Caller{AccountID: adminA}, want: organization.ErrNotFound},
	} {
		if _, err := svc.Details(t.Context(), c.caller); !errors.Is(err, c.want) {
			t.Errorf("%s: Details() = %v, want %v", name, err, c.want)
		}
		before := len(audit.events)
		if _, err := svc.SetDetails(t.Context(), c.caller, syntheticDetails); !errors.Is(err, c.want) {
			t.Errorf("%s: SetDetails() = %v, want %v", name, err, c.want)
		}
		if len(audit.events) != before+1 || audit.events[before].Outcome != auditevent.OutcomeMinorFailure {
			t.Errorf("%s: the refusal is not audited", name)
		}
	}
	repo.err = errors.New("down")
	if _, err := svc.SetDetails(t.Context(), admin, syntheticDetails); err == nil || errors.Is(err, organization.ErrNotFound) {
		t.Errorf("SetDetails() with the repository down = %v", err)
	}
}

func TestBilling_RefusesMalformedDetails(t *testing.T) {
	repo := &fakeBillingRepository{details: map[string]organization.BillingDetails{clinicA: syntheticDetails}}
	svc := organization.NewBilling(organization.BillingPorts{Repo: repo, Audit: &fakeAuditor{}, Tx: fakeTransactor{}})
	for name, change := range map[string]func(*organization.BillingDetails){
		"tax id of 11 digits": func(d *organization.BillingDetails) { d.TaxID = "42000000000" },
		"tax id with letters": func(d *organization.BillingDetails) { d.TaxID = "42000000000AB" },
		"e-mail without host": func(d *organization.BillingDetails) { d.BillingEmail = "billing" },
		"e-mail with a name":  func(d *organization.BillingDetails) { d.BillingEmail = "Billing <billing@example.test>" },
		"a long legal name":   func(d *organization.BillingDetails) { d.LegalName = strings.Repeat("a", 201) },
	} {
		d := syntheticDetails
		change(&d)
		if _, err := svc.SetDetails(t.Context(), admin, d); !errors.Is(err, organization.ErrInvalidBillingDetails) {
			t.Errorf("%s: %v, want ErrInvalidBillingDetails", name, err)
		}
	}
	if got, err := svc.SetDetails(t.Context(), admin, organization.BillingDetails{}); err != nil || got != (organization.BillingDetails{}) {
		t.Errorf("clearing the details: %+v, %v", got, err)
	}
}

type stubBilling struct {
	err error
	in  organization.BillingDetails
}

func (s *stubBilling) Details(context.Context, organization.Caller) (organization.BillingDetails, error) {
	return syntheticDetails, s.err
}

func (s *stubBilling) SetDetails(_ context.Context, _ organization.Caller, in organization.BillingDetails) (organization.BillingDetails, error) {
	s.in = in
	return in, s.err
}

func serveBilling(t *testing.T, svc *stubBilling, signedIn bool, method, body string) *httptest.ResponseRecorder {
	t.Helper()
	h := organization.NewBillingHandler(svc, func(context.Context) (organization.Caller, bool, error) { return adminCaller, signedIn, nil },
		slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	mux := http.NewServeMux()
	h.Routes(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), method, "/clinic/billing-details", strings.NewReader(body)))
	return rec
}

func TestBillingHandler_ServesAndReplacesTheDetails(t *testing.T) {
	svc := &stubBilling{}
	want := `{"vatRegistered":true,"legalName":"Synthetic clinic d.o.o.","taxId":"4200000000001","billingEmail":"billing@example.test"}` + "\n"
	if rec := serveBilling(t, svc, true, http.MethodGet, ""); rec.Code != http.StatusOK || rec.Body.String() != want {
		t.Errorf("GET = %d %s, want %s", rec.Code, rec.Body, want)
	}
	rec := serveBilling(t, svc, true, http.MethodPut, `{"vatRegistered":false,"taxId":"420000000001"}`)
	if rec.Code != http.StatusOK || rec.Body.String() != `{"vatRegistered":false,"taxId":"420000000001"}`+"\n" || svc.in.TaxID != "420000000001" {
		t.Errorf("PUT = %d %s, asked %+v", rec.Code, rec.Body, svc.in)
	}
	for name, body := range map[string]string{"unreadable": `{`, "unknown field": `{"vatRegistered":true,"iban":"x"}`, "no vatRegistered": `{"legalName":"x"}`, "two documents": `{"vatRegistered":true}{}`} {
		if rec := serveBilling(t, svc, true, http.MethodPut, body); rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "invalid_request") {
			t.Errorf("%s: %d %s, want 400 invalid_request", name, rec.Code, rec.Body)
		}
	}
	if rec := serveBilling(t, svc, false, http.MethodGet, ""); rec.Code != http.StatusUnauthorized {
		t.Errorf("signed out: %d, want 401", rec.Code)
	}
	if rec := serveBilling(t, svc, false, http.MethodPut, `{"vatRegistered":true}`); rec.Code != http.StatusUnauthorized {
		t.Errorf("signed out PUT: %d, want 401", rec.Code)
	}
	for err, code := range map[error]int{
		organization.ErrNotFound: http.StatusNotFound, organization.ErrForbidden: http.StatusForbidden,
		organization.ErrInvalidBillingDetails: http.StatusBadRequest, errors.New("secret detail"): http.StatusInternalServerError,
	} {
		rec := serveBilling(t, &stubBilling{err: err}, true, http.MethodPut, `{"vatRegistered":true}`)
		if rec.Code != code || strings.Contains(rec.Body.String(), "secret") {
			t.Errorf("%v: %d %s, want %d", err, rec.Code, rec.Body, code)
		}
	}
}

// A clinic stores and reads its own billing details, never another's.
func TestBillingRepository_StoresTheClinicsOwnDetails(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seedClinics(t, owner)
	repo := organization.NewPostgresBillingRepository(app)
	in := func(org string, fn func(ctx context.Context) error) error {
		return postgres.NewTransactor(app).WithTx(t.Context(), func(ctx context.Context) error {
			if err := repo.SetOrganization(ctx, org); err != nil {
				return err
			}
			return fn(ctx)
		})
	}
	var got organization.BillingDetails
	if err := in(clinicA, func(ctx context.Context) (err error) { got, err = repo.BillingDetails(ctx, clinicA); return err }); err != nil || got != (organization.BillingDetails{}) {
		t.Fatalf("BillingDetails() = %+v, %v, want none stated", got, err)
	}
	if err := in(clinicA, func(ctx context.Context) error { return repo.SetBillingDetails(ctx, clinicA, syntheticDetails) }); err != nil {
		t.Fatal(err)
	}
	if err := in(clinicA, func(ctx context.Context) (err error) { got, err = repo.BillingDetails(ctx, clinicA); return err }); err != nil || got != syntheticDetails {
		t.Errorf("BillingDetails() = %+v, %v, want what was stored", got, err)
	}
	if err := in(clinicB, func(ctx context.Context) error {
		return repo.SetBillingDetails(ctx, clinicA, organization.BillingDetails{})
	}); !errors.Is(err, organization.ErrNotFound) {
		t.Errorf("clinic B replacing clinic A's details: %v, want ErrNotFound", err)
	}
	if err := in(clinicB, func(ctx context.Context) (err error) { _, err = repo.BillingDetails(ctx, clinicA); return err }); !errors.Is(err, organization.ErrNotFound) {
		t.Errorf("clinic B reading clinic A's details: %v, want ErrNotFound", err)
	}
	if err := in(clinicA, func(ctx context.Context) error { return repo.SetBillingDetails(ctx, "x", syntheticDetails) }); !errors.Is(err, organization.ErrNotFound) {
		t.Errorf("an id that is no UUID: %v, want ErrNotFound", err)
	}
}

// ForClinic reads a clinic's billing details for its own receipts' seller (clinic-cms C): in that clinic's scope,
// whoever of its staff issues, unaudited (nothing in them is personal data); ErrNotFound for no clinic.
func TestBilling_ForClinicReadsTheDetailsInTheClinicsScope(t *testing.T) {
	repo := &fakeBillingRepository{details: map[string]organization.BillingDetails{clinicA: syntheticDetails}}
	audit := &fakeAuditor{}
	svc := organization.NewBilling(organization.BillingPorts{Repo: repo, Audit: audit, Tx: fakeTransactor{}})
	got, err := svc.ForClinic(t.Context(), clinicA)
	if err != nil || got != syntheticDetails || repo.clinic != clinicA {
		t.Errorf("ForClinic() = %+v, %v in scope %q, want the clinic's details in its scope", got, err, repo.clinic)
	}
	if len(audit.events) != 0 {
		t.Errorf("audit %+v, want none", audit.events)
	}
	for _, org := range []string{"", "00000000-0000-4000-8000-00000000ffff"} {
		if _, err := svc.ForClinic(t.Context(), org); !errors.Is(err, organization.ErrNotFound) {
			t.Errorf("ForClinic(%q) = %v, want ErrNotFound", org, err)
		}
	}
}
