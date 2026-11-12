// Package access holds the pure rules of who may see a patient's data (docs/booking/SPEC.md "Access rule"):
// no I/O, no imports from internal/. A service gathers the Facts (the consent's state, the patient's
// appointments in the clinic) in its transaction and asks Profile or Appointment for the level.
//
// Care records rest on Art. 9(2)(h) and sharing before care on consent (SPEC.md section 3): with an active
// consent the clinic sees the profile through an appointment; after a revocation it keeps the name, times and
// status of the appointments that started before the latest revocation, and nothing else, and their
// practitioner still records how they went (RecordsOutcome).
package access

import (
	"slices"
	"strconv"
	"time"
)

// Level is how much of a patient's data a viewer may see.
type Level int

// The levels.
const (
	// None: nothing; the resource's existence is not disclosed.
	None Level = iota
	// Name: the patient's name and, for an appointment, its times and status.
	Name
	// Full: the whole resource.
	Full
)

func (l Level) String() string {
	switch l {
	case None:
		return "None"
	case Name:
		return "Name"
	case Full:
		return "Full"
	}
	return "Level(" + strconv.Itoa(int(l)) + ")"
}

// Role is what the viewer is.
type Role int

// The roles; the zero Role is no one and sees nothing.
const (
	RolePatient Role = iota + 1
	RolePractitioner
	RoleAdmin
)

// Viewer is who asks.
type Viewer struct {
	Clinic           string // the clinic of staff
	PractitionerRole string // a practitioner's role in that clinic
	Patient          string // a patient's own patient id
	Role             Role
}

// Consent is the state of the patient's consents with one clinic.
type Consent struct {
	RevokedAt time.Time // the latest revocation of the care consent; zero when none was ever revoked
	// Shared is the encounters the history consent lists (share chosen visits from other clinics with this one,
	// ADR 0013); empty unless History.
	Shared        []string
	Active        bool // the care consent is active
	Transcription bool // the transcription consent (transcription, translation, AI assistance) is active
	History       bool // the history consent is active
}

// Visit is one of the patient's appointments in the clinic, of any status.
type Visit struct {
	Start            time.Time
	PractitionerRole string
}

// Facts are what is known about one patient in one clinic.
type Facts struct {
	Patient string
	Clinic  string  // the clinic the consent and the visits are about
	Visits  []Visit // the patient's appointments in Clinic
	Consent Consent
}

// Profile is what viewer may see of the patient's profile: the patient themselves everything; with an active
// consent, a practitioner with an appointment with the patient everything and the clinic's admin the name;
// without one, the practitioner of a past appointment and the admin of a clinic with a past appointment the
// name; everyone else nothing.
func Profile(v Viewer, f Facts) Level {
	if self(v, f) {
		return Full
	}
	if !staffOf(v, f) {
		return None
	}
	for _, a := range f.Visits {
		if v.Role == RolePractitioner && !ofPractitioner(v, a) {
			continue
		}
		switch {
		case f.Consent.Active && v.Role == RolePractitioner:
			return Full
		case past(f, a):
			return Name
		}
	}
	if f.Consent.Active && v.Role == RoleAdmin {
		return Name
	}
	return None
}

// Appointment is what viewer may see of the patient's appointment a: the patient everything; with an active
// consent, the appointment's practitioner and the clinic's admin everything (the patient's name per Profile);
// without one, the same viewers the name, times and status of a past appointment and nothing of any other;
// everyone else nothing.
func Appointment(v Viewer, f Facts, a Visit) Level {
	if self(v, f) {
		return Full
	}
	if !staffOf(v, f) || (v.Role == RolePractitioner && !ofPractitioner(v, a)) {
		return None
	}
	switch {
	case f.Consent.Active:
		return Full
	case past(f, a):
		return Name
	}
	return None
}

// RecordsOutcome reports whether viewer may record how appointment a went (fulfilled or noshow): its
// practitioner, with an active consent, or after a revocation for a past appointment, which is the care
// record (Art. 9(2)(h)). That a has started is the service's check; cancelling a past one stays refused.
func RecordsOutcome(v Viewer, f Facts, a Visit) bool {
	if v.Role != RolePractitioner || !staffOf(v, f) || !ofPractitioner(v, a) {
		return false
	}
	return f.Consent.Active || past(f, a)
}

// QuestionnaireResponse is what viewer may see of the patient's questionnaire response to appointment a
// (docs/questionnaire/SPEC.md §2 "Access rule"), submitted or not: the patient everything; with an active
// consent, the appointment's practitioner everything once submitted and the name (that it exists, its status)
// before, and the clinic's admin the name; without one, for a past appointment and a submitted response, the
// practitioner everything (D9, the care record) and the admin the name; everyone else nothing. Nobody but the
// patient and the appointment's practitioner is ever Full.
func QuestionnaireResponse(v Viewer, f Facts, a Visit, submitted bool) Level {
	if self(v, f) {
		return Full
	}
	if !staffOf(v, f) || (v.Role == RolePractitioner && !ofPractitioner(v, a)) {
		return None
	}
	switch {
	case !f.Consent.Active && (!past(f, a) || !submitted):
		return None
	case v.Role == RolePractitioner && submitted:
		return Full
	}
	return Name
}

