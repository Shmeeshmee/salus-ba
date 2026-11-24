package clinicexport_test

import (
	"errors"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/clinicexport"
	"github.com/Shmeeshmee/salus/internal/invoice"
	"github.com/Shmeeshmee/salus/internal/patient"
)

// The hand-over's record (docs/clinic-cms R2, joined with M2 in wave 4): the patient domain's record, then the
// patient's visits and receipts at the clinic, all read as the admin; the profile's read is the patient's request.
func TestPatientRecord_GivesTheAdminTheProfileVisitsAndReceiptsAtTheClinic(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	at := time.Date(2026, 11, 20, 8, 0, 0, 0, time.UTC)
	e.sources.records["pt1"] = patient.ClinicRecord{Patient: patient.Patient{ID: "pt1", RegisteredBy: clinicA, LastUpdated: at}}
	e.sources.appointments = []appointment.Seen{
		{Appointment: appointment.Appointment{ID: "a1", PatientID: "pt1", OrganizationID: clinicA}},
		{Appointment: appointment.Appointment{ID: "a2", PatientID: "pt2", OrganizationID: clinicA}}, // another patient's
		{Appointment: appointment.Appointment{ID: "a3", PatientID: "pt1", OrganizationID: clinicB}}, // another clinic's
	}
	e.sources.receipts = []invoice.Invoice{{ID: "i1", PatientID: "pt1"}, {ID: "i2", PatientID: "pt2"}}
	got, err := e.svc.PatientRecord(t.Context(), admin, "pt1")
	if err != nil || got.Patient.Patient.ID != "pt1" || len(got.Visits) != 1 || got.Visits[0].Appointment.ID != "a1" ||
		len(got.Receipts) != 1 || got.Receipts[0].ID != "i1" {
		t.Fatalf("PatientRecord = %+v, %v; want pt1 with a1 and i1", got, err)
	}
	if len(e.sources.purposes) != 1 || e.sources.purposes[0] != patient.PurposeHandOver {
		t.Errorf("read for %v, want the hand-over's purpose", e.sources.purposes)
	}
	if f := e.sources.filters; len(f) != 1 || f[0].PatientID != "pt1" {
		t.Errorf("appointments searched with %+v, want the patient's", f)
	}
}

func TestPatientRecord_ReadsNothingMoreWhenThePatientDomainRefuses(t *testing.T) {
	t.Parallel()
	for name, refusal := range map[string]error{"not the admin": patient.ErrForbidden, "not the clinic's": patient.ErrNotFound} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			e := newEnv(t)
			e.sources.recordErr = refusal
			c := admin
			c.Admin = refusal != patient.ErrForbidden
			if _, err := e.svc.PatientRecord(t.Context(), c, "pt1"); !errors.Is(err, refusal) {
				t.Fatalf("PatientRecord = %v, want %v", err, refusal)
			}
			if len(e.sources.visitCallers)+len(e.sources.receiptCallers) != 0 {
				t.Errorf("read visits %v and receipts %v after the refusal", e.sources.visitCallers, e.sources.receiptCallers)
			}
			if pc := e.sources.patientCalls; len(pc) != 1 || pc[0].Admin != c.Admin || pc[0].OrganizationID != clinicA {
				t.Errorf("asked the patient domain as %+v, want the caller", pc)
			}
		})
	}
}

var _ clinicexport.Exports = (*clinicexport.Service)(nil)
