package consent

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"

	"github.com/Shmeeshmee/salus/internal/auditevent"
)

// Visits gives the visits a patient may share with a clinic (history.Sharing): their finished visits at every other
// clinic, newest first, as the preview shows them (Selected unset), each read audited by its owning domain, in the
// transaction in ctx; ErrClinicNotFound when the clinic does not exist.
type Visits interface {
	Shareable(ctx context.Context, c Caller, organizationID string) ([]Candidate, error)
}

// Candidates returns the visits the calling patient may share with the clinic organizationID, each marked Selected
// when their history consent with it lists the visit now (docs/plans/patient-history.md, SAL-42): the preview at
// booking and in settings, exactly as the clinic would see each visit. An empty list without a profile.
// ErrClinicNotFound for an unknown clinic, ErrForbidden to staff. Audited as a search of the history consent.
func (s *Service) Candidates(ctx context.Context, c Caller, organizationID string) ([]Candidate, error) {
	if !c.patient() {
		return nil, s.refuse(ctx, c, auditevent.CodeSearchType, ErrForbidden)
	}
	out := []Candidate{}
	var refused error
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		patientID, err := s.ownPatient(ctx, c)
		if errors.Is(err, errNoProfile) {
			return s.record(ctx, c, auditevent.CodeSearchType, "", auditevent.OutcomeSuccess)
		}
		if err != nil {
			return err
		}
		cur, err := s.activeHistory(ctx, patientID, organizationID)
		if err != nil {
			return err
		}
		visits, err := s.visits.Shareable(ctx, c, organizationID)
		if errors.Is(err, ErrClinicNotFound) {
			refused = ErrClinicNotFound
			return s.record(ctx, c, auditevent.CodeSearchType, "", auditevent.OutcomeMinorFailure)
		}
		if err != nil {
			return err
		}
		listed := sharedIDs(cur)
		for _, v := range visits {
			v.Selected = slices.Contains(listed, v.EncounterID)
			out = append(out, v)
		}
		return s.record(ctx, c, auditevent.CodeSearchType, cur.ID, auditevent.OutcomeSuccess)
	})
	switch {
	case err != nil:
		return nil, fmt.Errorf("consent: candidates: %w", err)
	case refused != nil:
		return nil, refused
	}
	return out, nil
}

// SetSharing makes the calling patient's history consent with the clinic organizationID list exactly the visits of
// in, each a candidate (Candidates): it grants the consent when none is active (version 0), or changes it at
// version, removing the visits not sent and adding the new ones; no visit revokes it. Every change takes effect when
// its transaction commits; a removal and a revocation are final. ErrCareConsentRequired without an active care
// consent with the clinic (or a profile), ErrVersionConflict when the sharing is not at version (none for 0),
// ErrInvalidVisit for a visit that is not a candidate, ErrForbidden to staff. Audited as a create, or an update of
// the history consent, also when refused.
func (s *Service) SetSharing(ctx context.Context, c Caller, organizationID string, in SetSharing, version int32) (Sharing, error) {
	if !c.patient() {
		return Sharing{}, s.refuse(ctx, c, auditevent.CodeUpdate, ErrForbidden)
	}
	ids := unique(in.EncounterIDs)
	var out Sharing
	var refused error
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		var interaction string
		var err error
		out, interaction, refused, err = s.setSharing(ctx, c, organizationID, ids, version)
		switch {
		case err != nil:
			return err
		case refused != nil:
			return s.record(ctx, c, auditevent.CodeUpdate, "", auditevent.OutcomeMinorFailure)
		}
		err = s.record(ctx, c, interaction, out.ConsentID, auditevent.OutcomeSuccess)
		if len(out.Shared) == 0 { // revoked: the event names the consent, the sharing none
			out.ConsentID = ""
		}
		return err
	})
	switch {
	case errors.Is(err, ErrInvalidVisit), errors.Is(err, ErrClinicNotFound):
		// The insert aborted the transaction (row-level security refused a visit): audited in a transaction of its own.
		return Sharing{}, s.refuse(ctx, c, auditevent.CodeUpdate, refusal(err))
	case err != nil:
		return Sharing{}, fmt.Errorf("consent: set sharing: %w", err)
	case refused != nil:
		return Sharing{}, refused
	}
	return out, nil
}

