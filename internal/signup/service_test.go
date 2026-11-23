package signup_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"maps"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/auth"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/internal/signup"
	"github.com/Shmeeshmee/salus/internal/subscription"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

var now = time.Date(2026, 11, 23, 9, 0, 0, 0, time.UTC)

const newClinic = "c0000000-0000-4000-8000-00000000c0de"

type fakeRepo struct {
	signups map[string]signup.Signup
	hashes  map[string][]byte // by signup id
	steps   map[string]map[signup.Step]signup.StepState
	scope   string
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{signups: map[string]signup.Signup{}, hashes: map[string][]byte{}, steps: map[string]map[signup.Step]signup.StepState{}}
}

func (r *fakeRepo) Insert(_ context.Context, in signup.Request, hash []byte) (string, error) {
	id := "s" + string(rune('0'+len(r.signups)))
	r.signups[id] = signup.Signup{ID: id, CreatedAt: now, Email: in.Email, ClinicName: in.ClinicName, Language: in.Language}
	r.hashes[id] = hash
	return id, nil
}

func (r *fakeRepo) ByID(_ context.Context, id string) (signup.Signup, error) {
	s, ok := r.signups[id]
	if !ok {
		return signup.Signup{}, signup.ErrNotFound
	}
	return s, nil
}

func (r *fakeRepo) ByTokenHash(_ context.Context, hash []byte) (signup.Signup, error) {
	for id, h := range r.hashes {
		if string(h) == string(hash) {
			return r.signups[id], nil
		}
	}
	return signup.Signup{}, signup.ErrNotFound
}

func (r *fakeRepo) SetToken(_ context.Context, id string, hash []byte) (bool, error) {
	if s, ok := r.signups[id]; !ok || !s.VerifiedAt.IsZero() {
		return false, nil
	}
	r.hashes[id] = hash
	return true, nil
}

func (r *fakeRepo) MarkVerified(_ context.Context, id, org string, at time.Time) (bool, error) {
	s := r.signups[id]
	if !s.VerifiedAt.IsZero() {
		return false, nil
	}
	s.VerifiedAt, s.OrganizationID = at, org
	r.signups[id] = s
	return true, nil
}

func (r *fakeRepo) Verified(context.Context) ([]signup.Signup, error) {
	var out []signup.Signup
	for _, s := range r.signups {
		if !s.VerifiedAt.IsZero() {
			out = append(out, s)
		}
	}
	return out, nil
}

func (r *fakeRepo) Approve(_ context.Context, org string, at time.Time) (bool, error) {
	for id, s := range r.signups {
		if s.OrganizationID == org && s.ApprovedAt.IsZero() {
			s.ApprovedAt = at
			r.signups[id] = s
			return true, nil
		}
	}
	return false, nil
}

func (r *fakeRepo) SetOrganization(_ context.Context, org string) error {
	r.scope = org
	return nil
}

func (r *fakeRepo) Steps(_ context.Context, org string) (map[signup.Step]signup.StepState, error) {
	if r.scope != org {
		return nil, errors.New("out of scope")
	}
	return maps.Clone(r.steps[org]), nil
}

func (r *fakeRepo) PutStep(_ context.Context, org string, step signup.Step, state signup.StepState, _ time.Time) error {
	if r.scope != org {
		return errors.New("out of scope")
	}
	if r.steps[org] == nil {
		r.steps[org] = map[signup.Step]signup.StepState{}
	}
	if state == signup.StepOpen {
		delete(r.steps[org], step)
	} else {
		r.steps[org][step] = state
	}
	return nil
}

func (r *fakeRepo) DeleteUnverifiedBefore(context.Context, time.Time) (int64, error) { return 0, nil }

type fakeTx struct{ n int }

func (f *fakeTx) WithTx(ctx context.Context, fn func(context.Context) error) error {
	f.n++
	return fn(ctx)
}

type fakeJobs struct{ emails []string }

func (j *fakeJobs) EnqueueEmail(_ context.Context, id string) error {
	j.emails = append(j.emails, id)
	return nil
}

type fakeCounters struct{ counts map[string]int64 }

func (c *fakeCounters) Incr(_ context.Context, key string, _ time.Duration) (int64, time.Duration, error) {
	c.counts[key]++
	return c.counts[key], time.Minute, nil
}

type fakeClinics struct {
	created []string
	listed  []string
}

func (c *fakeClinics) Create(_ context.Context, name string) (organization.Organization, error) {
	c.created = append(c.created, name)
	return organization.Organization{ID: newClinic, Name: name}, nil
}

func (c *fakeClinics) List(_ context.Context, id string) error {
	c.listed = append(c.listed, id)
	return nil
}

type fakeAdmins struct{ invited []string }

