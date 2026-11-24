package clinicexport_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/chargeitemdefinition"
	"github.com/Shmeeshmee/salus/internal/clinicexport"
	"github.com/Shmeeshmee/salus/internal/invoice"
	"github.com/Shmeeshmee/salus/internal/patient"
	"github.com/Shmeeshmee/salus/internal/practitioner"
	"github.com/Shmeeshmee/salus/internal/practitionerrole"
)

// fakeRepo keeps the exports in memory and shows only the scope's clinic, as row-level security does.
type fakeRepo struct {
	rows  map[string]clinicexport.Export
	scope string
	next  int
	mu    sync.Mutex
}

func newFakeRepo() *fakeRepo { return &fakeRepo{rows: map[string]clinicexport.Export{}} }

func (r *fakeRepo) SetOrganization(_ context.Context, id string) error {
	r.scope = id
	return nil
}

func (r *fakeRepo) Insert(_ context.Context, e clinicexport.Export) (clinicexport.Export, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, row := range r.rows {
		if row.OrganizationID == e.OrganizationID && row.Status == clinicexport.StatusQueued {
			return clinicexport.Export{}, false, nil
		}
	}
	r.next++
	e.ID = "00000000-0000-4000-8000-" + leftPad(r.next)
	e.Status = clinicexport.StatusQueued
	r.rows[e.ID] = e
	return e, true, nil
}

func leftPad(n int) string {
	s := strconv.Itoa(n)
	for len(s) < 12 {
		s = "0" + s
	}
	return s
}

func (r *fakeRepo) Queued(_ context.Context, org string) (clinicexport.Export, error) {
	for _, row := range r.rows {
		if row.OrganizationID == org && row.OrganizationID == r.scope && row.Status == clinicexport.StatusQueued {
			return row, nil
		}
	}
	return clinicexport.Export{}, clinicexport.ErrNotFound
}

func (r *fakeRepo) ByID(_ context.Context, id string) (clinicexport.Export, error) {
	row, ok := r.rows[id]
	if !ok || row.OrganizationID != r.scope {
		return clinicexport.Export{}, clinicexport.ErrNotFound
	}
	return row, nil
}

func (r *fakeRepo) MarkReady(_ context.Context, e clinicexport.Export) (bool, error) {
	row, ok := r.rows[e.ID]
	if !ok || row.Status != clinicexport.StatusQueued || row.OrganizationID != r.scope {
		return false, nil
	}
	row.Status, row.ReadyAt, row.Size, row.KEKID, row.WrappedKey = clinicexport.StatusReady, e.ReadyAt, e.Size, e.KEKID, e.WrappedKey
	r.rows[e.ID] = row
	return true, nil
}

func (r *fakeRepo) MarkFailed(_ context.Context, id string) (bool, error) {
	row, ok := r.rows[id]
	if !ok || row.Status != clinicexport.StatusQueued || row.OrganizationID != r.scope {
		return false, nil
	}
	row.Status = clinicexport.StatusFailed
	r.rows[id] = row
	return true, nil
}

func (r *fakeRepo) RequestedBefore(_ context.Context, before time.Time, limit int32) ([]string, error) {
	var out []string
	for id, row := range r.rows {
		if row.RequestedAt.Before(before) && len(out) < int(limit) {
			out = append(out, id)
		}
	}
	slices.Sort(out)
	return out, nil
}

func (r *fakeRepo) Delete(_ context.Context, id string) (bool, error) {
	_, ok := r.rows[id]
	delete(r.rows, id)
	return ok, nil
}

type fakeTx struct{}

func (fakeTx) WithTx(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) }

type fakeAudit struct{ events []auditevent.AuditEvent }

func (a *fakeAudit) Record(_ context.Context, e auditevent.AuditEvent) error {
	a.events = append(a.events, e)
	return nil
}

type fakeJobs struct{ builds []clinicexport.BuildArgs }

func (j *fakeJobs) EnqueueBuild(_ context.Context, args clinicexport.BuildArgs) error {
	j.builds = append(j.builds, args)
	return nil
}

type fakeStore struct{ objects map[string][]byte }

func (s *fakeStore) Put(_ context.Context, key string, r io.Reader, size int64) error {
	b, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	if int64(len(b)) != size {
		return errors.New("fakeStore: size mismatch")
	}
	s.objects[key] = b
	return nil
}

