package history_test

import (
	"cmp"
	"context"
	"errors"
	"slices"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/binary"
	"github.com/Shmeeshmee/salus/internal/composition"
	"github.com/Shmeeshmee/salus/internal/encounter"
	"github.com/Shmeeshmee/salus/internal/history"
	"github.com/Shmeeshmee/salus/internal/practitionerrole"
	"github.com/Shmeeshmee/salus/internal/questionnaireresponse"
	"github.com/Shmeeshmee/salus/pkg/keyset"
)

// Synthetic ids.
const (
	clinicA  = "a0000000-0000-4000-8000-00000000000a"
	clinicB  = "b0000000-0000-4000-8000-00000000000b"
	patientA = "aa000000-0000-4000-8000-0000000000aa"
	accountA = "acc00000-0000-4000-8000-0000000000aa"
	roleA    = "a1000000-0000-4000-8000-0000000000a1"
	roleB    = "b1000000-0000-4000-8000-0000000000b1"
)

var (
	me         = appointment.Caller{AccountID: accountA, PatientID: patientA}
	pract      = appointment.Caller{AccountID: "ac100000-0000-4000-8000-0000000000a1", OrganizationID: clinicA, PractitionerID: "p1"}
	errBackend = errors.New("the database is down")
	day        = time.Date(2030, 1, 7, 9, 0, 0, 0, time.UTC)
)

// asked is one call a source received.
type asked struct {
	caller appointment.Caller
	org    string
	page   keyset.Page
}

// page keeps what is after p's position, in its range, newest first then by reference descending, p.Limit at most,
// as each owning domain's query does.
func page[T any](all []T, p keyset.Page, key func(T) (time.Time, string)) []T {
	var out []T
	for _, x := range all {
		at, ref := key(x)
		after := p.At.IsZero() || at.Before(p.At) || (at.Equal(p.At) && ref < p.Ref)
		if after && (p.From.IsZero() || !at.Before(p.From)) && (p.To.IsZero() || at.Before(p.To)) {
			out = append(out, x)
		}
	}
	slices.SortFunc(out, func(a, b T) int {
		at, ar := key(a)
		bt, br := key(b)
		if c := bt.Compare(at); c != 0 {
			return c
		}
		return cmp.Compare(br, ar)
	})
	return out[:min(len(out), p.Limit)]
}

// txKey marks the context fakeTx hands its function, as postgres.Transactor carries its transaction.
type txKey struct{}

func inTx(ctx context.Context) bool { return ctx.Value(txKey{}) != nil }

// fakeTx counts its transactions.
type fakeTx struct{ n int }

func (f *fakeTx) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	f.n++
	return fn(context.WithValue(ctx, txKey{}, true))
}

// fakeAudit keeps each RecordAll's events and whether it ran in the transaction.
type fakeAudit struct {
	err     error
	batches [][]auditevent.AuditEvent
	inTx    []bool
}

func (f *fakeAudit) RecordAll(ctx context.Context, es []auditevent.AuditEvent) error {
	f.batches, f.inTx = append(f.batches, es), append(f.inTx, inTx(ctx))
	return f.err
}

type fakeVisits struct {
	err   error
	all   []encounter.Encounter
	asked []asked
	inTx  []bool
}

func (f *fakeVisits) PatientHistory(ctx context.Context, c appointment.Caller, org string, p keyset.Page) ([]encounter.Encounter, error) {
	f.asked, f.inTx = append(f.asked, asked{c, org, p}), append(f.inTx, inTx(ctx))
	return page(f.all, p, func(e encounter.Encounter) (time.Time, string) { return e.PeriodStart, "Encounter/" + e.ID }), f.err
}

type fakeDocuments struct {
	err   error
	all   []composition.Document
	asked []asked
}

func (f *fakeDocuments) PatientHistory(_ context.Context, c appointment.Caller, org string, p keyset.Page) ([]composition.Document, error) {
	f.asked = append(f.asked, asked{c, org, p})
	return page(f.all, p, func(d composition.Document) (time.Time, string) { return d.Draft.SignedAt, "Composition/" + d.Draft.ID }), f.err
}

type fakeResponses struct {
	err      error
	all      []questionnaireresponse.Response
	files    []questionnaireresponse.Attached
	asked    []asked
	askedFor []asked
}

func (f *fakeResponses) PatientHistory(_ context.Context, c appointment.Caller, org string, p keyset.Page) ([]questionnaireresponse.Response, error) {
	f.asked = append(f.asked, asked{c, org, p})
	return page(f.all, p, func(r questionnaireresponse.Response) (time.Time, string) {
		return r.AuthoredAt, "QuestionnaireResponse/" + r.ID
	}), f.err
}

func (f *fakeResponses) PatientAttachments(_ context.Context, c appointment.Caller, org string, p keyset.Page) ([]questionnaireresponse.Attached, error) {
	f.askedFor = append(f.askedFor, asked{c, org, p})
	return page(f.files, p, func(a questionnaireresponse.Attached) (time.Time, string) {
		return a.Response.AuthoredAt, "Binary/" + a.File.BinaryID
	}), f.err
}

// fakeFiles finds every file clean but those it names.
type fakeFiles struct {
	err     error
	unclean []string
	callers []binary.Caller
	asked   [][]string
}

func (f *fakeFiles) Clean(_ context.Context, c binary.Caller, ids []string) ([]string, error) {
	f.callers, f.asked = append(f.callers, c), append(f.asked, ids)
	return slices.DeleteFunc(slices.Clone(ids), func(id string) bool { return slices.Contains(f.unclean, id) }), f.err
}

type fakeClinics struct {
	err   error
	asked [][]string
}

func (f *fakeClinics) Names(_ context.Context, ids []string) (map[string]string, error) {
	f.asked = append(f.asked, ids)
	names := map[string]string{clinicA: "Synthetic clinic A", clinicB: "Synthetic clinic B"}
	out := map[string]string{}
	for _, id := range ids {
		if n, ok := names[id]; ok {
			out[id] = n
		}
	}
	return out, f.err
}

// fakeRoles gives role A a specialty, role B none.
type fakeRoles struct {
	err   error
	asked []map[string][]string
}

func (f *fakeRoles) LookupAll(_ context.Context, ids map[string][]string) (map[string]practitionerrole.PractitionerRole, error) {
	f.asked = append(f.asked, ids)
	out := map[string]practitionerrole.PractitionerRole{}
	for org, roles := range ids {
		for _, id := range roles {
			out[id] = practitionerrole.PractitionerRole{ID: id, OrganizationID: org}
			if id == roleA {
				out[id] = practitionerrole.PractitionerRole{ID: id, OrganizationID: org, SpecialtyCode: "394579002"}
			}
		}
	}
	return out, f.err
}

var (
	_ history.Visits     = (*fakeVisits)(nil)
	_ history.Documents  = (*fakeDocuments)(nil)
	_ history.Responses  = (*fakeResponses)(nil)
	_ history.Files      = (*fakeFiles)(nil)
	_ history.Clinics    = (*fakeClinics)(nil)
	_ history.Roles      = (*fakeRoles)(nil)
	_ history.Auditor    = (*fakeAudit)(nil)
	_ history.Transactor = (*fakeTx)(nil)
)
