package questionnaireresponse_test

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/binary"
	"github.com/Shmeeshmee/salus/internal/questionnaire"
	"github.com/Shmeeshmee/salus/internal/questionnaireresponse"
)

var update = flag.Bool("update", false, "rewrite the golden files in testdata/")

// assertGolden compares got with testdata/<name>; -update rewrites the file.
func assertGolden(t *testing.T, name string, got []byte) {
	t.Helper()
	path := filepath.Join("testdata", name)
	if *update {
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Errorf("%s differs from the response:\n got: %s\nwant: %s", path, got, want)
	}
}

// stubReader answers with what it is given and keeps what it was asked.
type stubReader struct {
	err    error
	caller appointment.Caller
	id     string
	filter questionnaireresponse.Filter
	out    []questionnaireresponse.Seen
	calls  int
}

func (s *stubReader) Read(_ context.Context, c appointment.Caller, id string) (questionnaireresponse.Seen, error) {
	s.caller, s.id = c, id
	s.calls++
	if s.err != nil || len(s.out) == 0 {
		return questionnaireresponse.Seen{}, s.err
	}
	return s.out[0], nil
}

func (s *stubReader) Search(_ context.Context, c appointment.Caller, f questionnaireresponse.Filter) ([]questionnaireresponse.Seen, error) {
	s.caller, s.filter = c, f
	s.calls++
	return s.out, s.err
}

// stubVersions gives the version hQuestionnaire (version(), or v when set) and counts the reads of each id.
type stubVersions struct {
	err    error
	reads  map[string]int
	v      *questionnaire.Questionnaire
	caller questionnaire.Caller
}

func (s *stubVersions) Read(_ context.Context, c questionnaire.Caller, id string) (questionnaire.Questionnaire, error) {
	if s.reads == nil {
		s.reads = map[string]int{}
	}
	s.caller = c
	s.reads[id]++
	if s.err != nil {
		return questionnaire.Questionnaire{}, s.err
	}
	if s.v != nil {
		return *s.v, nil
	}
	return version(), nil
}

// version is a synthetic active version: a section with a text question (with help text) and a yes or no
// question, an instruction, and a multiple choice at the top level. Placeholder wording.
func version() questionnaire.Questionnaire {
	return questionnaire.Questionnaire{
		ID: hQuestionnaire, SeriesID: hSeries, OrganizationID: hClinic, Version: 2, VersionID: 4, Status: questionnaire.StatusActive,
		Title: "Placeholder intake questionnaire", LastUpdated: hNow.AddDate(0, 0, -2), Items: []questionnaire.Item{
			{LinkID: "s1", Text: "Placeholder section", Type: questionnaire.ItemGroup, Items: []questionnaire.Item{
				{LinkID: "q1", Text: "Placeholder question", Type: questionnaire.ItemString, Required: true, Help: "Placeholder help text"},
				{LinkID: "q2", Text: "Placeholder yes or no question", Type: questionnaire.ItemBoolean},
			}},
			{LinkID: "d1", Text: "Placeholder instructions", Type: questionnaire.ItemDisplay},
			{LinkID: "q3", Text: "Placeholder multiple choice", Type: questionnaire.ItemChoice, Repeats: true,
				Options: []questionnaire.Option{{Value: "Placeholder option A"}, {Value: "Placeholder option B"}}},
			{LinkID: "q4", Text: "Placeholder number", Type: questionnaire.ItemDecimal},
		},
	}
}

// Answers as saved: in the app's order, not the document's; q4 is not answered.
func savedAnswers() []questionnaireresponse.Answer {
	yes := true
	return []questionnaireresponse.Answer{
		{LinkID: "q3", Values: []questionnaireresponse.Value{{String: "Placeholder option A"}, {String: "Placeholder option B"}}},
		{LinkID: "q1", Values: []questionnaireresponse.Value{{String: placeholderAnswer}}},
		{LinkID: "q2", Values: []questionnaireresponse.Value{{Boolean: &yes}}},
	}
}

func inProgress() questionnaireresponse.Response {
	r := saved
	r.Answers = savedAnswers()
	return r
}

