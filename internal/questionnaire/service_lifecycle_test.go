package questionnaire_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/questionnaire"
)

func retired(f fixture) {
	f.repo.rows[0].Status, f.repo.rows[0].RetiredAt = questionnaire.StatusRetired, now
}

var lifecycleRefusals = []struct {
	setup     func(f fixture)
	call      func(f fixture, ctx context.Context) error
	name      string
	subtype   string
	entity    string
	want      error
	action    auditevent.Action
	unchanged string // the row that must keep its version
}{
	{name: "edit an active version", want: questionnaire.ErrNotDraft, subtype: auditevent.CodeUpdate, action: auditevent.ActionUpdate, entity: activeA, unchanged: activeA,
		call: func(f fixture, ctx context.Context) error {
			_, err := f.svc.UpdateDraft(ctx, adminA, activeA, content(), 2)
			return err
		}},
	{name: "edit a retired version", setup: retired, want: questionnaire.ErrNotDraft, subtype: auditevent.CodeUpdate, action: auditevent.ActionUpdate, entity: activeA, unchanged: activeA,
		call: func(f fixture, ctx context.Context) error {
			_, err := f.svc.UpdateDraft(ctx, adminA, activeA, content(), 2)
			return err
		}},
	{name: "edit a stale draft", want: questionnaire.ErrConflict, subtype: auditevent.CodeUpdate, action: auditevent.ActionUpdate, entity: draftA, unchanged: draftA,
		call: func(f fixture, ctx context.Context) error {
			_, err := f.svc.UpdateDraft(ctx, adminA, draftA, content(), 7)
			return err
		}},
	{name: "edit another clinic's", want: questionnaire.ErrNotFound, subtype: auditevent.CodeUpdate, action: auditevent.ActionUpdate, unchanged: activeB,
		call: func(f fixture, ctx context.Context) error {
			_, err := f.svc.UpdateDraft(ctx, adminA, activeB, content(), 2)
			return err
		}},
	{name: "edit a missing one", want: questionnaire.ErrNotFound, subtype: auditevent.CodeUpdate, action: auditevent.ActionUpdate, unchanged: draftA,
		call: func(f fixture, ctx context.Context) error {
			_, err := f.svc.UpdateDraft(ctx, adminA, missing, content(), 1)
			return err
		}},
	{name: "delete a published one", want: questionnaire.ErrNotDraft, subtype: auditevent.CodeDelete, action: auditevent.ActionDelete, entity: activeA, unchanged: activeA,
		call: func(f fixture, ctx context.Context) error { return f.svc.DeleteDraft(ctx, adminA, activeA) }},
	{name: "delete another clinic's", want: questionnaire.ErrNotFound, subtype: auditevent.CodeDelete, action: auditevent.ActionDelete, unchanged: activeB,
		call: func(f fixture, ctx context.Context) error { return f.svc.DeleteDraft(ctx, adminA, activeB) }},
	{name: "publish an active one", want: questionnaire.ErrNotDraft, subtype: auditevent.CodeUpdate, action: auditevent.ActionUpdate, entity: activeA, unchanged: activeA,
		call: func(f fixture, ctx context.Context) error {
			_, err := f.svc.Publish(ctx, adminA, activeA, 2)
			return err
		}},
	{name: "publish a retired one", setup: retired, want: questionnaire.ErrNotDraft, subtype: auditevent.CodeUpdate, action: auditevent.ActionUpdate, entity: activeA, unchanged: activeA,
		call: func(f fixture, ctx context.Context) error {
			_, err := f.svc.Publish(ctx, adminA, activeA, 2)
			return err
		}},
	{name: "publish a stale draft", want: questionnaire.ErrConflict, subtype: auditevent.CodeUpdate, action: auditevent.ActionUpdate, entity: draftA, unchanged: activeA,
		call: func(f fixture, ctx context.Context) error {
			_, err := f.svc.Publish(ctx, adminA, draftA, 7)
			return err
		}},
	{name: "publish another clinic's", want: questionnaire.ErrNotFound, subtype: auditevent.CodeUpdate, action: auditevent.ActionUpdate, unchanged: activeB,
		call: func(f fixture, ctx context.Context) error {
			_, err := f.svc.Publish(ctx, adminA, activeB, 2)
			return err
		}},
	{name: "retire a draft", want: questionnaire.ErrNotActive, subtype: auditevent.CodeUpdate, action: auditevent.ActionUpdate, entity: draftA, unchanged: draftA,
		call: func(f fixture, ctx context.Context) error { _, err := f.svc.Retire(ctx, adminA, draftA); return err }},
	{name: "retire a retired one", setup: retired, want: questionnaire.ErrNotActive, subtype: auditevent.CodeUpdate, action: auditevent.ActionUpdate, entity: activeA, unchanged: activeA,
		call: func(f fixture, ctx context.Context) error { _, err := f.svc.Retire(ctx, adminA, activeA); return err }},
	{name: "retire another clinic's", want: questionnaire.ErrNotFound, subtype: auditevent.CodeUpdate, action: auditevent.ActionUpdate, unchanged: activeB,
		call: func(f fixture, ctx context.Context) error { _, err := f.svc.Retire(ctx, adminA, activeB); return err }},
	{name: "revise with a draft existing", want: questionnaire.ErrDraftExists, subtype: auditevent.CodeCreate, action: auditevent.ActionCreate, entity: activeA, unchanged: draftA,
		call: func(f fixture, ctx context.Context) error { _, err := f.svc.Revise(ctx, adminA, activeA); return err }},
	{name: "revise a draft", want: questionnaire.ErrNotActive, subtype: auditevent.CodeCreate, action: auditevent.ActionCreate, entity: draftA, unchanged: draftA,
		call: func(f fixture, ctx context.Context) error { _, err := f.svc.Revise(ctx, adminA, draftA); return err }},
	{name: "revise another clinic's", want: questionnaire.ErrNotFound, subtype: auditevent.CodeCreate, action: auditevent.ActionCreate, unchanged: activeB,
		call: func(f fixture, ctx context.Context) error { _, err := f.svc.Revise(ctx, adminA, activeB); return err }},
}