func (a *fakeAdmins) InviteAdmin(_ context.Context, org, email string) (auth.Invite, error) {
	a.invited = append(a.invited, org+" "+email)
	return auth.Invite{ID: "i1", OrganizationID: org, Email: email}, nil
}

type fakeTrials struct {
	err     error
	started []string
}

func (f *fakeTrials) StartTrial(_ context.Context, org string) (subscription.Subscription, error) {
	if f.err != nil {
		return subscription.Subscription{}, f.err
	}
	f.started = append(f.started, org)
	return subscription.Subscription{}, nil
}

type fixture struct {
	repo     *fakeRepo
	jobs     *fakeJobs
	counters *fakeCounters
	clinics  *fakeClinics
	admins   *fakeAdmins
	trials   *fakeTrials
	svc      *signup.Service
}

func newFixture() *fixture {
	f := &fixture{
		repo: newFakeRepo(), jobs: &fakeJobs{}, counters: &fakeCounters{counts: map[string]int64{}},
		clinics: &fakeClinics{}, admins: &fakeAdmins{}, trials: &fakeTrials{},
	}
	f.svc = signup.NewService(signup.Ports{
		Repo: f.repo, Jobs: f.jobs, Counters: f.counters, Tx: &fakeTx{}, Clock: clock.Fixed{T: now},
		Clinics: f.clinics, Admins: f.admins, Trials: f.trials, RateLimitKey: []byte("synthetic key"),
	})
	return f
}

var valid = signup.Request{Email: " Admin@Example.test ", ClinicName: " Synthetic clinic ", Language: "en"}

func TestStart_StoresTheSignupAndEnqueuesItsEmail(t *testing.T) {
	f := newFixture()
	if err := f.svc.Start(t.Context(), valid, "192.0.2.1"); err != nil {
		t.Fatalf("Start() = %v", err)
	}
	got := f.repo.signups["s0"]
	if got.Email != "admin@example.test" || got.ClinicName != "Synthetic clinic" || got.Language != "en" {
		t.Errorf("stored %+v, want the e-mail lower-cased and both trimmed", got)
	}
	if len(f.jobs.emails) != 1 || f.jobs.emails[0] != "s0" {
		t.Errorf("e-mails enqueued %v, want the signup's", f.jobs.emails)
	}
	if len(f.repo.hashes["s0"]) != sha256.Size {
		t.Errorf("stored token hash of %d bytes, want a SHA-256", len(f.repo.hashes["s0"]))
	}
}

func TestStart_RefusesAnInvalidRequest(t *testing.T) {
	for name, r := range map[string]signup.Request{
		"no e-mail":          {ClinicName: "Synthetic clinic", Language: "en"},
		"not an e-mail":      {Email: "admin", ClinicName: "Synthetic clinic", Language: "en"},
		"a display name":     {Email: "Admin <admin@example.test>", ClinicName: "Synthetic clinic", Language: "en"},
		"too long an e-mail": {Email: strings.Repeat("a", 250) + "@example.test", ClinicName: "Synthetic clinic", Language: "en"},
		"no clinic name":     {Email: "admin@example.test", ClinicName: "  ", Language: "en"},
		"too long a name":    {Email: "admin@example.test", ClinicName: strings.Repeat("x", 201), Language: "en"},
		"a patient language": {Email: "admin@example.test", ClinicName: "Synthetic clinic", Language: "de"},
		"a line break":       {Email: "admin@example.test", ClinicName: "Synthetic\nclinic", Language: "en"},
		"a link":             {Email: "admin@example.test", ClinicName: "See https://example.test", Language: "en"},
	} {
		f := newFixture()
		if err := f.svc.Start(t.Context(), r, "192.0.2.1"); !errors.Is(err, signup.ErrInvalid) || len(f.repo.signups) != 0 {
			t.Errorf("%s: Start() = %v, stored %d; want ErrInvalid, nothing stored", name, err, len(f.repo.signups))
		}
	}
}

// Five signups an hour from one address, three a day for one e-mail; the counters' keys name neither.
func TestStart_IsRateLimitedPerAddressAndPerEmail(t *testing.T) {
	f := newFixture()
	for i := range 5 {
		r := valid
		r.Email = string(rune('a'+i)) + "@example.test"
		if err := f.svc.Start(t.Context(), r, "192.0.2.1"); err != nil {
			t.Fatalf("Start() #%d = %v", i+1, err)
		}
	}
	var limited *signup.RateLimitedError
	if err := f.svc.Start(t.Context(), valid, "192.0.2.1"); !errors.As(err, &limited) || len(f.repo.signups) != 5 {
		t.Errorf("Start() #6 from one address = %v, want RateLimitedError, nothing stored", err)
	}
	for i := range 3 {
		if err := f.svc.Start(t.Context(), valid, "192.0.2."+string(rune('2'+i))); err != nil {
			t.Fatalf("Start() of one e-mail #%d = %v", i+1, err)
		}
	}
	if err := f.svc.Start(t.Context(), valid, "192.0.2.9"); !errors.As(err, &limited) {
		t.Errorf("Start() #4 of one e-mail = %v, want RateLimitedError", err)
	}
	for key := range f.counters.counts {
		if strings.Contains(key, "192.0.2") || strings.Contains(key, "example") {
			t.Errorf("counter key %q names the address or the e-mail", key)
		}
	}
}

