package terminology

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// Queue returns a page of the review queue of q.Language for staff: unreviewed entries first, then those that need
// changes, then the approved ones, each the more common term first. ErrInvalidReview for an unknown language or
// status, a limit outside 1 to 100 or an After that is no page token. Not audited: reference data.
func (s *Reviews) Queue(ctx context.Context, c Caller, q QueueQuery) (QueuePage, error) {
	if !c.Staff {
		return QueuePage{}, ErrForbidden
	}
	f, ok := queueFilter(q)
	if !ok {
		return QueuePage{}, ErrInvalidReview
	}
	rows, err := s.p.Repo.Queue(ctx, f)
	if err != nil {
		return QueuePage{}, fmt.Errorf("terminology: review queue: %w", err)
	}
	langs := []string{q.Language}
	for _, l := range []string{"la", "en"} {
		if l != q.Language {
			langs = append(langs, l)
		}
	}
	page := QueuePage{Entries: make([]QueueEntry, len(rows))}
	for i, r := range rows {
		m := match(r.Term, langs)
		page.Entries[i] = QueueEntry{Translation: r.Translation, Explanation: r.Explanation, Status: r.Status, Labels: m.Labels, Term: m.Term}
	}
	if n := len(rows); n > 0 && n == int(q.Limit) {
		last := rows[n-1]
		page.Next = last.Status + "~" + strconv.Itoa(int(last.Term.Rank)) + "~" + last.Term.ID
	}
	return page, nil
}

// termIDForm is the form of a term id (a UUID) in a page token.
var termIDForm = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// queueFilter checks q and reads its page token, "<status>~<rank>~<term id>" of the last entry of the page before.
func queueFilter(q QueueQuery) (QueueFilter, bool) {
	f := QueueFilter{Language: q.Language, Status: q.Status, Limit: q.Limit}
	if !slices.Contains(languages, q.Language) || (q.Status != "" && !validStatus(q.Status)) || q.Limit < 1 || q.Limit > maxQueueLimit {
		return f, false
	}
	if q.After == "" {
		return f, true
	}
	parts := strings.SplitN(q.After, "~", 3)
	if len(parts) != 3 || !validStatus(parts[0]) || !termIDForm.MatchString(parts[2]) {
		return f, false
	}
	rank, err := strconv.ParseInt(parts[1], 10, 32)
	if err != nil {
		return f, false
	}
	f.AfterStatus, f.AfterRank, f.AfterID = parts[0], int32(rank), parts[2]
	return f, true
}

// Progress returns, for staff, how far the review is in every term language, in the enum's order. Not audited.
func (s *Reviews) Progress(ctx context.Context, c Caller) ([]LanguageProgress, error) {
	if !c.Staff {
		return nil, ErrForbidden
	}
	rows, err := s.p.Repo.Progress(ctx)
	if err != nil {
		return nil, fmt.Errorf("terminology: review progress: %w", err)
	}
	var terms int32
	for _, r := range rows {
		terms = max(terms, r.Terms)
	}
	out := make([]LanguageProgress, len(languages))
	for i, l := range languages {
		out[i] = LanguageProgress{Language: l, Terms: terms}
		if j := slices.IndexFunc(rows, func(r LanguageProgress) bool { return r.Language == l }); j >= 0 {
			out[i] = rows[j]
		}
	}
	return out, nil
}
