package history

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/Shmeeshmee/salus/internal/allergyintolerance"
	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/composition"
	"github.com/Shmeeshmee/salus/internal/condition"
	"github.com/Shmeeshmee/salus/internal/encounter"
	"github.com/Shmeeshmee/salus/internal/medicationstatement"
	"github.com/Shmeeshmee/salus/internal/questionnaire"
	"github.com/Shmeeshmee/salus/internal/questionnaireresponse"
)

// viewEntry is an item while the practitioner's view is built, with what completing and auditing it needs.
type viewEntry struct {
	role     string                   // the visit's practitioner role, for the specialty
	response string                   // the response a response or a file is part of, audited by its domain
	coded    []composition.DraftField // a document's filled coded fields, its diagnoses once their Conditions are read
	Item
}

// ForPractitioner returns one page of the patient's history as the practitioner caller may see it (ADR 0014,
// SAL-40), newest first, at most limit items after the cursor at: the visits encounter.Service.History gives them
// (their own at their clinic, the ones another clinic's patient shared with it, access.HistoryVisit) and what belongs
// to each, their signed documents with their diagnoses, completed responses and scanned files. Nothing else exists for
// them: an unseen visit is not listed, counted or hinted at. In one transaction, every view is audited: the view as a
// search of the patient, and each item shown by its owning domain. ErrNotFound, audited without the patient, when the
// caller has no history of the patient (no appointment with them under an active care consent, the admin, a patient).
func (s *Service) ForPractitioner(ctx context.Context, c appointment.Caller, patientID string, at Cursor, limit int) (Page, error) {
	var page Page
	var refused bool
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		visits, err := s.Encounters.History(ctx, c, patientID)
		switch {
		case errors.Is(err, encounter.ErrNotFound):
			refused = true
			return s.recordView(ctx, c, auditevent.CodeSearchType, "", auditevent.OutcomeMinorFailure)
		case err != nil:
			return err
		}
		all, err := s.entries(ctx, c, visits)
		if err != nil {
			return err
		}
		shown, next := cut(all, at, limit)
		if err := s.complete(ctx, c, shown); err != nil {
			return err
		}
		if err := s.disclose(ctx, c, shown); err != nil {
			return err
		}
		page = Page{Next: next, Items: items(shown)}
		return s.recordView(ctx, c, auditevent.CodeSearchType, patientID, auditevent.OutcomeSuccess)
	})
	switch {
	case err != nil:
		return Page{}, fmt.Errorf("history: for practitioner: %w", err)
	case refused:
		return Page{}, ErrNotFound
	}
	return page, nil
}

// entries are the items of the visits, newest first (the reference breaking a tie, as Cursor.Before reads it).
func (s *Service) entries(ctx context.Context, c appointment.Caller, visits []encounter.Visit) ([]viewEntry, error) {
	docs, err := s.Compositions.History(ctx, c, visits)
	if err != nil {
		return nil, err
	}
	responses, err := s.VisitResponses.History(ctx, c, visits)
	if err != nil {
		return nil, err
	}
	out := make([]viewEntry, 0, len(visits)+len(docs)+2*len(responses))
	byAppointment := make(map[string]encounter.Encounter, len(visits))
	for _, v := range visits {
		byAppointment[v.AppointmentID] = v.Encounter
		out = append(out, of(c, v.Encounter, TypeVisit, "Encounter/"+v.ID, v.PeriodStart))
	}
	for _, d := range docs {
		e := of(c, d.Encounter, TypeDocument, "Composition/"+d.Draft.ID, d.Draft.SignedAt)
		e.Title, e.coded = d.Draft.Title, coded(d.Draft)
		out = append(out, e)
	}
	for _, r := range responses {
		v, ok := byAppointment[r.AppointmentID]
		if !ok {
			continue // not a visit given: nothing of it is shown
		}
		e := of(c, v, TypeQuestionnaireResponse, "QuestionnaireResponse/"+r.ID, r.AuthoredAt)
		e.Questionnaire, e.response = r.QuestionnaireID, r.ID
		out = append(out, e)
		for _, f := range r.Attachments {
			file := of(c, v, TypeAttachment, "Binary/"+f.BinaryID, r.AuthoredAt)
			file.Title, file.response = f.Title, r.ID
			out = append(out, file)
		}
	}
	slices.SortFunc(out, func(a, b viewEntry) int {
		return cmp.Or(b.Date.Compare(a.Date), strings.Compare(b.Reference, a.Reference))
	})
	return out, nil
}