// startAndIssue signs a clinic up and returns the token of its verification e-mail.
func startAndIssue(t *testing.T, f *fixture) string {
	t.Helper()
	if err := f.svc.Start(t.Context(), valid, "192.0.2.1"); err != nil {
		t.Fatal(err)
	}
	v, err := f.svc.IssueVerification(t.Context(), "s0")
	if err != nil || v.Email != "admin@example.test" || v.ClinicName != "Synthetic clinic" || v.Language != "en" || v.Token == "" {
		t.Fatalf("IssueVerification() = %+v, %v", v, err)
	}
	return v.Token
}

func TestIssueVerification_ReplacesTheTokenUntilVerified(t *testing.T) {
	f := newFixture()
	first := startAndIssue(t, f)
	second, err := f.svc.IssueVerification(t.Context(), "s0")
	if err != nil || second.Token == first {
		t.Fatalf("IssueVerification() again = %+v, %v; want a new token", second, err)
	}
	if _, err := f.svc.Verify(t.Context(), first, "192.0.2.1"); !errors.Is(err, signup.ErrInvalidToken) {
		t.Errorf("Verify(the replaced token) = %v, want ErrInvalidToken", err)
	}
	if _, err := f.svc.Verify(t.Context(), second.Token, "192.0.2.1"); err != nil {
		t.Fatalf("Verify(the newest token) = %v", err)
	}
	if _, err := f.svc.IssueVerification(t.Context(), "s0"); !errors.Is(err, signup.ErrInvalidToken) {
		t.Errorf("IssueVerification() of a verified signup = %v, want ErrInvalidToken", err)
	}
	if _, err := f.svc.IssueVerification(t.Context(), "unknown"); !errors.Is(err, signup.ErrInvalidToken) {
		t.Errorf("IssueVerification() of an unknown signup = %v, want ErrInvalidToken", err)
	}
}

func TestVerify_CreatesTheClinicInvitesItsAdminAndStartsTheTrial(t *testing.T) {
	f := newFixture()
	token := startAndIssue(t, f)

	got, err := f.svc.Verify(t.Context(), token, "192.0.2.1")

	if err != nil || got.OrganizationID != newClinic || !got.VerifiedAt.Equal(now) {
		t.Fatalf("Verify() = %+v, %v; want the signup verified with its clinic", got, err)
	}
	if len(f.clinics.created) != 1 || f.clinics.created[0] != "Synthetic clinic" {
		t.Errorf("clinics created %v", f.clinics.created)
	}
	if len(f.admins.invited) != 1 || f.admins.invited[0] != newClinic+" admin@example.test" {
		t.Errorf("admins invited %v", f.admins.invited)
	}
	if len(f.trials.started) != 1 || f.trials.started[0] != newClinic {
		t.Errorf("trials started %v", f.trials.started)
	}
	if len(f.clinics.listed) != 0 {
		t.Errorf("clinics listed %v, want none before the operator approves", f.clinics.listed)
	}
	if _, err := f.svc.Verify(t.Context(), token, "192.0.2.1"); !errors.Is(err, signup.ErrInvalidToken) || len(f.clinics.created) != 1 {
		t.Errorf("Verify() twice = %v, created %v; want ErrInvalidToken and one clinic", err, f.clinics.created)
	}
}

func TestVerify_RefusesAnUnknownOrExpiredToken(t *testing.T) {
	f := newFixture()
	token := startAndIssue(t, f)
	s := f.repo.signups["s0"]
	s.CreatedAt = now.Add(-signup.TokenLifetime - time.Second)
	f.repo.signups["s0"] = s
	for name, tok := range map[string]string{"expired": token, "unknown": "unknown", "empty": ""} {
		if _, err := f.svc.Verify(t.Context(), tok, "192.0.2.1"); !errors.Is(err, signup.ErrInvalidToken) {
			t.Errorf("Verify(%s) = %v, want ErrInvalidToken", name, err)
		}
	}
	if len(f.clinics.created) != 0 {
		t.Errorf("clinics created %v, want none", f.clinics.created)
	}
}

