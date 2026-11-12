package history_test

import (
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/encounter"
	"github.com/Shmeeshmee/salus/internal/history"
	"github.com/Shmeeshmee/salus/internal/medicationstatement"
)

func refs(items []history.Item) []string {
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = it.Reference
	}
	return out
}

// The practitioner's view is what the encounters' rule lets them see (encounter.Service.History) and what belongs to
// it, newest first: each visit, its signed documents with their diagnoses, its completed responses and their files,
// marked with the visit's clinic and specialty, a shared visit's as shared.
func TestForPractitioner_ListsTheVisitsAndWhatBelongsToThemNewestFirst(t *testing.T) {
	t.Parallel()
	f := newPractFixture()
	got, err := f.svc.ForPractitioner(t.Context(), pract, patientA, history.Cursor{}, 30)
	if err != nil {
		t.Fatal(err)
	}
	a := history.Clinic{ID: clinicA, Name: "Synthetic clinic A"}
	b := history.Clinic{ID: clinicB, Name: "Synthetic clinic B"}
	want := []history.Item{
		{Date: day(1, 10), Clinic: a, Type: history.TypeDocument, Reference: "Composition/" + doc1, EncounterID: visit1, Specialty: "394579002",
			Title: "Placeholder report", Diagnoses: []history.Diagnosis{
				{Code: "J06.9", Label: "Placeholder diagnosis", Confirmed: true}, {Code: "K21.0", Label: "Placeholder rule out"}}},
		{Date: day(1, 9), Clinic: a, Type: history.TypeVisit, Reference: "Encounter/" + visit1, EncounterID: visit1, Specialty: "394579002"},
		{Date: day(1, 8), Clinic: b, Type: history.TypeVisit, Reference: "Encounter/" + visit2, EncounterID: visit2, Specialty: "419192003", Shared: true},
		{Date: day(1, 8), Clinic: b, Type: history.TypeDocument, Reference: "Composition/" + doc2, EncounterID: visit2, Specialty: "419192003",
			Title: "Placeholder referral", Diagnoses: []history.Diagnosis{{Code: "I10", Label: "Placeholder diagnosis"}}, Shared: true},
		{Date: day(1, 7), Clinic: a, Type: history.TypeQuestionnaireResponse, Reference: "QuestionnaireResponse/" + resp1, EncounterID: visit1,
			Specialty: "394579002", Title: "Placeholder intake", Questionnaire: version1},
		{Date: day(1, 7), Clinic: a, Type: history.TypeAttachment, Reference: "Binary/" + file1, EncounterID: visit1, Specialty: "394579002",
			Title: "Placeholder photo"},
	}
	if !reflect.DeepEqual(got.Items, want) || got.Next != "" {
		t.Errorf("ForPractitioner() =\n%+v, next %q\nwant\n%+v", got.Items, got.Next, want)
	}
	if !slices.Equal(f.encounters.asked, []string{practAcc + " " + patientA}) {
		t.Errorf("asked the encounters for %v, want the caller's history of patient A once", f.encounters.asked)
	}
}

// A coded field is a diagnosis as its Condition says: confirmed only when it is (a code 00042 backfilled from a
// document signed before SAL-19 is unconfirmed, as is one the conditions do not know), with the field's label so a
// "rule out" field never reads as a diagnosis; asked once, as the caller, for the documents the page shows.
func TestForPractitioner_MarksADiagnosisConfirmedOnlyByItsCondition(t *testing.T) {
	t.Parallel()
	f := newPractFixture()
	got, err := f.svc.ForPractitioner(t.Context(), pract, patientA, history.Cursor{}, 1)
	if err != nil || len(got.Items) != 1 {
		t.Fatalf("ForPractitioner() = %v, %v", refs(got.Items), err)
	}
	want := []history.Diagnosis{{Code: "J06.9", Label: "Placeholder diagnosis", Confirmed: true}, {Code: "K21.0", Label: "Placeholder rule out"}}
	if !reflect.DeepEqual(got.Items[0].Diagnoses, want) {
		t.Errorf("diagnoses %+v, want %+v", got.Items[0].Diagnoses, want)
	}
	if want := []string{practAcc + " " + doc1}; !slices.Equal(f.conditions.asked, want) {
		t.Errorf("asked the conditions for %v, want %v", f.conditions.asked, want)
	}
	f = newPractFixture()
	f.conditions.err = errBackend
	if _, err := f.svc.ForPractitioner(t.Context(), pract, patientA, history.Cursor{}, 30); !errors.Is(err, errBackend) || len(f.audit.events) != 0 {
		t.Errorf("conditions down: %v, %d events kept; want the failure, none", err, len(f.audit.events))
	}
}

