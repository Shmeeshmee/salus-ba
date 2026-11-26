package encounter

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/pkg/keyset"
)

// Read returns the encounter id to whoever access.Appointment gives the name or more of its appointment (its
// patient; with an active consent its practitioner and the clinic's admin; after a revocation, for a past
// appointment, at the name level), read in the caller's own scope and audited. ErrNotFound to anyone else,
// audited without the entity.
func (s *Service) Read(ctx context.Context, c appointment.Caller, id string) (Encounter, error) {
	var out Encounter
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		e, err := s.read(ctx, c, id)
		if errors.Is(err, ErrNotFound) {
			return s.record(ctx, c, auditevent.CodeRead, "", auditevent.OutcomeMinorFailure)
		}
		if err != nil {
			return err
		}
		out = e
		return s.record(ctx, c, auditevent.CodeRead, e.ID, auditevent.OutcomeSuccess)
	})
	switch {
	case err != nil:
		return Encounter{}, fmt.Errorf("encounter: read: %w", err)
	case out.ID == "":
		return Encounter{}, ErrNotFound
	}
	return out, nil
}

// read gives the encounter id when the caller may see its appointment, in the transaction in ctx.
func (s *Service) read(ctx context.Context, c appointment.Caller, id string) (Encounter, error) {
	ok, err := s.scope(ctx, c)
	switch {
	case err != nil:
		return Encounter{}, err
	case !ok:
		return Encounter{}, ErrNotFound
	}
	e, err := s.Repo.ByID(ctx, id)
	if err != nil {
		return Encounter{}, err
	}
	_, ok, err = s.attendance(ctx, c, e)
	switch {
	case err != nil:
		return Encounter{}, err
	case !ok:
		return Encounter{}, ErrNotFound
	}
	return e, nil
}

// scope scopes the transaction in ctx to the caller: a patient's own, or the staff caller's clinic; false for
// a patient without a profile, who has none.
func (s *Service) scope(ctx context.Context, c appointment.Caller) (bool, error) {
	if c.OrganizationID != "" {
		return true, s.Repo.SetClinic(ctx, c.OrganizationID)
	}
	if c.PatientID == "" {
		return false, nil
	}
	return true, s.Repo.SetPatient(ctx, c.PatientID)
}

// attendance gives the caller's attendance of e's appointment, and whether it lets them see e.
func (s *Service) attendance(ctx context.Context, c appointment.Caller, e Encounter) (appointment.Attendance, bool, error) {
	att, err := s.Appointments.Attendance(ctx, c, e.AppointmentID)
	switch {
	case errors.Is(err, appointment.ErrNotFound):
		return appointment.Attendance{}, false, nil
	case err != nil:
		return appointment.Attendance{}, false, err
	}
	return att, att.Level == access.Name || att.Level == access.Full, nil
}

// Search returns the encounters of the filter as the caller may see them, scoped like the Appointment
// search: a patient only their own (whatever patient the filter names), a practitioner only those of their
// own role (ErrForbidden, audited, when the filter names another practitioner), the clinic's admin the
// clinic's. PractitionerID is the practitioner behind the role. Each encounter returned is audited; a search
// that returns none writes one event without an entity.
func (s *Service) Search(ctx context.Context, c appointment.Caller, f Filter) ([]Encounter, error) {
	f.AfterStart, f.AfterID = time.Time{}, ""
	p, err := s.find(ctx, c, f, 0)
	return p.Encounters, err
}

// SearchPage is Search, a page at a time, for the FHIR search (api/fhir/CapabilityStatement.json): the first count
// encounters after f's position (AfterStart, AfterID) in the search's order, by period start then id. A staff search
// not narrowed to one appointment (which has one encounter at most) returns the clinic's or the role's whole history,
// so it is paged, by appointment.DefaultCount unless count asks for another number; so is any search given a count or
// a position. A patient's own returns every match, as Search does. The encounters the caller may not see are read
// past and left out as Search leaves them out; the page's are audited as Search audits them.
func (s *Service) SearchPage(ctx context.Context, c appointment.Caller, f Filter, count int) (Page, error) {
	if count == 0 && (f.AfterID != "" || (c.OrganizationID != "" && f.AppointmentID == "")) {
		count = appointment.DefaultCount
	}
	return s.find(ctx, c, f, count)
}

// find is Search, a page of count when count is set: it runs the search in a transaction and audits a refusal or
// an empty result with one event.
func (s *Service) find(ctx context.Context, c appointment.Caller, f Filter, count int) (Page, error) {
	var out Page
	var refused error
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		var err error
		if out, refused, err = s.search(ctx, c, f, count); err != nil {
			return err
		}
		switch {
		case refused != nil:
			return s.record(ctx, c, auditevent.CodeSearchType, "", auditevent.OutcomeMinorFailure)
		case len(out.Encounters) == 0:
			return s.record(ctx, c, auditevent.CodeSearchType, "", auditevent.OutcomeSuccess)
		}
		return nil
	})
	switch {
	case err != nil:
		return Page{}, fmt.Errorf("encounter: search: %w", err)
	case refused != nil:
		return Page{}, refused
	}
	return out, nil
}

// search runs the caller's search in the transaction in ctx and audits each result, or gives the refusal.
func (s *Service) search(ctx context.Context, c appointment.Caller, f Filter, count int) (out Page, refused, err error) {
	staff, practitioner := c.OrganizationID != "", c.OrganizationID != "" && !c.Admin
	switch {
	case practitioner && f.PractitionerID != "" && f.PractitionerID != c.PractitionerID:
		return Page{}, ErrForbidden, nil
	case !staff:
		f.PatientID = c.PatientID
	}
	if ok, err := s.scope(ctx, c); err != nil || !ok {
		return Page{}, nil, err
	}
	f.Limit = 0
	keep := func(batch []Encounter) ([]Encounter, error) {
		var kept []Encounter
		for _, e := range batch {
			att, ok, err := s.attendance(ctx, c, e)
			switch {
			case err != nil:
				return nil, err
			case !ok, practitioner && !att.Practitioner, f.PractitionerID != "" && att.PractitionerID != f.PractitionerID:
				continue
			}
			kept = append(kept, e)
		}
		return kept, nil
	}
	if count == 0 {
		var all []Encounter
		if all, err = s.Repo.Search(ctx, f); err == nil {
			out.Encounters, err = keep(all)
		}
	} else {
		out.Encounters, out.More, err = keyset.Fill(count, func(after *Encounter, limit int) ([]Encounter, error) {
			g := f
			if after != nil {
				g.AfterStart, g.AfterID = after.PeriodStart, after.ID
			}
			g.Limit = limit
			return s.Repo.Search(ctx, g)
		}, keep)
	}
	if err != nil {
		return Page{}, nil, err
	}
	for _, e := range out.Encounters {
		if err := s.record(ctx, c, auditevent.CodeSearchType, e.ID, auditevent.OutcomeSuccess); err != nil {
			return Page{}, nil, err
		}
	}
	return out, nil, nil
}
