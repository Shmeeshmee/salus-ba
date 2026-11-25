package patient_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/patient"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

const requestA = "f1000000-0000-4000-8000-0000000000f1"

func (r *fakeRepository) Restrict(ctx context.Context, patientID, org, by, requestID string, at time.Time) error {
	r.seen(ctx, "Restrict")
	if org != r.clinic || by == "" || !at.Equal(now) {
		return errors.New("restricted outside the clinic's scope, by nobody or at another time")
	}
	if r.restricted == nil {
		r.restricted = map[string]string{}
	}
	if _, ok := r.restricted[patientID+"/"+org]; !ok {
		r.restricted[patientID+"/"+org] = by
	}
	_ = requestID
	return r.err
}

func (r *fakeRepository) Unrestrict(ctx context.Context, patientID, org string) error {
	r.seen(ctx, "Unrestrict")
	if org != r.clinic {
		return errors.New("lifted outside the clinic's scope")
	}
	if r.restricted[patientID+"/"+org] != "" { // the erasure's restriction stays (row-level security)
		delete(r.restricted, patientID+"/"+org)
	}
	return r.err
}

func (r *fakeRepository) Restricted(ctx context.Context, patientID, org string) (bool, error) {
	r.seen(ctx, "Restricted")
	_, ok := r.restricted[patientID+"/"+org]
	return ok, r.err
}

// restrictionService is a Service whose patient A has the consent state and visits given at clinic A.
func restrictionService(repo *fakeRepository, audit *fakeAuditor, state access.Consent, visits []access.Visit) *patient.Service {
	return patient.NewService(repo, audit, fakeTransactor{}, clock.Fixed{T: now}, &fakeConsents{state: state}, &fakeAppointments{visits: visits})
}

// The clinic's admin restricts the record of one of the clinic's patients (docs/clinic-cms/WAVE5.md S7), optionally
// for a request, and lifts it; both idempotent, both audited as updates of the Patient at the patient's request.
func TestRestriction_TheAdminRestrictsAndLifts(t *testing.T) {
	repo, audit := newFakeRepository(), &fakeAuditor{}
	s := restrictionService(repo, audit, access.Consent{Active: true}, nil)
	for range 2 {
		if err := s.Restrict(t.Context(), admin, patientA, requestA); err != nil {
			t.Fatalf("Restrict: %v", err)
		}
	}
	if repo.restricted[patientA+"/"+clinicA] != staffA {
		t.Fatalf("restrictions %v, want patient A's at clinic A by the admin", repo.restricted)
	}
	for range 2 {
		if err := s.LiftRestriction(t.Context(), admin, patientA); err != nil {
			t.Fatalf("LiftRestriction: %v", err)
		}
	}
	if len(repo.restricted) != 0 {
		t.Errorf("restrictions %v after lifting, want none", repo.restricted)
	}
	want := slices.Repeat([]auditevent.AuditEvent{restrictionEvent(patientA, auditevent.OutcomeSuccess)}, 4)
	if !slices.Equal(audit.events, want) {
		t.Errorf("audit\n got %+v\nwant %+v", audit.events, want)
	}
	if slices.ContainsFunc(repo.calls, func(c string) bool { return c[:min(len(c), 8)] == "outside:" }) {
		t.Errorf("calls %v, want all in a transaction", repo.calls)
	}
	repo.restricted = map[string]string{patientA + "/" + clinicA: staffA}
	if got, err := s.Restricted(t.Context(), patientA, clinicA); err != nil || !got {
		t.Errorf("Restricted = %v, %v; want true", got, err)
	}
}

