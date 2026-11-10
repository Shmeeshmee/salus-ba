package terminology_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/terminology"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

// The review panel's rules (docs/plans/v1-finish.md "Contract", docs/terminology/SPEC.md §3 "Review").

var (
	reviewNow = time.Date(2026, 11, 10, 9, 30, 0, 0, time.UTC)
	earlier   = time.Date(2026, 11, 1, 8, 0, 0, 0, time.UTC)
	reviewer  = terminology.Caller{AccountID: "a0000000-0000-4000-8000-0000000000r1", OrganizationID: "o1", Staff: true, Reviewer: true}
)

type reviewFixture struct {
	repo  *fakeReviews
	audit *fakeAudit
	tx    *fakeReviewTx
	svc   *terminology.Reviews
}

func newReviewFixture() reviewFixture {
	f := reviewFixture{repo: newFakeReviews(), audit: &fakeAudit{}, tx: &fakeReviewTx{}}
	f.svc = terminology.NewReviews(terminology.ReviewPorts{Repo: f.repo, Audit: f.audit, Tx: f.tx, Clock: clock.Fixed{T: reviewNow}})
	return f
}

// approved is a review the reviewer r1 gave earlier.
func approved(note string) terminology.Review {
	return terminology.Review{Status: terminology.ReviewApproved, ReviewedBy: "a-earlier", ReviewedAt: earlier, Note: note}
}

// writeEvent is the audit event of a reviewer's successful write on table.
func writeEvent(action auditevent.Action, table string) auditevent.AuditEvent {
	code := map[auditevent.Action]string{auditevent.ActionCreate: auditevent.CodeCreate, auditevent.ActionUpdate: auditevent.CodeUpdate}[action]
	return auditevent.AuditEvent{
		Type:           auditevent.Coding{System: auditevent.SystemAuditEventType, Code: auditevent.CodeRest},
		Subtype:        auditevent.Coding{System: auditevent.SystemRestfulInteraction, Code: code},
		Purpose:        auditevent.Coding{System: auditevent.SystemActReason, Code: auditevent.CodeHOPERAT},
		Action:         action,
		Outcome:        auditevent.OutcomeSuccess,
		AgentAccountID: reviewer.AccountID,
		OrganizationID: reviewer.OrganizationID,
		EntityName:     table,
		EntityCount:    1,
	}
}

func (f reviewFixture) onlyEvent(t *testing.T) auditevent.AuditEvent {
	t.Helper()
	if len(f.audit.events) != 1 || f.audit.outside != 0 {
		t.Fatalf("audit events %+v (%d outside a transaction), want exactly one inside", f.audit.events, f.audit.outside)
	}
	if len(f.repo.outside) != 0 {
		t.Errorf("repository calls outside a transaction: %v", f.repo.outside)
	}
	return f.audit.events[0]
}

func TestEditTranslation_SetsAnApprovedEntryBackToUnreviewedKeepingItsLastReview(t *testing.T) {
	t.Parallel()
	f := newReviewFixture()
	term := f.repo.add("t1", "la", "hr", "en")
	term.translations["hr"] = terminology.Translation{TermID: "t1", Language: "hr", VersionID: 3, Review: approved("Fine.")}
	term.review = approved("")

	got, err := f.svc.EditTranslation(t.Context(), reviewer, "t1", "hr", terminology.LabelsEdit{Preferred: "  jetra ", Synonyms: []string{"džigerica", " jetra"}}, 3)
	if err != nil {
		t.Fatalf("EditTranslation() error = %v", err)
	}
	want := terminology.Translation{
		TermID: "t1", Language: "hr", VersionID: 4, UpdatedAt: reviewNow,
		Review: terminology.Review{Status: terminology.ReviewUnreviewed, ReviewedBy: "a-earlier", ReviewedAt: earlier, Note: "Fine."},
	}
	if !reflect.DeepEqual(got, want) || !reflect.DeepEqual(term.translations["hr"], want) {
		t.Errorf("EditTranslation() = %+v, stored %+v;\nwant %+v", got, term.translations["hr"], want)
	}
	wantLabels := []terminology.Label{{Language: "hr", Text: "jetra", Preferred: true}, {Language: "hr", Text: "džigerica"}}
	if !reflect.DeepEqual(term.labels["hr"], wantLabels) {
		t.Errorf("labels = %+v, want %+v (trimmed, the preferred one first, the repeat dropped)", term.labels["hr"], wantLabels)
	}
	if term.review.Status != terminology.ReviewUnreviewed || term.review.ReviewedBy != "a-earlier" || !term.review.ReviewedAt.Equal(earlier) {
		t.Errorf("the term's review = %+v, want unreviewed with its last review kept", term.review)
	}
	if e := f.onlyEvent(t); e != writeEvent(auditevent.ActionUpdate, "term_translation") {
		t.Errorf("audit event %+v, want the update of term_translation by the reviewer", e)
	}
}

