package communication_test

import (
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/communication"
)

func bodies(rs []communication.SearchResult) []string {
	var out []string
	for _, r := range rs {
		out = append(out, r.Message.Body)
	}
	return out
}

// Search within a conversation (SAL-30): the words of the query, each a prefix, all of them; newest first; only what
// the caller reads (after a later practitioner's history_from_seq); one search event with the count.
func TestSearch_FindsTheConversationsMessagesTheCallerReadsNewestFirst(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	c := f.open(t)
	for i, body := range []string{"Synthetic headache since Monday", "Synthetic placeholder", "Headache again, synthetic"} {
		f.clock.set(start.Add(time.Duration(i) * time.Minute))
		f.send(t, me, c.ID, body)
	}
	got, err := f.svc.Search(t.Context(), pract, c.ID, "  HEAD synth ")
	if want := []string{"Headache again, synthetic", "Synthetic headache since Monday"}; err != nil || !slices.Equal(bodies(got), want) {
		t.Fatalf("Search() = %v, %v; want %v", bodies(got), err, want)
	}
	if got[0].ConversationID != c.ID || f.repo.queries[0] != "'HEAD':* & 'synth':*" {
		t.Errorf("result %+v, query %q", got[0], f.repo.queries[0])
	}
	if e := f.audit.last(t); e.Subtype.Code != auditevent.CodeSearchType || e.EntityName != "communication" || e.EntityCount != 2 ||
		e.EntityReference != "Patient/"+patientA || e.Purpose.Code != auditevent.CodeTREAT {
		t.Errorf("audit = %+v", e)
	}
	f.repo.addPractitioner(c.ID, roleA2, 2)
	if got, err := f.svc.Search(t.Context(), pract2, c.ID, "headache"); err != nil || !slices.Equal(bodies(got), []string{"Headache again, synthetic"}) {
		t.Errorf("a practitioner who joined after message 2: %v, %v", bodies(got), err)
	}
	if _, err := f.svc.Search(t.Context(), other, c.ID, "headache"); !errors.Is(err, communication.ErrNotFound) {
		t.Errorf("another patient: %v, want ErrNotFound", err)
	}
	f.revoke()
	if _, err := f.svc.Search(t.Context(), pract, c.ID, "headache"); !errors.Is(err, communication.ErrNotFound) {
		t.Errorf("staff after the revocation: %v, want ErrNotFound", err)
	}
}

// A query is words of letters, digits and marks: anything else separates them and never reaches the text search's
// syntax; a word of one letter is dropped; nothing left, or a query over 200 characters, is invalid.
func TestSearch_TakesOnlyWordsAndRefusesAnEmptyOrLongQuery(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	c := f.open(t)
	if _, err := f.svc.Search(t.Context(), me, c.ID, `mučnina' | !x & (glava):*`); err != nil {
		t.Fatal(err)
	}
	if got := f.repo.queries[0]; got != "'mučnina':* & 'glava':*" {
		t.Errorf("query %q, want the two words", got)
	}
	for _, q := range []string{"", "   ", "a ! b", strings.Repeat("x", 201), "' & |"} {
		if _, err := f.svc.Search(t.Context(), me, c.ID, q); !errors.Is(err, communication.ErrInvalid) {
			t.Errorf("Search(%q) = %v, want ErrInvalid", q, err)
		}
		if _, err := f.svc.SearchAll(t.Context(), pract, q); !errors.Is(err, communication.ErrInvalid) {
			t.Errorf("SearchAll(%q) = %v, want ErrInvalid", q, err)
		}
	}
	words := strings.Repeat("abc ", 12)
	if _, err := f.svc.Search(t.Context(), me, c.ID, words); err != nil || strings.Count(f.repo.queries[len(f.repo.queries)-1], "'abc'") != 8 {
		t.Errorf("twelve words: %v, query %q; want the first eight", err, f.repo.queries[len(f.repo.queries)-1])
	}
}

// Search across the caller's conversations: a patient's own, never a doctor-only thread; a practitioner's through
// the role's scope; an admin none. Audited as one search.
func TestSearchAll_FindsTheMessagesOfTheCallersConversations(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	c := f.open(t)
	f.send(t, me, c.ID, "Synthetic rash on the arm")
	f.clock.set(start.Add(time.Minute))
	f.send(t, pract, c.ID, "Synthetic rash cream")
	got, err := f.svc.SearchAll(t.Context(), pract, "rash")
	if want := []string{"Synthetic rash cream", "Synthetic rash on the arm"}; err != nil || !slices.Equal(bodies(got), want) {
		t.Fatalf("practitioner: %v, %v; want %v", bodies(got), err, want)
	}
	if e := f.audit.last(t); e.Subtype.Code != auditevent.CodeSearchType || e.EntityName != "communication" || e.EntityCount != 2 || e.AgentAccountID != pract.AccountID {
		t.Errorf("audit = %+v", e)
	}
	if got, err := f.svc.SearchAll(t.Context(), me, "rash cream"); err != nil || !slices.Equal(bodies(got), []string{"Synthetic rash cream"}) {
		t.Errorf("patient: %v, %v", bodies(got), err)
	}
	if got, err := f.svc.SearchAll(t.Context(), pract2, "rash"); err != nil || len(got) != 0 {
		t.Errorf("a practitioner who takes no part: %v, %v; want none", bodies(got), err)
	}
	if got, err := f.svc.SearchAll(t.Context(), other, "rash"); err != nil || len(got) != 0 {
		t.Errorf("another patient: %v, %v; want none", bodies(got), err)
	}
	if _, err := f.svc.SearchAll(t.Context(), admin, "rash"); !errors.Is(err, communication.ErrForbidden) {
		t.Errorf("admin: %v, want ErrForbidden", err)
	}
	// The service decides as enter does, whatever the scope returned (the fake's search knows no consent): after the
	// revocation the practitioner finds nothing, the patient their own.
	f.revoke()
	if got, err := f.svc.SearchAll(t.Context(), pract, "rash"); err != nil || len(got) != 0 {
		t.Errorf("the practitioner after the revocation: %v, %v; want none", bodies(got), err)
	}
	if e := f.audit.last(t); e.EntityCount != 0 {
		t.Errorf("the search's audit counts %d, want what was returned (0)", e.EntityCount)
	}
	if got, err := f.svc.SearchAll(t.Context(), me, "rash"); err != nil || len(got) != 2 {
		t.Errorf("the patient after the revocation: %v, %v; want their two", bodies(got), err)
	}
}
