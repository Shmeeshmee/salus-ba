package patient_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/patient"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

// Synthetic ids and data only.
const (
	accountA = "c1000000-0000-4000-8000-0000000000c1"
	accountB = "c2000000-0000-4000-8000-0000000000c2"
	patientA = "d1000000-0000-4000-8000-0000000000d1"
	patientB = "d2000000-0000-4000-8000-0000000000d2"
	clinicA  = "a0000000-0000-4000-8000-00000000000a"
	staffA   = "a1000000-0000-4000-8000-0000000000a1"
	roleA    = "e1000000-0000-4000-8000-0000000000e1" // a practitioner's role in clinic A
	roleA2   = "e2000000-0000-4000-8000-0000000000e2" // another practitioner's role in clinic A
)

var now = time.Date(2026, 11, 4, 9, 30, 0, 0, time.UTC)

func date(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }

var (
	me    = patient.Caller{AccountID: accountA}
	other = patient.Caller{AccountID: "c3000000-0000-4000-8000-0000000000c3"}
	staff = patient.Caller{AccountID: staffA, OrganizationID: clinicA}
	// practitioner holds role A in clinic A; admin is the clinic's admin.
	practitioner = patient.Caller{AccountID: staffA, OrganizationID: clinicA, PractitionerRoleID: roleA}
	admin        = patient.Caller{AccountID: staffA, OrganizationID: clinicA, Admin: true}
)

type txKey struct{}

type fakeTransactor struct{}

func (fakeTransactor) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(context.WithValue(ctx, txKey{}, true))
}

// fakeRepository holds profiles by account and shows only the account set on the transaction, as row-level
// security does.
type fakeRepository struct {
	err      error // returned by every read and write when set
	profiles map[string]patient.Patient
	account  string   // set by SetAccount
	clinic   string   // set by SetClinic
	calls    []string // the calls, "outside:" prefixed when made outside a transaction
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{profiles: map[string]patient.Patient{
		accountA: {ID: patientA, AccountID: accountA, GivenName: "Ada", FamilyName: "Example", BirthDate: date(1990, 1, 1), VersionID: 2, LastUpdated: now.Add(-time.Hour)},
		accountB: {ID: patientB, AccountID: accountB, GivenName: "Bo", FamilyName: "Example", BirthDate: date(1991, 2, 2), VersionID: 1, LastUpdated: now.Add(-time.Hour)},
	}}
}

func (r *fakeRepository) seen(ctx context.Context, call string) {
	if ctx.Value(txKey{}) == nil {
		call = "outside:" + call
	}
	r.calls = append(r.calls, call)
}

func (r *fakeRepository) SetAccount(ctx context.Context, id string) error {
	r.seen(ctx, "SetAccount")
	r.account = id
	return nil
}

func (r *fakeRepository) SetClinic(ctx context.Context, id string) error {
	r.seen(ctx, "SetClinic")
	r.clinic = id
	return nil
}

func (r *fakeRepository) ByAccount(ctx context.Context, accountID string) (patient.Patient, error) {
	r.seen(ctx, "ByAccount")
	if r.err != nil {
		return patient.Patient{}, r.err
	}
	p, ok := r.profiles[accountID]
	if !ok || accountID != r.account {
		return patient.Patient{}, patient.ErrNotFound
	}
	return p, nil
}

func (r *fakeRepository) ByID(ctx context.Context, id string) (patient.Patient, error) {
	r.seen(ctx, "ByID")
	if r.err != nil {
		return patient.Patient{}, r.err
	}
	if r.account == "" && r.clinic != "" { // a clinic's scope: the rows of active consents, all here
		for _, p := range r.profiles {
			if p.ID == id {
				return p, nil
			}
		}
	}
	p, ok := r.profiles[r.account]
	if !ok || p.ID != id {
		return patient.Patient{}, patient.ErrNotFound
	}
	return p, nil
}

func (r *fakeRepository) Create(ctx context.Context, p patient.Patient, at time.Time) (patient.Patient, error) {
	r.seen(ctx, "Create")
	if r.err != nil {
		return patient.Patient{}, r.err
	}
	if _, ok := r.profiles[p.AccountID]; ok || p.AccountID != r.account {
		return patient.Patient{}, patient.ErrConflict
	}
	p.ID, p.VersionID, p.LastUpdated = "d9000000-0000-4000-8000-0000000000d9", 1, at
	r.profiles[p.AccountID] = p
	return p, nil
}

