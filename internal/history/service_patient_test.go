package history_test

import (
	"errors"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/binary"
	"github.com/Shmeeshmee/salus/internal/composition"
	"github.com/Shmeeshmee/salus/internal/encounter"
	"github.com/Shmeeshmee/salus/internal/history"
	"github.com/Shmeeshmee/salus/internal/questionnaireresponse"
	"github.com/Shmeeshmee/salus/pkg/keyset"
)

const (
	visitA    = "ea000000-0000-4000-8000-0000000000a1"
	visitB    = "eb000000-0000-4000-8000-0000000000b1"
	docA      = "da000000-0000-4000-8000-0000000000a1"
	responseB = "fb000000-0000-4000-8000-0000000000b1"
	fileB     = "0b000000-0000-4000-8000-0000000000b1"
	fileX     = "0c000000-0000-4000-8000-0000000000c1" // listed by its response, not clean
)

type patientFixture struct {
	svc       *history.Service
	visits    *fakeVisits
	documents *fakeDocuments
	responses *fakeResponses
	files     *fakeFiles
	clinics   *fakeClinics
	roles     *fakeRoles
	audit     *fakeAudit
	tx        *fakeTx
}

// newPatientFixture is patient A's history at two clinics: a visit at clinic A with a signed document, and at
// clinic B a day later a visit, and before it a completed questionnaire response with a clean file and one that
// is not.
func newPatientFixture() *patientFixture {
	encA := encounter.Encounter{ID: visitA, OrganizationID: clinicA, PatientID: patientA, PractitionerRoleID: roleA, PeriodStart: day}
	encB := encounter.Encounter{ID: visitB, OrganizationID: clinicB, PatientID: patientA, PractitionerRoleID: roleB, PeriodStart: day.Add(24 * time.Hour)}
	resp := questionnaireresponse.Response{
		ID: responseB, OrganizationID: clinicB, PatientID: patientA, PractitionerRoleID: roleB, AuthoredAt: day.Add(23 * time.Hour),
		Status: questionnaireresponse.StatusCompleted,
	}
	f := &patientFixture{
		visits: &fakeVisits{all: []encounter.Encounter{encA, encB}},
		documents: &fakeDocuments{all: []composition.Document{{
			Encounter: encA, Draft: composition.Draft{ID: docA, EncounterID: visitA, Title: "Placeholder report", SignedAt: day.Add(time.Hour), Status: composition.StatusFinal},
		}}},
		responses: &fakeResponses{all: []questionnaireresponse.Response{resp}, files: []questionnaireresponse.Attached{
			{Response: resp, File: binary.Attachment{BinaryID: fileB, Title: "Placeholder.png"}},
			{Response: resp, File: binary.Attachment{BinaryID: fileX, Title: "Unclean.png"}},
		}},
		files:   &fakeFiles{unclean: []string{fileX}},
		clinics: &fakeClinics{},
		roles:   &fakeRoles{},
		audit:   &fakeAudit{},
		tx:      &fakeTx{},
	}
	f.svc = history.NewService(history.Ports{
		Visits: f.visits, Documents: f.documents, Responses: f.responses, Files: f.files, Clinics: f.clinics, Roles: f.roles,
		Audit: f.audit, Tx: f.tx,
	})
	return f
}

var (
	clinicOfA = history.Clinic{ID: clinicA, Name: "Synthetic clinic A"}
	clinicOfB = history.Clinic{ID: clinicB, Name: "Synthetic clinic B"}
	wantAll   = []history.Item{
		{Date: day.Add(24 * time.Hour), Clinic: clinicOfB, Type: history.TypeVisit, Reference: "Encounter/" + visitB, EncounterID: visitB},
		{Date: day.Add(23 * time.Hour), Clinic: clinicOfB, Type: history.TypeQuestionnaireResponse, Reference: "QuestionnaireResponse/" + responseB},
		{Date: day.Add(23 * time.Hour), Clinic: clinicOfB, Type: history.TypeAttachment, Reference: "Binary/" + fileB, Title: "Placeholder.png"},
		{Date: day.Add(time.Hour), Clinic: clinicOfA, Type: history.TypeDocument, Reference: "Composition/" + docA, EncounterID: visitA, Specialty: "394579002", Title: "Placeholder report"},
		{Date: day, Clinic: clinicOfA, Type: history.TypeVisit, Reference: "Encounter/" + visitA, EncounterID: visitA, Specialty: "394579002"},
	}
)