func TestEditTranslation_GivesALanguageWithoutLabelsItsFirstUnreviewed(t *testing.T) {
	t.Parallel()
	f := newReviewFixture()
	term := f.repo.add("t1", "la")

	got, err := f.svc.EditTranslation(t.Context(), reviewer, "t1", "bs", terminology.LabelsEdit{Preferred: "čir"}, 0)
	if err != nil {
		t.Fatalf("EditTranslation() error = %v", err)
	}
	want := terminology.Translation{TermID: "t1", Language: "bs", VersionID: 1, UpdatedAt: reviewNow, Review: terminology.Review{Status: terminology.ReviewUnreviewed}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("EditTranslation() = %+v, want %+v", got, want)
	}
	if !reflect.DeepEqual(term.labels["bs"], []terminology.Label{{Language: "bs", Text: "čir", Preferred: true}}) {
		t.Errorf("labels = %+v", term.labels["bs"])
	}
	if e := f.onlyEvent(t); e != writeEvent(auditevent.ActionCreate, "term_translation") {
		t.Errorf("audit event %+v, want the creation of a term_translation", e)
	}
}

func TestEditTranslation_TheSameLabelsChangeNothing(t *testing.T) {
	t.Parallel()
	f := newReviewFixture()
	term := f.repo.add("t1", "hr")
	term.labels["hr"] = []terminology.Label{{Language: "hr", Text: "jetra", Preferred: true}, {Language: "hr", Text: "džigerica"}}
	term.translations["hr"] = terminology.Translation{TermID: "t1", Language: "hr", VersionID: 2, Review: approved("")}

	got, err := f.svc.EditTranslation(t.Context(), reviewer, "t1", "hr", terminology.LabelsEdit{Preferred: "jetra", Synonyms: []string{"džigerica"}}, 2)
	if err != nil || got.Review.Status != terminology.ReviewApproved || got.VersionID != 2 {
		t.Errorf("EditTranslation() = %+v, %v; want the approved translation unchanged", got, err)
	}
	if len(f.repo.writes) != 0 || len(f.audit.events) != 0 {
		t.Errorf("writes %v, audit events %d; want none", f.repo.writes, len(f.audit.events))
	}
}

func TestEditTranslation_RefusesLabelsTheTermListDoesNotTake(t *testing.T) {
	t.Parallel()
	many := make([]string, 21)
	for i := range many {
		many[i] = "s" + strings.Repeat("x", i)
	}
	for name, edit := range map[string]terminology.LabelsEdit{
		"no preferred label":   {Preferred: "  ", Synonyms: []string{"jetra"}},
		"a label over 300":     {Preferred: strings.Repeat("a", 301)},
		"a label with |":       {Preferred: "jetra | hepar"},
		"more than 20 synonym": {Preferred: "jetra", Synonyms: many},
		"invalid UTF-8":        {Preferred: "jetra\xff"},
	} {
		f := newReviewFixture()
		f.repo.add("t1", "hr")
		if _, err := f.svc.EditTranslation(t.Context(), reviewer, "t1", "hr", edit, 1); !errors.Is(err, terminology.ErrInvalidReview) {
			t.Errorf("%s: EditTranslation() = %v, want ErrInvalidReview", name, err)
		}
		if len(f.repo.writes) != 0 {
			t.Errorf("%s: writes %v, want none", name, f.repo.writes)
		}
	}
	f := newReviewFixture()
	f.repo.add("t1", "hr")
	if _, err := f.svc.EditTranslation(t.Context(), reviewer, "t1", "xx", terminology.LabelsEdit{Preferred: "jetra"}, 1); !errors.Is(err, terminology.ErrInvalidReview) {
		t.Errorf("an unknown language: %v, want ErrInvalidReview", err)
	}
}

