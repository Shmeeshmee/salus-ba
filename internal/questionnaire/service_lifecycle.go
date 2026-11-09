package questionnaire

import (
	"context"
	"errors"
)

// Publish makes the draft id, at version (the version the caller last read), its series' active version,
// and retires the version that was active, in one transaction. Once published it never changes. A published
// version gives ErrNotDraft, another version ErrConflict, and a draft with a section without items (which a
// draft may keep, FHIR que-1 refuses) an *InvalidError naming those sections, and a draft with an incomplete
// translation a *TranslationIncompleteError for the first in the enum's order (docs/i18n/SPEC.md §3).
func (s *Service) Publish(ctx context.Context, c Caller, id string, version int32) (Questionnaire, error) {
	var out Questionnaire
	err := s.do(ctx, c, opPublish, func(ctx context.Context) ([]string, error, error) {
		// Locked first, so two publishes of one draft run one after the other and the second sees it active.
		cur, failed, err := s.version(ctx, c, id, true)
		switch {
		case err != nil || failed != nil:
			return nil, failed, err
		case cur.Status != StatusDraft:
			return []string{cur.ID}, ErrNotDraft, nil
		case cur.VersionID != version:
			return []string{cur.ID}, ErrConflict, nil
		}
		if invalid := publishable(cur.Items); invalid != nil {
			return []string{cur.ID}, invalid, nil
		}
		ts, err := s.Repo.Translations(ctx, cur.ID)
		if err != nil {
			return nil, nil, err
		}
		for _, t := range ts {
			if whole, linkIDs := incomplete(cur, t); whole || len(linkIDs) > 0 {
				return []string{cur.ID}, &TranslationIncompleteError{Language: t.Language, LinkIDs: linkIDs}, nil
			}
		}
		at := s.Clock.Now()
		if err := s.Repo.RetireActive(ctx, cur.SeriesID, at); err != nil {
			return nil, nil, err
		}
		out, err = s.Repo.Publish(ctx, cur.ID, at)
		out.Translations = ts
		return []string{cur.ID}, nil, err
	})
	if err != nil {
		return Questionnaire{}, err
	}
	return out, nil
}

// Retire retires the active version id: it is no longer requested, and requests already made keep it. A
// version that is not active gives ErrNotActive.
func (s *Service) Retire(ctx context.Context, c Caller, id string) (Questionnaire, error) {
	var out Questionnaire
	err := s.do(ctx, c, opRetire, func(ctx context.Context) ([]string, error, error) {
		cur, failed, err := s.version(ctx, c, id, true)
		switch {
		case err != nil || failed != nil:
			return nil, failed, err
		case cur.Status != StatusActive:
			return []string{cur.ID}, ErrNotActive, nil
		}
		if out, err = s.Repo.Retire(ctx, cur.ID, s.Clock.Now()); err != nil {
			return nil, nil, err
		}
		return []string{cur.ID}, nil, s.translate(ctx, &out, false)
	})
	if err != nil {
		return Questionnaire{}, err
	}
	return out, nil
}

// Revise copies the active version id into the next version of its series, a draft with the same content
// and linkIds and its translations ("Edit as new version"); the active version stays active until the draft is published. A
// version that is not active gives ErrNotActive, a series that has a draft ErrDraftExists. The event names
// the new draft, or the active version when refused.
func (s *Service) Revise(ctx context.Context, c Caller, id string) (Questionnaire, error) {
	var out Questionnaire
	err := s.do(ctx, c, opRevise, func(ctx context.Context) ([]string, error, error) {
		cur, failed, err := s.version(ctx, c, id, false)
		switch {
		case err != nil || failed != nil:
			return nil, failed, err
		case cur.Status != StatusActive:
			return []string{cur.ID}, ErrNotActive, nil
		}
		out, err = s.Repo.CreateVersion(ctx, cur, s.Clock.Now())
		switch {
		case errors.Is(err, ErrDraftExists):
			return []string{cur.ID}, ErrDraftExists, nil
		case err != nil:
			return nil, nil, err
		}
		return []string{out.ID}, nil, s.translate(ctx, &out, false)
	})
	if err != nil {
		return Questionnaire{}, err
	}
	return out, nil
}
