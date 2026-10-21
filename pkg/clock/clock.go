// Package clock lets code take the current time as a dependency, so tests can fix it.
package clock

import "time"

// Clock tells the time.
type Clock interface {
	Now() time.Time
}

// System is the real clock.
type System struct{}

// Now returns the current time.
func (System) Now() time.Time { return time.Now() }

// Fixed always returns T. Use it in tests.
type Fixed struct {
	T time.Time
}

// Now returns T.
func (f Fixed) Now() time.Time { return f.T }