func TestLifecycle_RefusesEachStepOnTheWrongVersionAndAuditsTheRefusal(t *testing.T) {
	t.Parallel()
	for _, tt := range lifecycleRefusals {
		f := newFixture()
		if tt.setup != nil {
			tt.setup(f)
		}
		before, rows := f.row(tt.unchanged), len(f.repo.rows)
		err := tt.call(f, t.Context())
		if !errors.Is(err, tt.want) {
			t.Errorf("%s = %v, want %v", tt.name, err, tt.want)
		}
		if after := f.row(tt.unchanged); !sameRow(before, after) || len(f.repo.rows) != rows {
			t.Errorf("%s changed the repository: %+v, then %+v", tt.name, before, after)
		}
		entity := ""
		if tt.entity != "" {
			entity = "Questionnaire/" + tt.entity
		}
		assertEvents(t, f.audit, staffEvent(adminA, tt.subtype, tt.action, entity, auditevent.OutcomeMinorFailure))
	}
}

func TestPublish_MakesTheDraftActiveAndRetiresTheActiveVersionOfItsSeries(t *testing.T) {
	t.Parallel()
	f := newFixture()

	got, err := f.svc.Publish(t.Context(), practitionerA1, draftA, 1)
	if err != nil || got.ID != draftA || got.Status != questionnaire.StatusActive || !got.PublishedAt.Equal(now) {
		t.Fatalf("Publish() = %+v, %v; want draft A active, published now", got, err)
	}
	if v1 := f.row(activeA); v1.Status != questionnaire.StatusRetired || !v1.RetiredAt.Equal(now) {
		t.Errorf("version 1 after publishing version 2: %+v; want retired now", v1)
	}
	if b := f.row(activeB); b.Status != questionnaire.StatusActive {
		t.Errorf("clinic B's series: %s, want active", b.Status)
	}
	assertEvents(t, f.audit, staffEvent(practitionerA1, auditevent.CodeUpdate, auditevent.ActionUpdate, "Questionnaire/"+draftA, auditevent.OutcomeSuccess))
}

