package appointment

import (
	"context"
	"fmt"

	"github.com/Shmeeshmee/salus/internal/access"
)

// Attendance is an appointment with what the caller is to it, for the consultation (docs/consultation/SPEC.md
// §2 "Who may join"), which decides and audits the access itself.
type Attendance struct {
	PractitionerID string // the practitioner of the appointment's role
	Appointment    Appointment
	Level          access.Level // what the caller may see of the appointment (access.Appointment)
	Profile        access.Level // what the caller may see of its patient (access.Profile)
	ConsentActive  bool         // the patient's consent with the clinic is active
	Patient        bool         // the caller is the appointment's patient
	Practitioner   bool         // the caller is the practitioner of the appointment's role
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
			ConsentActive: v.facts.Consent.Active,
			Patient:       c.patient(),
			Practitioner:  !c.patient() && v.viewer.PractitionerRole != "" && v.viewer.PractitionerRole == v.a.PractitionerRoleID,
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
