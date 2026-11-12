package clinicstats_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/clinicstats"
)

// fakeRepo answers every read with its canned figures and keeps the clinic set and the queries asked.
type fakeRepo struct {
	err      error
	scope    string
	asked    []any // the AppointmentQuery or DiagnosisQuery of each read
	rows     []clinicstats.Count
	series   []clinicstats.Period
	heatmap  []clinicstats.HeatCell
	totals   []clinicstats.PractitionerTotal
	readOrgs []string
}

func (f *fakeRepo) SetClinic(_ context.Context, org string) error {
	f.scope = org
	return nil
}

func (f *fakeRepo) Series(_ context.Context, org string, q clinicstats.AppointmentQuery) ([]clinicstats.Period, error) {
	f.readOrgs, f.asked = append(f.readOrgs, org), append(f.asked, q)
	return f.series, f.err
}

func (f *fakeRepo) Heatmap(_ context.Context, org string, q clinicstats.AppointmentQuery) ([]clinicstats.HeatCell, error) {
	f.readOrgs, f.asked = append(f.readOrgs, org), append(f.asked, q)
	return f.heatmap, nil
}

func (f *fakeRepo) Practitioners(_ context.Context, org string, q clinicstats.AppointmentQuery) ([]clinicstats.PractitionerTotal, error) {
	f.readOrgs, f.asked = append(f.readOrgs, org), append(f.asked, q)
	return f.totals, nil
}

func (f *fakeRepo) Diagnoses(_ context.Context, org string, q clinicstats.DiagnosisQuery) ([]clinicstats.Count, error) {
	f.readOrgs, f.asked = append(f.readOrgs, org), append(f.asked, q)
	return f.rows, f.err
}

type readFixture struct {
	repo  *fakeRepo
	audit *fakeAuditor
	svc   *clinicstats.Service
}

func newReadFixture() *readFixture {
	f := &readFixture{
		repo: &fakeRepo{
			series:  []clinicstats.Period{{Period: "2026-01", Booked: 3, Fulfilled: 2, Cancelled: 1}},
			heatmap: []clinicstats.HeatCell{{Weekday: 1, Hour: 9, Count: 2}},
			totals:  []clinicstats.PractitionerTotal{{PractitionerRoleID: roleA1, Fulfilled: 2}},
			rows:    []clinicstats.Count{{Key: "respiratory-infections", Count: 31}},
		},
		audit: &fakeAuditor{},
	}
	f.svc = clinicstats.NewService(clinicstats.Ports{Repo: f.repo, Audit: f.audit, Tx: &fakeTx{}})
	return f
}

var (
	admin      = clinicstats.Caller{AccountID: "acc-admin", OrganizationID: clinicA, Admin: true}
	doctor     = clinicstats.Caller{AccountID: "acc-doc", OrganizationID: clinicA, Practitioner: true, PractitionerRoleID: roleA1}
	adminAlone = clinicstats.Caller{AccountID: "acc-adm-doc", OrganizationID: clinicA, Admin: true, Practitioner: true, PractitionerRoleID: roleA2}
	roleless   = clinicstats.Caller{AccountID: "acc-new", OrganizationID: clinicA, Practitioner: true}
	staffOnly  = clinicstats.Caller{AccountID: "acc-staff", OrganizationID: clinicA}
	aPatient   = clinicstats.Caller{AccountID: "acc-patient"}
	year       = clinicstats.AppointmentQuery{From: day(2026, 1, 1), To: day(2026, 12, 31)}
)

func diagnoses(by clinicstats.Breakdown) clinicstats.DiagnosisQuery {
	return clinicstats.DiagnosisQuery{From: day(2026, 1, 1), To: day(2026, 12, 31), By: by}
}

// checkAudit wants one event of the read of the clinic's figures by c with outcome.
func checkAudit(t *testing.T, f *readFixture, c clinicstats.Caller, outcome auditevent.Outcome) {
	t.Helper()
	if len(f.audit.events) != 1 {
		t.Fatalf("%d audit events, want 1", len(f.audit.events))
	}
	e := f.audit.events[0]
	ref := ""
	if c.OrganizationID != "" {
		ref = "Organization/" + c.OrganizationID
	}
	if e.EntityReference != ref || e.AgentAccountID != c.AccountID || e.OrganizationID != c.OrganizationID ||
		e.Purpose != (auditevent.Coding{System: auditevent.SystemActReason, Code: auditevent.CodeHOPERAT}) ||
		e.Subtype.Code != auditevent.CodeSearchType || e.Action != auditevent.ActionExecute || e.Outcome != outcome {
		t.Errorf("audit %+v, want the search of %s by %s with outcome %d", e, ref, c.AccountID, outcome)
	}
}

