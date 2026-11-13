package encounter

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/practitionerrole"
	"github.com/Shmeeshmee/salus/pkg/fhir"
	"github.com/Shmeeshmee/salus/pkg/keyset"
)

// History returns the patient's finished encounters the practitioner caller may see in their history
// (access.HistoryVisit, ADR 0014): the ones they held themselves at their clinic, and the other clinics' the
// patient shared with it (Shared), newest first. ErrNotFound when the caller may not see the patient's history at
// all: anyone but a practitioner of the clinic with an appointment with the patient under an active care consent
// (access.Profile at Full), the clinic's admin included. Read in the clinic's scope in the transaction in ctx (or
// one of its own) and not audited: it is the history's access decision, and the history audits each item it shows
// (Disclosed, and the other domains' for theirs) and the view.
func (s *Service) History(ctx context.Context, c appointment.Caller, patientID string) ([]Visit, error) {
	var out []Visit
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		v, f, err := s.historyFacts(ctx, c, patientID)
		if err != nil {
			return err
		}
		if access.Profile(v, f) != access.Full {
			return ErrNotFound
		}
		all, err := s.Repo.Search(ctx, Filter{PatientID: patientID})
		if err != nil {
			return err
		}
		for _, e := range all {
			if e.Status == StatusFinished && e.PatientID == patientID && access.HistoryVisit(v, f, record(e)) {
				out = append(out, Visit{Encounter: e, Shared: e.OrganizationID != c.OrganizationID})
			}
		}
		return nil
	})
	switch {
	case errors.Is(err, ErrNotFound):
		return nil, ErrNotFound
	case err != nil:
		return nil, fmt.Errorf("encounter: history: %w", err)
	}
	slices.SortFunc(out, func(a, b Visit) int {
		return cmp.Or(b.PeriodStart.Compare(a.PeriodStart), cmp.Compare(b.ID, a.ID))
	})
	return out, nil
}

// Shared returns the encounter id when it is another clinic's visit the caller may see in the patient's history
// (shared with the caller's clinic, access.HistoryVisit): for a service that reads what belongs to it (its signed
// documents, its completed questionnaire answers and their files) and audits its own read. A visit of the caller's
// own clinic is not one: its own rule applies. In the clinic's scope in the transaction in ctx (or one of its own),
// unaudited; ErrNotFound otherwise.
func (s *Service) Shared(ctx context.Context, c appointment.Caller, id string) (Encounter, error) {
	return s.shared(ctx, c, "shared", func(ctx context.Context) (Encounter, error) { return s.Repo.ByID(ctx, id) })
}

// SharedAppointment is Shared for the encounter of the appointment id, which the caller's scope does not show.
func (s *Service) SharedAppointment(ctx context.Context, c appointment.Caller, appointmentID string) (Encounter, error) {
	return s.shared(ctx, c, "shared appointment", func(ctx context.Context) (Encounter, error) {
		if !fhir.IsUUID(appointmentID) {
			return Encounter{}, ErrNotFound
		}
		all, err := s.Repo.Search(ctx, Filter{AppointmentID: appointmentID})
		if err != nil || len(all) != 1 {
			return Encounter{}, cmp.Or(err, ErrNotFound)
		}
		return all[0], nil
	})
}

func (s *Service) shared(ctx context.Context, c appointment.Caller, op string, find func(ctx context.Context) (Encounter, error)) (Encounter, error) {
	var out Encounter
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		if c.OrganizationID == "" {
			return ErrNotFound
		}
		if err := s.Repo.SetClinic(ctx, c.OrganizationID); err != nil {
			return err
		}
		e, err := find(ctx)
		switch {
		case err != nil:
			return err
		case e.OrganizationID == c.OrganizationID || e.Status != StatusFinished:
			return ErrNotFound
		}
		v, f, err := s.historyFacts(ctx, c, e.PatientID)
		if err != nil {
			return err
		}
		if !access.HistoryVisit(v, f, record(e)) {
			return ErrNotFound
		}
		out = e
		return nil
	})
	switch {
	case errors.Is(err, ErrNotFound):
		return Encounter{}, ErrNotFound
	case err != nil:
		return Encounter{}, fmt.Errorf("encounter: %s: %w", op, err)
	}
	return out, nil
}

// Disclosed audits the visits ids the history shows the caller, each as a search of it, in the transaction in ctx
// (or one of its own).
func (s *Service) Disclosed(ctx context.Context, c appointment.Caller, ids []string) error {
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		for _, id := range ids {
			if err := s.record(ctx, c, auditevent.CodeSearchType, id, auditevent.OutcomeSuccess); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("encounter: disclosed: %w", err)
	}
	return nil
}

