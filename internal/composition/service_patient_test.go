package composition_test

import (
	"bytes"
	"errors"
	"io"
	"slices"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/composition"
)

// signedOne drafts and signs one document as the encounter's practitioner, and forgets the audit of it.
func (f *fixture) signedOne(t *testing.T) composition.Draft {
	t.Helper()
	d, err := f.svc.Sign(t.Context(), pract, f.drafted(t).ID, values(), 1)
	if err != nil {
		t.Fatal(err)
	}
	f.audit.events = nil
	return d
}

// SAL-17: the encounter's patient lists its signed documents and reads each one's Composition, DocumentReference
// and PDF, each audited as the patient's own request (PATRQT) in the encounter's clinic; also after revoking the
// consent, as it is their own record.
func TestSigned_ThePatientListsAndReadsTheirSignedDocuments(t *testing.T) {
	t.Parallel()
	for name, active := range map[string]bool{"active consent": true, "revoked": false} {
		f := newFixture(t)
		signed := f.signedOne(t)
		revoked := attendance(false, true, active)
		revoked.RevokedAt = now.Add(-time.Minute)
		f.encounters.attendances[accountA] = []appointment.Attendance{revoked}

		list, err := f.svc.Signed(t.Context(), me, encA)
		if err != nil || len(list) != 1 || list[0].Draft.ID != signed.ID || list[0].Encounter.PatientID != patientA {
			t.Fatalf("%s: Signed() = %+v, %v; want the signed document", name, list, err)
		}
		if _, err := f.svc.Composition(t.Context(), me, signed.ID); err != nil {
			t.Errorf("%s: Composition() = %v", name, err)
		}
		if _, err := f.svc.DocumentReference(t.Context(), me, signed.DocumentReferenceID); err != nil {
			t.Errorf("%s: DocumentReference() = %v", name, err)
		}
		r, _, err := f.svc.PDF(t.Context(), me, signed.DocumentReferenceID)
		if err != nil {
			t.Fatalf("%s: PDF() = %v", name, err)
		}
		b, _ := io.ReadAll(r)
		_ = r.Close()
		if !bytes.HasPrefix(b, []byte("%PDF-")) {
			t.Errorf("%s: PDF() is no PDF", name)
		}
		want := []string{
			"search-type 0 Encounter/" + encA, "read 0 Composition/" + signed.ID,
			"read 0 DocumentReference/" + signed.DocumentReferenceID, "read 0 DocumentReference/" + signed.DocumentReferenceID,
		}
		if got := f.codes(); !slices.Equal(got, want) {
			t.Errorf("%s: audit = %q, want %q", name, got, want)
		}
		for _, e := range f.audit.events {
			if e.Purpose.Code != auditevent.CodePATRQT || e.OrganizationID != clinicA || e.AgentAccountID != accountA {
				t.Errorf("%s: audit event %+v, want the patient's own request in clinic A", name, e)
			}
		}
	}
}

// A draft that is not signed is never the patient's: not listed, and not found by its Composition; nor are its
// drafts searchable by them.
func TestSigned_NeverADraftForThePatient(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	d := f.drafted(t)
	list, err := f.svc.Signed(t.Context(), me, encA)
	if err != nil || len(list) != 0 {
		t.Errorf("Signed() with only a draft = %+v, %v; want none", list, err)
	}
	if _, err := f.svc.Composition(t.Context(), me, d.ID); !errors.Is(err, composition.ErrNotFound) {
		t.Errorf("Composition(draft) by the patient = %v, want ErrNotFound", err)
	}
	if _, err := f.svc.Drafts(t.Context(), me, encA); !errors.Is(err, composition.ErrForbidden) {
		t.Errorf("Drafts() by the patient = %v, want ErrForbidden", err)
	}
}

// The practitioner lists the signed documents too, with an active consent; the clinic's admin, another
// practitioner and an unknown encounter are refused, audited.
func TestSigned_ThePractitionerAndTheRefusals(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	signed := f.signedOne(t)
	_ = f.drafted(t) // an open draft of another template is not listed
	f.audit.events = nil
	list, err := f.svc.Signed(t.Context(), pract, encA)
	if err != nil || len(list) != 1 || list[0].Draft.ID != signed.ID {
		t.Errorf("Signed() by the practitioner = %+v, %v; want the signed one", list, err)
	}
	for _, tt := range []struct {
		want error
		name string
		id   string
		c    appointment.Caller
	}{
		{composition.ErrForbidden, "admin", encA, admin},
		{composition.ErrForbidden, "other", encA, pract2},
		{composition.ErrNotFound, "unknown", unknownID, me},
	} {
		if _, err := f.svc.Signed(t.Context(), tt.c, tt.id); !errors.Is(err, tt.want) {
			t.Errorf("%s: Signed() = %v, want %v", tt.name, err, tt.want)
		}
	}
	want := []string{"search-type 0 Encounter/" + encA, "search-type 4 Encounter/" + encA, "search-type 4 Encounter/" + encA, "search-type 4 "}
	if got := f.codes(); !slices.Equal(got, want) {
		t.Errorf("audit = %q, want %q", got, want)
	}
	f.repo.err = errBackend
	if _, err := f.svc.Signed(t.Context(), pract, encA); !errors.Is(err, errBackend) {
		t.Errorf("Signed(down) = %v, want the failure", err)
	}
}

// A signature enqueues the patient's e-mail in its final transaction, with the appointment's and clinic's ids
// only; a refused or failed signature enqueues nothing.
func TestSign_EnqueuesThePatientsEmail(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	d := f.drafted(t)
	if _, err := f.svc.Sign(t.Context(), pract2, d.ID, values(), 1); err == nil {
		t.Fatal("Sign() by another practitioner succeeded")
	}
	f.jobs.err = errBackend
	if _, err := f.svc.Sign(t.Context(), pract, d.ID, values(), 1); !errors.Is(err, errBackend) {
		t.Errorf("Sign() with the queue down = %v, want the failure", err)
	}
	if len(f.jobs.signed) != 0 {
		t.Errorf("jobs = %q after a refusal and a failure, want none", f.jobs.signed)
	}
	f.jobs.err = nil
	if _, err := f.svc.Sign(t.Context(), pract, d.ID, values(), 1); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(f.jobs.signed, []string{apptA + " " + clinicA}) {
		t.Errorf("jobs = %q, want the appointment's e-mail", f.jobs.signed)
	}
}
