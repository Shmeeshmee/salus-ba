package clinicexport

import (
	"context"
	"errors"
	"fmt"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/consent"
	"github.com/Shmeeshmee/salus/internal/invoice"
	"github.com/Shmeeshmee/salus/internal/patient"
)

// OwnProfile reads the calling patient's own profile, audited (patient.Service).
type OwnProfile interface {
	Me(ctx context.Context, c patient.Caller) (patient.Patient, error)
}

// OwnConsents reads the calling patient's own consents, audited (consent.Service).
type OwnConsents interface {
	Mine(ctx context.Context, c consent.Caller) ([]consent.Consent, error)
}

// OwnAppointments reads the calling patient's own appointments, every one of theirs, each audited (appointment.Service).
type OwnAppointments interface {
	Search(ctx context.Context, c appointment.Caller, f appointment.Filter) ([]appointment.Seen, error)
}

// OwnReceipts reads the receipts issued to the calling patient, audited (invoice.Service).
type OwnReceipts interface {
	Mine(ctx context.Context, c invoice.Caller) ([]invoice.Invoice, error)
}

// OwnPorts are the services a patient reads their own record through in the app.
type OwnPorts struct {
	Profile      OwnProfile
	Consents     OwnConsents
	Appointments OwnAppointments
	Receipts     OwnReceipts
}

// OwnRecords is the record builder in the patient's own scope (docs/clinic-cms/WAVE5.md D2, S4): the hand-over's
// Bundle, read as the patient through the services they read in the app, so every read is audited as the patient's
// own request.
type OwnRecords struct {
	p OwnPorts
}

// NewOwnRecords returns the builder on its ports.
func NewOwnRecords(p OwnPorts) *OwnRecords {
	return &OwnRecords{p: p}
}

// Bundle returns the record of the patient of account as a FHIR Bundle of type collection (the hand-over's file): their
// profile, and their consents, appointments (those a withdrawal cancelled included, S11) and receipts, from every
// clinic when org is empty and from the clinic org otherwise. Nil, and nothing more read, for an account with no
// profile.
func (o *OwnRecords) Bundle(ctx context.Context, account, org string) ([]byte, error) {
	me, err := o.p.Profile.Me(ctx, patient.Caller{AccountID: account})
	if errors.Is(err, patient.ErrNotFound) {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("clinicexport: own record: %w", err)
	}
	rec, err := o.read(ctx, account, me)
	if err != nil {
		return nil, fmt.Errorf("clinicexport: own record: %w", err)
	}
	if org != "" {
		rec = rec.at(org)
	}
	return bundleJSON(rec)
}

// read reads the patient me's consents, appointments and receipts as the patient.
func (o *OwnRecords) read(ctx context.Context, account string, me patient.Patient) (Record, error) {
	out := Record{Patient: patient.ClinicRecord{Patient: me}}
	var err error
	if out.Patient.Consents, err = o.p.Consents.Mine(ctx, consent.Caller{AccountID: account, PatientID: me.ID}); err != nil {
		return Record{}, err
	}
	if out.Visits, err = o.p.Appointments.Search(ctx, appointment.Caller{AccountID: account, PatientID: me.ID},
		appointment.Filter{PatientID: me.ID}); err != nil {
		return Record{}, err
	}
	if out.Receipts, err = o.p.Receipts.Mine(ctx, invoice.Caller{AccountID: account, PatientID: me.ID}); err != nil {
		return Record{}, err
	}
	return out, nil
}

// at is the record r of the clinic org only: the profile, and that clinic's consents, visits and receipts.
func (r Record) at(org string) Record {
	out := Record{Patient: patient.ClinicRecord{Patient: r.Patient.Patient}}
	for _, c := range r.Patient.Consents {
		if c.OrganizationID == org {
			out.Patient.Consents = append(out.Patient.Consents, c)
		}
	}
	for _, v := range r.Visits {
		if v.Appointment.OrganizationID == org {
			out.Visits = append(out.Visits, v)
		}
	}
	for _, i := range r.Receipts {
		if i.OrganizationID == org {
			out.Receipts = append(out.Receipts, i)
		}
	}
	return out
}
