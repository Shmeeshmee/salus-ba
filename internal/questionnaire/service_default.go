package questionnaire

import (
	"context"
	"errors"
	"fmt"

	"github.com/Shmeeshmee/salus/internal/auditevent"

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

// opJobRead is a job's read of a role's default or a version.
var opJobRead = op{"active default", auditevent.CodeRead, "", auditevent.ActionRead}

// ActiveDefault returns the active version of the default questionnaire of the role roleID of the clinic org,
// for the job agent (questionnaire_default_request), in the clinic's scope, in the transaction in ctx or one of
// its own. ErrNotFound when the role has no default there, ErrNotActive when its series has no active version.
// The read of the default, and of the version when there is one, is audited with the job as agent.
func (s *Service) ActiveDefault(ctx context.Context, agent, org, roleID string) (Questionnaire, error) {
	var out Questionnaire
	var failed error
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.Repo.SetOrganization(ctx, org); err != nil {
			return err
		}
		d, err := s.Repo.Default(ctx, roleID)
		switch {
		case errors.Is(err, ErrNotFound) || (err == nil && d.OrganizationID != org): // not on row-level security alone
			failed = ErrNotFound
		case err != nil:
			return err
		}
		if err := s.recordJob(ctx, agent, org, "PractitionerRole/"+roleID); err != nil || failed != nil {
			return err
		}
		active, err := s.Repo.Search(ctx, Filter{SeriesID: d.SeriesID, Status: StatusActive})
		if err != nil {
			return err
		}
		if len(active) == 0 || active[0].OrganizationID != org {
			failed = ErrNotActive
			return nil
		}
		out = active[0]
		return s.recordJob(ctx, agent, org, "Questionnaire/"+out.ID)
	})
	switch {
	case err != nil:
		return Questionnaire{}, fmt.Errorf("questionnaire: active default: %w", err)
	case failed != nil:
		return Questionnaire{}, failed
	}
	return out, nil
}

// recordJob audits the job agent's read of entity in the clinic org, for operations.
func (s *Service) recordJob(ctx context.Context, agent, org, entity string) error {
	e := entry(Caller{OrganizationID: org}, opJobRead, "", auditevent.OutcomeSuccess)
	e.AgentName, e.EntityReference = agent, entity
	return s.Audit.Record(ctx, e)
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
