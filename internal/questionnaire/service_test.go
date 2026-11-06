package questionnaire_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/questionnaire"
)

const (
	clinicA       = "a0000000-0000-4000-8000-00000000000a"
	clinicB       = "b0000000-0000-4000-8000-00000000000b"
	roleA1        = "a4000000-0000-4000-8000-0000000000a1"
	roleA2        = "a4000000-0000-4000-8000-0000000000a2"
	roleB         = "b4000000-0000-4000-8000-0000000000b1"
	practitioner1 = "a3000000-0000-4000-8000-0000000000a1"
	practitioner2 = "a3000000-0000-4000-8000-0000000000a2"
	seriesA       = "a6000000-0000-4000-8000-0000000000a1"
	seriesB       = "b6000000-0000-4000-8000-0000000000b1"
	activeA       = "a7000000-0000-4000-8000-0000000000a1" // version 1 of seriesA
	draftA        = "a7000000-0000-4000-8000-0000000000a2" // version 2 of seriesA
	activeB       = "b7000000-0000-4000-8000-0000000000b1"
	missing       = "00000000-0000-4000-8000-000000000000"

	// Synthetic, placeholder wording: asserted absent from every error.
	placeholderTitle = "Placeholder intake questionnaire"
	placeholderText  = "Placeholder question about symptoms"
)

var now = time.Date(2026, 11, 6, 9, 30, 0, 0, time.UTC)

var (
	practitionerA1 = questionnaire.Caller{AccountID: "a1000000-0000-4000-8000-0000000000a1", OrganizationID: clinicA, PractitionerID: practitioner1}
	adminA         = questionnaire.Caller{AccountID: "a1000000-0000-4000-8000-0000000000ad", OrganizationID: clinicA, Admin: true}
	adminB         = questionnaire.Caller{AccountID: "b1000000-0000-4000-8000-0000000000bd", OrganizationID: clinicB, Admin: true}
	patient        = questionnaire.Caller{AccountID: "c1000000-0000-4000-8000-0000000000c1"}
)

func items() []questionnaire.Item {
	return []questionnaire.Item{
		{LinkID: "q1", Text: placeholderText, Type: questionnaire.ItemBoolean, Required: true},
		{LinkID: "q2", Text: "Placeholder choice", Type: questionnaire.ItemChoice, Options: []questionnaire.Option{{Value: "One"}, {Value: "Two"}}},
	}
}

func content() questionnaire.Content {
	return questionnaire.Content{Title: placeholderTitle, Description: "Placeholder", Items: items()}
}

func TestCreate_StoresAValidQuestionnaireAsVersion1DraftOfANewSeries(t *testing.T) {
	t.Parallel()
	f := newFixture()

	got, err := f.svc.Create(t.Context(), practitionerA1, content())
	if err != nil {
		t.Fatal(err)
	}
	if got.ID == "" || got.SeriesID == "" || got.Version != 1 || got.Status != questionnaire.StatusDraft ||
		got.OrganizationID != clinicA || got.Title != placeholderTitle || len(got.Items) != 2 || !got.LastUpdated.Equal(now) {
		t.Errorf("Create() = %+v; want version 1 draft of clinic A with the content, changed now", got)
	}
	assertEvents(t, f.audit, staffEvent(practitionerA1, auditevent.CodeCreate, auditevent.ActionCreate, "Questionnaire/"+got.ID, auditevent.OutcomeSuccess))
	if len(f.repo.outside) != 0 {
		t.Errorf("repository calls outside a transaction: %v", f.repo.outside)
	}
}

