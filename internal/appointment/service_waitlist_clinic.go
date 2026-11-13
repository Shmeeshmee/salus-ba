package appointment

import (
	"context"
	"errors"
	"fmt"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/practitionerrole"
)

// The clinic's side of the waitlist (SAL-35): its list in offer order with the names access.Profile gives, and the
// priority staff move an entry to.

// ClinicWaitlist returns the waitlist of the staff caller's clinic in offer order (priority descending, then join
// time), each entry with its open offer and its patient's name where access.Profile gives it; with the filter's role,
// the entries it could be offered (its own and its specialty's), with its status, those in it. Each entry returned and
// each name given is audited. ErrForbidden to a patient.
func (s *Service) ClinicWaitlist(ctx context.Context, c Caller, f WaitlistFilter) ([]ClinicEntry, error) {
	if c.patient() {
		return nil, s.refuse(ctx, c, auditevent.CodeSearchType, ErrForbidden)
	}
	var out []ClinicEntry
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		var err error
		out, err = s.clinicWaitlist(ctx, c, f)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("appointment: clinic waitlist: %w", err)
	}
	return out, nil
}

func (s *Service) clinicWaitlist(ctx context.Context, c Caller, f WaitlistFilter) ([]ClinicEntry, error) {
	org := c.OrganizationID
	if err := s.Repo.SetClinic(ctx, org); err != nil {
		return nil, err
	}
	specialty := ""
	if f.PractitionerRoleID != "" {
		role, err := s.Roles.Lookup(ctx, org, f.PractitionerRoleID)
		switch {
		case errors.Is(err, practitionerrole.ErrNotFound):
			return nil, s.record(ctx, c, auditevent.CodeSearchType, "", "", auditevent.OutcomeSuccess)
		case err != nil:
			return nil, err
		}
		specialty = role.SpecialtyCode
	}
	all, err := s.Waitlist.ClinicEntries(ctx, org, f.PractitionerRoleID, specialty)
	if err != nil {
		return nil, err
	}
	n := namer{s: s, c: c, facts: map[string]access.Facts{}}
	var out []ClinicEntry
	for _, e := range all {
		e = expired(e, s.Clock.Now())
		if e.OrganizationID != org || (f.Status != "" && e.Status != f.Status) {
			continue
		}
		ce, err := n.name(ctx, e)
		if err != nil {
			return nil, err
		}
		out = append(out, ce)
	}
	if len(out) == 0 {
		return nil, s.record(ctx, c, auditevent.CodeSearchType, "", "", auditevent.OutcomeSuccess)
	}
	return out, nil
}

// namer gives a clinic's entries their patients' names as access.Profile allows the staff caller, reading each
// patient's facts once.
type namer struct {
	s      *Service
	facts  map[string]access.Facts
	c      Caller
	viewer access.Viewer
	ready  bool
}

// name audits the entry e and, where the caller may see it, its patient's name.
func (n *namer) name(ctx context.Context, e WaitlistEntry) (ClinicEntry, error) {
	if err := n.s.record(ctx, n.c, auditevent.CodeSearchType, "Appointment", e.ID, auditevent.OutcomeSuccess); err != nil {
		return ClinicEntry{}, err
	}
	if !n.ready {
		v, err := n.s.ownViewer(ctx, n.c)
		if err != nil {
			return ClinicEntry{}, err
		}
		n.viewer, n.ready = v, true
	}
	f, ok := n.facts[e.PatientID]
	if !ok {
		var err error
		if f, err = n.s.facts(ctx, e.PatientID, n.c.OrganizationID); err != nil {
			return ClinicEntry{}, err
		}
		n.facts[e.PatientID] = f
	}
	out := ClinicEntry{Entry: e}
	if access.Profile(n.viewer, f) == access.None {
		return out, nil
	}
	p, err := n.s.Patients.Lookup(ctx, e.PatientID)
	if err != nil {
		return ClinicEntry{}, err
	}
	out.Patient, out.Named = Name{Given: p.GivenName, Family: p.FamilyName}, true
	return out, n.s.record(ctx, n.c, auditevent.CodeSearchType, "Patient", p.ID, auditevent.OutcomeSuccess)
}

// ownViewer is staff caller c as access sees them, with their own role in the clinic for a practitioner.
func (s *Service) ownViewer(ctx context.Context, c Caller) (access.Viewer, error) {
	if c.Admin || c.PractitionerID == "" {
		return s.staffViewer(ctx, c, "")
	}
	role, err := s.Roles.OfPractitioner(ctx, c.OrganizationID, c.PractitionerID)
	switch {
	case errors.Is(err, practitionerrole.ErrNotFound):
		return s.staffViewer(ctx, c, "")
	case err != nil:
		return access.Viewer{}, err
	}
	return s.staffViewer(ctx, c, role.ID)
}

// SetPriority moves the active entry id of the staff caller's clinic to priority (0 to 100) and returns it as the
// clinic sees it. Audited, refusals too. ErrInvalidPriority (unaudited, before any access), ErrForbidden to a patient,
// ErrNotFound for an entry the clinic does not see.
func (s *Service) SetPriority(ctx context.Context, c Caller, id string, priority int) (ClinicEntry, error) {
	if priority < 0 || priority > maxPriority {
		return ClinicEntry{}, ErrInvalidPriority
	}
	if c.patient() {
		return ClinicEntry{}, s.refuse(ctx, c, auditevent.CodeUpdate, ErrForbidden)
	}
	var out ClinicEntry
	var refused error
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		e, refusal, err := s.entryOf(ctx, c, id)
		if refused = refusal; err != nil || refused != nil {
			if err == nil {
				err = s.record(ctx, c, auditevent.CodeUpdate, "", "", auditevent.OutcomeMinorFailure)
			}
			return err
		}
		if e, err = s.Waitlist.SetPriority(ctx, e.ID, priority, s.Clock.Now()); err != nil {
			return err
		}
		if err := s.record(ctx, c, auditevent.CodeUpdate, "Appointment", e.ID, auditevent.OutcomeSuccess); err != nil {
			return err
		}
		n := namer{s: s, c: c, facts: map[string]access.Facts{}}
		out, err = n.name(ctx, e)
		return err
	})
	switch {
	case err != nil:
		return ClinicEntry{}, fmt.Errorf("appointment: waitlist priority: %w", err)
	case refused != nil:
		return ClinicEntry{}, refused
	}
	return out, nil
}
