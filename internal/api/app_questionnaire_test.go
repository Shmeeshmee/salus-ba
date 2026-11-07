package api_test

import (
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/api"
)

// Synthetic, placeholder wording: asserted absent from the log.
const (
	qTitle = "Placeholder questionnaire title"
	qText  = "Placeholder question text"
	qDraft = `{"title":"` + qTitle + `","item":[{"linkId":"q1","text":"` + qText + `","type":"boolean","required":true,"help":"Placeholder help"}]}`
)

// step is one JSON API call and what it answers.
type step struct {
	c                                 *client
	name, method, path, ifMatch, body string
	want                              string
	status                            int
}

func run(t *testing.T, steps []step) {
	t.Helper()
	for _, s := range steps {
		s.c.ifMatch = s.ifMatch
		if rec := s.c.do(s.method, s.path, s.body); rec.Code != s.status || !strings.Contains(rec.Body.String(), s.want) {
			t.Errorf("%s: %s %s = %d %s, want %d with %s", s.name, s.method, s.path, rec.Code, rec.Body, s.status, s.want)
		}
		s.c.ifMatch = ""
	}
}

// The library through the JSON API and FHIR with row-level security on (salus_app): the clinic's practitioner
// and admin build, publish, revise and retire, and set the practitioner's role default; a patient and another
// clinic's admin see nothing; every call is audited, refusals included; the log holds no title or text.
func TestNew_ServesTheQuestionnaireLibraryToStaffOfTheClinicAndAuditsIt(t *testing.T) {
	cfg := testConfig(t)
	var log syncBuffer
	app, err := api.New(t.Context(), cfg, testLiveKit(), slog.New(slog.NewTextHandler(&log, nil)))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(app.Close)
	owner := ownerPool(t, cfg)
	d := newDirectory(t, app.Handler, owner)
	other := otherClinicAdmin(t, app.Handler, owner)
	count := openDB(t, cfg)
	audits := []struct {
		name, where string
		want, base  int
	}{
		{"the creates", "subtype_code = 'create' AND entity_reference LIKE 'Questionnaire/%' AND outcome = 0", 2, 0}, // the draft, the revise
		{"the updates of the version", "subtype_code = 'update' AND entity_reference = 'Questionnaire/' || $1 AND outcome = 0", 3, 0},
		{"the refused updates", "subtype_code = 'update' AND entity_reference = 'Questionnaire/' || $1 AND outcome = 4", 3, 0},
		{"the refused revises", "subtype_code = 'create' AND entity_reference = 'Questionnaire/' || $1 AND outcome = 4", 2, 0},
		// The patient's 5 refused calls, and before each the caller's refused lookup of their profile (they have none).
		{"the refusals without a clinic", "entity_reference IS NULL AND organization_id IS NULL AND outcome = 4 AND purpose_code = 'PATRQT'", 10, 0},
		{"another clinic's refusals", "entity_reference IS NULL AND organization_id = 'b0000000-0000-4000-8000-00000000000b' AND outcome = 4", 4, 0},
		{"the reads of the version", "subtype_code = 'read' AND entity_reference = 'Questionnaire/' || $1 AND outcome = 0", 2, 0},
		{"each version a search returned", "subtype_code = 'search-type' AND entity_reference LIKE 'Questionnaire/%'", 3, 0},
		{"the empty search", "subtype_code = 'search-type' AND entity_reference IS NULL AND outcome = 0", 1, 0},
		{"the role's default", "entity_reference = 'PractitionerRole/' || $2", 5, 0},
	}
	audited := func(where, v1 string) int {
		return count("SELECT count(*) FROM audit_event WHERE "+where+" AND $1 <> '' AND $2 <> ''", v1, d.roleID)
	}
	for i := range audits {
		audits[i].base = audited(audits[i].where, "none")
	}

	rec := d.practitioner.do(http.MethodPost, "/questionnaires", qDraft)
	var v1 struct{ ID, URL, Status string }
	decode(t, rec, &v1)
	if rec.Code != http.StatusCreated || rec.Header().Get("ETag") != `W/"1"` || v1.Status != "draft" ||
		!strings.HasPrefix(v1.URL, "https://salus.invalid/fhir/Questionnaire/") {
		t.Fatalf("POST /questionnaires as the practitioner = %d %s, want 201 a draft at version 1", rec.Code, rec.Body)
	}
	q := "/questionnaires/" + v1.ID
	run(t, []step{
		{d.admin, "the admin edits the draft", http.MethodPut, q, `W/"1"`, qDraft, `"versionId":2`, http.StatusOK},
		{d.practitioner, "an edit at a stale version", http.MethodPut, q, `W/"1"`, qDraft, `"version_conflict"`, http.StatusPreconditionFailed},
		{d.practitioner, "an attachment item without its settings", http.MethodPut, q, `W/"2"`,
			`{"title":"x","item":[{"linkId":"a1","text":"x","type":"attachment"}]}`, `"linkIds":["a1"]`, http.StatusUnprocessableEntity},
		{d.practitioner, "a publish without If-Match", http.MethodPost, q + "/publish", "", "", `"invalid_request"`, http.StatusBadRequest},
		{d.practitioner, "a revise of a draft", http.MethodPost, q + "/revise", "", "", `"not_active"`, http.StatusConflict},
		{d.practitioner, "the publish", http.MethodPost, q + "/publish", `W/"2"`, "", `"status":"active"`, http.StatusOK},
		{d.practitioner, "an edit of an active version", http.MethodPut, q, `W/"3"`, qDraft, `"not_draft"`, http.StatusConflict},
		{d.practitioner, "a delete of an active version", http.MethodDelete, q, "", "", `"not_draft"`, http.StatusConflict},
		{d.patient, "a patient's create", http.MethodPost, "/questionnaires", "", qDraft, `"forbidden"`, http.StatusForbidden},
		{d.patient, "a patient's publish", http.MethodPost, q + "/publish", `W/"3"`, "", `"forbidden"`, http.StatusForbidden},
		{other, "another clinic's edit", http.MethodPut, q, `W/"3"`, qDraft, `"not_found"`, http.StatusNotFound},
		{other, "another clinic's retire", http.MethodPost, q + "/retire", "", "", `"not_found"`, http.StatusNotFound},
		{&client{t: t, h: app.Handler}, "nobody", http.MethodPost, "/questionnaires", "", qDraft, `"unauthenticated"`, http.StatusUnauthorized},
	})

	v2 := checkRevise(t, d, q)
	checkDefault(t, d, other, v1.ID, v1.URL)
	checkQuestionnaireFHIR(t, d, other, cfg.FHIRBaseURL, v1.ID, v2, v1.URL)
	run(t, []step{
		{d.admin, "the admin retires the active version", http.MethodPost, q + "/retire", "", "", `"status":"retired"`, http.StatusOK},
		{d.practitioner, "a default to a retired questionnaire", http.MethodPut, "/practitioner-roles/" + d.roleID + "/questionnaire", "",
			`{"questionnaireId":"` + v1.ID + `"}`, `"not_active"`, http.StatusConflict},
		{d.practitioner, "the practitioner discards the new draft", http.MethodDelete, "/questionnaires/" + v2, "", "", "", http.StatusNoContent},
	})

	for _, tt := range audits {
		if n := audited(tt.where, v1.ID) - tt.base; n != tt.want {
			t.Errorf("%s: %d audit events, want %d", tt.name, n, tt.want)
		}
	}
	if n := count("SELECT count(*) FROM questionnaire WHERE id::text = $1 AND status = 'retired' AND items->0->>'linkId' = 'q1'", v1.ID); n != 1 {
		t.Errorf("%d retired rows with the items for version 1, want 1", n)
	}
	for _, s := range []string{qTitle, qText, "Placeholder help"} {
		if strings.Contains(log.String(), s) {
			t.Errorf("the log holds %q: %s", s, log.String())
		}
	}
}