func TestService_RefusesAPatientWithForbiddenAndAuditsIt(t *testing.T) {
	t.Parallel()
	calls := map[string]func(svc *questionnaire.Service, ctx context.Context) error{
		"Create": func(svc *questionnaire.Service, ctx context.Context) error {
			_, err := svc.Create(ctx, patient, content())
			return err
		},
		"UpdateDraft": func(svc *questionnaire.Service, ctx context.Context) error {
			_, err := svc.UpdateDraft(ctx, patient, draftA, content(), 1)
			return err
		},
		"DeleteDraft": func(svc *questionnaire.Service, ctx context.Context) error {
			return svc.DeleteDraft(ctx, patient, draftA)
		},
		"Publish": func(svc *questionnaire.Service, ctx context.Context) error {
			_, err := svc.Publish(ctx, patient, draftA, 1)
			return err
		},
		"Retire": func(svc *questionnaire.Service, ctx context.Context) error {
			_, err := svc.Retire(ctx, patient, activeA)
			return err
		},
		"Revise": func(svc *questionnaire.Service, ctx context.Context) error {
			_, err := svc.Revise(ctx, patient, activeA)
			return err
		},
		"Read": func(svc *questionnaire.Service, ctx context.Context) error {
			_, err := svc.Read(ctx, patient, activeA)
			return err
		},
		"Search": func(svc *questionnaire.Service, ctx context.Context) error {
			_, err := svc.Search(ctx, patient, questionnaire.Filter{})
			return err
		},
		"Default": func(svc *questionnaire.Service, ctx context.Context) error {
			_, err := svc.Default(ctx, patient, roleA1)
			return err
		},
		"SetDefault": func(svc *questionnaire.Service, ctx context.Context) error {
			_, err := svc.SetDefault(ctx, patient, roleA1, activeA)
			return err
		},
		"ClearDefault": func(svc *questionnaire.Service, ctx context.Context) error {
			return svc.ClearDefault(ctx, patient, roleA1)
		},
	}
	for name, call := range calls {
		f := newFixture()
		before := f.repo.snapshot()
		if err := call(f.svc, t.Context()); !errors.Is(err, questionnaire.ErrForbidden) {
			t.Errorf("%s as a patient = %v, want ErrForbidden", name, err)
		}
		if !slices.EqualFunc(before.rows, f.repo.rows, sameRow) || len(f.repo.defaults) != 0 {
			t.Errorf("%s as a patient changed the repository", name)
		}
		got := events(f.audit)
		if len(got) != 1 || got[0].agent != patient.AccountID || got[0].purpose != auditevent.CodePATRQT ||
			got[0].entity != "" || got[0].outcome != auditevent.OutcomeMinorFailure {
			t.Errorf("%s as a patient audited %+v; want one refusal by the patient, PATRQT, no entity", name, got)
		}
	}
}

func sameRow(a, b questionnaire.Questionnaire) bool {
	return a.ID == b.ID && a.Status == b.Status && a.VersionID == b.VersionID
}

func TestUpdateDraft_SetsTheContentOfTheCurrentDraftVersion(t *testing.T) {
	t.Parallel()
	f := newFixture()
	in := questionnaire.Content{Title: "Placeholder renamed", Items: items()[:1]}

	got, err := f.svc.UpdateDraft(t.Context(), adminA, draftA, in, 1)
	if err != nil || got.Title != "Placeholder renamed" || len(got.Items) != 1 || got.VersionID != 2 || !got.LastUpdated.Equal(now) {
		t.Fatalf("UpdateDraft() = %+v, %v; want the new content at version 2", got, err)
	}
	assertEvents(t, f.audit, staffEvent(adminA, auditevent.CodeUpdate, auditevent.ActionUpdate, "Questionnaire/"+draftA, auditevent.OutcomeSuccess))
}

// retired retires version 1 of series A.
// lifecycleRefusals are the lifecycle steps on the wrong version, by clinic A's admin.
func TestDeleteDraft_DeletesTheDraft(t *testing.T) {
	t.Parallel()
	f := newFixture()
	if err := f.svc.DeleteDraft(t.Context(), practitionerA1, draftA); err != nil {
		t.Fatal(err)
	}
	if f.row(draftA).ID != "" {
		t.Error("the draft is still there")
	}
	assertEvents(t, f.audit, staffEvent(practitionerA1, auditevent.CodeDelete, auditevent.ActionDelete, "Questionnaire/"+draftA, auditevent.OutcomeSuccess))
}

