// Package terminology is the medical term list (docs/terminology/SPEC.md): concepts of anatomy, diagnoses,
// symptoms, procedures and drugs with their codes and their labels in Latin and the app languages, seeded from
// Wikidata and curated by the owner's doctor partner. Reference data: no personal data, read-only for the app.
package terminology

import "time"

// Term is one concept with its codes. A code is empty when the term has none; several codes of one kind are
// joined with ", ".
type Term struct {
	ReviewedAt   time.Time // zero while the term is a draft
	ID           string
	Category     string
	WikidataID   string // e.g. Q9368
	ICD10        string
	TA98         string // Terminologia Anatomica 98
	ATC          string
	PCS          string // ICD-10-PCS or ICD-9-CM procedure code
	ReviewStatus string
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
	Text      string
	Category  string // empty for any
	Languages []string
	Limit     int32
}

// Caller is who searches: their account, and whether they are staff of a clinic (its admin or one of its
// practitioners); a patient is not.
type Caller struct {
	AccountID string
	Staff     bool
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
