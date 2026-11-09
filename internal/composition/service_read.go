package composition

import (
	"context"
	"io"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/encounter"
	"github.com/Shmeeshmee/salus/pkg/envelope"
	"github.com/Shmeeshmee/salus/pkg/fhir"
)

// Document is a draft with its encounter, which gives its FHIR resources their subject, author and custodian.
type Document struct {
	Encounter encounter.Encounter
	Draft     Draft
}

// Drafts returns the encounter's drafts, oldest first, to its practitioner with the patient's active consent,
// audited as a search of the encounter's documents, also when refused: ErrNotFound, ErrForbidden,
// ErrConsentRequired.
func (s *Service) Drafts(ctx context.Context, c appointment.Caller, encounterID string) ([]Draft, error) {
	var out []Draft
	var refused error
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		e := entry{c: c, interaction: auditevent.CodeSearchType}
		enc, err := s.allowed(ctx, c, encounterID)
		e.reference = encounterRef(enc.ID)
		if refused = refusal(err); refused != nil {
			e.outcome = auditevent.OutcomeMinorFailure
			return s.record(ctx, e)
		}
		if err != nil {
			return err
		}
		if out, err = s.Repo.Drafts(ctx, enc.ID); err != nil {
			return err
		}
		e.org = enc.OrganizationID
		return s.record(ctx, e)
	})
	if err := wrapErr("drafts", err, refused); err != nil {
		return nil, err
	}
	if out == nil {
		out = []Draft{}
	}
	return out, nil
}

// Composition returns the draft id, preliminary or final, to its encounter's practitioner with the patient's
// active consent, audited, also when refused: ErrNotFound, ErrForbidden, ErrConsentRequired.
func (s *Service) Composition(ctx context.Context, c appointment.Caller, id string) (Document, error) {
	return s.read(ctx, c, "Composition", id, func(ctx context.Context, id string) (Draft, error) { return s.Repo.Draft(ctx, id, false) })
}

// DocumentReference returns the signed draft whose DocumentReference is id, under Composition's rule.
func (s *Service) DocumentReference(ctx context.Context, c appointment.Caller, id string) (Document, error) {
	return s.read(ctx, c, "DocumentReference", id, s.Repo.ByDocumentReference)
}

// Provenance returns the signed draft whose Provenance is id, under Composition's rule.
func (s *Service) Provenance(ctx context.Context, c appointment.Caller, id string) (Document, error) {
	return s.read(ctx, c, "Provenance", id, s.Repo.ByProvenance)
}

// PDF returns the plaintext PDF of the DocumentReference id and its size, decrypted as it is read, under
// Composition's rule, audited as a read of the DocumentReference. The caller closes the reader.
func (s *Service) PDF(ctx context.Context, c appointment.Caller, id string) (io.ReadCloser, int64, error) {
	doc, err := s.DocumentReference(ctx, c, id)
	if err != nil {
		return nil, 0, err
	}
	d := doc.Draft
	dek, err := s.KEK.Unwrap(d.PDF.WrappedKey, d.DocumentReferenceID)
	if err != nil {
		return nil, 0, wrapErr("pdf", err, nil)
	}
	src, err := s.Store.GetRange(ctx, ObjectKey(d.DocumentReferenceID), 0, envelope.CiphertextSize(d.PDF.Size)-1)
	if err != nil {
		return nil, 0, wrapErr("pdf", err, nil)
	}
	plain, err := envelope.Open(src, dek, d.DocumentReferenceID)
	if err != nil {
		_ = src.Close()
		return nil, 0, wrapErr("pdf", err, nil)
	}
	return readCloser{Reader: plain, Closer: src}, d.PDF.Size, nil
}

type readCloser struct {
	io.Reader
	io.Closer
}

// read finds a draft by find and returns it with its encounter when the caller may read it, audited as a read
// of the resource typ/id, a refusal too.
func (s *Service) read(ctx context.Context, c appointment.Caller, typ, id string, find func(ctx context.Context, id string) (Draft, error)) (Document, error) {
	var out Document
	var refused error
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		e := entry{c: c, interaction: auditevent.CodeRead}
		d, enc, err := s.reach(ctx, c, id, find)
		if refused = refusal(err); refused != nil {
			e.outcome = auditevent.OutcomeMinorFailure
			if fhir.IsUUID(id) { // the resource asked for, as other refusals are audited; not an arbitrary path
				e.reference = typ + "/" + id
			}
			return s.record(ctx, e)
		}
		if err != nil {
			return err
		}
		out = Document{Draft: d, Encounter: enc}
		e.reference, e.org = typ+"/"+id, enc.OrganizationID
		return s.record(ctx, e)
	})
	return out, wrapErr("read", err, refused)
}
