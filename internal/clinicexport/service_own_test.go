package clinicexport_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/binary"
	"github.com/Shmeeshmee/salus/internal/clinicexport"
	"github.com/Shmeeshmee/salus/internal/communication"
	"github.com/Shmeeshmee/salus/internal/composition"
	"github.com/Shmeeshmee/salus/internal/consent"
	"github.com/Shmeeshmee/salus/internal/diagnosticreport"
	"github.com/Shmeeshmee/salus/internal/encounter"
	"github.com/Shmeeshmee/salus/internal/invoice"
	"github.com/Shmeeshmee/salus/internal/patient"
	"github.com/Shmeeshmee/salus/internal/questionnaire"
	"github.com/Shmeeshmee/salus/internal/questionnaireresponse"
)

// ownSources is the patient's own reads, as each owning service gives them to the calling patient, and every call.
type ownSources struct {
	meErr      error
	me         patient.Patient
	consents   []consent.Consent
	visits     []appointment.Seen
	receipts   []invoice.Invoice
	quotes     []invoice.Quote
	labs       []diagnosticreport.Report
	encs       []encounter.Encounter
	docs       []composition.Document
	answers    []questionnaireresponse.Seen
	files      map[string]string // clean files by binary id; any other is not found
	convs      []communication.Conversation
	msgs       map[string][]communication.Message // by conversation
	chatFiles  map[string]string                  // the chat's clean files by attachment id
	unreadable map[string]bool                    // the ids whose file or PDF cannot be opened
	calls      []string
	visitOrgs  []string
}

// opened is the body of the file or PDF id, or an error when it cannot be opened.
func (o *ownSources) opened(body, id string) (io.ReadCloser, error) {
	if o.unreadable[id] {
		return nil, errors.New("synthetic: the key was rotated")
	}
	return io.NopCloser(strings.NewReader(body)), nil
}

// ownLabs is the patient's released lab results (diagnosticreport.Service).
type ownLabs struct{ o *ownSources }

func (l ownLabs) MineAt(_ context.Context, c diagnosticreport.Patient, org string) ([]diagnosticreport.Report, error) {
	var out []diagnosticreport.Report
	for _, r := range l.o.labs {
		if ofClinic(r.OrganizationID, org) && c.PatientID == r.PatientID {
			l.o.call("lab result " + r.ID)
			out = append(out, r)
		}
	}
	return out, nil
}

// ofClinic is whether id is org's, or org is empty.
func ofClinic(id, org string) bool { return org == "" || id == org }

func (o *ownSources) call(s string) { o.calls = append(o.calls, s) }

func (o *ownSources) Me(_ context.Context, c patient.Caller) (patient.Patient, error) {
	o.call("profile " + c.AccountID)
	return o.me, o.meErr
}

func (o *ownSources) MineAt(_ context.Context, c consent.Caller, org string) ([]consent.Consent, error) {
	var out []consent.Consent
	for _, x := range o.consents {
		if ofClinic(x.OrganizationID, org) {
			o.call("consent " + x.ID)
			out = append(out, x)
		}
	}
	return out, nil
}

func (o *ownSources) Search(_ context.Context, c appointment.Caller, f appointment.Filter) ([]appointment.Seen, error) {
	o.call("visits " + c.PatientID)
	o.visitOrgs = append(o.visitOrgs, f.OrganizationID)
	var out []appointment.Seen
	for _, v := range o.visits {
		if f.OrganizationID == "" || v.Appointment.OrganizationID == f.OrganizationID {
			out = append(out, v)
		}
	}
	return out, nil
}

type ownInvoices struct{ o *ownSources }

func (r ownInvoices) MineAt(_ context.Context, _ invoice.Caller, org string) ([]invoice.Invoice, error) {
	var out []invoice.Invoice
	for _, x := range r.o.receipts {
		if ofClinic(x.OrganizationID, org) {
			r.o.call("receipt " + x.ID)
			out = append(out, x)
		}
	}
	return out, nil
}

func (r ownInvoices) PDF(_ context.Context, c invoice.Caller, id string) ([]byte, error) {
	r.o.call("receipt pdf " + id)
	return r.pdf("%PDF receipt "+id, id, c)
}

func (r ownInvoices) MyQuotesAt(_ context.Context, _ invoice.Caller, org string) ([]invoice.Quote, error) {
	var out []invoice.Quote
	for _, x := range r.o.quotes {
		if ofClinic(x.OrganizationID, org) {
			r.o.call("quote " + x.ID)
			out = append(out, x)
		}
	}
	return out, nil
}