func TestRead_GivesStaffTheirClinicsVersionsOnly(t *testing.T) {
	t.Parallel()
	f := newFixture()

	got, err := f.svc.Read(t.Context(), practitionerA1, activeA)
	if err != nil || got.ID != activeA {
		t.Errorf("Read() of clinic A's = %+v, %v; want it", got, err)
	}
	for _, id := range []string{activeB, missing, "not-a-uuid"} {
		if _, err := f.svc.Read(t.Context(), practitionerA1, id); !errors.Is(err, questionnaire.ErrNotFound) {
			t.Errorf("Read(%s) = %v, want ErrNotFound", id, err)
		}
	}
	refused := staffEvent(practitionerA1, auditevent.CodeRead, auditevent.ActionRead, "", auditevent.OutcomeMinorFailure)
	assertEvents(t, f.audit, staffEvent(practitionerA1, auditevent.CodeRead, auditevent.ActionRead, "Questionnaire/"+activeA, auditevent.OutcomeSuccess),
		refused, refused, refused)
}

func TestRead_DoesNotRestOnRowLevelSecurityAlone(t *testing.T) {
	t.Parallel()
	f := newFixture()
	f.repo.leak = true

	if _, err := f.svc.Read(t.Context(), practitionerA1, activeB); !errors.Is(err, questionnaire.ErrNotFound) {
		t.Errorf("Read() of clinic B's with a leaking scope = %v, want ErrNotFound", err)
	}
	got, err := f.svc.Search(t.Context(), practitionerA1, questionnaire.Filter{})
	if err != nil || len(got) != 2 {
		t.Errorf("Search() with a leaking scope = %d versions, %v; want clinic A's 2", len(got), err)
	}
}

func TestSearch_ReturnsTheClinicsVersionsByFilterAndAuditsEach(t *testing.T) {
	t.Parallel()
	f := newFixture()

	got, err := f.svc.Search(t.Context(), adminA, questionnaire.Filter{SeriesID: seriesA, Status: questionnaire.StatusActive})
	if err != nil || len(got) != 1 || got[0].ID != activeA {
		t.Fatalf("Search(series A, active) = %+v, %v; want version 1", got, err)
	}
	if got, err := f.svc.Search(t.Context(), adminA, questionnaire.Filter{SeriesID: seriesB}); err != nil || len(got) != 0 {
		t.Errorf("Search(series B) in clinic A = %+v, %v; want none", got, err)
	}
	all, err := f.svc.Search(t.Context(), adminA, questionnaire.Filter{})
	if err != nil || len(all) != 2 {
		t.Errorf("Search() in clinic A = %d, %v; want 2", len(all), err)
	}
	search := func(entity string) event {
		return staffEvent(adminA, auditevent.CodeSearchType, auditevent.ActionExecute, entity, auditevent.OutcomeSuccess)
	}
	assertEvents(t, f.audit, search("Questionnaire/"+activeA), search(""), search("Questionnaire/"+activeA), search("Questionnaire/"+draftA))
}

func TestService_ReturnsTheRepositoryErrorWrapped(t *testing.T) {
	t.Parallel()
	f := newFixture()
	f.repo.err = errors.New("db down")

	if _, err := f.svc.Read(t.Context(), adminA, activeA); err == nil || errors.Is(err, questionnaire.ErrNotFound) {
		t.Errorf("Read() with the repository failing = %v, want a wrapped error", err)
	}
	if _, err := f.svc.Search(t.Context(), adminA, questionnaire.Filter{}); err == nil {
		t.Error("Search() with the repository failing = nil, want an error")
	}
	if len(f.audit.events) != 0 {
		t.Errorf("audited %d events for failed accesses that rolled back", len(f.audit.events))
	}
}

func TestService_AnotherClinicsAdminSeesNothingOfClinicA(t *testing.T) {
	t.Parallel()
	f := newFixture()
	if _, err := f.svc.Read(t.Context(), adminB, activeA); !errors.Is(err, questionnaire.ErrNotFound) {
		t.Errorf("Read() of clinic A's by clinic B's admin = %v, want ErrNotFound", err)
	}
	if _, err := f.svc.Publish(t.Context(), adminB, draftA, 1); !errors.Is(err, questionnaire.ErrNotFound) {
		t.Errorf("Publish() of clinic A's by clinic B's admin = %v, want ErrNotFound", err)
	}
}
