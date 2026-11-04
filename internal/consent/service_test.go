package consent_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/consent"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

const staffA = "a1000000-0000-4000-8000-0000000000a1"

var now = time.Date(2026, 11, 4, 9, 30, 0, 0, time.UTC)

var (
	me      = consent.Caller{AccountID: accountA, PatientID: patientA}
	other   = consent.Caller{AccountID: accountB, PatientID: patientB}
	noProf  = consent.Caller{AccountID: "c3000000-0000-4000-8000-0000000000c3"} // a patient account without a profile
	staff   = consent.Caller{AccountID: staffA, OrganizationID: clinicA}
	staffB  = consent.Caller{AccountID: staffA, OrganizationID: clinicB}
	newID   = "f9000000-0000-4000-8000-0000000000f9"
	revoked = granted.Add(24 * time.Hour)
)

type txKey struct{}

// fakeTransactor runs fn in a "transaction" and, when fn fails, drops the audit events recorded in it, as a
// rollback does.
type fakeTransactor struct {
	audit *fakeAuditor
	jobs  *fakeJobs
}

func (f fakeTransactor) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	before, jobs := len(f.audit.events), len(f.jobs.enqueued)
	err := fn(context.WithValue(ctx, txKey{}, true))
	if err != nil {
		f.audit.events, f.jobs.enqueued = f.audit.events[:before], f.jobs.enqueued[:jobs]
	}
	return err
}

// fakeJobs records the consent_revoked jobs enqueued: "<patient id> <clinic id>".
type fakeJobs struct {
	err      error
	enqueued []string
}

func (j *fakeJobs) EnqueueConsentRevoked(ctx context.Context, patientID, organizationID string) error {
	if ctx.Value(txKey{}) == nil {
		return errors.New("enqueued outside a transaction")
	}
	if j.err != nil {
		return j.err
	}
	j.enqueued = append(j.enqueued, patientID+" "+organizationID)
	return nil
}

// fakeRepository holds consents and shows only those of the patient or clinic set on the transaction, as
// row-level security does.
type fakeRepository struct {
	err      error // returned by every read and write when set
	consents []consent.Consent
	patient  string   // set by SetPatient
	clinic   string   // set by SetClinic
	calls    []string // the calls, "outside:" prefixed when made outside a transaction
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{consents: []consent.Consent{
		{ID: consentAA, PatientID: patientA, OrganizationID: clinicA, Status: consent.StatusActive, DateTime: granted, LastUpdated: granted, VersionID: 1},
		{ID: consentAB, PatientID: patientA, OrganizationID: clinicB, Status: consent.StatusInactive, DateTime: granted, RevokedAt: revoked, LastUpdated: revoked, VersionID: 2},
	}}
}

func (r *fakeRepository) seen(ctx context.Context, call string) {
	if ctx.Value(txKey{}) == nil {
		call = "outside:" + call
	}
	r.calls = append(r.calls, call)
}

func (r *fakeRepository) visible(c consent.Consent) bool {
	return (r.patient != "" && c.PatientID == r.patient) || (r.patient == "" && r.clinic != "" && c.OrganizationID == r.clinic)
}

func (r *fakeRepository) SetPatient(ctx context.Context, id string) error {
	r.seen(ctx, "SetPatient")
	r.patient = id
	return nil
}

func (r *fakeRepository) SetClinic(ctx context.Context, id string) error {
	r.seen(ctx, "SetClinic")
	r.clinic = id
	return nil
}

func (r *fakeRepository) Grant(ctx context.Context, patientID, organizationID string, at time.Time) (consent.Consent, error) {
	r.seen(ctx, "Grant")
	if r.err != nil {
		return consent.Consent{}, r.err
	}
	if organizationID != clinicA && organizationID != clinicB {
		return consent.Consent{}, consent.ErrClinicNotFound
	}
	for _, c := range r.consents {
		if c.PatientID == patientID && c.OrganizationID == organizationID && c.Status == consent.StatusActive {
			return consent.Consent{}, consent.ErrActive
		}
	}
	c := consent.Consent{ID: newID, PatientID: patientID, OrganizationID: organizationID, Status: consent.StatusActive, DateTime: at, LastUpdated: at, VersionID: 1}
	r.consents = append(r.consents, c)
	return c, nil
}