// of is an viewEntry of the visit v, shared when v is another clinic's.
func of(c appointment.Caller, v encounter.Encounter, typ ItemType, ref string, at time.Time) viewEntry {
	return viewEntry{role: v.PractitionerRoleID, Item: Item{
		Date: at, Clinic: Clinic{ID: v.OrganizationID}, Type: typ, Reference: ref, EncounterID: v.ID,
		Shared: v.OrganizationID != c.OrganizationID,
	}}
}

// coded is d's filled coded fields, in the template's order.
func coded(d composition.Draft) []composition.DraftField {
	var out []composition.DraftField
	for _, f := range d.Fields {
		if f.Type == composition.FieldCoded && !f.Missing && f.Value != "" {
			out = append(out, f)
		}
	}
	return out
}

// cut is the entries after the cursor at, at most limit, and the cursor of the next page ("" on the last).
func cut(all []viewEntry, at Cursor, limit int) ([]viewEntry, string) {
	var out []viewEntry
	for _, e := range all {
		if at.Before(e.Item) {
			out = append(out, e)
		}
	}
	if len(out) > limit {
		return out[:limit], CursorAt(out[limit-1].Item).String()
	}
	return out, ""
}

func items(es []viewEntry) []Item {
	out := make([]Item, len(es))
	for i, e := range es {
		out[i] = e.Item
	}
	return out
}

// complete fills in the clinic name and specialty of each viewEntry, a response's title and a document's diagnoses, as
// the caller's clinic scope shows them: the titles read (and audited) through the questionnaire's service, the
// diagnoses' confirmation through the conditions, each in one read for the page.
func (s *Service) complete(ctx context.Context, c appointment.Caller, es []viewEntry) error {
	if len(es) == 0 {
		return nil
	}
	var clinics, roles, versions, docs []string
	for _, e := range es {
		clinics, roles = append(clinics, e.Clinic.ID), append(roles, e.role)
		switch {
		case e.Type == TypeQuestionnaireResponse:
			versions = append(versions, e.Questionnaire)
		case e.Type == TypeDocument && len(e.coded) > 0:
			docs = append(docs, strings.TrimPrefix(e.Reference, "Composition/"))
		}
	}
	names, err := s.ScopeClinics.ScopeNames(ctx, c.OrganizationID, distinct(clinics))
	if err != nil {
		return err
	}
	specialties, err := s.RoleSpecialties.Specialties(ctx, c.OrganizationID, distinct(roles))
	if err != nil {
		return err
	}
	titles, confirmed, err := s.titlesAndConfirmed(ctx, c, distinct(versions), docs)
	if err != nil {
		return err
	}
	for i := range es {
		e := &es[i]
		e.Clinic.Name, e.Specialty = names[e.Clinic.ID], specialties[e.role]
		if e.Type == TypeQuestionnaireResponse {
			e.Title = titles[e.Questionnaire]
		}
		keys := confirmed[strings.TrimPrefix(e.Reference, "Composition/")]
		for _, f := range e.coded {
			// Confirmed only for this field and this code: a Condition of another code is not its confirmation.
			confirmedHere := slices.Contains(keys, condition.FieldCode{Key: f.Key, Code: f.Value})
			e.Diagnoses = append(e.Diagnoses, Diagnosis{Code: f.Value, Label: f.Label, Confirmed: confirmedHere})
		}
	}
	return nil
}

// titlesAndConfirmed reads the titles of the versions and the confirmed coded fields of the documents, each only
// when there is one.
func (s *Service) titlesAndConfirmed(ctx context.Context, c appointment.Caller, versions, docs []string) (map[string]string, map[string][]condition.FieldCode, error) {
	var titles map[string]string
	var confirmed map[string][]condition.FieldCode
	var err error
	if len(versions) > 0 {
		qc := questionnaire.Caller{AccountID: c.AccountID, OrganizationID: c.OrganizationID, PractitionerID: c.PractitionerID, PatientID: c.PatientID, Admin: c.Admin}
		if titles, err = s.Questionnaires.Titles(ctx, qc, versions); err != nil {
			return nil, nil, err
		}
	}
	if len(docs) > 0 {
		if confirmed, err = s.Conditions.Confirmed(ctx, c, docs); err != nil {
			return nil, nil, err
		}
	}
	return titles, confirmed, nil
}