func (r *fakeRepository) Update(ctx context.Context, p patient.Patient, at time.Time) (patient.Patient, error) {
	r.seen(ctx, "Update")
	if r.err != nil {
		return patient.Patient{}, r.err
	}
	cur, ok := r.profiles[p.AccountID]
	if !ok || p.AccountID != r.account || cur.VersionID != p.VersionID {
		return patient.Patient{}, patient.ErrConflict
	}
	p.ID, p.VersionID, p.LastUpdated = cur.ID, cur.VersionID+1, at
	r.profiles[p.AccountID] = p
	return p, nil
}

type fakeAuditor struct {
	err    error
	events []auditevent.AuditEvent
}

func (a *fakeAuditor) Record(_ context.Context, e auditevent.AuditEvent) error {
	if a.err != nil {
		return a.err
	}
	a.events = append(a.events, e)
	return nil
}

// fakeConsents is consent.Service.State: the given state, whatever is asked.
type fakeConsents struct {
	err   error
	args  []string
	state access.Consent
}

func (f *fakeConsents) State(_ context.Context, patientID, organizationID string) (access.Consent, error) {
	f.args = append(f.args, patientID, organizationID)
	return f.state, f.err
}

// fakeAppointments gives the patient's appointments in the clinic: the given ones, whatever is asked.
type fakeAppointments struct {
	err    error
	visits []access.Visit
	args   []string
}

func (f *fakeAppointments) Visits(_ context.Context, patientID, organizationID string) ([]access.Visit, error) {
	f.args = append(f.args, patientID, organizationID)
	return f.visits, f.err
}

func newService(repo *fakeRepository, audit *fakeAuditor) *patient.Service {
	return patient.NewService(repo, audit, fakeTransactor{}, clock.Fixed{T: now}, &fakeConsents{}, &fakeAppointments{})
}

// event is the audit event of interaction by c on Patient/id ("" for none).
func event(c patient.Caller, interaction, id string, outcome auditevent.Outcome) auditevent.AuditEvent {
	action := map[string]auditevent.Action{
		auditevent.CodeRead: auditevent.ActionRead, auditevent.CodeCreate: auditevent.ActionCreate,
		auditevent.CodeUpdate: auditevent.ActionUpdate,
	}[interaction]
	purpose := auditevent.CodePATRQT
	if c.OrganizationID != "" {
		purpose = auditevent.CodeTREAT
	}
	e := auditevent.AuditEvent{
		Type:           auditevent.Coding{System: auditevent.SystemAuditEventType, Code: auditevent.CodeRest},
		Subtype:        auditevent.Coding{System: auditevent.SystemRestfulInteraction, Code: interaction},
		Purpose:        auditevent.Coding{System: auditevent.SystemActReason, Code: purpose},
		Action:         action,
		Outcome:        outcome,
		AgentAccountID: c.AccountID,
		OrganizationID: c.OrganizationID,
	}
	if id != "" {
		e.EntityReference = "Patient/" + id
	}
	return e
}

func checkEvents(t *testing.T, audit *fakeAuditor, want ...auditevent.AuditEvent) {
	t.Helper()
	if !slices.Equal(audit.events, want) {
		t.Errorf("audit events %+v,\nwant %+v", audit.events, want)
	}
}

func TestMe_GivesThePatientTheirProfileThroughTheirScopeAndAuditsIt(t *testing.T) {
	repo, audit := newFakeRepository(), &fakeAuditor{}
	got, err := newService(repo, audit).Me(t.Context(), me)
	if err != nil || got != repo.profiles[accountA] {
		t.Fatalf("Me() = %+v, %v; want profile A", got, err)
	}
	if want := []string{"SetAccount", "ByAccount"}; !slices.Equal(repo.calls, want) || repo.account != accountA {
		t.Errorf("calls %v with account %q, want %v inside the transaction with account A", repo.calls, repo.account, want)
	}
	checkEvents(t, audit, event(me, auditevent.CodeRead, patientA, auditevent.OutcomeSuccess))
}

func TestMe_IsNotFoundBeforeTheFirstSaveAndAudited(t *testing.T) {
	repo, audit := newFakeRepository(), &fakeAuditor{}
	_, err := newService(repo, audit).Me(t.Context(), other)
	if !errors.Is(err, patient.ErrNotFound) {
		t.Fatalf("Me() = %v, want ErrNotFound", err)
	}
	checkEvents(t, audit, event(other, auditevent.CodeRead, "", auditevent.OutcomeMinorFailure))
}