func completed() questionnaireresponse.Response {
	r := inProgress()
	r.Status, r.AuthoredAt, r.VersionID = questionnaireresponse.StatusCompleted, hNow, 5
	return r
}

func asCaller(c appointment.Caller) func(context.Context) (appointment.Caller, bool, error) {
	return func(context.Context) (appointment.Caller, bool, error) { return c, true, nil }
}

func nobodyCaller(context.Context) (appointment.Caller, bool, error) {
	return appointment.Caller{}, false, nil
}

const base = "https://api.example.test/fhir/R4"

func getFHIR(t *testing.T, svc *stubReader, versions *stubVersions, caller func(context.Context) (appointment.Caller, bool, error), log *bytes.Buffer, path string) *httptest.ResponseRecorder {
	t.Helper()
	mux := http.NewServeMux()
	questionnaireresponse.NewHandler(svc, versions, caller, base, slog.New(slog.NewJSONHandler(log, nil))).Routes(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil))
	return rec
}

// The patient's own in-progress response: the answers nested under the version's sections, in document
// order; no authored before the submit.
func TestHandler_ReadGivesThePatientTheirInProgressResponse(t *testing.T) {
	t.Parallel()
	svc := &stubReader{out: []questionnaireresponse.Seen{{Response: inProgress(), Level: access.Full}}}
	versions := &stubVersions{}
	rec := getFHIR(t, svc, versions, asCaller(me), &bytes.Buffer{}, "/fhir/R4/QuestionnaireResponse/"+hResponse)
	if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "application/fhir+json" ||
		rec.Header().Get("ETag") != `W/"3"` || rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("read = %d %v %s", rec.Code, rec.Header(), rec.Body)
	}
	assertGolden(t, "response_in_progress.json", rec.Body.Bytes())
	if svc.caller != me || svc.id != hResponse {
		t.Errorf("read asked %+v for %s, want the patient for the id", svc.caller, svc.id)
	}
	if want := (questionnaire.Caller{AccountID: me.AccountID, PatientID: me.PatientID}); versions.caller != want || versions.reads[hQuestionnaire] != 1 {
		t.Errorf("the version was read %v as %+v, want once as the patient %+v", versions.reads, versions.caller, want)
	}
}

// The appointment's practitioner reads a submitted response with its answers and authored; the version is read
// as staff of the clinic.
func TestHandler_ReadGivesThePractitionerASubmittedResponse(t *testing.T) {
	t.Parallel()
	versions := &stubVersions{}
	rec := getFHIR(t, &stubReader{out: []questionnaireresponse.Seen{{Response: completed(), Level: access.Full}}}, versions, asCaller(pract), &bytes.Buffer{},
		"/fhir/R4/QuestionnaireResponse/"+hResponse)
	if rec.Code != http.StatusOK {
		t.Fatalf("read = %d %s", rec.Code, rec.Body)
	}
	assertGolden(t, "response_completed.json", rec.Body.Bytes())
	if want := (questionnaire.Caller{AccountID: pract.AccountID, OrganizationID: clinicA, PractitionerID: practA1}); versions.caller != want {
		t.Errorf("the version was read as %+v, want %+v", versions.caller, want)
	}
}

// A Name view (the clinic's admin, the practitioner before the submit) carries no item, whatever the service
// left in it.
func TestHandler_ANameViewCarriesNoItem(t *testing.T) {
	t.Parallel()
	r := inProgress()
	rec := getFHIR(t, &stubReader{out: []questionnaireresponse.Seen{{Response: r, Level: access.Name}}}, &stubVersions{}, asCaller(admin), &bytes.Buffer{},
		"/fhir/R4/QuestionnaireResponse/"+hResponse)
	if rec.Code != http.StatusOK || strings.Contains(rec.Body.String(), `"item"`) || strings.Contains(rec.Body.String(), placeholderAnswer) {
		t.Fatalf("read of a Name view = %d %s, want no item", rec.Code, rec.Body)
	}
	assertGolden(t, "response_name.json", rec.Body.Bytes())
}

