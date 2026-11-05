package videosession_test

import (
	"errors"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/videosession"
)

// checkEvents fails unless the audit log holds exactly want.
func checkEvents(t *testing.T, what string, a *fakeAuditor, want ...auditevent.AuditEvent) {
	t.Helper()
	if !slices.Equal(a.events, want) {
		t.Errorf("%s audited %+v;\nwant %+v", what, a.events, want)
	}
}

func TestJoin_GivesThePatientATokenForTheRoomAndThePractitionersName(t *testing.T) {
	f := newFixture(now)
	got, err := f.svc.Join(t.Context(), me, apptA)
	if err != nil {
		t.Fatalf("Join() error = %v", err)
	}
	s := f.repo.sessions[apptA]
	want := videosession.Join{
		URL: publicURL, Token: "token:" + s.PatientIdentity, ExpiresAt: now.Add(5 * time.Minute),
		PeerName: "Cy Example", Role: videosession.RolePatient,
	}
	if got != want {
		t.Errorf("Join() = %+v,\nwant %+v", got, want)
	}
	if want := []videosession.Grant{{Room: s.Room, Identity: s.PatientIdentity, ValidFor: 5 * time.Minute}}; !slices.Equal(f.provider.grants, want) {
		t.Errorf("granted %+v, want %+v", f.provider.grants, want)
	}
	if f.repo.patient != patientA || f.repo.clinic != "" {
		t.Errorf("opened in patient %q, clinic %q; want patient A's scope", f.repo.patient, f.repo.clinic)
	}
	if len(f.patients.lookup) != 0 {
		t.Errorf("looked up patients %v, want none: the patient gets the practitioner's public name", f.patients.lookup)
	}
	checkEvents(t, "Join", f.audit, joinEvent(me, "Appointment/"+apptA, auditevent.OutcomeSuccess))
}

// The practitioner gets the patient's name at the appointment's access level, and that read is audited.
func TestJoin_GivesThePractitionerTheirOwnIdentityAndThePatientsNameAudited(t *testing.T) {
	f := newFixture(now)
	got, err := f.svc.Join(t.Context(), pract, apptA)
	s := f.repo.sessions[apptA]
	if err != nil || got.Token != "token:"+s.PractitionerIdentity || got.PeerName != "Ada Example" || got.Role != videosession.RolePractitioner {
		t.Fatalf("Join() = %+v, %v; want the practitioner's token, patient A's name and the practitioner role", got, err)
	}
	if f.repo.clinic != clinicA || f.repo.patient != "" {
		t.Errorf("opened in patient %q, clinic %q; want clinic A's scope", f.repo.patient, f.repo.clinic)
	}
	if len(f.practitioners.asked) != 0 {
		t.Errorf("asked the directory for %v, want nothing", f.practitioners.asked)
	}
	checkEvents(t, "Join", f.audit, patientRead(pract), joinEvent(pract, "Appointment/"+apptA, auditevent.OutcomeSuccess))
}

// The room: two participants at most, no metadata, kept while empty until 2 hours after the scheduled end
// (the hard cap), created outside the transaction on every request (CreateRoom is idempotent).
func TestJoin_CreatesTheRoomOnEveryRequestOutsideTheTransaction(t *testing.T) {
	f := newFixture(now)
	for _, c := range []appointment.Caller{me, pract, me} {
		if _, err := f.svc.Join(t.Context(), c, apptA); err != nil {
			t.Fatal(err)
		}
	}
	room := videosession.Room{Name: f.repo.sessions[apptA].Room, MaxParticipants: 2, EmptyTimeout: end.Add(2 * time.Hour).Sub(now)}
	if want := []videosession.Room{room, room, room}; !slices.Equal(f.provider.rooms, want) {
		t.Errorf("created %+v, want %+v", f.provider.rooms, want)
	}
	if f.provider.inTx {
		t.Error("the provider was called inside the transaction")
	}
	if len(f.repo.sessions) != 1 {
		t.Errorf("%d sessions, want one per appointment", len(f.repo.sessions))
	}
}

