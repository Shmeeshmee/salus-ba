package organization_test

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Shmeeshmee/salus/internal/organization"
)

// A patient's read of a clinic that serves them (readiness: suspended clinics) goes with the handler's Served, and
// gives the clinic as the directory shows it: without its location and its contribution, which are its staff's.
func TestHandlerRead_GivesAPatientTheClinicAsTheDirectoryShowsIt(t *testing.T) {
	o := clinic(3)
	o.Listed = true
	o.Region, o.City, o.CityName, o.AddressLine, o.PostalCode = "BA-09", "Q1270190", "Centar", "Synthetic street 1", "71000"
	svc := &stubClinics{org: o}
	served := func(context.Context) ([]string, error) { return []string{clinicA}, nil }
	caller := func(context.Context) (organization.Caller, bool, error) { return patient, true, nil }
	mux := http.NewServeMux()
	organization.NewHandler(svc, caller, "https://api.example.test/fhir/R4", slog.New(slog.DiscardHandler)).WithServed(served).Routes(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, clinicPath, nil))
	if rec.Code != http.StatusOK || svc.caller != patient || svc.served == nil {
		t.Fatalf("status = %d %s as %+v, served %v; want 200 with the handler's Served", rec.Code, rec.Body, svc.caller, svc.served != nil)
	}
	if got, err := svc.served(t.Context()); err != nil || len(got) != 1 || got[0] != clinicA {
		t.Errorf("served() = %v, %v; want clinic A", got, err)
	}
	assertGolden(t, "organization_served.json", rec.Body.Bytes())
}
