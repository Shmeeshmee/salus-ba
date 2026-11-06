package questionnaire

import (
	"context"
	"errors"

	"github.com/Shmeeshmee/salus/internal/practitionerrole"
)

// Default returns the default questionnaire of the role roleID, or one without a series at version 0 while
// it has none. The role's practitioner and the clinic's admin may; anyone else of the clinic gets
// ErrNotFound, as for a role of another clinic.
func (s *Service) Default(ctx context.Context, c Caller, roleID string) (Default, error) {
	var out Default
	err := s.do(ctx, c, opDefault, func(ctx context.Context) ([]string, error, error) {
		if failed, err := s.ownRole(ctx, c, roleID); err != nil || failed != nil {
			return nil, failed, err
		}
		var err error
		out, err = s.Repo.Default(ctx, roleID)
		if errors.Is(err, ErrNotFound) || (err == nil && out.OrganizationID != c.OrganizationID) {
			out, err = Default{PractitionerRoleID: roleID, OrganizationID: c.OrganizationID}, nil
		}
		return []string{roleID}, nil, err
	})
	if err != nil {
		return Default{}, err
	}
	return out, nil
}

// SetDefault makes the questionnaire whose version questionnaireID is (any version of its series) the
// default of the role roleID: from now on, its active version is requested with every booking with the
// role. Who may is as for Default. A questionnaire not of c's clinic gives ErrNotFound, one without an
// active version ErrNotActive.
func (s *Service) SetDefault(ctx context.Context, c Caller, roleID, questionnaireID string) (Default, error) {
	var out Default
	err := s.do(ctx, c, opSetDefault, func(ctx context.Context) ([]string, error, error) {
		if failed, err := s.ownRole(ctx, c, roleID); err != nil || failed != nil {
			return nil, failed, err
		}
		q, failed, err := s.version(ctx, c, questionnaireID, false)
		if err != nil || failed != nil {
			return []string{roleID}, failed, err
		}
		active, err := s.Repo.Search(ctx, Filter{SeriesID: q.SeriesID, Status: StatusActive})
		if err != nil {
			return nil, nil, err
		}
		if len(active) == 0 {
			return []string{roleID}, ErrNotActive, nil
		}
		out, err = s.Repo.SetDefault(ctx, Default{PractitionerRoleID: roleID, OrganizationID: c.OrganizationID, SeriesID: q.SeriesID}, s.Clock.Now())
		return []string{roleID}, nil, err
	})
	if err != nil {
		return Default{}, err
	}
	return out, nil
}

// ClearDefault removes the default questionnaire of the role roleID, if it has one. Who may is as for
// Default.
func (s *Service) ClearDefault(ctx context.Context, c Caller, roleID string) error {
	return s.do(ctx, c, opClearDefault, func(ctx context.Context) ([]string, error, error) {
		if failed, err := s.ownRole(ctx, c, roleID); err != nil || failed != nil {
			return nil, failed, err
		}
		return []string{roleID}, nil, s.Repo.ClearDefault(ctx, roleID)
	})
}

// ownRole returns ErrNotFound as failed unless the role roleID is of c's clinic and c is its practitioner
// or the clinic's admin.
func (s *Service) ownRole(ctx context.Context, c Caller, roleID string) (failed, err error) {
	role, err := s.Roles.Lookup(ctx, c.OrganizationID, roleID)
	switch {
	case errors.Is(err, practitionerrole.ErrNotFound):
		return ErrNotFound, nil
	case err != nil:
		return nil, err
	case role.OrganizationID != c.OrganizationID:
		return ErrNotFound, nil
	case !c.Admin && (c.PractitionerID == "" || role.PractitionerID != c.PractitionerID):
		return ErrNotFound, nil
	}
	return nil, nil
}
