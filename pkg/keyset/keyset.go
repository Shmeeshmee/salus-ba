// Package keyset is one page of a list read newest first by an instant, a reference breaking a tie (keyset
// pagination): the position the page starts after, a range of instants, and how many items at most. A reference
// is compared byte by byte (Go's string order, COLLATE "C" in Postgres), so every reader of a list orders it alike.
package keyset

import "time"

// Page asks for the items strictly after the position (At, Ref) in the order newest first, then reference
// descending, within [From, To), Limit at most. A zero At is the start; a zero From or To does not bound.
type Page struct {
	At    time.Time
	From  time.Time
	To    time.Time
	Ref   string
	Limit int
}
