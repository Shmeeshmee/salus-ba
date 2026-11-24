package clinicexport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"strconv"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/binary"
	"github.com/Shmeeshmee/salus/internal/composition"
	"github.com/Shmeeshmee/salus/internal/consent"
	"github.com/Shmeeshmee/salus/internal/encounter"
	"github.com/Shmeeshmee/salus/internal/invoice"
	"github.com/Shmeeshmee/salus/internal/patient"
	"github.com/Shmeeshmee/salus/internal/questionnaire"
	"github.com/Shmeeshmee/salus/internal/questionnaireresponse"
)

// OwnProfile reads the calling patient's own profile, audited (patient.Service).
type OwnProfile interface {
	Me(ctx context.Context, c patient.Caller) (patient.Patient, error)
}

// OwnConsents reads the calling patient's own consents, audited (consent.Service).
type OwnConsents interface {
	Mine(ctx context.Context, c consent.Caller) ([]consent.Consent, error)
}

// OwnAppointments reads the calling patient's own appointments, every one of theirs or one clinic's, each audited
// (appointment.Service).
type OwnAppointments interface {
	Search(ctx context.Context, c appointment.Caller, f appointment.Filter) ([]appointment.Seen, error)
}

// OwnReceipts reads the receipts issued to the calling patient, audited (invoice.Service).
type OwnReceipts interface {
	Mine(ctx context.Context, c invoice.Caller) ([]invoice.Invoice, error)
}

// OwnQuotes reads the quotes sent to the calling patient, audited (invoice.Service).
type OwnQuotes interface {
	MyQuotes(ctx context.Context, c invoice.Caller) ([]invoice.Quote, error)
}

// OwnEncounters reads the calling patient's encounters of an appointment, each audited (encounter.Service).
type OwnEncounters interface {
	Search(ctx context.Context, c appointment.Caller, f encounter.Filter) ([]encounter.Encounter, error)
}

// OwnDocuments reads the signed documents of the calling patient's encounter and their PDFs, audited
// (composition.Service).
type OwnDocuments interface {
	Signed(ctx context.Context, c appointment.Caller, encounterID string) ([]composition.Document, error)
	PDF(ctx context.Context, c appointment.Caller, id string) (io.ReadCloser, int64, error)
}

// OwnAnswers reads the calling patient's questionnaire responses of an appointment, each audited
// (questionnaireresponse.Service).
type OwnAnswers interface {
	Search(ctx context.Context, c appointment.Caller, f questionnaireresponse.Filter) ([]questionnaireresponse.Seen, error)
}

// OwnQuestionnaires reads the questionnaire version a response of the calling patient answers (questionnaire.Service).
type OwnQuestionnaires interface {
	Read(ctx context.Context, c questionnaire.Caller, id string) (questionnaire.Questionnaire, error)
}

// OwnFiles opens the calling patient's uploaded files that are clean, audited, and reads their bytes (binary.Service).
type OwnFiles interface {
	Download(ctx context.Context, c binary.Caller, id string) (binary.File, error)
	Content(ctx context.Context, f binary.File, from, to int64) (io.ReadCloser, error)
}

// OwnPorts are the services a patient reads their own record through in the app.
type OwnPorts struct {
	Profile        OwnProfile
	Consents       OwnConsents
	Appointments   OwnAppointments
	Receipts       OwnReceipts
	Quotes         OwnQuotes
	Encounters     OwnEncounters
	Documents      OwnDocuments
	Answers        OwnAnswers
	Questionnaires OwnQuestionnaires
	Files          OwnFiles
}

// OwnRecords is the record builder in the patient's own scope (docs/clinic-cms/WAVE5.md D2, D7, S4): what the patient
// reads in the app, read as the patient through the services the app reads, so every read is audited as the
// patient's own request, by the same rules that decide what the app shows.
type OwnRecords struct {
	p OwnPorts
}

// NewOwnRecords returns the builder on its ports.
func NewOwnRecords(p OwnPorts) *OwnRecords {
	return &OwnRecords{p: p}
}

// OwnFile is one file of a clinic's part of the export: its path in the part and its bytes.
type OwnFile struct {
	Name string
	Body []byte
}

// OwnPart is what the patient reads in the app of one clinic: record.json (a FHIR Bundle of type collection: the
// profile, consents, appointments, encounters, signed Compositions, submitted QuestionnaireResponses and receipts),
// quotes.json when there are quotes, and files/<id>.<ext>, the signed documents' PDFs (by their Composition's id) and
// the answers' uploaded files (by Binary id). Holds says what it holds, a line each, for the manifest.
type OwnPart struct {
	OrganizationID string
	Files          []OwnFile
	Holds          []string
}

// OwnExport is the patient's own export: their profile as a Bundle of the Patient alone, and a part per clinic.
type OwnExport struct {
	Patient []byte
	Parts   []OwnPart
}

