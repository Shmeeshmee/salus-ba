package diagnosticreport

import (
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"
)

// loinc is a LOINC code's form (http://loinc.org), as the migration checks it.
var loinc = regexp.MustCompile(`^[0-9]{1,7}-[0-9]$`)

// MaxResults is how many values one report holds (SaveDiagnosticReportRequest.results).
const MaxResults = 100

// validate reports ErrInvalid for a draft the contract and the migration refuse.
func (d Draft) validate() error {
	title := strings.TrimSpace(d.Title)
	if title == "" || utf8.RuneCountInString(title) > 200 || !code(d.LOINC) || len(d.Results) > MaxResults {
		return ErrInvalid
	}
	for _, r := range d.Results {
		if !r.valid() {
			return ErrInvalid
		}
	}
	return nil
}

func (r Result) valid() bool {
	switch {
	case strings.TrimSpace(r.Test) == "", utf8.RuneCountInString(r.Test) > 200, !code(r.LOINC):
		return false
	case (r.Value == nil) == (r.ValueText == ""), r.Unit != "" && r.Value == nil:
		return false
	case utf8.RuneCountInString(r.Unit) > 40, utf8.RuneCountInString(r.ValueText) > 200, utf8.RuneCountInString(r.ReferenceRange) > 100:
		return false
	}
	return r.Interpretation == "" || slices.Contains(Interpretations, r.Interpretation)
}

// code reports whether c is absent or a LOINC code.
func code(c string) bool { return c == "" || loinc.MatchString(c) }

// report is r with the draft's title, code, date and values.
func (d Draft) report(r Report) Report {
	r.Title, r.LOINC, r.EffectiveDate = strings.TrimSpace(d.Title), d.LOINC, d.EffectiveDate
	r.Results = slices.Clone(d.Results)
	if r.Results == nil {
		r.Results = []Result{}
	}
	return r
}
