package organization

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/Shmeeshmee/salus/internal/auditevent"
)

// maxProfileDescription bounds the profile's description and each translation of it (api/openapi.yaml ClinicProfile).
const maxProfileDescription = 2000 // characters

// languages are the tags of the language enum: the languages a clinic's staff may speak and its description be
// translated into (docs/i18n/SPEC.md §3 "Tags").
var languages = []string{"bs", "hr", "sr-Latn", "sr-Cyrl", "en", "de", "ar", "tr", "it", "fr"}

// Profile returns the profile of the listed clinic id, as the public directory shows it, for anyone: a clinic that is
// not listed, or whose membership is stopped (the directory's policy, package I), gives ErrNotFound, as the directory
// does. Every translation comes with it; the app picks the patient's language. Not audited: not personal data.
func (s *Service) Profile(ctx context.Context, id string) (Profile, error) {
	var p Profile
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.repo.SetDirectory(ctx); err != nil {
			return err
		}
		o, err := s.repo.ByID(ctx, id)
		if err != nil {
			return err
		}
		if !o.Listed { // not on row-level security alone
			return ErrNotFound
		}
		p, err = s.repo.ProfileOf(ctx, id)
		return err
	})
	switch {
	case errors.Is(err, ErrNotFound):
		return Profile{}, ErrNotFound
	case err != nil:
		return Profile{}, fmt.Errorf("organization: profile: %w", err)
	}
	return p, nil
}

// ClinicProfile returns the profile of the caller's clinic to its staff, at the organization's version; ErrNotFound
// to anyone else. Not audited: not personal data.
func (s *Service) ClinicProfile(ctx context.Context, c Caller) (Profile, error) {
	if c.OrganizationID == "" {
		return Profile{}, ErrNotFound
	}
	var p Profile
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.repo.SetOrganization(ctx, c.OrganizationID); err != nil {
			return err
		}
		var err error
		p, err = s.repo.ProfileOf(ctx, c.OrganizationID)
		return err
	})
	switch {
	case errors.Is(err, ErrNotFound):
		return Profile{}, ErrNotFound
	case err != nil:
		return Profile{}, fmt.Errorf("organization: clinic profile: %w", err)
	}
	return p, nil
}

// SaveProfile replaces the profile of the admin caller's clinic at version (the organization's If-Match) and returns
// it at the clinic's next version. An invalid profile gives ErrInvalidProfile before anything is accessed; a member who
// is not the admin ErrForbidden, anyone else ErrNotFound; a version that is not the current one ErrConflict. Audited
// as an update of the organization, also when refused.
func (s *Service) SaveProfile(ctx context.Context, c Caller, p Profile, version int32) (Profile, error) {
	p, err := validProfile(p)
	if err != nil {
		return Profile{}, err
	}
	if refused := refusal(c, c.OrganizationID); refused != nil {
		if err := s.record(ctx, c, "", auditevent.OutcomeMinorFailure); err != nil {
			return Profile{}, fmt.Errorf("organization: save profile: %w", err)
		}
		return Profile{}, refused
	}
	p.VersionID = version
	var saved Profile
	var failed error // a conflict, audited, and so committed
	err = s.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.repo.SetOrganization(ctx, c.OrganizationID); err != nil {
			return err
		}
		var err error
		saved, err = s.repo.PutProfile(ctx, c.OrganizationID, p, s.clock.Now())
		outcome := auditevent.OutcomeSuccess
		switch {
		case errors.Is(err, ErrConflict):
			failed, outcome = ErrConflict, auditevent.OutcomeMinorFailure
		case err != nil:
			return err
		}
		return s.record(ctx, c, c.OrganizationID, outcome)
	})
	switch {
	case err != nil:
		return Profile{}, fmt.Errorf("organization: save profile: %w", err)
	case failed != nil:
		return Profile{}, failed
	}
	return saved, nil
}

// ListListedSpeaking returns the listed clinics whose staff speak language, by name: the directory's language filter,
// for anyone. A language that is not of the enum gives ErrInvalidProfile. Not audited, as ListListed.
func (s *Service) ListListedSpeaking(ctx context.Context, language string) ([]Organization, error) {
	if !slices.Contains(languages, language) {
		return nil, ErrInvalidProfile
	}
	var out []Organization
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.repo.SetDirectory(ctx); err != nil {
			return err
		}
		all, err := s.repo.ListedSpeaking(ctx, language)
		for _, o := range all {
			if o.Listed { // not on row-level security alone
				out = append(out, o)
			}
		}
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("organization: list listed speaking: %w", err)
	}
	return out, nil
}

// validProfile is p with its description and translations trimmed, or ErrInvalidProfile.
func validProfile(p Profile) (Profile, error) {
	for i, l := range p.Languages {
		if !slices.Contains(languages, l) || slices.Contains(p.Languages[:i], l) {
			return Profile{}, ErrInvalidProfile
		}
	}
	p.Description = strings.TrimSpace(p.Description)
	if utf8.RuneCountInString(p.Description) > maxProfileDescription || (p.Description == "" && len(p.Translations) > 0) {
		return Profile{}, ErrInvalidProfile
	}
	out := make([]Translation, 0, len(p.Translations))
	for _, t := range p.Translations {
		t.Description = strings.TrimSpace(t.Description)
		if !slices.Contains(languages, t.Language) || t.Description == "" || utf8.RuneCountInString(t.Description) > maxProfileDescription ||
			slices.ContainsFunc(out, func(o Translation) bool { return o.Language == t.Language }) {
			return Profile{}, ErrInvalidProfile
		}
		out = append(out, t)
	}
	p.Translations = out
	return p, nil
}