func TestMe_RefusesAndAuditsStaff(t *testing.T) {
	repo, audit := newFakeRepository(), &fakeAuditor{}
	_, err := newService(repo, audit).Me(t.Context(), staff)
	if !errors.Is(err, patient.ErrForbidden) || len(repo.calls) != 0 {
		t.Fatalf("Me() as staff = %v after calls %v, want ErrForbidden without touching the repository", err, repo.calls)
	}
	checkEvents(t, audit, event(staff, auditevent.CodeRead, "", auditevent.OutcomeMinorFailure))
}

func TestSaveMe_CreatesTheProfileOnTheFirstSaveTrimmedAndAudited(t *testing.T) {
	repo, audit := newFakeRepository(), &fakeAuditor{}
	in := patient.Patient{GivenName: "  Cy ", FamilyName: " Example", BirthDate: date(1985, 6, 15), Phone: "+38760000001"}
	got, err := newService(repo, audit).SaveMe(t.Context(), other, in)
	want := patient.Patient{ID: "d9000000-0000-4000-8000-0000000000d9", AccountID: other.AccountID, GivenName: "Cy",
		FamilyName: "Example", BirthDate: date(1985, 6, 15), Phone: "+38760000001", VersionID: 1, LastUpdated: now}
	if err != nil || got != want {
		t.Fatalf("SaveMe() = %+v, %v;\nwant %+v", got, err, want)
	}
	if want := []string{"SetAccount", "Create"}; !slices.Equal(repo.calls, want) || repo.account != other.AccountID {
		t.Errorf("calls %v with account %q, want %v inside the transaction with the caller's account", repo.calls, repo.account, want)
	}
	checkEvents(t, audit, event(other, auditevent.CodeCreate, want.ID, auditevent.OutcomeSuccess))
}

func TestSaveMe_UpdatesTheCurrentVersionAndAuditsIt(t *testing.T) {
	repo, audit := newFakeRepository(), &fakeAuditor{}
	in := patient.Patient{GivenName: "Ada", FamilyName: "Renamed", BirthDate: date(1990, 1, 1), VersionID: 2}
	got, err := newService(repo, audit).SaveMe(t.Context(), me, in)
	if err != nil || got.ID != patientA || got.FamilyName != "Renamed" || got.VersionID != 3 || !got.LastUpdated.Equal(now) {
		t.Fatalf("SaveMe() = %+v, %v; want profile A renamed at version 3", got, err)
	}
	if want := []string{"SetAccount", "Update"}; !slices.Equal(repo.calls, want) {
		t.Errorf("calls %v, want %v", repo.calls, want)
	}
	checkEvents(t, audit, event(me, auditevent.CodeUpdate, patientA, auditevent.OutcomeSuccess))
}

func TestSaveMe_GivesAConflictForAStaleVersionOrASecondCreate(t *testing.T) {
	tests := []struct {
		name        string
		interaction string
		version     int32
	}{
		{name: "stale version", version: 1, interaction: auditevent.CodeUpdate},
		{name: "a version from the future", version: 3, interaction: auditevent.CodeUpdate},
		{name: "a create when the profile exists", version: 0, interaction: auditevent.CodeCreate},
	}
	for _, tt := range tests {
		repo, audit := newFakeRepository(), &fakeAuditor{}
		in := patient.Patient{GivenName: "Ada", FamilyName: "Example", BirthDate: date(1990, 1, 1), VersionID: tt.version}
		_, err := newService(repo, audit).SaveMe(t.Context(), me, in)
		if !errors.Is(err, patient.ErrConflict) {
			t.Errorf("%s: SaveMe() = %v, want ErrConflict", tt.name, err)
		}
		if repo.profiles[accountA].VersionID != 2 {
			t.Errorf("%s: profile changed to %+v", tt.name, repo.profiles[accountA])
		}
		checkEvents(t, audit, event(me, tt.interaction, "", auditevent.OutcomeMinorFailure))
	}
}

