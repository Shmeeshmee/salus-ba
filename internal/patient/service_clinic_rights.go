package patient

import (
	"context"
	"errors"
	"fmt"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/consent"
)

// The rights of a patient the clinic registered (docs/clinic-cms R2, V2), exercised through the clinic: its staff
// correct the profile, record the withdrawal of the consent given in person, and its admin hands over the record. Each
// works only on a profile the caller's own clinic registered: another clinic's, or a patient's own, is not found.

// ClinicRecord is what the clinic holds of a patient it registered, as its admin hands it over: the profile and the
// consents the patient gave the clinic.
type ClinicRecord struct {
	Patient  Patient
	Consents []consent.Consent
}

// Correct replaces the names, birth date and phone of the profile in.ID that the clinic of its admin or receptionist c
// registered, at version in.VersionID, and returns it with the next version; audited as an update. ErrForbidden to
// anyone else, ErrNotFound for a profile the clinic did not register, ErrConflict for a stale version (each audited),
// ErrInvalid… for a field that breaks SPEC.md "Profile" (before any access).
func (s *Service) Correct(ctx context.Context, c Caller, in Patient) (Patient, error) {
	if !c.desk() {
		return Patient{}, s.refuseRights(ctx, c, auditevent.CodeUpdate, c.purpose(), ErrForbidden)
	}
	in.Phone = phoneOf(in.Phone) // as the import and the desk's registration read it
	in, err := s.valid(in)
	if err != nil {
		return Patient{}, err
	}
	var out Patient
	var refused error
	err = s.tx.WithTx(ctx, func(ctx context.Context) error {
		_, err := s.registered(ctx, c, in.ID)
		if err == nil {
			out, err = s.repo.Correct(ctx, in, c.OrganizationID, s.clock.Now())
		}
		if err == nil {
			out.ConsentActive, err = s.consentActive(ctx, out.ID, c.OrganizationID)
		}
		if errors.Is(err, ErrNotFound) || errors.Is(err, ErrConflict) {
			refused = err
			return s.record(ctx, c, auditevent.CodeUpdate, "", auditevent.OutcomeMinorFailure)
		}
		if err != nil {
			return err
		}
		return s.record(ctx, c, auditevent.CodeUpdate, out.ID, auditevent.OutcomeSuccess)
	})
	switch {
	case err != nil:
		return Patient{}, fmt.Errorf("patient: correct: %w", err)
	case refused != nil:
		return Patient{}, refused
	}
	return out, nil
}

// WithdrawConsent records, as the clinic's admin or receptionist c, that the patient id the clinic registered withdrew
// the care consent they gave it in person (consent.Service.WithdrawInPerson, which audits it and does what a patient's
// own revocation does), in one transaction with the read of the profile, audited. ErrForbidden to anyone else,
// ErrNotFound for a profile the clinic did not register, consent.ErrInactive when no such consent is active.
func (s *Service) WithdrawConsent(ctx context.Context, c Caller, id string) (consent.Consent, error) {
	if !c.desk() {
		return consent.Consent{}, s.refuseRights(ctx, c, auditevent.CodeRead, c.purpose(), ErrForbidden)
	}
	var out consent.Consent
	var refused error
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		_, err := s.registered(ctx, c, id)
		if errors.Is(err, ErrNotFound) {
			refused = err
			return s.record(ctx, c, auditevent.CodeRead, "", auditevent.OutcomeMinorFailure)
		}
		if err != nil {
			return err
		}
		if err := s.record(ctx, c, auditevent.CodeRead, id, auditevent.OutcomeSuccess); err != nil {
			return err
		}
		out, err = s.consents.WithdrawInPerson(ctx, consent.Caller{AccountID: c.AccountID, OrganizationID: c.OrganizationID}, id)
		if errors.Is(err, consent.ErrInactive) { // refused and audited by the consent domain: keep its event and the read's
			refused, err = err, nil
		}
		return err
	})
	switch {
	case err != nil:
		return consent.Consent{}, fmt.Errorf("patient: withdraw consent: %w", err)
	case refused != nil:
		return consent.Consent{}, refused
	}
	return out, nil
}

// The purposes a record of a registered patient is read for (ClinicRecord): the hand-over at the patient's request,
// and the clinic's data export (healthcare operations).
const (
	PurposeHandOver = auditevent.CodePATRQT
	PurposeExport   = auditevent.CodeHOPERAT
)

