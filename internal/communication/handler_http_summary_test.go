package communication_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/communication"
)

// The SummaryResponse schema, not cached; the language is optional, an unknown field refused.
func TestHTTP_SummaryAnswersTheSummaryInTheAskedLanguage(t *testing.T) {
	t.Parallel()
	svc := &stubChats{summary: communication.Summary{Summary: "Placeholder summary", GeneratedAt: start, UpToSeq: 60, AIGenerated: true}}
	srv := newServer(t, svc, newFakeLive(), true)
	code, body, h := do(t, srv, http.MethodPost, convPath+"/summary", `{"lang":"hr"}`, nil)
	want := `{"generated_at":"2026-11-12T10:00:00Z","summary":"Placeholder summary","up_to_seq":60,"ai_generated":true}`
	if code != http.StatusOK || body != want || h.Get("Cache-Control") != "no-store" {
		t.Errorf("summary = %d %s %v, want %s", code, body, h, want)
	}
	if code, _, _ := do(t, srv, http.MethodPost, convPath+"/summary", "", nil); code != http.StatusOK {
		t.Errorf("summary without a body = %d", code)
	}
	if code, _, _ := do(t, srv, http.MethodPost, convPath+"/summary", `{"lang":"hr","extra":1}`, nil); code != http.StatusBadRequest {
		t.Errorf("unknown field = %d, want 400", code)
	}
	if got := strings.Join(svc.calls(), "|"); got != "summary hr|summary " {
		t.Errorf("calls = %q", got)
	}
	svc.err = communication.ErrForbidden
	if code, body, _ := do(t, srv, http.MethodPost, convPath+"/summary", "{}", nil); code != http.StatusForbidden || body != `{"error":"forbidden"}` {
		t.Errorf("refused = %d %s", code, body)
	}
}
