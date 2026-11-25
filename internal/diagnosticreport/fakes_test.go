package diagnosticreport_test

import (
	"context"
	"strconv"
	"time"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/diagnosticreport"
)

// The synthetic ids of the tests.
const (
	clinic       = "00000000-0000-4000-8000-0000000000c1"
	practitioner = "00000000-0000-4000-8000-0000000000a1"
	admin        = "00000000-0000-4000-8000-0000000000a2"
	patientID    = "00000000-0000-4000-8000-0000000000b1"
	visitID      = "00000000-0000-4000-8000-0000000000d1"
)

// fakeRepo keeps reports in memory, in the scope last set.
type fakeRepo struct {
	reports map[string]diagnosticreport.Report
	org     string
	patient string
	next    int
}

func newFakeRepo() *fakeRepo { return &fakeRepo{reports: map[string]diagnosticreport.Report{}} }

func (f *fakeRepo) SetOrganization(_ context.Context, id string) error { f.org = id; return nil }
func (f *fakeRepo) SetPatient(_ context.Context, id string) error      { f.patient = id; return nil }

func (f *fakeRepo) Insert(_ context.Context, r diagnosticreport.Report, _ string) (diagnosticreport.Report, error) {
	f.next++
	r.ID = "00000000-0000-4000-8000-" + leftPad(strconv.Itoa(f.next))
	r.VersionID = 1
	r.Status = diagnosticreport.StatusPreliminary
	f.reports[r.ID] = r
	return r, nil
}

func leftPad(s string) string {
	for len(s) < 12 {
		s = "0" + s
	}
	return s
}

func (f *fakeRepo) Update(_ context.Context, r diagnosticreport.Report) (bool, error) {
	old, ok := f.reports[r.ID]
	if !ok || old.VersionID != r.VersionID-1 {
		return false, nil
	}
	f.reports[r.ID] = r
	return true, nil
}

func (f *fakeRepo) Release(_ context.Context, id, _ string, version int, at time.Time) (bool, error) {
	r, ok := f.reports[id]
	if !ok || r.VersionID != version {
		return false, nil
	}
	r.Status, r.ReleasedAt, r.VersionID = diagnosticreport.StatusFinal, &at, version+1
	f.reports[id] = r
	return true, nil
}

func (f *fakeRepo) visible(r diagnosticreport.Report) bool {
	if f.patient != "" {
		return r.PatientID == f.patient && r.Status == diagnosticreport.StatusFinal
	}
	return r.OrganizationID == f.org
}

func (f *fakeRepo) ByID(_ context.Context, id string) (diagnosticreport.Report, error) {
	r, ok := f.reports[id]
	if !ok || !f.visible(r) {
		return diagnosticreport.Report{}, diagnosticreport.ErrNotFound
	}
	return r, nil
}

func (f *fakeRepo) ForAppointment(_ context.Context, id string) ([]diagnosticreport.Report, error) {
	var out []diagnosticreport.Report
	for _, r := range f.reports {
		if r.AppointmentID == id && f.visible(r) {
			out = append(out, r)
		}
	}
	return out, nil
}

func (f *fakeRepo) ForPatient(_ context.Context, id string) ([]diagnosticreport.Report, error) {
	var out []diagnosticreport.Report
	for _, r := range f.reports {
		if r.PatientID == id && f.visible(r) {
			out = append(out, r)
		}
	}
	return out, nil
}

// fakeVisits answers one visit: Full to its practitioner and the admin with an active consent, Name after a
// revocation (a past visit), nothing to other staff.
type fakeVisits struct {
	consent bool
}

func (f *fakeVisits) Attendance(_ context.Context, c appointment.Caller, id string) (appointment.Attendance, error) {
	if id != visitID || c.OrganizationID != clinic {
		return appointment.Attendance{}, appointment.ErrNotFound
	}
	level := access.Full
	if !f.consent {
		level = access.Name
	}
	if !c.Admin && c.PractitionerID != practitioner {
		return appointment.Attendance{}, appointment.ErrNotFound
	}
	return appointment.Attendance{
		Appointment: appointment.Appointment{ID: visitID, PatientID: patientID, OrganizationID: clinic},
		Level:       level, ConsentActive: f.consent, Practitioner: c.PractitionerID == practitioner,
	}, nil
}

// fakeAudit keeps what was recorded, and what a transaction rolled back is dropped by fakeTx.
type fakeAudit struct {
	events []auditevent.AuditEvent
}

func (f *fakeAudit) Record(_ context.Context, e auditevent.AuditEvent) error {
	f.events = append(f.events, e)
	return nil
}

// fakeTx runs fn and drops the audit events of a transaction that failed.
type fakeTx struct {
	audit *fakeAudit
}

func (t fakeTx) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	n := len(t.audit.events)
	err := fn(ctx)
	if err != nil {
		t.audit.events = t.audit.events[:n]
	}
	return err
}
