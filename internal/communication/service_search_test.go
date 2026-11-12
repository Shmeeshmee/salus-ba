package communication_test

import (
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/access"
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

// A search across a practitioner's conversations is audited once for each patient whose messages it returns, with
// their count (SAL-30 review, finding 2); a patient's for themselves; one that returns nothing once, naming nobody.
func TestSearchAll_AuditsEachPatientWhoseMessagesItReturns(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.mu.Lock()
	f.consent[patientB+clinicA] = access.Consent{Active: true}
	f.visits[patientB+clinicA] = []access.Visit{{PractitionerRole: roleA1, Start: start.Add(-48 * time.Hour)}}
	f.mu.Unlock()
	a := f.open(t)
	b, _, err := f.svc.Open(t.Context(), other, communication.Open{OrganizationID: clinicA, PractitionerRoleID: roleA1})
	if err != nil {
		t.Fatal(err)
	}
	f.send(t, me, a.ID, "Synthetic rash on the arm")
	f.send(t, pract, a.ID, "Synthetic rash cream")
	f.send(t, other, b.ID, "Synthetic rash again")
	type seen struct {
		ref   string
		count int64
	}
	searched := func(c communication.Caller, q string) []seen {
		t.Helper()
		before := f.audit.count()
		if _, err := f.svc.SearchAll(t.Context(), c, q); err != nil {
			t.Fatal(err)
		}
		var out []seen
		for _, e := range f.audit.events[before:] {
			if e.Subtype.Code != auditevent.CodeSearchType || e.EntityName != "communication" || e.AgentAccountID != c.AccountID {
				t.Errorf("audit = %+v", e)
			}
			out = append(out, seen{e.EntityReference, e.EntityCount})
		}
		slices.SortFunc(out, func(x, y seen) int { return strings.Compare(x.ref, y.ref) })
		return out
	}
	if got, want := searched(pract, "rash"), []seen{{"Patient/" + patientA, 2}, {"Patient/" + patientB, 1}}; !slices.Equal(got, want) {
		t.Errorf("practitioner: audited %v, want %v", got, want)
	}
	if got, want := searched(me, "rash"), []seen{{"Patient/" + patientA, 2}}; !slices.Equal(got, want) {
		t.Errorf("patient: audited %v, want %v", got, want)
	}
	if got, want := searched(pract, "nothing"), []seen{{"", 0}}; !slices.Equal(got, want) {
		t.Errorf("nothing found: audited %v, want %v", got, want)
	}
}

// guestSearch is a conversation of patient A with clinic A where clinic B's doctor was invited (SAL-28), in a state:
// a message of the conversation, one of a side thread opened before the invitation (without the guest) and, once
// granted, one of the side thread the guest opened.
type guestSearch struct {
	f                   *fixture
	conv, without, mine string
}

// The states of the guest's invitation a search sees.
const (
	searchPending     = "pending"
	searchGranted     = "granted"
	searchDeclined    = "declined"
	searchChatRevoked = "chat consent revoked"
	searchCareRevoked = "care consent revoked"
	searchRemoved     = "removed"
)

func newGuestSearch(t *testing.T, f *fixture, state string) guestSearch {
	t.Helper()
	g := guestSearch{f: f, conv: f.open(t).ID}
	f.send(t, me, g.conv, "Synthetic rash on the arm")
	without, err := f.svc.OpenThread(t.Context(), pract, g.conv)
	if err != nil {
		t.Fatal(err)
	}
	g.without = without.ID
	f.send(t, pract, g.without, "Synthetic rash, between the clinic's doctors")
	p := f.add(t, g.conv, roleB1, clinicB)
	if state == searchPending {
		return g
	}
	if state == searchDeclined {
		if err := f.svc.DecideInvitation(t.Context(), me, p.ID, communication.DecisionDecline); err != nil {
			t.Fatal(err)
		}
		return g
	}
	if err := f.svc.DecideInvitation(t.Context(), me, p.ID, communication.DecisionGrant); err != nil {
		t.Fatal(err)
	}
	mine, err := f.svc.OpenThread(t.Context(), guest, g.conv)
	if err != nil {
		t.Fatal(err)
	}
	g.mine = mine.ID
	f.clock.set(start.Add(time.Minute))
	f.send(t, guest, g.mine, "Synthetic rash, the guest's question")
	switch state {
	case searchChatRevoked:
		err = f.svc.RevokeInvitation(t.Context(), me, p.ID)
	case searchCareRevoked:
		f.revoke()
	case searchRemoved:
		err = f.svc.RemoveParticipant(t.Context(), pract, g.conv, p.ID)
	}
	if err != nil {
		t.Fatal(err)
	}
	return g
}

// check asserts what the guest finds in the state: granted, the conversation's message and their side thread's,
// never the side thread without them; nothing otherwise, and each conversation not found.
func (g guestSearch) check(t *testing.T, state string) {
	t.Helper()
	ctx := t.Context()
	granted := state == searchGranted
	var want []string
	if granted {
		want = []string{"Synthetic rash, the guest's question", "Synthetic rash on the arm"}
	}
	got, err := g.f.svc.SearchAll(ctx, guest, "rash")
	if err != nil || !slices.Equal(bodies(got), want) {
		t.Errorf("SearchAll = %v, %v; want %v", bodies(got), err, want)
	}
	for name, tt := range map[string]struct {
		id   string
		want []string
	}{
		"the conversation":             {id: g.conv, want: []string{"Synthetic rash on the arm"}},
		"their side thread":            {id: g.mine, want: []string{"Synthetic rash, the guest's question"}},
		"the side thread without them": {id: g.without},
	} {
		if tt.id == "" {
			continue
		}
		got, err := g.f.svc.Search(ctx, guest, tt.id, "rash")
		switch {
		case granted && tt.want != nil:
			if err != nil || !slices.Equal(bodies(got), tt.want) {
				t.Errorf("Search in %s = %v, %v; want %v", name, bodies(got), err, tt.want)
			}
		case !errors.Is(err, communication.ErrNotFound) && !errors.Is(err, communication.ErrForbidden):
			t.Errorf("Search in %s = %v, %v; want it refused", name, bodies(got), err)
		}
	}
}

// A doctor of another clinic (SAL-28) searches only while the patient's chat consent for them and the care consent
// with the treating clinic are active, as an active participant: in the conversation and the side threads they take
// part in, never one without them; nothing while pending, after a decline, a revocation of either consent or a
// removal (SAL-30 after SAL-28, findings, 7). The service decides as enter does, whatever the scope returned.
func TestSearch_AGuestFindsOnlyWhileGranted(t *testing.T) {
	t.Parallel()
	for _, state := range []string{searchPending, searchGranted, searchDeclined, searchChatRevoked, searchCareRevoked, searchRemoved} {
		t.Run(state, func(t *testing.T) {
			t.Parallel()
			f := newFixture(t)
			newGuestSearch(t, f, state).check(t, state)
		})
	}
}