func TestHandler_ReadRefusals(t *testing.T) {
	t.Parallel()
	path := "/fhir/R4/QuestionnaireResponse/" + hResponse
	rec := getFHIR(t, &stubReader{err: questionnaireresponse.ErrNotFound}, &stubVersions{}, asCaller(pract2), &bytes.Buffer{}, path)
	if rec.Code != http.StatusNotFound {
		t.Errorf("read of one the caller may not see = %d, want 404", rec.Code)
	}
	assertGolden(t, "not_found.json", rec.Body.Bytes())

	svc := &stubReader{}
	rec = getFHIR(t, svc, &stubVersions{}, nobodyCaller, &bytes.Buffer{}, path)
	if rec.Code != http.StatusUnauthorized || svc.calls != 0 {
		t.Errorf("read by nobody = %d with %d calls, want 401 without a call", rec.Code, svc.calls)
	}
	assertGolden(t, "unauthenticated.json", rec.Body.Bytes())
}

// Any other error is a 500 with the fixed message; the log holds the error, never an answer.
func TestHandler_AnInternalErrorIs500AndTheLogHoldsNoAnswer(t *testing.T) {
	t.Parallel()
	path := "/fhir/R4/QuestionnaireResponse/" + hResponse
	for name, c := range map[string]struct {
		svc      *stubReader
		versions *stubVersions
		caller   func(context.Context) (appointment.Caller, bool, error)
	}{
		"the read fails":         {&stubReader{err: errors.New("the database is down")}, &stubVersions{}, asCaller(me)},
		"the version read fails": {&stubReader{out: []questionnaireresponse.Seen{{Response: inProgress(), Level: access.Full}}}, &stubVersions{err: errors.New("the database is down")}, asCaller(me)},
		"the caller fails": {&stubReader{}, &stubVersions{}, func(context.Context) (appointment.Caller, bool, error) {
			return appointment.Caller{}, false, errors.New("the database is down")
		}},
	} {
		var log bytes.Buffer
		rec := getFHIR(t, c.svc, c.versions, c.caller, &log, path)
		if rec.Code != http.StatusInternalServerError || strings.Contains(rec.Body.String(), "database") {
			t.Errorf("%s: read = %d %s, want 500 without the error", name, rec.Code, rec.Body)
		}
		if !strings.Contains(log.String(), "the database is down") || strings.Contains(log.String(), placeholderAnswer) {
			t.Errorf("%s: the log %q, want the error and no answer", name, log.String())
		}
	}
}

// A search with _include brings each version its responses use once, after the matches; without it the
// versions are still read once each (for the canonical url) and not included.
func TestHandler_SearchIncludesEachVersionOnce(t *testing.T) {
	t.Parallel()
	second := completed()
	second.ID, second.Answers = "c5000000-0000-4000-8000-0000000000c2", nil
	matches := []questionnaireresponse.Seen{{Response: completed(), Level: access.Full}, {Response: second, Level: access.Name}}

	svc, versions := &stubReader{out: matches}, &stubVersions{}
	rec := getFHIR(t, svc, versions, asCaller(pract), &bytes.Buffer{},
		"/fhir/R4/QuestionnaireResponse?appointment=Appointment/"+hAppointment+"&_include=QuestionnaireResponse:questionnaire")
	if rec.Code != http.StatusOK || rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("search = %d %s", rec.Code, rec.Body)
	}
	assertGolden(t, "search_include.json", rec.Body.Bytes())
	if svc.filter != (questionnaireresponse.Filter{AppointmentID: hAppointment}) || svc.caller != pract {
		t.Errorf("searched %+v as %+v, want the appointment as the practitioner", svc.filter, svc.caller)
	}
	if versions.reads[hQuestionnaire] != 1 || len(versions.reads) != 1 {
		t.Errorf("versions read %v, want the one version once", versions.reads)
	}

	versions = &stubVersions{}
	rec = getFHIR(t, &stubReader{out: matches}, versions, asCaller(pract), &bytes.Buffer{}, "/fhir/R4/QuestionnaireResponse?appointment="+hAppointment)
	if rec.Code != http.StatusOK || strings.Contains(rec.Body.String(), `"mode":"include"`) || versions.reads[hQuestionnaire] != 1 {
		t.Errorf("search without _include = %d, versions read %v: %s; want no include and one read", rec.Code, versions.reads, rec.Body)
	}
}

