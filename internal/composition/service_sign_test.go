package composition_test

import (
	"bytes"
	"crypto/sha1" //nolint:gosec // R4's Attachment.hash is SHA-1
	"encoding/json"
	"errors"
	"io"
	"slices"
	"testing"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/composition"
	"github.com/Shmeeshmee/salus/pkg/envelope"
	"github.com/Shmeeshmee/salus/pkg/fhir"
)

// drafted is a generated draft of the placeholder template: findings from seg1, tests from seg2, the rest
// missing.
func (f *fixture) drafted(t *testing.T) composition.Draft {
	t.Helper()
	tpl := f.template(t)
	f.drafter.Fields = filled()
	d, err := generate(t, f, pract, tpl.ID, "")
	if err != nil {
		t.Fatal(err)
	}
	f.audit.events, f.publisher.envelopes, f.publisher.streams = nil, nil, nil
	return d
}

// values are the practitioner's values: the findings as drafted, the tests edited, a control date typed, the
// diagnosis left empty.
func values() []composition.Value {
	return []composition.Value{
		{Key: "diagnosis", Value: ""},
		{Key: "findings", Value: "Placeholder findings"},
		{Key: "tests", Value: "Placeholder test one"},
		{Key: "control", Value: "2030-02-01"},
	}
}

