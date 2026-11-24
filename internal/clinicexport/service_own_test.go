package clinicexport_test

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/clinicexport"
	"github.com/Shmeeshmee/salus/internal/consent"
	"github.com/Shmeeshmee/salus/internal/invoice"
	"github.com/Shmeeshmee/salus/internal/patient"
)

// ownSources is the patient's own reads, as each owning service gives them to the calling patient.
type ownSources struct {
	meErr        error
	me           patient.Patient
	consents     []consent.Consent
	visits       []appointment.Seen
	receipts     []invoice.Invoice
	profileCalls []patient.Caller
	consentCalls []consent.Caller
	visitCalls   []appointment.Caller
	receiptCalls []invoice.Caller
}

func (o *ownSources) Me(_ context.Context, c patient.Caller) (patient.Patient, error) {
	o.profileCalls = append(o.profileCalls, c)
	return o.me, o.meErr
}

func (o *ownSources) Mine(_ context.Context, c consent.Caller) ([]consent.Consent, error) {
	o.consentCalls = append(o.consentCalls, c)
	return o.consents, nil
}

func (o *ownSources) Search(_ context.Context, c appointment.Caller, _ appointment.Filter) ([]appointment.Seen, error) {
	o.visitCalls = append(o.visitCalls, c)
	return o.visits, nil
}

type ownReceipts struct{ o *ownSources }

func (r ownReceipts) Mine(_ context.Context, c invoice.Caller) ([]invoice.Invoice, error) {
	r.o.receiptCalls = append(r.o.receiptCalls, c)
	return r.o.receipts, nil
}

func newOwn() (*ownSources, *clinicexport.OwnRecords) {
	o := &ownSources{
		me: patient.Patient{ID: "pt1", AccountID: "acc1", LastUpdated: time.Date(2026, 11, 1, 8, 0, 0, 0, time.UTC)},
		consents: []consent.Consent{
			{ID: "c1", PatientID: "pt1", OrganizationID: clinicA}, {ID: "c2", PatientID: "pt1", OrganizationID: clinicB},
		},
		visits: []appointment.Seen{
			{Appointment: appointment.Appointment{ID: "a1", PatientID: "pt1", OrganizationID: clinicA, Status: appointment.StatusCancelled}},
			{Appointment: appointment.Appointment{ID: "a2", PatientID: "pt1", OrganizationID: clinicB}},
		},
		receipts: []invoice.Invoice{{ID: "i1", PatientID: "pt1", OrganizationID: clinicA}, {ID: "i2", PatientID: "pt1", OrganizationID: clinicB}},
	}
	return o, clinicexport.NewOwnRecords(clinicexport.OwnPorts{Profile: o, Consents: o, Appointments: o, Receipts: ownReceipts{o}})
}

// entries are the fullUrls of a Bundle's entries.
func entries(t *testing.T, body []byte) []string {
	t.Helper()
	var b struct {
		ResourceType string `json:"resourceType"`
		Type         string `json:"type"`
		Entry        []struct {
			FullURL string `json:"fullUrl"`
		} `json:"entry"`
	}
	if err := json.Unmarshal(body, &b); err != nil {
		t.Fatal(err)
	}
	if b.ResourceType != "Bundle" || b.Type != "collection" {
		t.Fatalf("a %s of type %s, want a collection Bundle", b.ResourceType, b.Type)
	}
	var out []string
	for _, e := range b.Entry {
		out = append(out, e.FullURL)
	}
	return out
}

// The patient's own export (docs/clinic-cms/WAVE5.md D2, S4): the hand-over's Bundle, read as the patient through the
// services they read in the app, from every clinic for the account target (the cancelled visits too, S11), from the
// one clinic for a clinic target.
func TestOwnRecords_BundlesThePatientsOwnReadsByTarget(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		org  string
		want []string
	}{
		"every clinic": {"", []string{"urn:uuid:pt1", "urn:uuid:c1", "urn:uuid:c2", "urn:uuid:a1", "urn:uuid:a2", "urn:uuid:i1", "urn:uuid:i2"}},
		"one clinic":   {clinicA, []string{"urn:uuid:pt1", "urn:uuid:c1", "urn:uuid:a1", "urn:uuid:i1"}},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			o, own := newOwn()
			body, err := own.Bundle(t.Context(), "acc1", tc.org)
			if err != nil {
				t.Fatal(err)
			}
			if got := entries(t, body); !slices.Equal(got, tc.want) {
				t.Errorf("entries %v, want %v", got, tc.want)
			}
			if c := o.profileCalls; len(c) != 1 || c[0] != (patient.Caller{AccountID: "acc1"}) {
				t.Errorf("profile read as %+v, want the patient", c)
			}
			if c := o.consentCalls; len(c) != 1 || c[0] != (consent.Caller{AccountID: "acc1", PatientID: "pt1"}) {
				t.Errorf("consents read as %+v, want the patient", c)
			}
			if c := o.visitCalls; len(c) != 1 || c[0] != (appointment.Caller{AccountID: "acc1", PatientID: "pt1"}) {
				t.Errorf("visits read as %+v, want the patient", c)
			}
			if c := o.receiptCalls; len(c) != 1 || c[0].AccountID != "acc1" || c[0].PatientID != "pt1" || c[0].OrganizationID != "" {
				t.Errorf("receipts read as %+v, want the patient", c)
			}
		})
	}
}

// A patient without a profile has no record: no Bundle, and nothing else is read.
func TestOwnRecords_NoProfileNoBundle(t *testing.T) {
	t.Parallel()
	o, own := newOwn()
	o.meErr = patient.ErrNotFound
	body, err := own.Bundle(t.Context(), "acc1", "")
	if err != nil || body != nil {
		t.Fatalf("Bundle = %q, %v; want none", body, err)
	}
	if len(o.consentCalls)+len(o.visitCalls)+len(o.receiptCalls) != 0 {
		t.Error("read more without a profile")
	}
	o.meErr = errors.New("down")
	if _, err := own.Bundle(t.Context(), "acc1", ""); err == nil {
		t.Error("no error when the profile cannot be read")
	}
}
