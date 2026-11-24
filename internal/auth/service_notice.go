package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/Shmeeshmee/salus/internal/auditevent"
)

// AccountEmail returns the e-mail of the account id for a message the job agent (its kind) sends, such as an
// appointment's confirmation to its patient, and the language to write it in: the account's chosen language,
// else the one its client last matched, else "" for the sender's fallback (docs/i18n/SPEC.md §3 "The
// account's language fields"). The read is audited with the job as agent, also when the account is unknown
// (ErrNotFound). Errors carry no address.
func (s *Service) AccountEmail(ctx context.Context, agent, id string) (email, language string, err error) {
	a, err := s.accountFor(ctx, agent, id)
	if err != nil {
		return "", "", fmt.Errorf("auth: account e-mail: %w", err)
	}
	return a.Email, languageOf(a), nil
}

// AccountLanguage returns only the language of the account id, for the transcription agent (SAL-4), which
// recognises the patient's speech in it: the chosen language, else the one its client last matched, else "".
// The read is audited with agent as the agent, also when the account is unknown (ErrNotFound).
func (s *Service) AccountLanguage(ctx context.Context, agent, id string) (string, error) {
	a, err := s.accountFor(ctx, agent, id)
	if err != nil {
		return "", fmt.Errorf("auth: account language: %w", err)
	}
	return languageOf(a), nil
}

// accountFor reads the account id for the job agent, audited with the agent, also when it is unknown
// (ErrNotFound).
func (s *Service) accountFor(ctx context.Context, agent, id string) (Account, error) {
	var a Account
	var missing bool
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		var err error
		a, err = s.repo.ByID(ctx, id)
		if missing = errors.Is(err, ErrNotFound); err != nil && !missing {
			return err
		}
		e := event(restCoding, interaction(auditevent.CodeRead), auditevent.ActionRead, a, auditevent.OutcomeSuccess)
		if missing {
			e = event(restCoding, interaction(auditevent.CodeRead), auditevent.ActionRead, Account{}, auditevent.OutcomeMinorFailure)
		}
		e.AgentAccountID, e.AgentName = "", agent
		return s.audit.Record(ctx, e)
	})
	switch {
	case err != nil:
		return Account{}, err
	case missing:
		return Account{}, ErrNotFound
	}
	return a, nil
}

// languageOf is the account's chosen language, else the one its client last matched.
func languageOf(a Account) string {
	if a.Language != "" {
		return a.Language
	}
	return a.DetectedLanguage
}

// ClinicAdmins returns the e-mail and language of each admin of the clinic org, for a message the job agent sends to
// them (a data request's deadline warning, docs/clinic-cms/WAVE5.md D5). The read is audited as one, with the job as
// agent and the number of accounts read. Errors carry no address.
func (s *Service) ClinicAdmins(ctx context.Context, agent, org string) ([]Contact, error) {
	var out []Contact
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		var err error
		if out, err = s.repo.ClinicAdmins(ctx, org); err != nil {
			return err
		}
		e := event(restCoding, interaction(auditevent.CodeSearchType), auditevent.ActionRead, Account{OrganizationID: org}, auditevent.OutcomeSuccess)
		e.AgentName, e.EntityName, e.EntityCount = agent, "account", int64(len(out))
		return s.audit.Record(ctx, e)
	})
	if err != nil {
		return nil, fmt.Errorf("auth: clinic admins: %w", err)
	}
	return out, nil
}

// PractitionerAccount returns the account of the practitioner practitionerID of the clinic org, with its e-mail and
// language, for a message the job agent sends to them (the chat's notifications, SAL-29). The read is audited with
// the job as agent, also when there is no such account in the clinic (ErrNotFound). Errors carry no address.
func (s *Service) PractitionerAccount(ctx context.Context, agent, org, practitionerID string) (id, email, language string, err error) {
	var a Account
	var missing bool
	err = s.tx.WithTx(ctx, func(ctx context.Context) error {
		var err error
		a, err = s.repo.ByPractitioner(ctx, practitionerID)
		if missing = errors.Is(err, ErrNotFound) || (err == nil && a.OrganizationID != org); err != nil && !missing {
			return err
		}
		e := event(restCoding, interaction(auditevent.CodeRead), auditevent.ActionRead, a, auditevent.OutcomeSuccess)
		if missing {
			e = event(restCoding, interaction(auditevent.CodeRead), auditevent.ActionRead, Account{OrganizationID: org}, auditevent.OutcomeMinorFailure)
		}
		e.AgentAccountID, e.AgentName = "", agent
		return s.audit.Record(ctx, e)
	})
	switch {
	case err != nil:
		return "", "", "", fmt.Errorf("auth: practitioner account: %w", err)
	case missing:
		return "", "", "", ErrNotFound
	}
	return a.ID, a.Email, languageOf(a), nil
}
