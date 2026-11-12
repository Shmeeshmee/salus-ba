package encounter

import (
	"context"
	"fmt"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/pkg/keyset"
)

// PatientHistory returns the calling patient's own encounters at every clinic, or at the clinic org when set,
// after the page's position, newest first by start, p.Limit at most: the visits of their timeline (SAL-39,
// docs/plans/patient-history.md), which the patient always sees (access.HistoryVisit). Read in the patient's own
// scope and audited, each one returned as a search of their own request (in one insert); none is one event without an entity.
// The timeline is the patient's: staff get ErrForbidden, audited. A patient without a profile has none.
func (s *Service) PatientHistory(ctx context.Context, c appointment.Caller, org string, p keyset.Page) ([]Encounter, error) {
	var out []Encounter
	staff := c.OrganizationID != ""
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		switch {
		case staff:
			return s.record(ctx, c, auditevent.CodeSearchType, "", auditevent.OutcomeMinorFailure)
		case c.PatientID != "":
			all, err := s.ownPage(ctx, c, org, p)
			if err != nil {
				return err
			}
			out = all
		}
		if len(out) == 0 {
			return s.record(ctx, c, auditevent.CodeSearchType, "", auditevent.OutcomeSuccess)
		}
		events := make([]auditevent.AuditEvent, len(out))
		for i, e := range out {
			events[i] = callerEntry(c, auditevent.CodeSearchType, e.ID, auditevent.OutcomeSuccess)
		}
		return s.Audit.RecordAll(ctx, events) // one insert for the page
	})
	switch {
	case err != nil:
		return nil, fmt.Errorf("encounter: patient history: %w", err)
	case staff:
		return nil, ErrForbidden
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
		all, err := s.Repo.Search(ctx, Filter{PatientID: c.PatientID})
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
