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

// ReadRepository reads a Condition in the caller's scope, set on the transaction in ctx.
type ReadRepository interface {
	// SetClinic scopes the rest of the transaction in ctx to the clinic id.
	SetClinic(ctx context.Context, id string) error
	// SetPatient scopes the rest of the transaction in ctx to the patient id.
	SetPatient(ctx context.Context, id string) error
	// ByID returns the Condition id, ErrNotFound when the scope does not see it.
	ByID(ctx context.Context, id string) (Condition, error)
	// Confirmed returns, by document, the fields (key and code) of the signed documents documentIDs whose Condition
	// is confirmed: the clinic org's own, and a visit's shared with it when practitionerID has an appointment with its
	// patient there (history_confirmed_conditions). An id that is not a UUID matches none.
	Confirmed(ctx context.Context, org, practitionerID string, documentIDs []string) (map[string][]FieldCode, error)
}
