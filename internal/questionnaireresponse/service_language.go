package questionnaireresponse

import (
	"context"
	"slices"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/questionnaire"
)

// Start sets the language the patient answers the response id in, as its patient (docs/i18n/SPEC.md §3 "The
// patient's language"), and returns it at its next version. The language is the version's original or one of
// the translations the questionnaire service gives the patient, which are its complete ones
// (ErrLanguageUnavailable otherwise); it may be chosen again until an answer or a file is saved
// (ErrLanguageLocked after). No If-Match; otherwise the same conditions, transaction, audit and refusals as
// SaveAnswers.
func (s *Service) Start(ctx context.Context, c appointment.Caller, id, language string) (Response, error) {
	return s.answer(ctx, c, id, nil, "start", func(ctx context.Context, r Response, q questionnaire.Questionnaire) (Response, error, error) {
		switch {
		case len(r.Answers) > 0 || len(r.Attachments) > 0:
			return Response{}, ErrLanguageLocked, nil
		case !offers(q, language):
			return Response{}, ErrLanguageUnavailable, nil
		}
		out, err := s.Repo.SetLanguage(ctx, r.ID, language, s.Clock.Now())
		return out, nil, err
	})
}

// offers reports whether q can be read in language: its original language, or a translation it carries.
func offers(q questionnaire.Questionnaire, language string) bool {
	return language != "" && (q.Language == language ||
		slices.ContainsFunc(q.Translations, func(t questionnaire.Translation) bool { return t.Language == language }))
}
