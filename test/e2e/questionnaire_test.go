//go:build e2e

package e2e_test

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/pkg/clock"
)

// E1 of docs/questionnaire/COPY.md, the sentence the booking confirmation adds when a questionnaire is outstanding.
const questionnaireOutstanding = "There is a questionnaire to fill in before your appointment. You can find it in the Salus app."

// TestQuestionnaire_TheRoleDefaultIsFilledInAndReadByThePractitioner: the practitioner publishes a questionnaire
// of every type with one condition and makes it their role's default; a patient books, and the confirmation says a
// questionnaire is outstanding without naming it; the default request appears; the practitioner sees it exists but
// no answers; the patient saves, saves again (hiding the conditional question) and submits; the practitioner's FHIR
// search brings the completed response with its version, the answers nested under their section and the hidden
// question's answer gone.
func TestQuestionnaire_TheRoleDefaultIsFilledInAndReadByThePractitioner(t *testing.T) {
	clinic, intake := newClinicWithDefault(t)
	p := newPatientWithProfile(t)
	appt := p.bookWithQuestionnaire(clinic, intake.title, freeSlots(t, p.api, clinic.scheduleID)[0])

	r := p.waitForRequests(1)[0]
	if r.Questionnaire != intake.canonical() || r.Status != "in-progress" {
		t.Fatalf("the default request asks %q, status %q; want %q in progress", r.Questionnaire, r.Status, intake.canonical())
	}
	seen := practitionerSearch(t, clinic, appt)
	if len(seen.responses) != 1 || seen.responses[0].Status != "in-progress" || seen.responses[0].Item != nil {
		t.Fatalf("before the submit the practitioner sees %+v, want the one request in progress without answers", seen.responses)
	}

	version := p.saveAnswers(r, []answer{
		newAnswer("q-bool", map[string]any{"valueBoolean": true}),
		newAnswer("q-str", map[string]any{"valueString": "Placeholder short answer"}),
	})
	version = p.saveAnswers(r.at(version), []answer{
		newAnswer("q-bool", map[string]any{"valueBoolean": false}), // hides q-str, whose saved answer the submit removes
		newAnswer("q-str", map[string]any{"valueString": "Placeholder short answer"}),
		newAnswer("q-text", map[string]any{"valueString": "Placeholder longer answer"}),
		newAnswer("q-single", map[string]any{"valueString": "Placeholder option B"}),
		newAnswer("q-multi", map[string]any{"valueString": "Placeholder option A"}, map[string]any{"valueString": "Placeholder option C"}),
		newAnswer("q-int", map[string]any{"valueInteger": 42}),
		newAnswer("q-dec", map[string]any{"valueDecimal": 36.6}),
		newAnswer("q-date", map[string]any{"valueDate": "2026-01-15"}),
	})
	p.mustWith(ifMatch(version), http.StatusOK, http.MethodPost, "/questionnaire-responses/"+r.ID+"/submit", nil, nil)

	got := practitionerSearch(t, clinic, appt)
	if len(got.responses) != 1 || got.responses[0].Status != "completed" || got.responses[0].Authored == "" {
		t.Fatalf("after the submit the practitioner sees %+v, want the one completed response", got.responses)
	}
	if len(got.questionnaires) != 1 || got.questionnaires[0].ID != intake.id || got.questionnaires[0].URL+"|"+got.questionnaires[0].Version != got.responses[0].Questionnaire {
		t.Errorf("the search includes %+v, want the version %s the response asks", got.questionnaires, intake.id)
	}
	want := "s1[q-bool:false q-text:Placeholder longer answer] q-single:Placeholder option B " +
		"q-multi:Placeholder option A,Placeholder option C q-int:42 q-dec:36.6 q-date:2026-01-15"
	if nested := flatten(got.responses[0].Item); nested != want {
		t.Errorf("the practitioner reads the answers\n%s\nwant\n%s", nested, want)
	}
}

// TestQuestionnaire_ACancelledAppointmentDiscardsItsResponses: a patient books more than a day ahead and gets the
// role's default; the practitioner adds a second questionnaire; the patient saves part of it and cancels; the
// discard job deletes both responses, the saved one included.
func TestQuestionnaire_ACancelledAppointmentDiscardsItsResponses(t *testing.T) {
	clinic, intake := newClinicWithDefault(t)
	followUp := publish(t, clinic, "Placeholder follow-up questionnaire "+suffix(t), []map[string]any{
		{"linkId": "f-text", "text": "Placeholder question", "type": "text", "required": true},
		{"linkId": "f-bool", "text": "Placeholder yes or no question", "type": "boolean"},
	})
	p := newPatientWithProfile(t)
	appt := p.bookWithQuestionnaire(clinic, intake.title, slotAfter(t, p.api, clinic.scheduleID, 25*time.Hour))
	p.waitForRequests(1)

	var added struct{ ID string }
	clinic.practitioner.must(http.StatusCreated, http.MethodPost, "/appointments/"+appt+"/questionnaires", map[string]string{"questionnaireId": followUp.id}, &added)
	var second *response
	for _, r := range p.waitForRequests(2) {
		if r.ID == added.ID {
			second = &r
		}
	}
	if second == nil || second.Questionnaire != followUp.canonical() {
		t.Fatalf("the patient's requests do not hold the added %s", followUp.canonical())
	}
	p.saveAnswers(*second, []answer{newAnswer("f-text", map[string]any{"valueString": "Placeholder partial answer"})})

	p.must(http.StatusOK, http.MethodPost, "/appointments/"+appt+"/cancel", nil, nil)
	checkAppointmentMail(t, p.addr, cancelledSubject, clinic.name)
	eventually(t, "the questionnaire_responses_discard job deletes the cancelled appointment's responses", func() bool {
		return len(p.requests()) == 0
	})
	if got := practitionerSearch(t, clinic, appt); len(got.responses) != 0 {
		t.Errorf("the practitioner still finds %d responses of the cancelled appointment", len(got.responses))
	}
}