func (r ownInvoices) QuotePDF(_ context.Context, c invoice.Caller, id string) ([]byte, error) {
	r.o.call("quote pdf " + id)
	return r.pdf("%PDF quote "+id, id, c)
}

func (r ownInvoices) pdf(body, id string, c invoice.Caller) ([]byte, error) {
	if c.PatientID != "pt1" || c.OrganizationID != "" {
		return nil, invoice.ErrNotFound
	}
	rc, err := r.o.opened(body, id)
	if err != nil {
		return nil, err
	}
	return io.ReadAll(rc)
}

type ownEncounters struct{ o *ownSources }

func (e ownEncounters) Search(_ context.Context, c appointment.Caller, f encounter.Filter) ([]encounter.Encounter, error) {
	e.o.call("encounters " + f.AppointmentID)
	var out []encounter.Encounter
	for _, x := range e.o.encs {
		if x.AppointmentID == f.AppointmentID && c.PatientID == "pt1" {
			out = append(out, x)
		}
	}
	return out, nil
}

type ownDocuments struct{ o *ownSources }

func (d ownDocuments) Signed(_ context.Context, _ appointment.Caller, encounterID string) ([]composition.Document, error) {
	d.o.call("documents " + encounterID)
	var out []composition.Document
	for _, x := range d.o.docs {
		if x.Encounter.ID == encounterID {
			out = append(out, x)
		}
	}
	return out, nil
}

func (d ownDocuments) PDF(_ context.Context, _ appointment.Caller, id string) (io.ReadCloser, int64, error) {
	d.o.call("pdf " + id)
	rc, err := d.o.opened("%PDF "+id, id)
	return rc, 5, err
}

type ownAnswers struct{ o *ownSources }

func (a ownAnswers) Search(_ context.Context, _ appointment.Caller, f questionnaireresponse.Filter) ([]questionnaireresponse.Seen, error) {
	a.o.call("answers " + f.AppointmentID + " " + string(f.Status))
	var out []questionnaireresponse.Seen
	for _, x := range a.o.answers {
		if x.Response.AppointmentID == f.AppointmentID && x.Response.Status == f.Status {
			out = append(out, x)
		}
	}
	return out, nil
}

func (a ownAnswers) Read(_ context.Context, _ questionnaire.Caller, id string) (questionnaire.Questionnaire, error) {
	a.o.call("questionnaire " + id)
	return questionnaire.Questionnaire{ID: id, SeriesID: "s1", Version: 1, Items: []questionnaire.Item{{LinkID: "photo"}}}, nil
}

type ownFiles struct{ o *ownSources }

func (f ownFiles) Download(_ context.Context, c binary.Caller, id string) (binary.File, error) {
	f.o.call("file " + id)
	body, ok := f.o.files[id]
	if !ok || c.PatientID != "pt1" {
		return binary.File{}, binary.ErrNotFound
	}
	return binary.File{Binary: binary.Binary{ID: id, Size: int64(len(body))}}, nil
}

func (f ownFiles) Content(_ context.Context, file binary.File, from, to int64) (io.ReadCloser, error) {
	return f.o.opened(f.o.files[file.ID][from:to+1], file.ID)
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
		quotes:   []invoice.Quote{{ID: "q2", PatientID: "pt1", OrganizationID: clinicB}},
		labs: []diagnosticreport.Report{{ID: "l2", PatientID: "pt1", OrganizationID: clinicB, AppointmentID: "a2", Title: "Synthetic panel",
			Status: diagnosticreport.StatusFinal, ReleasedAt: &released, VersionID: 2,
			Results: []diagnosticreport.Result{{Test: "Synthetic marker", ValueText: "negative"}}}},
		encs: []encounter.Encounter{{ID: "e2", AppointmentID: "a2", PatientID: "pt1", OrganizationID: clinicB}},
		docs: []composition.Document{{
			Encounter: encounter.Encounter{ID: "e2", PatientID: "pt1", OrganizationID: clinicB},
			Draft:     composition.Draft{ID: "d2", EncounterID: "e2", DocumentReferenceID: "dr2", Status: composition.StatusFinal},
		}},
		answers: []questionnaireresponse.Seen{{Response: questionnaireresponse.Response{
			ID: "r2", AppointmentID: "a2", PatientID: "pt1", OrganizationID: clinicB, QuestionnaireID: "qv1",
			Status: questionnaireresponse.StatusCompleted,
			Attachments: []binary.Attachment{{LinkID: "photo", BinaryID: "b2", ContentType: "image/png"},
				{LinkID: "photo", BinaryID: "b3", ContentType: "image/png"}}, // b3 is not clean: the app cannot open it
		}, Level: access.Full}},
		files: map[string]string{"b2": "png bytes"},
	}
	inv := ownInvoices{o}
	return o, clinicexport.NewOwnRecords(clinicexport.OwnPorts{
		Profile: o, Consents: o, Appointments: o, Receipts: inv, Quotes: inv, Encounters: ownEncounters{o},
		Documents: ownDocuments{o}, Answers: ownAnswers{o}, Questionnaires: ownAnswers{o}, Files: ownFiles{o}, Chat: ownChat{o},
		LabResults: ownLabs{o},
	})
}

