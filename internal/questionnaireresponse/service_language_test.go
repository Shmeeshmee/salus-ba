package questionnaireresponse_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/questionnaire"
	"github.com/Shmeeshmee/salus/internal/questionnaireresponse"
)

// translated gives the active version the original language bs and translations into the languages; the
// questionnaire service gives a patient only its complete ones (docs/i18n/SPEC.md §3 "Complete or nothing").
func (f *fixture) translated(languages ...string) {
	v := f.questionnaires.versions[qActive]
	v.Language = "bs"
	for _, l := range languages {
		v.Translations = append(v.Translations, questionnaire.Translation{Language: l, Title: "Placeholder title"})
	}
	f.questionnaires.versions[qActive] = v
}

// fresh seeds an in-progress response of appointment A with nothing answered, and returns its id.
func (f *fixture) fresh() string {
	id := f.seed(apptA, questionnaireresponse.StatusInProgress)
	f.answered(id, nil)
	return id
}

// The patient starts in a translation the questionnaire has, and may choose again (the original too) until
// the first answer is saved: stored, a new version each time, audited as an update beside their audited read
// of the version (docs/i18n/SPEC.md §3 "The patient's language").
func TestStart_SetsTheOriginalOrACompleteTranslationAndAudits(t *testing.T) {
	f := newFixture(now)
	f.translated("en", "de")
	id := f.fresh()

	got, err := f.svc.Start(t.Context(), me, id, "en")

	if err != nil || got.Language != "en" || got.VersionID != 4 || !got.LastUpdated.Equal(now) || got.Status != questionnaireresponse.StatusInProgress {
		t.Fatalf("Start(en) = %+v, %v; want English at version 4, in progress", got, err)
	}
	if f.repo.responses[0].Language != "en" {
		t.Errorf("stored %q, want en", f.repo.responses[0].Language)
	}
	if !slices.Equal(f.repo.locked, []string{id}) || f.repo.patient != patientA || f.repo.clinic != "" {
		t.Errorf("locked %v in patient %q, clinic %q; want the response, in the patient's scope", f.repo.locked, f.repo.patient, f.repo.clinic)
	}
	if !slices.Equal(f.questionnaires.patients, []string{patientA}) {
		t.Errorf("read the version for %v, want patient A (the patient's read: complete translations only)", f.questionnaires.patients)
	}
	if len(f.audit.events) != 2 || f.audit.events[0].EntityReference != "Questionnaire/"+qActive {
		t.Errorf("audit %+v, want the version's read, then the update", f.audit.events)
	}
	if got, want := own(f.audit.events), []auditevent.AuditEvent{event(me, auditevent.CodeUpdate, id, auditevent.OutcomeSuccess)}; !slices.Equal(got, want) {
		t.Errorf("audited %+v,\nwant %+v", got, want)
	}

	got, err = f.svc.Start(t.Context(), me, id, "bs")
	if err != nil || got.Language != "bs" || got.VersionID != 5 {
		t.Errorf("Start(bs) again before any answer = %+v, %v; want the original at version 5", got, err)
	}
}

// A language that is neither the original nor a translation the questionnaire service gives the patient (a
// tag of no translation, or one it does not know) is refused, changes nothing and is audited as a failed
// update.
func TestStart_RefusesALanguageTheQuestionnaireDoesNotHave(t *testing.T) {
	for _, language := range []string{"de", "xx", ""} {
		f := newFixture(now)
		f.translated("en")
		id := f.fresh()
		before := f.repo.responses[0]

		_, err := f.svc.Start(t.Context(), me, id, language)

		if !errors.Is(err, questionnaireresponse.ErrLanguageUnavailable) {
			t.Errorf("Start(%q) = %v, want ErrLanguageUnavailable", language, err)
		}
		if !equal(f.repo.responses[0], before) {
			t.Errorf("Start(%q) stored %+v, want it unchanged", language, f.repo.responses[0])
		}
		if got, want := own(f.audit.events), []auditevent.AuditEvent{event(me, auditevent.CodeUpdate, id, auditevent.OutcomeMinorFailure)}; !slices.Equal(got, want) {
			t.Errorf("Start(%q) audited %+v,\nwant %+v", language, got, want)
		}
	}
}