// The first request stores the session with its window's close; later ones reuse it, so the room and each
// role's identity stay the same.
func TestJoin_KeepsOneSessionPerAppointment(t *testing.T) {
	f := newFixture(now)
	first, err := f.svc.Join(t.Context(), me, apptA)
	if err != nil {
		t.Fatal(err)
	}
	again, err := f.svc.Join(t.Context(), me, apptA)
	if err != nil || again.Token != first.Token {
		t.Errorf("second Join() = %+v, %v; want the same identity's token %q", again, err, first.Token)
	}
	s := f.repo.opened[0]
	if s.AppointmentID != apptA || s.OrganizationID != clinicA || !s.ClosesAt.Equal(end.Add(30*time.Minute)) || !s.CreatedAt.Equal(now) {
		t.Errorf("opened %+v; want appointment A's of clinic A, closing 30 minutes after its end, created now", s)
	}
}

// Room and identities are 128 random bits, base32, prefixed r_ and p_; they differ per session and per role,
// and are never one of our ids.
func TestJoin_NamesRoomsAndIdentitiesAtRandom(t *testing.T) {
	f := newFixture(now)
	f.appts.attendance[accountA+apptA2] = f.appts.attendance[accountA+apptA]
	for _, id := range []string{apptA, apptA2} {
		if _, err := f.svc.Join(t.Context(), me, id); err != nil {
			t.Fatal(err)
		}
	}
	room, identity := regexp.MustCompile(`^r_[a-z2-7]{26}$`), regexp.MustCompile(`^p_[a-z2-7]{26}$`)
	seen := map[string]bool{}
	for _, s := range f.repo.sessions {
		for _, name := range []string{s.Room, s.PatientIdentity, s.PractitionerIdentity} {
			if seen[name] {
				t.Errorf("%q is used twice", name)
			}
			seen[name] = true
			for _, ours := range []string{apptA, apptA2, patientA, accountA, clinicA, roleA1, practA1, s.ID} {
				if strings.Contains(name, ours) || strings.Contains(name, ours[:8]) {
					t.Errorf("%q holds our id %s", name, ours)
				}
			}
		}
		if !room.MatchString(s.Room) || !identity.MatchString(s.PatientIdentity) || !identity.MatchString(s.PractitionerIdentity) {
			t.Errorf("session %+v: want r_ and p_ with 26 base32 characters", s)
		}
	}
}

// The window opens 10 minutes before the start and closes 30 minutes after the end.
func TestJoin_IssuesTokensOnlyInsideTheWindow(t *testing.T) {
	opens, closes := start.Add(-10*time.Minute), end.Add(30*time.Minute)
	tests := []struct {
		at   time.Time
		want error
	}{
		{opens.Add(-time.Second), videosession.ErrJoinWindowNotOpen},
		{opens, nil},
		{opens.Add(time.Second), nil},
		{closes.Add(-time.Second), nil},
		{closes, videosession.ErrConsultationEnded},
		{closes.Add(time.Second), videosession.ErrConsultationEnded},
	}
	for _, tt := range tests {
		for _, c := range []appointment.Caller{me, pract} {
			f := newFixture(tt.at)
			_, err := f.svc.Join(t.Context(), c, apptA)
			if !errors.Is(err, tt.want) {
				t.Errorf("Join() as %s at %s = %v, want %v", c.AccountID, tt.at.Format(time.TimeOnly), err, tt.want)
			}
			if tt.want != nil && (len(f.repo.sessions) != 0 || len(f.provider.grants) != 0) {
				t.Errorf("at %s: a session or a token outside the window", tt.at.Format(time.TimeOnly))
			}
		}
	}
}

// Only the appointment's patient and its practitioner join; anyone else gets ErrNotFound, audited, with no
// session and no token. The entity is stored only when the caller may see the appointment.
func TestJoin_IsNotFoundForAnyoneButTheTwoParticipants(t *testing.T) {
	tests := []struct {
		name   string
		entity string
		caller appointment.Caller
	}{
		{name: "another patient", caller: other, entity: ""},
		{name: "another practitioner of the clinic", caller: pract2, entity: ""},
		{name: "the clinic's admin", caller: admin, entity: "Appointment/" + apptA},
		{name: "another clinic's admin", caller: adminB, entity: ""},
		{name: "anonymous", caller: anonymous, entity: ""},
	}
	for _, tt := range tests {
		f := newFixture(now)
		if _, err := f.svc.Join(t.Context(), tt.caller, apptA); !errors.Is(err, videosession.ErrNotFound) {
			t.Errorf("%s: Join() = %v, want ErrNotFound", tt.name, err)
		}
		if len(f.repo.sessions) != 0 || len(f.provider.grants) != 0 || len(f.provider.rooms) != 0 {
			t.Errorf("%s: a session, room or token for someone else", tt.name)
		}
		checkEvents(t, tt.name, f.audit, joinEvent(tt.caller, tt.entity, auditevent.OutcomeMinorFailure))
	}
}

