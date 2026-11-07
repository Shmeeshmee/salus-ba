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