// released is when the synthetic lab result was released.
var released = time.Date(2026, 11, 3, 8, 0, 0, 0, time.UTC)

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

// names are a part's file names.
func names(p clinicexport.OwnPart) []string {
	var out []string
	for _, f := range p.Files {
		out = append(out, f.Name)
	}
	return out
}

func file(t *testing.T, p clinicexport.OwnPart, name string) []byte {
	t.Helper()
	for _, f := range p.Files {
		if f.Name == name {
			return f.Body
		}
	}
	t.Fatalf("no %s in %v", name, names(p))
	return nil
}

// The patient's own export (docs/clinic-cms/WAVE5.md S4, D7): one part per clinic with what the patient reads in the
// app there, read as the patient through the services the app reads (each audits it): the visits (the cancelled
// ones too, S11) with their encounters, the signed documents with their PDFs, the submitted answers with their clean
// files, consents, receipts and quotes, as a FHIR Bundle beside the files, named by opaque ids.
func TestOwnRecords_OnePartPerClinicWithTheClinicalContent(t *testing.T) {
	t.Parallel()
	o, own := newOwn()
	x, err := own.Export(t.Context(), "acc1", "")
	if err != nil || x == nil {
		t.Fatalf("Export = %+v, %v", x, err)
	}
	if got := entries(t, x.Patient); !slices.Equal(got, []string{"urn:uuid:pt1"}) {
		t.Errorf("the profile %v, want the Patient alone", got)
	}
	if len(x.Parts) != 2 || x.Parts[0].OrganizationID != clinicA || x.Parts[1].OrganizationID != clinicB {
		t.Fatalf("parts %+v, want clinics A and B in order", x.Parts)
	}
	a, b := x.Parts[0], x.Parts[1]
	if got := entries(t, file(t, a, "record.json")); !slices.Equal(got, []string{"urn:uuid:pt1", "urn:uuid:c1", "urn:uuid:a1", "urn:uuid:i1"}) {
		t.Errorf("clinic A's record %v", got)
	}
	if got := entries(t, file(t, b, "record.json")); !slices.Equal(got, []string{"urn:uuid:pt1", "urn:uuid:c2", "urn:uuid:a2",
		"urn:uuid:e2", "urn:uuid:d2", "urn:uuid:r2", "urn:uuid:l2", "urn:uuid:i2"}) {
		t.Errorf("clinic B's record %v", got)
	}
	if want := []string{"record.json", "files/i1.pdf"}; !slices.Equal(names(a), want) {
		t.Errorf("clinic A's files %v, want %v", names(a), want)
	}
	if want := []string{"record.json", "quotes.json", "files/d2.pdf", "files/b2.png", "files/i2.pdf", "files/q2.pdf"}; !slices.Equal(names(b), want) {
		t.Errorf("clinic B's files %v, want %v", names(b), want)
	}
	if !bytes.Equal(file(t, b, "files/d2.pdf"), []byte("%PDF dr2")) || !bytes.Equal(file(t, b, "files/b2.png"), []byte("png bytes")) {
		t.Error("the files' bytes are not the documents' and uploads'")
	}
	if !bytes.Equal(file(t, b, "files/i2.pdf"), []byte("%PDF receipt i2")) || !bytes.Equal(file(t, b, "files/q2.pdf"), []byte("%PDF quote q2")) {
		t.Error("the receipt's and the quote's PDFs are not theirs")
	}
	if rec := file(t, b, "record.json"); !bytes.Contains(rec, []byte(`"url":"files/b2.png"`)) || bytes.Contains(rec, []byte("b3")) {
		t.Error("the answer does not name its file in the archive")
	}
	if !bytes.Contains(file(t, b, "quotes.json"), []byte(`"id": "q2"`)) {
		t.Errorf("quotes.json %s", file(t, b, "quotes.json"))
	}
	if want := []string{"1 consent", "1 appointment (the cancelled ones too)", "1 visit record (Encounter)",
		"1 signed document (its PDF in files/)", "1 submitted questionnaire answer", "1 file you uploaded with your answers (in files/)",
		"1 receipt (its PDF in files/)", "1 quote (quotes.json, its PDF in files/)", "1 released lab result (DiagnosticReport)"}; !slices.Equal(b.Holds, want) {
		t.Errorf("clinic B holds %q", b.Holds)
	}
	if !slices.Contains(o.calls, "answers a2 completed") || !slices.Contains(o.calls, "file b3") {
		t.Errorf("calls %v, want the submitted answers and each of their files asked", o.calls)
	}
}

