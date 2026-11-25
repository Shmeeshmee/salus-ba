package location

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/organization"
)

// The limits of a location (api/openapi.yaml SaveLocationRequest; the table's checks).
const (
	maxNameLength   = 200
	maxPostalLength = 20
	maxZoneLength   = 64
)

// phonePattern is E.164, as the table's check.
var phonePattern = regexp.MustCompile(`^\+[0-9]{8,15}$`)

// Clinics is the public directory of listed clinics (organization.Service).
type Clinics interface {
	ListListed(ctx context.Context) ([]organization.Organization, error)
}

// Locations returns the staff's clinic's locations, active first, by name (GET /clinic/locations). Not audited: a
// location is no personal data. ErrNotFound for a caller of no clinic.
func (s *Service) Locations(ctx context.Context, st Staff) ([]Location, error) {
	if st.OrganizationID == "" {
		return nil, ErrNotFound
	}
	return s.OfClinic(ctx, st.OrganizationID)
}

// OfClinic returns every location of the clinic org, active first, by name, in the clinic's scope: for the clinic's
// own services (schedule's rules) in their transaction.
func (s *Service) OfClinic(ctx context.Context, org string) ([]Location, error) {
	var out []Location
	err := s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.p.Repo.SetOrganization(ctx, org); err != nil {
			return err
		}
		var err error
		out, err = s.p.Repo.Locations(ctx, org)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("location: list: %w", err)
	}
	return out, nil
}

// SaveLocation adds a location to the admin's clinic (no id), or changes one at l.VersionID, the version the admin
// read (POST /clinic/locations, PUT /clinic/locations/{id}), and returns it at its new version. ErrInvalid for a
// location outside the limits; ErrForbidden for the clinic's other staff, ErrNotFound for anyone else and for a
// location that is not the clinic's, ErrVersionConflict for another version: audited as refused. A time zone left out
// is the clinic's for a new location and unchanged for an existing one.
func (s *Service) SaveLocation(ctx context.Context, st Staff, l Location) (Location, error) {
	code, action := auditevent.CodeCreate, auditevent.ActionCreate
	if l.ID != "" {
		code, action = auditevent.CodeUpdate, auditevent.ActionUpdate
		l.VersionID++ // the version it will have
	}
	switch {
	case st.OrganizationID == "":
		return Location{}, s.refused(ctx, st, code, action, l.ID, ErrNotFound)
	case !st.Admin:
		return Location{}, s.refused(ctx, st, code, action, l.ID, ErrForbidden)
	}
	l, err := normalized(l)
	if err != nil {
		return Location{}, err
	}
	l.OrganizationID = st.OrganizationID
	var out Location
	err = s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.p.Repo.SetOrganization(ctx, st.OrganizationID); err != nil {
			return err
		}
		saved, current, err := s.p.Repo.SaveLocation(ctx, l)
		switch {
		case err != nil:
			return err
		case !current:
			return ErrVersionConflict
		}
		out = saved
		return s.p.Audit.Record(ctx, event(code, action, st, "Location/"+out.ID, auditevent.OutcomeSuccess))
	})
	switch {
	case errors.Is(err, ErrNotFound), errors.Is(err, ErrVersionConflict): // the write's transaction rolled back
		return Location{}, s.refused(ctx, st, code, action, l.ID, err)
	case errors.Is(err, ErrInvalid): // an unknown city
		return Location{}, err
	case err != nil:
		return Location{}, fmt.Errorf("location: save: %w", err)
	}
	return out, nil
}

// PublicLocations returns the active locations of a listed clinic, by name, for the patient booking in person (GET
// /organizations/{id}/locations). ErrNotFound for a clinic that is not listed.
func (s *Service) PublicLocations(ctx context.Context, org string) ([]Location, error) {
	listed, err := s.p.Clinics.ListListed(ctx)
	if err != nil {
		return nil, fmt.Errorf("location: listed clinics: %w", err)
	}
	if !slices.ContainsFunc(listed, func(o organization.Organization) bool { return o.ID == org }) {
		return nil, ErrNotFound
	}
	var out []Location
	err = s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.p.Repo.SetDirectory(ctx); err != nil {
			return err
		}
		var err error
		out, err = s.p.Repo.PublicLocations(ctx, org)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("location: public: %w", err)
	}
	return out, nil
}

// refused audits a refused write in a transaction of its own and returns why it was refused, or the auditor's error.
func (s *Service) refused(ctx context.Context, st Staff, code string, action auditevent.Action, id string, why error) error {
	entity := ""
	if id != "" {
		entity = "Location/" + id
	}
	if err := s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		return s.p.Audit.Record(ctx, event(code, action, st, entity, auditevent.OutcomeMinorFailure))
	}); err != nil {
		return fmt.Errorf("location: audit: %w", err)
	}
	return why
}

// normalized is l with its texts trimmed, or ErrInvalid for one outside the limits.
func normalized(l Location) (Location, error) {
	l.Name, l.AddressLine, l.PostalCode = strings.TrimSpace(l.Name), strings.TrimSpace(l.AddressLine), strings.TrimSpace(l.PostalCode)
	l.CityID, l.Phone, l.TimeZone = strings.TrimSpace(l.CityID), strings.TrimSpace(l.Phone), strings.TrimSpace(l.TimeZone)
	switch {
	case l.Name == "", utf8.RuneCountInString(l.Name) > maxNameLength, strings.ContainsRune(l.Name+l.AddressLine+l.PostalCode+l.CityID+l.TimeZone, 0), utf8.RuneCountInString(l.AddressLine) > maxNameLength,
		utf8.RuneCountInString(l.PostalCode) > maxPostalLength,
		l.Phone != "" && !phonePattern.MatchString(l.Phone),
		l.TimeZone != "" && !validZone(l.TimeZone):
		return Location{}, ErrInvalid
	}
	return l, nil
}

// validZone reports whether zone names an IANA time zone (the binaries embed the zone data with time/tzdata).
func validZone(zone string) bool {
	if zone == "Local" || len(zone) > maxZoneLength {
		return false
	}
	_, err := time.LoadLocation(zone)
	return err == nil
}

// event is the audit of a write of a location by st: its reference, or none for a refused new one.
func event(code string, action auditevent.Action, st Staff, entity string, outcome auditevent.Outcome) auditevent.AuditEvent {
	return auditevent.AuditEvent{
		Type:            auditevent.Coding{System: auditevent.SystemAuditEventType, Code: auditevent.CodeRest},
		Subtype:         auditevent.Coding{System: auditevent.SystemRestfulInteraction, Code: code},
		Purpose:         auditevent.Coding{System: auditevent.SystemActReason, Code: auditevent.CodeHOPERAT},
		Action:          action,
		Outcome:         outcome,
		AgentAccountID:  st.AccountID,
		OrganizationID:  st.OrganizationID,
		EntityReference: entity,
	}
}
