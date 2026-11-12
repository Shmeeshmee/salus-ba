package condition

import (
	"context"
	"crypto/rand"
	"fmt"
	"regexp"
	"slices"
	"time"

	"github.com/Shmeeshmee/salus/pkg/fhir"
	"github.com/Shmeeshmee/salus/pkg/pseudonym"
)

var icd10 = regexp.MustCompile(`^[A-Z][0-9]{2}(\.[0-9A-Z]{1,4})?$`)

// Service holds the Conditions' rules. It makes no access decision of its own: Record is called by composition
// inside its signing transaction, after its access check, and composition audits each Condition with the
// signature; Recorded serves the aggregate builds on salus_stats (they audit their build); Unconfirmed serves
// salusctl on the owner's connection.
type Service struct {
	repo Repository
}

// NewService returns the Service on repo.
func NewService(repo Repository) *Service { return &Service{repo: repo} }

// Record writes the Conditions of a signed document in the transaction in ctx, each with a new id when it has
// none; ErrInvalid, and nothing written, when one is not a Condition the model takes.
func (s *Service) Record(ctx context.Context, cs []Condition) error {
	if len(cs) == 0 {
		return nil
	}
	cs = slices.Clone(cs)
	for i := range cs {
		if cs[i].ID == "" {
			cs[i].ID = newID()
		}
		if !valid(cs[i]) {
			return ErrInvalid
		}
	}
	if err := s.repo.Insert(ctx, cs); err != nil {
		return fmt.Errorf("condition: record: %w", err)
	}
	return nil
}

func valid(c Condition) bool {
	for _, id := range []string{c.ID, c.OrganizationID, c.PatientID, c.EncounterID, c.DocumentID, c.PractitionerRoleID} {
		if !fhir.IsUUID(id) {
			return false
		}
	}
	return icd10.MatchString(c.Code) && c.FieldKey != "" && !c.RecordedAt.IsZero() &&
		(c.VerificationStatus == VerificationConfirmed || c.VerificationStatus == VerificationUnconfirmed)
}

// Recorded returns a clinic's confirmed Conditions recorded in [from, to), oldest first, each patient as
// HMAC-SHA256 of their id under key: the same patient gives the same pseudonym within a build, nothing links it
// back once the build discards the key. Without a key the pseudonym is zero.
func (s *Service) Recorded(ctx context.Context, orgID string, from, to time.Time, key []byte) ([]Recorded, error) {
	ds, err := s.repo.Diagnoses(ctx, orgID, from, to)
	if err != nil {
		return nil, fmt.Errorf("condition: recorded: %w", err)
	}
	out := make([]Recorded, len(ds))
	for i, d := range ds {
		out[i] = Recorded{RecordedAt: d.RecordedAt, PractitionerRoleID: d.PractitionerRoleID, Code: d.Code}
		if key != nil {
			out[i].Patient = pseudonym.Of(key, d.PatientID)
		}
	}
	return out, nil
}

// Unconfirmed returns the ids of the unconfirmed Conditions: the ones backfilled from documents signed before
// the confirmation existed (salusctl conditions unconfirmed).
func (s *Service) Unconfirmed(ctx context.Context) ([]string, error) {
	ids, err := s.repo.Unconfirmed(ctx)
	if err != nil {
		return nil, fmt.Errorf("condition: unconfirmed: %w", err)
	}
	return ids, nil
}

// newID returns a random (version 4) UUID.
func newID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
