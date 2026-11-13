package organization

import (
	"context"
	"fmt"
	"time"

	"github.com/Shmeeshmee/salus/internal/terminology"
)

// Places are the places of a clinic's location (terminology.Catalog): reference data.
type Places interface {
	AllPlaces(ctx context.Context) ([]terminology.Place, error)
}

// Place kinds of a location (terminology's places).
const (
	kindRegion = "region"
	kindCity   = "city"
)

// checkLocation refuses a location in ch that is not a city under the leaf region given with it, or one of the two
// alone (ErrInvalidLocation), and returns the city's name; "" when ch sets no location or clears it.
func (s *Service) checkLocation(ctx context.Context, ch Changes) (string, error) {
	if ch.Region == nil && ch.City == nil {
		return "", nil
	}
	if ch.Region == nil || ch.City == nil || (*ch.Region == "") != (*ch.City == "") {
		return "", ErrInvalidLocation
	}
	if *ch.City == "" {
		return "", nil
	}
	if s.places == nil {
		return "", fmt.Errorf("organization: no places to check the location")
	}
	all, err := s.places.AllPlaces(ctx)
	if err != nil {
		return "", fmt.Errorf("organization: location: %w", err)
	}
	var region, city terminology.Place
	for _, p := range all {
		switch p.ID {
		case *ch.Region:
			region = p
		case *ch.City:
			city = p
		}
	}
	if region.Kind != kindRegion || city.Kind != kindCity || city.Parent != region.ID {
		return "", ErrInvalidLocation
	}
	return city.NameLatn, nil
}

// withLocation is o with the location and the contribution ch sets, each with the time it changed when it
// changes (at); the same value keeps its time.
func withLocation(o Organization, ch Changes, at time.Time) Organization {
	if ch.Region != nil && ch.City != nil && (*ch.Region != o.Region || *ch.City != o.City) {
		o.Region, o.City, o.LocationSince = *ch.Region, *ch.City, at
		if o.City == "" {
			o.LocationSince = time.Time{}
		}
	}
	if ch.AnalyticsContributes != nil && *ch.AnalyticsContributes != o.AnalyticsContributes {
		o.AnalyticsContributes, o.ContributesSince = *ch.AnalyticsContributes, at
	}
	return o
}

// cityName is the Latin name of the city id, "" for none or without places.
func (s *Service) cityName(ctx context.Context, id string) (string, error) {
	if id == "" || s.places == nil {
		return "", nil
	}
	all, err := s.places.AllPlaces(ctx)
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

// Contributes tells whether the clinic organizationID contributes to the platform figures, and so whether its
// staff see them (docs/plans/analytics.md O6). Read like TranscriptCopy: in the clinic's scope, unaudited;
// ErrNotFound for none.
func (s *Service) Contributes(ctx context.Context, organizationID string) (bool, error) {
	o, err := s.byID(ctx, organizationID)
	if err != nil {
		return false, fmt.Errorf("organization: contributes: %w", err)
	}
	return o.AnalyticsContributes, nil
}
