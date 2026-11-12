package composition

import (
	"context"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/encounter"
	"github.com/Shmeeshmee/salus/pkg/keyset"
)

// PatientHistory returns the signed documents of the calling patient's own encounters at every clinic, or at the
// clinic org when set, each with its encounter, after the page's position, newest first by signature, p.Limit at
// most: the documents of their timeline (SAL-39), which the patient always sees (access.SignedDocument). Never a
// draft. Read in the patient's own scope and audited, each one returned as a search of their own request in its
// clinic; none is one event without an entity. The timeline is the patient's: staff get ErrForbidden, audited.
func (s *Service) PatientHistory(ctx context.Context, c appointment.Caller, org string, p keyset.Page) ([]Document, error) {
	var out []Document
	var refused error
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		e := entry{c: c, interaction: auditevent.CodeSearchType}
		if c.OrganizationID != "" {
			refused, e.outcome = ErrForbidden, auditevent.OutcomeMinorFailure
			return s.record(ctx, e)
		}
		var err error
		if out, err = s.signedOf(ctx, c, org, p); err != nil || len(out) == 0 {
			return firstErr(err, s.record(ctx, e))
		}
		for _, d := range out {
			e.reference, e.org = "Composition/"+d.Draft.ID, d.Encounter.OrganizationID
			if err := s.record(ctx, e); err != nil {
				return err
			}
		}
		return nil
	})
	return out, wrapErr("patient history", err, refused)
}

// signedOf reads the page of the signed documents of the patient's encounters in their scope, in the transaction
// in ctx, keeping those access.SignedDocument lets them see.
func (s *Service) signedOf(ctx context.Context, c appointment.Caller, org string, p keyset.Page) ([]Document, error) {
	encs, err := s.Encounters.Own(ctx, c, org)
	if err != nil || len(encs) == 0 {
		return nil, err
	}
	byID := make(map[string]encounter.Encounter, len(encs))
	ids := make([]string, len(encs))
	for i, e := range encs {
		byID[e.ID], ids[i] = e, e.ID
	}
	if err := s.Repo.SetPatient(ctx, c.PatientID); err != nil {
		return nil, err
	}
	drafts, err := s.Repo.SignedOf(ctx, ids, p)
	if err != nil {
		return nil, err
	}
	viewer := access.Viewer{Role: access.RolePatient, Patient: c.PatientID}
	out := make([]Document, 0, len(drafts))
	for _, d := range drafts {
		enc := byID[d.EncounterID]
		facts := access.Facts{Patient: enc.PatientID, Clinic: enc.OrganizationID}
		if access.SignedDocument(viewer, facts, access.Visit{PractitionerRole: enc.PractitionerRoleID}, d.Status == StatusFinal) {
			out = append(out, Document{Draft: d, Encounter: enc})
		}
	}
	return out, nil
}

// firstErr is the first error that is not nil.
func firstErr(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}
