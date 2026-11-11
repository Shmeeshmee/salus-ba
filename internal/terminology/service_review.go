package terminology

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

// The bounds of the review panel (api/openapi.yaml, listTermReview and the edits).
const (
	maxQueueLimit  = 100
	maxSynonyms    = 20
	maxNote        = 500 // characters
	maxExplanation = 500 // characters, as term_explanation's check
)

// statusLanguages are the languages a term's own status follows: Latin and the staff languages, the ones the
// cheatsheet shows its "Not reviewed yet" mark for (docs/terminology/SPEC.md §3 "Review").
var statusLanguages = []string{"la", "bs", "hr", "sr-Latn", "sr-Cyrl", "en"}

// The tables a review write is audited on: the term list is no FHIR resource (as document_template).
const (
	tableTranslation = "term_translation"
	tableExplanation = "term_explanation"
)

// Auditor records an audit event in the transaction in the context (auditevent.Service).
type Auditor interface {
	Record(ctx context.Context, e auditevent.AuditEvent) error
}

// ReviewPorts are what Reviews is built on.
type ReviewPorts struct {
	Repo  ReviewRepository
	Audit Auditor
	Tx    Transactor
	Clock clock.Clock
}

// Reviews is the review panel's rules (docs/terminology/SPEC.md §3 "Review", docs/plans/v1-finish.md "Contract"):
// every staff member reads the queue; only a reviewer, a permission the operator grants, edits a term's labels or
// explanation and approves them or marks them as needing changes. Editing an entry sets it back to unreviewed; a
// review names the reviewer and the time; every write is audited on its table, in its transaction.
type Reviews struct {
	p ReviewPorts
}

// NewReviews returns Reviews on p.
func NewReviews(p ReviewPorts) *Reviews { return &Reviews{p: p} }

// EditTranslation replaces the term's labels in language with edit, for a reviewer, at version (0 while the term
// has no label there). A change sets the translation to unreviewed, keeping its last review, and the term's status
// follows; the same labels change nothing and are not audited.
func (s *Reviews) EditTranslation(ctx context.Context, c Caller, termID, language string, edit LabelsEdit, version int32) (Translation, error) {
	if !c.Staff || !c.Reviewer {
		return Translation{}, s.refuse(ctx, c, tableTranslation)
	}
	labels, ok := editedLabels(language, edit)
	if !ok {
		return Translation{}, ErrInvalidReview
	}
	var out Translation
	err := s.write(ctx, c, termID, language, tableTranslation, func(ctx context.Context, st EntryState, now time.Time) (auditevent.Action, error) {
		tr, found := translationOf(st, termID, language)
		if tr.VersionID != version {
			return "", ErrVersionConflict
		}
		stored := slices.Clone(st.Labels)
		sortLabels(stored)
		if found && slices.Equal(stored, labels) {
			out = tr
			return "", nil
		}
		if err := s.p.Repo.ReplaceLabels(ctx, termID, language, labels); err != nil {
			return "", err
		}
		tr.Review.Status, tr.VersionID, tr.UpdatedAt = ReviewUnreviewed, tr.VersionID+1, now
		if err := s.putTranslation(ctx, c, st, tr, now); err != nil {
			return "", err
		}
		out = tr
		if !found {
			return auditevent.ActionCreate, nil
		}
		return auditevent.ActionUpdate, nil
	})
	return out, err
}

// ReviewTranslation approves the term's labels in language or marks them as needing changes, for a reviewer, at
// version: the reviewer and now are the review's, the note is d's; the term's status follows. ErrNoTranslation
// when the term has no label there.
func (s *Reviews) ReviewTranslation(ctx context.Context, c Caller, termID, language string, d Decision, version int32) (Translation, error) {
	if !c.Staff || !c.Reviewer {
		return Translation{}, s.refuse(ctx, c, tableTranslation)
	}
	d, err := decision(d)
	if err != nil || !slices.Contains(languages, language) {
		return Translation{}, cmpOr(err, ErrInvalidReview)
	}
	var out Translation
	err = s.write(ctx, c, termID, language, tableTranslation, func(ctx context.Context, st EntryState, now time.Time) (auditevent.Action, error) {
		tr, found := translationOf(st, termID, language)
		switch {
		case !found:
			return "", ErrNoTranslation
		case tr.VersionID != version:
			return "", ErrVersionConflict
		}
		tr.Review = Review{Status: d.Status, ReviewedBy: c.AccountID, ReviewedAt: now, Note: d.Note}
		tr.VersionID, tr.UpdatedAt = tr.VersionID+1, now
		out = tr
		return auditevent.ActionUpdate, s.putTranslation(ctx, c, st, tr, now)
	})
	return out, err
}

