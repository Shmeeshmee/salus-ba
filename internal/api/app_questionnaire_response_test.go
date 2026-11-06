package api_test

import (
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/api"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// Synthetic, placeholder wording and answers: asserted absent from the log.
const (
	qrTitle  = "Placeholder pre-visit questionnaire"
	qrText   = "Placeholder free-text question"
	qrAnswer = "Placeholder free-text answer"
	qrBody   = `{"title":"` + qrTitle + `","item":[{"linkId":"s1","text":"Placeholder section","type":"group","item":[` +
		`{"linkId":"q1","text":"` + qrText + `","type":"string","required":true},` +
		`{"linkId":"q2","text":"Placeholder yes or no","type":"boolean"}]}]}`
	qrAnswers = `{"answers":[{"linkId":"q1","values":[{"valueString":"` + qrAnswer + `"}]},{"linkId":"q2","values":[{"valueBoolean":true}]}]}`
)

// publishQuestionnaire creates and publishes a questionnaire as c and returns the active version's id.
func publishQuestionnaire(t *testing.T, c *client, body string) string {
	t.Helper()
	rec := c.do(http.MethodPost, "/questionnaires", body)
	var q struct{ ID string }
	decode(t, rec, &q)
	c.ifMatch = `W/"1"`
	if rec := c.do(http.MethodPost, "/questionnaires/"+q.ID+"/publish", ""); rec.Code != http.StatusOK {
		t.Fatalf("publishing = %d %s", rec.Code, rec.Body)
	}
	c.ifMatch = ""
	return q.ID
}

// requested is the response a request created, as the JSON API answers it.
type requested struct{ ID, Status string }

func request(t *testing.T, m matrix, appointment, questionnaire string) requested {
	t.Helper()
	rec := m.practitioner.do(http.MethodPost, "/appointments/"+appointment+"/questionnaires", `{"questionnaireId":"`+questionnaire+`"}`)
	var r requested
	decode(t, rec, &r)
	if rec.Code != http.StatusCreated || r.Status != "in-progress" || rec.Header().Get("ETag") != `W/"1"` {
		t.Fatalf("the practitioner's request = %d %s, want 201 in progress at version 1", rec.Code, rec.Body)
	}
	return r
}

// The responses through the JSON API and FHIR with row-level security on (salus_app): the appointment's
// practitioner requests, the patient answers and submits, who reads what before and after the submit and after a
// revocation (D9), every error code, the audit of each read and of each response a search returns, and a log
// without a title, a question or an answer.
func TestNew_ServesQuestionnaireResponsesUnderTheAccessRuleAndAuditsIt(t *testing.T) {
	cfg := testConfig(t)
	owner := ownerPool(t, cfg)
	cfg.DatabaseURL = pgtest.AppURL(t, cfg.DatabaseURL)
	var log syncBuffer
	app, err := api.New(t.Context(), cfg, testLiveKit(), slog.New(slog.NewTextHandler(&log, nil)))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(app.Close)
	m := newMatrix(t, app.Handler, owner, cfg.FHIRBaseURL)
	v1 := publishQuestionnaire(t, m.practitioner, qrBody)
	v2 := publishQuestionnaire(t, m.admin, qrBody)

	first := request(t, m, m.future, v1)
	second := request(t, m, m.future, v2)
	checkRequestRefusals(t, m, v1)
	checkAnswering(t, m, first.ID)
	checkWithdrawal(t, m, first.ID, second.ID)
	third := request(t, m, m.future, v2) // requested again after the withdrawal, left in progress
	checkResponsesFHIR(t, m, first.ID, third.ID, v1)

	past := completedOnThePast(t, m, owner, v1)
	if rec := m.patient.do(http.MethodPost, "/consents/"+m.consentID+"/revoke", ""); rec.Code != http.StatusOK {
		t.Fatalf("revoking = %d %s", rec.Code, rec.Body)
	}
	checkResponsesAfterRevocation(t, m, first.ID, third.ID, past)
	checkResponseAudit(t, m, owner, first.ID)
	for _, s := range []string{qrTitle, qrText, qrAnswer, "Placeholder yes or no"} {
		if strings.Contains(log.String(), s) {
			t.Errorf("the log holds %q", s)
		}
	}
}

// checkRequestRefusals: only the appointment's practitioner requests, an active version, once, on a booked
// appointment that has not started.
func checkRequestRefusals(t *testing.T, m matrix, v1 string) {
	t.Helper()
	path, body := "/appointments/"+m.future+"/questionnaires", `{"questionnaireId":"`+v1+`"}`
	draft := m.practitioner.do(http.MethodPost, "/questionnaires", qrBody)
	var d struct{ ID string }
	decode(t, draft, &d)
	run(t, []step{
		{m.practitioner, "a second request of the questionnaire", http.MethodPost, path, "", body, `"already_requested"`, http.StatusConflict},
		{m.practitioner, "a request of a draft", http.MethodPost, path, "", `{"questionnaireId":"` + d.ID + `"}`, `"not_active"`, http.StatusConflict},
		{m.practitioner, "a request to an appointment that started", http.MethodPost, "/appointments/" + m.past + "/questionnaires", "", body, `"appointment_started"`, http.StatusConflict},
		{m.other, "another practitioner's request", http.MethodPost, path, "", body, `"not_found"`, http.StatusNotFound},
		{m.admin, "the admin's request", http.MethodPost, path, "", body, `"not_found"`, http.StatusNotFound},
		{m.patient, "the patient's request", http.MethodPost, path, "", body, `"not_found"`, http.StatusNotFound},
		{m.otherAdmin, "another clinic's request", http.MethodPost, path, "", body, `"not_found"`, http.StatusNotFound},
		{m.anonymous, "nobody", http.MethodPost, path, "", body, `"unauthenticated"`, http.StatusUnauthorized},
		{m.practitioner, "a request without a questionnaire", http.MethodPost, path, "", `{}`, `"invalid_request"`, http.StatusBadRequest},
	})
}

// checkAnswering: the patient saves (412 on a stale version, 422 with the linkIds, 400 on a value that does not
// decode), submits, and can change nothing after; staff change no answers.
func checkAnswering(t *testing.T, m matrix, id string) {
	t.Helper()
	answers, submit := "/questionnaire-responses/"+id+"/answers", "/questionnaire-responses/"+id+"/submit"
	run(t, []step{
		{m.patient, "a submit with the required question unanswered", http.MethodPost, submit, `W/"1"`, "", `"linkIds":["q1"]`, http.StatusUnprocessableEntity},
		{m.patient, "a save of an answer of the wrong type", http.MethodPut, answers, `W/"1"`, `{"answers":[{"linkId":"q2","values":[{"valueString":"x"}]}]}`,
			`{"error":"answers_invalid","linkIds":["q2"]}`, http.StatusUnprocessableEntity},
		{m.patient, "a save of an integer outside 32 bits", http.MethodPut, answers, `W/"1"`, `{"answers":[{"linkId":"q2","values":[{"valueInteger":2147483648}]}]}`,
			`"invalid_request"`, http.StatusBadRequest},
		{m.patient, "the save", http.MethodPut, answers, `W/"1"`, qrAnswers, `"versionId":2`, http.StatusOK},
		{m.patient, "a save at a stale version", http.MethodPut, answers, `W/"1"`, qrAnswers, `"version_conflict"`, http.StatusPreconditionFailed},
		{m.practitioner, "the practitioner's save", http.MethodPut, answers, `W/"2"`, qrAnswers, `"forbidden"`, http.StatusForbidden},
		{m.other, "another practitioner's save", http.MethodPut, answers, `W/"2"`, qrAnswers, `"not_found"`, http.StatusNotFound},
		{m.patient, "the submit", http.MethodPost, submit, `W/"2"`, "", `"status":"completed"`, http.StatusOK},
		{m.patient, "a second submit", http.MethodPost, submit, `W/"3"`, "", `"submitted"`, http.StatusConflict},
		{m.patient, "a save after the submit", http.MethodPut, answers, `W/"3"`, qrAnswers, `"submitted"`, http.StatusConflict},
	})
}

// checkWithdrawal: the appointment's practitioner withdraws an unsubmitted request; not a submitted one, and
// nobody else.
func checkWithdrawal(t *testing.T, m matrix, submitted, open string) {
	t.Helper()
	run(t, []step{
		{m.patient, "the patient's withdrawal", http.MethodPost, "/questionnaire-responses/" + open + "/withdraw", "", "", `"forbidden"`, http.StatusForbidden},
		{m.admin, "the admin's withdrawal", http.MethodPost, "/questionnaire-responses/" + open + "/withdraw", "", "", `"forbidden"`, http.StatusForbidden},
		{m.other, "another practitioner's withdrawal", http.MethodPost, "/questionnaire-responses/" + open + "/withdraw", "", "", `"not_found"`, http.StatusNotFound},
		{m.practitioner, "the withdrawal of a submitted one", http.MethodPost, "/questionnaire-responses/" + submitted + "/withdraw", "", "", `"submitted"`, http.StatusConflict},
		{m.practitioner, "the withdrawal", http.MethodPost, "/questionnaire-responses/" + open + "/withdraw", "", "", `"status":"stopped"`, http.StatusOK},
		{m.practitioner, "a second withdrawal", http.MethodPost, "/questionnaire-responses/" + open + "/withdraw", "", "", `"withdrawn"`, http.StatusConflict},
		{m.patient, "a save of a withdrawn one", http.MethodPut, "/questionnaire-responses/" + open + "/answers", `W/"2"`, qrAnswers, `"withdrawn"`, http.StatusConflict},
	})
}

// checkResponsesFHIR: the access matrix of the read and the search with the consent active; the patient reads the
// versions their responses answer.
func checkResponsesFHIR(t *testing.T, m matrix, submitted, open, v1 string) {
	t.Helper()
	done, inProgress := "/fhir/R4/QuestionnaireResponse/"+submitted, "/fhir/R4/QuestionnaireResponse/"+open
	nested := `"item":[{"linkId":"s1","item":[{"linkId":"q1","answer":[{"valueString":"` + qrAnswer + `"}]}`
	expect(t, "the patient's read", m.patient, done, http.StatusOK, nested, `"authored":`)
	expect(t, "the practitioner's read once submitted", m.practitioner, done, http.StatusOK, nested, `"Appointment/`+m.future+`"`)
	for label, c := range map[string]*client{"the admin's read": m.admin, "the practitioner's read before the submit": m.practitioner} {
		path := map[bool]string{true: done, false: inProgress}[c == m.admin]
		if body := expect(t, label, c, path, http.StatusOK, `"status":`); strings.Contains(body, `"item"`) {
			t.Errorf("%s = %s, want no item", label, body)
		}
	}
	expect(t, "another practitioner's read", m.other, done, http.StatusNotFound, `"not-found"`)
	expect(t, "another clinic's read", m.otherAdmin, done, http.StatusNotFound, `"not-found"`)
	expect(t, "nobody's read", m.anonymous, done, http.StatusUnauthorized, `"login"`)

	search := "/fhir/R4/QuestionnaireResponse?appointment=" + m.future + "&_include=QuestionnaireResponse:questionnaire"
	for label, want := range map[*client]int{m.practitioner: 3, m.admin: 3, m.patient: 3, m.other: 0, m.otherAdmin: 0} {
		body := expect(t, "the search", label, search, http.StatusOK)
		if n := total(t, body); n != want {
			t.Errorf("the search by %p = total %d, want %d: %s", label, n, want, body)
		}
		if want > 0 && strings.Count(body, `"resourceType":"Questionnaire",`) != 2 {
			t.Errorf("the search includes %d versions, want each of the two once: %s", strings.Count(body, `"resourceType":"Questionnaire",`), body)
		}
	}
	expect(t, "the search by status", m.patient, "/fhir/R4/QuestionnaireResponse?status=completed", http.StatusOK, `"total":1`)
	expect(t, "an unknown parameter", m.patient, "/fhir/R4/QuestionnaireResponse?subject=x", http.StatusBadRequest, `"not-supported"`)
	expect(t, "the patient's read of the version", m.patient, "/fhir/R4/Questionnaire/"+v1, http.StatusOK, `"title":"`+qrTitle+`"`)
	if body := expect(t, "the patient's search of the versions", m.patient, "/fhir/R4/Questionnaire?status=active", http.StatusOK); total(t, body) != 2 {
		t.Errorf("the patient's Questionnaire search = %s, want the two versions their responses answer", body)
	}
	expect(t, "another practitioner reads the version", m.other, "/fhir/R4/Questionnaire/"+v1, http.StatusOK)
}

// completedOnThePast stores, as the owner, a completed response of v1 on the past appointment, and returns its id.
func completedOnThePast(t *testing.T, m matrix, owner *pgxpool.Pool, v1 string) string {
	t.Helper()
	var id string
	if err := owner.QueryRow(t.Context(), `INSERT INTO questionnaire_response (organization_id, appointment_id, patient_id, practitioner_role_id,
		questionnaire_id, series_id, source, status, answers, due_at, authored_at)
		SELECT a.organization_id, a.id, a.patient_id, a.practitioner_role_id, q.id, q.series_id, 'added', 'completed',
			'[{"linkId":"q1","values":[{"valueString":"`+qrAnswer+`"}]}]', lower(a.period), lower(a.period) - interval '1 hour'
		FROM appointment a, questionnaire q WHERE a.id = $1 AND q.id = $2 RETURNING id::text`, m.past, v1).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

// checkResponsesAfterRevocation: right after the revocation commits, the appointment's practitioner keeps the
// submitted answers of the past appointment (D9), the admin its status; the future appointment's are gone for
// the clinic; the patient's next save is refused.
func checkResponsesAfterRevocation(t *testing.T, m matrix, submittedFuture, open, past string) {
	t.Helper()
	pastPath := "/fhir/R4/QuestionnaireResponse/" + past
	expect(t, "the practitioner's read of the past one", m.practitioner, pastPath, http.StatusOK, `"valueString":"`+qrAnswer+`"`)
	if body := expect(t, "the admin's read of the past one", m.admin, pastPath, http.StatusOK, `"status":"completed"`); strings.Contains(body, `"item"`) {
		t.Errorf("the admin's read after the revocation = %s, want no item", body)
	}
	expect(t, "another practitioner's read of the past one", m.other, pastPath, http.StatusNotFound)
	for _, c := range []*client{m.practitioner, m.admin} {
		expect(t, "the clinic's read of the future one", c, "/fhir/R4/QuestionnaireResponse/"+submittedFuture, http.StatusNotFound)
	}
	if body := expect(t, "the practitioner's search", m.practitioner, "/fhir/R4/QuestionnaireResponse?patient=Patient/"+m.patientID, http.StatusOK); total(t, body) != 1 ||
		!strings.Contains(body, past) {
		t.Errorf("the practitioner's search after the revocation = %s, want the past one only", body)
	}
	if body := expect(t, "the patient's search", m.patient, "/fhir/R4/QuestionnaireResponse", http.StatusOK); total(t, body) != 4 {
		t.Errorf("the patient's search after the revocation = %s, want all four of theirs", body)
	}
	run(t, []step{{m.patient, "a save right after the revocation", http.MethodPut, "/questionnaire-responses/" + open + "/answers", `W/"1"`, qrAnswers,
		`"consent_required"`, http.StatusConflict}})
}

// checkResponseAudit: every call wrote its audit events, refusals included (counted on the first response).
func checkResponseAudit(t *testing.T, m matrix, owner *pgxpool.Pool, first string) {
	t.Helper()
	count := func(where string, args ...any) int {
		t.Helper()
		var n int
		if err := owner.QueryRow(t.Context(), "SELECT count(*) FROM audit_event WHERE "+where, args...).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	ref := "QuestionnaireResponse/" + first
	for _, tt := range []struct {
		name, where string
		want        int
	}{
		{"the request", "subtype_code = 'create' AND outcome = 0 AND entity_reference = $1", 1},
		{"the saves and submits, refusals included", "subtype_code = 'update' AND entity_reference = $1 AND purpose_code = 'PATRQT'", 7},
		{"the practitioner's refused save and withdrawal", "subtype_code = 'update' AND outcome = 4 AND entity_reference = $1 AND purpose_code = 'TREAT'", 2},
		{"the reads", "subtype_code = 'read' AND outcome = 0 AND entity_reference = $1", 3},
		{"each search that returned it", "subtype_code = 'search-type' AND entity_reference = $1", 5},
	} {
		if n := count(tt.where, ref); n != tt.want {
			t.Errorf("audit events for %s: %d, want %d", tt.name, n, tt.want)
		}
	}
	if n := count("subtype_code = 'read' AND outcome = 4 AND entity_reference IS NULL AND agent_account_id = $1", m.otherAccount); n < 1 {
		t.Errorf("another practitioner's refused reads were not audited")
	}
}