// questionnaireVersion is a published version as the journeys know it.
type questionnaireVersion struct {
	id, url, title string
	version        int
}

func (q questionnaireVersion) canonical() string { return q.url + "|" + strconv.Itoa(q.version) }

// newClinicWithDefault makes a bookable clinic whose practitioner publishes the intake questionnaire (every item
// type, help text and one condition, placeholder wording) and makes it their role's default.
func newClinicWithDefault(t *testing.T) (*bookableClinic, questionnaireVersion) {
	t.Helper()
	clinic := newBookableClinic(t)
	options := func(names ...string) []map[string]string {
		o := make([]map[string]string, 0, len(names))
		for _, n := range names {
			o = append(o, map[string]string{"valueString": "Placeholder option " + n})
		}
		return o
	}
	intake := publish(t, clinic, "Placeholder intake questionnaire "+suffix(t), []map[string]any{
		{"linkId": "s1", "text": "Placeholder section", "type": "group", "item": []map[string]any{
			{"linkId": "q-bool", "text": "Placeholder yes or no question", "type": "boolean", "required": true},
			{"linkId": "q-str", "text": "Placeholder follow-up question", "type": "string", "required": true,
				"enableWhen": map[string]any{"question": "q-bool", "answerBoolean": true}},
			{"linkId": "q-text", "text": "Placeholder long question", "type": "text"},
		}},
		{"linkId": "d1", "text": "Placeholder instructions", "type": "display"},
		{"linkId": "q-single", "text": "Placeholder single choice", "type": "choice", "required": true, "answerOption": options("A", "B")},
		{"linkId": "q-multi", "text": "Placeholder multiple choice", "type": "choice", "repeats": true, "answerOption": options("A", "B", "C")},
		{"linkId": "q-int", "text": "Placeholder whole number", "type": "integer"},
		{"linkId": "q-dec", "text": "Placeholder decimal number", "type": "decimal"},
		{"linkId": "q-date", "text": "Placeholder date", "type": "date", "help": "Placeholder help text"},
	})
	clinic.practitioner.must(http.StatusOK, http.MethodPut, "/practitioner-roles/"+publicRole(t, clinic.id)+"/questionnaire",
		map[string]string{"questionnaireId": intake.id}, nil)
	return clinic, intake
}

// publish creates the questionnaire as the clinic's practitioner and publishes it.
func publish(t *testing.T, clinic *bookableClinic, title string, items []map[string]any) questionnaireVersion {
	t.Helper()
	var draft struct {
		ID, URL   string
		Version   int
		VersionID int
	}
	clinic.practitioner.must(http.StatusCreated, http.MethodPost, "/questionnaires", map[string]any{"title": title, "item": items}, &draft)
	var active struct{ Status string }
	clinic.practitioner.mustWith(ifMatch(draft.VersionID), http.StatusOK, http.MethodPost, "/questionnaires/"+draft.ID+"/publish", nil, &active)
	if active.Status != "active" {
		t.Fatalf("the published questionnaire is %q, want active", active.Status)
	}
	return questionnaireVersion{id: draft.ID, url: draft.URL, title: title, version: draft.Version}
}

// bookWithQuestionnaire consents and books the slot. The confirmation must say a questionnaire is outstanding (E1)
// and must not name it.
func (p *patient) bookWithQuestionnaire(clinic *bookableClinic, title, slot string) string {
	p.t.Helper()
	p.must(http.StatusCreated, http.MethodPost, "/consents", map[string]string{"organizationId": clinic.id}, nil)
	var booked struct{ ID string }
	p.must(http.StatusCreated, http.MethodPost, "/appointments", map[string]string{"slotId": slot}, &booked)
	text := mailText(p.t, p.addr, bookedSubject)
	if !strings.Contains(text, clinic.name) || !strings.Contains(text, questionnaireOutstanding) {
		p.t.Errorf("the booking confirmation does not name the clinic or say a questionnaire is outstanding (E1)")
	}
	if strings.Contains(text, title) {
		p.t.Errorf("the booking confirmation names the questionnaire")
	}
	return booked.ID
}

