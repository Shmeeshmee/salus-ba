package location_test

import (
	"context"
	"testing"

	"github.com/Shmeeshmee/salus/internal/location"
	"github.com/Shmeeshmee/salus/internal/terminology"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

type lcPlaces []terminology.Place

func (p lcPlaces) AllPlaces(context.Context) ([]terminology.Place, error) { return p, nil }

// A location's address on one line, as a clinic's (organization.Organization.Address): the street, then the postal code
// and the city's name; what a message or a receipt says of where an in-person visit is (docs/clinic-cms/WAVE6.md U2).
func TestAddress_IsTheLocationsStreetPostalCodeAndCity(t *testing.T) {
	t.Parallel()
	repo := &lcRepo{rows: []location.Location{
		{ID: "a1", OrganizationID: lcClinicA, Name: "Centre", AddressLine: "Synthetic street 1", PostalCode: "71000", CityID: "Q11194", Active: true},
		{ID: "a2", OrganizationID: lcClinicA, Name: "Annex", AddressLine: "Synthetic annex 2", Active: true},
		{ID: "a3", OrganizationID: lcClinicA, Name: "No address", Active: true},
		{ID: "b1", OrganizationID: lcClinicB, Name: "Other", AddressLine: "Synthetic other 3"},
	}}
	svc := location.NewService(location.Ports{Repo: repo, Clock: clock.Fixed{T: lcNow}, Audit: &lcAuditor{}, Tx: lcTx{},
		Places: lcPlaces{{ID: "Q11194", Kind: "city", NameLatn: "Sarajevo"}}})
	for id, want := range map[string]string{"a1": "Synthetic street 1, 71000 Sarajevo", "a2": "Synthetic annex 2", "a3": "", "b1": "", "zz": ""} {
		if got, err := svc.Address(t.Context(), lcClinicA, id); err != nil || got != want {
			t.Errorf("Address(%s) = %q, %v; want %q", id, got, err, want)
		}
	}
}

// A location's place, as a receipt names where a service was given: its name, then its address when it has one.
func TestPlace_IsTheLocationsNameAndAddress(t *testing.T) {
	t.Parallel()
	repo := &lcRepo{rows: []location.Location{
		{ID: "a1", OrganizationID: lcClinicA, Name: "Centre", AddressLine: "Synthetic street 1", PostalCode: "71000", Active: true},
		{ID: "a2", OrganizationID: lcClinicA, Name: "Annex", Active: true},
	}}
	svc := location.NewService(location.Ports{Repo: repo, Clock: clock.Fixed{T: lcNow}, Audit: &lcAuditor{}, Tx: lcTx{}})
	for id, want := range map[string]string{"a1": "Centre, Synthetic street 1, 71000", "a2": "Annex", "zz": ""} {
		if got, err := svc.Place(t.Context(), lcClinicA, id); err != nil || got != want {
			t.Errorf("Place(%s) = %q, %v; want %q", id, got, err, want)
		}
	}
}