func TestSaveMe_RefusesAndAuditsStaff(t *testing.T) {
	repo, audit := newFakeRepository(), &fakeAuditor{}
	in := patient.Patient{GivenName: "Ada", FamilyName: "Example", BirthDate: date(1990, 1, 1)}
	_, err := newService(repo, audit).SaveMe(t.Context(), staff, in)
	if !errors.Is(err, patient.ErrForbidden) || len(repo.calls) != 0 {
		t.Fatalf("SaveMe() as staff = %v after calls %v, want ErrForbidden without touching the repository", err, repo.calls)
	}
	checkEvents(t, audit, event(staff, auditevent.CodeCreate, "", auditevent.OutcomeMinorFailure))
}

func TestSaveMe_ValidatesTheProfileBeforeAnyAccess(t *testing.T) {
	valid := patient.Patient{GivenName: "Ada", FamilyName: "Example", BirthDate: date(1990, 1, 1)}
	with := func(f func(p *patient.Patient)) patient.Patient {
		p := valid
		f(&p)
		return p
	}
	hundred := strings.Repeat("é", 100)
	tests := []struct {
		want error // nil: valid
		name string
		in   patient.Patient
	}{
		{name: "valid, no phone", in: valid},
		{name: "names of 1 and 100 characters", in: with(func(p *patient.Patient) { p.GivenName, p.FamilyName = "A", hundred })},
		{name: "a name of 100 characters inside spaces", in: with(func(p *patient.Patient) { p.GivenName = "  " + hundred + " " })},
		{name: "blank given name", in: with(func(p *patient.Patient) { p.GivenName = "   " }), want: patient.ErrInvalidName},
		{name: "empty family name", in: with(func(p *patient.Patient) { p.FamilyName = "" }), want: patient.ErrInvalidName},
		{name: "a name of 101 characters", in: with(func(p *patient.Patient) { p.FamilyName = hundred + "x" }), want: patient.ErrInvalidName},
		{name: "born 1900-01-01", in: with(func(p *patient.Patient) { p.BirthDate = date(1900, 1, 1) })},
		{name: "born 1899-12-31", in: with(func(p *patient.Patient) { p.BirthDate = date(1899, 12, 31) }), want: patient.ErrInvalidBirthDate},
		{name: "born yesterday", in: with(func(p *patient.Patient) { p.BirthDate = date(2026, 11, 3) })},
		{name: "born today", in: with(func(p *patient.Patient) { p.BirthDate = date(2026, 11, 4) }), want: patient.ErrInvalidBirthDate},
		{name: "born tomorrow", in: with(func(p *patient.Patient) { p.BirthDate = date(2026, 11, 5) }), want: patient.ErrInvalidBirthDate},
		{name: "no birth date", in: with(func(p *patient.Patient) { p.BirthDate = time.Time{} }), want: patient.ErrInvalidBirthDate},
		{name: "phone of 8 digits", in: with(func(p *patient.Patient) { p.Phone = "+38760000" })},
		{name: "phone of 15 digits", in: with(func(p *patient.Patient) { p.Phone = "+387600000000001" })},
		{name: "phone of 7 digits", in: with(func(p *patient.Patient) { p.Phone = "+3876000" }), want: patient.ErrInvalidPhone},
		{name: "phone of 16 digits", in: with(func(p *patient.Patient) { p.Phone = "+3876000000000001" }), want: patient.ErrInvalidPhone},
		{name: "phone without +", in: with(func(p *patient.Patient) { p.Phone = "38760000001" }), want: patient.ErrInvalidPhone},
		{name: "phone with spaces", in: with(func(p *patient.Patient) { p.Phone = "+387 60 000 001" }), want: patient.ErrInvalidPhone},
		{name: "phone with letters", in: with(func(p *patient.Patient) { p.Phone = "+38760000abc" }), want: patient.ErrInvalidPhone},
		{name: "phone with non-ASCII digits", in: with(func(p *patient.Patient) { p.Phone = "+٣٨٧٦٠٠٠٠٠٠١" }), want: patient.ErrInvalidPhone},
	}
	for _, tt := range tests {
		repo, audit := newFakeRepository(), &fakeAuditor{}
		_, err := newService(repo, audit).SaveMe(t.Context(), other, tt.in)
		if !errors.Is(err, tt.want) {
			t.Errorf("%s: SaveMe() = %v, want %v", tt.name, err, tt.want)
		}
		if tt.want != nil && (len(repo.calls) != 0 || len(audit.events) != 0) {
			t.Errorf("%s: calls %v, events %d; want the refusal before any access", tt.name, repo.calls, len(audit.events))
		}
	}
}