func (s *fakeStore) GetRange(_ context.Context, key string, from, to int64) (io.ReadCloser, error) {
	b, ok := s.objects[key]
	if !ok {
		return nil, errors.New("fakeStore: no object")
	}
	return io.NopCloser(bytes.NewReader(b[from : to+1])), nil
}

func (s *fakeStore) Delete(_ context.Context, key string) error {
	delete(s.objects, key)
	return nil
}

// fakeSources are the clinic's data, and who read it.
type fakeSources struct {
	prices       []chargeitemdefinition.Definition
	roles        []practitionerrole.PractitionerRole
	people       map[string]practitioner.Practitioner
	appointments []appointment.Seen
	receipts     []invoice.Invoice
	quotes       []invoice.Quote
	registered   []string
	records      map[string]patient.ClinicRecord // by patient id; any other is not found
	recordErr    error                           // ClinicRecord's refusal, when set
	err          error

	roleCallers    []practitionerrole.Caller
	peopleCallers  []practitioner.Caller
	visitCallers   []appointment.Caller
	receiptCallers []invoice.Caller
	periods        []invoice.Period
	purposes       []string // ClinicRecord's, in order
	patientCalls   []patient.Caller
	filters        []appointment.Filter
	patientReceipt []string // OfClinicPatient's patient ids
}

func (f *fakeSources) OfClinic(_ context.Context, org string) ([]chargeitemdefinition.Definition, error) {
	return f.prices, f.err
}

func (f *fakeSources) Search(_ context.Context, c practitionerrole.Caller, org string) ([]practitionerrole.PractitionerRole, error) {
	f.roleCallers = append(f.roleCallers, c)
	return f.roles, nil
}

type practitioners struct{ *fakeSources }

func (f practitioners) Read(_ context.Context, c practitioner.Caller, id string) (practitioner.Practitioner, error) {
	f.peopleCallers = append(f.peopleCallers, c)
	p, ok := f.people[id]
	if !ok {
		return practitioner.Practitioner{}, practitioner.ErrNotFound
	}
	return p, nil
}

type visits struct{ *fakeSources }

// ClinicRecord is the clinic record's read (S11): what the clinic holds, the visits a withdrawal cancelled included.
func (f visits) ClinicRecord(_ context.Context, c appointment.Caller, filter appointment.Filter) ([]appointment.Seen, error) {
	f.visitCallers = append(f.visitCallers, c)
	f.filters = append(f.filters, filter)
	return f.appointments, nil
}

func (f *fakeSources) OfClinicPatient(_ context.Context, c invoice.Caller, patientID string) ([]invoice.Invoice, error) {
	f.receiptCallers = append(f.receiptCallers, c)
	f.patientReceipt = append(f.patientReceipt, patientID)
	var out []invoice.Invoice
	for _, i := range f.receipts {
		if i.PatientID == patientID {
			out = append(out, i)
		}
	}
	return out, nil
}

func (f *fakeSources) Quotes(_ context.Context, c invoice.Caller, appointmentID string) ([]invoice.Quote, error) {
	f.receiptCallers = append(f.receiptCallers, c)
	if appointmentID != "" {
		return nil, errors.New("asked for one visit's quotes")
	}
	return f.quotes, nil
}

type patients struct{ *fakeSources }

func (f patients) Registered(_ context.Context, c patient.Caller) ([]string, error) {
	f.patientCalls = append(f.patientCalls, c)
	return f.registered, nil
}

func (f patients) ClinicRecord(_ context.Context, c patient.Caller, id, purpose string) (patient.ClinicRecord, error) {
	f.patientCalls = append(f.patientCalls, c)
	f.purposes = append(f.purposes, purpose)
	if f.recordErr != nil {
		return patient.ClinicRecord{}, f.recordErr
	}
	rec, ok := f.records[id]
	if !ok {
		return patient.ClinicRecord{}, patient.ErrNotFound
	}
	return rec, nil
}

func (f *fakeSources) List(_ context.Context, c invoice.Caller, p invoice.Period) ([]invoice.Invoice, error) {
	f.receiptCallers = append(f.receiptCallers, c)
	f.periods = append(f.periods, p)
	return f.receipts, nil
}
