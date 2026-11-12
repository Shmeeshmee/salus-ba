package terminology_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/terminology"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

// fakeGroupNames is the GroupReviewRepository: the catalog's groups with their names kept by group and language.
type fakeGroupNames struct {
	fakeCatalog
	names map[[2]string]terminology.GroupName
	at    time.Time
}

func newFakeGroupNames() *fakeGroupNames {
	f := &fakeGroupNames{names: map[[2]string]terminology.GroupName{}}
	groups, _, _ := f.fakeCatalog.DiagnosisGroups(context.Background())
	for _, g := range groups {
		for _, n := range g.Names {
			n.VersionID = 1
			f.names[[2]string{g.Code, n.Language}] = n
		}
	}
	return f
}

func (f *fakeGroupNames) DiagnosisGroups(ctx context.Context) ([]terminology.DiagnosisGroup, []terminology.GroupRange, error) {
	groups, ranges, err := f.fakeCatalog.DiagnosisGroups(ctx)
	for i, g := range groups {
		for j, n := range g.Names {
			groups[i].Names[j] = f.names[[2]string{g.Code, n.Language}]
		}
	}
	return groups, ranges, err
}

func (f *fakeGroupNames) LockGroupName(ctx context.Context, group, language string) (terminology.GroupName, error) {
	if ctx.Value(txKey{}) == nil {
		return terminology.GroupName{}, errors.New("locked outside a transaction")
	}
	n, ok := f.names[[2]string{group, language}]
	if !ok {
		return terminology.GroupName{}, terminology.ErrNoGroupName
	}
	return n, nil
}

func (f *fakeGroupNames) PutGroupName(_ context.Context, group string, n terminology.GroupName, at time.Time) error {
	f.names[[2]string{group, n.Language}], f.at = n, at
	return nil
}

var (
	groupReviewer  = terminology.Caller{AccountID: "00000000-0000-4000-8000-0000000000e1", Staff: true, Reviewer: true}
	groupReviewNow = time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)
)

func groupReviews(repo *fakeGroupNames) (*terminology.GroupReviews, *fakeAudit) {
	audit := &fakeAudit{}
	return terminology.NewGroupReviews(terminology.GroupReviewPorts{
		Repo: repo, Audit: audit, Tx: &fakeReviewTx{}, Clock: clock.Fixed{T: groupReviewNow},
	}), audit
}

// Every staff member reads the group names of a language with their reviews, for the panel's list.
func TestGroupReviewsList_GivesEachGroupsNameInTheLanguage(t *testing.T) {
	svc, _ := groupReviews(newFakeGroupNames())
	got, err := svc.List(context.Background(), catalogStaff, "en")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || got[1].Code != "common-cold" || got[1].Parent != "respiratory-infections" ||
		got[1].Name.Text != "Common cold (en)" || got[1].Name.VersionID != 1 || got[1].Name.Status != terminology.ReviewApproved {
		t.Errorf("List = %+v", got)
	}
	if _, err := svc.List(context.Background(), terminology.Caller{}, "en"); !errors.Is(err, terminology.ErrForbidden) {
		t.Errorf("a patient: %v", err)
	}
	if _, err := svc.List(context.Background(), catalogStaff, "la"); !errors.Is(err, terminology.ErrInvalidReview) {
		t.Errorf("Latin: %v", err)
	}
}

// A groupReviewer's edit replaces the name, trimmed, sets it back to unreviewed keeping its last review, and is audited
// on diagnosis_group_name; the same text changes nothing and is not audited.
func TestEditGroupName_ReplacesTheNameAsUnreviewed(t *testing.T) {
	repo := newFakeGroupNames()
	svc, audit := groupReviews(repo)
	got, err := svc.EditGroupName(context.Background(), groupReviewer, "common-cold", "en", "  Head cold ", 1)
	if err != nil {
		t.Fatal(err)
	}
	if got.Text != "Head cold" || got.Status != terminology.ReviewUnreviewed || got.VersionID != 2 || !repo.at.Equal(groupReviewNow) ||
		repo.names[[2]string{"common-cold", "en"}] != got {
		t.Errorf("EditGroupName = %+v, stored at %v", got, repo.at)
	}
	if len(audit.events) != 1 || audit.events[0].EntityName != "diagnosis_group_name" || audit.events[0].Outcome != auditevent.OutcomeSuccess ||
		audit.outside != 0 {
		t.Errorf("audit = %+v", audit.events)
	}
	if _, err := svc.EditGroupName(context.Background(), groupReviewer, "common-cold", "en", "Head cold", 2); err != nil || len(audit.events) != 1 {
		t.Errorf("the same text: %v, %d events", err, len(audit.events))
	}
}