// C1, C3: an admin gets their own clinic's appointment figures, the breakdown per practitioner included, read in
// their clinic's scope and audited.
func TestAppointments_AnAdminGetsEverythingOfTheirClinic(t *testing.T) {
	t.Parallel()
	f := newReadFixture()
	got, err := f.svc.Appointments(t.Context(), admin, year)
	if err != nil {
		t.Fatal(err)
	}
	if !got.PerPractitioner || len(got.Practitioners) != 1 || len(got.Series) != 1 || len(got.Heatmap) != 1 {
		t.Errorf("Appointments() = %+v, want the series, the heatmap and the practitioners", got)
	}
	if f.repo.scope != clinicA || slices.ContainsFunc(f.repo.readOrgs, func(o string) bool { return o != clinicA }) {
		t.Errorf("scope %q, read %v; want only clinic A", f.repo.scope, f.repo.readOrgs)
	}
	checkAudit(t, f, admin, auditevent.OutcomeSuccess)
}

// C2: a practitioner who is not an admin gets the clinic's appointment figures, alone or for their own role, never
// the breakdown per practitioner nor another role's figures (refused and audited).
func TestAppointments_APractitionerGetsTheClinicsFiguresOrTheirOwn(t *testing.T) {
	t.Parallel()
	for name, tt := range map[string]struct {
		want error
		role string
	}{
		"the clinic's": {nil, ""},
		"their own":    {nil, roleA1},
		"another's":    {clinicstats.ErrForbidden, roleA2},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			f := newReadFixture()
			q := year
			q.PractitionerRoleID = tt.role
			got, err := f.svc.Appointments(t.Context(), doctor, q)
			if !errors.Is(err, tt.want) {
				t.Fatalf("Appointments() = %v, want %v", err, tt.want)
			}
			if got.PerPractitioner || got.Practitioners != nil {
				t.Errorf("a practitioner got the breakdown per practitioner: %+v", got)
			}
			if tt.want != nil {
				checkAudit(t, f, doctor, auditevent.OutcomeMinorFailure)
				return
			}
			if len(got.Series) != 1 || len(got.Heatmap) != 1 {
				t.Errorf("Appointments() = %+v", got)
			}
			checkAudit(t, f, doctor, auditevent.OutcomeSuccess)
		})
	}
}

// C2, owner O3: a practitioner who is not an admin gets the diagnoses of their own role only, whatever they ask
// for; another role's are refused, and so is a practitioner with no role. An admin, practitioner or not, gets the
// clinic's full totals.
func TestDiagnoses_APractitionerGetsTheirOwnRolesAndAnAdminTheClinics(t *testing.T) {
	t.Parallel()
	for name, tt := range map[string]struct {
		want     error
		role     string
		wantRole string // the role the repository is asked for
		c        clinicstats.Caller
	}{
		"a practitioner":                   {c: doctor, wantRole: roleA1},
		"a practitioner, their own role":   {c: doctor, role: roleA1, wantRole: roleA1},
		"a practitioner, another's":        {c: doctor, role: roleA2, want: clinicstats.ErrForbidden},
		"a practitioner without a role":    {c: roleless, want: clinicstats.ErrForbidden},
		"an admin":                         {c: admin},
		"an admin, a role":                 {c: admin, role: roleA2, wantRole: roleA2},
		"an admin who is a practitioner":   {c: adminAlone},
		"staff neither admin nor a doctor": {c: staffOnly, want: clinicstats.ErrForbidden},
		"a patient":                        {c: aPatient, want: clinicstats.ErrForbidden},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			f := newReadFixture()
			q := diagnoses(clinicstats.ByPractitionerRole)
			q.PractitionerRoleID = tt.role
			got, err := f.svc.Diagnoses(t.Context(), tt.c, q)
			if !errors.Is(err, tt.want) {
				t.Fatalf("Diagnoses() = %v, want %v", err, tt.want)
			}
			if tt.want != nil {
				if len(f.repo.asked) != 0 || got != nil {
					t.Errorf("read %v and returned %v on a refusal", f.repo.asked, got)
				}
				checkAudit(t, f, tt.c, auditevent.OutcomeMinorFailure)
				return
			}
			asked, _ := f.repo.asked[0].(clinicstats.DiagnosisQuery)
			if len(f.repo.asked) != 1 || asked.PractitionerRoleID != tt.wantRole || f.repo.scope != clinicA || len(got) != 1 {
				t.Errorf("asked %+v in %q, got %v; want the rows of role %q of clinic A", f.repo.asked, f.repo.scope, got, tt.wantRole)
			}
			checkAudit(t, f, tt.c, auditevent.OutcomeSuccess)
		})
	}
}

