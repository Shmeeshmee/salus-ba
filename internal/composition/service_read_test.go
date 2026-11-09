package composition_test

import (
	"bytes"
	"errors"
	"io"
	"slices"
	"testing"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/composition"
)

// The encounter's practitioner reads its drafts, audited as a search of its encounter's documents; nobody else.
func TestDrafts_ReadsTheEncountersDraftsForItsPractitioner(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	d := f.drafted(t)
	got, err := f.svc.Drafts(t.Context(), pract, encA)
	if err != nil || len(got) != 1 || got[0].ID != d.ID {
		t.Fatalf("Drafts() = %+v, %v", got, err)
	}
	if codes := f.codes(); !slices.Equal(codes, []string{"search-type 0 Encounter/" + encA}) {
		t.Errorf("audit = %q", codes)
	}
	for name, tt := range map[string]struct {
		c    appointment.Caller
		id   string
		want error
	}{
		"admin":   {admin, encA, composition.ErrForbidden},
		"patient": {me, encA, composition.ErrForbidden},
		"unknown": {pract, unknownID, composition.ErrNotFound},
	} {
		if _, err := f.svc.Drafts(t.Context(), tt.c, tt.id); !errors.Is(err, tt.want) {
			t.Errorf("%s: Drafts() = %v, want %v", name, err, tt.want)
		}
	}
	if len(f.audit.events) != 4 {
		t.Errorf("audit = %q, want each refusal too", f.codes())
	}
}

// A signed document is read as its Composition, its DocumentReference, its Provenance and its PDF, each with
// its encounter, each audited as a read of that resource, by the encounter's practitioner only.
func TestRead_TheSignedDocumentsResources(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	d := f.drafted(t)
	signed, err := f.svc.Sign(t.Context(), pract, d.ID, values(), 1)
	if err != nil {
		t.Fatal(err)
	}
	f.audit.events = nil
	for name, read := range map[string]func(c appointment.Caller) (composition.Document, error){
		"Composition/" + d.ID: func(c appointment.Caller) (composition.Document, error) { return f.svc.Composition(t.Context(), c, d.ID) },
		"DocumentReference/" + signed.DocumentReferenceID: func(c appointment.Caller) (composition.Document, error) {
			return f.svc.DocumentReference(t.Context(), c, signed.DocumentReferenceID)
		},
		"Provenance/" + signed.ProvenanceID: func(c appointment.Caller) (composition.Document, error) {
			return f.svc.Provenance(t.Context(), c, signed.ProvenanceID)
		},
	} {
		f.audit.events = nil
		got, err := read(pract)
		if err != nil || got.Draft.ID != d.ID || got.Encounter.PatientID != patientA || !slices.Equal(f.codes(), []string{"read 0 " + name}) {
			t.Errorf("%s: read = %+v, %v; audit %q", name, got.Draft.ID, err, f.codes())
		}
		if _, err := read(pract2); !errors.Is(err, composition.ErrForbidden) {
			t.Errorf("%s: read by another practitioner = %v, want ErrForbidden", name, err)
		}
	}
	f.audit.events = nil
	r, size, err := f.svc.PDF(t.Context(), pract, signed.DocumentReferenceID)
	if err != nil {
		t.Fatal(err)
	}
	b, err := io.ReadAll(r)
	_ = r.Close()
	if err != nil || int64(len(b)) != size || size != signed.PDF.Size || !bytes.HasPrefix(b, []byte("%PDF-")) {
		t.Errorf("PDF() = %d bytes of %d, %v", len(b), size, err)
	}
	if !slices.Equal(f.codes(), []string{"read 0 DocumentReference/" + signed.DocumentReferenceID}) {
		t.Errorf("audit = %q", f.codes())
	}
	if _, _, err := f.svc.PDF(t.Context(), admin, signed.DocumentReferenceID); !errors.Is(err, composition.ErrForbidden) {
		t.Errorf("PDF(admin) = %v, want ErrForbidden", err)
	}
}

// A draft that is not signed has no DocumentReference or Provenance; unknown ids are not found, as is anything
// for a patient; the Composition of a preliminary draft is read.
func TestRead_NotFound(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	d := f.drafted(t)
	if got, err := f.svc.Composition(t.Context(), pract, d.ID); err != nil || got.Draft.Status != composition.StatusPreliminary {
		t.Errorf("Composition(preliminary) = %+v, %v", got, err)
	}
	for name, err := range map[string]error{
		"unknown composition": func() error { _, err := f.svc.Composition(t.Context(), pract, unknownID); return err }(),
		"patient":             func() error { _, err := f.svc.Composition(t.Context(), me, d.ID); return err }(),
		"unsigned reference":  func() error { _, err := f.svc.DocumentReference(t.Context(), pract, d.ID); return err }(),
		"unsigned provenance": func() error { _, err := f.svc.Provenance(t.Context(), pract, d.ID); return err }(),
		"unknown pdf":         func() error { _, _, err := f.svc.PDF(t.Context(), pract, unknownID); return err }(),
	} {
		if !errors.Is(err, composition.ErrNotFound) {
			t.Errorf("%s: %v, want ErrNotFound", name, err)
		}
	}
	f.repo.err = errBackend
	if _, err := f.svc.Composition(t.Context(), pract, d.ID); !errors.Is(err, errBackend) {
		t.Errorf("Composition(down) = %v, want the failure", err)
	}
}
