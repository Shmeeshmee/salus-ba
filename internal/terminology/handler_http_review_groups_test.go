package terminology_test

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/terminology"
)

// stubGroupReviews is the GroupReviewService: fixed answers, what it was asked.
type stubGroupReviews struct {
	asked []string
	err   error
}

var stubName = terminology.GroupName{
	Language: "en", Text: "Head cold", VersionID: 2,
	Review: terminology.Review{Status: "needs_changes", ReviewedBy: "00000000-0000-4000-8000-0000000000e1", Note: "Placeholder note",
		ReviewedAt: time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)},
}

func (s *stubGroupReviews) List(_ context.Context, _ terminology.Caller, language string) ([]terminology.GroupReviewEntry, error) {
	s.asked = append(s.asked, "list "+language)
	return []terminology.GroupReviewEntry{{Code: "common-cold", Parent: "respiratory-infections", Name: stubName}}, s.err
}

func (s *stubGroupReviews) EditGroupName(_ context.Context, _ terminology.Caller, group, language, text string, version int32) (terminology.GroupName, error) {
	s.asked = append(s.asked, "edit "+group+" "+language+" "+text+" "+string(rune('0'+version)))
	return stubName, s.err
}

func (s *stubGroupReviews) ReviewGroupName(_ context.Context, _ terminology.Caller, group, language string, d terminology.Decision, version int32) (terminology.GroupName, error) {
	s.asked = append(s.asked, "review "+group+" "+language+" "+d.Status+" "+d.Note+" "+string(rune('0'+version)))
	return stubName, s.err
}

func serveGroupReviews(t *testing.T, svc terminology.GroupReviewService, method, target, body, ifMatch string) *httptest.ResponseRecorder {
	t.Helper()
	mux := http.NewServeMux()
	terminology.NewGroupReviewHandler(svc, signedIn(groupReviewer), slog.New(slog.DiscardHandler)).Routes(mux)
	req := httptest.NewRequestWithContext(t.Context(), method, target, strings.NewReader(body))
	if ifMatch != "" {
		req.Header.Set("If-Match", ifMatch)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

const stubNameJSON = `{"code":"common-cold","parent":"respiratory-infections","language":"en","text":"Head cold",` +
	`"review":{"status":"needs_changes","reviewedBy":"00000000-0000-4000-8000-0000000000e1","reviewedAt":"2026-11-12T10:00:00Z","note":"Placeholder note"},"versionId":2}`

func TestHTTPGroupReviews_ListsEditsAndReviews(t *testing.T) {
	t.Parallel()
	svc := &stubGroupReviews{}
	rec := serveGroupReviews(t, svc, http.MethodGet, "/terminology/review/diagnosis-groups?language=en", "", "")
	if want := `{"groups":[` + stubNameJSON + `]}` + "\n"; rec.Code != http.StatusOK || rec.Body.String() != want {
		t.Errorf("GET = %d %s, want %s", rec.Code, rec.Body, want)
	}
	rec = serveGroupReviews(t, svc, http.MethodPut, "/terminology/diagnosis-groups/common-cold/names/en", `{"text":"Head cold"}`, `W/"1"`)
	want := strings.Replace(stubNameJSON, `"parent":"respiratory-infections",`, "", 1) + "\n"
	if rec.Code != http.StatusOK || rec.Body.String() != want {
		t.Errorf("PUT = %d %s, want %s", rec.Code, rec.Body, want)
	}
	rec = serveGroupReviews(t, svc, http.MethodPost, "/terminology/diagnosis-groups/common-cold/names/en/review", `{"status":"needs_changes","note":"Placeholder note"}`, `"1"`)
	if rec.Code != http.StatusOK {
		t.Errorf("POST review = %d %s", rec.Code, rec.Body)
	}
	if got := strings.Join(svc.asked, "|"); got != "list en|edit common-cold en Head cold 1|review common-cold en needs_changes Placeholder note 1" {
		t.Errorf("asked %q", got)
	}
}

func TestHTTPGroupReviews_MapsTheErrors(t *testing.T) {
	t.Parallel()
	put := "/terminology/diagnosis-groups/common-cold/names/en"
	for name, tt := range map[string]struct {
		err            error
		body, ifMatch  string
		want           int
		method, target string
	}{
		"no If-Match":     {nil, `{"text":"x"}`, "", http.StatusBadRequest, http.MethodPut, put},
		"no text":         {nil, `{}`, `"1"`, http.StatusBadRequest, http.MethodPut, put},
		"no status":       {nil, `{"note":"x"}`, `"1"`, http.StatusBadRequest, http.MethodPost, put + "/review"},
		"not a reviewer":  {terminology.ErrForbidden, `{"text":"x"}`, `"1"`, http.StatusForbidden, http.MethodPut, put},
		"a stale version": {terminology.ErrVersionConflict, `{"text":"x"}`, `"1"`, http.StatusPreconditionFailed, http.MethodPut, put},
		"no such name":    {terminology.ErrNoGroupName, `{"text":"x"}`, `"1"`, http.StatusNotFound, http.MethodPut, put},
		"a bad language":  {terminology.ErrInvalidReview, "", "", http.StatusBadRequest, http.MethodGet, "/terminology/review/diagnosis-groups?language=la"},
	} {
		if rec := serveGroupReviews(t, &stubGroupReviews{err: tt.err}, tt.method, tt.target, tt.body, tt.ifMatch); rec.Code != tt.want {
			t.Errorf("%s = %d %s, want %d", name, rec.Code, rec.Body, tt.want)
		}
	}
}
