package appointment_test

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"slices"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

// fakeSelfService confirms and moves the fixture's appointments (in f.repo.appts, so the fixture's rollback covers
// them) and keeps the stored links of its tests.
type fakeSelfService struct {
	f      *fixture
	onLock func() // runs before ByIDForUpdate reads: a concurrent change that commits first
	tokens []appointment.Token
	calls  []string
}

func (s *fakeSelfService) find(ctx context.Context, call, id string) (int, error) {
	s.calls = append(s.calls, call)
	if ctx.Value(txKey{}) == nil {
		return 0, errors.New(call + " outside a transaction")
	}
	if s.f.repo.err != nil {
		return 0, s.f.repo.err
	}
	for i, a := range s.f.repo.appts {
		if a.ID == id && s.f.repo.patient != "" && a.PatientID == s.f.repo.patient {
			return i, nil
		}
	}
	return 0, appointment.ErrNotFound
}

func (s *fakeSelfService) ByIDForUpdate(ctx context.Context, id string) (appointment.Appointment, error) {
	if s.onLock != nil {
		s.onLock()
	}
	i, err := s.find(ctx, "ByIDForUpdate", id)
	if err != nil {
		return appointment.Appointment{}, err
	}
	return s.f.repo.appts[i], nil
}

func (s *fakeSelfService) Confirm(ctx context.Context, id string, via appointment.Via, at time.Time) (appointment.Appointment, error) {
	i, err := s.find(ctx, "Confirm", id)
	switch {
	case errors.Is(err, appointment.ErrNotFound):
		return appointment.Appointment{}, appointment.ErrClosed
	case err != nil:
		return appointment.Appointment{}, err
	}
	a := &s.f.repo.appts[i]
	if a.Status != appointment.StatusBooked || !a.ConfirmedAt.IsZero() {
		return appointment.Appointment{}, appointment.ErrClosed
	}
	a.ConfirmedAt, a.ConfirmedVia, a.LastUpdated = at, via, at
	a.VersionID++
	return *a, nil
}

func (s *fakeSelfService) Reschedule(ctx context.Context, id string, m appointment.Move, at time.Time) (appointment.Appointment, error) {
	i, err := s.find(ctx, "Reschedule", id)
	switch {
	case errors.Is(err, appointment.ErrNotFound):
		return appointment.Appointment{}, appointment.ErrClosed
	case err != nil:
		return appointment.Appointment{}, err
	}
	a := &s.f.repo.appts[i]
	if a.Status != appointment.StatusBooked || a.RescheduleCount != m.From {
		return appointment.Appointment{}, appointment.ErrClosed
	}
	for _, b := range s.f.repo.appts {
		if b.ID != a.ID && b.PractitionerRoleID == a.PractitionerRoleID && b.Status != appointment.StatusCancelled && b.Start.Before(m.End) && m.Start.Before(b.End) {
			return appointment.Appointment{}, appointment.ErrSlotUnavailable
		}
	}
	a.Start, a.End, a.SlotID, a.LastUpdated = m.Start, m.End, m.SlotID, at
	a.RescheduleCount++
	a.VersionID++
	if a.ConfirmedAt.IsZero() {
		a.ConfirmedAt, a.ConfirmedVia = at, appointment.ViaApp
	}
	return *a, nil
}

func (s *fakeSelfService) FindToken(_ context.Context, hash [32]byte) (appointment.Token, error) {
	s.calls = append(s.calls, "FindToken")
	for _, t := range s.tokens {
		if t.Hash == hash {
			return appointment.Token{ID: t.ID, AppointmentID: t.AppointmentID, OrganizationID: t.OrganizationID, PatientID: t.PatientID, AccountID: t.AccountID}, nil
		}
	}
	return appointment.Token{}, appointment.ErrLinkInvalid
}

func (s *fakeSelfService) TokenByID(ctx context.Context, id string) (appointment.Token, error) {
	s.calls = append(s.calls, "TokenByID")
	if ctx.Value(txKey{}) == nil {
		return appointment.Token{}, errors.New("TokenByID outside a transaction")
	}
	for _, t := range s.tokens {
		if t.ID == id && t.PatientID == s.f.repo.patient {
			return t, nil
		}
	}
	return appointment.Token{}, appointment.ErrLinkInvalid
}

