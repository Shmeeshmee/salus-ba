// Package terminology is the medical term list (docs/terminology/SPEC.md): concepts of anatomy, diagnoses,
// symptoms, procedures and drugs with their codes and their labels in Latin and the app languages, seeded from
// Wikidata and curated by the owner's doctor partner. Reference data; the reviews name the reviewer's account.
package terminology

import "time"

// The review statuses of a term, a translation and an explanation (review_status, 00037): one status for the
// cheatsheet's "Not reviewed yet" mark, the curator's file and the review panel (docs/plans/v1-finish.md).
const (
	ReviewUnreviewed   = "unreviewed"
	ReviewApproved     = "approved"
	ReviewNeedsChanges = "needs_changes"
)

// Review is where an entry is in its review: its status, when it was last set to approved or needs changes, by
// whom (an account id; empty when the curator's file set it), and the reviewer's note.
type Review struct {
	ReviewedAt time.Time // zero until the first review; kept when the entry goes back to unreviewed
	Status     string
	ReviewedBy string
	Note       string
}

// Translation is the review of a term's labels in one language. A term and language with labels and no stored
// translation is unreviewed.
type Translation struct {
	UpdatedAt time.Time
	Review    Review
	TermID    string
	Language  string
	VersionID int32
}

// Explanation is a short plain-language explanation of a term in an app language (never Latin), at most 500
// characters: informational, never a diagnosis.
type Explanation struct {
	UpdatedAt time.Time
	Review    Review
	TermID    string
	Language  string
	Text      string
	VersionID int32
}

// Term is one concept with its codes. A code is empty when the term has none; several codes of one kind are
// joined with ", ".
type Term struct {
	ReviewedAt   time.Time // zero until the term's first review
	ID           string
	ReviewedBy   string // the account that set the status in the review panel; empty when the curator's file did
	Category     string
	WikidataID   string // e.g. Q9368
	ICD10        string
	TA98         string // Terminologia Anatomica 98
	ATC          string
	PCS          string // ICD-10-PCS or ICD-9-CM procedure code
	ReviewStatus string
	Note         string // the curator's
	Labels       []Label
	Rank         int32 // the Wikidata item's sitelinks: higher is more common
}

// Label is one name of a term in one language: the preferred one, or a synonym.
type Label struct {
	Language  string
	Text      string
	Preferred bool
}

// Search is what a search asks for: text matched against every label in every language, an optional
// category, the languages whose labels come back, and how many terms at most.
type Search struct {
	Text       string
	Category   string // empty for any
	UILanguage string // the caller's UI language: its labels and Latin rank first within a match type
	Languages  []string
	Limit      int32
}

// Caller is who searches or reviews: their account, the session's clinic (for the audit), whether they are staff of
// a clinic (its admin or one of its practitioners; a patient is not), and whether the operator made them a reviewer
// of the term list (docs/plans/v1-finish.md "Decisions").
type Caller struct {
	AccountID      string
	OrganizationID string
	Staff          bool
	Reviewer       bool
}

// Match is a term as a search returns it: Term without its Labels, which Labels holds grouped per requested
// language.
type Match struct {
	Labels []Names
	Term   Term
}

// Names are a term's labels in one language: the preferred one (the marked label, else the first
// alphabetically) and the others alphabetically.
type Names struct {
	Language  string
	Preferred string
	Synonyms  []string
}

// ImportRow is one term of an import file and the line it starts on.
type ImportRow struct {
	Term Term
	Line int
}

// RowError is a row of an import file that was refused, and why. The reason names a column, never a value.
type RowError struct {
	ID     string // the row's term id, when it has one
	Reason string
	Line   int
}

// ImportReport is what an import did: rows that created a term, changed one, changed nothing, and those refused.
type ImportReport struct {
	Refused   []RowError
	Created   int
	Updated   int
	Unchanged int
}

// CategoryCount is how many terms a category holds.
type CategoryCount struct {
	Category string
	Terms    int
}

// GlossaryQuery is what a line's glossary looks up: the phrases of the line that may be terms (Candidates),
// matched against the labels in From, case and accents aside, each found term with its label in To; at most
// Limit of them.
type GlossaryQuery struct {
	From       string
	To         string
	Candidates []string
	Limit      int32
}

// GlossaryEntry is a term found in a line: its label as the line's language has it, and its preferred label in
// the language the line is translated into.
type GlossaryEntry struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

// QueueQuery is what a page of the review queue asks for: the entries of the terms in Language, of review status
// Status (any when empty), after the page whose Next is After (the first page when empty), at most Limit.
type QueueQuery struct {
	Language string
	Status   string
	After    string
	Limit    int32
}

// QueuePage is a page of the review queue and the After of the next one, empty on the last.
type QueuePage struct {
	Next    string
	Entries []QueueEntry
}

// QueueEntry is a term in the queue's language: its labels there, then in Latin and English, the review of its
// labels in the language (nil while it has none) and its explanation there (nil while there is none), and the
// entry's status (docs/terminology/SPEC.md §3 "Review").
type QueueEntry struct {
	Translation *Translation
	Explanation *Explanation
	Status      string
	Labels      []Names
	Term        Term // without Labels
}

// QueueFilter is a queue page as the repository reads it: the entries of Status (any when empty) after the entry
// of AfterStatus, AfterRank and AfterID (from the first when AfterID is empty).
type QueueFilter struct {
	Language    string
	Status      string
	AfterStatus string
	AfterID     string
	AfterRank   int32
	Limit       int32
}

// QueueRow is an entry as the repository reads it: the term with its labels in the language, Latin and English.
type QueueRow struct {
	Translation *Translation
	Explanation *Explanation
	Status      string
	Term        Term
}

// LanguageProgress is how far the review is in one language: how many terms, and how many of their entries in the
// language are approved or need changes.
type LanguageProgress struct {
	Language     string
	Terms        int32
	Approved     int32
	NeedsChanges int32
}

// LabelsEdit is a reviewer's edit of a term's labels in one language.
type LabelsEdit struct {
	Preferred string
	Synonyms  []string
}

// Decision is a reviewer's approval of an entry or its mark as needing changes, with the note.
type Decision struct {
	Status string // approved or needs_changes
	Note   string
}

// EntryState is a term and language as a review write reads them, the term locked for the transaction: the term's
// own review, its labels in the language, every translation of the term, and its explanation in the language (nil
// when there is none).
type EntryState struct {
	Explanation  *Explanation
	TermReview   Review
	Labels       []Label
	Translations []Translation
}

// Text is a text to tag (SAL-13) and the app language it is written in: a transcript line or a document's line, which
// is patient data, never logged, stored or put into an error.
type Text struct {
	Language string
	Body     string
}

// Span is a term found in a text: where, as UTF-16 offsets [Start, End) (the web and the mobile app index strings so),
// and the term. A span is an overlay on the text: the text itself is never changed.
type Span struct {
	TermID string `json:"term_id"`
	Start  int    `json:"start"`
	End    int    `json:"end"`
}

// Candidate is a phrase of a text that may be a term, in the text's language.
type Candidate struct {
	Language string
	Phrase   string
}

// CandidateTerm is a candidate that names a term, as the repository found it.
type CandidateTerm struct {
	Language string
	Phrase   string
	TermID   string
}
