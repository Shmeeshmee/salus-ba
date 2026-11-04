package appointment

import (
	"context"
	"errors"
	"fmt"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/practitionerrole"
)

// Read returns the appointment id as the caller may see it (SPEC.md "Access rule"): the patient their own;
// with an active consent its practitioner and the clinic's admin all of it; after a revocation, a past one at
// the name, times and status only. The patient's name is given at the level Profile allows, and that read is
// audited as a Patient read beside the Appointment's. ErrNotFound to anyone else, audited.
func (s *Service) Read(ctx context.Context, c Caller, id string) (Seen, error) {
	var out Seen
	var missing bool
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		v, err := s.view(ctx, c, id)
		if err != nil {
			return err
		}
		if missing = v.level == access.None; missing {
			return s.record(ctx, c, auditevent.CodeRead, "", "", auditevent.OutcomeMinorFailure)
		}
		practitionerID, err := s.practitionerOf(ctx, map[string]string{}, v.a)
		if err != nil {
			return err
		}
		out, err = s.seen(ctx, c, v, auditevent.CodeRead)
		out.PractitionerID = practitionerID
		return err
	})
	switch {
	case err != nil:
		return Seen{}, fmt.Errorf("appointment: read: %w", err)
	case missing:
		return Seen{}, ErrNotFound
	}
	return out, nil
}

// seen gives v as the caller sees it, with the patient's name where Profile allows it, and audits the
// interaction on the appointment and, when the name is given, on the patient.
func (s *Service) seen(ctx context.Context, c Caller, v view, interaction string) (Seen, error) {
	out := Seen{Appointment: v.a, Level: v.level}
	if v.level == access.Name {
		out.Appointment = minimal(v.a)
	}
	if err := s.record(ctx, c, interaction, "Appointment", v.a.ID, auditevent.OutcomeSuccess); err != nil {
		return Seen{}, err
	}
	if access.Profile(v.viewer, v.facts) == access.None {
		return out, nil
	}
	p, err := s.Patients.Lookup(ctx, v.a.PatientID)
	if err != nil {
		return Seen{}, err
	}
	out.Patient = Name{Given: p.GivenName, Family: p.FamilyName}
	return out, s.record(ctx, c, interaction, "Patient", p.ID, auditevent.OutcomeSuccess)
}

// practitionerOf returns the practitioner of a's role, looked up in a's clinic once per role (kept in known).
func (s *Service) practitionerOf(ctx context.Context, known map[string]string, a Appointment) (string, error) {
	if id, ok := known[a.PractitionerRoleID]; ok {
		return id, nil
	}
	role, err := s.Roles.Lookup(ctx, a.OrganizationID, a.PractitionerRoleID)
	if err != nil {
		return "", err
	}
	known[a.PractitionerRoleID] = role.PractitionerID
	return role.PractitionerID, nil
}

// minimal is a as the name level shows it: the times, the status and the participants.
func minimal(a Appointment) Appointment {
	return Appointment{
		ID: a.ID, PatientID: a.PatientID, PractitionerRoleID: a.PractitionerRoleID, OrganizationID: a.OrganizationID,
		Start: a.Start, End: a.End, Status: a.Status, VersionID: a.VersionID, LastUpdated: a.LastUpdated,
	}
}

// Search returns the appointments of the filter as the caller may see them, by start: a patient only their
// own (whatever patient the filter names), a practitioner only those of their own role (which the filter
// must name, by the role or the practitioner: ErrForbidden otherwise, audited), the clinic's admin the
// clinic's. Each appointment returned is
// audited, and each patient name given; a search that returns none writes one event without an entity.
func (s *Service) Search(ctx context.Context, c Caller, f Filter) ([]Seen, error) {
	var out []Seen
	var refused error
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		var err error
		if out, refused, err = s.search(ctx, c, f); err != nil {
			return err
		}
		if refused != nil {
			return s.record(ctx, c, auditevent.CodeSearchType, "", "", auditevent.OutcomeMinorFailure)
		}
		if len(out) == 0 {
			return s.record(ctx, c, auditevent.CodeSearchType, "", "", auditevent.OutcomeSuccess)
		}
		return nil
	})
	switch {
	case err != nil:
		return nil, fmt.Errorf("appointment: search: %w", err)
	case refused != nil:
		return nil, refused
	}
	return out, nil
}