// The encounter's practitioner signs the draft as they left it: a value as drafted keeps its sources, an edited
// or typed one has none, an emptied one is missing. The draft becomes the final Composition, signed now by the
// appointment's PractitionerRole, with a DocumentReference whose PDF is stored encrypted, and a Provenance; all
// three writes are audited, and the final draft is pushed on the stream.
func TestSign_MakesTheDraftFinalWithItsPDFAndProvenance(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	d := f.drafted(t)

	got, err := f.svc.Sign(t.Context(), pract, d.ID, values(), d.VersionID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != composition.StatusFinal || !got.SignedAt.Equal(now) || got.SignerRoleID != roleA1 || got.VersionID != 2 ||
		!fhir.IsUUID(got.DocumentReferenceID) || !fhir.IsUUID(got.ProvenanceID) || got.PDF.KEKID != "kek-test" {
		t.Fatalf("Sign() = %+v", got)
	}
	want := []composition.Field{
		{Key: "findings", Value: "Placeholder findings", SourceSegmentIDs: []string{seg1}},
		{Key: "tests", Value: "Placeholder test one"},
		{Key: "control", Value: "2030-02-01"},
		{Key: "diagnosis", Missing: true},
	}
	for i, w := range want {
		g := got.Fields[i].Field
		if g.Key != w.Key || g.Value != w.Value || g.Missing != w.Missing || !slices.Equal(g.SourceSegmentIDs, w.SourceSegmentIDs) {
			t.Errorf("field %d = %+v, want %+v", i, g, w)
		}
	}
	pdf := f.openPDF(t, got)
	if sum := sha1.Sum(pdf); !bytes.HasPrefix(pdf, []byte("%PDF-")) || int64(len(pdf)) != got.PDF.Size || !bytes.Equal(sum[:], got.PDF.SHA1) { //nolint:gosec // see the import
		t.Errorf("the stored PDF: %d bytes, want a PDF of %d with its SHA-1", len(pdf), got.PDF.Size)
	}
	if codes := f.codes(); !slices.Equal(codes, []string{
		"update 0 Composition/" + d.ID, "create 0 DocumentReference/" + got.DocumentReferenceID, "create 0 Provenance/" + got.ProvenanceID,
	}) {
		t.Errorf("audit = %q", codes)
	}
	var e composition.Event
	if len(f.publisher.envelopes) != 1 || json.Unmarshal(f.publisher.envelopes[0].Payload, &e) != nil || e.Status != composition.StatusFinal || e.VersionID != 2 {
		t.Errorf("published %+v", f.publisher.envelopes)
	}
}

// openPDF reads the signed draft's PDF back from the store and decrypts it.
func (f *fixture) openPDF(t *testing.T, d composition.Draft) []byte {
	t.Helper()
	kek, _ := envelope.NewKEK("kek-test", bytes.Repeat([]byte{7}, 32))
	dek, err := kek.Unwrap(d.PDF.WrappedKey, d.DocumentReferenceID)
	if err != nil {
		t.Fatal(err)
	}
	r, err := envelope.Open(bytes.NewReader(f.store.objects[composition.ObjectKey(d.DocumentReferenceID)]), dek, d.DocumentReferenceID)
	if err != nil {
		t.Fatal(err)
	}
	b, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// A sign is refused, audited, with nothing stored or pushed: for a stale version, a draft already signed, values
// that do not match the fields or their types, a required field left empty, a caller other than the
// encounter's practitioner, a draft the caller does not see, and after a revocation.
func TestSign_Refusals(t *testing.T) {
	t.Parallel()
	edit := func(change func([]composition.Value) []composition.Value) []composition.Value {
		return change(values())
	}
	for name, tt := range map[string]struct {
		want    error
		id      string
		c       appointment.Caller
		values  []composition.Value
		keys    []string
		version int32
	}{
		"a stale version": {c: pract, values: values(), version: 2, want: composition.ErrVersionMismatch},
		"a field left out": {c: pract, values: edit(func(v []composition.Value) []composition.Value { return v[1:] }),
			version: 1, want: composition.ErrInvalidValues, keys: []string{"diagnosis"}},
		"an unknown field": {c: pract, values: edit(func(v []composition.Value) []composition.Value {
			return append(v, composition.Value{Key: "other", Value: "x"})
		}), version: 1, want: composition.ErrInvalidValues},
		"a field twice": {c: pract, values: edit(func(v []composition.Value) []composition.Value { return append(v, v[1]) }),
			version: 1, want: composition.ErrInvalidValues, keys: []string{"findings"}},
		"no date": {c: pract, values: edit(func(v []composition.Value) []composition.Value { v[3].Value = "soon"; return v }),
			version: 1, want: composition.ErrInvalidValues, keys: []string{"control"}},
		"required left empty": {c: pract, values: edit(func(v []composition.Value) []composition.Value { v[1].Value = " "; return v }),
			version: 1, want: composition.ErrRequiredMissing, keys: []string{"findings"}},
		"the admin":            {c: admin, values: values(), version: 1, want: composition.ErrForbidden},
		"another practitioner": {c: pract2, values: values(), version: 1, want: composition.ErrForbidden},
		"the patient":          {c: me, values: values(), version: 1, want: composition.ErrNotFound},
		"an unknown draft":     {c: pract, id: unknownID, values: values(), version: 1, want: composition.ErrNotFound},
	} {
		f := newFixture(t)
		d := f.drafted(t)
		id := tt.id
		if id == "" {
			id = d.ID
		}
		_, err := f.svc.Sign(t.Context(), tt.c, id, tt.values, tt.version)
		if !errors.Is(err, tt.want) {
			t.Errorf("%s: Sign() = %v, want %v", name, err, tt.want)
			continue
		}
		var fe *composition.FieldsError
		if tt.keys != nil && (!errors.As(err, &fe) || !slices.Equal(fe.Keys, tt.keys)) {
			t.Errorf("%s: Sign() = %v, want the keys %q", name, err, tt.keys)
		}
		if len(f.store.objects) != 0 || len(f.publisher.envelopes) != 0 || f.repo.drafts[d.ID].Status != composition.StatusPreliminary ||
			len(f.audit.events) != 1 || f.audit.events[0].Outcome == 0 {
			t.Errorf("%s: %d objects, %d published, status %s, audit %q", name, len(f.store.objects), len(f.publisher.envelopes), f.repo.drafts[d.ID].Status, f.codes())
		}
	}
}

// A draft is signed once: a second sign is refused, and a revocation, before or while the PDF is stored, signs
// nothing and leaves no object.
func TestSign_OnceAndNotAfterARevocation(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	d := f.drafted(t)
	signed, err := f.svc.Sign(t.Context(), pract, d.ID, values(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Sign(t.Context(), pract, d.ID, values(), signed.VersionID); !errors.Is(err, composition.ErrNotPreliminary) {
		t.Errorf("Sign() again = %v, want ErrNotPreliminary", err)
	}

	f = newFixture(t)
	d = f.drafted(t)
	f.encounters.calls[practAcc] = 0
	f.encounters.attendances[practAcc] = []appointment.Attendance{attendance(true, false, true), attendance(true, false, false)}
	if _, err := f.svc.Sign(t.Context(), pract, d.ID, values(), 1); !errors.Is(err, composition.ErrConsentRequired) {
		t.Errorf("Sign(revoked meanwhile) = %v, want ErrConsentRequired", err)
	}
	if len(f.store.objects) != 0 || f.repo.drafts[d.ID].Status != composition.StatusPreliminary {
		t.Errorf("%d objects, status %s; want none, preliminary", len(f.store.objects), f.repo.drafts[d.ID].Status)
	}
}

// A store that fails signs nothing.
func TestSign_AFailingStoreSignsNothing(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	d := f.drafted(t)
	f.store.putErr = errBackend
	if _, err := f.svc.Sign(t.Context(), pract, d.ID, values(), 1); !errors.Is(err, errBackend) {
		t.Errorf("Sign() = %v, want the failure", err)
	}
	f.store.putErr, f.repo.err = nil, errBackend
	if _, err := f.svc.Sign(t.Context(), pract, d.ID, values(), 1); !errors.Is(err, errBackend) {
		t.Errorf("Sign(database down) = %v, want the failure", err)
	}
	if f.repo.drafts[d.ID].Status != composition.StatusPreliminary || len(f.store.objects) != 0 {
		t.Errorf("status %s, %d objects", f.repo.drafts[d.ID].Status, len(f.store.objects))
	}
}