// setSharing does SetSharing's work in the transaction in ctx and returns the sharing with the interaction to audit
// (a create or an update; of the revoked consent after a revocation), or the refusal.
func (s *Service) setSharing(ctx context.Context, c Caller, organizationID string, ids []string, version int32) (out Sharing, interaction string, refused, err error) {
	patientID, err := s.ownPatient(ctx, c)
	if errors.Is(err, errNoProfile) {
		return Sharing{}, "", ErrCareConsentRequired, nil
	}
	if err != nil {
		return Sharing{}, "", nil, err
	}
	if st, err := s.repo.State(ctx, patientID, organizationID); err != nil || !st.Active {
		return Sharing{}, "", ErrCareConsentRequired, err
	}
	cur, err := s.activeHistory(ctx, patientID, organizationID)
	if err != nil || cur.VersionID != version {
		return Sharing{}, "", ErrVersionConflict, err
	}
	if len(ids) == 0 {
		return s.stopSharing(ctx, patientID, organizationID, cur)
	}
	if refused, err := s.shareable(ctx, c, organizationID, ids); refused != nil || err != nil {
		return Sharing{}, "", refused, err
	}
	var x Consent
	if cur.ID == "" {
		x, err = s.repo.GrantHistory(ctx, patientID, organizationID, ids, s.clock.Now())
		interaction = auditevent.CodeCreate
	} else {
		x, err = s.change(ctx, cur, ids)
		interaction = auditevent.CodeUpdate
	}
	if errors.Is(err, ErrActive) || errors.Is(err, ErrVersionConflict) { // a concurrent change committed first
		return Sharing{}, "", ErrVersionConflict, nil
	}
	return toSharing(organizationID, x), interaction, nil, err
}

// change sets the visits the active history consent cur lists to ids, when they differ, and returns it.
func (s *Service) change(ctx context.Context, cur Consent, ids []string) (Consent, error) {
	listed := sharedIDs(cur)
	var add, remove []string
	for _, id := range ids {
		if !slices.Contains(listed, id) {
			add = append(add, id)
		}
	}
	for _, id := range listed {
		if !slices.Contains(ids, id) {
			remove = append(remove, id)
		}
	}
	if len(add) == 0 && len(remove) == 0 {
		return cur, nil
	}
	return s.repo.ChangeHistory(ctx, cur.ID, cur.VersionID, add, remove, s.clock.Now())
}

// shareable refuses, with ErrInvalidVisit, an id that is not a visit the patient c may share with the clinic.
func (s *Service) shareable(ctx context.Context, c Caller, organizationID string, ids []string) (refused, err error) {
	visits, err := s.visits.Shareable(ctx, c, organizationID)
	if err != nil {
		return nil, err
	}
	for _, id := range ids {
		if !slices.ContainsFunc(visits, func(v Candidate) bool { return v.EncounterID == id }) {
			return ErrInvalidVisit, nil
		}
	}
	return nil, nil
}

// stopSharing revokes the patient's history consent cur with the clinic, when there is one, and returns the empty
// sharing.
func (s *Service) stopSharing(ctx context.Context, patientID, organizationID string, cur Consent) (Sharing, string, error, error) {
	out := Sharing{OrganizationID: organizationID, Shared: []string{}}
	if cur.ID == "" {
		return out, auditevent.CodeUpdate, nil, nil
	}
	_, err := s.repo.RevokeHistory(ctx, patientID, organizationID, s.clock.Now())
	switch {
	case errors.Is(err, ErrInactive): // a concurrent revocation committed first
		return Sharing{}, "", ErrVersionConflict, nil
	case err != nil:
		return Sharing{}, "", nil, err
	}
	out.ConsentID = cur.ID
	return out, auditevent.CodeUpdate, nil, nil
}