// A clinic target reads that clinic's visits only, and so only their encounters, documents and answers (D7).
func TestOwnRecords_AClinicTargetReadsThatClinic(t *testing.T) {
	t.Parallel()
	o, own := newOwn()
	x, err := own.Export(t.Context(), "acc1", clinicA)
	if err != nil {
		t.Fatal(err)
	}
	if len(x.Parts) != 1 || x.Parts[0].OrganizationID != clinicA {
		t.Fatalf("parts %+v, want clinic A's only", x.Parts)
	}
	if !slices.Equal(o.visitOrgs, []string{clinicA}) {
		t.Errorf("visits read for %v, want clinic A", o.visitOrgs)
	}
	for _, c := range o.calls {
		if strings.Contains(c, "a2") || strings.Contains(c, "e2") || strings.Contains(c, "b2") ||
			strings.Contains(c, "c2") || strings.Contains(c, "i2") || strings.Contains(c, "q2") || strings.Contains(c, "l2") {
			t.Errorf("read %q of clinic B", c)
		}
	}
	if !slices.Contains(o.calls, "consent c1") || !slices.Contains(o.calls, "receipt pdf i1") {
		t.Errorf("calls %v, want clinic A's consent and receipt read", o.calls)
	}
}

// A file the build cannot open (a rotated key, a missing object) is left out and named, and the rest is built
// (docs/clinic-cms/WAVE6.md PX); a done context still fails the build.
func TestOwnRecords_AFileThatCannotBeOpenedIsNamedNotFatal(t *testing.T) {
	t.Parallel()
	o, own := newOwn()
	o.unreadable = map[string]bool{"dr2": true, "b2": true, "q2": true}
	x, err := own.Export(t.Context(), "acc1", "")
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	b := x.Parts[1]
	if want := []string{"record.json", "quotes.json", "files/i2.pdf"}; !slices.Equal(names(b), want) {
		t.Errorf("clinic B's files %v, want %v", names(b), want)
	}
	if want := []string{"files/d2.pdf", "files/b2.png", "files/q2.pdf"}; !slices.Equal(b.Unreadable, want) {
		t.Errorf("unreadable %v, want %v", b.Unreadable, want)
	}
	if got := entries(t, file(t, b, "record.json")); !slices.Contains(got, "urn:uuid:d2") {
		t.Errorf("the document whose PDF failed is not in the record: %v", got)
	}
	if len(x.Parts[0].Unreadable) != 0 {
		t.Errorf("clinic A unreadable %v", x.Parts[0].Unreadable)
	}

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := own.Export(ctx, "acc1", ""); err == nil {
		t.Error("a cancelled build did not fail")
	}
}

// A patient without a profile has no record: nil, and nothing else is read.
func TestOwnRecords_NoProfileNoRecord(t *testing.T) {
	t.Parallel()
	o, own := newOwn()
	o.meErr = patient.ErrNotFound
	x, err := own.Export(t.Context(), "acc1", "")
	if err != nil || x != nil {
		t.Fatalf("Export = %+v, %v; want none", x, err)
	}
	if len(o.calls) != 1 {
		t.Errorf("read %v without a profile", o.calls)
	}
	o.meErr = errors.New("down")
	if _, err := own.Export(t.Context(), "acc1", ""); err == nil {
		t.Error("no error when the profile cannot be read")
	}
}