// Every write is a reviewer's, and checks the version read: a staff member who is no reviewer is refused (and
// audited), as is a reviewer whose session has no clinic; a stale version is a conflict; an unknown term is not
// found.
func TestReviewWrites_RefuseANonReviewerAStaleVersionAndAnUnknownTerm(t *testing.T) {
	t.Parallel()
	type write func(f reviewFixture, c terminology.Caller, term string, version int32) error
	writes := map[string]write{
		"EditTranslation": func(f reviewFixture, c terminology.Caller, term string, v int32) error {
			_, err := f.svc.EditTranslation(t.Context(), c, term, "hr", terminology.LabelsEdit{Preferred: "nova"}, v)
			return err
		},
		"ReviewTranslation": func(f reviewFixture, c terminology.Caller, term string, v int32) error {
			_, err := f.svc.ReviewTranslation(t.Context(), c, term, "hr", terminology.Decision{Status: terminology.ReviewApproved}, v)
			return err
		},
		"WriteExplanation": func(f reviewFixture, c terminology.Caller, term string, v int32) error {
			_, err := f.svc.WriteExplanation(t.Context(), c, term, "hr", "Nova.", v)
			return err
		},
		"ReviewExplanation": func(f reviewFixture, c terminology.Caller, term string, v int32) error {
			_, err := f.svc.ReviewExplanation(t.Context(), c, term, "hr", terminology.Decision{Status: terminology.ReviewApproved}, v)
			return err
		},
	}
	staffOnly := terminology.Caller{AccountID: "a-staff", OrganizationID: "o1", Staff: true}
	notStaff := terminology.Caller{AccountID: "a-gone", Reviewer: true}
	for name, w := range writes {
		f := newReviewFixture()
		term := f.repo.add("t1", "hr")
		term.explanations["hr"] = terminology.Explanation{TermID: "t1", Language: "hr", Text: "Jetra.", VersionID: 1, Review: terminology.Review{Status: terminology.ReviewUnreviewed}}

		for who, c := range map[string]terminology.Caller{"staff who is no reviewer": staffOnly, "a reviewer without a clinic": notStaff} {
			f.audit.events = nil
			if err := w(f, c, "t1", 1); !errors.Is(err, terminology.ErrForbidden) {
				t.Errorf("%s by %s = %v, want ErrForbidden", name, who, err)
			}
			if len(f.audit.events) != 1 || f.audit.events[0].Outcome != auditevent.OutcomeMinorFailure || f.audit.events[0].EntityCount != 0 ||
				f.audit.events[0].AgentAccountID != c.AccountID {
				t.Errorf("%s by %s: audit events %+v, want one refusal of the caller on no row", name, who, f.audit.events)
			}
		}
		f.audit.events = nil
		if err := w(f, reviewer, "t1", 2); !errors.Is(err, terminology.ErrVersionConflict) {
			t.Errorf("%s at a stale version = %v, want ErrVersionConflict", name, err)
		}
		if err := w(f, reviewer, "t-unknown", 1); !errors.Is(err, terminology.ErrTermNotFound) {
			t.Errorf("%s of an unknown term = %v, want ErrTermNotFound", name, err)
		}
		if len(f.repo.writes) != 0 || len(f.audit.events) != 0 {
			t.Errorf("%s refused: writes %v, audit events %d; want none", name, f.repo.writes, len(f.audit.events))
		}
		f.repo.err = errors.New("connection reset")
		if err := w(f, reviewer, "t1", 1); err == nil || errors.Is(err, terminology.ErrTermNotFound) {
			t.Errorf("%s with a failing repository = %v, want its error", name, err)
		}
	}
}