// Export reads what the patient of account reads in the app: from every clinic when org is empty and from the clinic
// org otherwise, whose visits alone are read and so only their encounters, documents and answers. The visits a
// withdrawal cancelled are in it (S11); a draft, an unsubmitted answer and a file not found clean are not, as the
// app shows none of them. Nil, and nothing more read, for an account with no profile.
func (o *OwnRecords) Export(ctx context.Context, account, org string) (*OwnExport, error) {
	me, err := o.p.Profile.Me(ctx, patient.Caller{AccountID: account})
	if errors.Is(err, patient.ErrNotFound) {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("clinicexport: own record: %w", err)
	}
	all, err := o.read(ctx, appointment.Caller{AccountID: account, PatientID: me.ID}, org)
	if err != nil {
		return nil, fmt.Errorf("clinicexport: own record: %w", err)
	}
	out := &OwnExport{}
	if out.Patient, err = bundleJSON(Record{Patient: patient.ClinicRecord{Patient: me}}); err != nil {
		return nil, fmt.Errorf("clinicexport: own record: %w", err)
	}
	for _, id := range all.clinics(org) {
		part, err := all.part(me, id)
		if err != nil {
			return nil, fmt.Errorf("clinicexport: own record: %w", err)
		}
		out.Parts = append(out.Parts, part)
	}
	return out, nil
}

// own is everything read of the patient, of every clinic or one; the files by clinic.
type own struct {
	files      map[string][]OwnFile
	consents   []consent.Consent
	visits     []appointment.Seen
	encounters []encounter.Encounter
	documents  []composition.Document
	answers    []answered
	receipts   []invoice.Invoice
	quotes     []invoice.Quote
}

// answered is a submitted response with the questionnaire version it answers.
type answered struct {
	version questionnaire.Questionnaire
	seen    questionnaireresponse.Seen
}

// read reads, as the patient c, their consents, the visits of org (every clinic's when empty) with their encounters,
// signed documents and PDFs, submitted answers and their clean files, and their receipts and quotes.
func (o *OwnRecords) read(ctx context.Context, c appointment.Caller, org string) (own, error) {
	out := own{files: map[string][]OwnFile{}}
	var err error
	if out.consents, err = o.p.Consents.Mine(ctx, consent.Caller{AccountID: c.AccountID, PatientID: c.PatientID}); err != nil {
		return own{}, err
	}
	if out.visits, err = o.p.Appointments.Search(ctx, c, appointment.Filter{PatientID: c.PatientID, OrganizationID: org}); err != nil {
		return own{}, err
	}
	for _, v := range out.visits {
		if err := o.visit(ctx, c, v.Appointment, &out); err != nil {
			return own{}, err
		}
	}
	ic := invoice.Caller{AccountID: c.AccountID, PatientID: c.PatientID}
	if out.receipts, err = o.p.Receipts.Mine(ctx, ic); err != nil {
		return own{}, err
	}
	if out.quotes, err = o.p.Quotes.MyQuotes(ctx, ic); err != nil {
		return own{}, err
	}
	return out, nil
}

// visit reads the visit a's encounters with their signed documents and PDFs, and its submitted answers with their
// clean files, into out.
func (o *OwnRecords) visit(ctx context.Context, c appointment.Caller, a appointment.Appointment, out *own) error {
	encs, err := o.p.Encounters.Search(ctx, c, encounter.Filter{AppointmentID: a.ID, PatientID: c.PatientID})
	if err != nil {
		return err
	}
	for _, e := range encs {
		out.encounters = append(out.encounters, e)
		docs, err := o.p.Documents.Signed(ctx, c, e.ID)
		if err != nil {
			return err
		}
		for _, d := range docs {
			r, _, err := o.p.Documents.PDF(ctx, c, d.Draft.DocumentReferenceID)
			body, err := readAll(r, err)
			if err != nil {
				return err
			}
			out.documents = append(out.documents, d)
			out.add(e.OrganizationID, OwnFile{Name: "files/" + d.Draft.ID + ".pdf", Body: body})
		}
	}
	seen, err := o.p.Answers.Search(ctx, c, questionnaireresponse.Filter{AppointmentID: a.ID, Status: questionnaireresponse.StatusCompleted})
	if err != nil {
		return err
	}
	qc := questionnaire.Caller{AccountID: c.AccountID, PatientID: c.PatientID}
	for _, s := range seen {
		v, err := o.p.Questionnaires.Read(ctx, qc, s.Response.QuestionnaireID)
		if err != nil {
			return err
		}
		if s.Response.Attachments, err = o.files(ctx, c, s.Response, out); err != nil {
			return err
		}
		out.answers = append(out.answers, answered{version: v, seen: s})
	}
	return nil
}

