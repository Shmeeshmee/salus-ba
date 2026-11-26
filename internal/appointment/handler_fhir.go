package appointment

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/Shmeeshmee/salus/internal/slot"
	"github.com/Shmeeshmee/salus/pkg/fhir"
)

// Reader reads and searches appointments for a caller (Service).
type Reader interface {
	Read(ctx context.Context, c Caller, id string) (Seen, error)
	SearchPage(ctx context.Context, c Caller, f Filter, count int) (Page, error)
}

// Handler serves the Appointment read and search of the FHIR API (api/fhir/CapabilityStatement.json), each
// appointment as access lets the caller see it (SPEC.md "Access rule", "FHIR reads and searches").
type Handler struct {
	svc Reader
	// caller is who is signed in for the request, with a patient's profile id or a staff account's role in its
	// clinic, and false for nobody (in internal/api).
	caller func(ctx context.Context) (Caller, bool, error)
	log    *slog.Logger
	base   string // the FHIR API's absolute base URL (FHIR_BASE_URL)
}

// NewHandler returns the handler on svc. base is the FHIR API's absolute base URL (https://…/fhir/R4).
func NewHandler(svc Reader, caller func(ctx context.Context) (Caller, bool, error), base string, log *slog.Logger) *Handler {
	return &Handler{svc: svc, caller: caller, base: base, log: log}
}

// Routes declares the Appointment routes on mux.
func (h *Handler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /fhir/R4/Appointment/{id}", h.read)
	mux.HandleFunc("GET /fhir/R4/Appointment", h.search)
}

// resource is an Appointment as FHIR R4 JSON. At the name level (a past appointment after a revocation) the
// service leaves out the slot, the created time and the cancellation details, and so does the resource.
type resource struct {
	ResourceType      string                `json:"resourceType"`
	ID                string                `json:"id"`
	Extension         []extension           `json:"extension"`
	Status            Status                `json:"status"`
	Start             string                `json:"start"`
	End               string                `json:"end"`
	Created           string                `json:"created,omitempty"`
	CancelationReason *fhir.CodeableConcept `json:"cancelationReason,omitempty"`
	Meta              fhir.Meta             `json:"meta"`
	Slot              []fhir.Reference      `json:"slot,omitempty"`
	Participant       []participant         `json:"participant"`
}

// extension is an Appointment's extension: the visit mode (a code) or the service the booking named (a reference).
type extension struct {
	ValueReference *fhir.Reference `json:"valueReference,omitempty"`
	URL            string          `json:"url"`
	ValueCode      string          `json:"valueCode,omitempty"`
}

// ExtensionService is the extension of an Appointment that names the price-list service the patient picked when
// booking, a reference to its ChargeItemDefinition (api/fhir/StructureDefinition-appointment-service.json,
// docs/plans/clinic-cms.md E). R4 Appointment.serviceType is a coded concept and the clinic's services have no code
// system; absent when the booking named none.
const ExtensionService = "https://salus.invalid/fhir/StructureDefinition/appointment-service"

// participant is the patient, the practitioner or the practitioner role: the practitioner and the role accepted, the
// patient as patientStatus says.
type participant struct {
	Status string         `json:"status"`
	Actor  fhir.Reference `json:"actor"`
}

// patientStatus is the patient participant's ParticipationStatus (docs/plans/appointments.md Q3, ADR 0015): declined
// once they cancelled it themselves (reason pat, by their account), accepted once they confirmed, tentative until then.
// A cancellation by staff or by the revocation of the consent (reason pat, by no account) leaves it as it was.
func patientStatus(a Appointment) string {
	switch {
	case a.Status == StatusCancelled && a.Reason == ReasonPatient && a.CancelledBy != "":
		return "declined"
	case !a.ConfirmedAt.IsZero():
		return "accepted"
	}
	return "tentative"
}

func instant(t time.Time) string { return t.UTC().Format(time.RFC3339) }

// FHIR is the appointment s as a FHIR R4 Appointment resource, for a record that holds it (the clinic's hand-over and
// export of a registered patient's record, docs/clinic-cms wave 4).
func FHIR(s Seen) any { return toFHIR(s) }

