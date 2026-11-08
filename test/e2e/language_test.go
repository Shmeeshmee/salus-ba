//go:build e2e

package e2e_test

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"slices"
	"strings"
	"testing"
	"time"
)

// The journeys of docs/i18n (IMPLEMENTATION_PLAN.md 3.1): the language an account chose or its client matched,
// the clinic's default language, and a questionnaire answered in a translation. The German and Italian texts
// are the e-mail templates' (internal/notification/templates).
const (
	defaultLanguageURL = "https://salus.invalid/fhir/StructureDefinition/organization-default-language"
	germanBooked       = "Ihr Termin ist gebucht"
	germanOutstanding  = "Vor Ihrem Termin ist ein Fragebogen auszufüllen."
	italianBooked      = "Il suo appuntamento è prenotato"
	englishInvite      = "Your invitation to Salus"
)

// TestLanguage_AGermanPhoneRegistersAndTheConfirmationIsInGerman: a patient whose phone the app matched to German
// registers with detectedLanguage de and books; the booking confirmation arrives in German.
func TestLanguage_AGermanPhoneRegistersAndTheConfirmationIsInGerman(t *testing.T) {
	clinic := newBookableClinic(t)
	p := newPatientIn(t, "de")
	p.bookIn(clinic, freeSlots(t, p.api, clinic.scheduleID)[0], germanBooked)
}

// TestLanguage_TheClinicsDefaultLanguageAndAStaffChoice: the admin sets the clinic's default language to English
// and invites a practitioner, whose invite e-mail is in English. The admin chooses Croatian; a patient language
// is refused; the next sign-in, from a browser in English, keeps Croatian; clearing the choice follows the
// browser again.
func TestLanguage_TheClinicsDefaultLanguageAndAStaffChoice(t *testing.T) {
	waitReady(t)
	adminAddr, practitionerAddr := email(t, "admin"), email(t, "practitioner")
	clinic := field(t, salusctl(t, "create-clinic", "-name", "E2E synthetic clinic "+suffix(t), "-admin-email", adminAddr), "organization")
	admin, totp := enrolFromInvite(t, adminAddr)

	setDefaultLanguage(t, admin, clinic, "en")
	admin.must(http.StatusCreated, http.MethodPost, "/invites", map[string]string{
		"email": practitionerAddr, "role": "practitioner", "givenName": "Cy", "familyName": "Synthetic",
	}, nil)
	if text := mailText(t, practitionerAddr, englishInvite); !inviteLink.MatchString(text) || !strings.Contains(text, "You are invited to join") {
		t.Errorf("the invite e-mail of a clinic whose default language is English is not the English invite")
	}

	var me struct{ Language string }
	admin.must(http.StatusOK, http.MethodPatch, "/auth/me", map[string]any{"language": "hr"}, &me)
	if me.Language != "hr" {
		t.Fatalf("PATCH /auth/me hr answered language %q, want hr", me.Language)
	}
	if got, code := admin.refusal(http.MethodPatch, "/auth/me", map[string]any{"language": "de"}); got != http.StatusUnprocessableEntity || code != "language_unavailable" {
		t.Errorf("a staff account choosing de = %d %q, want 422 language_unavailable", got, code)
	}

	other := newAPI(t) // another browser, in English
	var l login
	other.must(http.StatusOK, http.MethodPost, "/auth/login", map[string]string{"email": adminAddr, "password": password, "detectedLanguage": "en"}, &l)
	other.must(http.StatusOK, http.MethodPost, "/auth/mfa/verify", map[string]string{"challenge": l.Challenge, "code": totp.code()}, &l)
	other.bearer = l.Token
	other.must(http.StatusOK, http.MethodGet, "/auth/me", nil, &me)
	if me.Language != "hr" {
		t.Fatalf("GET /auth/me after signing in from an English browser = %q, want the chosen hr", me.Language)
	}

	other.must(http.StatusOK, http.MethodPatch, "/auth/me", map[string]any{"language": nil}, nil)
	me.Language = ""
	admin.must(http.StatusOK, http.MethodGet, "/auth/me", nil, &me)
	if me.Language != "" {
		t.Errorf("GET /auth/me after clearing the choice = %q, want none (the browser's language)", me.Language)
	}
}