// files reads the clean files of the response r into out and returns its attachments that are in it: one the app
// cannot open (still being scanned, quarantined, gone) is left out of both.
func (o *OwnRecords) files(ctx context.Context, c appointment.Caller, r questionnaireresponse.Response, out *own) ([]binary.Attachment, error) {
	var kept []binary.Attachment
	for _, a := range r.Attachments {
		f, err := o.p.Files.Download(ctx, binary.Caller{AccountID: c.AccountID, PatientID: c.PatientID}, a.BinaryID)
		if errors.Is(err, binary.ErrNotFound) || errors.Is(err, binary.ErrScanning) {
			continue
		} else if err != nil {
			return nil, err
		}
		body, err := readAll(o.p.Files.Content(ctx, f, 0, f.Size-1))
		if err != nil {
			return nil, err
		}
		kept = append(kept, a)
		out.add(r.OrganizationID, OwnFile{Name: fileName(a.BinaryID, a.ContentType), Body: body})
	}
	return kept, nil
}

func (x *own) add(org string, f OwnFile) { x.files[org] = append(x.files[org], f) }

// clinics are the clinics the patient has anything at, in order; org alone when it is set.
func (x own) clinics(org string) []string {
	var out []string
	add := func(id string) {
		if id != "" && (org == "" || id == org) && !slices.Contains(out, id) {
			out = append(out, id)
		}
	}
	for _, c := range x.consents {
		add(c.OrganizationID)
	}
	for _, v := range x.visits {
		add(v.Appointment.OrganizationID)
	}
	for _, i := range x.receipts {
		add(i.OrganizationID)
	}
	for _, q := range x.quotes {
		add(q.OrganizationID)
	}
	slices.Sort(out)
	return out
}

// part is the clinic org's part: its record.json, quotes.json and files.
func (x own) part(me patient.Patient, org string) (OwnPart, error) {
	rec, b := x.at(me, org)
	var quotes []any
	for _, q := range x.quotes {
		if q.OrganizationID == org {
			quotes = append(quotes, invoice.QuoteJSON(q))
		}
	}
	body, err := ownBundleJSON(rec, b)
	if err != nil {
		return OwnPart{}, err
	}
	out := OwnPart{OrganizationID: org, Files: []OwnFile{{Name: "record.json", Body: body}}}
	if len(quotes) > 0 {
		q, err := json.MarshalIndent(quotes, "", "  ")
		if err != nil {
			return OwnPart{}, err
		}
		out.Files = append(out.Files, OwnFile{Name: "quotes.json", Body: append(q, '\n')})
	}
	out.Files = append(out.Files, x.files[org]...)
	out.Holds = holds([]int{len(rec.Patient.Consents), len(rec.Visits), len(b.encounters), len(b.documents), len(b.answers),
		len(x.files[org]) - len(b.documents), len(rec.Receipts), len(quotes)})
	return out, nil
}

// at is what x holds of the clinic org: the hand-over's record and the clinical content beside it.
func (x own) at(me patient.Patient, org string) (Record, clinical) {
	rec := Record{Patient: patient.ClinicRecord{Patient: me}}
	var b clinical
	for _, c := range x.consents {
		if c.OrganizationID == org {
			rec.Patient.Consents = append(rec.Patient.Consents, c)
		}
	}
	for _, v := range x.visits {
		if v.Appointment.OrganizationID == org {
			rec.Visits = append(rec.Visits, v)
		}
	}
	for _, e := range x.encounters {
		if e.OrganizationID == org {
			b.encounters = append(b.encounters, e)
		}
	}
	for _, d := range x.documents {
		if d.Encounter.OrganizationID == org {
			b.documents = append(b.documents, d)
		}
	}
	for _, a := range x.answers {
		if a.seen.Response.OrganizationID == org {
			b.answers = append(b.answers, a)
		}
	}
	for _, i := range x.receipts {
		if i.OrganizationID == org {
			rec.Receipts = append(rec.Receipts, i)
		}
	}
	return rec, b
}

// clinical is the clinical content of a part's record.json beside the hand-over's record.
type clinical struct {
	encounters []encounter.Encounter
	documents  []composition.Document
	answers    []answered
}

// holdsOf names the counts holds takes, in its order.
var holdsOf = []string{
	"consent", "appointment (the cancelled ones too)", "visit record (Encounter)", "signed document (its PDF in files/)",
	"submitted questionnaire answer", "file you uploaded with your answers (in files/)", "receipt", "quote (quotes.json)",
}

// holds are the manifest's lines of a part: what it holds, by count in holdsOf's order, leaving out what it has none
// of.
func holds(counts []int) []string {
	var out []string
	for i, n := range counts {
		if n > 0 {
			out = append(out, strconv.Itoa(n)+" "+holdsOf[i])
		}
	}
	return out
}

// readAll reads and closes what a service opened.
func readAll(r io.ReadCloser, err error) ([]byte, error) {
	if err != nil {
		return nil, err
	}
	defer func() { _ = r.Close() }()
	return io.ReadAll(r)
}

// fileName is where the uploaded file id of type contentType goes in its part, named by its opaque Binary id.
func fileName(id, contentType string) string {
	ext := map[string]string{
		"application/pdf": ".pdf", "image/jpeg": ".jpg", "image/png": ".png", "image/heic": ".heic", "image/webp": ".webp",
		"video/mp4": ".mp4", "video/quicktime": ".mov",
	}[contentType]
	return "files/" + id + ext
}
