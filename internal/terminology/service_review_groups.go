package terminology

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

// tableGroupName is the table a group name's review write is audited on (no FHIR resource, as term_translation).
const tableGroupName = "diagnosis_group_name"

// maxGroupName is the longest group name, in characters.
const maxGroupName = 100

// GroupReviewPorts are what GroupReviews is built on.
type GroupReviewPorts struct {
	Repo  GroupReviewRepository
	Audit Auditor
	Tx    Transactor
	Clock clock.Clock
}

// GroupReviews is the review of the diagnosis groups' names in the term review panel (docs/plans/analytics.md
// §3, owner 2026-11-12: the doctor partner reviews them): the same rules as a term's translation. Every staff
// member reads them; only a reviewer edits a name, which sets it back to unreviewed, or approves it or marks it
// as needing changes; every write is audited on diagnosis_group_name, in its transaction.
type GroupReviews struct {
	p GroupReviewPorts
}

// NewGroupReviews returns GroupReviews on p.
func NewGroupReviews(p GroupReviewPorts) *GroupReviews { return &GroupReviews{p: p} }

// GroupReviewEntry is a group as the panel lists it: its code, its parent, and its name in one language with the
// name's review and version.
type GroupReviewEntry struct {
	Code   string
	Parent string
	Name   GroupName
}

// List returns every group in its order with its name in language, an app language (ErrInvalidReview
// otherwise). Staff only.
func (s *GroupReviews) List(ctx context.Context, c Caller, language string) ([]GroupReviewEntry, error) {
	if !c.Staff {
		return nil, ErrForbidden
	}
	if !explanationLanguage(language) {
		return nil, ErrInvalidReview
	}
	groups, _, err := s.p.Repo.DiagnosisGroups(ctx)
	if err != nil {
		return nil, fmt.Errorf("terminology: group names: %w", err)
	}
	out := make([]GroupReviewEntry, len(groups))
	for i, g := range groups {
		out[i] = GroupReviewEntry{Code: g.Code, Parent: g.Parent}
		for _, n := range g.Names {
			if n.Language == language {
				out[i].Name = n
			}
		}
	}
	return out, nil
}

// EditGroupName replaces the group's name in language with text, trimmed, 1 to 100 characters, for a reviewer,
// at version. A change sets the name to unreviewed, keeping its last review; the same text changes nothing and
// is not audited.
func (s *GroupReviews) EditGroupName(ctx context.Context, c Caller, group, language, text string, version int32) (GroupName, error) {
	if !c.Staff || !c.Reviewer {
		return GroupName{}, s.refuse(ctx, c)
	}
	text = strings.TrimSpace(text)
	if n := utf8.RuneCountInString(text); n == 0 || n > maxGroupName || !utf8.ValidString(text) || strings.ContainsRune(text, 0) || !explanationLanguage(language) {
		return GroupName{}, ErrInvalidReview
	}
	return s.write(ctx, c, group, language, version, func(n GroupName, _ time.Time) (GroupName, bool) {
		if n.Text == text {
			return n, false
		}
		n.Text, n.Status = text, ReviewUnreviewed
		return n, true
	})
}

// ReviewGroupName approves the group's name in language or marks it as needing changes, for a reviewer, at
// version: the reviewer and now are the review's, the note is d's.
func (s *GroupReviews) ReviewGroupName(ctx context.Context, c Caller, group, language string, d Decision, version int32) (GroupName, error) {
	if !c.Staff || !c.Reviewer {
		return GroupName{}, s.refuse(ctx, c)
	}
	d, err := decision(d)
	if err != nil || !explanationLanguage(language) {
		return GroupName{}, cmpOr(err, ErrInvalidReview)
	}
	return s.write(ctx, c, group, language, version, func(n GroupName, now time.Time) (GroupName, bool) {
		n.Review = Review{Status: d.Status, ReviewedBy: c.AccountID, ReviewedAt: now, Note: d.Note}
		return n, true
	})
}

// write locks the group's name in language, checks version, applies change and, when it changed the name,
// stores it at the next version and audits the update, in one transaction.
func (s *GroupReviews) write(ctx context.Context, c Caller, group, language string, version int32,
	change func(n GroupName, now time.Time) (GroupName, bool),
) (GroupName, error) {
	var out GroupName
	err := s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		n, err := s.p.Repo.LockGroupName(ctx, group, language)
		if err != nil {
			return err
		}
		if n.VersionID != version {
			return ErrVersionConflict
		}
		now := s.p.Clock.Now()
		changed, ok := change(n, now)
		if out = changed; !ok {
			return nil
		}
		out.VersionID++
		if err := s.p.Repo.PutGroupName(ctx, group, out, now); err != nil {
			return err
		}
		return s.p.Audit.Record(ctx, reviewEvent(c, tableGroupName, auditevent.ActionUpdate, auditevent.OutcomeSuccess))
	})
	if err != nil {
		return GroupName{}, fmt.Errorf("terminology: group name write: %w", err)
	}
	return out, nil
}

// refuse audits a write by c, who is no reviewer, as refused, and returns ErrForbidden.
func (s *GroupReviews) refuse(ctx context.Context, c Caller) error {
	err := s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		return s.p.Audit.Record(ctx, reviewEvent(c, tableGroupName, auditevent.ActionUpdate, auditevent.OutcomeMinorFailure))
	})
	if err != nil {
		return fmt.Errorf("terminology: audit a refused group name write: %w", err)
	}
	return ErrForbidden
}
