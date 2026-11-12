package condition_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/condition"
	"github.com/Shmeeshmee/salus/internal/encounter"
)

type txKey struct{}

// fakeTx runs fn in a marked context.
type fakeTx struct{}

func (fakeTx) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(context.WithValue(ctx, txKey{}, true))
}

// fakeReadRepo holds one Condition, seen in the scope of its clinic or its patient.
type fakeReadRepo struct {
	scope string
	c     condition.Condition
}

func (f *fakeReadRepo) SetClinic(_ context.Context, id string) error {
	f.scope = "clinic " + id
	return nil
}
func (f *fakeReadRepo) SetPatient(_ context.Context, id string) error {
	f.scope = "patient " + id
	return nil
}
func (f *fakeReadRepo) ByID(_ context.Context, id string) (condition.Condition, error) {
	if id != f.c.ID || (f.scope != "clinic "+f.c.OrganizationID && f.scope != "patient "+f.c.PatientID) {
		return condition.Condition{}, condition.ErrNotFound
	}
	return f.c, nil
}

// fakeEncounters gives the encounter with the caller's attendance.
type fakeEncounters struct{ att appointment.Attendance }

func (f fakeEncounters) Access(context.Context, appointment.Caller, string) (encounter.Encounter, appointment.Attendance, error) {
	return encounter.Encounter{ID: other, OrganizationID: clinic, PatientID: patient}, f.att, nil
}

type fakeAudit struct{ events []auditevent.AuditEvent }

func (a *fakeAudit) Record(ctx context.Context, e auditevent.AuditEvent) error {
	if ctx.Value(txKey{}) == nil {
		return errors.New("audited outside a transaction")
	}
	a.events = append(a.events, e)
	return nil
}

const (
	condID    = "00000000-0000-4000-8000-0000000000c9"
	account   = "00000000-0000-4000-8000-0000000000a9"
	unrelated = "00000000-0000-4000-8000-0000000000ee"
)

var (
	practitioner = appointment.Caller{AccountID: account, OrganizationID: clinic, PractitionerID: unrelated}
	thePatient   = appointment.Caller{AccountID: account, PatientID: patient}
)

func attendance(isPractitioner, isPatient, consent bool) appointment.Attendance {
	return appointment.Attendance{
		Appointment:   appointment.Appointment{PatientID: patient, PractitionerRoleID: role, OrganizationID: clinic, Start: signedAt.Add(-time.Hour)},
		ConsentActive: consent, Practitioner: isPractitioner, Patient: isPatient,
	}
}

func reads(att appointment.Attendance) (*condition.Reads, *fakeAudit) {
	c := valid()
	c.ID, c.VersionID, c.LastUpdated = condID, 1, signedAt
	audit := &fakeAudit{}
	return condition.NewReads(condition.ReadPorts{
		Repo: &fakeReadRepo{c: c}, Encounters: fakeEncounters{att: att}, Audit: audit, Tx: fakeTx{},
	}), audit
}

// The encounter's practitioner with the patient's consent, and the patient whatever the consent, read a
// Condition: it is part of the signed document (access.SignedDocument). Audited as a read of it.
func TestGet_ThePractitionerAndThePatientReadTheCondition(t *testing.T) {
	for name, tt := range map[string]struct {
		c   appointment.Caller
		att appointment.Attendance
	}{
		"the practitioner":                   {practitioner, attendance(true, false, true)},
		"the patient":                        {thePatient, attendance(false, true, true)},
		"the patient after their revocation": {thePatient, attendance(false, true, false)},
	} {
		svc, audit := reads(tt.att)
		got, err := svc.Get(context.Background(), tt.c, condID)
		if err != nil || got.Code != "J06.9" {
			t.Errorf("%s: Get = %+v, %v", name, got, err)
			continue
		}
		if len(audit.events) != 1 || audit.events[0].EntityReference != "Condition/"+condID || audit.events[0].Outcome != auditevent.OutcomeSuccess ||
			audit.events[0].Subtype.Code != auditevent.CodeRead || audit.events[0].OrganizationID != clinic {
			t.Errorf("%s: audit = %+v", name, audit.events)
		}
	}
}

// Anyone else is refused, audited: another practitioner of the clinic and its admin (forbidden), the practitioner
// after a revocation (consent required), a Condition the caller's scope does not see (not found).
func TestGet_Refusals(t *testing.T) {
	admin := appointment.Caller{AccountID: account, OrganizationID: clinic, Admin: true}
	for name, tt := range map[string]struct {
		want error
		id   string
		c    appointment.Caller
		att  appointment.Attendance
	}{
		"another practitioner":     {condition.ErrForbidden, condID, practitioner, attendance(false, false, true)},
		"the admin":                {condition.ErrForbidden, condID, admin, attendance(false, false, true)},
		"after a revocation":       {condition.ErrConsentRequired, condID, practitioner, attendance(true, false, false)},
		"another clinic":           {condition.ErrNotFound, condID, appointment.Caller{AccountID: account, OrganizationID: unrelated}, attendance(true, false, true)},
		"another patient":          {condition.ErrNotFound, condID, appointment.Caller{AccountID: account, PatientID: unrelated}, attendance(false, true, true)},
		"no such Condition":        {condition.ErrNotFound, unrelated, practitioner, attendance(true, false, true)},
		"an account with no scope": {condition.ErrNotFound, condID, appointment.Caller{AccountID: account}, attendance(true, false, true)},
	} {
		svc, audit := reads(tt.att)
		if _, err := svc.Get(context.Background(), tt.c, tt.id); !errors.Is(err, tt.want) {
			t.Errorf("%s: Get = %v, want %v", name, err, tt.want)
		}
		if len(audit.events) != 1 || audit.events[0].Outcome == auditevent.OutcomeSuccess {
			t.Fatalf("%s: audit = %+v, want the refusal", name, audit.events)
		}
		// A refusal of a Condition the caller sees names it, under its clinic; one it does not see names nothing.
		ref, org := "", tt.c.OrganizationID
		if !errors.Is(tt.want, condition.ErrNotFound) {
			ref, org = "Condition/"+condID, clinic
		}
		if e := audit.events[0]; e.EntityReference != ref || e.OrganizationID != org {
			t.Errorf("%s: audited %q under %q, want %q under %q", name, e.EntityReference, e.OrganizationID, ref, org)
		}
	}
}