func toFHIR(s Seen) resource {
	a := s.Appointment
	patient := fhir.Reference{Reference: "Patient/" + a.PatientID}
	if s.Patient.Family != "" { // the name only where Profile allows it
		patient.Display = s.Patient.Given + " " + s.Patient.Family
	}
	res := resource{
		ResourceType: "Appointment", ID: a.ID, Status: a.Status, Meta: fhir.NewMeta(a.VersionID, a.LastUpdated),
		Start: instant(a.Start), End: instant(a.End), Participant: []participant{{Status: patientStatus(a), Actor: patient}},
		Extension: []extension{{URL: slot.ExtensionVisitMode, ValueCode: string(visitMode(a.Mode))}},
	}
	if a.ServiceID != "" {
		res.Extension = append(res.Extension, extension{URL: ExtensionService, ValueReference: &fhir.Reference{Reference: "ChargeItemDefinition/" + a.ServiceID}})
	}
	if s.PractitionerID != "" {
		res.Participant = append(res.Participant, participant{Status: "accepted", Actor: fhir.Reference{Reference: "Practitioner/" + s.PractitionerID}})
	}
	res.Participant = append(res.Participant, participant{Status: "accepted", Actor: fhir.Reference{Reference: "PractitionerRole/" + a.PractitionerRoleID}})
	if a.LocationID != "" { // an in-person visit's site (docs/clinic-cms/WAVE6.md U2)
		res.Participant = append(res.Participant, participant{Status: "accepted", Actor: fhir.Reference{Reference: "Location/" + a.LocationID}})
	}
	if !a.Created.IsZero() {
		res.Created = instant(a.Created)
	}
	if a.SlotID != "" {
		res.Slot = []fhir.Reference{{Reference: "Slot/" + a.SlotID}}
	}
	if a.Reason != "" {
		res.CancelationReason = &fhir.CodeableConcept{Coding: []fhir.Coding{{System: SystemCancellationReason, Code: string(a.Reason)}}}
	}
	return res
}

func (h *Handler) read(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	s, err := h.svc.Read(r.Context(), c, r.PathValue("id"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("ETag", fhir.ETag(strconv.FormatInt(int64(s.Appointment.VersionID), 10)))
	fhir.Write(w, http.StatusOK, toFHIR(s))
}

// search answers Appointment?[patient=…][&practitioner=…][&date=ge…][&date=lt…][&status=…][&_count=…][&_cursor=…],
// a page at a time where the service pages it (SearchPage), with the next page's link while more may follow.
func (h *Handler) search(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	q, f, count, ok := searchQuery(r.URL.RawQuery)
	if !ok {
		fhir.WriteError(w, http.StatusBadRequest, "not-supported",
			"Search by patient, practitioner, date (ge and lt, instants with an offset), status and _count (1 to 200), each once, from a next link's _cursor.")
		return
	}
	page, err := h.svc.SearchPage(r.Context(), c, f, count)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	matches := make([]fhir.BundleEntry, 0, len(page.Seen))
	for _, s := range page.Seen {
		matches = append(matches, fhir.BundleEntry{FullURL: h.base + "/Appointment/" + s.Appointment.ID, Resource: toFHIR(s)})
	}
	var next string
	if page.More {
		last := page.Seen[len(page.Seen)-1].Appointment
		next = fhir.NextLink(h.base+"/Appointment", q, fhir.NewCursor(last.Start, last.ID))
	}
	w.Header().Set("Cache-Control", "no-store")
	fhir.Write(w, http.StatusOK, fhir.NewPage(fhir.SelfLink(h.base+"/Appointment", q), next, f.AfterID == "", matches, nil))
}

// searchQuery reads the search into a filter and the count it asks for (0 for none), and false for an unknown or
// repeated parameter or a value it cannot read.
func searchQuery(raw string) (url.Values, Filter, int, bool) {
	q, err := url.ParseQuery(raw)
	if err != nil {
		return nil, Filter{}, 0, false
	}
	var f Filter
	var count int
	for key, values := range q {
		ok := len(values) == 1
		switch key {
		case "patient":
			f.PatientID, ok = fhir.ReferenceID(values[0], "Patient")
			ok = ok && len(values) == 1
		case "practitioner":
			f.PractitionerID, ok = fhir.ReferenceID(values[0], "Practitioner")
			ok = ok && len(values) == 1
		case "status":
			f.Status = Status(values[0])
			ok = ok && (f.Status == StatusBooked || f.Status == StatusArrived || f.Status == StatusCancelled || f.Status == StatusFulfilled || f.Status == StatusNoShow)
		case "date":
			f.From, f.To, ok = fhir.DateRange(values)
		case "_count":
			count, ok = fhir.Count(values, MaxCount)
		case "_cursor":
			f.AfterStart, f.AfterID, ok = fhir.ParseCursor(values[0])
			ok = ok && len(values) == 1 && !f.AfterStart.IsZero()
		default:
			ok = false
		}
		if !ok {
			return nil, Filter{}, 0, false
		}
	}
	return q, f, count, true
}

// signedIn returns the caller, or answers 401 (nobody signed in) or 500 and false.
func (h *Handler) signedIn(w http.ResponseWriter, r *http.Request) (Caller, bool) {
	c, ok, err := h.caller(r.Context())
	switch {
	case err != nil:
		fhir.InternalError(w, r, h.log, err)
	case !ok:
		fhir.WriteError(w, http.StatusUnauthorized, "login", "Sign in to use this resource.")
	}
	return c, ok && err == nil
}

func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		fhir.WriteError(w, http.StatusNotFound, "not-found", "Resource not found.")
	case errors.Is(err, ErrForbidden):
		fhir.WriteError(w, http.StatusForbidden, "forbidden", "Search your own appointments only.")
	default:
		fhir.InternalError(w, r, h.log, err)
	}
}