// SAL-39: the patient's timeline is their visits, signed documents, completed questionnaire answers and clean
// files from every clinic, newest first, each with its clinic's name, its visit and its specialty; a file its
// owning domain does not find clean is left out. Each source is asked as the patient, for one more than the page.
func TestTimeline_MergesEverySourceNewestFirst(t *testing.T) {
	f := newPatientFixture()
	got, err := f.svc.Timeline(t.Context(), me, history.Filter{}, history.Cursor{}, 10)
	if err != nil || got.Next != "" || !reflect.DeepEqual(got.Items, wantAll) {
		t.Fatalf("Timeline() = %+v, %v;\nwant %+v and no next page", got, err, wantAll)
	}
	want := asked{caller: me, page: keyset.Page{Limit: 11}}
	for name, a := range map[string][]asked{
		"visits": f.visits.asked, "documents": f.documents.asked, "responses": f.responses.asked, "attachments": f.responses.askedFor,
	} {
		if !slices.Equal(a, []asked{want}) {
			t.Errorf("%s asked %+v, want %+v", name, a, want)
		}
	}
	if !reflect.DeepEqual(f.files.asked, [][]string{{fileX, fileB}}) || f.files.callers[0] != (binary.Caller{AccountID: accountA, PatientID: patientA}) {
		t.Errorf("files asked %v by %+v, want both files as the patient", f.files.asked, f.files.callers)
	}
	if want := []map[string][]string{{clinicA: {roleA}, clinicB: {roleB}}}; !reflect.DeepEqual(f.roles.asked, want) {
		t.Errorf("roles asked %v, want each clinic's roles once, in one call (finding 5)", f.roles.asked)
	}
}

// Pages follow each other by the cursor without a gap or a repeat, the last one without a next page.
func TestTimeline_PagesByTheCursor(t *testing.T) {
	f := newPatientFixture()
	var all []history.Item
	var cursor history.Cursor
	for range 10 {
		got, err := f.svc.Timeline(t.Context(), me, history.Filter{}, cursor, 2)
		if err != nil {
			t.Fatal(err)
		}
		all = append(all, got.Items...)
		if got.Next == "" {
			break
		}
		if cursor, err = history.ParseCursor(got.Next); err != nil {
			t.Fatal(err)
		}
	}
	if !reflect.DeepEqual(all, wantAll) {
		t.Errorf("pages gave %+v,\nwant %+v", all, wantAll)
	}
	last := f.visits.asked[len(f.visits.asked)-1].page
	if last.Ref == "" || last.At.IsZero() || last.Limit != 3 {
		t.Errorf("the last page asked %+v, want the cursor's position and one more than the page", last)
	}
}

// The filter reaches every source it asks: the clinic and the range; the types ask only their own sources, and
// a chat (reserved) asks none.
func TestTimeline_Filters(t *testing.T) {
	f := newPatientFixture()
	from, to := day, day.Add(48*time.Hour)
	got, err := f.svc.Timeline(t.Context(), me, history.Filter{OrganizationID: clinicB, From: from, To: to}, history.Cursor{}, 10)
	if err != nil {
		t.Fatalf("Timeline(clinic B) = %+v, %v", got, err)
	}
	if a := f.visits.asked[0]; a.org != clinicB || !a.page.From.Equal(from) || !a.page.To.Equal(to) {
		t.Errorf("visits asked %+v, want clinic B and the range", a)
	}
	f = newPatientFixture()
	got, err = f.svc.Timeline(t.Context(), me, history.Filter{Types: []history.ItemType{history.TypeVisit, history.TypeAttachment}}, history.Cursor{}, 10)
	if err != nil || len(got.Items) != 3 || len(f.documents.asked) != 0 || len(f.responses.asked) != 0 || len(f.responses.askedFor) != 1 {
		t.Errorf("Timeline(visits, attachments) = %+v, %v; want those only, from their sources only", got, err)
	}
	f = newPatientFixture()
	got, err = f.svc.Timeline(t.Context(), me, history.Filter{Types: []history.ItemType{history.TypeCommunication}}, history.Cursor{}, 10)
	if err != nil || len(got.Items) != 0 || got.Items == nil || len(f.visits.asked)+len(f.files.asked)+len(f.clinics.asked) != 0 {
		t.Errorf("Timeline(chats) = %+v, %v; want an empty page and no source asked", got, err)
	}
}

// The timeline is the patient's own: staff get ErrForbidden and no source is asked.
func TestTimeline_IsForbiddenToStaff(t *testing.T) {
	f := newPatientFixture()
	if _, err := f.svc.Timeline(t.Context(), pract, history.Filter{}, history.Cursor{}, 10); !errors.Is(err, history.ErrForbidden) {
		t.Errorf("Timeline() = %v, want ErrForbidden", err)
	}
	if len(f.visits.asked)+len(f.documents.asked)+len(f.responses.asked) != 0 {
		t.Error("a source was asked")
	}
}

// A limit outside 1 to 100 is ErrLimit, before anything is read.
func TestTimeline_RefusesALimitOutOfRange(t *testing.T) {
	for _, limit := range []int{0, -1, 101} {
		f := newPatientFixture()
		if _, err := f.svc.Timeline(t.Context(), me, history.Filter{}, history.Cursor{}, limit); !errors.Is(err, history.ErrLimit) {
			t.Errorf("Timeline(limit %d) = %v, want ErrLimit", limit, err)
		}
		if len(f.visits.asked) != 0 {
			t.Errorf("limit %d: a source was asked", limit)
		}
	}
}