// The appointment figures are for staff of a clinic: an account that is neither its admin nor a practitioner, and a
// patient, are refused.
func TestAppointments_RefusesWhoIsNotAdminOrPractitioner(t *testing.T) {
	t.Parallel()
	for _, c := range []clinicstats.Caller{staffOnly, aPatient} {
		f := newReadFixture()
		if _, err := f.svc.Appointments(t.Context(), c, year); !errors.Is(err, clinicstats.ErrForbidden) {
			t.Errorf("%s: %v, want ErrForbidden", c.AccountID, err)
		}
		checkAudit(t, f, c, auditevent.OutcomeMinorFailure)
	}
}

// A range must be given, in order and at most ten years long, and a breakdown be one of the six; such a query
// reads nothing.
func TestQueries_RefuseAnInvalidRangeOrBreakdown(t *testing.T) {
	t.Parallel()
	for name, q := range map[string]clinicstats.DiagnosisQuery{
		"no range":       {By: clinicstats.ByGroup},
		"no end":         {From: day(2026, 1, 1), By: clinicstats.ByGroup},
		"reversed":       {From: day(2026, 2, 1), To: day(2026, 1, 1), By: clinicstats.ByGroup},
		"over ten years": {From: day(2016, 1, 1), To: day(2026, 1, 2), By: clinicstats.ByGroup},
		"no breakdown":   {From: day(2026, 1, 1), To: day(2026, 1, 1)},
		"a breakdown":    {From: day(2026, 1, 1), To: day(2026, 1, 1), By: "patient"},
	} {
		f := newReadFixture()
		if _, err := f.svc.Diagnoses(t.Context(), admin, q); !errors.Is(err, clinicstats.ErrInvalidQuery) {
			t.Errorf("%s: Diagnoses() = %v, want ErrInvalidQuery", name, err)
		}
		if q.By == clinicstats.ByGroup {
			a := clinicstats.AppointmentQuery{From: q.From, To: q.To}
			if _, err := f.svc.Appointments(t.Context(), admin, a); !errors.Is(err, clinicstats.ErrInvalidQuery) {
				t.Errorf("%s: Appointments() = %v, want ErrInvalidQuery", name, err)
			}
		}
		if len(f.repo.asked) != 0 || len(f.audit.events) != 0 {
			t.Errorf("%s: read %v, audited %d", name, f.repo.asked, len(f.audit.events))
		}
	}
	f := newReadFixture()
	if _, err := f.svc.Diagnoses(t.Context(), admin, clinicstats.DiagnosisQuery{From: day(2016, 1, 2), To: day(2026, 1, 1), By: clinicstats.ByMonth}); err != nil {
		t.Errorf("ten years: %v", err)
	}
}

// A failing read is an error, not a refusal.
func TestQueries_ReturnARepositoryError(t *testing.T) {
	t.Parallel()
	f := newReadFixture()
	boom := errors.New("boom")
	f.repo.err = boom
	if _, err := f.svc.Diagnoses(t.Context(), admin, diagnoses(clinicstats.ByCode)); !errors.Is(err, boom) {
		t.Errorf("Diagnoses() = %v", err)
	}
	if _, err := f.svc.Appointments(t.Context(), admin, year); !errors.Is(err, boom) {
		t.Errorf("Appointments() = %v", err)
	}
}
