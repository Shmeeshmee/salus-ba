package medicationstatement

import (
	"context"
	"errors"
	"fmt"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/auditevent"
)

// Provenance returns the Provenance id of a practitioner's write of a medication to the patient and to staff of the
// writing clinic who see the medication (access.Attributed: it names the clinic; the patient's own Provenances are theirs
// only), audited as a read of it; anyone else gets ErrNotFound, audited as a refused read of it. ok is false, with nothing audited, when id is no medication's Provenance
// the caller's scope sees: another resource's Provenance may have it (GET /fhir/R4/Provenance/{id} has several
// sources).
func (s *Service) Provenance(ctx context.Context, c Caller, id string) (Provenance, bool, error) {
	var out Provenance
	var found, refused bool
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		v, err := s.identify(ctx, c)
		if err != nil || v.Role == 0 {
			return err
		}
		p, err := s.Repo.Provenance(ctx, id)
		switch {
		case errors.Is(err, ErrNotFound):
			return nil
		case err != nil:
			return err
		}
		found = true
		ok, err := s.attributed(ctx, c, v, p)
		switch {
		case err != nil:
			return err
		case !ok:
			refused = true
			return s.audit(ctx, c, auditevent.CodeRead, "Provenance/"+p.ID, auditevent.OutcomeMinorFailure)
		}
		out = p
		return s.audit(ctx, c, auditevent.CodeRead, "Provenance/"+p.ID, auditevent.OutcomeSuccess)
	})
	switch {
	case err != nil:
		return Provenance{}, true, fmt.Errorf("medicationstatement: provenance: %w", err)
	case refused:
		return Provenance{}, true, ErrNotFound
	}
	return out, found, nil
}

// attributed reports whether v sees the medication p is of and may see that p's clinic wrote it.
func (s *Service) attributed(ctx context.Context, c Caller, v access.Viewer, p Provenance) (bool, error) {
	a, err := s.Repo.ByID(ctx, p.EntryID)
	switch {
	case errors.Is(err, ErrNotFound):
		return false, nil
	case err != nil:
		return false, err
	}
	f, err := s.facts(ctx, c, v, a.PatientID)
	if err != nil {
		return false, err
	}
	_, seen := see(v, f, a)
	return seen && access.Attributed(v, f, p.OrganizationID), nil
}

// attest writes the Provenance of the caller's write of a, at the version it produced (activity a v3-DataOperation
// code, agent a provenance-participant-type one), and audits it: by the practitioner role on behalf of the caller's
// clinic, or by the calling patient (roleID empty).
func (s *Service) attest(ctx context.Context, c Caller, roleID string, a MedicationStatement, activity, agent string) (MedicationStatement, error) {
	p := Provenance{Recorded: a.LastUpdated, ID: newID(), EntryID: a.ID, RoleID: roleID, OrganizationID: c.OrganizationID,
		Activity: activity, AgentType: agent, Version: a.VersionID}
	if c.OrganizationID == "" {
		p.PatientID = c.PatientID
	}
	if err := s.Repo.InsertProvenance(ctx, p); err != nil {
		return named(a), err
	}
	return a, s.audit(ctx, c, auditevent.CodeCreate, "Provenance/"+p.ID, auditevent.OutcomeSuccess)
}
