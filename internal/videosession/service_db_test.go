package videosession_test

import (
	"context"
	"slices"
	"testing"

	"github.com/Shmeeshmee/salus/internal/videosession"
)

// Against the compose LiveKit: the token the service issues joins the room it created, as the session's
// patient identity.
func TestJoin_TheServicesTokenJoinsTheComposeLiveKit(t *testing.T) {
	t.Parallel()
	lk := newLiveKitServer(t)
	f := newFixture(now)
	p := lk.provider()
	svc := videosession.NewService(videosession.Ports{
		Repo: f.repo, Provider: p, Audit: f.audit, Tx: fakeTransactor{f: f}, Clock: f.svc.Clock,
		Appointments: f.appts, Patients: f.patients, Practitioners: f.practitioners, PublicURL: publicURL,
		Jobs: f.jobs, Encounters: f.encounters,
	})

	j, err := svc.Join(t.Context(), me, apptA)
	if err != nil {
		t.Fatal(err)
	}
	s := f.repo.sessions[apptA]
	t.Cleanup(func() { _ = p.DeleteRoom(context.Background(), s.Room) })
	if r := lk.listRoom(t, s.Room); r == nil || r.MaxParticipants != 2 || r.Metadata != "" {
		t.Errorf("room %+v; want the service's room for two, without metadata", r)
	}
	if c, _ := lk.connect(t, j.Token); c == nil {
		t.Fatal("the service's token did not join")
	}
	if got, err := p.Participants(t.Context(), s.Room); err != nil || !slices.Equal(got, []string{s.PatientIdentity}) {
		t.Errorf("Participants() = %v, %v; want the patient's identity", got, err)
	}
}
