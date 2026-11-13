package appointment

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/auditevent"
)

// Attendance is an appointment with what the caller is to it, for the consultation (docs/consultation/SPEC.md
// §2 "Who may join"), which decides and audits the access itself.
type Attendance struct {
	RevokedAt      time.Time // the latest revocation of the patient's consent with the clinic; zero when none
	PractitionerID string    // the practitioner of the appointment's role
	Appointment    Appointment
	Level          access.Level // what the caller may see of the appointment (access.Appointment)
	Profile        access.Level // what the caller may see of its patient (access.Profile)
	ConsentActive  bool         // the patient's consent with the clinic is active
	// TranscriptionConsent: the consultation may be transcribed and translated, with AI assistance
	// (access.Transcribed: the care and the transcription consent active, SAL-12)
	TranscriptionConsent bool
	Patient              bool // the caller is the appointment's patient
	Practitioner         bool // the caller is the practitioner of the appointment's role
}

// Attendance returns the appointment id with the caller's access facts, read in the caller's scope (the
// patient's own or the clinic's, set here) in the transaction in ctx, or in one of its own. It writes no
// audit event: the caller's service audits its decision. ErrNotFound when the caller may not see it.
func (s *Service) Attendance(ctx context.Context, c Caller, id string) (Attendance, error) {
	var out Attendance
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		v, err := s.view(ctx, c, id)
		if err != nil || v.level == access.None {
			return err
		}
		if c.patient() { // view gathers a patient's facts without the consent: their own access needs none
			if v.facts.Consent, err = s.Consents.State(ctx, c.PatientID, v.a.OrganizationID); err != nil {
				return err
			}
		}
		practitionerID, err := s.practitionerOf(ctx, map[string]string{}, v.a)
		if err != nil {
			return err
		}
		out = Attendance{
			Appointment: v.a, PractitionerID: practitionerID, Level: v.level, Profile: access.Profile(v.viewer, v.facts),
			ConsentActive: v.facts.Consent.Active, RevokedAt: v.facts.Consent.RevokedAt,
			TranscriptionConsent: access.Transcribed(v.facts.Consent),
			Patient:              c.patient(),
			Practitioner:         !c.patient() && v.viewer.PractitionerRole != "" && v.viewer.PractitionerRole == v.a.PractitionerRoleID,
		}
		return nil
	})
	switch {
	case err != nil:
		return Attendance{}, fmt.Errorf("appointment: attendance: %w", err)
	case out.Appointment.ID == "":
		return Attendance{}, ErrNotFound
	}
	return out, nil
}

// Lookup returns the appointment id in the caller's transaction and scope, unaudited, for the consultation's
// lifecycle (the encounter, which audits its own write). ErrNotFound when the scope does not show it.
func (s *Service) Lookup(ctx context.Context, id string) (Appointment, error) {
	a, err := s.Repo.ByID(ctx, id)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return Appointment{}, fmt.Errorf("appointment: lookup: %w", err)
	}
	return a, err
}

// LookupLocked is Lookup, locking the appointment against a change until the caller's transaction ends: a
// cancellation waits for it, and one that committed first is seen (docs/questionnaire/SPEC.md §2 "Requests").
// For a questionnaire request or answer, which audits its own write. The scope must also let it change the
// appointment: the patient's own, or the clinic's with the patient's active consent; ErrNotFound otherwise.
func (s *Service) LookupLocked(ctx context.Context, id string) (Appointment, error) {
	a, err := s.Repo.ByIDForShare(ctx, id)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return Appointment{}, fmt.Errorf("appointment: lookup locked: %w", err)
	}
	return a, err
}

// LookupPractitioner returns the practitioner of the appointment id's role, read in the caller's transaction
// and scope, unaudited, for the consultation's room rule (docs/consultation/SPEC.md §2 "Consultation status"),
// which decides and audits the read itself. ErrNotFound when the scope does not show the appointment.
func (s *Service) LookupPractitioner(ctx context.Context, id string) (string, error) {
	a, err := s.Lookup(ctx, id)
	if err != nil {
		return "", err
	}
	practitionerID, err := s.practitionerOf(ctx, map[string]string{}, a)
	if err != nil {
		return "", fmt.Errorf("appointment: lookup practitioner: %w", err)
	}
	return practitionerID, nil
}

// RecordConsultationOutcome records the end of the appointment's consultation (docs/consultation/SPEC.md §2
// "Outcome"), a system method of the agent in the caller's transaction and scope: a booked appointment
// becomes fulfilled when the patient joined, else noshow, audited as an update by the agent and kept in its history. Any other status
// is left as it is (a practitioner who marked it by hand wins, a cancellation stays), and nothing is audited.
func (s *Service) RecordConsultationOutcome(ctx context.Context, agent, id string, patientJoined bool) error {
	status := StatusNoShow
	if patientJoined {
		status = StatusFulfilled
	}
	return s.Tx.WithTx(ctx, func(ctx context.Context) error {
		now := s.Clock.Now()
		a, err := s.Repo.SetStatus(ctx, id, status, now)
		switch {
		case errors.Is(err, ErrClosed):
			return nil
		case err != nil:
			return fmt.Errorf("appointment: record consultation outcome: %w", err)
		}
		if err := s.recordChange(ctx, Change{
			At: now, AppointmentID: a.ID, Kind: ChangeStatusSet, Via: ViaSystem, Agent: agent, FromStatus: StatusBooked, ToStatus: status,
		}); err != nil {
			return fmt.Errorf("appointment: record consultation outcome: %w", err)
		}
		return s.recordJob(ctx, agent, a.OrganizationID, auditevent.CodeUpdate, "Appointment", a.ID, auditevent.OutcomeSuccess)
	})
}