// The consent is checked for both (the practitioner sees and hears the patient), before the status: after a
// revocation the cancelled appointment answers consent_required.
func TestJoin_RequiresTheActiveConsent(t *testing.T) {
	for _, status := range []appointment.Status{appointment.StatusBooked, appointment.StatusCancelled} {
		for _, c := range []appointment.Caller{me, pract} {
			f := newFixture(now)
			f.change(func(a *appointment.Attendance) { a.ConsentActive, a.Appointment.Status = false, status })
			if _, err := f.svc.Join(t.Context(), c, apptA); !errors.Is(err, videosession.ErrConsentRequired) {
				t.Errorf("%s as %s without a consent: Join() = %v, want ErrConsentRequired", status, c.AccountID, err)
			}
			if len(f.repo.sessions) != 0 || len(f.provider.grants) != 0 {
				t.Errorf("%s as %s: a session or a token without a consent", status, c.AccountID)
			}
			checkEvents(t, "Join without a consent", f.audit, joinEvent(c, "Appointment/"+apptA, auditevent.OutcomeMinorFailure))
		}
	}
}

func TestJoin_EndsWithTheAppointment(t *testing.T) {
	for _, status := range []appointment.Status{appointment.StatusCancelled, appointment.StatusFulfilled, appointment.StatusNoShow} {
		f := newFixture(now)
		f.change(func(a *appointment.Attendance) { a.Appointment.Status = status })
		if _, err := f.svc.Join(t.Context(), me, apptA); !errors.Is(err, videosession.ErrConsultationEnded) {
			t.Errorf("%s: Join() = %v, want ErrConsultationEnded", status, err)
		}
		checkEvents(t, string(status), f.audit, joinEvent(me, "Appointment/"+apptA, auditevent.OutcomeMinorFailure))
	}
}

func TestJoin_RefusesAnEndedSession(t *testing.T) {
	f := newFixture(now)
	f.repo.sessions[apptA] = videosession.Session{AppointmentID: apptA, Room: "r_ended", EndedAt: now.Add(-time.Minute), EndReason: videosession.EndPractitioner}
	for _, c := range []appointment.Caller{me, pract} {
		if _, err := f.svc.Join(t.Context(), c, apptA); !errors.Is(err, videosession.ErrConsultationEnded) {
			t.Errorf("as %s: Join() = %v, want ErrConsultationEnded", c.AccountID, err)
		}
	}
	if len(f.provider.grants) != 0 || len(f.provider.rooms) != 0 {
		t.Error("a room or a token for an ended session")
	}
	checkEvents(t, "Join of an ended session", f.audit,
		joinEvent(me, "Appointment/"+apptA, auditevent.OutcomeMinorFailure), joinEvent(pract, "Appointment/"+apptA, auditevent.OutcomeMinorFailure))
}

// Without the practitioner in the public directory the patient still joins, without a name.
func TestJoin_GivesNoNameForAPractitionerOutsideTheDirectory(t *testing.T) {
	f := newFixture(now)
	f.change(func(a *appointment.Attendance) { a.PractitionerID = "p9000000-0000-4000-8000-000000000009" })
	if got, err := f.svc.Join(t.Context(), me, apptA); err != nil || got.PeerName != "" {
		t.Errorf("Join() = %+v, %v; want a token without a name", got, err)
	}
}