// slotAfter returns a free slot of the schedule that starts at least after from now (and within three days).
func slotAfter(t *testing.T, c *api, schedule string, after time.Duration) string {
	t.Helper()
	now := clock.System{}.Now().UTC()
	q := url.Values{
		"schedule": {"Schedule/" + schedule}, "status": {"free"},
		"start": {"ge" + now.Add(after).Format(time.RFC3339), "lt" + now.Add(72*time.Hour).Format(time.RFC3339)},
	}
	var slots bundle
	c.must(http.StatusOK, http.MethodGet, "/fhir/R4/Slot?"+q.Encode(), nil, &slots)
	ids := slots.ids("Slot")
	if len(ids) == 0 {
		t.Fatalf("no free slot more than %v ahead", after)
	}
	return ids[0]
}

// response is the part of a FHIR QuestionnaireResponse the journeys read.
type response struct {
	Meta          struct{ VersionID string }
	ID            string
	Questionnaire string
	Status        string
	Authored      string
	Item          []responseItem
}

type responseItem struct {
	LinkID string
	Answer []map[string]any
	Item   []responseItem
}

// at is the response at the version a save answered.
func (r response) at(version int) response {
	r.Meta.VersionID = strconv.Itoa(version)
	return r
}

// requests is the patient's FHIR search of their own questionnaire responses.
func (p *patient) requests() []response {
	p.t.Helper()
	var found struct {
		Entry []struct{ Resource response }
	}
	p.must(http.StatusOK, http.MethodGet, "/fhir/R4/QuestionnaireResponse?patient="+p.patientID, nil, &found)
	rs := make([]response, 0, len(found.Entry))
	for _, e := range found.Entry {
		rs = append(rs, e.Resource)
	}
	return rs
}

// waitForRequests waits for the patient to have n questionnaire responses: the default request is a job.
func (p *patient) waitForRequests(n int) []response {
	p.t.Helper()
	var rs []response
	eventually(p.t, fmt.Sprintf("the patient has %d questionnaire requests", n), func() bool {
		rs = p.requests()
		return len(rs) == n
	})
	return rs
}

// answer is one question's answer values, as the JSON API takes them.
type answer struct {
	linkID string
	values []map[string]any
}

func newAnswer(linkID string, values ...map[string]any) answer { return answer{linkID, values} }

// saveAnswers replaces the response's saved answers at its version and returns the new version.
func (p *patient) saveAnswers(r response, answers []answer) int {
	p.t.Helper()
	body := make([]map[string]any, 0, len(answers))
	for _, a := range answers {
		body = append(body, map[string]any{"linkId": a.linkID, "values": a.values})
	}
	version, err := strconv.Atoi(r.Meta.VersionID)
	if err != nil {
		p.t.Fatalf("the response's versionId %q: %v", r.Meta.VersionID, err)
	}
	var saved struct{ VersionID int }
	p.mustWith(ifMatch(version), http.StatusOK, http.MethodPut, "/questionnaire-responses/"+r.ID+"/answers", map[string]any{"answers": body}, &saved)
	return saved.VersionID
}

func ifMatch(version int) http.Header {
	return http.Header{"If-Match": {fmt.Sprintf(`W/"%d"`, version)}}
}

// searched is a practitioner's search of an appointment's responses with their versions included.
type searched struct {
	responses      []response
	questionnaires []struct{ ID, URL, Version string }
}

func practitionerSearch(t *testing.T, clinic *bookableClinic, appointment string) searched {
	t.Helper()
	q := url.Values{"appointment": {appointment}, "_include": {"QuestionnaireResponse:questionnaire"}}
	var found struct {
		Entry []struct {
			Resource struct {
				ResourceType, URL, Version string
				response
			}
		}
	}
	clinic.practitioner.must(http.StatusOK, http.MethodGet, "/fhir/R4/QuestionnaireResponse?"+q.Encode(), nil, &found)
	var s searched
	for _, e := range found.Entry {
		switch e.Resource.ResourceType {
		case "QuestionnaireResponse":
			s.responses = append(s.responses, e.Resource.response)
		case "Questionnaire":
			s.questionnaires = append(s.questionnaires, struct{ ID, URL, Version string }{e.Resource.ID, e.Resource.URL, e.Resource.Version})
		}
	}
	return s
}

// flatten writes the answer tree as "linkId:value,value", a section as "linkId[…]", in order.
func flatten(items []responseItem) string {
	parts := make([]string, 0, len(items))
	for _, it := range items {
		if it.Item != nil {
			parts = append(parts, it.LinkID+"["+flatten(it.Item)+"]")
			continue
		}
		values := make([]string, 0, len(it.Answer))
		for _, a := range it.Answer {
			for _, v := range a {
				values = append(values, fmt.Sprint(v))
			}
		}
		parts = append(parts, it.LinkID+":"+strings.Join(values, ","))
	}
	return strings.Join(parts, " ")
}
