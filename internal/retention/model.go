// Package retention says how long each table with personal data is kept and what happens after
// (docs/accounts/SPEC.md "Retention hook"). The rules are data; archcheck checks that every table marked
// personal_data in ownership.yaml has one. The scheduled job that applies them is the next privacy feature.
package retention

import "time"

// Then is what happens to a row once its period is over.
type Then string

// What can happen.
const (
	// ThenRemoveRange removes whole time ranges of rows at once, as the table's owner (the application role
	// cannot delete them).
	ThenRemoveRange Then = "remove-range"
)

// Rule is the retention of one table.
type Rule struct {
	Table string        // the table, as in ownership.yaml
	Basis string        // why the rows are kept that long: the legal ground
	Then  Then          // what happens after Keep
	Keep  time.Duration // how long a row is kept
}