func (r *fakeRepository) ByID(ctx context.Context, id string) (consent.Consent, error) {
	r.seen(ctx, "ByID")
	if r.err != nil {
		return consent.Consent{}, r.err
	}
	for _, c := range r.consents {
		if c.ID == id && r.visible(c) {
			return c, nil
		}
	}
	return consent.Consent{}, consent.ErrNotFound
}

func (r *fakeRepository) Revoke(ctx context.Context, id string, at time.Time) (consent.Consent, error) {
	r.seen(ctx, "Revoke")
	if r.err != nil {
		return consent.Consent{}, r.err
	}
	for i, c := range r.consents {
		if c.ID == id && r.patient != "" && c.PatientID == r.patient && c.Status == consent.StatusActive {
			c.Status, c.RevokedAt, c.LastUpdated, c.VersionID = consent.StatusInactive, at, at, c.VersionID+1
			r.consents[i] = c
			return c, nil
		}
	}
	return consent.Consent{}, consent.ErrInactive
}

func (r *fakeRepository) ByPatient(ctx context.Context, patientID string) ([]consent.Consent, error) {
	r.seen(ctx, "ByPatient")
	if r.err != nil {
		return nil, r.err
	}
	var out []consent.Consent
	for _, c := range r.consents {
		if c.PatientID == patientID && r.visible(c) {
			out = append(out, c)
		}
	}
	return out, nil
}

