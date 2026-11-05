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
