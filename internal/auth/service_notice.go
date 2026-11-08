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
	var a Account
	var missing bool
	err = s.tx.WithTx(ctx, func(ctx context.Context) error {
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
		return "", "", fmt.Errorf("auth: account e-mail: %w", err)
	case missing:
		return "", "", ErrNotFound
	}
	language = a.Language
	if language == "" {
		language = a.DetectedLanguage
	}
	return a.Email, language, nil
}
