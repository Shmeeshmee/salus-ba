//go:build e2e

package e2e_test

import (
	"encoding/json"
	"net/http"
	"slices"
	"testing"
)

// TestQuestionnaire_TheGynaecologyExampleKeepsOnlyThePathTaken is SAL-43's example on the real stack, through the
// calls the patient app makes: the practitioner publishes it as FHIR R4 enableWhen (a yes or no; on yes a section of
// a month and an age, with a nested question under 18; then the main questions everyone answers) and makes it the
// role's default. One patient takes the yes path at 17, one the no path, and one answers yes, then changes to no:
// the server refuses the submit while the hidden branch's answers are still saved (answer_not_enabled, naming them)
// and takes it once the app saves the path taken. The practitioner reads only the answers on each path.
func TestQuestionnaire_TheGynaecologyExampleKeepsOnlyThePathTaken(t *testing.T) {
	clinic := newBookableClinic(t)
	example := publish(t, clinic, "Placeholder branching questionnaire "+suffix(t), gynaecologyItems())
	clinic.practitioner.must(http.StatusOK, http.MethodPut, "/practitioner-roles/"+publicRole(t, clinic.id)+"/questionnaire",
		map[string]string{"questionnaireId": example.id}, nil)
	main := []answer{
		newAnswer("main-1", map[string]any{"valueString": "Placeholder answer"}),
		newAnswer("main-2", map[string]any{"valueBoolean": false}),
	}
	yes := []answer{
		newAnswer("abortion", map[string]any{"valueBoolean": true}),
		newAnswer("abortion-month", map[string]any{"valueInteger": 3}),
		newAnswer("abortion-age", map[string]any{"valueInteger": 17}),
	}
	no := []answer{newAnswer("abortion", map[string]any{"valueBoolean": false})}
	const mainRead = "main[main-1:Placeholder answer main-2:false]"

	t.Run("yes at 17 asks the month, the age and the question under 18, then the main questions", func(t *testing.T) {
		p, appt, r := bookExample(t, clinic, example)
		path := slices.Concat(yes, []answer{newAnswer("abortion-under-18", map[string]any{"valueString": "Placeholder answer"})}, main)
		p.mustWith(ifMatch(p.saveAnswers(r, path)), http.StatusOK, http.MethodPost, "/questionnaire-responses/"+r.ID+"/submit", nil, nil)
		want := "abortion:true abortion-yes[abortion-month:3 abortion-age:17 abortion-under-18:Placeholder answer] " + mainRead
		checkSubmitted(t, clinic, appt, want)
	})

	t.Run("no goes straight to the main questions", func(t *testing.T) {
		p, appt, r := bookExample(t, clinic, example)
		p.mustWith(ifMatch(p.saveAnswers(r, slices.Concat(no, main))), http.StatusOK, http.MethodPost, "/questionnaire-responses/"+r.ID+"/submit", nil, nil)
		checkSubmitted(t, clinic, appt, "abortion:false "+mainRead)
	})

	t.Run("a change from yes to no discards the branch's answers", func(t *testing.T) {
		p, appt, r := bookExample(t, clinic, example)
		version := p.saveAnswers(r, slices.Concat(yes, main))
		// The branch's answers still saved under no: the submit is refused and names them.
		stale := slices.Concat(no, yes[1:], main)
		version = p.saveAnswers(r.at(version), stale)
		status, raw := p.send(ifMatch(version), http.MethodPost, "/questionnaire-responses/"+r.ID+"/submit", nil)
		var refused struct {
			Error   string
			LinkIDs []string `json:"linkIds"`
		}
		_ = json.Unmarshal(raw, &refused)
		if status != http.StatusUnprocessableEntity || refused.Error != "answer_not_enabled" ||
			!slices.Equal(refused.LinkIDs, []string{"abortion-month", "abortion-age"}) {
			t.Fatalf("the submit with the hidden branch's answers = %d %s %v, want 422 answer_not_enabled [abortion-month abortion-age]",
				status, refused.Error, refused.LinkIDs)
		}
		// What the app saves: only the shown questions' answers.
		version = p.saveAnswers(r.at(version), slices.Concat(no, main))
		p.mustWith(ifMatch(version), http.StatusOK, http.MethodPost, "/questionnaire-responses/"+r.ID+"/submit", nil, nil)
		checkSubmitted(t, clinic, appt, "abortion:false "+mainRead)
	})
}

// gynaecologyItems is the ticket's example with placeholder wording beside its questions (test/testdata/enablewhen
// /gynaecology.json, the same items as the JSON API takes them).
func gynaecologyItems() []map[string]any {
	when := func(question, operator string, answer string, value any) []map[string]any {
		return []map[string]any{{"question": question, "operator": operator, answer: value}}
	}
	return []map[string]any{
		{"linkId": "abortion", "text": "Placeholder: have you had an abortion?", "type": "boolean"},
		{"linkId": "abortion-yes", "text": "Placeholder: about the abortion", "type": "group",
			"enableWhen": when("abortion", "=", "answerBoolean", true), "item": []map[string]any{
				{"linkId": "abortion-month", "text": "Placeholder: in which month of pregnancy?", "type": "integer"},
				{"linkId": "abortion-age", "text": "Placeholder: how old were you?", "type": "integer"},
				{"linkId": "abortion-under-18", "text": "Placeholder: an extra question under 18", "type": "text",
					"enableWhen": when("abortion-age", "<", "answerInteger", 18)},
			}},
		{"linkId": "main", "text": "Placeholder main questions", "type": "group", "item": []map[string]any{
			{"linkId": "main-1", "text": "Placeholder main question", "type": "string"},
			{"linkId": "main-2", "text": "Placeholder main yes or no", "type": "boolean"},
		}},
	}
}

// bookExample books a new patient into the clinic, whose role's default is the example, and returns the request.
func bookExample(t *testing.T, clinic *bookableClinic, example questionnaireVersion) (*patient, string, response) {
	t.Helper()
	p := newPatientWithProfile(t)
	appt := p.bookWithQuestionnaire(clinic, example.title, freeSlots(t, p.api, clinic.scheduleID)[0])
	r := p.waitForRequests(1)[0]
	if r.Questionnaire != example.canonical() {
		t.Fatalf("the default request asks %q, want %q", r.Questionnaire, example.canonical())
	}
	return p, appt, r
}

// checkSubmitted reads the appointment's one response as the practitioner and compares its answer tree.
func checkSubmitted(t *testing.T, clinic *bookableClinic, appt, want string) {
	t.Helper()
	got := practitionerSearch(t, clinic, appt)
	if len(got.responses) != 1 || got.responses[0].Status != "completed" {
		t.Fatalf("the practitioner sees %+v, want the one completed response", got.responses)
	}
	if read := flatten(got.responses[0].Item); read != want {
		t.Errorf("the practitioner reads\n%s\nwant\n%s", read, want)
	}
}
