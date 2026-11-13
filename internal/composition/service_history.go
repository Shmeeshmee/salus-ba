package composition

import (
	"context"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/encounter"
	"github.com/Shmeeshmee/salus/pkg/keyset"
)

// History returns the signed documents of visits, each with its visit, in the order of visits and then of signing:
// for the practitioner's view of the patient's history (SAL-40), with visits exactly as encounter.Service.History
// gave them to the caller (access.HistoryVisit applied). Never a draft. Read in the caller's clinic scope (the
// shared visits' documents through document_draft_shared) and unaudited: the history audits the ones it shows
// (Disclosed). ErrForbidden for anyone but a practitioner.
func (s *Service) History(ctx context.Context, c appointment.Caller, visits []encounter.Visit) ([]Document, error) {
	var out []Document
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		if c.OrganizationID == "" || c.Admin {
			return errRefused{ErrForbidden}
		}
		if err := s.Repo.SetClinic(ctx, c.OrganizationID); err != nil {
			return err
		}
		ids := make([]string, len(visits))
		for i, v := range visits {
			ids[i] = v.ID
		}
		// One query for every visit, not one each (PROGRESS, SAL-40).
		drafts, err := s.Repo.SignedOfVisits(ctx, ids)
		if err != nil {
			return err
		}
		byEncounter := map[string][]Draft{}
		for _, d := range drafts {
			byEncounter[d.EncounterID] = append(byEncounter[d.EncounterID], d)
		}
		for _, v := range visits {
			for _, d := range byEncounter[v.ID] {
				out = append(out, Document{Draft: d, Encounter: v.Encounter})
			}
		}
		return nil
	})
	if r := refusal(err); r != nil {
		return nil, r
	}
	return out, wrapErr("history", err, nil)
}

// Disclosed audits the signed documents ids (Composition ids) the history shows the caller, each as a search of
// it, in the transaction in ctx (or one of its own).
func (s *Service) Disclosed(ctx context.Context, c appointment.Caller, ids []string) error {
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		for _, id := range ids {
			if err := s.record(ctx, entry{c: c, interaction: auditevent.CodeSearchType, reference: "Composition/" + id}); err != nil {
				return err
			}
		}
		return nil
	})
	return wrapErr("disclosed", err, nil)
}

// PatientHistory returns the signed documents of the calling patient's own encounters at every clinic, or at the
// clinic org when set, each with its encounter, after the page's position, newest first by signature, p.Limit at
// most: the documents of their timeline (SAL-39), which the patient always sees (access.SignedDocument). Never a
// draft. Read in the patient's own scope, joining the transaction in ctx, unaudited: history.Timeline audits the
// items of the page it returns. The timeline is the patient's: staff get ErrForbidden.
func (s *Service) PatientHistory(ctx context.Context, c appointment.Caller, org string, p keyset.Page) ([]Document, error) {
	if c.OrganizationID != "" {
		return nil, ErrForbidden
	}
	var out []Document
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		var err error
		out, err = s.signedOf(ctx, c, org, p)
		return err
	})
	return out, wrapErr("patient history", err, nil)
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
