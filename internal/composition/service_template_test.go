package composition_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/composition"
)

// Staff of the clinic, practitioners and admins alike, create a template of their clinic; it is version 1,
// with its translations, and the write is audited on the document_template table.
func TestCreateTemplate_StoresTheClinicsTemplateAsVersion1(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name string
		who  int
	}{{"practitioner", 0}, {"admin", 1}} {
		f := newFixture(t)
		caller := pract
		if c.who == 1 {
			caller = admin
		}
		got, err := f.svc.CreateTemplate(t.Context(), caller, content())
		if err != nil {
			t.Fatalf("%s: CreateTemplate() = %v", c.name, err)
		}
		if got.OrganizationID != clinicA || got.VersionID != 1 || got.Title != content().Title || len(got.Translations) != 1 || got.LastUpdated != now {
			t.Errorf("%s: CreateTemplate() = %+v", c.name, got)
		}
		if len(f.audit.events) != 1 || f.audit.events[0].Action != auditevent.ActionCreate || f.audit.events[0].EntityName != "document_template" ||
			f.audit.events[0].AgentAccountID != caller.AccountID || f.audit.events[0].Outcome != auditevent.OutcomeSuccess {
			t.Errorf("%s: audit = %+v", c.name, f.audit.events)
		}
	}
}

// An invalid template is refused before anything is stored, audited as a failure; a patient may not keep
// templates.
func TestCreateTemplate_RefusesAnInvalidTemplateAndAPatient(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	c := content()
	c.Title = ""
	if _, err := f.svc.CreateTemplate(t.Context(), pract, c); !errors.Is(err, composition.ErrInvalidTemplate) {
		t.Errorf("CreateTemplate(no title) = %v, want ErrInvalidTemplate", err)
	}
	if _, err := f.svc.CreateTemplate(t.Context(), me, content()); !errors.Is(err, composition.ErrForbidden) {
		t.Errorf("CreateTemplate(patient) = %v, want ErrForbidden", err)
	}
	if len(f.repo.templates) != 0 || len(f.audit.events) != 2 || f.audit.events[0].Outcome != auditevent.OutcomeMinorFailure ||
		f.audit.events[1].Outcome != auditevent.OutcomeMinorFailure {
		t.Errorf("stored %d, audit %+v; want nothing stored, two refusals", len(f.repo.templates), f.audit.events)
	}
}

// An edit names the version it was read at and replaces the content as the next version; another version is
// refused, as is a template of another clinic (not found).
func TestUpdateTemplate_ReplacesTheContentAtTheVersionRead(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	tpl := f.template(t)
	c := content()
	c.Status, c.Translations = composition.TemplateActive, nil
	got, err := f.svc.UpdateTemplate(t.Context(), admin, tpl.ID, c, 1)
	if err != nil || got.VersionID != 2 || got.Status != composition.TemplateActive || len(got.Translations) != 0 {
		t.Fatalf("UpdateTemplate() = %+v, %v", got, err)
	}
	if _, err := f.svc.UpdateTemplate(t.Context(), admin, tpl.ID, c, 1); !errors.Is(err, composition.ErrVersionMismatch) {
		t.Errorf("UpdateTemplate(stale) = %v, want ErrVersionMismatch", err)
	}
	other := pract
	other.OrganizationID = clinicB
	if _, err := f.svc.UpdateTemplate(t.Context(), other, tpl.ID, c, 2); !errors.Is(err, composition.ErrTemplateNotFound) {
		t.Errorf("UpdateTemplate(other clinic) = %v, want ErrTemplateNotFound", err)
	}
	c.Fields = nil
	if _, err := f.svc.UpdateTemplate(t.Context(), admin, tpl.ID, c, 2); !errors.Is(err, composition.ErrInvalidTemplate) {
		t.Errorf("UpdateTemplate(no fields) = %v, want ErrInvalidTemplate", err)
	}
	var outcomes []auditevent.Outcome
	for _, e := range f.audit.events {
		outcomes = append(outcomes, e.Outcome)
	}
	if want := []auditevent.Outcome{0, 4, 4, 4}; !slices.Equal(outcomes, want) {
		t.Errorf("audit outcomes = %v, want %v", outcomes, want)
	}
}

// The clinic's staff list its templates (not another clinic's); a patient may not. Not audited: a template
// holds no personal data.
func TestTemplates_ListsTheClinicsTemplatesToItsStaff(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	tpl := f.template(t)
	got, err := f.svc.Templates(t.Context(), pract)
	if err != nil || len(got) != 1 || got[0].ID != tpl.ID {
		t.Errorf("Templates() = %+v, %v", got, err)
	}
	other := pract
	other.OrganizationID = clinicB
	if got, err := f.svc.Templates(t.Context(), other); err != nil || len(got) != 0 {
		t.Errorf("Templates(other clinic) = %+v, %v; want none", got, err)
	}
	if _, err := f.svc.Templates(t.Context(), me); !errors.Is(err, composition.ErrForbidden) {
		t.Errorf("Templates(patient) = %v, want ErrForbidden", err)
	}
	if len(f.audit.events) != 0 {
		t.Errorf("audit = %+v, want none", f.audit.events)
	}
	f.repo.err = errBackend
	if _, err := f.svc.Templates(t.Context(), pract); !errors.Is(err, errBackend) {
		t.Errorf("Templates(down) = %v, want the failure", err)
	}
}

// The operator seeds a template into a clinic once: a second seed of the same title creates nothing. The write
// is audited with the operator as its agent; invalid content is refused.
func TestSeedTemplate_CreatesTheTemplateOnce(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	got, created, err := f.svc.SeedTemplate(t.Context(), "salusctl", clinicA, content())
	if err != nil || !created || got.OrganizationID != clinicA || got.Status != composition.TemplateDraft {
		t.Fatalf("SeedTemplate() = %+v, %v, %v", got, created, err)
	}
	again, created, err := f.svc.SeedTemplate(t.Context(), "salusctl", clinicA, content())
	if err != nil || created || again.ID != got.ID || len(f.repo.templates) != 1 {
		t.Errorf("SeedTemplate() again = %+v, %v, %v; want the same template, not created", again, created, err)
	}
	if len(f.audit.events) != 1 || f.audit.events[0].AgentName != "salusctl" || f.audit.events[0].OrganizationID != clinicA ||
		f.audit.events[0].Action != auditevent.ActionCreate {
		t.Errorf("audit = %+v", f.audit.events)
	}
	c := content()
	c.Fields = nil
	if _, _, err := f.svc.SeedTemplate(t.Context(), "salusctl", clinicA, c); !errors.Is(err, composition.ErrInvalidTemplate) {
		t.Errorf("SeedTemplate(invalid) = %v, want ErrInvalidTemplate", err)
	}
}