// TestLanguage_TheWebAppFollowsTheBrowserThenTheSavedChoice runs the web journey (test/e2e/web/language.spec.ts): the
// sign-in screen of a browser in Serbian Cyrillic is in Serbian Cyrillic; the admin picks Hrvatski, signs out and
// signs in from another browser in English, where the app is in Croatian, and goes back to the browser's language.
// Then the account holds no chosen language.
func TestLanguage_TheWebAppFollowsTheBrowserThenTheSavedChoice(t *testing.T) {
	waitReady(t)
	adminAddr := email(t, "admin")
	salusctl(t, "create-clinic", "-name", "E2E synthetic clinic "+suffix(t), "-admin-email", adminAddr)
	admin, totp := enrolFromInvite(t, adminAddr)

	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "web/node_modules/.bin/playwright", "test", "--config", "web/playwright.config.ts", "language.spec.ts")
	cmd.Env = append(os.Environ(),
		"E2E_PRACTITIONER_EMAIL="+adminAddr,
		"E2E_PASSWORD="+password,
		"E2E_TOTP_SECRET="+hex.EncodeToString(totp.secret),
		fmt.Sprintf("E2E_TOTP_LAST_STEP=%d", totp.last),
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("the web journey failed: %v\nPlaywright:\n%s", err, out)
	}

	var me struct{ Language string }
	admin.must(http.StatusOK, http.MethodGet, "/auth/me", nil, &me)
	if me.Language != "" {
		t.Errorf("GET /auth/me after the web journey = %q, want no chosen language", me.Language)
	}
}

// TestLanguage_AGermanPatientAnswersTheEnglishTranslation: the practitioner writes a questionnaire in Bosnian with
// an English translation, whose publish is refused while half done and accepted once finished, and makes it the
// role's default. A German patient books; the questionnaire has Bosnian and English but not German, which the
// start refuses; the patient starts in English, answers and submits, and cannot change the language after the
// first answer. The practitioner reads the response in English.
func TestLanguage_AGermanPatientAnswersTheEnglishTranslation(t *testing.T) {
	clinic := newBookableClinic(t)
	translated := publishTranslated(t, clinic)
	p := newPatientIn(t, "de")
	appt := p.bookIn(clinic, freeSlots(t, p.api, clinic.scheduleID)[0], germanBooked, germanOutstanding)
	r := p.waitForRequests(1)[0]
	if r.Questionnaire != translated.canonical() {
		t.Fatalf("the default request asks %q, want %q", r.Questionnaire, translated.canonical())
	}

	if original, translations := p.questionnaireLanguages(r.Questionnaire); original != "bs" || !slices.Equal(translations, []string{"en"}) {
		t.Fatalf("the patient reads a questionnaire in %q translated into %v, want bs translated into en", original, translations)
	}
	if got, code := p.refusal(http.MethodPost, "/questionnaire-responses/"+r.ID+"/start", map[string]string{"language": "de"}); got != http.StatusUnprocessableEntity || code != "language_unavailable" {
		t.Errorf("starting in German = %d %q, want 422 language_unavailable", got, code)
	}
	version := p.start(r.ID, "en")
	version = p.saveAnswers(r.at(version), []answer{
		newAnswer("t-name", map[string]any{"valueString": "Placeholder answer"}),
		newAnswer("t-choice", map[string]any{"valueString": "Placeholder Bosnian option B"}),
	})
	if got, code := p.refusal(http.MethodPost, "/questionnaire-responses/"+r.ID+"/start", map[string]string{"language": "bs"}); got != http.StatusConflict || code != "language_locked" {
		t.Errorf("changing the language after the first answer = %d %q, want 409 language_locked", got, code)
	}
	p.mustWith(ifMatch(version), http.StatusOK, http.MethodPost, "/questionnaire-responses/"+r.ID+"/submit", nil, nil)

	got := practitionerSearch(t, clinic, appt)
	if len(got.responses) != 1 || got.responses[0].Status != "completed" || got.responses[0].Language != "en" {
		t.Fatalf("the practitioner reads %+v, want the one completed response answered in en", got.responses)
	}
}