// A groupReviewer approves a name or marks it as needing changes (with a note), naming themself and now.
func TestReviewGroupName_RecordsTheDecision(t *testing.T) {
	repo := newFakeGroupNames()
	svc, audit := groupReviews(repo)
	got, err := svc.ReviewGroupName(context.Background(), groupReviewer, "common-cold", "bs", terminology.Decision{Status: terminology.ReviewNeedsChanges, Note: " Too formal "}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != terminology.ReviewNeedsChanges || got.Note != "Too formal" || got.ReviewedBy != groupReviewer.AccountID ||
		!got.ReviewedAt.Equal(groupReviewNow) || got.VersionID != 2 || got.Text != "Common cold (bs)" || len(audit.events) != 1 {
		t.Errorf("ReviewGroupName = %+v, %d events", got, len(audit.events))
	}
}

func TestGroupReviews_Refusals(t *testing.T) {
	approve := terminology.Decision{Status: terminology.ReviewApproved}
	for name, tt := range map[string]struct {
		call func(*terminology.GroupReviews) error
		want error
	}{
		"edit by a non-groupReviewer": {func(s *terminology.GroupReviews) error {
			_, err := s.EditGroupName(context.Background(), catalogStaff, "common-cold", "en", "x", 1)
			return err
		}, terminology.ErrForbidden},
		"review by a non-groupReviewer": {func(s *terminology.GroupReviews) error {
			_, err := s.ReviewGroupName(context.Background(), catalogStaff, "common-cold", "en", approve, 1)
			return err
		}, terminology.ErrForbidden},
		"a stale version": {func(s *terminology.GroupReviews) error {
			_, err := s.ReviewGroupName(context.Background(), groupReviewer, "common-cold", "en", approve, 3)
			return err
		}, terminology.ErrVersionConflict},
		"an unknown group": {func(s *terminology.GroupReviews) error {
			_, err := s.EditGroupName(context.Background(), groupReviewer, "nothing", "en", "x", 1)
			return err
		}, terminology.ErrNoGroupName},
		"a blank name": {func(s *terminology.GroupReviews) error {
			_, err := s.EditGroupName(context.Background(), groupReviewer, "common-cold", "en", "  ", 1)
			return err
		}, terminology.ErrInvalidReview},
		"a name over 100 characters": {func(s *terminology.GroupReviews) error {
			_, err := s.EditGroupName(context.Background(), groupReviewer, "common-cold", "en", string(make([]rune, 101)), 1)
			return err
		}, terminology.ErrInvalidReview},
		"Latin": {func(s *terminology.GroupReviews) error {
			_, err := s.EditGroupName(context.Background(), groupReviewer, "common-cold", "la", "x", 1)
			return err
		}, terminology.ErrInvalidReview},
		"needs changes without a note": {func(s *terminology.GroupReviews) error {
			_, err := s.ReviewGroupName(context.Background(), groupReviewer, "common-cold", "en", terminology.Decision{Status: terminology.ReviewNeedsChanges}, 1)
			return err
		}, terminology.ErrNoteRequired},
	} {
		repo := newFakeGroupNames()
		svc, audit := groupReviews(repo)
		if err := tt.call(svc); !errors.Is(err, tt.want) {
			t.Errorf("%s: %v, want %v", name, err, tt.want)
		}
		if errors.Is(tt.want, terminology.ErrForbidden) && (len(audit.events) != 1 || audit.events[0].Outcome == auditevent.OutcomeSuccess) {
			t.Errorf("%s: the refusal is not audited: %+v", name, audit.events)
		}
		if n := repo.names[[2]string{"common-cold", "en"}]; n.VersionID != 1 {
			t.Errorf("%s: the name changed: %+v", name, n)
		}
	}
}
