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
