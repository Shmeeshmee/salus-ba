package terminology_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/Shmeeshmee/salus/internal/terminology"
)

func TestQueue_GivesStaffAPageWithTheLanguagesLabelsThenLatinAndEnglish(t *testing.T) {
	t.Parallel()
	f := newReviewFixture()
	tr := &terminology.Translation{TermID: "00000000-0000-4000-8000-0000000000b1", Language: "hr", VersionID: 1, Review: terminology.Review{Status: terminology.ReviewUnreviewed}}
	f.repo.rows = []terminology.QueueRow{
		{Status: terminology.ReviewUnreviewed, Translation: tr, Term: terminology.Term{ID: "00000000-0000-4000-8000-0000000000b1", Rank: 90, Category: "anatomy", Labels: []terminology.Label{
			{Language: "la", Text: "hepar", Preferred: true}, {Language: "hr", Text: "jetra", Preferred: true}, {Language: "hr", Text: "džigerica"}, {Language: "en", Text: "liver", Preferred: true},
		}}},
		{Status: terminology.ReviewUnreviewed, Term: terminology.Term{ID: "00000000-0000-4000-8000-0000000000b2", Rank: 80, Category: "drug", Labels: []terminology.Label{{Language: "en", Text: "aspirin", Preferred: true}}}},
	}
	page, err := f.svc.Queue(t.Context(), terminology.Caller{AccountID: "a-staff", Staff: true}, terminology.QueueQuery{Language: "hr", Status: terminology.ReviewUnreviewed, Limit: 2})
	if err != nil {
		t.Fatalf("Queue() error = %v", err)
	}
	if want := (terminology.QueueFilter{Language: "hr", Status: terminology.ReviewUnreviewed, Limit: 2}); *f.repo.filter != want {
		t.Errorf("filter = %+v, want %+v", *f.repo.filter, want)
	}
	wantFirst := terminology.QueueEntry{
		Status: terminology.ReviewUnreviewed, Translation: tr, Term: terminology.Term{ID: "00000000-0000-4000-8000-0000000000b1", Rank: 90, Category: "anatomy"},
		Labels: []terminology.Names{
			{Language: "hr", Preferred: "jetra", Synonyms: []string{"džigerica"}},
			{Language: "la", Preferred: "hepar", Synonyms: []string{}},
			{Language: "en", Preferred: "liver", Synonyms: []string{}},
		},
	}
	if len(page.Entries) != 2 || !reflect.DeepEqual(page.Entries[0], wantFirst) {
		t.Fatalf("entries = %+v;\nwant first %+v", page.Entries, wantFirst)
	}
	if page.Next == "" {
		t.Fatal("a full page has no Next")
	}

	// The next page starts after the last entry: its status, rank and id.
	if _, err := f.svc.Queue(t.Context(), reviewer, terminology.QueueQuery{Language: "hr", After: page.Next, Limit: 2}); err != nil {
		t.Fatalf("Queue(after) error = %v", err)
	}
	if want := (terminology.QueueFilter{Language: "hr", AfterStatus: terminology.ReviewUnreviewed, AfterRank: 80, AfterID: "00000000-0000-4000-8000-0000000000b2", Limit: 2}); *f.repo.filter != want {
		t.Errorf("filter of the next page = %+v, want %+v", *f.repo.filter, want)
	}
	f.repo.rows = f.repo.rows[:1]
	if last, err := f.svc.Queue(t.Context(), reviewer, terminology.QueueQuery{Language: "la", Limit: 2}); err != nil || last.Next != "" {
		t.Errorf("a page shorter than the limit: Next %q, %v; want none", last.Next, err)
	}
}

func TestQueue_RefusesAPatientAndAnInvalidQuery(t *testing.T) {
	t.Parallel()
	f := newReviewFixture()
	if _, err := f.svc.Queue(t.Context(), terminology.Caller{AccountID: "a-patient"}, terminology.QueueQuery{Language: "hr", Limit: 25}); !errors.Is(err, terminology.ErrForbidden) {
		t.Errorf("Queue as a patient = %v, want ErrForbidden", err)
	}
	for name, q := range map[string]terminology.QueueQuery{
		"no language":       {Limit: 25},
		"an unknown one":    {Language: "xx", Limit: 25},
		"an unknown status": {Language: "hr", Status: "reviewed", Limit: 25},
		"limit 0":           {Language: "hr", Limit: 0},
		"limit 101":         {Language: "hr", Limit: 101},
		"a made-up after":   {Language: "hr", Limit: 25, After: "x"},
		"an after's status": {Language: "hr", Limit: 25, After: "done~1~00000000-0000-4000-8000-0000000000b1"},
		"an after's rank":   {Language: "hr", Limit: 25, After: "approved~x~00000000-0000-4000-8000-0000000000b1"},
		"an after's id":     {Language: "hr", Limit: 25, After: "approved~1~not-a-uuid"},
	} {
		if _, err := f.svc.Queue(t.Context(), reviewer, q); !errors.Is(err, terminology.ErrInvalidReview) {
			t.Errorf("%s: Queue() = %v, want ErrInvalidReview", name, err)
		}
	}
	if f.repo.filter != nil {
		t.Errorf("the repository was asked %+v, want nothing", *f.repo.filter)
	}
	f.repo.err = errors.New("connection reset")
	if _, err := f.svc.Queue(t.Context(), reviewer, terminology.QueueQuery{Language: "hr", Limit: 25}); err == nil {
		t.Error("Queue() with a failing repository: error = nil")
	}
}

func TestProgress_GivesStaffEveryLanguageInTheEnumsOrder(t *testing.T) {
	t.Parallel()
	f := newReviewFixture()
	f.repo.progress = []terminology.LanguageProgress{{Language: "la", Terms: 3, Approved: 1}, {Language: "hr", Terms: 3, Approved: 2, NeedsChanges: 1}}
	got, err := f.svc.Progress(t.Context(), terminology.Caller{AccountID: "a-staff", Staff: true})
	if err != nil || len(got) != 11 {
		t.Fatalf("Progress() = %+v, %v; want the 11 term languages", got, err)
	}
	if got[0] != f.repo.progress[0] || got[2] != f.repo.progress[1] || got[1] != (terminology.LanguageProgress{Language: "bs", Terms: 3}) {
		t.Errorf("Progress() = %+v; want la and hr as read and bs with no review yet", got)
	}
	if _, err := f.svc.Progress(t.Context(), terminology.Caller{AccountID: "a-patient"}); !errors.Is(err, terminology.ErrForbidden) {
		t.Errorf("Progress as a patient = %v, want ErrForbidden", err)
	}
	f.repo.err = errors.New("connection reset")
	if _, err := f.svc.Progress(t.Context(), reviewer); err == nil {
		t.Error("Progress() with a failing repository: error = nil")
	}
}