// RequestsQuestionnaire reports whether viewer may ask the patient to fill in a questionnaire before
// appointment a: its practitioner, with an active consent; nobody else. That a is booked and has not started
// is the service's check.
func RequestsQuestionnaire(v Viewer, f Facts, a Visit) bool {
	return v.Role == RolePractitioner && staffOf(v, f) && ofPractitioner(v, a) && f.Consent.Active
}

// ConsultationRecord reports whether viewer may follow and read what was said in the consultation of
// appointment a: its live stream and its transcript (docs/plans/consult-intelligence.md "Access"). Its
// practitioner, with an active consent; nobody else, and nobody after a revocation, also for a consultation
// that started before it. The patient's own copies are SignedDocument's and TranscriptCopy's.
func ConsultationRecord(v Viewer, f Facts, a Visit) bool {
	return v.Role == RolePractitioner && staffOf(v, f) && ofPractitioner(v, a) && f.Consent.Active
}

// Transcribed reports whether the patient's consultations with the clinic may be transcribed and translated, with
// AI assistance for the practitioner (the transcription agent in the call, the stored transcript, suggestions,
// document drafts): only while both the care consent and the transcription consent are active (SAL-12,
// docs/plans/v1-finish.md "Decisions"). Declined or withdrawn, the consultation goes on without them.
func Transcribed(c Consent) bool {
	return c.Active && c.Transcription
}

// SignedDocument reports whether viewer may read a signed document of the consultation of appointment a (the
// final Composition, its DocumentReference and its PDF; docs/consultation/SPEC.md "Patient documents"): its
// practitioner under ConsultationRecord's rule, and, once signed, the patient themselves whatever the consent: it
// is their own record (GDPR Art. 15), which a revocation does not take from them. A draft (not signed) is only
// ConsultationRecord's.
func SignedDocument(v Viewer, f Facts, a Visit, signed bool) bool {
	return ConsultationRecord(v, f, a) || (signed && self(v, f))
}

// TranscriptCopy reports whether viewer may read the stored transcript of the consultation of appointment a: its
// practitioner under ConsultationRecord's rule, and the patient themselves while the clinic shares transcripts
// with its patients (shared, the clinic's setting), whatever the consent, as SignedDocument. The live stream
// stays ConsultationRecord's.
func TranscriptCopy(v Viewer, f Facts, a Visit, shared bool) bool {
	return ConsultationRecord(v, f, a) || (shared && self(v, f))
}

// self reports whether v is the patient the facts are about.
func self(v Viewer, f Facts) bool {
	return v.Role == RolePatient && v.Patient != "" && v.Patient == f.Patient
}

// staffOf reports whether v is staff of the facts' clinic.
func staffOf(v Viewer, f Facts) bool {
	return (v.Role == RolePractitioner || v.Role == RoleAdmin) && v.Clinic != "" && v.Clinic == f.Clinic
}

// ofPractitioner reports whether a is with the practitioner v.
func ofPractitioner(v Viewer, a Visit) bool {
	return v.PractitionerRole != "" && a.PractitionerRole == v.PractitionerRole
}

// past reports whether a started before the latest revocation, without an active consent.
func past(f Facts, a Visit) bool {
	return !f.Consent.Active && !f.Consent.RevokedAt.IsZero() && a.Start.Before(f.Consent.RevokedAt)
}

// OpensChat reports whether viewer may open a conversation between the facts' patient and the practitioner role
// in the facts' clinic (docs/plans/chat.md "Who opens a conversation"): the patient themselves or the practitioner
// of that role, with an active consent and an appointment of the patient with that role, of any time or status.
func OpensChat(v Viewer, f Facts, role string) bool {
	if !f.Consent.Active || role == "" || !slices.ContainsFunc(f.Visits, func(a Visit) bool { return a.PractitionerRole == role }) {
		return false
	}
	return self(v, f) || (v.Role == RolePractitioner && staffOf(v, f) && v.PractitionerRole == role)
}

// ReadsChat reports whether viewer may read a conversation about the facts' patient in the facts' clinic (plan,
// "Who reads and writes"): the patient themselves whatever the consent, as their own record (GDPR Art. 15); a
// practitioner of the clinic who takes part (participant) while the consent is active; nobody else, never an admin.
func ReadsChat(v Viewer, f Facts, participant bool) bool {
	return self(v, f) || (v.Role == RolePractitioner && staffOf(v, f) && participant && f.Consent.Active)
}

// WritesChat reports whether viewer may write in that conversation: as ReadsChat, and only while the consent is
// active, so nothing is sent that the clinic may not read.
func WritesChat(v Viewer, f Facts, participant bool) bool {
	return ReadsChat(v, f, participant) && f.Consent.Active
}
