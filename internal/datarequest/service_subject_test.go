package datarequest_test

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/datarequest"
	"github.com/Shmeeshmee/salus/internal/patient"
	"github.com/Shmeeshmee/salus/pkg/middleware"
)

// fakeSubjects is patient.Service.RequestSubject: patient A and the registered patient, to the clinic's admin only; it
// keeps who asked for whom and the request id the context named.
type fakeSubjects struct {
	asked []string // caller account, patient and the context's request id
}

func (s *fakeSubjects) RequestSubject(ctx context.Context, c patient.Caller, id string) (patient.Patient, error) {
	s.asked = append(s.asked, c.AccountID+" "+c.OrganizationID+" "+id+" "+middleware.RequestIDFrom(ctx))
	if !c.Admin {
		return patient.Patient{}, patient.ErrForbidden
	}
	switch id {
	case profileA:
		return patient.Patient{ID: id, GivenName: "Ada", FamilyName: "Example", BirthDate: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC), Phone: "+38761000111"}, nil
	case registered:
		return patient.Patient{ID: id, GivenName: "Rina", FamilyName: "Example", BirthDate: time.Date(1966, 6, 6, 0, 0, 0, 0, time.UTC), RegisteredBy: clinicA}, nil
	}
	return patient.Patient{}, patient.ErrNotFound
}

// The clinic's queue and detail name the patient of each request (walk finding 3): the names, the birth date and
// whether the clinic registered them, read as the admin, each read naming its request; a patient the read does not
// find leaves the request without one.
func TestClinicRequests_NameThePatient(t *testing.T) {
	t.Parallel()
	f := newFixture(filedAt)
	subjects := f.subjects
	open := datarequest.Request{OrganizationID: clinicA, Status: datarequest.StatusReceived, ReceivedAt: filedAt, DueAt: filedAt.AddDate(0, 1, 0), Version: 1}
	for id, p := range map[string]string{"r1": profileA, "r2": registered, "r3": "00000000-0000-4000-8000-0000000000f9"} {
		r := open
		r.ID, r.PatientID = id, p
		f.repo.put(r)
	}
	list, err := f.svc.ClinicRequests(t.Context(), admin)
	if err != nil || len(list) != 3 {
		t.Fatalf("ClinicRequests = %+v, %v", list, err)
	}
	got := map[string]datarequest.Subject{}
	for _, r := range list {
		got[r.ID] = r.Subject
	}
	want := map[string]datarequest.Subject{
		"r1": {GivenName: "Ada", FamilyName: "Example", BirthDate: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)},
		"r2": {GivenName: "Rina", FamilyName: "Example", BirthDate: time.Date(1966, 6, 6, 0, 0, 0, 0, time.UTC), RegisteredByClinic: true},
		"r3": {},
	}
	for id := range want {
		if got[id] != want[id] {
			t.Errorf("%s: subject %+v, want %+v", id, got[id], want[id])
		}
	}
	r, _, err := f.svc.ClinicRequest(t.Context(), admin, "r1")
	if err != nil || r.Subject != want["r1"] {
		t.Errorf("ClinicRequest = %+v, %v; want Ada Example", r.Subject, err)
	}
	for _, a := range subjects.asked {
		if !strings.HasPrefix(a, adminA+" "+clinicA+" ") {
			t.Errorf("asked %q, want as clinic A's admin", a)
		}
	}
	if !slices.Contains(subjects.asked, adminA+" "+clinicA+" "+profileA+" "+datarequest.AuditRequestID("r1")) {
		t.Errorf("asked %v, want patient A's read naming request r1", subjects.asked)
	}
}

// Nobody else gets the patient's name: the desk is refused the queue before any read, and the patient's own list, the
// answer, the extension and a filing at the desk carry none.
func TestClinicRequests_TheNameIsTheAdminsOnly(t *testing.T) {
	t.Parallel()
	f := newFixture(filedAt)
	subjects := f.subjects
	f.repo.put(datarequest.Request{ID: "r1", AccountID: accountA, PatientID: profileA, OrganizationID: clinicA, FiledBy: accountA, Kind: datarequest.KindAccess, Status: datarequest.StatusReceived, ReceivedAt: filedAt, DueAt: filedAt.AddDate(0, 1, 0), Version: 1})
	if _, err := f.svc.ClinicRequests(t.Context(), desk); err == nil {
		t.Error("the desk listed the clinic's requests")
	}
	if _, _, err := f.svc.ClinicRequest(t.Context(), desk, "r1"); err == nil {
		t.Error("the desk read a request")
	}
	if len(subjects.asked) != 0 {
		t.Errorf("asked %v for the desk, want nothing read", subjects.asked)
	}
	mine, err := f.svc.Mine(t.Context(), patientA)
	if err != nil || len(mine) != 1 || mine[0].Subject != (datarequest.Subject{}) {
		t.Errorf("Mine = %+v, %v; want the request without a subject", mine, err)
	}
	extended, err := f.svc.Extend(t.Context(), admin, "r1", datarequest.ExtensionComplex, 1)
	if err != nil || extended.Subject != (datarequest.Subject{}) {
		t.Errorf("Extend = %+v, %v; want no subject", extended.Subject, err)
	}
	answered, err := f.svc.Answer(t.Context(), admin, "r1", datarequest.Answer{Status: datarequest.StatusFulfilled, Code: datarequest.AnswerDone, Version: 2})
	if err != nil || answered.Subject != (datarequest.Subject{}) {
		t.Errorf("Answer = %+v, %v; want no subject", answered.Subject, err)
	}
	if len(subjects.asked) != 0 {
		t.Errorf("asked %v, want no read outside the clinic's queue and detail", subjects.asked)
	}
}