// WriteExplanation writes the term's explanation in language (an app language, never Latin), trimmed, 1 to 500
// characters, for a reviewer, at version (0 while there is none). A new or changed explanation is unreviewed,
// keeping its last review; the same text changes nothing and is not audited. Explanations do not change the
// term's status: the cheatsheet does not show them.
func (s *Reviews) WriteExplanation(ctx context.Context, c Caller, termID, language, text string, version int32) (Explanation, error) {
	if !c.Staff || !c.Reviewer {
		return Explanation{}, s.refuse(ctx, c, tableExplanation)
	}
	text = strings.TrimSpace(text)
	if n := utf8.RuneCountInString(text); !explanationLanguage(language) || n == 0 || n > maxExplanation || !utf8.ValidString(text) || strings.ContainsRune(text, 0) {
		return Explanation{}, ErrInvalidReview
	}
	var out Explanation
	err := s.write(ctx, c, termID, language, tableExplanation, func(ctx context.Context, st EntryState, now time.Time) (auditevent.Action, error) {
		e, found := explanationOf(st, termID, language)
		switch {
		case e.VersionID != version:
			return "", ErrVersionConflict
		case found && e.Text == text:
			out = e
			return "", nil
		}
		e.Text, e.Review.Status, e.VersionID, e.UpdatedAt = text, ReviewUnreviewed, e.VersionID+1, now
		out = e
		if err := s.p.Repo.PutExplanation(ctx, e); err != nil {
			return "", err
		}
		if !found {
			return auditevent.ActionCreate, nil
		}
		return auditevent.ActionUpdate, nil
	})
	return out, err
}

// ReviewExplanation approves the term's explanation in language or marks it as needing changes, for a reviewer, at
// version, as ReviewTranslation does. ErrNoExplanation when there is none.
func (s *Reviews) ReviewExplanation(ctx context.Context, c Caller, termID, language string, d Decision, version int32) (Explanation, error) {
	if !c.Staff || !c.Reviewer {
		return Explanation{}, s.refuse(ctx, c, tableExplanation)
	}
	d, err := decision(d)
	if err != nil || !explanationLanguage(language) {
		return Explanation{}, cmpOr(err, ErrInvalidReview)
	}
	var out Explanation
	err = s.write(ctx, c, termID, language, tableExplanation, func(ctx context.Context, st EntryState, now time.Time) (auditevent.Action, error) {
		e, found := explanationOf(st, termID, language)
		switch {
		case !found:
			return "", ErrNoExplanation
		case e.VersionID != version:
			return "", ErrVersionConflict
		}
		e.Review = Review{Status: d.Status, ReviewedBy: c.AccountID, ReviewedAt: now, Note: d.Note}
		e.VersionID, e.UpdatedAt = e.VersionID+1, now
		out = e
		return auditevent.ActionUpdate, s.p.Repo.PutExplanation(ctx, e)
	})
	return out, err
}

// write runs fn on the term's entry in language, locked, in one transaction with now as the time; fn returns the
// action it took, audited on table, or none when it changed nothing.
func (s *Reviews) write(ctx context.Context, c Caller, termID, language, table string,
	fn func(ctx context.Context, st EntryState, now time.Time) (auditevent.Action, error),
) error {
	err := s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		st, err := s.p.Repo.LockEntry(ctx, termID, language)
		if err != nil {
			return err
		}
		action, err := fn(ctx, st, s.p.Clock.Now())
		if err != nil || action == "" {
			return err
		}
		return s.p.Audit.Record(ctx, reviewEvent(c, table, action, auditevent.OutcomeSuccess))
	})
	if err != nil {
		return fmt.Errorf("terminology: review write: %w", err)
	}
	return nil
}

// refuse audits a write by c, who is no reviewer, as refused, and returns ErrForbidden.
func (s *Reviews) refuse(ctx context.Context, c Caller, table string) error {
	err := s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		return s.p.Audit.Record(ctx, reviewEvent(c, table, auditevent.ActionUpdate, auditevent.OutcomeMinorFailure))
	})
	if err != nil {
		return fmt.Errorf("terminology: audit a refused review write: %w", err)
	}
	return ErrForbidden
}

// putTranslation stores tr and sets the term's status from its translations with tr in place (followedStatus). A
// status that turns approved or needs_changes names c and now; one that turns unreviewed keeps the last review.
func (s *Reviews) putTranslation(ctx context.Context, c Caller, st EntryState, tr Translation, now time.Time) error {
	if err := s.p.Repo.PutTranslation(ctx, tr); err != nil {
		return err
	}
	status := followedStatus(st, tr)
	if status == st.TermReview.Status {
		return nil
	}
	r := st.TermReview
	r.Status = status
	if status != ReviewUnreviewed {
		r.ReviewedBy, r.ReviewedAt = c.AccountID, now
	}
	return s.p.Repo.SetTermReview(ctx, tr.TermID, r, now)
}