// Every view is audited: the view itself (the patient), and each item shown by its owning domain, in the same
// transaction; a file as part of its response.
func TestForPractitioner_AuditsTheViewAndEachItemShown(t *testing.T) {
	t.Parallel()
	f := newPractFixture()
	if _, err := f.svc.ForPractitioner(t.Context(), pract, patientA, history.Cursor{}, 30); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"questionnaire " + version1,
		"encounter " + visit1, "encounter " + visit2,
		"composition " + doc1, "composition " + doc2,
		"questionnaireresponse " + resp1,
		"binary " + file1,
		"history search-type Patient/" + patientA,
	}
	if !slices.Equal(*f.log, want) {
		t.Errorf("audit log %q,\nwant %q", *f.log, want)
	}
	e := f.audit.events[0]
	if e.OrganizationID != clinicA || e.AgentAccountID != practAcc || e.Purpose.Code != "TREAT" || e.Outcome != 0 {
		t.Errorf("view event %+v, want clinic A's practitioner for treatment", e)
	}
}

// A page holds at most limit items; its cursor opens the next, which starts strictly after it; only what a page shows
// is audited with it.
func TestForPractitioner_PagesByTheCursor(t *testing.T) {
	t.Parallel()
	f := newPractFixture()
	first, err := f.svc.ForPractitioner(t.Context(), pract, patientA, history.Cursor{}, 3)
	if err != nil || len(first.Items) != 3 || first.Next == "" {
		t.Fatalf("first page = %v, next %q, %v", refs(first.Items), first.Next, err)
	}
	if want := []string{"encounter " + visit1, "encounter " + visit2, "composition " + doc1, "history search-type Patient/" + patientA}; !slices.Equal(*f.log, want) {
		t.Errorf("first page audited %q, want %q", *f.log, want)
	}
	at, err := history.ParseCursor(first.Next)
	if err != nil {
		t.Fatal(err)
	}
	second, err := f.svc.ForPractitioner(t.Context(), pract, patientA, at, 3)
	if err != nil || second.Next != "" {
		t.Fatalf("second page = %v, next %q, %v", refs(second.Items), second.Next, err)
	}
	all := append(refs(first.Items), refs(second.Items)...)
	if want := []string{"Composition/" + doc1, "Encounter/" + visit1, "Encounter/" + visit2, "Composition/" + doc2,
		"QuestionnaireResponse/" + resp1, "Binary/" + file1}; !slices.Equal(all, want) {
		t.Errorf("pages = %v, want %v", all, want)
	}
}

// Without a history of the patient (no care relationship, the admin, another clinic) the view is ErrNotFound, which
// discloses nothing: no item is read and the refusal is audited without the patient.
func TestForPractitioner_IsNotFoundWithoutAHistory(t *testing.T) {
	t.Parallel()
	f := newPractFixture()
	f.encounters.err = encounter.ErrNotFound
	got, err := f.svc.ForPractitioner(t.Context(), pract, patientA, history.Cursor{}, 30)
	if !errors.Is(err, history.ErrNotFound) || got.Items != nil {
		t.Fatalf("ForPractitioner() = %+v, %v; want ErrNotFound", got, err)
	}
	if want := []string{"history search-type "}; !slices.Equal(*f.log, want) {
		t.Errorf("audit log %q, want only the refusal", *f.log)
	}
	if f.audit.events[0].Outcome != 4 {
		t.Errorf("refusal outcome %d, want a minor failure", f.audit.events[0].Outcome)
	}
}

// A visit the encounters' rule leaves out brings nothing: no item, no count, no empty state of it.
func TestForPractitioner_AnUnseenVisitBringsNothing(t *testing.T) {
	t.Parallel()
	f := newPractFixture()
	f.encounters.visits = f.encounters.visits[:1] // the patient stopped sharing visit 2
	got, err := f.svc.ForPractitioner(t.Context(), pract, patientA, history.Cursor{}, 30)
	if err != nil {
		t.Fatal(err)
	}
	for _, it := range got.Items {
		if it.EncounterID == visit2 || it.Clinic.ID == clinicB || it.Shared {
			t.Errorf("item %+v of the unshared visit", it)
		}
	}
	if len(got.Items) != 4 {
		t.Errorf("%d items, want visit 1's 4", len(got.Items))
	}
}

