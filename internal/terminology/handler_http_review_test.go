package terminology_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/terminology"
)

const termID = "00000000-0000-4000-8000-000000000001"

// stubReviews records its last call and returns its results or its error.
type stubReviews struct {
	err         error
	caller      *terminology.Caller
	query       *terminology.QueueQuery
	call        string // the method, the term, the language, the version
	edit        *terminology.LabelsEdit
	decision    *terminology.Decision
	text        string
	page        terminology.QueuePage
	progress    []terminology.LanguageProgress
	translation terminology.Translation
	explanation terminology.Explanation
}

func (s *stubReviews) record(c terminology.Caller, method, term, language string, version int32) {
	s.caller = &c
	s.call = method + " " + term + " " + language + " " + strconv.Itoa(int(version))
}

func (s *stubReviews) Queue(_ context.Context, c terminology.Caller, q terminology.QueueQuery) (terminology.QueuePage, error) {
	s.caller, s.query = &c, &q
	return s.page, s.err
}

func (s *stubReviews) Progress(_ context.Context, c terminology.Caller) ([]terminology.LanguageProgress, error) {
	s.caller = &c
	return s.progress, s.err
}

func (s *stubReviews) EditTranslation(_ context.Context, c terminology.Caller, term, language string, e terminology.LabelsEdit, v int32) (terminology.Translation, error) {
	s.record(c, "EditTranslation", term, language, v)
	s.edit = &e
	return s.translation, s.err
}

func (s *stubReviews) ReviewTranslation(_ context.Context, c terminology.Caller, term, language string, d terminology.Decision, v int32) (terminology.Translation, error) {
	s.record(c, "ReviewTranslation", term, language, v)
	s.decision = &d
	return s.translation, s.err
}

func (s *stubReviews) WriteExplanation(_ context.Context, c terminology.Caller, term, language, text string, v int32) (terminology.Explanation, error) {
	s.record(c, "WriteExplanation", term, language, v)
	s.text = text
	return s.explanation, s.err
}

func (s *stubReviews) ReviewExplanation(_ context.Context, c terminology.Caller, term, language string, d terminology.Decision, v int32) (terminology.Explanation, error) {
	s.record(c, "ReviewExplanation", term, language, v)
	s.decision = &d
	return s.explanation, s.err
}

// serveReview sends the request to the review handler on svc as the signed-in caller (nobody when nil) and
// returns the response and everything logged.
func serveReview(t *testing.T, svc terminology.ReviewService, c *terminology.Caller, method, target, ifMatch, body string) (*httptest.ResponseRecorder, string) {
	t.Helper()
	var logs bytes.Buffer
	caller := func(context.Context) (terminology.Caller, bool, error) { return terminology.Caller{}, false, nil }
	if c != nil {
		caller = signedIn(*c)
	}
	mux := http.NewServeMux()
	terminology.NewReviewHandler(svc, caller, slog.New(slog.NewJSONHandler(&logs, nil))).Routes(mux)
	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}
	req := httptest.NewRequestWithContext(t.Context(), method, target, r)
	if ifMatch != "" {
		req.Header.Set("If-Match", ifMatch)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec, logs.String()
}

