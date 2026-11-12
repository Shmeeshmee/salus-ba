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
	"github.com/Shmeeshmee/salus/internal/encounter"
	"github.com/Shmeeshmee/salus/internal/medicationstatement"
	"github.com/Shmeeshmee/salus/internal/questionnaire"
	"github.com/Shmeeshmee/salus/internal/questionnaireresponse"
)

// entry is an item while a view is built, with what completing and auditing it needs.
type entry struct {
	role          string // the visit's practitioner role, for the specialty
	questionnaire string // a response's version, for its title
	response      string // the response a response or a file is part of, audited by its domain
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
func (s *Service) entries(ctx context.Context, c appointment.Caller, visits []encounter.Visit) ([]entry, error) {
	docs, err := s.Compositions.History(ctx, c, visits)
	if err != nil {
		return nil, err
	}
	responses, err := s.Responses.History(ctx, c, visits)
	if err != nil {
		return nil, err
	}
	out := make([]entry, 0, len(visits)+len(docs)+2*len(responses))
	byAppointment := make(map[string]encounter.Encounter, len(visits))
	for _, v := range visits {
		byAppointment[v.AppointmentID] = v.Encounter
		out = append(out, of(c, v.Encounter, TypeVisit, "Encounter/"+v.ID, v.PeriodStart))
	}
	for _, d := range docs {
		e := of(c, d.Encounter, TypeDocument, "Composition/"+d.Draft.ID, d.Draft.SignedAt)
		e.Title, e.Diagnoses = d.Draft.Title, diagnoses(d.Draft)
		out = append(out, e)
	}
	for _, r := range responses {
		v, ok := byAppointment[r.AppointmentID]
		if !ok {
			continue // not a visit given: nothing of it is shown
		}
		e := of(c, v, TypeQuestionnaireResponse, "QuestionnaireResponse/"+r.ID, r.AuthoredAt)
		e.questionnaire, e.response = r.QuestionnaireID, r.ID
		out = append(out, e)
		for _, f := range r.Attachments {
			file := of(c, v, TypeAttachment, "Binary/"+f.BinaryID, r.AuthoredAt)
			file.Title, file.response = f.Title, r.ID
			out = append(out, file)
		}
	}
	slices.SortFunc(out, func(a, b entry) int {
		return cmp.Or(b.Date.Compare(a.Date), strings.Compare(b.Reference, a.Reference))
	})
	return out, nil
}

// of is an entry of the visit v, shared when v is another clinic's.
func of(c appointment.Caller, v encounter.Encounter, typ ItemType, ref string, at time.Time) entry {
	return entry{role: v.PractitionerRoleID, Item: Item{
		Date: at, Clinic: Clinic{ID: v.OrganizationID}, Type: typ, Reference: ref, EncounterID: v.ID,
		Shared: v.OrganizationID != c.OrganizationID,
	}}
}

// diagnoses are the ICD-10 codes of d's filled coded fields, in the template's order.
func diagnoses(d composition.Draft) []string {
	var out []string
	for _, f := range d.Fields {
		if f.Type == composition.FieldCoded && !f.Missing && f.Value != "" {
			out = append(out, f.Value)
		}
	}
	return out
}

// cut is the entries after the cursor at, at most limit, and the cursor of the next page ("" on the last).
func cut(all []entry, at Cursor, limit int) ([]entry, string) {
	var out []entry
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

func items(es []entry) []Item {
	out := make([]Item, len(es))
	for i, e := range es {
		out[i] = e.Item
	}
	return out
}

// complete fills in the clinic name and specialty of each entry and a response's title, as the caller's clinic scope
// shows them; the titles are read (and audited) through the questionnaire's service.
func (s *Service) complete(ctx context.Context, c appointment.Caller, es []entry) error {
	if len(es) == 0 {
		return nil
	}
	var clinics, roles []string
	for _, e := range es {
		clinics, roles = append(clinics, e.Clinic.ID), append(roles, e.role)
	}
	names, err := s.Clinics.Names(ctx, c.OrganizationID, distinct(clinics))
	if err != nil {
		return err
	}
	specialties, err := s.Roles.Specialties(ctx, c.OrganizationID, distinct(roles))
	if err != nil {
		return err
	}
	titles := map[string]string{}
	for i := range es {
		e := &es[i]
		e.Clinic.Name, e.Specialty = names[e.Clinic.ID], specialties[e.role]
		if e.questionnaire == "" || e.Type != TypeQuestionnaireResponse {
			continue
		}
		title, ok := titles[e.questionnaire]
		if !ok {
			if title, err = s.title(ctx, c, e.questionnaire); err != nil {
				return err
			}
			titles[e.questionnaire] = title
		}
		e.Title = title
	}
	return nil
}

// title is the title of the questionnaire version id, or "" when the caller may not read it.
func (s *Service) title(ctx context.Context, c appointment.Caller, id string) (string, error) {
	q, err := s.Questionnaires.Read(ctx, questionnaire.Caller{
		AccountID: c.AccountID, OrganizationID: c.OrganizationID, PractitionerID: c.PractitionerID, PatientID: c.PatientID, Admin: c.Admin,
	}, id)
	switch {
	case errors.Is(err, questionnaire.ErrNotFound):
		return "", nil
	case err != nil:
		return "", err
	}
	return q.Title, nil
}

// disclose has each owning domain audit the entries it gave that are shown: the visits, the documents, the
// responses (a file as part of its response).
func (s *Service) disclose(ctx context.Context, c appointment.Caller, es []entry) error {
	var visits, docs, responses []string
	for _, e := range es {
		switch e.Type {
		case TypeVisit:
			visits = append(visits, e.EncounterID)
		case TypeDocument:
			docs = append(docs, strings.TrimPrefix(e.Reference, "Composition/"))
		case TypeQuestionnaireResponse, TypeAttachment:
			responses = append(responses, e.response)
		}
	}
	for _, d := range []struct {
		disclosed func(ctx context.Context, c appointment.Caller, ids []string) error
		ids       []string
	}{{s.Encounters.Disclosed, visits}, {s.Compositions.Disclosed, docs}, {s.Responses.Disclosed, distinct(responses)}} {
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
	return s.Audit.Record(ctx, e)
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
		if out, err = s.summary(ctx, c, patientID, appointmentID, visits); err != nil {
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
	return out, nil
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

func (s *Service) summary(ctx context.Context, c appointment.Caller, patientID, appointmentID string, visits []encounter.Visit) (Summary, error) {
	out := Summary{Allergies: []SummaryAllergy{}, Medications: []SummaryMedication{}, OpenItems: []OpenItem{}}
	if len(visits) > 0 { // History gives them newest first
		last := []entry{of(c, visits[0].Encounter, TypeVisit, "Encounter/"+visits[0].ID, visits[0].PeriodStart)}
		if err := s.complete(ctx, c, last); err != nil {
			return Summary{}, err
		}
		if err := s.disclose(ctx, c, last); err != nil {
			return Summary{}, err
		}
		out.LastVisit = &last[0].Item
	}
	allergies, err := s.Allergies.Seen(ctx, c, patientID)
	if err != nil {
		return Summary{}, err
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
		return Summary{}, err
	}
	for _, m := range medications {
		if m.Status == medicationstatement.StatusActive {
			out.Medications = append(out.Medications, SummaryMedication{
				Reference: "MedicationStatement/" + m.ID, Text: m.Code.Text, Dosage: m.Dosage, Confirmed: !m.Confirmation.At.IsZero(),
			})
		}
	}
	if out.OpenItems, err = s.open(ctx, c, appointmentID); err != nil {
		return Summary{}, err
	}
	return out, nil
}

// open is what is open for the appointment: its questionnaires the patient has not completed, and the documents of
// its consultation not signed, each read (and audited) by its owning domain.
func (s *Service) open(ctx context.Context, c appointment.Caller, appointmentID string) ([]OpenItem, error) {
	out := []OpenItem{}
	responses, err := s.Responses.Search(ctx, c, questionnaireresponse.Filter{AppointmentID: appointmentID})
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
