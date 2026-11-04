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
	// ThenKeep keeps the row while its subject exists: no period runs (Keep is zero). Removing it is a
	// decision of the subject-rights feature (closure, erasure), not of the retention job.
	ThenKeep Then = "keep"
	// ThenDelete deletes each row once its period has run from the moment the row stopped being used (for a
	// session: from revoked_at or expires_at, whichever comes first).
	ThenDelete Then = "delete"
	// ThenRestrict keeps the row while it is active and restricts it, not deletes it, once it is not (for a
	// practitioner: after leaving the clinic), for as long as clinical records reference it. No period runs
	// (Keep is zero): the end of the restriction follows the records that reference the row.
	ThenRestrict Then = "restrict"
)

// Rule is the retention of one table.
type Rule struct {
	Table string        // the table, as in ownership.yaml
	Basis string        // why the rows are kept that long: the legal ground
	Then  Then          // what happens after Keep
	Keep  time.Duration // how long a row is kept; zero for ThenKeep and ThenRestrict
}
