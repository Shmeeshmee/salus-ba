// Package videosession is the video room of one appointment's consultation (docs/consultation/SPEC.md §2):
// a LiveKit room that only the appointment's patient and its practitioner may join, created on the first
// token request. Rooms and participant identities are random ids, so LiveKit learns nothing about who is in
// the call; the clients get the other person's name from our API. LiveKit sits behind the Provider port
// (repository_livekit.go), and its types never leave that file. Every token issued or refused is audited.
package videosession

import "time"

// Role is what the caller is in the consultation.
type Role string

// The roles.
const (
	RolePatient      Role = "patient"
	RolePractitioner Role = "practitioner"
)

// EndReason is why a session ended.
type EndReason string

// The reasons (SPEC.md §2 "Ending").
const (
	EndPractitioner       EndReason = "practitioner"        // the practitioner ended the consultation
	EndWindowClosed       EndReason = "window_closed"       // the window closed with the room empty
	EndPractitionerAbsent EndReason = "practitioner_absent" // the window closed and the practitioner never joined
	EndTimeLimit          EndReason = "time_limit"          // 2 hours after the scheduled end
	EndCancelled          EndReason = "cancelled"           // the appointment was cancelled
)

// Reason is why the session ended or its patient was removed, as the clients show it (SPEC.md §2
// "Consultation status", changed after package 6).
type Reason string

// The reasons. A removal records consent_withdrawn or cancelled on the session (RemovalReason).
const (
	ReasonConsentWithdrawn       Reason = "consent_withdrawn"         // the patient was removed: their consent was withdrawn
	ReasonCancelled              Reason = "cancelled"                 // the patient was removed, or it ended: the appointment was cancelled
	ReasonLeft                   Reason = "left"                      // open, and the patient who joined was not removed: they left on their own
	ReasonEnded                  Reason = "ended"                     // the practitioner ended it, or the window closed with the room empty
	ReasonPractitionerDidNotJoin Reason = "practitioner_did_not_join" // the window closed without the practitioner
	ReasonTimeLimit              Reason = "time_limit"                // 2 hours after the scheduled end
)

// Session is the room of one appointment. Room, PatientIdentity and PractitionerIdentity are random and
// opaque, never logged; the identity of each role is fixed for the session, so a reconnect from a second
// device replaces the first connection.
type Session struct {
	CreatedAt            time.Time
	ClosesAt             time.Time // the join window's close: the appointment's end + 30 minutes
	PatientJoinedAt      time.Time // zero until the patient joined
	PractitionerJoinedAt time.Time // zero until the practitioner joined
	EndedAt              time.Time // zero while not ended
	ID                   string
	AppointmentID        string
	OrganizationID       string
	Room                 string
	PatientIdentity      string
	PractitionerIdentity string
	EndReason            EndReason // empty while not ended
	RemovalReason        Reason    // why the patient was removed: consent_withdrawn or cancelled; empty until then
}

// Room is a room to create: no metadata, at most MaxParticipants, kept by LiveKit for EmptyTimeout while
// nobody is in it (or after the last one left).
type Room struct {
	Name            string
	EmptyTimeout    time.Duration
	MaxParticipants uint32
}

// Grant is what a join token allows: joining the one room as the identity, for ValidFor. Nothing else is
// granted (the provider: camera and microphone, subscribing, no data, no name, no metadata).
type Grant struct {
	Room     string
	Identity string
	ValidFor time.Duration
}

// Join is what a participant needs to join: the LiveKit URL for clients, the token and its expiry, the other
// person's display name and the caller's role.
type Join struct {
	ExpiresAt time.Time
	URL       string
	Token     string
	PeerName  string
	Role      Role
}

// State is where an appointment's consultation stands (SPEC.md §2 "Consultation status").
type State string

// The states.
const (
	StateNotStarted State = "not_started" // nobody has asked for a token yet
	StateOpen       State = "open"        // the session exists and has not ended
	StateEnded      State = "ended"
)

// Status is an appointment's consultation as its participants and its clinic see it: the state, when each
// side first joined (zero for never), and when and why it ended. No room or identity.
type Status struct {
	PatientJoinedAt      time.Time
	PractitionerJoinedAt time.Time
	EndedAt              time.Time
	State                State
	EndReason            EndReason
	Reason               Reason // why it ended or the patient was removed; empty when neither
}

// status is the session's Status.
func (s Session) status() Status {
	state := StateOpen
	if !s.EndedAt.IsZero() {
		state = StateEnded
	}
	return Status{
		State: state, PatientJoinedAt: s.PatientJoinedAt, PractitionerJoinedAt: s.PractitionerJoinedAt,
		EndedAt: s.EndedAt, EndReason: s.EndReason, Reason: s.reason(),
	}
}

// reasonOfEnd is the Reason of each EndReason.
var reasonOfEnd = map[EndReason]Reason{
	EndPractitioner: ReasonEnded, EndWindowClosed: ReasonEnded, EndPractitionerAbsent: ReasonPractitionerDidNotJoin,
	EndTimeLimit: ReasonTimeLimit, EndCancelled: ReasonCancelled,
}

// reason is why the session ended or its patient was removed: the removal's reason first (a revocation before
// the start also ends the session cancelled), then the end's; while open, left once the patient joined.
func (s Session) reason() Reason {
	switch {
	case s.RemovalReason != "":
		return s.RemovalReason
	case !s.EndedAt.IsZero():
		return reasonOfEnd[s.EndReason]
	case !s.PatientJoinedAt.IsZero():
		return ReasonLeft
	}
	return ""
}

// refusal is why the ended session refuses a join: the practitioner did not join when it ended with them
// absent, else the consultation ended.
func (s Session) refusal() error {
	if s.EndReason == EndPractitionerAbsent {
		return ErrPractitionerDidNotJoin
	}
	return ErrConsultationEnded
}

// EventKind is what a LiveKit webhook event reports, as the consultation tells them apart.
type EventKind string

// The kinds (SPEC.md §2 "Idempotent, order-free handling"); every other kind of LiveKit's is EventOther.
const (
	EventParticipantJoined EventKind = "participant_joined"
	EventParticipantLeft   EventKind = "participant_left"
	EventRoomFinished      EventKind = "room_finished"
	EventOther             EventKind = "other"
)

// Event is a verified LiveKit webhook event: its kind, the room and the participant's identity (both random
// and opaque, never logged), and when the participant joined for a join, else when the event happened.
type Event struct {
	At       time.Time
	Kind     EventKind
	Room     string
	Identity string
}

// WindowCloseArgs is the consultation_window_close job on the default queue, scheduled at the join window's
// close (SPEC.md §2 "The window closes"). Opaque ids only.
type WindowCloseArgs struct {
	SessionID      string `json:"session_id"`
	OrganizationID string `json:"organization_id"`
}

// Kind names the job in river_job.
func (WindowCloseArgs) Kind() string { return "consultation_window_close" }