func TestHTTPQueue_ReturnsThePage(t *testing.T) {
	t.Parallel()
	at := time.Date(2026, 11, 9, 14, 0, 0, 0, time.UTC)
	svc := &stubReviews{page: terminology.QueuePage{Next: "approved~100~" + termID, Entries: []terminology.QueueEntry{
		{
			Status: "approved",
			Term:   terminology.Term{ID: termID, Category: "anatomy", WikidataID: "Q9368", ReviewStatus: "approved", Rank: 100},
			Labels: []terminology.Names{{Language: "hr", Preferred: "jetra", Synonyms: []string{}}, {Language: "la", Preferred: "hepar", Synonyms: []string{"iecur"}}},
			Translation: &terminology.Translation{TermID: termID, Language: "hr", VersionID: 3, UpdatedAt: at,
				Review: terminology.Review{Status: "approved", ReviewedBy: reviewerID, ReviewedAt: at, Note: "Fine."}},
			Explanation: &terminology.Explanation{TermID: termID, Language: "hr", Text: "Jetra je organ.", VersionID: 1, UpdatedAt: at,
				Review: terminology.Review{Status: "unreviewed"}},
		},
		{Status: "unreviewed", Term: terminology.Term{ID: "00000000-0000-4000-8000-000000000002", Category: "drug", ReviewStatus: "unreviewed"}, Labels: []terminology.Names{}},
	}}}

	rec, _ := serveReview(t, svc, &staff, http.MethodGet, "/terminology/review?language=hr&status=approved&after=x&limit=2", "", "")

	want := `{"next":"approved~100~` + termID + `","entries":[` +
		`{"translation":{"termId":"` + termID + `","language":"hr","review":{"status":"approved","reviewedBy":"` + reviewerID + `","reviewedAt":"2026-11-09T14:00:00Z","note":"Fine."},"lastUpdated":"2026-11-09T14:00:00Z","versionId":3},` +
		`"explanation":{"termId":"` + termID + `","language":"hr","text":"Jetra je organ.","review":{"status":"unreviewed"},"lastUpdated":"2026-11-09T14:00:00Z","versionId":1},` +
		`"status":"approved",` +
		`"term":{"id":"` + termID + `","category":"anatomy","reviewStatus":"approved","codes":{"wikidata":"Q9368"},` +
		`"labels":[{"language":"hr","preferred":"jetra","synonyms":[]},{"language":"la","preferred":"hepar","synonyms":["iecur"]}]}},` +
		`{"status":"unreviewed","term":{"id":"00000000-0000-4000-8000-000000000002","category":"drug","reviewStatus":"unreviewed","codes":{},"labels":[]}}]}` + "\n"
	if rec.Code != http.StatusOK || rec.Body.String() != want {
		t.Errorf("GET = %d %s\nwant 200 %s", rec.Code, rec.Body, want)
	}
	if wantQ := (terminology.QueueQuery{Language: "hr", Status: "approved", After: "x", Limit: 2}); *svc.query != wantQ || *svc.caller != staff {
		t.Errorf("asked %+v as %+v, want %+v as staff", *svc.query, *svc.caller, wantQ)
	}
	if _, _ = serveReview(t, svc, &staff, http.MethodGet, "/terminology/review?language=bs", "", ""); svc.query.Limit != 25 {
		t.Errorf("default limit = %d, want 25", svc.query.Limit)
	}
	svc.page = terminology.QueuePage{Entries: []terminology.QueueEntry{}}
	if rec, _ := serveReview(t, svc, &staff, http.MethodGet, "/terminology/review?language=bs", "", ""); rec.Body.String() != `{"entries":[]}`+"\n" {
		t.Errorf("an empty last page = %s, want no next", rec.Body)
	}
}

func TestHTTPProgress_ReturnsEveryLanguage(t *testing.T) {
	t.Parallel()
	svc := &stubReviews{progress: []terminology.LanguageProgress{{Language: "la", Terms: 4, Approved: 1}, {Language: "bs", Terms: 4, Approved: 2, NeedsChanges: 1}}}
	rec, _ := serveReview(t, svc, &staff, http.MethodGet, "/terminology/review/progress", "", "")
	want := `{"languages":[{"language":"la","terms":4,"approved":1,"needsChanges":0},{"language":"bs","terms":4,"approved":2,"needsChanges":1}]}` + "\n"
	if rec.Code != http.StatusOK || rec.Body.String() != want {
		t.Errorf("GET = %d %s\nwant 200 %s", rec.Code, rec.Body, want)
	}
}