func TestTimeline_FailsWithASource(t *testing.T) {
	for name, fail := range map[string]func(f *patientFixture){
		"visits":    func(f *patientFixture) { f.visits.err = errBackend },
		"documents": func(f *patientFixture) { f.documents.err = errBackend },
		"responses": func(f *patientFixture) { f.responses.err = errBackend },
		"files":     func(f *patientFixture) { f.files.err = errBackend },
		"clinics":   func(f *patientFixture) { f.clinics.err = errBackend },
		"roles":     func(f *patientFixture) { f.roles.err = errBackend },
	} {
		f := newPatientFixture()
		fail(f)
		if _, err := f.svc.Timeline(t.Context(), me, history.Filter{}, history.Cursor{}, 10); !errors.Is(err, errBackend) {
			t.Errorf("%s: Timeline() = %v, want its error", name, err)
		}
	}
}

// searched is the audit event of the page's item ref at its clinic: a search of the patient's own request.
func searched(ref, org string) auditevent.AuditEvent {
	return auditevent.AuditEvent{
		Type:           auditevent.Coding{System: auditevent.SystemAuditEventType, Code: auditevent.CodeRest},
		Subtype:        auditevent.Coding{System: auditevent.SystemRestfulInteraction, Code: auditevent.CodeSearchType},
		Purpose:        auditevent.Coding{System: auditevent.SystemActReason, Code: auditevent.CodePATRQT},
		Action:         auditevent.ActionExecute,
		Outcome:        auditevent.OutcomeSuccess,
		AgentAccountID: accountA, EntityReference: ref, OrganizationID: org,
	}
}

// Only what the page returns is audited, each item once with its own reference and its clinic, in one insert in
// the transaction the sources read in: not the extra item each source gives to know of a next page, not a file
// binary does not find clean (the review of SAL-39, finding 1 and 8).
func TestTimeline_AuditsWhatThePageReturns(t *testing.T) {
	f := newPatientFixture()
	got, err := f.svc.Timeline(t.Context(), me, history.Filter{}, history.Cursor{}, 2)
	if err != nil || len(got.Items) != 2 {
		t.Fatalf("Timeline() = %+v, %v", got, err)
	}
	want := []auditevent.AuditEvent{searched("Encounter/"+visitB, clinicB), searched("QuestionnaireResponse/"+responseB, clinicB)}
	if len(f.audit.batches) != 1 || !slices.Equal(f.audit.batches[0], want) || !f.audit.inTx[0] {
		t.Errorf("audited %+v, want one insert in the transaction of %+v", f.audit.batches, want)
	}
	if f.tx.n != 1 || !slices.Equal(f.visits.inTx, []bool{true}) {
		t.Errorf("%d transactions, visits read in one %v; want the sources and the audit in one", f.tx.n, f.visits.inTx)
	}

	f = newPatientFixture()
	if _, err := f.svc.Timeline(t.Context(), me, history.Filter{}, history.Cursor{}, 10); err != nil {
		t.Fatal(err)
	}
	var refs []string
	for _, e := range f.audit.batches[0] {
		refs = append(refs, e.EntityReference)
	}
	wantRefs := []string{"Encounter/" + visitB, "QuestionnaireResponse/" + responseB, "Binary/" + fileB, "Composition/" + docA, "Encounter/" + visitA}
	if !slices.Equal(refs, wantRefs) {
		t.Errorf("audited %v, want %v: the unclean file left out", refs, wantRefs)
	}
}

// An empty page is one event without an entity; a failing source audits nothing and fails the page.
func TestTimeline_AuditsAnEmptyPageAndNothingOnAFailure(t *testing.T) {
	f := newPatientFixture()
	if _, err := f.svc.Timeline(t.Context(), me, history.Filter{}, history.Cursor{Date: day.Add(-time.Hour), Reference: "Encounter/x"}, 10); err != nil {
		t.Fatal(err)
	}
	if want := []auditevent.AuditEvent{searched("", "")}; len(f.audit.batches) != 1 || !slices.Equal(f.audit.batches[0], want) {
		t.Errorf("audited %+v, want %+v", f.audit.batches, want)
	}
	f = newPatientFixture()
	f.documents.err = errBackend
	if _, err := f.svc.Timeline(t.Context(), me, history.Filter{}, history.Cursor{}, 10); !errors.Is(err, errBackend) || len(f.audit.batches) != 0 {
		t.Errorf("Timeline() = %v with %d inserts, want the error and nothing audited", err, len(f.audit.batches))
	}
	f = newPatientFixture()
	f.audit.err = errBackend
	if _, err := f.svc.Timeline(t.Context(), me, history.Filter{}, history.Cursor{}, 10); !errors.Is(err, errBackend) {
		t.Errorf("Timeline() = %v, want the audit's error", err)
	}
}
