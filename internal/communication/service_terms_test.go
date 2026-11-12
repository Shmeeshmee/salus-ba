package communication_test

import (
	"slices"
	"testing"

	"github.com/Shmeeshmee/salus/internal/terminology"
)

// The terms of a message's body (SAL-13), on read: in the reader's language, then English where none of theirs is;
// the body unchanged and nothing stored; a reader without a language gets English; a failure leaves them out.
func TestMessages_TagTheTermsInTheReadersLanguageWithEnglishAsTheFallback(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	c := f.open(t)
	body := "Glavobolja, maybe a migraine"
	f.send(t, me, c.ID, body)
	f.sal30.languages.by[pract.AccountID] = "bs"
	page, err := f.svc.Messages(t.Context(), pract, c.ID, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	m := page.Messages[0]
	want := []terminology.Span{{TermID: termHeadache, Start: 0, End: 10}, {TermID: termMigraine, Start: 20, End: 28}}
	if !slices.Equal(m.Terms, want) || m.Body != body {
		t.Errorf("terms %v (body %q), want %v", m.Terms, m.Body, want)
	}
	if stored := f.repo.msgs[c.ID][0]; stored.Terms != nil || stored.Body != body {
		t.Errorf("stored %+v, want no terms", stored)
	}
	page, _ = f.svc.Messages(t.Context(), me, c.ID, nil, 0)
	if got := page.Messages[0].Terms; !slices.Equal(got, want[1:]) {
		t.Errorf("a reader without a language: %v, want English's only", got)
	}
	f.sal30.terms.mu.Lock()
	f.sal30.terms.err = errTag
	f.sal30.terms.mu.Unlock()
	page, err = f.svc.Messages(t.Context(), pract, c.ID, nil, 0)
	if err != nil || page.Messages[0].Terms != nil {
		t.Errorf("tagging down: %v, %v; want the page without terms", page.Messages[0].Terms, err)
	}
}

// A send's and an edit's answers carry the terms of their body for the writer, as a page does.
func TestSendAndEdit_AnswerWithTheTermsOfTheBody(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	c := f.open(t)
	m := f.send(t, me, c.ID, "A migraine again")
	if want := []terminology.Span{{TermID: termMigraine, Start: 2, End: 10}}; !slices.Equal(m.Terms, want) {
		t.Errorf("send's terms %v, want %v", m.Terms, want)
	}
	edited, err := f.svc.Edit(t.Context(), me, c.ID, m.ID, "Migraine", 1)
	if want := []terminology.Span{{TermID: termMigraine, Start: 0, End: 8}}; err != nil || !slices.Equal(edited.Terms, want) {
		t.Errorf("edit's terms %v, %v; want %v", edited.Terms, err, want)
	}
}