func TestHTTPReviewWrites_DecodeCallAndEncode(t *testing.T) {
	t.Parallel()
	at := time.Date(2026, 11, 10, 9, 30, 0, 0, time.UTC)
	tr := terminology.Translation{TermID: termID, Language: "hr", VersionID: 4, UpdatedAt: at, Review: terminology.Review{Status: "unreviewed"}}
	ex := terminology.Explanation{TermID: termID, Language: "hr", Text: "Jetra.", VersionID: 2, UpdatedAt: at, Review: terminology.Review{Status: "needs_changes", ReviewedBy: reviewerID, ReviewedAt: at, Note: "Kratko."}}
	trJSON := `{"termId":"` + termID + `","language":"hr","review":{"status":"unreviewed"},"lastUpdated":"2026-11-10T09:30:00Z","versionId":4}` + "\n"
	exJSON := `{"termId":"` + termID + `","language":"hr","text":"Jetra.","review":{"status":"needs_changes","reviewedBy":"` + reviewerID + `","reviewedAt":"2026-11-10T09:30:00Z","note":"Kratko."},"lastUpdated":"2026-11-10T09:30:00Z","versionId":2}` + "\n"
	base := "/terminology/terms/" + termID
	tests := []struct {
		check            func(s *stubReviews) bool
		method, path     string
		ifMatch, body    string
		wantCall, wantOK string
	}{
		{
			method: http.MethodPut, path: base + "/translations/hr", ifMatch: `W/"3"`, body: `{"preferred":"jetra","synonyms":["džigerica"]}`,
			wantCall: "EditTranslation " + termID + " hr 3", wantOK: trJSON,
			check: func(s *stubReviews) bool {
				return reflect.DeepEqual(*s.edit, terminology.LabelsEdit{Preferred: "jetra", Synonyms: []string{"džigerica"}})
			},
		},
		{
			method: http.MethodPost, path: base + "/translations/hr/review", ifMatch: `W/"0"`, body: `{"status":"approved"}`,
			wantCall: "ReviewTranslation " + termID + " hr 0", wantOK: trJSON,
			check: func(s *stubReviews) bool { return *s.decision == terminology.Decision{Status: "approved"} },
		},
		{
			method: http.MethodPut, path: base + "/explanations/hr", ifMatch: `"1"`, body: `{"text":"Jetra."}`,
			wantCall: "WriteExplanation " + termID + " hr 1", wantOK: exJSON,
			check: func(s *stubReviews) bool { return s.text == "Jetra." },
		},
		{
			method: http.MethodPost, path: base + "/explanations/hr/review", ifMatch: `W/"1"`, body: `{"status":"needs_changes","note":"Kratko."}`,
			wantCall: "ReviewExplanation " + termID + " hr 1", wantOK: exJSON,
			check: func(s *stubReviews) bool {
				return *s.decision == terminology.Decision{Status: "needs_changes", Note: "Kratko."}
			},
		},
	}
	for _, tt := range tests {
		svc := &stubReviews{translation: tr, explanation: ex}
		rec, _ := serveReview(t, svc, &reviewer, tt.method, tt.path, tt.ifMatch, tt.body)
		if rec.Code != http.StatusOK || rec.Body.String() != tt.wantOK || svc.call != tt.wantCall || !tt.check(svc) || *svc.caller != reviewer {
			t.Errorf("%s %s = %d %s, called %q as %+v;\nwant 200 %s, %q", tt.method, tt.path, rec.Code, rec.Body, svc.call, svc.caller, tt.wantOK, tt.wantCall)
		}
		for name, bad := range map[string]struct{ ifMatch, body string }{
			"no If-Match":     {"", tt.body},
			"a bad If-Match":  {`W/"x"`, tt.body},
			"an unknown key":  {tt.ifMatch, strings.TrimSuffix(tt.body, "}") + `,"extra":1}`},
			"no JSON":         {tt.ifMatch, "nope"},
			"two JSON values": {tt.ifMatch, tt.body + tt.body},
		} {
			svc := &stubReviews{}
			if rec, _ := serveReview(t, svc, &reviewer, tt.method, tt.path, bad.ifMatch, bad.body); rec.Code != http.StatusBadRequest || svc.call != "" {
				t.Errorf("%s %s with %s = %d %s, called %q; want 400 and no call", tt.method, tt.path, name, rec.Code, rec.Body, svc.call)
			}
		}
	}
	for _, body := range []string{`{"synonyms":[]}`, `{"preferred":"jetra"}`, `{"preferred":"jetra","synonyms":null}`} {
		if rec, _ := serveReview(t, &stubReviews{}, &reviewer, http.MethodPut, base+"/translations/hr", `W/"1"`, body); rec.Code != http.StatusBadRequest {
			t.Errorf("PUT labels %s = %d, want 400", body, rec.Code)
		}
	}
	for _, body := range []string{`{}`, `{"note":"x"}`} {
		if rec, _ := serveReview(t, &stubReviews{}, &reviewer, http.MethodPost, base+"/translations/hr/review", `W/"1"`, body); rec.Code != http.StatusBadRequest {
			t.Errorf("POST review %s = %d, want 400", body, rec.Code)
		}
	}
	if rec, _ := serveReview(t, &stubReviews{}, &reviewer, http.MethodPut, base+"/explanations/hr", `W/"1"`, `{}`); rec.Code != http.StatusBadRequest {
		t.Errorf("PUT explanation {} = %d, want 400", rec.Code)
	}
}