// disclose has each owning domain audit the entries it gave that are shown: the visits, the documents, the
// responses (a file as part of its response).
func (s *Service) disclose(ctx context.Context, c appointment.Caller, es []viewEntry) error {
	var visits, docs, responses, files []string
	for _, e := range es {
		switch e.Type {
		case TypeVisit:
			visits = append(visits, e.EncounterID)
		case TypeDocument:
			docs = append(docs, strings.TrimPrefix(e.Reference, "Composition/"))
		case TypeQuestionnaireResponse:
			responses = append(responses, e.response)
		case TypeAttachment:
			responses, files = append(responses, e.response), append(files, strings.TrimPrefix(e.Reference, "Binary/"))
		}
	}
	for _, d := range []struct {
		disclosed func(ctx context.Context, c appointment.Caller, ids []string) error
		ids       []string
	}{{s.Encounters.Disclosed, visits}, {s.Compositions.Disclosed, docs}, {s.VisitResponses.Disclosed, distinct(responses)}, {s.VisitResponses.DisclosedFiles, files}} {
		if len(d.ids) == 0 {
			continue
		}
		if err := d.disclosed(ctx, c, d.ids); err != nil {
			return err
		}
	}
	return nil
}

// distinct is ids without repeats or empty ones, in order.
func distinct(ids []string) []string {
	var out []string
	for _, id := range ids {
		if id != "" && !slices.Contains(out, id) {
			out = append(out, id)
		}
	}
	return out
}

// recordView audits a view by c of the patient patientID ("" when none was disclosed: no id from the request is
// stored), interaction a search for the history and a read for the summary, for treatment.
func (s *Service) recordView(ctx context.Context, c appointment.Caller, interaction, patientID string, outcome auditevent.Outcome) error {
	action := auditevent.ActionExecute
	if interaction == auditevent.CodeRead {
		action = auditevent.ActionRead
	}
	e := auditevent.AuditEvent{
		Type:           auditevent.Coding{System: auditevent.SystemAuditEventType, Code: auditevent.CodeRest},
		Subtype:        auditevent.Coding{System: auditevent.SystemRestfulInteraction, Code: interaction},
		Purpose:        auditevent.Coding{System: auditevent.SystemActReason, Code: auditevent.CodeTREAT},
		Action:         action,
		Outcome:        outcome,
		AgentAccountID: c.AccountID,
		OrganizationID: c.OrganizationID,
	}
	if patientID != "" {
		e.EntityReference = "Patient/" + patientID
	}
	return s.Audit.RecordAll(ctx, []auditevent.AuditEvent{e})
}

// Summary returns the pre-visit summary of the patient for the appointment appointmentID, one of the practitioner
// caller's own with the patient (SAL-40): the active allergies and medications they may see (access.ProfileEntry, by
// their services), the latest visit of their history, and what is open for the appointment (a questionnaire not
// completed, a document not signed). Built from the same visibility as ForPractitioner. In one transaction, audited as
// a read of the patient, with what it shows audited by its owning domains. ErrNotFound, audited without the patient,
// for an appointment that is not the caller's with the patient or a patient they have no history of.
func (s *Service) Summary(ctx context.Context, c appointment.Caller, patientID, appointmentID string) (Summary, error) {
	var out Summary
	var refused bool
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		visits, ok, err := s.summaryAccess(ctx, c, patientID, appointmentID)
		switch {
		case err != nil:
			return err
		case !ok:
			refused = true
			return s.recordView(ctx, c, auditevent.CodeRead, "", auditevent.OutcomeMinorFailure)
		}
		if out, err = s.summary(ctx, c, appointmentID, visits); err != nil {
			return err
		}
		return s.recordView(ctx, c, auditevent.CodeRead, patientID, auditevent.OutcomeSuccess)
	})
	switch {
	case err != nil:
		return Summary{}, fmt.Errorf("history: summary: %w", err)
	case refused:
		return Summary{}, ErrNotFound
	}
	if err := s.profile(ctx, c, patientID, &out); err != nil {
		return out, fmt.Errorf("%w: %w", ErrPartial, err)
	}
	return out, nil
}