func TestForPractitioner_FailuresRollBack(t *testing.T) {
	t.Parallel()
	for name, fail := range map[string]func(*practFixture){
		"encounters":   func(f *practFixture) { f.encounters.err = errBackend },
		"documents":    func(f *practFixture) { f.compositions.err = errBackend },
		"responses":    func(f *practFixture) { f.responses.err = errBackend },
		"the disclose": func(f *practFixture) { f.encounters.discloseE = errBackend },
		"the audit":    func(f *practFixture) { f.audit.err = errBackend },
	} {
		f := newPractFixture()
		fail(f)
		if _, err := f.svc.ForPractitioner(t.Context(), pract, patientA, history.Cursor{}, 30); !errors.Is(err, errBackend) {
			t.Errorf("%s down: %v, want the failure", name, err)
		}
		if len(f.audit.events) != 0 {
			t.Errorf("%s down: %d events kept", name, len(f.audit.events))
		}
	}
}

// The pre-visit summary, for one of the practitioner's own appointments with the patient: the active allergies and
// medications they may see, the latest visit they may see, and what is open for the appointment (a questionnaire not
// completed, a document not signed). Audited as a read of the patient, with what it shows.
func TestSummary_GivesWhatThePractitionerNeedsBeforeTheVisit(t *testing.T) {
	t.Parallel()
	f := newPractFixture()
	got, err := f.svc.Summary(t.Context(), pract, patientA, apptNow)
	if err != nil {
		t.Fatal(err)
	}
	want := history.Summary{
		LastVisit: &history.Item{Date: day(1, 9), Clinic: history.Clinic{ID: clinicA, Name: "Synthetic clinic A"}, Type: history.TypeVisit,
			Reference: "Encounter/" + visit1, EncounterID: visit1, Specialty: "394579002"},
		Allergies: []history.SummaryAllergy{{Reference: "AllergyIntolerance/" + allergy1, Text: "Placeholder substance",
			Reaction: "Placeholder reaction", Severity: "severe", Confirmed: true}},
		Medications: []history.SummaryMedication{{Reference: "MedicationStatement/" + medicine1, Text: "Placeholder medicine",
			Dosage: medicationstatement.Dosage{DoseValue: 1, DoseUnit: "{tablet}"}}},
		OpenItems: []history.OpenItem{
			{Type: history.OpenQuestionnaire, Reference: "QuestionnaireResponse/" + respNow},
			{Type: history.OpenDraft, Reference: "Composition/" + draftNow},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Summary() =\n%+v\nwant\n%+v", got, want)
	}
	if f.allergies.callers[0] != pract || f.medications.callers[0] != pract {
		t.Errorf("profile asked as %+v, %+v; want the caller", f.allergies.callers, f.medications.callers)
	}
	if last := (*f.log)[len(*f.log)-1]; last != "history read Patient/"+patientA {
		t.Errorf("audit log %q, want the summary's read last", *f.log)
	}
	if !slices.Contains(*f.log, "encounter "+visit1) {
		t.Errorf("audit log %q, want the last visit disclosed", *f.log)
	}
}

// With nothing to show the summary is empty lists, never null.
func TestSummary_IsEmptyListsWithNothingToShow(t *testing.T) {
	t.Parallel()
	f := newPractFixture()
	f.encounters.visits, f.encounters.byAppt = nil, nil
	f.allergies.seen, f.medications.seen, f.responses.all = nil, nil, nil
	got, err := f.svc.Summary(t.Context(), pract, patientA, apptNow)
	if err != nil || got.LastVisit != nil || got.Allergies == nil || got.Medications == nil || got.OpenItems == nil ||
		len(got.Allergies)+len(got.Medications)+len(got.OpenItems) != 0 {
		t.Errorf("Summary() = %+v, %v; want empty lists", got, err)
	}
}

// The summary is for an appointment of the practitioner's own with the patient; anything else is ErrNotFound,
// audited without the patient.
func TestSummary_IsNotFoundForAnotherAppointment(t *testing.T) {
	t.Parallel()
	for name, tt := range map[string]struct {
		setup   func(*practFixture)
		patient string
		appt    string
	}{
		"unknown appointment":       {patient: patientA, appt: "e9000000-0000-4000-8000-0000000000a9"},
		"not theirs":                {patient: patientA, appt: appt2},
		"another patient's":         {patient: "bb000000-0000-4000-8000-0000000000bb", appt: apptNow},
		"no history of the patient": {patient: patientA, appt: apptNow, setup: func(f *practFixture) { f.encounters.err = encounter.ErrNotFound }},
	} {
		f := newPractFixture()
		if tt.setup != nil {
			tt.setup(f)
		}
		if _, err := f.svc.Summary(t.Context(), pract, tt.patient, tt.appt); !errors.Is(err, history.ErrNotFound) {
			t.Errorf("%s: Summary() = %v, want ErrNotFound", name, err)
		}
		if len(f.audit.events) != 1 || f.audit.events[0].EntityReference != "" || f.audit.events[0].Outcome != 4 {
			t.Errorf("%s: audit %+v, want one refusal without an entity", name, f.audit.events)
		}
		if len(f.allergies.callers) != 0 {
			t.Errorf("%s: asked the profile", name)
		}
	}
}

func TestSummary_FailuresRollBack(t *testing.T) {
	t.Parallel()
	for name, fail := range map[string]func(*practFixture){
		"drafts": func(f *practFixture) { f.compositions.draftErr = errBackend },
	} {
		f := newPractFixture()
		fail(f)
		if _, err := f.svc.Summary(t.Context(), pract, patientA, apptNow); !errors.Is(err, errBackend) {
			t.Errorf("%s down: %v, want the failure", name, err)
		}
		if len(f.audit.events) != 0 {
			t.Errorf("%s down: %d events kept", name, len(f.audit.events))
		}
	}
}

// A failed read of the allergies or the medications marks that section unavailable, never empty ("none recorded"
// would read as safe): the rest of the summary is served and its audit kept, and ErrPartial carries the failure.
func TestSummary_AFailedProfileReadIsUnavailableNotEmpty(t *testing.T) {
	t.Parallel()
	for name, tt := range map[string]struct {
		fail                   func(*practFixture)
		allergies, medications bool
	}{
		"allergies":   {fail: func(f *practFixture) { f.allergies.err = errBackend }, allergies: true},
		"medications": {fail: func(f *practFixture) { f.medications.err = errBackend }, medications: true},
		"both":        {fail: func(f *practFixture) { f.allergies.err, f.medications.err = errBackend, errBackend }, allergies: true, medications: true},
	} {
		f := newPractFixture()
		tt.fail(f)
		got, err := f.svc.Summary(t.Context(), pract, patientA, apptNow)
		if !errors.Is(err, history.ErrPartial) || !errors.Is(err, errBackend) {
			t.Errorf("%s down: %v, want ErrPartial with the failure", name, err)
		}
		if got.AllergiesUnavailable != tt.allergies || got.MedicationsUnavailable != tt.medications {
			t.Errorf("%s down: unavailable %v and %v, want %v and %v", name, got.AllergiesUnavailable, got.MedicationsUnavailable, tt.allergies, tt.medications)
		}
		if (tt.allergies && len(got.Allergies) != 0) || (!tt.allergies && len(got.Allergies) != 1) ||
			(tt.medications && len(got.Medications) != 0) || (!tt.medications && len(got.Medications) != 1) {
			t.Errorf("%s down: allergies %v, medications %v", name, got.Allergies, got.Medications)
		}
		if got.LastVisit == nil || len(got.OpenItems) != 2 {
			t.Errorf("%s down: last visit %v, open %v; want the rest of the summary", name, got.LastVisit, got.OpenItems)
		}
		if !slices.Contains(*f.log, "history read Patient/"+patientA) || len(f.audit.events) == 0 {
			t.Errorf("%s down: audit %q, want the view kept", name, *f.log)
		}
	}
}

// With both read, the summary says nothing is unavailable.
func TestSummary_IsAvailableWhenBothAreRead(t *testing.T) {
	t.Parallel()
	got, err := newPractFixture().svc.Summary(t.Context(), pract, patientA, apptNow)
	if err != nil || got.AllergiesUnavailable || got.MedicationsUnavailable {
		t.Errorf("Summary() = %+v, %v", got, err)
	}
}

// The summary reads, and so audits, only the responses it can show as open: the appointment's in-progress ones,
// never its completed or stopped ones.
func TestSummary_AuditsOnlyTheOpenResponsesItShows(t *testing.T) {
	t.Parallel()
	f := newPractFixture()
	if _, err := f.svc.Summary(t.Context(), pract, patientA, apptNow); err != nil {
		t.Fatal(err)
	}
	var searched []string
	for _, l := range *f.log {
		if strings.HasPrefix(l, "questionnaireresponse search ") {
			searched = append(searched, strings.TrimPrefix(l, "questionnaireresponse search "))
		}
	}
	if !slices.Equal(searched, []string{respNow}) {
		t.Errorf("responses read %v, want the open one only", searched)
	}
}
