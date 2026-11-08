package auth

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/Shmeeshmee/salus/internal/auditevent"
)

// languages are the tags of the enum language (docs/i18n/SPEC.md §3 "Tags"); the first staffLanguages of
// them are the staff languages, the ones the web app offers.
var languages = []string{"bs", "hr", "sr-Latn", "sr-Cyrl", "en", "de", "ar", "tr", "it", "fr"}

const staffLanguages = 5

// checkLanguages returns ErrInvalidLanguage unless every tag is empty (none sent) or one of languages.
func checkLanguages(tags ...string) error {
	for _, tag := range tags {
		if tag != "" && !slices.Contains(languages, tag) {
			return ErrInvalidLanguage
		}
	}
	return nil
}

// takes reports whether an account of role may have the language tag: a patient any, staff only a staff
// language. Every role takes "", no language.
func takes(role Role, tag string) bool {
	return tag == "" || role == RolePatient || slices.Contains(languages[:staffLanguages], tag)
}

// forRole is tag when role takes it, else "".
func forRole(role Role, tag string) string {
	if takes(role, tag) {
		return tag
	}
	return ""
}

// SetLanguage saves the language the account's user chose (PATCH /auth/me), or clears it for "", and
// returns the account. A staff account takes only a staff language (ErrLanguageUnavailable). The change is
// audited, also when it is refused.
func (s *Service) SetLanguage(ctx context.Context, id, language string) (Account, error) {
	if err := checkLanguages(language); err != nil {
		return Account{}, err
	}
	var a Account
	var refused error
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		var err error
		a, err = s.repo.ByID(ctx, id)
		switch {
		case errors.Is(err, ErrNotFound):
			a, refused = Account{ID: id}, ErrNotFound
		case err != nil:
			return err
		case !takes(a.Role, language):
			refused = ErrLanguageUnavailable
		default:
			if err := s.repo.SetLanguage(ctx, id, language); err != nil {
				return err
			}
			a.Language = language
		}
		outcome := auditevent.OutcomeSuccess
		if refused != nil {
			outcome = auditevent.OutcomeMinorFailure
		}
		return s.audit.Record(ctx, event(restCoding, interaction(auditevent.CodeUpdate), auditevent.ActionUpdate, a, outcome))
	})
	switch {
	case err != nil:
		return Account{}, fmt.Errorf("auth: set language: %w", err)
	case refused != nil:
		return Account{}, refused
	}
	a.PasswordHash = ""
	return a, nil
}

// saveLanguages stores, inside the caller's transaction, the language chosen and the language detected that
// a sign-in sent for a, each when it is set, differs from the stored one and a's role takes it, and audits
// the change when there is one. A language a staff account does not take is dropped: it never stops a
// sign-in.
func (s *Service) saveLanguages(ctx context.Context, a Account, language, detected string) error {
	changed := false
	if language = forRole(a.Role, language); language != "" && language != a.Language {
		if err := s.repo.SetLanguage(ctx, a.ID, language); err != nil {
			return err
		}
		changed = true
	}
	if detected = forRole(a.Role, detected); detected != "" && detected != a.DetectedLanguage {
		if err := s.repo.SetDetectedLanguage(ctx, a.ID, detected); err != nil {
			return err
		}
		changed = true
	}
	if !changed {
		return nil
	}
	return s.audit.Record(ctx, event(restCoding, interaction(auditevent.CodeUpdate), auditevent.ActionUpdate, a, auditevent.OutcomeSuccess))
}