func TestHandler_SearchReadsEachParameterOnce(t *testing.T) {
	t.Parallel()
	for query, want := range map[string]questionnaireresponse.Filter{
		"":                            {},
		"patient=Patient/" + patientA: {PatientID: patientA},
		"patient=" + patientA:         {PatientID: patientA},
		"status=completed":            {Status: questionnaireresponse.StatusCompleted},
		"status=in-progress":          {Status: questionnaireresponse.StatusInProgress},
		"status=stopped":              {Status: questionnaireresponse.StatusStopped},
		"appointment=" + apptA + "&status=completed&patient=" + patientA: {AppointmentID: apptA, Status: questionnaireresponse.StatusCompleted, PatientID: patientA},
	} {
		svc := &stubReader{}
		rec := getFHIR(t, svc, &stubVersions{}, asCaller(me), &bytes.Buffer{}, "/fhir/R4/QuestionnaireResponse?"+query)
		if rec.Code != http.StatusOK || svc.filter != want {
			t.Errorf("search %q = %d with %+v, want 200 with %+v", query, rec.Code, svc.filter, want)
		}
		if !strings.Contains(rec.Body.String(), `"total":0`) {
			t.Errorf("search %q gave %s, want an empty searchset", query, rec.Body)
		}
	}
	for _, query := range []string{
		"subject=Patient/" + patientA,
		"status=amended",
		"status=completed&status=in-progress",
		"appointment=" + apptA + "&appointment=" + apptB,
		"appointment=Patient/" + apptA,
		"patient=Appointment/" + patientA,
		"_include=QuestionnaireResponse:subject",
		"_include=QuestionnaireResponse:questionnaire&_include=QuestionnaireResponse:questionnaire",
		"_count=10",
		"appointment=%zz",
	} {
		svc := &stubReader{}
		rec := getFHIR(t, svc, &stubVersions{}, asCaller(me), &bytes.Buffer{}, "/fhir/R4/QuestionnaireResponse?"+query)
		if rec.Code != http.StatusBadRequest || svc.calls != 0 {
			t.Errorf("search %q = %d with %d calls, want 400 without a call", query, rec.Code, svc.calls)
		}
		if query == "_count=10" {
			assertGolden(t, "unsupported_search.json", rec.Body.Bytes())
		}
	}
	svc := &stubReader{}
	if rec := getFHIR(t, svc, &stubVersions{}, nobodyCaller, &bytes.Buffer{}, "/fhir/R4/QuestionnaireResponse"); rec.Code != http.StatusUnauthorized || svc.calls != 0 {
		t.Errorf("search by nobody = %d with %d calls, want 401 without a call", rec.Code, svc.calls)
	}
	var log bytes.Buffer
	rec := getFHIR(t, &stubReader{err: errors.New("the database is down")}, &stubVersions{}, asCaller(me), &log, "/fhir/R4/QuestionnaireResponse")
	if rec.Code != http.StatusInternalServerError || !strings.Contains(log.String(), "the database is down") {
		t.Errorf("a failing search = %d, log %q; want 500 and the error logged", rec.Code, log.String())
	}
	rec = getFHIR(t, &stubReader{out: []questionnaireresponse.Seen{{Response: completed(), Level: access.Full}}}, &stubVersions{err: errors.New("the database is down")},
		asCaller(me), &bytes.Buffer{}, "/fhir/R4/QuestionnaireResponse")
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("a search whose version read fails = %d, want 500", rec.Code)
	}
}

