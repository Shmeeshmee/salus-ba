// Package trends builds and serves the platform-wide anonymized figures (docs/plans/analytics.md §5 and §6, ADR
// 0013): distinct-patient counts per closed month (G×D, G×S, S×D) and quarter (the heatmap), suppressed as a whole
// (Suppress) and frozen once written, and the weekly flags of unusual rises. No table, type or response of this
// package has a clinic, practitioner or patient column (A1): a clinic and a patient exist here only as an int32
// inside one build, which discards them with the build's pseudonym key.
package trends

import "time"

// The tables of a period (§5): geography × diagnosis, geography × specialty, specialty × diagnosis (months) and
// the heatmap (quarters).
const (
	TableGD      = "gd"
	TableGS      = "gs"
	TableSD      = "sd"
	TableHeatmap = "hm"
)

// The kinds of period, each built once (A3).
const (
	KindMonth   = "month"
	KindQuarter = "quarter"
	KindWeek    = "week"
)

// The statuses of a built period: published, or refused because the audit still failed (A7).
const (
	StatusPublished = "published"
	StatusRefused   = "refused"
)

// Key is a cell's place in its period's tables, one node per dimension, "" for "all". A month's dimensions are
// geography, specialty and diagnosis; a quarter's geography, weekday and hour. A geography node is "e:<entity>",
// "r:<region>" or "c:<city>"; a diagnosis node "g:<group>", "s:<subgroup>" or "c:<ICD-10 code>"; a specialty its
// SNOMED CT code; a weekday "1" (Monday) to "7"; an hour "0" to "23".
type Key [3]string

// The dimensions of a Key.
const (
	DimGeo       = 0
	DimSpecialty = 1 // months
	DimDiagnosis = 2 // months
	DimWeekday   = 1 // quarters
	DimHour      = 2 // quarters
)

// Contribution is a cell inside a build: each clinic's distinct patients in it (a clinic and a patient as an
// int32 of this build only; patient lists sorted, without repeats). It never leaves the build.
type Contribution map[int32][]int32

// Row is one stored cell of a table: its dimensions as columns ("" for all), its rounded count, or none when it
// is suppressed.
type Row struct {
	Period, Table                    string
	Entity, Region, City             string
	Group, Subgroup, Code, Specialty string
	Weekday, Hour                    string
	Count                            int
	Suppressed                       bool
}

// Period is a built period.
type Period struct {
	Built     time.Time
	Period    string // 2026-10, 2026-Q3, 2026-W44
	Kind      string
	Status    string
	MinCount  int
	CodeLevel bool // false when the audit dropped the month's code level (A7)
}

// Flag is an unusual rise of a diagnosis group in a leaf region in an ISO week (§6): no count.
type Flag struct {
	Week, Region, Group string
}

// Summary is what a build did, in counts only (A13).
type Summary struct {
	Cells, Suppressed, Rises int
	Refused, CodeDropped     bool
}
