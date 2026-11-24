package clinicexport

import (
	"context"
	"fmt"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/invoice"
	"github.com/Shmeeshmee/salus/internal/patient"
)

// Record is what the clinic holds of a patient it registered, as it hands it over (docs/clinic-cms R2, V2) and as its
// data export writes it (M2): the profile, the consents the patient gave the clinic, their appointments (those a
// withdrawal of consent cancelled included, docs/clinic-cms/WAVE5.md S11) and the
// receipts the clinic issued to them. Clinical documents, questionnaire answers and the history are left out until
// the owner and a lawyer decide who may hand them over (docs/clinic-cms/PROGRESS.md, wave 4 integration).
type Record struct {
	Patient  patient.ClinicRecord
	Visits   []appointment.Seen
	Receipts []invoice.Invoice
}

// PatientRecord returns, to the clinic's admin c, the record of the patient id the clinic registered, to hand over to
// the patient (GET /clinic/patients/{id}/record), every read audited with the patient's request as its purpose where
// the owning service records one. patient.ErrForbidden to anyone but the admin and patient.ErrNotFound for a profile
// the clinic did not register, each audited by the patient domain; nothing else is read then.
func (s *Service) PatientRecord(ctx context.Context, c Caller, id string) (Record, error) {
	// No transaction around the reads: each owning service runs its own, so a refusal's audit event commits even
	// though the call fails (a joined outer transaction would roll it back with the error).
	out, err := s.record(ctx, c.AccountID, c.OrganizationID, c.Admin, id, patient.PurposeHandOver)
	if err != nil {
		return Record{}, fmt.Errorf("clinicexport: patient record: %w", err)
	}
	return out, nil
}

// record reads the record of the patient id as the account of the clinic org (its admin when admin), for the purpose:
// the profile and consents first, which the patient domain refuses to anyone but the admin and for a patient the
// clinic did not register; then the patient's appointments at the clinic and the receipts it issued to them. The one
// builder of the hand-over and of the export.
func (s *Service) record(ctx context.Context, account, org string, admin bool, id, purpose string) (Record, error) {
	rec, err := s.p.Patients.ClinicRecord(ctx, patient.Caller{AccountID: account, OrganizationID: org, Admin: admin}, id, purpose)
	if err != nil {
		return Record{}, err
	}
	out := Record{Patient: rec}
	visits, err := s.p.Appointments.ClinicRecord(ctx, appointment.Caller{AccountID: account, OrganizationID: org, Admin: true}, appointment.Filter{PatientID: id})
	if err != nil {
		return Record{}, err
	}
	for _, v := range visits {
		if v.Appointment.PatientID == id && v.Appointment.OrganizationID == org { // not on the search alone
			out.Visits = append(out.Visits, v)
		}
	}
	out.Receipts, err = s.p.Receipts.OfClinicPatient(ctx, invoice.Caller{AccountID: account, OrganizationID: org, Admin: true}, id)
	if err != nil {
		return Record{}, err
	}
	return out, nil
}