// ClinicRecord returns, to the clinic's admin c, the record of the patient id the clinic registered: the profile and
// the clinic's consents of the patient, in one transaction, for the purpose (PurposeHandOver or PurposeExport; any
// other is read as the hand-over). The profile's read is audited with the purpose, each consent's by the consent
// domain. ErrForbidden to anyone else, ErrNotFound for a profile the clinic did not register (each audited).
func (s *Service) ClinicRecord(ctx context.Context, c Caller, id, purpose string) (ClinicRecord, error) {
	if purpose != PurposeExport {
		purpose = PurposeHandOver
	}
	if c.patient() || !c.Admin {
		return ClinicRecord{}, s.refuseRights(ctx, c, auditevent.CodeRead, purpose, ErrForbidden)
	}
	var out ClinicRecord
	var refused error
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		var err error
		out.Patient, err = s.registered(ctx, c, id)
		if errors.Is(err, ErrNotFound) {
			refused = err
			return s.audit.Record(ctx, s.event(c, auditevent.CodeRead, purpose, "", auditevent.OutcomeMinorFailure))
		}
		if err != nil {
			return err
		}
		if out.Consents, err = s.consents.OfClinicPatient(ctx, consent.Caller{AccountID: c.AccountID, OrganizationID: c.OrganizationID}, id); err != nil {
			return err
		}
		return s.audit.Record(ctx, s.event(c, auditevent.CodeRead, purpose, id, auditevent.OutcomeSuccess))
	})
	switch {
	case err != nil:
		return ClinicRecord{}, fmt.Errorf("patient: clinic record: %w", err)
	case refused != nil:
		return ClinicRecord{}, refused
	}
	return out, nil
}

// Registered returns, to the clinic's admin c, the ids of the patients the clinic registered, for its data export,
// audited as one search with the export's purpose. ErrForbidden, audited, to anyone else.
func (s *Service) Registered(ctx context.Context, c Caller) ([]string, error) {
	if c.patient() || !c.Admin {
		return nil, s.refuseRights(ctx, c, auditevent.CodeSearchType, PurposeExport, ErrForbidden)
	}
	var out []string
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.repo.SetClinic(ctx, c.OrganizationID); err != nil {
			return err
		}
		var err error
		if out, err = s.repo.RegisteredBy(ctx, c.OrganizationID); err != nil {
			return err
		}
		return s.audit.Record(ctx, s.event(c, auditevent.CodeSearchType, PurposeExport, "", auditevent.OutcomeSuccess))
	})
	if err != nil {
		return nil, fmt.Errorf("patient: registered: %w", err)
	}
	return out, nil
}

// RestrictedRecords returns the ids of the patients whose record c's clinic restricts (S7), for the clinic's data
// export's manifest: the clinic's admin only, in its scope, audited as one search for the export. ErrForbidden, audited,
// to anyone else.
func (s *Service) RestrictedRecords(ctx context.Context, c Caller) ([]string, error) {
	if c.patient() || !c.Admin {
		return nil, s.refuseRights(ctx, c, auditevent.CodeSearchType, PurposeExport, ErrForbidden)
	}
	var out []string
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.repo.SetClinic(ctx, c.OrganizationID); err != nil {
			return err
		}
		var err error
		if out, err = s.repo.RestrictedAt(ctx, c.OrganizationID); err != nil {
			return err
		}
		return s.audit.Record(ctx, s.event(c, auditevent.CodeSearchType, PurposeExport, "", auditevent.OutcomeSuccess))
	})
	if err != nil {
		return nil, fmt.Errorf("patient: restricted records: %w", err)
	}
	return out, nil
}

// registered scopes the transaction in ctx to c's clinic and returns the profile id, or ErrNotFound unless the profile id is one that clinic
// registered (checked here, not on row-level security alone: the clinic's scope also shows patients with its consent).
func (s *Service) registered(ctx context.Context, c Caller, id string) (Patient, error) {
	if err := s.repo.SetClinic(ctx, c.OrganizationID); err != nil {
		return Patient{}, err
	}
	p, err := s.repo.ByID(ctx, id)
	if errors.Is(err, ErrNotFound) || (err == nil && p.RegisteredBy != c.OrganizationID) {
		return Patient{}, ErrNotFound
	}
	return p, err
}

// refuseRights audits a refused interaction by c with the purpose, disclosing no id, in a transaction of its own, and
// returns refusal.
func (s *Service) refuseRights(ctx context.Context, c Caller, interaction, purpose string, refusal error) error {
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		return s.audit.Record(ctx, s.event(c, interaction, purpose, "", auditevent.OutcomeMinorFailure))
	})
	if err != nil {
		return err
	}
	return refusal
}
