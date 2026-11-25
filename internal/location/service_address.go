package location

import (
	"context"
	"fmt"
	"strings"

	"github.com/Shmeeshmee/salus/internal/terminology"
)

// Places is terminology's list of places, for a city's name (terminology.Service).
type Places interface {
	AllPlaces(ctx context.Context) ([]terminology.Place, error)
}

// Address is the address of the location id of the clinic org on one line, as a clinic's (organization.Address): the
// street, then the postal code and the city's name; "" for a location without a street or that is not the clinic's.
// What a message or a receipt says of where an in-person visit is (docs/clinic-cms/WAVE6.md U2). Read in the clinic's
// scope; not audited (a location is no personal data).
func (s *Service) Address(ctx context.Context, org, id string) (string, error) {
	_, address, err := s.site(ctx, org, id)
	return address, err
}

// Place is the location id of the clinic org as a receipt names where a service was given (docs/clinic-cms/WAVE6.md
// U2): its name, then its address (Address) when it has one; "" for a location that is not the clinic's.
func (s *Service) Place(ctx context.Context, org, id string) (string, error) {
	name, address, err := s.site(ctx, org, id)
	if address == "" || err != nil {
		return name, err
	}
	return name + ", " + address, nil
}

// site is the name and the address on one line of the location id of the clinic org, "" and "" for one that is not
// the clinic's.
func (s *Service) site(ctx context.Context, org, id string) (name, address string, err error) {
	all, err := s.OfClinic(ctx, org)
	if err != nil {
		return "", "", err
	}
	for _, l := range all {
		if l.ID != id || l.OrganizationID != org {
			continue
		}
		if l.AddressLine == "" {
			return l.Name, "", nil
		}
		city, err := s.cityName(ctx, l.CityID)
		if err != nil {
			return "", "", fmt.Errorf("location: address: %w", err)
		}
		if place := strings.TrimSpace(l.PostalCode + " " + city); place != "" {
			return l.Name, l.AddressLine + ", " + place, nil
		}
		return l.Name, l.AddressLine, nil
	}
	return "", "", nil
}

// cityName is the Latin name of the place id, "" for none or an unknown one.
func (s *Service) cityName(ctx context.Context, id string) (string, error) {
	if id == "" || s.p.Places == nil {
		return "", nil
	}
	all, err := s.p.Places.AllPlaces(ctx)
	if err != nil {
		return "", err
	}
	for _, p := range all {
		if p.ID == id {
			return p.NameLatn, nil
		}
	}
	return "", nil
}