func TestSaveMe_ValidatesAgainstTheDateInUTC(t *testing.T) {
	// 00:30 in Sarajevo on 5 November is still 4 November in UTC: a birth date of 4 November is today.
	clk := clock.Fixed{T: time.Date(2026, 11, 4, 23, 30, 0, 0, time.UTC)}
	svc := patient.NewService(newFakeRepository(), &fakeAuditor{}, fakeTransactor{}, clk, &fakeConsents{}, &fakeAppointments{})
	in := patient.Patient{GivenName: "Ada", FamilyName: "Example", BirthDate: date(2026, 11, 4)}
	if _, err := svc.SaveMe(t.Context(), other, in); !errors.Is(err, patient.ErrInvalidBirthDate) {
		t.Errorf("SaveMe(born today in UTC) = %v, want ErrInvalidBirthDate", err)
	}
}

func TestRead_GivesThePatientTheirOwnProfileAndAuditsIt(t *testing.T) {
	repo, audit := newFakeRepository(), &fakeAuditor{}
	got, err := newService(repo, audit).Read(t.Context(), me, patientA)
	if err != nil || got != repo.profiles[accountA] {
		t.Fatalf("Read() = %+v, %v; want profile A", got, err)
	}
	if want := []string{"SetAccount", "ByID"}; !slices.Equal(repo.calls, want) || repo.account != accountA {
		t.Errorf("calls %v with account %q, want %v with account A", repo.calls, repo.account, want)
	}
	checkEvents(t, audit, event(me, auditevent.CodeRead, patientA, auditevent.OutcomeSuccess))
}

func TestRead_HidesAnotherPatientAndEveryProfileFromStaffAndAuditsIt(t *testing.T) {
	tests := []struct {
		name   string
		caller patient.Caller
		calls  []string
	}{
		{name: "another patient", caller: me, calls: []string{"SetAccount", "ByID"}},
		{name: "staff", caller: staff, calls: []string{"SetClinic"}},
	}
	for _, tt := range tests {
		repo, audit := newFakeRepository(), &fakeAuditor{}
		_, err := newService(repo, audit).Read(t.Context(), tt.caller, patientB)
		if !errors.Is(err, patient.ErrNotFound) || !slices.Equal(repo.calls, tt.calls) {
			t.Errorf("%s: Read() = %v after calls %v, want ErrNotFound after %v", tt.name, err, repo.calls, tt.calls)
		}
		checkEvents(t, audit, event(tt.caller, auditevent.CodeRead, "", auditevent.OutcomeMinorFailure))
	}
}

func TestService_FailsOnARepositoryOrAuditError(t *testing.T) {
	valid := patient.Patient{GivenName: "Ada", FamilyName: "Example", BirthDate: date(1990, 1, 1), VersionID: 2}
	calls := map[string]func(s *patient.Service) error{
		"Me":          func(s *patient.Service) error { _, err := s.Me(t.Context(), me); return err },
		"Me as staff": func(s *patient.Service) error { _, err := s.Me(t.Context(), staff); return err },
		"SaveMe":      func(s *patient.Service) error { _, err := s.SaveMe(t.Context(), me, valid); return err },
		"SaveMe as staff": func(s *patient.Service) error {
			_, err := s.SaveMe(t.Context(), staff, valid)
			return err
		},
		"Read":          func(s *patient.Service) error { _, err := s.Read(t.Context(), me, patientA); return err },
		"Read as staff": func(s *patient.Service) error { _, err := s.Read(t.Context(), staff, patientA); return err },
	}
	boom := errors.New("connection reset")
	for name, call := range calls {
		for _, broken := range []string{"repository", "audit"} {
			repo, audit := newFakeRepository(), &fakeAuditor{}
			if broken == "repository" {
				repo.err = boom
			} else {
				audit.err = boom
			}
			err := call(newService(repo, audit))
			if broken == "repository" && strings.HasSuffix(name, "staff") {
				continue // staff never reach the repository
			}
			if !errors.Is(err, boom) {
				t.Errorf("%s with a broken %s = %v, want the error", name, broken, err)
			}
		}
	}
}