func TestReviewTranslation_ApprovesOrMarksNeedsChangesWithTheReviewerAndTheNote(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		decision terminology.Decision
		want     terminology.Review
	}{
		{
			name:     "approved without a note clears the old one",
			decision: terminology.Decision{Status: terminology.ReviewApproved},
			want:     terminology.Review{Status: terminology.ReviewApproved, ReviewedBy: reviewer.AccountID, ReviewedAt: reviewNow},
		},
		{
			name:     "approved with a note",
			decision: terminology.Decision{Status: terminology.ReviewApproved, Note: " Checked against TA98. "},
			want:     terminology.Review{Status: terminology.ReviewApproved, ReviewedBy: reviewer.AccountID, ReviewedAt: reviewNow, Note: "Checked against TA98."},
		},
		{
			name:     "needs changes with its note",
			decision: terminology.Decision{Status: terminology.ReviewNeedsChanges, Note: "Use the Croatian standard term."},
			want:     terminology.Review{Status: terminology.ReviewNeedsChanges, ReviewedBy: reviewer.AccountID, ReviewedAt: reviewNow, Note: "Use the Croatian standard term."},
		},
	}
	for _, tt := range tests {
		f := newReviewFixture()
		term := f.repo.add("t1", "hr")
		tr := term.translations["hr"]
		tr.Review.Note = "Old note."
		term.translations["hr"] = tr

		got, err := f.svc.ReviewTranslation(t.Context(), reviewer, "t1", "hr", tt.decision, 1)
		want := terminology.Translation{TermID: "t1", Language: "hr", VersionID: 2, UpdatedAt: reviewNow, Review: tt.want}
		if err != nil || !reflect.DeepEqual(got, want) || !reflect.DeepEqual(term.translations["hr"], want) {
			t.Errorf("%s: ReviewTranslation() = %+v, %v; stored %+v;\nwant %+v", tt.name, got, err, term.translations["hr"], want)
		}
		if e := f.onlyEvent(t); e != writeEvent(auditevent.ActionUpdate, "term_translation") {
			t.Errorf("%s: audit event %+v", tt.name, e)
		}
	}
}

func TestReviewTranslation_RefusesNeedsChangesWithoutANoteAndAnyOtherStatus(t *testing.T) {
	t.Parallel()
	f := newReviewFixture()
	f.repo.add("t1", "hr")
	if _, err := f.svc.ReviewTranslation(t.Context(), reviewer, "t1", "hr", terminology.Decision{Status: terminology.ReviewNeedsChanges, Note: "  "}, 1); !errors.Is(err, terminology.ErrNoteRequired) {
		t.Errorf("needs changes without a note = %v, want ErrNoteRequired", err)
	}
	for _, d := range []terminology.Decision{
		{Status: terminology.ReviewUnreviewed},
		{Status: "reviewed"},
		{Status: terminology.ReviewApproved, Note: strings.Repeat("a", 501)},
	} {
		if _, err := f.svc.ReviewTranslation(t.Context(), reviewer, "t1", "hr", d, 1); !errors.Is(err, terminology.ErrInvalidReview) {
			t.Errorf("ReviewTranslation(%.20q) = %v, want ErrInvalidReview", d.Status, err)
		}
	}
	if _, err := f.svc.ReviewTranslation(t.Context(), reviewer, "t1", "bs", terminology.Decision{Status: terminology.ReviewApproved}, 1); !errors.Is(err, terminology.ErrNoTranslation) {
		t.Errorf("reviewing a language without labels = %v, want ErrNoTranslation", err)
	}
	if len(f.repo.writes) != 0 {
		t.Errorf("writes %v, want none", f.repo.writes)
	}
}