// checkRevise: "Edit as new version" makes version 2 a draft with the same linkIds; a second one is
// draft_exists. It returns the new draft's id.
func checkRevise(t *testing.T, d directory, q string) string {
	t.Helper()
	rec := d.practitioner.do(http.MethodPost, q+"/revise", "")
	var v2 struct {
		ID, Status string
		Item       []struct{ LinkID string }
		Version    int
	}
	decode(t, rec, &v2)
	if rec.Code != http.StatusCreated || v2.Status != "draft" || v2.Version != 2 || len(v2.Item) != 1 || v2.Item[0].LinkID != "q1" {
		t.Fatalf("POST %s/revise = %d %s, want 201 version 2, a draft with linkId q1", q, rec.Code, rec.Body)
	}
	run(t, []step{{d.admin, "a revise with a draft", http.MethodPost, q + "/revise", "", "", `"draft_exists"`, http.StatusConflict}})
	return v2.ID
}

// checkDefault: the practitioner sets and clears their role's default, the admin reads it, a patient and
// another clinic cannot.
func checkDefault(t *testing.T, d directory, other *client, v1, url string) {
	t.Helper()
	path := "/practitioner-roles/" + d.roleID + "/questionnaire"
	run(t, []step{
		{d.practitioner, "the practitioner reads their role's default before one is set", http.MethodGet, path, "", "", `"versionId":0}`, http.StatusOK},
		{d.practitioner, "the practitioner sets it", http.MethodPut, path, "", `{"questionnaireId":"` + v1 + `"}`, `"url":"` + url + `"`, http.StatusOK},
		{d.admin, "the admin reads it", http.MethodGet, path, "", "", `"url":"` + url + `"`, http.StatusOK},
		{d.patient, "a patient's read", http.MethodGet, path, "", "", `"forbidden"`, http.StatusForbidden},
		{other, "another clinic's read", http.MethodGet, path, "", "", `"not_found"`, http.StatusNotFound},
		{d.practitioner, "the practitioner clears it", http.MethodDelete, path, "", "", "", http.StatusNoContent},
	})
}