func TestVerify_IsRateLimitedPerAddress(t *testing.T) {
	f := newFixture()
	for range 20 {
		if _, err := f.svc.Verify(t.Context(), "unknown", "192.0.2.1"); !errors.Is(err, signup.ErrInvalidToken) {
			t.Fatalf("Verify() = %v", err)
		}
	}
	var limited *signup.RateLimitedError
	if _, err := f.svc.Verify(t.Context(), "unknown", "192.0.2.1"); !errors.As(err, &limited) {
		t.Errorf("Verify() #21 = %v, want RateLimitedError", err)
	}
}

// As salusctl create-clinic: without the owner's default plan the clinic is created without a trial.
func TestCreateClinic_WithoutAPlanCreatesTheClinicWithoutATrial(t *testing.T) {
	f := newFixture()
	f.trials.err = subscription.ErrNoPlan
	got, err := f.svc.CreateClinic(t.Context(), "Synthetic clinic", "admin@example.test")
	if want := (signup.Created{OrganizationID: newClinic, InviteID: "i1"}); err != nil || got != want {
		t.Errorf("CreateClinic() = %+v, %v; want %+v", got, err, want)
	}
	f.trials.err = nil
	if got, err := f.svc.CreateClinic(t.Context(), "Synthetic clinic", "admin@example.test"); err != nil || !got.Trial {
		t.Errorf("CreateClinic() with a plan = %+v, %v; want the trial started", got, err)
	}
}

func TestApprove_ListsTheVerifiedClinicOnce(t *testing.T) {
	f := newFixture()
	if _, err := f.svc.Verify(t.Context(), startAndIssue(t, f), "192.0.2.1"); err != nil {
		t.Fatal(err)
	}
	if err := f.svc.Approve(t.Context(), newClinic); err != nil || len(f.clinics.listed) != 1 || f.repo.signups["s0"].ApprovedAt.IsZero() {
		t.Fatalf("Approve() = %v, listed %v", err, f.clinics.listed)
	}
	if err := f.svc.Approve(t.Context(), newClinic); !errors.Is(err, signup.ErrNotFound) || len(f.clinics.listed) != 1 {
		t.Errorf("Approve() twice = %v, listed %v; want ErrNotFound", err, f.clinics.listed)
	}
	list, err := f.svc.Signups(t.Context())
	if err != nil || len(list) != 1 || list[0].OrganizationID != newClinic {
		t.Errorf("Signups() = %+v, %v", list, err)
	}
}

func TestSetup_EveryStepOpenUntilSavedAndOnlyTheAdminSaves(t *testing.T) {
	f := newFixture()
	admin := signup.Caller{AccountID: "a1", OrganizationID: newClinic, Admin: true}
	got, err := f.svc.Setup(t.Context(), admin)
	if err != nil || len(got.Steps) != 5 || got.Steps[signup.StepDetails] != signup.StepOpen {
		t.Fatalf("Setup() = %+v, %v; want five open steps", got, err)
	}
	in := signup.Setup{Steps: map[signup.Step]signup.StepState{
		signup.StepDetails: signup.StepDone, signup.StepBilling: signup.StepSkipped, signup.StepHours: signup.StepOpen,
		signup.StepPractitioner: signup.StepOpen, signup.StepServices: signup.StepOpen,
	}}
	saved, err := f.svc.SaveSetup(t.Context(), admin, in)
	if err != nil || !maps.Equal(saved.Steps, in.Steps) {
		t.Fatalf("SaveSetup() = %+v, %v", saved, err)
	}
	member := signup.Caller{AccountID: "a2", OrganizationID: newClinic}
	if got, err := f.svc.Setup(t.Context(), member); err != nil || !maps.Equal(got.Steps, in.Steps) {
		t.Errorf("Setup() by a member = %+v, %v; want the saved progress", got, err)
	}
	if _, err := f.svc.SaveSetup(t.Context(), member, in); !errors.Is(err, signup.ErrForbidden) {
		t.Errorf("SaveSetup() by a member = %v, want ErrForbidden", err)
	}
	if _, err := f.svc.Setup(t.Context(), signup.Caller{AccountID: "p1"}); !errors.Is(err, signup.ErrNotFound) {
		t.Errorf("Setup() without a clinic = %v, want ErrNotFound", err)
	}
	for name, steps := range map[string]map[signup.Step]signup.StepState{
		"unknown step":  {"payments": signup.StepDone},
		"unknown state": {signup.StepDetails: "half"},
	} {
		if _, err := f.svc.SaveSetup(t.Context(), admin, signup.Setup{Steps: steps}); !errors.Is(err, signup.ErrInvalidSetup) {
			t.Errorf("SaveSetup(%s) = %v, want ErrInvalidSetup", name, err)
		}
	}
}