// A failure fails the join; its text holds no token, room, identity or name, and a failure inside the
// transaction leaves no session and no event.
func TestJoin_FailsWithTheBackendWithoutLeakingNames(t *testing.T) {
	tests := []struct {
		fail func(f *fixture)
		name string
		inTx bool
	}{
		{name: "appointments", fail: func(f *fixture) { f.appts.err = errBackend }, inTx: true},
		{name: "repository", fail: func(f *fixture) { f.repo.err = errBackend }, inTx: true},
		{name: "patients", fail: func(f *fixture) { f.patients.err = errBackend }, inTx: true},
		{name: "audit", fail: func(f *fixture) { f.audit.err = errBackend }, inTx: true},
		{name: "directory", fail: func(f *fixture) { f.practitioners.err = errBackend }, inTx: false},
		{name: "room", fail: func(f *fixture) { f.provider.err = errBackend }, inTx: false},
		{name: "token", fail: func(f *fixture) { f.provider.tokenErr = errBackend }, inTx: false},
	}
	for _, tt := range tests {
		for _, c := range []appointment.Caller{me, pract} {
			f := newFixture(now)
			tt.fail(f)
			got, err := f.svc.Join(t.Context(), c, apptA)
			if tt.name == "patients" && c == me || tt.name == "directory" && c == pract {
				continue // the patient reads no profile, the practitioner no directory
			}
			if !errors.Is(err, errBackend) || got != (videosession.Join{}) {
				t.Errorf("%s as %s: Join() = %+v, %v; want the backend's error", tt.name, c.AccountID, got, err)
				continue
			}
			s := f.repo.opened
			for _, secret := range []string{"token:", "Ada", "Cy", "r_", "p_"} {
				if strings.Contains(err.Error(), secret) {
					t.Errorf("%s: error %q holds %q", tt.name, err, secret)
				}
			}
			if tt.inTx && (len(f.repo.sessions) != 0 || len(f.audit.events) != 0) {
				t.Errorf("%s as %s: %d sessions and %v after a failed transaction (opened %d)", tt.name, c.AccountID, len(f.repo.sessions), f.audit.events, len(s))
			}
		}
	}
}

// A join on a session that ended without the practitioner, once the window has closed, answers
// ErrPractitionerDidNotJoin to either side (the patient's "the practitioner did not join"), audited; any other
// end, or no session at all, ErrConsultationEnded. No session is opened.
func TestJoin_SaysThePractitionerDidNotJoinOnceTheSessionEndedWithoutThem(t *testing.T) {
	for _, tt := range []struct {
		want   error
		reason videosession.EndReason
	}{
		{reason: videosession.EndPractitionerAbsent, want: videosession.ErrPractitionerDidNotJoin},
		{reason: videosession.EndWindowClosed, want: videosession.ErrConsultationEnded},
		{reason: videosession.EndTimeLimit, want: videosession.ErrConsultationEnded},
		{reason: "", want: videosession.ErrConsultationEnded}, // no session: nobody asked for a token in the window
	} {
		for _, c := range []appointment.Caller{me, pract} {
			f := newFixture(end.Add(31 * time.Minute))
			if tt.reason != "" {
				f.repo.sessions[apptA] = videosession.Session{
					ID: "5e000000-0000-4000-8000-000000000001", AppointmentID: apptA, OrganizationID: clinicA, ClosesAt: end.Add(30 * time.Minute),
					Room: "r_room", PatientIdentity: "p_patient", PractitionerIdentity: "p_practitioner",
					PatientJoinedAt: start, EndedAt: end.Add(30 * time.Minute), EndReason: tt.reason,
				}
			}
			if _, err := f.svc.Join(t.Context(), c, apptA); !errors.Is(err, tt.want) {
				t.Errorf("%q: Join() by %s = %v, want %v", tt.reason, c.AccountID, err, tt.want)
			}
			if len(f.repo.opened)+len(f.provider.rooms)+len(f.provider.grants) != 0 {
				t.Errorf("%q: opened %v, rooms %v, grants %v; want none", tt.reason, f.repo.opened, f.provider.rooms, f.provider.grants)
			}
			checkEvents(t, "the refused join", f.audit, joinEvent(c, "Appointment/"+apptA, auditevent.OutcomeMinorFailure))
		}
	}
}

// The appointment the clinic cancelled after the practitioner stayed away still tells the patient so.
func TestJoin_SaysThePractitionerDidNotJoinAlsoAfterACancellation(t *testing.T) {
	f := opened(end.Add(31*time.Minute), time.Time{}, start)
	s := f.session()
	s.EndedAt, s.EndReason = closes, videosession.EndPractitionerAbsent
	f.repo.sessions[apptA] = s
	f.change(func(a *appointment.Attendance) { a.Appointment.Status = appointment.StatusCancelled })
	if _, err := f.svc.Join(t.Context(), me, apptA); !errors.Is(err, videosession.ErrPractitionerDidNotJoin) {
		t.Errorf("Join() = %v, want ErrPractitionerDidNotJoin", err)
	}
}

func TestJoin_FailsWhenTheEndedSessionCannotBeRead(t *testing.T) {
	f := newFixture(end.Add(31 * time.Minute))
	f.repo.err = errBackend
	if _, err := f.svc.Join(t.Context(), me, apptA); !errors.Is(err, errBackend) {
		t.Errorf("Join() = %v, want the backend's error", err)
	}
	checkEvents(t, "the failed join", f.audit)
}