// filesVersion is a synthetic version with a section holding a text question and a photo-and-document item of
// up to three files, and a video item of one at the top level. Placeholder wording.
func filesVersion() *questionnaire.Questionnaire {
	v := version()
	v.Items = []questionnaire.Item{
		{LinkID: "s1", Text: "Placeholder section", Type: questionnaire.ItemGroup, Items: []questionnaire.Item{
			{LinkID: "q1", Text: "Placeholder question", Type: questionnaire.ItemString},
			{LinkID: "f1", Text: "Placeholder file upload", Type: questionnaire.ItemAttachment, Required: true,
				Accept: []questionnaire.FileKind{questionnaire.FilePhoto, questionnaire.FileDocument}, MaxFiles: 3},
		}},
		{LinkID: "v1", Text: "Placeholder video upload", Type: questionnaire.ItemAttachment,
			Accept: []questionnaire.FileKind{questionnaire.FileVideo}, MaxFiles: 1},
	}
	return &v
}

// The synthetic files of the golden responses: a photo, a PDF and a video (their SHA-1 is of no real file).
func goldenFile(linkID, id, contentType, title string, size int64, pending bool) binary.Attachment {
	return binary.Attachment{LinkID: linkID, BinaryID: id, ContentType: contentType, Title: title,
		Hash: []byte("0123456789abcdefghij"), Size: size, Pending: pending}
}

var (
	gPhoto = goldenFile("f1", "cc000000-0000-4000-8000-0000000000c1", "image/jpeg", "Placeholder photo.jpg", 245760, false)
	gPDF   = goldenFile("f1", "cc000000-0000-4000-8000-0000000000c2", "application/pdf", "Placeholder document.pdf", 102400, true)
	gVideo = goldenFile("v1", "cc000000-0000-4000-8000-0000000000c3", "video/mp4", "Placeholder video.mp4", 52428800, false)
)

// The patient's in-progress response lists their files under the item as valueAttachment, a pending one too
// (the app shows its status), each with the download URL on the api's base, never the bucket.
func TestHandler_ReadGivesThePatientTheirFilesPendingOnesIncluded(t *testing.T) {
	t.Parallel()
	r := inProgress()
	r.Answers = []questionnaireresponse.Answer{{LinkID: "q1", Values: []questionnaireresponse.Value{{String: placeholderAnswer}}}}
	r.Attachments = []binary.Attachment{gPDF, gPhoto}
	var log bytes.Buffer
	rec := getFHIR(t, &stubReader{out: []questionnaireresponse.Seen{{Response: r, Level: access.Full}}}, &stubVersions{v: filesVersion()}, asCaller(me), &log,
		"/fhir/R4/QuestionnaireResponse/"+hResponse)
	if rec.Code != http.StatusOK {
		t.Fatalf("read = %d %s", rec.Code, rec.Body)
	}
	assertGolden(t, "response_attachments_in_progress.json", rec.Body.Bytes())
	if strings.Contains(log.String(), "Placeholder") {
		t.Errorf("the log %q holds a title or an answer", log.String())
	}
}

// The practitioner reads a submitted response with a photo and a PDF on one item and a video on another.
func TestHandler_ReadGivesThePractitionerTheFilesOfASubmittedResponse(t *testing.T) {
	t.Parallel()
	r := completed()
	r.Answers = nil
	pdf := gPDF
	pdf.Pending = false
	r.Attachments = []binary.Attachment{gPhoto, gVideo, pdf}
	rec := getFHIR(t, &stubReader{out: []questionnaireresponse.Seen{{Response: r, Level: access.Full}}}, &stubVersions{v: filesVersion()}, asCaller(pract), &bytes.Buffer{},
		"/fhir/R4/QuestionnaireResponse/"+hResponse)
	if rec.Code != http.StatusOK {
		t.Fatalf("read = %d %s", rec.Code, rec.Body)
	}
	assertGolden(t, "response_attachments_completed.json", rec.Body.Bytes())

	r.Attachments = []binary.Attachment{gPhoto}
	rec = getFHIR(t, &stubReader{out: []questionnaireresponse.Seen{{Response: r, Level: access.Name}}}, &stubVersions{v: filesVersion()}, asCaller(admin), &bytes.Buffer{},
		"/fhir/R4/QuestionnaireResponse/"+hResponse)
	if rec.Code != http.StatusOK || strings.Contains(rec.Body.String(), "valueAttachment") || strings.Contains(rec.Body.String(), "Placeholder photo") {
		t.Errorf("a Name view = %d %s, want no file", rec.Code, rec.Body)
	}
}