// The term's own status, which the cheatsheet shows, follows its translations in Latin and the staff languages:
// needs changes when one does, approved when all do, else unreviewed. The reviewer and the time are set when it
// turns approved or needs changes, and kept when it goes back to unreviewed. Labels in a patient-only language
// (German here) do not count: the cheatsheet does not show them.
func TestReviewTranslation_TheTermsStatusFollowsItsTranslations(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		others map[string]string // the other translations' statuses
		before terminology.Review
		review terminology.Decision
		want   terminology.Review
	}{
		{
			name: "the last one approved", others: map[string]string{"la": terminology.ReviewApproved, "en": terminology.ReviewApproved, "de": terminology.ReviewUnreviewed},
			before: terminology.Review{Status: terminology.ReviewUnreviewed}, review: terminology.Decision{Status: terminology.ReviewApproved},
			want: terminology.Review{Status: terminology.ReviewApproved, ReviewedBy: reviewer.AccountID, ReviewedAt: reviewNow},
		},
		{
			name: "one still unreviewed", others: map[string]string{"la": terminology.ReviewApproved, "en": terminology.ReviewUnreviewed},
			before: terminology.Review{Status: terminology.ReviewUnreviewed}, review: terminology.Decision{Status: terminology.ReviewApproved},
			want: terminology.Review{Status: terminology.ReviewUnreviewed},
		},
		{
			name: "one needs changes", others: map[string]string{"la": terminology.ReviewApproved, "en": terminology.ReviewUnreviewed},
			before: terminology.Review{Status: terminology.ReviewUnreviewed}, review: terminology.Decision{Status: terminology.ReviewNeedsChanges, Note: "Wrong."},
			want: terminology.Review{Status: terminology.ReviewNeedsChanges, ReviewedBy: reviewer.AccountID, ReviewedAt: reviewNow},
		},
		{
			name: "approved stays approved, its review kept", others: map[string]string{"la": terminology.ReviewApproved},
			before: approved("Curator's."), review: terminology.Decision{Status: terminology.ReviewApproved},
			want: approved("Curator's."),
		},
	}
	for _, tt := range tests {
		f := newReviewFixture()
		term := f.repo.add("t1", "hr")
		for lang, status := range tt.others {
			term.translations[lang] = terminology.Translation{TermID: "t1", Language: lang, VersionID: 1, Review: terminology.Review{Status: status, ReviewedAt: earlier}}
		}
		term.review = tt.before
		if _, err := f.svc.ReviewTranslation(t.Context(), reviewer, "t1", "hr", tt.review, 1); err != nil {
			t.Fatalf("%s: ReviewTranslation() error = %v", tt.name, err)
		}
		if !reflect.DeepEqual(term.review, tt.want) {
			t.Errorf("%s: the term's review = %+v,\nwant %+v", tt.name, term.review, tt.want)
		}
		wroteTerm := slicesContains(f.repo.writes, "SetTermReview")
		if changed := tt.before.Status != tt.want.Status; wroteTerm != changed {
			t.Errorf("%s: the term written %v, want %v (only when its status changes)", tt.name, wroteTerm, changed)
		}
		if wroteTerm && !term.termAt.Equal(reviewNow) {
			t.Errorf("%s: the term's updated_at = %v, want now", tt.name, term.termAt)
		}
	}
}

func slicesContains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

func TestWriteExplanation_NewOrChangedIsUnreviewedAndTheSameTextChangesNothing(t *testing.T) {
	t.Parallel()
	f := newReviewFixture()
	term := f.repo.add("t1", "hr")
	term.review = approved("")

	got, err := f.svc.WriteExplanation(t.Context(), reviewer, "t1", "hr", "  Jetra je organ u trbuhu.\n", 0)
	want := terminology.Explanation{TermID: "t1", Language: "hr", Text: "Jetra je organ u trbuhu.", VersionID: 1, UpdatedAt: reviewNow, Review: terminology.Review{Status: terminology.ReviewUnreviewed}}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("WriteExplanation(new) = %+v, %v;\nwant %+v", got, err, want)
	}
	if e := f.onlyEvent(t); e != writeEvent(auditevent.ActionCreate, "term_explanation") {
		t.Errorf("audit event %+v", e)
	}
	if term.review.Status != terminology.ReviewApproved || slicesContains(f.repo.writes, "SetTermReview") {
		t.Errorf("the term's review = %+v after an explanation; want it unchanged (explanations do not count)", term.review)
	}

	e := term.explanations["hr"]
	e.Review = approved("Clear.")
	term.explanations["hr"] = e
	f.audit.events, f.repo.writes = nil, nil
	if same, err := f.svc.WriteExplanation(t.Context(), reviewer, "t1", "hr", "Jetra je organ u trbuhu.", 1); err != nil || same.Review.Status != terminology.ReviewApproved || len(f.repo.writes) != 0 {
		t.Errorf("WriteExplanation(the same text) = %+v, %v, writes %v; want the approved explanation unchanged", same, err, f.repo.writes)
	}

	got, err = f.svc.WriteExplanation(t.Context(), reviewer, "t1", "hr", "Jetra je najveći organ u trbuhu.", 1)
	want = terminology.Explanation{
		TermID: "t1", Language: "hr", Text: "Jetra je najveći organ u trbuhu.", VersionID: 2, UpdatedAt: reviewNow,
		Review: terminology.Review{Status: terminology.ReviewUnreviewed, ReviewedBy: "a-earlier", ReviewedAt: earlier, Note: "Clear."},
	}
	if err != nil || !reflect.DeepEqual(got, want) || !reflect.DeepEqual(term.explanations["hr"], want) {
		t.Errorf("WriteExplanation(changed) = %+v, %v;\nwant %+v", got, err, want)
	}
	if e := f.onlyEvent(t); e != writeEvent(auditevent.ActionUpdate, "term_explanation") {
		t.Errorf("audit event %+v", e)
	}
}