// followedStatus is the term's status from its translations in st with tr in place: needs_changes when one in
// statusLanguages needs changes, approved when all of those it has are approved, else unreviewed.
func followedStatus(st EntryState, tr Translation) string {
	approved, any := true, false
	for _, l := range statusLanguages {
		t, found := translationOf(st, tr.TermID, l)
		if l == tr.Language {
			t, found = tr, true
		}
		if !found {
			continue
		}
		if t.Review.Status == ReviewNeedsChanges {
			return ReviewNeedsChanges
		}
		any = true
		approved = approved && t.Review.Status == ReviewApproved
	}
	if any && approved {
		return ReviewApproved
	}
	return ReviewUnreviewed
}

// translationOf is st's translation in language, and false when there is none: then the unreviewed translation a
// first edit creates, at version 0.
func translationOf(st EntryState, termID, language string) (Translation, bool) {
	for _, t := range st.Translations {
		if t.Language == language {
			return t, true
		}
	}
	return Translation{TermID: termID, Language: language, Review: Review{Status: ReviewUnreviewed}}, false
}

// explanationOf is st's explanation, and false when there is none: then the unreviewed one a first write creates,
// at version 0.
func explanationOf(st EntryState, termID, language string) (Explanation, bool) {
	if st.Explanation != nil {
		return *st.Explanation, true
	}
	return Explanation{TermID: termID, Language: language, Review: Review{Status: ReviewUnreviewed}}, false
}

// editedLabels are edit's labels in language as the term list stores them (normaliseLabels), and false when they
// are no valid set: a preferred label is needed, at most 20 synonyms.
func editedLabels(language string, edit LabelsEdit) ([]Label, bool) {
	if !slices.Contains(languages, language) || len(edit.Synonyms) > maxSynonyms {
		return nil, false
	}
	in := []Label{{Language: language, Text: edit.Preferred, Preferred: true}}
	for _, syn := range edit.Synonyms {
		in = append(in, Label{Language: language, Text: syn})
	}
	for _, l := range in {
		if !utf8.ValidString(l.Text) || strings.ContainsRune(l.Text, 0) {
			return nil, false
		}
	}
	labels, err := normaliseLabels(in)
	if err != nil || len(labels) == 0 || !labels[0].Preferred {
		return nil, false
	}
	return labels, true
}

// decision is d with its note trimmed: ErrInvalidReview for a status other than approved or needs_changes or a
// note over 500 characters, ErrNoteRequired for needs_changes without one.
func decision(d Decision) (Decision, error) {
	d.Note = strings.TrimSpace(d.Note)
	switch {
	case d.Status != ReviewApproved && d.Status != ReviewNeedsChanges:
		return d, ErrInvalidReview
	case utf8.RuneCountInString(d.Note) > maxNote || !utf8.ValidString(d.Note) || strings.ContainsRune(d.Note, 0):
		return d, ErrInvalidReview
	case d.Status == ReviewNeedsChanges && d.Note == "":
		return d, ErrNoteRequired
	}
	return d, nil
}

// explanationLanguage reports whether language is an app language, the languages an explanation is written in.
func explanationLanguage(language string) bool {
	return language != "la" && slices.Contains(languages, language)
}

// cmpOr is err, else fallback.
func cmpOr(err, fallback error) error {
	if err != nil {
		return err
	}
	return fallback
}

// reviewEvent is the audit event of a review write by c on table: who, the interaction, the outcome; one row when
// it succeeded, none when it was refused.
func reviewEvent(c Caller, table string, action auditevent.Action, outcome auditevent.Outcome) auditevent.AuditEvent {
	code := auditevent.CodeUpdate
	if action == auditevent.ActionCreate {
		code = auditevent.CodeCreate
	}
	e := auditevent.AuditEvent{
		Type:           auditevent.Coding{System: auditevent.SystemAuditEventType, Code: auditevent.CodeRest},
		Subtype:        auditevent.Coding{System: auditevent.SystemRestfulInteraction, Code: code},
		Purpose:        auditevent.Coding{System: auditevent.SystemActReason, Code: auditevent.CodeHOPERAT},
		Action:         action,
		Outcome:        outcome,
		AgentAccountID: c.AccountID,
		OrganizationID: c.OrganizationID,
		EntityName:     table,
		EntityCount:    1,
	}
	if outcome != auditevent.OutcomeSuccess {
		e.EntityCount = 0
	}
	return e
}
