package diagnosticreport_test

import (
	"bytes"
	"context"
	"io"
	"slices"
	"strconv"
	"time"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/binary"
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

func (f *fakeRepo) SetOrganization(_ context.Context, id string) error {
	f.org, f.patient = id, ""
	return nil
}

func (f *fakeRepo) SetPatient(_ context.Context, id string) error {
	f.patient, f.org = id, ""
	return nil
}

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

func (f *fakeRepo) Touch(_ context.Context, id string, version int) (bool, error) {
	r, ok := f.reports[id]
	if !ok || r.VersionID != version || r.Status != diagnosticreport.StatusPreliminary {
		return false, nil
	}
	r.VersionID = version + 1
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

// fakeFiles is binary's pipeline as the report sees it: a PDF from its first bytes, at most binary.MaxDocumentSize
// bytes, stored in a transaction after the report's commit, replacing the report's file; the scan is set by status.
type fakeFiles struct {
	tx           fakeTx
	beforeCommit func() // runs once the bytes are read, before the commit (another write meanwhile)
	files        map[string]binary.Binary
	bodies       map[string][]byte
	next         int
}

func newFakeFiles(tx fakeTx) *fakeFiles {
	return &fakeFiles{tx: tx, files: map[string]binary.Binary{}, bodies: map[string][]byte{}}
}

func (f *fakeFiles) UploadReport(ctx context.Context, in binary.ReportUpload, commit func(ctx context.Context, b binary.Binary) error) (binary.Binary, error) {
	switch {
	case in.Size < 0:
		return binary.Binary{}, binary.ErrLengthRequired
	case in.Size > binary.MaxDocumentSize:
		return binary.Binary{}, binary.ErrTooLarge
	}
	body, err := io.ReadAll(in.Body)
	switch {
	case err != nil:
		return binary.Binary{}, err
	case int64(len(body)) != in.Size:
		return binary.Binary{}, binary.ErrShortBody
	case !bytes.HasPrefix(body, []byte("%PDF-")):
		return binary.Binary{}, binary.ErrUnsupportedType
	}
	if f.beforeCommit != nil {
		f.beforeCommit()
	}
	f.next++
	b := binary.Binary{
		ID: "00000000-0000-4000-8000-1000000000" + leftPad(strconv.Itoa(f.next))[10:], DiagnosticReportID: in.ReportID,
		OrganizationID: in.OrganizationID, PatientID: in.PatientID, ContentType: "application/pdf", Title: in.Title, Size: in.Size,
		Status: binary.StatusQuarantined,
	}
	err = f.tx.WithTx(ctx, func(ctx context.Context) error { return commit(ctx, b) })
	if err != nil {
		return binary.Binary{}, err
	}
	f.files[in.ReportID], f.bodies[b.ID] = b, body
	return b, nil
}

// scanned sets the scan's result on the report's file.
func (f *fakeFiles) scanned(reportID string, status binary.Status) {
	b := f.files[reportID]
	b.Status = status
	f.files[reportID] = b
}

func (f *fakeFiles) ReportFiles(_ context.Context, reportIDs []string) ([]binary.Binary, error) {
	var out []binary.Binary
	for _, id := range reportIDs {
		if b, ok := f.files[id]; ok && b.Status != binary.StatusRejected {
			out = append(out, b)
		}
	}
	return out, nil
}

func (f *fakeFiles) OpenReport(_ context.Context, b binary.Binary) (binary.File, error) {
	switch b.Status {
	case binary.StatusQuarantined:
		return binary.File{}, binary.ErrScanning
	case binary.StatusClean:
		return binary.File{Binary: b}, nil
	}
	return binary.File{}, binary.ErrNotFound
}

func (f *fakeFiles) Content(_ context.Context, file binary.File, from, to int64) (io.ReadCloser, error) {
	body := f.bodies[file.ID]
	return io.NopCloser(bytes.NewReader(slices.Clone(body[from : to+1]))), nil
}