func TestRead_GivesStaffTheProfileOnlyAtFullAccessAndAuditsIt(t *testing.T) {
	revokedAt := now.Add(-time.Hour)
	past := access.Visit{PractitionerRole: roleA, Start: revokedAt.Add(-24 * time.Hour)}
	future := access.Visit{PractitionerRole: roleA, Start: now.Add(24 * time.Hour)}
	tests := []struct {
		name    string
		id      string
		visits  []access.Visit
		calls   []string
		caller  patient.Caller
		consent access.Consent
		want    bool // the profile is returned
	}{
		{name: "practitioner, active consent, an appointment with them", caller: practitioner, id: patientA,
			consent: access.Consent{Active: true}, visits: []access.Visit{future}, want: true, calls: []string{"SetClinic", "ByID"}},
		{name: "practitioner, active consent, no appointment (none until package 6)", caller: practitioner, id: patientA,
			consent: access.Consent{Active: true}, calls: []string{"SetClinic"}},
		{name: "practitioner, active consent, another practitioner's appointment", caller: practitioner, id: patientA,
			consent: access.Consent{Active: true}, visits: []access.Visit{{PractitionerRole: roleA2, Start: future.Start}}, calls: []string{"SetClinic"}},
		{name: "practitioner, revoked, a past appointment: the name only", caller: practitioner, id: patientA,
			consent: access.Consent{RevokedAt: revokedAt}, visits: []access.Visit{past}, calls: []string{"SetClinic"}},
		{name: "practitioner, never consented", caller: practitioner, id: patientA, visits: []access.Visit{future}, calls: []string{"SetClinic"}},
		{name: "admin, active consent: the name only", caller: admin, id: patientA,
			consent: access.Consent{Active: true}, visits: []access.Visit{future}, calls: []string{"SetClinic"}},
		{name: "full access but no row visible", caller: practitioner, id: "d9000000-0000-4000-8000-0000000000d9",
			consent: access.Consent{Active: true}, visits: []access.Visit{future}, calls: []string{"SetClinic", "ByID"}},
	}
	for _, tt := range tests {
		repo, audit := newFakeRepository(), &fakeAuditor{}
		consents, appts := &fakeConsents{state: tt.consent}, &fakeAppointments{visits: tt.visits}
		svc := patient.NewService(repo, audit, fakeTransactor{}, clock.Fixed{T: now}, consents, appts)
		got, err := svc.Read(t.Context(), tt.caller, tt.id)
		switch {
		case tt.want && (err != nil || got != repo.profiles[accountA]):
			t.Errorf("%s: Read() = %+v, %v; want profile A", tt.name, got, err)
		case !tt.want && !errors.Is(err, patient.ErrNotFound):
			t.Errorf("%s: Read() = %+v, %v; want ErrNotFound", tt.name, got, err)
		}
		if !slices.Equal(repo.calls, tt.calls) || repo.clinic != clinicA || repo.account != "" {
			t.Errorf("%s: calls %v in clinic %q, account %q; want %v in clinic A only", tt.name, repo.calls, repo.clinic, repo.account, tt.calls)
		}
		if want := []string{tt.id, clinicA}; !slices.Equal(consents.args, want) || !slices.Equal(appts.args, want) {
			t.Errorf("%s: facts asked for %v and %v, want %v", tt.name, consents.args, appts.args, want)
		}
		if tt.want {
			checkEvents(t, audit, event(tt.caller, auditevent.CodeRead, patientA, auditevent.OutcomeSuccess))
		} else {
			checkEvents(t, audit, event(tt.caller, auditevent.CodeRead, "", auditevent.OutcomeMinorFailure))
		}
	}
}

func TestRead_FailsForStaffOnAConsentOrAppointmentError(t *testing.T) {
	boom := errors.New("connection reset")
	for name, ports := range map[string]struct {
		consents *fakeConsents
		appts    *fakeAppointments
	}{
		"consents":     {consents: &fakeConsents{err: boom}, appts: &fakeAppointments{}},
		"appointments": {consents: &fakeConsents{state: access.Consent{Active: true}}, appts: &fakeAppointments{err: boom}},
	} {
		audit := &fakeAuditor{}
		svc := patient.NewService(newFakeRepository(), audit, fakeTransactor{}, clock.Fixed{T: now}, ports.consents, ports.appts)
		if _, err := svc.Read(t.Context(), practitioner, patientA); !errors.Is(err, boom) {
			t.Errorf("Read() with broken %s = %v, want the error", name, err)
		}
	}
}
