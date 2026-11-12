package consent_test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
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

// fakeJobs records the consent_revoked jobs enqueued, "<patient id> <clinic id>", and the
// transcription_consent_revoked ones, "transcription <patient id> <clinic id>".
type fakeJobs struct {
	err      error
	enqueued []string
}

func (j *fakeJobs) EnqueueConsentRevoked(ctx context.Context, patientID, organizationID string) error {
	return j.enqueue(ctx, patientID+" "+organizationID)
}

func (j *fakeJobs) EnqueueTranscriptionRevoked(ctx context.Context, patientID, organizationID string) error {
	return j.enqueue(ctx, "transcription "+patientID+" "+organizationID)
}

func (j *fakeJobs) enqueue(ctx context.Context, job string) error {
	if ctx.Value(txKey{}) == nil {
		return errors.New("enqueued outside a transaction")
	}
	if j.err != nil {
		return j.err
	}
	j.enqueued = append(j.enqueued, job)
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
	changes  []string // the history consents' changes, "ChangeHistory <id> v<version> +[<added>] -[<removed>]"
	// race makes GrantHistory and ChangeHistory lose to a concurrent change; refuseVisits makes GrantHistory meet
	// row-level security's refusal of a visit.
	race, refuseVisits bool
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{consents: []consent.Consent{
		{ID: consentAA, PatientID: patientA, OrganizationID: clinicA, Kind: consent.KindCare, Status: consent.StatusActive, DateTime: granted, LastUpdated: granted, VersionID: 1},
		{ID: consentAB, PatientID: patientA, OrganizationID: clinicB, Kind: consent.KindCare, Status: consent.StatusInactive, DateTime: granted, RevokedAt: revoked, LastUpdated: revoked, VersionID: 2},
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

func (r *fakeRepository) Grant(ctx context.Context, kind consent.Kind, patientID, organizationID string, at time.Time) (consent.Consent, error) {
	r.seen(ctx, "Grant")
	if r.err != nil {
		return consent.Consent{}, r.err
	}
	if organizationID != clinicA && organizationID != clinicB {
		return consent.Consent{}, consent.ErrClinicNotFound
	}
	for _, c := range r.consents {
		if c.PatientID == patientID && c.OrganizationID == organizationID && c.Kind == kind && c.Status == consent.StatusActive {
			return consent.Consent{}, consent.ErrActive
		}
	}
	c := consent.Consent{ID: newID, PatientID: patientID, OrganizationID: organizationID, Kind: kind, Status: consent.StatusActive, DateTime: at, LastUpdated: at, VersionID: 1}
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

func (r *fakeRepository) Revoke(ctx context.Context, kind consent.Kind, id string, at time.Time) (consent.Consent, error) {
	r.seen(ctx, "Revoke "+string(kind))
	return r.revoke(func(c consent.Consent) bool { return c.ID == id && c.Kind == kind }, at)
}

func (r *fakeRepository) RevokeTranscription(ctx context.Context, patientID, organizationID string, at time.Time) (consent.Consent, error) {
	r.seen(ctx, "RevokeTranscription")
	return r.revoke(func(c consent.Consent) bool {
		return c.PatientID == patientID && c.OrganizationID == organizationID && c.Kind == consent.KindTranscription
	}, at)
}

// revoke revokes the active consent of the patient's scope that match picks.
func (r *fakeRepository) revoke(match func(consent.Consent) bool, at time.Time) (consent.Consent, error) {
	if r.err != nil {
		return consent.Consent{}, r.err
	}
	for i, c := range r.consents {
		if match(c) && r.patient != "" && c.PatientID == r.patient && c.Status == consent.StatusActive {
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
		if c.PatientID == patientID && c.OrganizationID == organizationID && c.Kind == consent.KindTranscription {
			s.Transcription = s.Transcription || c.Status == consent.StatusActive
		} else if c.PatientID == patientID && c.OrganizationID == organizationID && c.Kind == consent.KindHistory {
			if c.Status == consent.StatusActive {
				s.History, s.Shared = true, shared(c)
			}
		} else if c.PatientID == patientID && c.OrganizationID == organizationID {
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
	repo   *fakeRepository
	audit  *fakeAuditor
	jobs   *fakeJobs
	visits *fakeVisits
	svc    *consent.Service
}

func newFixture() fixture {
	f := fixture{repo: newFakeRepository(), audit: &fakeAuditor{}, jobs: &fakeJobs{}, visits: &fakeVisits{}}
	f.svc = consent.NewService(f.repo, f.audit, fakeTransactor{audit: f.audit, jobs: f.jobs}, clock.Fixed{T: now}, f.jobs).WithVisits(f.visits)
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
	got, err := f.svc.Grant(t.Context(), other, clinicA, consent.KindCare)
	want := consent.Consent{ID: newID, PatientID: patientB, OrganizationID: clinicA, Kind: consent.KindCare, Status: consent.StatusActive, DateTime: now, LastUpdated: now, VersionID: 1}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("Grant() = %+v, %v;\nwant %+v", got, err, want)
	}
	if want := []string{"SetPatient", "Grant"}; !slices.Equal(f.repo.calls, want) || f.repo.patient != patientB {
		t.Errorf("calls %v with patient %q, want %v inside the transaction with patient B", f.repo.calls, f.repo.patient, want)
	}
	checkEvents(t, "Grant", f.audit, event(other, auditevent.CodeCreate, newID, auditevent.OutcomeSuccess))
}

func TestGrant_AfterARevocationAddsANewConsent(t *testing.T) {
	f := newFixture()
	got, err := f.svc.Grant(t.Context(), me, clinicB, consent.KindCare)
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
		_, err := f.svc.Grant(t.Context(), tt.caller, tt.clinic, consent.KindCare)
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
	if want := []string{"SetPatient", "ByID", "Revoke care", "RevokeTranscription", "RevokeHistory"}; !slices.Equal(f.repo.calls, want) || f.repo.patient != patientA {
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

func (r *racingRepository) Revoke(context.Context, consent.Kind, string, time.Time) (consent.Consent, error) {
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
		"Grant": func(s *consent.Service) error {
			_, err := s.Grant(t.Context(), me, clinicB, consent.KindCare)
			return err
		},
		"Revoke": func(s *consent.Service) error { _, err := s.Revoke(t.Context(), me, consentAA); return err },
		"Mine":   func(s *consent.Service) error { _, err := s.Mine(t.Context(), me); return err },
		"Read":   func(s *consent.Service) error { _, err := s.Read(t.Context(), me, consentAA); return err },
		"Read as staff": func(s *consent.Service) error {
			_, err := s.Read(t.Context(), staff, consentAA)
			return err
		},
		"State": func(s *consent.Service) error { _, err := s.State(t.Context(), patientA, clinicA); return err },
		"Grant as staff": func(s *consent.Service) error {
			_, err := s.Grant(t.Context(), staff, clinicA, consent.KindCare)
			return err
		},
		"Grant to an unknown clinic": func(s *consent.Service) error {
			_, err := s.Grant(t.Context(), other, "x", consent.KindCare)
			return err
		},
		"Candidates": func(s *consent.Service) error { _, err := s.Candidates(t.Context(), me, clinicA); return err },
		"SetSharing": func(s *consent.Service) error {
			_, err := s.SetSharing(t.Context(), me, clinicA, consent.SetSharing{EncounterIDs: []string{visit1}}, 0)
			return err
		},
		"StopSharing": func(s *consent.Service) error { return s.StopSharing(t.Context(), me, clinicA) },
		"MySharing":   func(s *consent.Service) error { _, err := s.MySharing(t.Context(), me); return err },
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

// withTranscription is the fixture with patient A's active transcription consent with clinic A.
func withTranscription() (fixture, consent.Consent) {
	f := newFixture()
	tc := consent.Consent{ID: transcriptionAA, PatientID: patientA, OrganizationID: clinicA, Kind: consent.KindTranscription, Status: consent.StatusActive, DateTime: granted, LastUpdated: granted, VersionID: 1}
	f.repo.consents = append(f.repo.consents, tc)
	return f, tc
}

const transcriptionAA = "f3000000-0000-4000-8000-0000000000f3"

func TestGrant_StoresATranscriptionConsentBesideTheCareConsent(t *testing.T) {
	f := newFixture()
	got, err := f.svc.Grant(t.Context(), me, clinicA, consent.KindTranscription)
	want := consent.Consent{ID: newID, PatientID: patientA, OrganizationID: clinicA, Kind: consent.KindTranscription, Status: consent.StatusActive, DateTime: now, LastUpdated: now, VersionID: 1}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("Grant(transcription) = %+v, %v;\nwant %+v", got, err, want)
	}
	checkEvents(t, "Grant", f.audit, event(me, auditevent.CodeCreate, newID, auditevent.OutcomeSuccess))
	if _, err := f.svc.Grant(t.Context(), me, clinicA, consent.KindTranscription); !errors.Is(err, consent.ErrActive) {
		t.Errorf("a second Grant(transcription) = %v, want ErrActive", err)
	}
	st, err := f.svc.State(t.Context(), patientA, clinicA)
	if err != nil || !st.Active || !st.Transcription {
		t.Errorf("State() = %+v, %v; want both consents active", st, err)
	}
}

func TestGrant_RefusesAnUnknownKindAndAuditsIt(t *testing.T) {
	f := newFixture()
	if _, err := f.svc.Grant(t.Context(), me, clinicA, "research"); !errors.Is(err, consent.ErrKind) {
		t.Errorf("Grant(research) = %v, want ErrKind", err)
	}
	if len(f.repo.calls) != 0 {
		t.Errorf("calls %v, want none", f.repo.calls)
	}
	checkEvents(t, "unknown kind", f.audit, event(me, auditevent.CodeCreate, "", auditevent.OutcomeMinorFailure))
}

// A withdrawal of the transcription consent takes the agent out of the call (transcription_consent_revoked) and
// leaves the care consent, the appointments and the call as they are: no consent_revoked.
func TestRevoke_ATranscriptionConsentEnqueuesOnlyTheAgentsRemoval(t *testing.T) {
	f, tc := withTranscription()
	got, err := f.svc.Revoke(t.Context(), me, tc.ID)
	if err != nil || got.Kind != consent.KindTranscription || got.Status != consent.StatusInactive || !got.RevokedAt.Equal(now) {
		t.Fatalf("Revoke() = %+v, %v; want the transcription consent revoked now", got, err)
	}
	if want := []string{"SetPatient", "ByID", "Revoke transcription"}; !slices.Equal(f.repo.calls, want) {
		t.Errorf("calls %v, want %v", f.repo.calls, want)
	}
	if want := []string{"transcription " + patientA + " " + clinicA}; !slices.Equal(f.jobs.enqueued, want) {
		t.Errorf("enqueued %v, want %v", f.jobs.enqueued, want)
	}
	checkEvents(t, "Revoke", f.audit, event(me, auditevent.CodeUpdate, tc.ID, auditevent.OutcomeSuccess))
	if f.repo.consents[0].Status != consent.StatusActive {
		t.Errorf("the care consent became %+v, want it active", f.repo.consents[0])
	}
}

// A revocation of the care consent revokes the transcription consent with the clinic in the same transaction,
// audited: granting care again asks for the transcription consent again.
func TestRevoke_TheCareConsentRevokesTheTranscriptionConsentToo(t *testing.T) {
	f, tc := withTranscription()
	if _, err := f.svc.Revoke(t.Context(), me, consentAA); err != nil {
		t.Fatal(err)
	}
	if got := f.repo.consents[2]; got.ID != tc.ID || got.Status != consent.StatusInactive || !got.RevokedAt.Equal(now) {
		t.Errorf("the transcription consent became %+v, want it revoked now", got)
	}
	checkEvents(t, "Revoke", f.audit,
		event(me, auditevent.CodeUpdate, tc.ID, auditevent.OutcomeSuccess),
		event(me, auditevent.CodeUpdate, consentAA, auditevent.OutcomeSuccess))
	if want := []string{patientA + " " + clinicA}; !slices.Equal(f.jobs.enqueued, want) {
		t.Errorf("enqueued %v, want only consent_revoked %v (it removes the agent with the patient)", f.jobs.enqueued, want)
	}
	if st, _ := f.svc.State(t.Context(), patientA, clinicA); st.Active || st.Transcription {
		t.Errorf("State() = %+v, want neither active", st)
	}
}

// The chat consents (SAL-28) of the fake: patient A's conversation with clinic A, and clinic B's role.
const (
	chatConversation = "e1000000-0000-4000-8000-0000000000e1"
	chatRole         = "b2000000-0000-4000-8000-0000000000b2"
)

var chatGrant = consent.ChatGrant{ConversationID: chatConversation, OrganizationID: clinicA, PractitionerRoleID: chatRole}

func (r *fakeRepository) GrantChat(ctx context.Context, patientID string, g consent.ChatGrant, at time.Time) (consent.Consent, error) {
	r.seen(ctx, "GrantChat")
	if r.err != nil {
		return consent.Consent{}, r.err
	}
	for _, c := range r.consents {
		if c.Kind == consent.KindChat && c.ConversationID == g.ConversationID && c.PractitionerRoleID == g.PractitionerRoleID && c.Status == consent.StatusActive {
			return consent.Consent{}, errors.New("unique violation")
		}
	}
	c := consent.Consent{ID: fmt.Sprintf("f%07d-0000-4000-8000-000000000000", len(r.consents)), PatientID: patientID, OrganizationID: g.OrganizationID,
		ConversationID: g.ConversationID, PractitionerRoleID: g.PractitionerRoleID, Kind: consent.KindChat, Status: consent.StatusActive,
		DateTime: at, LastUpdated: at, VersionID: 1}
	r.consents = append(r.consents, c)
	return c, nil
}

func (r *fakeRepository) RevokeChat(ctx context.Context, conversationID, roleID string, at time.Time) (consent.Consent, error) {
	r.seen(ctx, "RevokeChat")
	return r.revoke(func(c consent.Consent) bool {
		return c.Kind == consent.KindChat && c.ConversationID == conversationID && c.PractitionerRoleID == roleID
	}, at)
}

func (r *fakeRepository) ChatGranted(ctx context.Context, conversationID, roleID string) (bool, error) {
	r.seen(ctx, "ChatGranted")
	for _, c := range r.consents {
		if c.Kind == consent.KindChat && c.ConversationID == conversationID && c.PractitionerRoleID == roleID && c.Status == consent.StatusActive {
			return true, r.err
		}
	}
	return false, r.err
}

func TestGrantChat_StoresThePatientsConsentForTheConversationAndRoleAndAuditsIt(t *testing.T) {
	f := newFixture()
	got, err := f.svc.GrantChat(t.Context(), me, chatGrant)
	if err != nil || got.Kind != consent.KindChat || got.PatientID != patientA || got.ConversationID != chatConversation ||
		got.PractitionerRoleID != chatRole || got.OrganizationID != clinicA || got.Status != consent.StatusActive || !got.DateTime.Equal(now) {
		t.Fatalf("GrantChat() = %+v, %v", got, err)
	}
	if want := []string{"SetPatient", "RevokeChat", "GrantChat"}; !slices.Equal(f.repo.calls, want) || f.repo.patient != patientA {
		t.Errorf("calls %v with patient %q, want %v with patient A", f.repo.calls, f.repo.patient, want)
	}
	checkEvents(t, "GrantChat", f.audit, event(me, auditevent.CodeCreate, got.ID, auditevent.OutcomeSuccess))
	if ok, err := f.svc.ChatGranted(t.Context(), chatConversation, chatRole); err != nil || !ok {
		t.Errorf("ChatGranted() = %v, %v; want true", ok, err)
	}
	if len(f.audit.events) != 1 {
		t.Errorf("ChatGranted was audited: %+v", f.audit.events)
	}
}

// An invitation granted again, after the doctor was removed and invited anew, ends the earlier consent first: at
// most one is active, and both are audited.
func TestGrantChat_EndsAnEarlierActiveConsentForTheConversationAndRole(t *testing.T) {
	f := newFixture()
	first, err := f.svc.GrantChat(t.Context(), me, chatGrant)
	if err != nil {
		t.Fatal(err)
	}
	second, err := f.svc.GrantChat(t.Context(), me, chatGrant)
	if err != nil || second.ID == first.ID {
		t.Fatalf("GrantChat() again = %+v, %v", second, err)
	}
	var active int
	for _, c := range f.repo.consents {
		if c.Kind == consent.KindChat && c.Status == consent.StatusActive {
			active++
		}
	}
	if active != 1 {
		t.Errorf("%d active chat consents, want 1", active)
	}
	checkEvents(t, "GrantChat twice", f.audit, event(me, auditevent.CodeCreate, first.ID, auditevent.OutcomeSuccess),
		event(me, auditevent.CodeUpdate, first.ID, auditevent.OutcomeSuccess), event(me, auditevent.CodeCreate, second.ID, auditevent.OutcomeSuccess))
}

func TestGrantChatAndRevokeChat_RefuseStaffAndAPatientWithoutAProfile(t *testing.T) {
	for _, tt := range []struct {
		want   error
		caller consent.Caller
		name   string
	}{
		{name: "staff", caller: staff, want: consent.ErrForbidden},
		{name: "a patient without a profile", caller: noProf, want: consent.ErrProfileRequired},
	} {
		f := newFixture()
		if _, err := f.svc.GrantChat(t.Context(), tt.caller, chatGrant); !errors.Is(err, tt.want) {
			t.Errorf("%s: GrantChat() = %v, want %v", tt.name, err, tt.want)
		}
		if _, err := f.svc.RevokeChat(t.Context(), tt.caller, chatConversation, chatRole); !errors.Is(err, consent.ErrForbidden) && !errors.Is(err, consent.ErrInactive) {
			t.Errorf("%s: RevokeChat() = %v", tt.name, err)
		}
		for _, e := range f.audit.events {
			if e.Outcome != auditevent.OutcomeMinorFailure {
				t.Errorf("%s: audited %+v, want refusals only", tt.name, e)
			}
		}
		if len(f.audit.events) != 2 {
			t.Errorf("%s: %d events, want both refusals audited", tt.name, len(f.audit.events))
		}
	}
}

func TestRevokeChat_EndsTheActiveConsentAndAuditsIt(t *testing.T) {
	f := newFixture()
	granted, err := f.svc.GrantChat(t.Context(), me, chatGrant)
	if err != nil {
		t.Fatal(err)
	}
	got, err := f.svc.RevokeChat(t.Context(), me, chatConversation, chatRole)
	if err != nil || got.ID != granted.ID || got.Status != consent.StatusInactive || !got.RevokedAt.Equal(now) {
		t.Fatalf("RevokeChat() = %+v, %v", got, err)
	}
	if e := f.audit.events[len(f.audit.events)-1]; e != event(me, auditevent.CodeUpdate, got.ID, auditevent.OutcomeSuccess) {
		t.Errorf("audit = %+v", e)
	}
	if _, err := f.svc.RevokeChat(t.Context(), me, chatConversation, chatRole); !errors.Is(err, consent.ErrInactive) {
		t.Errorf("a second RevokeChat() = %v, want ErrInactive", err)
	}
	if e := f.audit.events[len(f.audit.events)-1]; e.Outcome != auditevent.OutcomeMinorFailure {
		t.Errorf("the refused revocation's audit = %+v", e)
	}
	if ok, _ := f.svc.ChatGranted(t.Context(), chatConversation, chatRole); ok {
		t.Error("ChatGranted() after the revocation: true")
	}
}

// The treating practitioner's removal of the invited doctor ends the patient's chat consent for them as the
// conversation's clinic, audited (SAL-28 review, finding 1): the patient sees it ended, never still granted.
func TestEndChat_TheClinicEndsTheActiveConsentAndAuditsIt(t *testing.T) {
	f := newFixture()
	granted, err := f.svc.GrantChat(t.Context(), me, chatGrant)
	if err != nil {
		t.Fatal(err)
	}
	f.repo.calls = nil
	got, err := f.svc.EndChat(t.Context(), staff, chatConversation, chatRole)
	if err != nil || got.ID != granted.ID || got.Status != consent.StatusInactive || !got.RevokedAt.Equal(now) {
		t.Fatalf("EndChat() = %+v, %v", got, err)
	}
	if want := []string{"SetClinic", "RevokeChat"}; !slices.Equal(f.repo.calls, want) || f.repo.clinic != clinicA {
		t.Errorf("calls %v in clinic %q, want %v in clinic A", f.repo.calls, f.repo.clinic, want)
	}
	if e := f.audit.events[len(f.audit.events)-1]; e != event(staff, auditevent.CodeUpdate, got.ID, auditevent.OutcomeSuccess) {
		t.Errorf("audit = %+v", e)
	}
	if _, err := f.svc.EndChat(t.Context(), staff, chatConversation, chatRole); !errors.Is(err, consent.ErrInactive) {
		t.Errorf("a second EndChat() = %v, want ErrInactive", err)
	}
	if ok, _ := f.svc.ChatGranted(t.Context(), chatConversation, chatRole); ok {
		t.Error("ChatGranted() after EndChat: true")
	}
}

func TestEndChat_RefusesAPatient(t *testing.T) {
	f := newFixture()
	if _, err := f.svc.GrantChat(t.Context(), me, chatGrant); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.EndChat(t.Context(), me, chatConversation, chatRole); !errors.Is(err, consent.ErrForbidden) {
		t.Errorf("EndChat() as the patient = %v, want ErrForbidden", err)
	}
	if e := f.audit.events[len(f.audit.events)-1]; e.Outcome != auditevent.OutcomeMinorFailure {
		t.Errorf("the refusal's audit = %+v", e)
	}
	if ok, _ := f.svc.ChatGranted(t.Context(), chatConversation, chatRole); !ok {
		t.Error("the patient's EndChat ended the consent")
	}
}