func (s *fakeSelfService) UseToken(ctx context.Context, id string, action appointment.LinkAction, at time.Time) error {
	s.calls = append(s.calls, "UseToken:"+string(action))
	if ctx.Value(txKey{}) == nil {
		return errors.New("UseToken outside a transaction")
	}
	for i, t := range s.tokens {
		if t.ID == id && t.PatientID == s.f.repo.patient {
			if !t.UsedAt.IsZero() {
				return appointment.ErrLinkInvalid
			}
			s.tokens[i].UsedAt, s.tokens[i].UsedAction = at, action
			return nil
		}
	}
	return appointment.ErrLinkInvalid
}

// fakeSettings gives each clinic its appointment settings, the defaults for one without.
type fakeSettings struct {
	err  error
	repo *fakeRepository // For sets the clinic's scope on the transaction, as organization's does
	set  map[string]organization.AppointmentSettings
}

func (s *fakeSettings) For(_ context.Context, org string) (organization.AppointmentSettings, error) {
	if s.repo != nil {
		s.repo.clinic = org
		s.repo.calls = append(s.repo.calls, "For")
	}
	if v, ok := s.set[org]; ok {
		return v, s.err
	}
	return organization.DefaultAppointmentSettings(), s.err
}

// fakeLimiter counts lookups per key and allows up to the limit; keys are kept to check no address is in them.
type fakeLimiter struct {
	err    error
	counts map[string]int
}

func (l *fakeLimiter) Allow(_ context.Context, key string, limit int, _ time.Duration) (bool, error) {
	if l.counts == nil {
		l.counts = map[string]int{}
	}
	l.counts[key]++
	return l.counts[key] <= limit, l.err
}

// selfTx is the fixture's transactor that also rolls back the stored links.
type selfTx struct {
	inner fakeTransactor
	s     *fakeSelfService
}

func (t selfTx) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if ctx.Value(txKey{}) != nil {
		return fn(ctx)
	}
	saved := slices.Clone(t.s.tokens)
	err := t.inner.WithTx(ctx, fn)
	if err != nil {
		t.s.tokens = saved
	}
	return err
}

// selfFixture is newFixture with the ports of the patient's self-service.
type selfFixture struct {
	*fixture
	self     *fakeSelfService
	settings *fakeSettings
	limiter  *fakeLimiter
}

func newSelfFixture(appts ...appointment.Appointment) *selfFixture {
	return newSelfFixtureAt(now, appts...)
}

// newSelfFixtureAt is newSelfFixture with its clock at at.
func newSelfFixtureAt(at time.Time, appts ...appointment.Appointment) *selfFixture {
	f := &selfFixture{fixture: newFixture(appts...), settings: &fakeSettings{set: map[string]organization.AppointmentSettings{}}, limiter: &fakeLimiter{}}
	f.self = &fakeSelfService{f: f.fixture}
	f.settings.repo = f.repo
	f.svc = appointment.NewService(appointment.Ports{
		Repo: f.repo, Audit: f.audit, Tx: selfTx{inner: fakeTransactor{f: f.fixture}, s: f.self}, Clock: clock.Fixed{T: at},
		Consents: f.consents, Slots: f.slots, Patients: f.patients, Clinics: f.clinics, Roles: f.roles, Jobs: f.jobs,
		Tokens: f.tokens, LinkURL: linkURL, Stats: f.stats,
		Reminders: f.reminders, Rules: f.settings,
		SelfService: f.self, Limiter: f.limiter, LinkKey: []byte("a synthetic key of at least 32 bytes"),
	})
	return f
}

// link stores a one-time link of the appointment a issued for its start, and returns its token.
func (f *selfFixture) link(a appointment.Appointment, id, language string) string {
	raw := sha256.Sum256([]byte(id)) // any 32 bytes
	f.self.tokens = append(f.self.tokens, appointment.Token{
		ID: id, CreatedAt: now.Add(-time.Hour), StartAt: a.Start, AppointmentID: a.ID, OrganizationID: a.OrganizationID,
		PatientID: a.PatientID, AccountID: map[string]string{patientA: accountA, patientB: accountB}[a.PatientID],
		Language: language, Channel: appointment.ChannelEmail, Hash: sha256.Sum256(raw[:]),
	})
	return base64.RawURLEncoding.EncodeToString(raw[:])
}