// search runs the caller's search in the transaction in ctx and audits each result, or gives the refusal.
func (s *Service) search(ctx context.Context, c Caller, f Filter) (out []Seen, refused, err error) {
	f.OrganizationID = ""
	var viewer access.Viewer
	switch {
	case c.patient() && c.PatientID == "":
		return nil, nil, nil
	case c.patient():
		f.PatientID = c.PatientID
		viewer = access.Viewer{Role: access.RolePatient, Patient: c.PatientID}
		err = s.Repo.SetPatient(ctx, c.PatientID)
	default:
		var none bool
		if f, none, refused, err = s.staffFilter(ctx, c, f); none || refused != nil || err != nil {
			return nil, refused, err
		}
		viewer, err = s.staffViewer(ctx, c, f.PractitionerRoleID)
		if err == nil && !c.Admin && (viewer.PractitionerRole == "" || viewer.PractitionerRole != f.PractitionerRoleID) {
			return nil, ErrForbidden, nil
		}
	}
	if err != nil {
		return nil, nil, err
	}
	all, err := s.Repo.Search(ctx, f)
	if err != nil {
		return nil, nil, err
	}
	facts, practitioners := map[string]access.Facts{}, map[string]string{}
	for _, a := range all {
		v, err := s.viewOf(ctx, c, viewer, facts, a)
		if err != nil {
			return nil, nil, err
		}
		if v.level == access.None {
			continue
		}
		practitionerID, err := s.practitionerOf(ctx, practitioners, a)
		if err != nil {
			return nil, nil, err
		}
		if f.PractitionerID != "" && practitionerID != f.PractitionerID { // a patient's search by practitioner
			continue
		}
		seen, err := s.seen(ctx, c, v, auditevent.CodeSearchType)
		if err != nil {
			return nil, nil, err
		}
		seen.PractitionerID = practitionerID
		out = append(out, seen)
	}
	return out, nil, nil
}

// staffFilter scopes the transaction in ctx to the staff caller's clinic and gives f for it, with the role of
// the practitioner f names; none when that practitioner has no role in the clinic, refused when a
// practitioner names another.
func (s *Service) staffFilter(ctx context.Context, c Caller, f Filter) (out Filter, none bool, refused, err error) {
	f.OrganizationID = c.OrganizationID
	if err := s.Repo.SetClinic(ctx, c.OrganizationID); err != nil || f.PractitionerID == "" {
		return f, false, nil, err
	}
	if !c.Admin && f.PractitionerID != c.PractitionerID {
		return f, false, ErrForbidden, nil
	}
	role, err := s.Roles.OfPractitioner(ctx, c.OrganizationID, f.PractitionerID)
	switch {
	case errors.Is(err, practitionerrole.ErrNotFound):
		return f, true, nil, nil
	case err != nil:
		return f, false, nil, err
	}
	f.PractitionerRoleID = role.ID
	return f, false, nil, nil
}

// viewOf decides viewer's level for the appointment a of a search, with the facts of its patient (gathered
// once per patient into facts).
func (s *Service) viewOf(ctx context.Context, c Caller, viewer access.Viewer, facts map[string]access.Facts, a Appointment) (view, error) {
	if c.patient() {
		f := access.Facts{Patient: a.PatientID, Clinic: a.OrganizationID}
		return view{a: a, viewer: viewer, facts: f, level: access.Appointment(viewer, f, visit(a))}, nil
	}
	f, ok := facts[a.PatientID]
	if !ok {
		var err error
		if f, err = s.facts(ctx, a.PatientID, c.OrganizationID); err != nil {
			return view{}, err
		}
		facts[a.PatientID] = f
	}
	return view{a: a, viewer: viewer, facts: f, level: access.Appointment(viewer, f, visit(a))}, nil
}

// Notice returns the appointment id of the clinic org with its patient's account, for the e-mail job agent
// (its kind), whatever the consent: a cancellation after a revocation is mailed too. It reads in the clinic's
// scope narrowed to the one appointment and audits the Appointment and Patient reads with the job as agent.
// ErrNotFound, audited, when there is no such appointment in the clinic.
func (s *Service) Notice(ctx context.Context, agent, org, id string) (Notice, error) {
	var out Notice
	var missing bool
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.Repo.SetClinic(ctx, org); err != nil {
			return err
		}
		if err := s.Repo.SetNotice(ctx, id); err != nil {
			return err
		}
		a, err := s.Repo.ByID(ctx, id)
		if errors.Is(err, ErrNotFound) || (err == nil && a.OrganizationID != org) {
			missing = true
			return s.recordJob(ctx, agent, org, auditevent.CodeRead, "", "", auditevent.OutcomeMinorFailure)
		}
		if err != nil {
			return err
		}
		p, err := s.Patients.Lookup(ctx, a.PatientID)
		if err != nil {
			return err
		}
		out = Notice{Appointment: a, AccountID: p.AccountID}
		if err := s.recordJob(ctx, agent, org, auditevent.CodeRead, "Appointment", a.ID, auditevent.OutcomeSuccess); err != nil {
			return err
		}
		return s.recordJob(ctx, agent, org, auditevent.CodeRead, "Patient", p.ID, auditevent.OutcomeSuccess)
	})
	switch {
	case err != nil:
		return Notice{}, fmt.Errorf("appointment: notice: %w", err)
	case missing:
		return Notice{}, ErrNotFound
	}
	return out, nil
}

// Facts gives the patient domain the appointment facts access needs. It is built on the repository alone, so
// the patient service, which the appointment service reads profiles through, can be built first.
type Facts struct {
	repo Repository
}

// NewFacts returns Facts on repo.
func NewFacts(repo Repository) *Facts {
	return &Facts{repo: repo}
}

// Visits returns the patient's appointments in the clinic org, of any status, as the caller's transaction
// and scope show them. Not audited: it is the access check inside the caller's audited access.
func (f *Facts) Visits(ctx context.Context, patientID, org string) ([]access.Visit, error) {
	all, err := f.repo.Search(ctx, Filter{PatientID: patientID, OrganizationID: org})
	if err != nil {
		return nil, fmt.Errorf("appointment: visits: %w", err)
	}
	return visits(all), nil
}
