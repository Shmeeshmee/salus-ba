package appointment_test

import (
	"context"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/organization"
)

// fakeNoShows corrects the appointments of the fixture's repository as the UPDATE policies let it: an appointment
// the scope shows, in the status it is corrected from.
type fakeNoShows struct{ r *fakeRepository }

func (n fakeNoShows) CorrectStatus(ctx context.Context, id string, from, to appointment.Status, at time.Time) (appointment.Appointment, error) {
	n.r.seen(ctx, "CorrectStatus:"+string(to))
	if n.r.err != nil {
		return appointment.Appointment{}, n.r.err
	}
	for i, a := range n.r.appts {
		if a.ID == id && n.r.visible(a) && a.Status == from {
			a.Status, a.LastUpdated = to, at
			a.VersionID++
			n.r.appts[i] = a
			return a, nil
		}
	}
	return appointment.Appointment{}, appointment.ErrClosed
}

// fakeRules is the clinics' appointment settings, the defaults for any clinic without its own.
type fakeRules struct {
	err error
	set map[string]organization.AppointmentSettings
}

func (r fakeRules) For(_ context.Context, org string) (organization.AppointmentSettings, error) {
	if s, ok := r.set[org]; ok {
		return s, r.err
	}
	return organization.DefaultAppointmentSettings(), r.err
}

// noShowFixture is cancelFixture with the no-show ports, and clinic A's flag at threshold with a window of days.
func noShowFixture(at time.Time, threshold, days int) (*fixture, fakeRules) {
	f := withNoShows(cancelFixture(at), threshold, days)
	return f, f.svc.Rules.(fakeRules)
}

// racedNoShows is fakeNoShows after another correction committed first: the appointment is no longer in the status
// it is corrected from.
type racedNoShows struct{ fakeNoShows }

func (n racedNoShows) CorrectStatus(ctx context.Context, id string, _, to appointment.Status, at time.Time) (appointment.Appointment, error) {
	return n.fakeNoShows.CorrectStatus(ctx, id, to, to, at)
}

// withNoShows gives f's Service the no-show ports, and clinic A's flag at threshold with a window of days.
func withNoShows(f *fixture, threshold, days int) *fixture {
	set := organization.DefaultAppointmentSettings()
	set.NoShowFlagThreshold, set.NoShowFlagWindowDays = threshold, days
	p := f.svc.Ports
	p.NoShows, p.Rules = fakeNoShows{r: f.repo}, fakeRules{set: map[string]organization.AppointmentSettings{clinicA: set}}
	f.svc = appointment.NewService(p)
	return f
}

// withStatus is a with status.
func withStatus(a appointment.Appointment, status appointment.Status) appointment.Appointment {
	a.Status = status
	return a
}