// TestLanguage_AnItalianPatientFillsInABosnianOnlyQuestionnaire: the role's default questionnaire is written in
// Bosnian, the clinic's default language, and has no translation. An Italian patient gets the confirmation in
// Italian; Italian and English are refused; the patient answers in Bosnian, which the practitioner reads.
func TestLanguage_AnItalianPatientFillsInABosnianOnlyQuestionnaire(t *testing.T) {
	clinic, intake := newClinicWithDefault(t)
	p := newPatientIn(t, "it")
	appt := p.bookIn(clinic, freeSlots(t, p.api, clinic.scheduleID)[0], italianBooked)
	r := p.waitForRequests(1)[0]

	if original, translations := p.questionnaireLanguages(intake.canonical()); original != "bs" || len(translations) != 0 {
		t.Fatalf("the patient reads a questionnaire in %q translated into %v, want Bosnian only", original, translations)
	}
	for _, language := range []string{"it", "en"} {
		if got, code := p.refusal(http.MethodPost, "/questionnaire-responses/"+r.ID+"/start", map[string]string{"language": language}); got != http.StatusUnprocessableEntity || code != "language_unavailable" {
			t.Errorf("starting in %s = %d %q, want 422 language_unavailable", language, got, code)
		}
	}
	version := p.start(r.ID, "bs")
	version = p.saveAnswers(r.at(version), []answer{
		newAnswer("q-bool", map[string]any{"valueBoolean": false}),
		newAnswer("q-single", map[string]any{"valueString": "Placeholder option A"}),
	})
	p.mustWith(ifMatch(version), http.StatusOK, http.MethodPost, "/questionnaire-responses/"+r.ID+"/submit", nil, nil)

	got := practitionerSearch(t, clinic, appt)
	if len(got.responses) != 1 || got.responses[0].Status != "completed" || got.responses[0].Language != "bs" {
		t.Fatalf("the practitioner reads %+v, want the one completed response answered in bs", got.responses)
	}
}

// setDefaultLanguage sets the clinic's default language through the FHIR update of its Organization.
func setDefaultLanguage(t *testing.T, admin *api, clinic, language string) {
	t.Helper()
	var org map[string]any
	admin.must(http.StatusOK, http.MethodGet, "/fhir/R4/Organization/"+clinic, nil, &org)
	meta, _ := org["meta"].(map[string]any)
	version, _ := meta["versionId"].(string)
	extensions := []any{map[string]any{"url": defaultLanguageURL, "valueCode": language}}
	old, _ := org["extension"].([]any)
	for _, e := range old {
		if m, _ := e.(map[string]any); m["url"] != defaultLanguageURL {
			extensions = append(extensions, e)
		}
	}
	org["extension"] = extensions
	var updated map[string]any
	admin.mustWith(http.Header{"If-Match": {`W/"` + version + `"`}}, http.StatusOK, http.MethodPut, "/fhir/R4/Organization/"+clinic, org, &updated)
	if got, _ := updated["extension"].([]any); !slices.ContainsFunc(got, func(e any) bool {
		m, _ := e.(map[string]any)
		return m["url"] == defaultLanguageURL && m["valueCode"] == language
	}) {
		t.Fatalf("the updated Organization does not carry the default language %s", language)
	}
}