// Once an answer or a file is saved the language stays, whichever is asked: 409 language_locked, audited.
func TestStart_IsLockedOnceAnAnswerOrAFileIsSaved(t *testing.T) {
	for name, setup := range map[string]func(f *fixture, id string){
		"an answer": func(f *fixture, id string) { f.answered(id, answers) },
		"a file":    func(f *fixture, id string) { f.withFiles(id, photo("f", binA, false)) },
	} {
		for _, language := range []string{"en", "bs"} {
			f := newFixture(now)
			f.translated("en")
			id := f.fresh()
			setup(f, id)
			before := f.repo.responses[0]

			_, err := f.svc.Start(t.Context(), me, id, language)

			if !errors.Is(err, questionnaireresponse.ErrLanguageLocked) {
				t.Errorf("%s, Start(%s) = %v, want ErrLanguageLocked", name, language, err)
			}
			if !equal(f.repo.responses[0], before) {
				t.Errorf("%s, Start(%s) stored %+v, want it unchanged", name, language, f.repo.responses[0])
			}
			if got, want := own(f.audit.events), []auditevent.AuditEvent{event(me, auditevent.CodeUpdate, id, auditevent.OutcomeMinorFailure)}; !slices.Equal(got, want) {
				t.Errorf("%s, Start(%s) audited %+v,\nwant %+v", name, language, got, want)
			}
		}
	}
}

// Starting is the patient's alone, on an in-progress response, before it is due, with an active consent, as
// saving the answers is (it takes no If-Match). Every refusal changes nothing and is audited as a failed
// update, without reading the version.
func TestStart_RefusesAnyoneButThePatientAndAClosedResponse(t *testing.T) {
	for _, tt := range answerRefusals {
		if tt.version != 0 {
			continue // no If-Match: the version is not checked
		}
		at := now
		if !tt.at.IsZero() {
			at = tt.at
		}
		f := newFixture(at)
		f.translated("en")
		status := tt.status
		if status == "" {
			status = questionnaireresponse.StatusInProgress
		}
		id := f.seed(apptA, status)
		f.answered(id, nil)
		if tt.setup != nil {
			tt.setup(f)
		}
		before := f.repo.responses[0]
		if tt.unknown {
			id = unknownID
		}

		_, err := f.svc.Start(t.Context(), tt.caller, id, "en")

		if !errors.Is(err, tt.want) {
			t.Errorf("%s: = %v, want %v", tt.name, err, tt.want)
		}
		if !equal(f.repo.responses[0], before) {
			t.Errorf("%s: stored %+v, want it unchanged", tt.name, f.repo.responses[0])
		}
		entity := ""
		if tt.entity {
			entity = id
		}
		if want := []auditevent.AuditEvent{event(tt.caller, auditevent.CodeUpdate, entity, auditevent.OutcomeMinorFailure)}; !slices.Equal(f.audit.events, want) {
			t.Errorf("%s: audited %+v,\nwant %+v (and no read of the version)", tt.name, f.audit.events, want)
		}
	}
}

// A backend failure rolls everything back, the audit included.
func TestStart_FailsWithTheBackendAndKeepsNothing(t *testing.T) {
	for name, setup := range map[string]func(f *fixture){
		"the repository":     func(f *fixture) { f.repo.err = errBackend },
		"the appointments":   func(f *fixture) { f.appts.err = errBackend },
		"the questionnaires": func(f *fixture) { f.questionnaires.err = errBackend },
		"a missing version":  func(f *fixture) { delete(f.questionnaires.versions, qActive) },
	} {
		f := newFixture(now)
		f.translated("en")
		id := f.fresh()
		before := f.repo.responses[0]
		setup(f)
		if _, err := f.svc.Start(t.Context(), me, id, "en"); err == nil || errors.Is(err, questionnaireresponse.ErrNotFound) {
			t.Errorf("%s: = %v, want the failure", name, err)
		}
		if !equal(f.repo.responses[0], before) || len(f.audit.events) != 0 {
			t.Errorf("%s: kept %+v and %d events, want nothing changed (rolled back)", name, f.repo.responses[0], len(f.audit.events))
		}
	}
}
