package questionnaire_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/questionnaire"
)

func TestSetDefault_ThePractitionerSetsTheirOwnRolesAndAnAdminAnyRoleOfTheClinic(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		role   string
		caller questionnaire.Caller
	}{
		{caller: practitionerA1, role: roleA1},
		{caller: adminA, role: roleA2},
	} {
		f := newFixture()
		// Any version of the series names it: here the draft of series A, whose version 1 is active.
		got, err := f.svc.SetDefault(t.Context(), tt.caller, tt.role, draftA)
		if err != nil || got.PractitionerRoleID != tt.role || got.SeriesID != seriesA || got.OrganizationID != clinicA || got.VersionID != 1 || !got.LastUpdated.Equal(now) {
			t.Errorf("SetDefault(%s) = %+v, %v; want series A as the role's default", tt.role, got, err)
		}
		read, err := f.svc.Default(t.Context(), tt.caller, tt.role)
		if err != nil || read.SeriesID != seriesA {
			t.Errorf("Default(%s) after setting = %+v, %v; want series A", tt.role, read, err)
		}
		if err := f.svc.ClearDefault(t.Context(), tt.caller, tt.role); err != nil {
			t.Errorf("ClearDefault(%s) = %v", tt.role, err)
		}
		read, err = f.svc.Default(t.Context(), tt.caller, tt.role)
		if err != nil || read.SeriesID != "" || read.VersionID != 0 || read.PractitionerRoleID != tt.role {
			t.Errorf("Default(%s) after clearing = %+v, %v; want none at version 0", tt.role, read, err)
		}
		entity := "PractitionerRole/" + tt.role
		assertEvents(t, f.audit,
			staffEvent(tt.caller, auditevent.CodeUpdate, auditevent.ActionUpdate, entity, auditevent.OutcomeSuccess),
			staffEvent(tt.caller, auditevent.CodeRead, auditevent.ActionRead, entity, auditevent.OutcomeSuccess),
			staffEvent(tt.caller, auditevent.CodeDelete, auditevent.ActionDelete, entity, auditevent.OutcomeSuccess),
			staffEvent(tt.caller, auditevent.CodeRead, auditevent.ActionRead, entity, auditevent.OutcomeSuccess),
		)
	}
}

func TestSetDefault_ReplacesTheRolesDefault(t *testing.T) {
	t.Parallel()
	f := newFixture()
	f.repo.rows = append(f.repo.rows, questionnaire.Questionnaire{ID: "a7000000-0000-4000-8000-0000000000a9", SeriesID: "a6000000-0000-4000-8000-0000000000a9",
		OrganizationID: clinicA, Version: 1, VersionID: 2, Status: questionnaire.StatusActive, Title: placeholderTitle, PublishedAt: now})
	if _, err := f.svc.SetDefault(t.Context(), adminA, roleA1, activeA); err != nil {
		t.Fatal(err)
	}
	got, err := f.svc.SetDefault(t.Context(), adminA, roleA1, "a7000000-0000-4000-8000-0000000000a9")
	if err != nil || got.SeriesID != "a6000000-0000-4000-8000-0000000000a9" || got.VersionID != 2 || len(f.repo.defaults) != 1 {
		t.Errorf("a second SetDefault() = %+v, %v (%d defaults); want the other series at version 2, one default", got, err, len(f.repo.defaults))
	}
}

func TestDefault_RefusesAndAuditsEveryoneElse(t *testing.T) {
	t.Parallel()
	retiredOnly := func(f fixture) {
		f.repo.rows = f.repo.rows[2:] // series A without versions in clinic A
		f.repo.rows = append(f.repo.rows,
			questionnaire.Questionnaire{ID: activeA, SeriesID: seriesA, OrganizationID: clinicA, Version: 1, VersionID: 3, Status: questionnaire.StatusRetired, Title: placeholderTitle, PublishedAt: now, RetiredAt: now})
	}
	draftOnly := func(f fixture) { f.repo.rows = f.repo.rows[1:] } // series A has only its draft
	tests := []struct {
		setup         func(f fixture)
		name, role, q string
		want          error
		caller        questionnaire.Caller
		entity        bool // the refusal names the role: it is the caller's to see
		readsToo      bool // Default and ClearDefault are refused too
	}{
		{name: "another practitioner's role", caller: practitionerA1, role: roleA2, q: activeA, want: questionnaire.ErrNotFound, readsToo: true},
		{name: "a role of another clinic, by the admin", caller: adminA, role: roleB, q: activeA, want: questionnaire.ErrNotFound, readsToo: true},
		{name: "another clinic's admin", caller: adminB, role: roleA1, q: activeB, want: questionnaire.ErrNotFound, readsToo: true},
		{name: "a missing role", caller: adminA, role: missing, q: activeA, want: questionnaire.ErrNotFound, readsToo: true},
		{name: "a questionnaire of another clinic", caller: adminA, role: roleA1, q: activeB, want: questionnaire.ErrNotFound, entity: true},
		{name: "a missing questionnaire", caller: adminA, role: roleA1, q: missing, want: questionnaire.ErrNotFound, entity: true},
		{name: "a questionnaire with only a draft", setup: draftOnly, caller: adminA, role: roleA1, q: draftA, want: questionnaire.ErrNotActive, entity: true},
		{name: "a retired questionnaire", setup: retiredOnly, caller: adminA, role: roleA1, q: activeA, want: questionnaire.ErrNotActive, entity: true},
	}
	for _, tt := range tests {
		f := newFixture()
		if tt.setup != nil {
			tt.setup(f)
		}
		entity := ""
		if tt.entity {
			entity = "PractitionerRole/" + tt.role
		}
		if _, err := f.svc.SetDefault(t.Context(), tt.caller, tt.role, tt.q); !errors.Is(err, tt.want) {
			t.Errorf("%s: SetDefault() = %v, want %v", tt.name, err, tt.want)
		}
		if len(f.repo.defaults) != 0 {
			t.Errorf("%s: a default was stored", tt.name)
		}
		want := []event{staffEvent(tt.caller, auditevent.CodeUpdate, auditevent.ActionUpdate, entity, auditevent.OutcomeMinorFailure)}
		if tt.readsToo {
			if _, err := f.svc.Default(t.Context(), tt.caller, tt.role); !errors.Is(err, tt.want) {
				t.Errorf("%s: Default() = %v, want %v", tt.name, err, tt.want)
			}
			if err := f.svc.ClearDefault(t.Context(), tt.caller, tt.role); !errors.Is(err, tt.want) {
				t.Errorf("%s: ClearDefault() = %v, want %v", tt.name, err, tt.want)
			}
			want = append(want,
				staffEvent(tt.caller, auditevent.CodeRead, auditevent.ActionRead, entity, auditevent.OutcomeMinorFailure),
				staffEvent(tt.caller, auditevent.CodeDelete, auditevent.ActionDelete, entity, auditevent.OutcomeMinorFailure))
		}
		assertEvents(t, f.audit, want...)
	}
}

func TestClearDefault_KeepsAnotherRolesDefault(t *testing.T) {
	t.Parallel()
	f := newFixture()
	ctx := context.Background()
	if _, err := f.svc.SetDefault(ctx, adminA, roleA2, activeA); err != nil {
		t.Fatal(err)
	}
	if err := f.svc.ClearDefault(ctx, adminA, roleA1); err != nil {
		t.Fatal(err)
	}
	if len(f.repo.defaults) != 1 || f.repo.defaults[0].PractitionerRoleID != roleA2 {
		t.Errorf("defaults after clearing role A1's: %+v; want role A2's", f.repo.defaults)
	}
}
