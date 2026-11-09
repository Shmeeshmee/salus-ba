package composition

import (
	"context"
	"errors"
	"slices"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
)

// Templates returns the caller's clinic's templates, by title, with their translations, to its staff
// (practitioners and admins); ErrForbidden to a patient. Not audited: a template holds no personal data.
func (s *Service) Templates(ctx context.Context, c appointment.Caller) ([]Template, error) {
	var out []Template
	var refused error
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		err := s.staff(ctx, c)
		if refused = refusal(err); refused != nil {
			return nil
		}
		if err != nil {
			return err
		}
		out, err = s.Repo.Templates(ctx)
		return err
	})
	if err := wrapErr("templates", err, refused); err != nil {
		return nil, err
	}
	if out == nil {
		out = []Template{}
	}
	return out, nil
}

// CreateTemplate stores a new template of the caller's clinic as version 1, for its staff. ErrInvalidTemplate
// (an *InvalidTemplateError) for content ValidateTemplate refuses, ErrForbidden to a patient. Audited, also
// when refused.
func (s *Service) CreateTemplate(ctx context.Context, c appointment.Caller, in TemplateContent) (Template, error) {
	var out Template
	var refused error
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		e := entry{c: c, interaction: auditevent.CodeCreate, template: true}
		err := s.staff(ctx, c)
		if refused = refusal(err); refused == nil && err == nil {
			refused = ValidateTemplate(in)
		}
		if refused != nil {
			e.outcome = auditevent.OutcomeMinorFailure
			return s.record(ctx, e)
		}
		if err != nil {
			return err
		}
		in.Translations = sorted(in.Translations)
		if out, err = s.Repo.InsertTemplate(ctx, c.OrganizationID, in, s.Clock.Now()); err != nil {
			return err
		}
		return s.record(ctx, e)
	})
	return out, wrapErr("create template", err, refused)
}

// UpdateTemplate replaces the content of the caller's clinic's template id, read at version, as its next
// version. CreateTemplate's errors, ErrTemplateNotFound for a template the clinic does not have, and
// ErrVersionMismatch for another version. Drafts keep the texts they were drafted with. Audited, also when
// refused.
func (s *Service) UpdateTemplate(ctx context.Context, c appointment.Caller, id string, in TemplateContent, version int32) (Template, error) {
	var out Template
	var refused error
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		e := entry{c: c, interaction: auditevent.CodeUpdate, template: true}
		err := s.staff(ctx, c)
		if refused = refusal(err); refused == nil && err == nil {
			refused, err = s.editable(ctx, id, in, version)
		}
		if refused != nil {
			e.outcome = auditevent.OutcomeMinorFailure
			return s.record(ctx, e)
		}
		if err != nil {
			return err
		}
		in.Translations = sorted(in.Translations)
		if out, err = s.Repo.UpdateTemplate(ctx, id, in, s.Clock.Now()); err != nil {
			return err
		}
		return s.record(ctx, e)
	})
	return out, wrapErr("update template", err, refused)
}

// editable is the refusal of an edit of the template id at version with in, or a failure.
func (s *Service) editable(ctx context.Context, id string, in TemplateContent, version int32) (refused, failed error) {
	cur, err := s.Repo.Template(ctx, id, true)
	switch {
	case errors.Is(err, ErrTemplateNotFound):
		return ErrTemplateNotFound, nil
	case err != nil:
		return nil, err
	case cur.VersionID != version:
		return ErrVersionMismatch, nil
	}
	return ValidateTemplate(in), nil
}

// sorted orders translations as the enum does.
func sorted(ts []Translation) []Translation {
	out := slices.Clone(ts)
	slices.SortFunc(out, func(a, b Translation) int {
		return slices.Index(languages, a.Language) - slices.Index(languages, b.Language)
	})
	return out
}