func TestHTTPReview_MapsTheErrorsToFixedCodes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		err    error
		code   string
		status int
	}{
		{terminology.ErrForbidden, "forbidden", http.StatusForbidden},
		{terminology.ErrInvalidReview, "invalid_request", http.StatusBadRequest},
		{terminology.ErrTermNotFound, "not_found", http.StatusNotFound},
		{terminology.ErrNoTranslation, "not_found", http.StatusNotFound},
		{terminology.ErrNoExplanation, "not_found", http.StatusNotFound},
		{terminology.ErrVersionConflict, "version_conflict", http.StatusPreconditionFailed},
		{terminology.ErrNoteRequired, "note_required", http.StatusUnprocessableEntity},
		{errors.New("pq: Marko Marković"), "internal error", http.StatusInternalServerError},
	}
	requests := []struct{ method, path, body string }{
		{http.MethodGet, "/terminology/review?language=hr", ""},
		{http.MethodGet, "/terminology/review/progress", ""},
		{http.MethodPut, "/terminology/terms/" + termID + "/translations/hr", `{"preferred":"jetra","synonyms":[]}`},
		{http.MethodPost, "/terminology/terms/" + termID + "/translations/hr/review", `{"status":"approved"}`},
		{http.MethodPut, "/terminology/terms/" + termID + "/explanations/hr", `{"text":"Jetra."}`},
		{http.MethodPost, "/terminology/terms/" + termID + "/explanations/hr/review", `{"status":"approved"}`},
	}
	for _, tt := range tests {
		for _, r := range requests {
			rec, _ := serveReview(t, &stubReviews{err: tt.err}, &reviewer, r.method, r.path, `W/"1"`, r.body)
			want := `{"error":"` + tt.code + `"}` + "\n" // never the internal error's text
			if rec.Code != tt.status || rec.Body.String() != want {
				t.Errorf("%s %s with %v = %d %s, want %d %s", r.method, r.path, tt.err, rec.Code, rec.Body, tt.status, want)
			}
		}
	}
	for _, r := range requests {
		if rec, _ := serveReview(t, &stubReviews{}, nil, r.method, r.path, `W/"1"`, r.body); rec.Code != http.StatusUnauthorized {
			t.Errorf("%s %s with nobody signed in = %d, want 401", r.method, r.path, rec.Code)
		}
	}
	if rec, _ := serveReview(t, &stubReviews{}, &staff, http.MethodGet, "/terminology/review?language=hr&limit=x", "", ""); rec.Code != http.StatusBadRequest {
		t.Errorf("an unreadable limit = %d, want 400", rec.Code)
	}
}