// StopSharing revokes the calling patient's history consent with the clinic organizationID, and does nothing when
// there is none; it takes effect when its transaction commits. ErrForbidden to staff. Audited as an update.
func (s *Service) StopSharing(ctx context.Context, c Caller, organizationID string) error {
	if !c.patient() {
		return s.refuse(ctx, c, auditevent.CodeUpdate, ErrForbidden)
	}
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		patientID, err := s.ownPatient(ctx, c)
		if errors.Is(err, errNoProfile) {
			return s.record(ctx, c, auditevent.CodeUpdate, "", auditevent.OutcomeSuccess)
		}
		if err != nil {
			return err
		}
		x, err := s.repo.RevokeHistory(ctx, patientID, organizationID, s.clock.Now())
		if err != nil && !errors.Is(err, ErrInactive) {
			return err
		}
		return s.record(ctx, c, auditevent.CodeUpdate, x.ID, auditevent.OutcomeSuccess)
	})
	if err != nil {
		return fmt.Errorf("consent: stop sharing: %w", err)
	}
	return nil
}

// MySharing returns what the calling patient shares with each clinic now: their active history consents, the latest
// granted first, and an empty list without a profile. Each is audited; a list of none writes one event without an
// entity. ErrForbidden to staff.
func (s *Service) MySharing(ctx context.Context, c Caller) ([]Sharing, error) {
	if !c.patient() {
		return nil, s.refuse(ctx, c, auditevent.CodeSearchType, ErrForbidden)
	}
	out := []Sharing{}
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		patientID, err := s.ownPatient(ctx, c)
		if err != nil && !errors.Is(err, errNoProfile) {
			return err
		}
		if err == nil {
			all, err := s.repo.ByPatient(ctx, patientID)
			if err != nil {
				return err
			}
			for _, x := range all {
				if x.PatientID == patientID && x.Kind == KindHistory && x.Status == StatusActive { // not on row-level security alone
					out = append(out, toSharing(x.OrganizationID, x))
				}
			}
		}
		if len(out) == 0 {
			return s.record(ctx, c, auditevent.CodeSearchType, "", auditevent.OutcomeSuccess)
		}
		for _, x := range out {
			if err := s.record(ctx, c, auditevent.CodeSearchType, x.ConsentID, auditevent.OutcomeSuccess); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("consent: my sharing: %w", err)
	}
	return out, nil
}

// refusal is the domain error err wraps, of ErrInvalidVisit and ErrClinicNotFound.
func refusal(err error) error {
	if errors.Is(err, ErrClinicNotFound) {
		return ErrClinicNotFound
	}
	return ErrInvalidVisit
}

// activeHistory is the patient's active history consent with the clinic, or the zero Consent for none.
func (s *Service) activeHistory(ctx context.Context, patientID, organizationID string) (Consent, error) {
	cur, err := s.repo.ActiveHistory(ctx, patientID, organizationID)
	if errors.Is(err, ErrNotFound) || (err == nil && cur.PatientID != patientID) { // not on row-level security alone
		return Consent{}, nil
	}
	return cur, err
}

func toSharing(organizationID string, x Consent) Sharing {
	return Sharing{OrganizationID: organizationID, ConsentID: x.ID, VersionID: strconv.Itoa(int(x.VersionID)), Shared: sharedIDs(x)}
}

// sharedIDs are the encounters x lists, never nil.
func sharedIDs(x Consent) []string {
	ids := make([]string, 0, len(x.Shared))
	for _, v := range x.Shared {
		ids = append(ids, v.EncounterID)
	}
	return ids
}

// unique is ids without repeats, in their first order.
func unique(ids []string) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if !slices.Contains(out, id) {
			out = append(out, id)
		}
	}
	return out
}