// historyFacts are the practitioner caller as access sees them and the patient's facts in their clinic (the consent
// state with the shared visits, the appointments), with the transaction in ctx scoped to the clinic; ErrNotFound for
// anyone but a practitioner with a role there, or a patient id that is none.
func (s *Service) historyFacts(ctx context.Context, c appointment.Caller, patientID string) (access.Viewer, access.Facts, error) {
	if c.OrganizationID == "" || c.Admin || c.PractitionerID == "" || !fhir.IsUUID(patientID) {
		return access.Viewer{}, access.Facts{}, ErrNotFound
	}
	role, err := s.Roles.OfPractitioner(ctx, c.OrganizationID, c.PractitionerID)
	switch {
	case errors.Is(err, practitionerrole.ErrNotFound):
		return access.Viewer{}, access.Facts{}, ErrNotFound
	case err != nil:
		return access.Viewer{}, access.Facts{}, err
	}
	if err := s.Repo.SetClinic(ctx, c.OrganizationID); err != nil {
		return access.Viewer{}, access.Facts{}, err
	}
	state, err := s.Consents.State(ctx, patientID, c.OrganizationID)
	if err != nil {
		return access.Viewer{}, access.Facts{}, err
	}
	visits, err := s.Visits.Visits(ctx, patientID, c.OrganizationID)
	if err != nil {
		return access.Viewer{}, access.Facts{}, err
	}
	v := access.Viewer{Role: access.RolePractitioner, Clinic: c.OrganizationID, PractitionerRole: role.ID}
	return v, access.Facts{Patient: patientID, Clinic: c.OrganizationID, Visits: visits, Consent: state}, nil
}

// record is the encounter as the history's rule sees it.
func record(e Encounter) access.Record {
	return access.Record{Encounter: e.ID, Clinic: e.OrganizationID, PractitionerRole: e.PractitionerRoleID}
}

// PatientHistory returns the calling patient's own encounters at every clinic, or at the clinic org when set,
// after the page's position, newest first by start, p.Limit at most: the visits of their timeline (SAL-39,
// docs/plans/patient-history.md), which the patient always sees (access.HistoryVisit). Read in the patient's own
// scope, joining the transaction in ctx, unaudited: history.Timeline audits the items of the page it returns. The
// timeline is the patient's: staff get ErrForbidden. A patient without a profile has none.
func (s *Service) PatientHistory(ctx context.Context, c appointment.Caller, org string, p keyset.Page) ([]Encounter, error) {
	switch {
	case c.OrganizationID != "":
		return nil, ErrForbidden
	case c.PatientID == "":
		return nil, nil
	}
	var out []Encounter
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		var err error
		out, err = s.ownPage(ctx, c, org, p)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("encounter: patient history: %w", err)
	}
	return out, nil
}

// ownPage reads the patient's page in their scope, in the transaction in ctx, keeping only what
// access.HistoryVisit lets them see: their own.
func (s *Service) ownPage(ctx context.Context, c appointment.Caller, org string, p keyset.Page) ([]Encounter, error) {
	if err := s.Repo.SetPatient(ctx, c.PatientID); err != nil {
		return nil, err
	}
	all, err := s.Repo.PatientHistory(ctx, c.PatientID, org, p)
	if err != nil {
		return nil, err
	}
	viewer := access.Viewer{Role: access.RolePatient, Patient: c.PatientID}
	out := all[:0]
	for _, e := range all {
		facts := access.Facts{Patient: e.PatientID, Clinic: e.OrganizationID}
		if access.HistoryVisit(viewer, facts, access.Record{Encounter: e.ID, Clinic: e.OrganizationID, PractitionerRole: e.PractitionerRoleID}) {
			out = append(out, e)
		}
	}
	return out, nil
}

// Own returns the calling patient's own encounters, at the clinic org when set, read in their scope, unaudited:
// for a service that lists what belongs to them (the signed documents of their timeline) and audits its own
// decision. Staff and a patient without a profile get none.
func (s *Service) Own(ctx context.Context, c appointment.Caller, org string) ([]Encounter, error) {
	if c.OrganizationID != "" || c.PatientID == "" {
		return nil, nil
	}
	var out []Encounter
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.Repo.SetPatient(ctx, c.PatientID); err != nil {
			return err
		}
		all, err := s.Repo.OfPatient(ctx, c.PatientID)
		for _, e := range all {
			if e.PatientID == c.PatientID && (org == "" || e.OrganizationID == org) {
				out = append(out, e)
			}
		}
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("encounter: own: %w", err)
	}
	return out, nil
}
