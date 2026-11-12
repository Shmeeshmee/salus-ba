package condition

import (
	"context"
	"time"
)

// Repository stores the Conditions.
type Repository interface {
	// Insert writes the Conditions in the transaction in ctx, in the clinic's scope; a second Condition of a
	// document's field is an error.
	Insert(ctx context.Context, cs []Condition) error
	// Diagnoses returns a clinic's confirmed Conditions recorded in [from, to), oldest first (salus_stats).
	Diagnoses(ctx context.Context, orgID string, from, to time.Time) ([]Diagnosis, error)
	// Unconfirmed returns the ids of the unconfirmed Conditions (the owner, for salusctl).
	Unconfirmed(ctx context.Context) ([]string, error)
}
