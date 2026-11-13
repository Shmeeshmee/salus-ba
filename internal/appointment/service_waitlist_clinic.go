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

// The pages of the clinic's waitlist.
const (
	defaultWaitlistPage = 100
	maxWaitlistPage     = 500
)

// ClinicWaitlist returns a page of the waitlist of the staff caller's clinic in offer order (priority descending, then
// join time), each entry with its open offer and its patient's name where access.Profile gives it; with the filter's
// role, the entries it could be offered (its own and its specialty's); with its status, those in it (active when
// none: not booked, withdrawn or past their last day). Each entry returned and each name given is audited, in one
// insert. ErrForbidden to a patient.
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
	q := ClinicQuery{
		Now: s.Clock.Now(), OrganizationID: org, PractitionerRoleID: f.PractitionerRoleID, Status: f.Status,
		Limit: f.Limit, Offset: max(f.Offset, 0),
	}
	if q.Status == "" {
		q.Status = WaitlistActive
	}
	if q.Limit <= 0 || q.Limit > maxWaitlistPage {
		q.Limit = defaultWaitlistPage
	}
	if f.PractitionerRoleID != "" {
		role, err := s.Roles.Lookup(ctx, org, f.PractitionerRoleID)
		switch {
		case errors.Is(err, practitionerrole.ErrNotFound):
			return nil, s.record(ctx, c, auditevent.CodeSearchType, "", "", auditevent.OutcomeSuccess)
		case err != nil:
			return nil, err
		}
		q.Specialty = role.SpecialtyCode
	}
	all, err := s.Waitlist.ClinicEntries(ctx, q)
	if err != nil {
		return nil, err
	}
	page := make([]WaitlistEntry, 0, len(all))
	for _, e := range all {
		if e.OrganizationID == org { // not on row-level security alone
			page = append(page, expired(e, q.Now))
		}
	}
	if len(page) == 0 {
		return nil, s.record(ctx, c, auditevent.CodeSearchType, "", "", auditevent.OutcomeSuccess)
	}
	return s.named(ctx, c, page)
}

// named gives the clinic's entries their patients' names as access.Profile allows the staff caller: the caller's own
// visits read once (a practitioner's role), each patient's consent and name once, and every entry and name audited in
// one insert.
func (s *Service) named(ctx context.Context, c Caller, entries []WaitlistEntry) ([]ClinicEntry, error) {
	viewer, err := s.ownViewer(ctx, c)
	if err != nil {
		return nil, err
	}
	visits := map[string][]access.Visit{}
	if viewer.PractitionerRole != "" {
		appts, err := s.Repo.Search(ctx, Filter{OrganizationID: c.OrganizationID, PractitionerRoleID: viewer.PractitionerRole})
		if err != nil {
			return nil, err
		}
		for _, a := range appts {
			visits[a.PatientID] = append(visits[a.PatientID], visit(a))
		}
	}
	names := map[string]*Name{} // nil: not to be shown
	out := make([]ClinicEntry, 0, len(entries))
	events := make([]auditevent.AuditEvent, 0, 2*len(entries))
	for _, e := range entries {
		events = append(events, s.event(c, auditevent.CodeSearchType, "Appointment", e.ID))
		name, seen := names[e.PatientID]
		if !seen {
			if name, err = s.nameOf(ctx, c, viewer, e.PatientID, visits[e.PatientID]); err != nil {
				return nil, err
			}
			names[e.PatientID] = name
			if name != nil {
				events = append(events, s.event(c, auditevent.CodeSearchType, "Patient", e.PatientID))
			}
		}
		ce := ClinicEntry{Entry: e}
		if name != nil {
			ce.Patient, ce.Named = *name, true
		}
		out = append(out, ce)
	}
	return out, s.Audit.RecordAll(ctx, events)
}

// nameOf is the patient's name where access.Profile gives it to viewer, from the patient's consent with the clinic and
// the viewer's visits with them; nil where it does not.
func (s *Service) nameOf(ctx context.Context, c Caller, viewer access.Viewer, patientID string, visits []access.Visit) (*Name, error) {
	state, err := s.Consents.State(ctx, patientID, c.OrganizationID)
	if err != nil {
		return nil, err
	}
	f := access.Facts{Patient: patientID, Clinic: c.OrganizationID, Consent: state, Visits: visits}
	if access.Profile(viewer, f) == access.None {
		return nil, nil
	}
	p, err := s.Patients.Lookup(ctx, patientID)
	if err != nil {
		return nil, err
	}
	return &Name{Given: p.GivenName, Family: p.FamilyName}, nil
}

// event is the audit event of interaction by staff caller c on resource/id.
func (s *Service) event(c Caller, interaction, resource, id string) auditevent.AuditEvent {
	e := entry(interaction, resource, id, auditevent.OutcomeSuccess, auditevent.CodeTREAT)
	e.AgentAccountID, e.OrganizationID = c.AccountID, c.OrganizationID
	return e
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
		named, err := s.named(ctx, c, []WaitlistEntry{e})
		if err == nil {
			out = named[0]
		}
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