func TestWriteExplanation_RefusesABlankOrLongTextAndLatin(t *testing.T) {
	t.Parallel()
	f := newReviewFixture()
	f.repo.add("t1", "la", "hr")
	for name, c := range map[string]struct{ lang, text string }{
		"blank":          {"hr", " \n "},
		"over 500":       {"hr", strings.Repeat("a", 501)},
		"Latin":          {"la", "Iecur."},
		"an unknown tag": {"xx", "Jetra."},
		"invalid UTF-8":  {"hr", "Jetra\xff"},
	} {
		if _, err := f.svc.WriteExplanation(t.Context(), reviewer, "t1", c.lang, c.text, 0); !errors.Is(err, terminology.ErrInvalidReview) {
			t.Errorf("%s: WriteExplanation() = %v, want ErrInvalidReview", name, err)
		}
	}
	if _, err := f.svc.WriteExplanation(t.Context(), reviewer, "t1", "hr", strings.Repeat("č", 500), 0); err != nil {
		t.Errorf("500 characters (1,000 bytes): %v, want it taken", err)
	}
}

func TestReviewExplanation_ApprovesOrMarksNeedsChanges(t *testing.T) {
	t.Parallel()
	f := newReviewFixture()
	term := f.repo.add("t1", "hr")
	term.explanations["hr"] = terminology.Explanation{TermID: "t1", Language: "hr", Text: "Jetra.", VersionID: 4, Review: terminology.Review{Status: terminology.ReviewUnreviewed}}

	if _, err := f.svc.ReviewExplanation(t.Context(), reviewer, "t1", "hr", terminology.Decision{Status: terminology.ReviewNeedsChanges}, 4); !errors.Is(err, terminology.ErrNoteRequired) {
		t.Errorf("needs changes without a note = %v, want ErrNoteRequired", err)
	}
	got, err := f.svc.ReviewExplanation(t.Context(), reviewer, "t1", "hr", terminology.Decision{Status: terminology.ReviewNeedsChanges, Note: "Too short."}, 4)
	want := terminology.Explanation{
		TermID: "t1", Language: "hr", Text: "Jetra.", VersionID: 5, UpdatedAt: reviewNow,
		Review: terminology.Review{Status: terminology.ReviewNeedsChanges, ReviewedBy: reviewer.AccountID, ReviewedAt: reviewNow, Note: "Too short."},
	}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("ReviewExplanation() = %+v, %v;\nwant %+v", got, err, want)
	}
	if e := f.onlyEvent(t); e != writeEvent(auditevent.ActionUpdate, "term_explanation") {
		t.Errorf("audit event %+v", e)
	}
	if _, err := f.svc.ReviewExplanation(t.Context(), reviewer, "t1", "bs", terminology.Decision{Status: terminology.ReviewApproved}, 1); !errors.Is(err, terminology.ErrNoExplanation) {
		t.Errorf("reviewing a missing explanation = %v, want ErrNoExplanation", err)
	}
}

func TestReviewWrites_FailWhenTheAuditFails(t *testing.T) {
	t.Parallel()
	f := newReviewFixture()
	f.repo.add("t1", "hr")
	f.audit.err = errors.New("audit down")
	if _, err := f.svc.ReviewTranslation(t.Context(), reviewer, "t1", "hr", terminology.Decision{Status: terminology.ReviewApproved}, 1); err == nil {
		t.Error("ReviewTranslation() with a failing audit: error = nil")
	}
}