// profile adds to out the active allergies and medications the caller may see (access.ProfileEntry, by their
// services, each reading and auditing in a transaction of its own after the summary's), and marks a section
// unavailable when its read fails: a failure there loses neither the rest of the summary nor its audit, and is never
// shown as "none recorded". It returns the failures.
func (s *Service) profile(ctx context.Context, c appointment.Caller, patientID string, out *Summary) error {
	var failed []error
	allergies, err := s.Allergies.Seen(ctx, c, patientID)
	if err != nil {
		allergies, out.AllergiesUnavailable, failed = nil, true, append(failed, fmt.Errorf("allergies: %w", err))
	}
	for _, a := range allergies {
		if a.ClinicalStatus == allergyintolerance.ClinicalActive {
			out.Allergies = append(out.Allergies, SummaryAllergy{
				Reference: "AllergyIntolerance/" + a.ID, Text: a.Code.Text, Reaction: a.Reaction, Severity: string(a.Severity),
				Confirmed: a.VerificationStatus == allergyintolerance.VerificationConfirmed,
			})
		}
	}
	medications, err := s.Medications.Seen(ctx, c, patientID)
	if err != nil {
		medications, out.MedicationsUnavailable, failed = nil, true, append(failed, fmt.Errorf("medications: %w", err))
	}
	for _, m := range medications {
		if m.Status == medicationstatement.StatusActive {
			out.Medications = append(out.Medications, SummaryMedication{
				Reference: "MedicationStatement/" + m.ID, Text: m.Code.Text, Dosage: m.Dosage, Confirmed: !m.Confirmation.At.IsZero(),
			})
		}
	}
	return errors.Join(failed...)
}

// summaryAccess gives the caller's visits of the patient's history when the appointment is theirs with the patient,
// and false when it is not or they have no history of the patient.
func (s *Service) summaryAccess(ctx context.Context, c appointment.Caller, patientID, appointmentID string) ([]encounter.Visit, bool, error) {
	att, err := s.Appointments.Attendance(ctx, c, appointmentID)
	switch {
	case errors.Is(err, appointment.ErrNotFound):
		return nil, false, nil
	case err != nil:
		return nil, false, err
	case !att.Practitioner || att.Appointment.PatientID != patientID:
		return nil, false, nil
	}
	visits, err := s.Encounters.History(ctx, c, patientID)
	switch {
	case errors.Is(err, encounter.ErrNotFound):
		return nil, false, nil
	case err != nil:
		return nil, false, err
	}
	return visits, true, nil
}

func (s *Service) summary(ctx context.Context, c appointment.Caller, appointmentID string, visits []encounter.Visit) (Summary, error) {
	out := Summary{Allergies: []SummaryAllergy{}, Medications: []SummaryMedication{}, OpenItems: []OpenItem{}}
	if len(visits) > 0 { // History gives them newest first
		last := []viewEntry{of(c, visits[0].Encounter, TypeVisit, "Encounter/"+visits[0].ID, visits[0].PeriodStart)}
		if err := s.complete(ctx, c, last); err != nil {
			return Summary{}, err
		}
		if err := s.disclose(ctx, c, last); err != nil {
			return Summary{}, err
		}
		out.LastVisit = &last[0].Item
	}
	var err error
	if out.OpenItems, err = s.open(ctx, c, appointmentID); err != nil {
		return Summary{}, err
	}
	return out, nil
}

// open is what is open for the appointment: its questionnaires the patient has not completed, and the documents of
// its consultation not signed, each read (and audited) by its owning domain.
func (s *Service) open(ctx context.Context, c appointment.Caller, appointmentID string) ([]OpenItem, error) {
	out := []OpenItem{}
	responses, err := s.VisitResponses.Search(ctx, c, questionnaireresponse.Filter{AppointmentID: appointmentID, Status: questionnaireresponse.StatusInProgress})
	if err != nil {
		return nil, err
	}
	for _, r := range responses {
		if r.Response.Status == questionnaireresponse.StatusInProgress {
			out = append(out, OpenItem{Type: OpenQuestionnaire, Reference: "QuestionnaireResponse/" + r.Response.ID})
		}
	}
	encounters, err := s.Encounters.Search(ctx, c, encounter.Filter{AppointmentID: appointmentID})
	if err != nil {
		return nil, err
	}
	for _, e := range encounters {
		drafts, err := s.Compositions.Drafts(ctx, c, e.ID)
		switch {
		case errors.Is(err, composition.ErrNotFound), errors.Is(err, composition.ErrForbidden), errors.Is(err, composition.ErrConsentRequired):
			continue // the documents are not the caller's to see: none is open for them
		case err != nil:
			return nil, err
		}
		for _, d := range drafts {
			if d.Status == composition.StatusPreliminary {
				out = append(out, OpenItem{Type: OpenDraft, Reference: "Composition/" + d.ID})
			}
		}
	}
	return out, nil
}