// A patient of the clinic is one it registered, has a consent with (active or withdrawn) or has a visit with; any
// other id is not found, audited without the id.
func TestRestriction_OnlyTheClinicsPatients(t *testing.T) {
	cases := map[string]struct {
		visits []access.Visit
		state  access.Consent
		found  bool
	}{
		"active consent": {state: access.Consent{Active: true}, found: true},
		"withdrawn":      {state: access.Consent{RevokedAt: now.Add(-time.Hour)}, found: true},
		"a visit":        {visits: []access.Visit{{Start: now}}, found: true},
		"none":           {},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			repo, audit := newFakeRepository(), &fakeAuditor{}
			s := restrictionService(repo, audit, tc.state, tc.visits)
			err := s.Restrict(t.Context(), admin, patientA, "")
			want := restrictionEvent(patientA, auditevent.OutcomeSuccess)
			if !tc.found {
				want = restrictionEvent("", auditevent.OutcomeMinorFailure)
				if !errors.Is(err, patient.ErrNotFound) || len(repo.restricted) != 0 {
					t.Errorf("Restrict = %v with %v, want ErrNotFound and nothing restricted", err, repo.restricted)
				}
			} else if err != nil {
				t.Errorf("Restrict = %v", err)
			}
			if !slices.Equal(audit.events, []auditevent.AuditEvent{want}) {
				t.Errorf("audit %+v, want %+v", audit.events, want)
			}
		})
	}
}

// A registered patient with no consent or visit is the clinic's too.
func TestRestriction_ARegisteredPatient(t *testing.T) {
	repo, audit := newFakeRepository(), &fakeAuditor{}
	repo.profiles["registered"] = patient.Patient{ID: registeredID, RegisteredBy: clinicA}
	s := restrictionService(repo, audit, access.Consent{}, nil)
	if err := s.Restrict(t.Context(), admin, registeredID, ""); err != nil {
		t.Fatalf("Restrict: %v", err)
	}
}

// Only the clinic's admin restricts and lifts: a practitioner, a receptionist and a patient are refused, audited, and
// nothing changes. The erasure's restriction is not the clinic's to lift (S6).
func TestRestriction_RefusesAllButTheAdmin(t *testing.T) {
	for _, c := range []patient.Caller{practitioner, me, {AccountID: staffA, OrganizationID: clinicA, Reception: true}} {
		repo, audit := newFakeRepository(), &fakeAuditor{}
		s := restrictionService(repo, audit, access.Consent{Active: true}, nil)
		if err := s.Restrict(t.Context(), c, patientA, ""); !errors.Is(err, patient.ErrForbidden) {
			t.Errorf("Restrict(%+v) = %v, want ErrForbidden", c, err)
		}
		if err := s.LiftRestriction(t.Context(), c, patientA); !errors.Is(err, patient.ErrForbidden) {
			t.Errorf("LiftRestriction(%+v) = %v, want ErrForbidden", c, err)
		}
		if len(repo.restricted) != 0 || len(audit.events) != 2 || audit.events[0].Outcome != auditevent.OutcomeMinorFailure {
			t.Errorf("restrictions %v, audit %+v; want none and two refusals", repo.restricted, audit.events)
		}
	}
	repo, audit := newFakeRepository(), &fakeAuditor{}
	repo.restricted = map[string]string{patientA + "/" + clinicA: ""}
	s := restrictionService(repo, audit, access.Consent{Active: true}, nil)
	if err := s.LiftRestriction(t.Context(), admin, patientA); !errors.Is(err, patient.ErrForbidden) {
		t.Errorf("LiftRestriction(the erasure's) = %v, want ErrForbidden", err)
	}
	if len(repo.restricted) != 1 {
		t.Errorf("restrictions %v, want the erasure's kept", repo.restricted)
	}
}

// A practitioner of the clinic does not read the profile of a restricted patient, as for one they may not see.
func TestRestriction_HidesTheProfileFromStaff(t *testing.T) {
	repo, audit := newFakeRepository(), &fakeAuditor{}
	repo.restricted = map[string]string{patientA + "/" + clinicA: staffA}
	s := restrictionService(repo, audit, access.Consent{Active: true}, []access.Visit{{PractitionerRole: roleA, Start: now}})
	if _, err := s.Read(t.Context(), practitioner, patientA); !errors.Is(err, patient.ErrNotFound) {
		t.Errorf("Read(restricted) = %v, want ErrNotFound", err)
	}
	delete(repo.restricted, patientA+"/"+clinicA)
	if _, err := s.Read(t.Context(), practitioner, patientA); err != nil {
		t.Errorf("Read(unrestricted) = %v, want the profile", err)
	}
}

func restrictionEvent(id string, outcome auditevent.Outcome) auditevent.AuditEvent {
	e := event(admin, auditevent.CodeUpdate, id, outcome)
	e.Purpose.Code = auditevent.CodePATRQT
	return e
}