// publishTranslated creates a questionnaire in Bosnian with a half-done English translation, which publishing
// refuses with the items it lacks; finishes the translation, publishes it and makes it the role's default.
func publishTranslated(t *testing.T, clinic *bookableClinic) questionnaireVersion {
	t.Helper()
	title := "Placeholder Bosnian questionnaire " + suffix(t)
	items := []map[string]any{
		{"linkId": "t-name", "text": "Placeholder Bosnian question", "type": "string", "required": true, "help": "Placeholder Bosnian help"},
		{"linkId": "t-choice", "text": "Placeholder Bosnian choice", "type": "choice", "answerOption": []map[string]string{
			{"valueString": "Placeholder Bosnian option A"}, {"valueString": "Placeholder Bosnian option B"},
		}},
	}
	half := map[string]any{"language": "en", "title": "Placeholder English questionnaire", "items": map[string]any{
		"t-name": map[string]any{"text": "Placeholder English question", "help": "Placeholder English help"},
	}}
	var draft struct {
		ID, URL, Language  string
		Version, VersionID int
	}
	pr := clinic.practitioner
	pr.must(http.StatusCreated, http.MethodPost, "/questionnaires", map[string]any{
		"title": title, "item": items, "language": "bs", "translations": []any{half},
	}, &draft)
	if draft.Language != "bs" {
		t.Fatalf("the new questionnaire is in %q, want bs", draft.Language)
	}

	status, raw := pr.send(ifMatch(draft.VersionID), http.MethodPost, "/questionnaires/"+draft.ID+"/publish", nil)
	var refused struct {
		Error, Language string
		LinkIDs         []string `json:"linkIds"`
	}
	_ = json.Unmarshal(raw, &refused)
	if status != http.StatusUnprocessableEntity || refused.Error != "translation_incomplete" || refused.Language != "en" || !slices.Equal(refused.LinkIDs, []string{"t-choice"}) {
		t.Fatalf("publishing a half-done translation = %d %+v, want 422 translation_incomplete in en for t-choice", status, refused)
	}

	half["items"].(map[string]any)["t-choice"] = map[string]any{
		"text": "Placeholder English choice", "options": []string{"Placeholder English option A", "Placeholder English option B"},
	}
	pr.mustWith(ifMatch(draft.VersionID), http.StatusOK, http.MethodPut, "/questionnaires/"+draft.ID, map[string]any{
		"title": title, "item": items, "translations": []any{half},
	}, &draft)
	var active struct{ Status string }
	pr.mustWith(ifMatch(draft.VersionID), http.StatusOK, http.MethodPost, "/questionnaires/"+draft.ID+"/publish", nil, &active)
	if active.Status != "active" {
		t.Fatalf("the published questionnaire is %q, want active", active.Status)
	}
	pr.must(http.StatusOK, http.MethodPut, "/practitioner-roles/"+publicRole(t, clinic.id)+"/questionnaire",
		map[string]string{"questionnaireId": draft.ID}, nil)
	return questionnaireVersion{id: draft.ID, url: draft.URL, title: title, version: draft.Version}
}

// bookIn consents and books the slot; the confirmation must arrive with the subject and hold the clinic's name
// and every text of want.
func (p *patient) bookIn(clinic *bookableClinic, slot, subject string, want ...string) string {
	p.t.Helper()
	p.must(http.StatusCreated, http.MethodPost, "/consents", map[string]string{"organizationId": clinic.id}, nil)
	var booked struct{ ID string }
	p.must(http.StatusCreated, http.MethodPost, "/appointments", map[string]string{"slotId": slot}, &booked)
	text := mailText(p.t, p.addr, subject)
	for _, w := range append([]string{clinic.name, "/appointments"}, want...) {
		if !strings.Contains(text, w) {
			p.t.Errorf("the booking confirmation %q does not hold %q", subject, w)
		}
	}
	return booked.ID
}

// start starts the response in the language and returns the version it answers.
func (p *patient) start(id, language string) int {
	p.t.Helper()
	var started struct {
		Language  string
		VersionID int
	}
	p.must(http.StatusOK, http.MethodPost, "/questionnaire-responses/"+id+"/start", map[string]string{"language": language}, &started)
	if started.Language != language {
		p.t.Fatalf("the started response is in %q, want %s", started.Language, language)
	}
	return started.VersionID
}

// questionnaireLanguages reads the version as the app does (the patient's response search with the versions
// included) and returns its original language and the languages of its title's translations.
func (p *patient) questionnaireLanguages(canonical string) (string, []string) {
	p.t.Helper()
	var found struct {
		Entry []struct {
			Resource struct {
				Title *struct {
					Extension []struct {
						URL       string
						Extension []struct{ URL, ValueCode string }
					}
				} `json:"_title"`
				ResourceType, URL, Version, Language string
			}
		}
	}
	p.must(http.StatusOK, http.MethodGet, "/fhir/R4/QuestionnaireResponse?patient=Patient/"+p.patientID+"&_include=QuestionnaireResponse:questionnaire", nil, &found)
	for _, e := range found.Entry {
		q := e.Resource
		if q.ResourceType != "Questionnaire" || q.URL+"|"+q.Version != canonical {
			continue
		}
		var translations []string
		if q.Title != nil {
			for _, ext := range q.Title.Extension {
				for _, part := range ext.Extension {
					if ext.URL == "http://hl7.org/fhir/StructureDefinition/translation" && part.URL == "lang" {
						translations = append(translations, part.ValueCode)
					}
				}
			}
		}
		return q.Language, translations
	}
	p.t.Fatalf("the patient's search does not include the questionnaire %s", canonical)
	return "", nil
}