func (r *fakeRepository) State(ctx context.Context, patientID, organizationID string) (access.Consent, error) {
	r.seen(ctx, "State")
	if r.err != nil {
		return access.Consent{}, r.err
	}
	var s access.Consent
	for _, c := range r.consents {
		if c.PatientID == patientID && c.OrganizationID == organizationID {
			s.Active = s.Active || c.Status == consent.StatusActive
			if c.RevokedAt.After(s.RevokedAt) {
				s.RevokedAt = c.RevokedAt
			}
		}
	}
	return s, nil
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

type fixture struct {
	repo  *fakeRepository
	audit *fakeAuditor
	jobs  *fakeJobs
	svc   *consent.Service
}

func newFixture() fixture {
	f := fixture{repo: newFakeRepository(), audit: &fakeAuditor{}, jobs: &fakeJobs{}}
	f.svc = consent.NewService(f.repo, f.audit, fakeTransactor{audit: f.audit, jobs: f.jobs}, clock.Fixed{T: now}, f.jobs)
	return f
}

// event is the audit event of interaction by c on Consent/id ("" for none).
func event(c consent.Caller, interaction, id string, outcome auditevent.Outcome) auditevent.AuditEvent {
	action := map[string]auditevent.Action{
		auditevent.CodeRead: auditevent.ActionRead, auditevent.CodeCreate: auditevent.ActionCreate,
		auditevent.CodeUpdate: auditevent.ActionUpdate, auditevent.CodeSearchType: auditevent.ActionExecute,
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
		e.EntityReference = "Consent/" + id
	}
	return e
}

func checkEvents(t *testing.T, name string, audit *fakeAuditor, want ...auditevent.AuditEvent) {
	t.Helper()
	if !slices.Equal(audit.events, want) {
		t.Errorf("%s: audit events %+v,\nwant %+v", name, audit.events, want)
	}
}

func TestGrant_StoresAnActiveConsentOfTheCallingPatientAndAuditsIt(t *testing.T) {
	f := newFixture()
	got, err := f.svc.Grant(t.Context(), other, clinicA)
	want := consent.Consent{ID: newID, PatientID: patientB, OrganizationID: clinicA, Status: consent.StatusActive, DateTime: now, LastUpdated: now, VersionID: 1}
	if err != nil || got != want {
		t.Fatalf("Grant() = %+v, %v;\nwant %+v", got, err, want)
	}
	if want := []string{"SetPatient", "Grant"}; !slices.Equal(f.repo.calls, want) || f.repo.patient != patientB {
		t.Errorf("calls %v with patient %q, want %v inside the transaction with patient B", f.repo.calls, f.repo.patient, want)
	}
	checkEvents(t, "Grant", f.audit, event(other, auditevent.CodeCreate, newID, auditevent.OutcomeSuccess))
}

func TestGrant_AfterARevocationAddsANewConsent(t *testing.T) {
	f := newFixture()
	got, err := f.svc.Grant(t.Context(), me, clinicB)
	if err != nil || got.ID != newID || got.Status != consent.StatusActive || len(f.repo.consents) != 3 {
		t.Fatalf("Grant() = %+v, %v with %d consents; want a third, active consent", got, err, len(f.repo.consents))
	}
	if old := f.repo.consents[1]; old.ID != consentAB || old.Status != consent.StatusInactive {
		t.Errorf("the revoked consent became %+v, want it kept", old)
	}
}

func TestGrant_RefusesAndAuditsEveryCaseOfRefusal(t *testing.T) {
	tests := []struct {
		want   error
		caller consent.Caller
		name   string
		clinic string
		calls  []string
	}{
		{name: "a second active consent", caller: me, clinic: clinicA, want: consent.ErrActive, calls: []string{"SetPatient", "Grant"}},
		{name: "a patient without a profile", caller: noProf, clinic: clinicA, want: consent.ErrProfileRequired},
		{name: "staff", caller: staff, clinic: clinicA, want: consent.ErrForbidden},
		{name: "an unknown clinic, after the rollback", caller: other, clinic: "f0000000-0000-4000-8000-00000000000f", want: consent.ErrClinicNotFound, calls: []string{"SetPatient", "Grant"}},
	}
	for _, tt := range tests {
		f := newFixture()
		_, err := f.svc.Grant(t.Context(), tt.caller, tt.clinic)
		if !errors.Is(err, tt.want) || !slices.Equal(f.repo.calls, tt.calls) {
			t.Errorf("%s: Grant() = %v after calls %v, want %v after %v", tt.name, err, f.repo.calls, tt.want, tt.calls)
		}
		checkEvents(t, tt.name, f.audit, event(tt.caller, auditevent.CodeCreate, "", auditevent.OutcomeMinorFailure))
	}
}

func TestRevoke_EndsTheCallingPatientsActiveConsentAndAuditsIt(t *testing.T) {
	f := newFixture()
	got, err := f.svc.Revoke(t.Context(), me, consentAA)
	if err != nil || got.ID != consentAA || got.Status != consent.StatusInactive || !got.RevokedAt.Equal(now) || got.VersionID != 2 {
		t.Fatalf("Revoke() = %+v, %v; want consent A-A inactive, revoked now, version 2", got, err)
	}
	if want := []string{"SetPatient", "ByID", "Revoke"}; !slices.Equal(f.repo.calls, want) || f.repo.patient != patientA {
		t.Errorf("calls %v with patient %q, want %v inside the transaction with patient A", f.repo.calls, f.repo.patient, want)
	}
	checkEvents(t, "Revoke", f.audit, event(me, auditevent.CodeUpdate, consentAA, auditevent.OutcomeSuccess))
	if want := []string{patientA + " " + clinicA}; !slices.Equal(f.jobs.enqueued, want) {
		t.Errorf("enqueued %v, want consent_revoked for patient A and clinic A %v in the transaction", f.jobs.enqueued, want)
	}
}

// The job is enqueued in the revocation's transaction: when the revocation fails, no job is left.
func TestRevoke_EnqueuesNothingWhenTheRevocationFails(t *testing.T) {
	f := newFixture()
	f.audit.err = errors.New("connection reset")
	if _, err := f.svc.Revoke(t.Context(), me, consentAA); err == nil {
		t.Fatal("Revoke() error = nil with a failing audit")
	}
	if len(f.jobs.enqueued) != 0 {
		t.Errorf("enqueued %v after the rollback, want nothing", f.jobs.enqueued)
	}
	g := newFixture()
	g.jobs.err = errors.New("connection reset")
	if _, err := g.svc.Revoke(t.Context(), me, consentAA); err == nil || len(g.audit.events) != 0 {
		t.Errorf("Revoke() with a failing queue = %v with events %v, want the error and nothing audited", err, g.audit.events)
	}
}

func TestRevoke_RefusesAndAuditsEveryCaseOfRefusal(t *testing.T) {
	tests := []struct {
		want   error
		caller consent.Caller
		name   string
		id     string
	}{
		{name: "another patient's consent", caller: other, id: consentAA, want: consent.ErrNotFound},
		{name: "an unknown consent", caller: me, id: newID, want: consent.ErrNotFound},
		{name: "a patient without a profile", caller: noProf, id: consentAA, want: consent.ErrNotFound},
		{name: "an inactive consent", caller: me, id: consentAB, want: consent.ErrInactive},
		{name: "staff", caller: staff, id: consentAA, want: consent.ErrForbidden},
	}
	for _, tt := range tests {
		f := newFixture()
		_, err := f.svc.Revoke(t.Context(), tt.caller, tt.id)
		if !errors.Is(err, tt.want) {
			t.Errorf("%s: Revoke() = %v, want %v", tt.name, err, tt.want)
		}
		if f.repo.consents[0].Status != consent.StatusActive {
			t.Errorf("%s: consent A-A became %+v", tt.name, f.repo.consents[0])
		}
		checkEvents(t, tt.name, f.audit, event(tt.caller, auditevent.CodeUpdate, "", auditevent.OutcomeMinorFailure))
		if len(f.jobs.enqueued) != 0 {
			t.Errorf("%s: enqueued %v, want nothing", tt.name, f.jobs.enqueued)
		}
	}
}

func TestRevoke_LosingARaceIsInactive(t *testing.T) {
	f := newFixture()
	// Another revocation commits between the read and the update: ByID still saw it active.
	repo := &racingRepository{fakeRepository: f.repo}
	svc := consent.NewService(repo, f.audit, fakeTransactor{audit: f.audit, jobs: f.jobs}, clock.Fixed{T: now}, f.jobs)
	if _, err := svc.Revoke(t.Context(), me, consentAA); !errors.Is(err, consent.ErrInactive) {
		t.Errorf("Revoke() = %v, want ErrInactive", err)
	}
	checkEvents(t, "race", f.audit, event(me, auditevent.CodeUpdate, "", auditevent.OutcomeMinorFailure))
}

type racingRepository struct{ *fakeRepository }

func (r *racingRepository) Revoke(context.Context, string, time.Time) (consent.Consent, error) {
	return consent.Consent{}, consent.ErrInactive
}

func TestMine_ListsTheCallingPatientsConsentsAndAuditsEach(t *testing.T) {
	f := newFixture()
	got, err := f.svc.Mine(t.Context(), me)
	if err != nil || len(got) != 2 || got[0].ID != consentAA || got[1].ID != consentAB {
		t.Fatalf("Mine() = %+v, %v; want consents A-A and A-B", got, err)
	}
	if want := []string{"SetPatient", "ByPatient"}; !slices.Equal(f.repo.calls, want) || f.repo.patient != patientA {
		t.Errorf("calls %v with patient %q, want %v with patient A", f.repo.calls, f.repo.patient, want)
	}
	checkEvents(t, "Mine", f.audit,
		event(me, auditevent.CodeSearchType, consentAA, auditevent.OutcomeSuccess),
		event(me, auditevent.CodeSearchType, consentAB, auditevent.OutcomeSuccess))
}

func TestMine_IsEmptyWithoutConsentsOrAProfileAndAuditsOneEvent(t *testing.T) {
	for _, c := range []consent.Caller{other, noProf} {
		f := newFixture()
		got, err := f.svc.Mine(t.Context(), c)
		if err != nil || got == nil || len(got) != 0 {
			t.Errorf("Mine(%s) = %#v, %v; want an empty list", c.AccountID, got, err)
		}
		checkEvents(t, "Mine", f.audit, event(c, auditevent.CodeSearchType, "", auditevent.OutcomeSuccess))
	}
}

func TestMine_RefusesAndAuditsStaff(t *testing.T) {
	f := newFixture()
	if _, err := f.svc.Mine(t.Context(), staff); !errors.Is(err, consent.ErrForbidden) || len(f.repo.calls) != 0 {
		t.Errorf("Mine() as staff = %v after calls %v, want ErrForbidden without touching the repository", err, f.repo.calls)
	}
	checkEvents(t, "Mine", f.audit, event(staff, auditevent.CodeSearchType, "", auditevent.OutcomeMinorFailure))
}

func TestRead_GivesAConsentToItsPatientAndToStaffOfItsClinic(t *testing.T) {
	tests := []struct {
		caller consent.Caller
		name   string
		calls  []string
	}{
		{name: "its patient", caller: me, calls: []string{"SetPatient", "ByID"}},
		{name: "staff of its clinic", caller: staff, calls: []string{"SetClinic", "ByID"}},
	}
	for _, tt := range tests {
		f := newFixture()
		got, err := f.svc.Read(t.Context(), tt.caller, consentAA)
		if err != nil || got.ID != consentAA {
			t.Errorf("%s: Read() = %+v, %v; want consent A-A", tt.name, got, err)
		}
		if !slices.Equal(f.repo.calls, tt.calls) {
			t.Errorf("%s: calls %v, want %v", tt.name, f.repo.calls, tt.calls)
		}
		checkEvents(t, tt.name, f.audit, event(tt.caller, auditevent.CodeRead, consentAA, auditevent.OutcomeSuccess))
	}
}

func TestRead_HidesAConsentFromEveryoneElseAndAuditsIt(t *testing.T) {
	tests := []struct {
		caller consent.Caller
		name   string
		id     string
	}{
		{name: "another patient", caller: other, id: consentAA},
		{name: "a patient without a profile", caller: noProf, id: consentAA},
		{name: "staff of another clinic", caller: staffB, id: consentAA},
		{name: "an unknown consent", caller: staff, id: newID},
	}
	for _, tt := range tests {
		f := newFixture()
		if _, err := f.svc.Read(t.Context(), tt.caller, tt.id); !errors.Is(err, consent.ErrNotFound) {
			t.Errorf("%s: Read() = %v, want ErrNotFound", tt.name, err)
		}
		checkEvents(t, tt.name, f.audit, event(tt.caller, auditevent.CodeRead, "", auditevent.OutcomeMinorFailure))
	}
}

func TestRead_DoesNotRestOnRowLevelSecurityAlone(t *testing.T) {
	f := newFixture()
	leaky := &leakyRepository{fakeRepository: f.repo}
	svc := consent.NewService(leaky, f.audit, fakeTransactor{audit: f.audit, jobs: f.jobs}, clock.Fixed{T: now}, f.jobs)
	for _, c := range []consent.Caller{other, staffB} {
		if _, err := svc.Read(t.Context(), c, consentAA); !errors.Is(err, consent.ErrNotFound) {
			t.Errorf("Read(%+v) through a repository that shows every row = %v, want ErrNotFound", c, err)
		}
	}
}

// leakyRepository shows every consent, whatever the scope.
type leakyRepository struct{ *fakeRepository }

func (r *leakyRepository) ByID(_ context.Context, id string) (consent.Consent, error) {
	for _, c := range r.consents {
		if c.ID == id {
			return c, nil
		}
	}
	return consent.Consent{}, consent.ErrNotFound
}

func TestState_GivesTheConsentStateInTheCallersTransactionWithoutAnAudit(t *testing.T) {
	f := newFixture()
	got, err := f.svc.State(context.WithValue(t.Context(), txKey{}, true), patientA, clinicB)
	if err != nil || got.Active || !got.RevokedAt.Equal(revoked) {
		t.Errorf("State() = %+v, %v; want inactive, revoked a day after the grant", got, err)
	}
	if !slices.Equal(f.repo.calls, []string{"State"}) || len(f.audit.events) != 0 {
		t.Errorf("calls %v and %d events, want only State and no event (the caller audits its access)", f.repo.calls, len(f.audit.events))
	}
}

func TestService_FailsOnARepositoryOrAuditError(t *testing.T) {
	calls := map[string]func(s *consent.Service) error{
		"Grant":  func(s *consent.Service) error { _, err := s.Grant(t.Context(), me, clinicB); return err },
		"Revoke": func(s *consent.Service) error { _, err := s.Revoke(t.Context(), me, consentAA); return err },
		"Mine":   func(s *consent.Service) error { _, err := s.Mine(t.Context(), me); return err },
		"Read":   func(s *consent.Service) error { _, err := s.Read(t.Context(), me, consentAA); return err },
		"Read as staff": func(s *consent.Service) error {
			_, err := s.Read(t.Context(), staff, consentAA)
			return err
		},
		"State": func(s *consent.Service) error { _, err := s.State(t.Context(), patientA, clinicA); return err },
		"Grant as staff": func(s *consent.Service) error {
			_, err := s.Grant(t.Context(), staff, clinicA)
			return err
		},
		"Grant to an unknown clinic": func(s *consent.Service) error { _, err := s.Grant(t.Context(), other, "x"); return err },
	}
	boom := errors.New("connection reset")
	for name, call := range calls {
		for _, broken := range []string{"repository", "audit"} {
			f := newFixture()
			if broken == "repository" {
				f.repo.err = boom
			} else {
				f.audit.err = boom
			}
			err := call(f.svc)
			reaches := map[string]bool{"repository": name != "Grant as staff", "audit": name != "State"}[broken]
			if reaches && !errors.Is(err, boom) {
				t.Errorf("%s with a broken %s = %v, want the error", name, broken, err)
			}
			if err != nil && strings.Contains(fmt.Sprint(err), patientA) {
				t.Errorf("%s: the error %q holds an id", name, err)
			}
		}
	}
}