// A draft may keep a section without items, but it does not publish (FHIR que-1): 422 naming each empty
// section, in document order; nothing changes, and the refusal is audited.
func TestPublish_RefusesADraftWithAnEmptySection(t *testing.T) {
	t.Parallel()
	f := newFixture()
	for i := range f.repo.rows {
		if f.repo.rows[i].ID == draftA {
			f.repo.rows[i].Items = []questionnaire.Item{section("s1"), section("s2", question("q", questionnaire.ItemString)), section("s3")}
		}
	}
	before, active := f.row(draftA), f.row(activeA)

	_, err := f.svc.Publish(t.Context(), practitionerA1, draftA, 1)

	invalid, ok := errors.AsType[*questionnaire.InvalidError](err)
	if !ok || !slices.Equal(invalid.LinkIDs, []string{"s1", "s3"}) {
		t.Fatalf("Publish() = %v, want an *InvalidError naming s1 and s3", err)
	}
	if !sameRow(f.row(draftA), before) || !sameRow(f.row(activeA), active) {
		t.Errorf("Publish() changed the series: %+v, %+v", f.row(draftA), f.row(activeA))
	}
	assertEvents(t, f.audit, staffEvent(practitionerA1, auditevent.CodeUpdate, auditevent.ActionUpdate, "Questionnaire/"+draftA, auditevent.OutcomeMinorFailure))
}

func TestPublish_AFailedAuditRollsTheRetirementBack(t *testing.T) {
	t.Parallel()
	f := newFixture()
	f.audit.err = errors.New("audit down")

	if _, err := f.svc.Publish(t.Context(), practitionerA1, draftA, 1); err == nil {
		t.Fatal("Publish() with the audit failing = nil, want an error")
	}
	if v1, v2 := f.row(activeA), f.row(draftA); v1.Status != questionnaire.StatusActive || v2.Status != questionnaire.StatusDraft {
		t.Errorf("after the rollback: version 1 %s, version 2 %s; want active and draft", v1.Status, v2.Status)
	}
}

func TestRetire_RetiresTheActiveVersion(t *testing.T) {
	t.Parallel()
	f := newFixture()

	got, err := f.svc.Retire(t.Context(), adminA, activeA)
	if err != nil || got.Status != questionnaire.StatusRetired || !got.RetiredAt.Equal(now) {
		t.Fatalf("Retire() = %+v, %v; want retired now", got, err)
	}
	assertEvents(t, f.audit, staffEvent(adminA, auditevent.CodeUpdate, auditevent.ActionUpdate, "Questionnaire/"+activeA, auditevent.OutcomeSuccess))
}

func TestRevise_CopiesTheActiveVersionToTheNextDraftWithTheSameLinkIDs(t *testing.T) {
	t.Parallel()
	f := newFixture()
	if err := f.svc.DeleteDraft(t.Context(), adminA, draftA); err != nil {
		t.Fatal(err)
	}
	f.audit.events = nil

	got, err := f.svc.Revise(t.Context(), practitionerA1, activeA)
	if err != nil {
		t.Fatal(err)
	}
	links := func(q questionnaire.Questionnaire) []string {
		var out []string
		for _, it := range q.Items {
			out = append(out, it.LinkID)
		}
		return out
	}
	if got.SeriesID != seriesA || got.Version != 2 || got.Status != questionnaire.StatusDraft || got.Title != placeholderTitle ||
		!slices.Equal(links(got), links(f.row(activeA))) || got.ID == activeA {
		t.Errorf("Revise() = %+v; want a new version 2 draft of series A with version 1's content and linkIds", got)
	}
	if f.row(activeA).Status != questionnaire.StatusActive {
		t.Error("version 1 is no longer active after the revise; it stays active until version 2 is published")
	}
	assertEvents(t, f.audit, staffEvent(practitionerA1, auditevent.CodeCreate, auditevent.ActionCreate, "Questionnaire/"+got.ID, auditevent.OutcomeSuccess))
}