// checkQuestionnaireFHIR: staff of the clinic read and search the versions over FHIR; a patient and another
// clinic do not; an unknown parameter is 400.
func checkQuestionnaireFHIR(t *testing.T, d directory, other *client, base, v1, v2, url string) {
	t.Helper()
	read1, read2 := "/fhir/R4/Questionnaire/"+v1, "/fhir/R4/Questionnaire/"+v2
	expect(t, "the admin's read", d.admin, read1, http.StatusOK, `"status":"active"`, `"version":"1"`, `"url":"`+url+`"`, `"linkId":"q1.help"`)
	expect(t, "the practitioner's read", d.practitioner, read1, http.StatusOK, `"versionId":"3"`)
	expect(t, "the practitioner's read of the draft", d.practitioner, read2, http.StatusOK, `"status":"draft"`, `"version":"2"`)
	expect(t, "a patient's read", d.patient, read1, http.StatusNotFound, `"not-found"`)
	expect(t, "another clinic's read", other, read1, http.StatusNotFound, `"not-found"`)
	expect(t, "nobody's read", &client{t: t, h: d.admin.h}, read1, http.StatusUnauthorized, `"login"`)

	series := "/fhir/R4/Questionnaire?url=" + url
	if n := total(t, expect(t, "the admin's search of the series", d.admin, series, http.StatusOK, `"fullUrl":"`+base+"/Questionnaire/"+v2+`"`)); n != 2 {
		t.Errorf("the admin's search of the series: total %d, want 2", n)
	}
	if n := total(t, expect(t, "the practitioner's search of its active version", d.practitioner, series+"&status=active", http.StatusOK)); n != 1 {
		t.Errorf("the practitioner's search of the active version: total %d, want 1", n)
	}
	if n := total(t, expect(t, "another clinic's search", other, series, http.StatusOK)); n != 0 {
		t.Errorf("another clinic's search: total %d, want 0", n)
	}
	expect(t, "a patient's search", d.patient, series, http.StatusForbidden, `"forbidden"`)
	expect(t, "an unknown parameter", d.admin, "/fhir/R4/Questionnaire?name=x", http.StatusBadRequest, `"not-supported"`)
}

// otherClinicAdmin is the signed-in admin of another clinic.
func otherClinicAdmin(t *testing.T, h http.Handler, owner *pgxpool.Pool) *client {
	t.Helper()
	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := owner.Exec(t.Context(), q, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec("INSERT INTO organization (id, name) VALUES ($1, 'Other synthetic clinic')", otherClinic)
	exec("INSERT INTO invite (organization_id, expires_at, role, email) VALUES ($1, now() + interval '72 hours', 'admin', 'fay@example.test')", otherClinic)
	c := &client{t: t, h: h}
	c.bearer = accept(t, c, issueToken(t, exec, "fay@example.test"))
	return c
}
