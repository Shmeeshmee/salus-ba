package appointment

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/internal/patient"
	"github.com/Shmeeshmee/salus/internal/schedule"
	"github.com/Shmeeshmee/salus/internal/slot"
)

// StaffBooking is a visit staff book for a patient of their clinic (package F2 of docs/clinic-cms): the patient,
// any free slot of the clinic, and optionally the service (empty for none).
type StaffBooking struct {
	PatientID string
	SlotID    string
	ServiceID string
}

// DeskVisit is one visit on the desk's day: the appointment and its patient (docs/clinic-cms W2: name, birth date
// and phone; nothing clinical).
type DeskVisit struct {
	Patient     patient.Patient
	Appointment Appointment
}

// desk reports whether c is the clinic's admin or receptionist, who book, move and cancel for any of its patients.
func (c Caller) desk() bool { return !c.patient() && (c.Admin || c.Reception) }

// asDesk is c with the admin's access to the clinic's appointments when c is at the desk: a receptionist books,
// cancels and checks in as the admin does (W2), and the audit names c's own account.
func (c Caller) asDesk() Caller {
	if c.desk() {
		c.Admin = true
	}
	return c
}

// actsFor reports whether staff caller c may book or move for the practitioner role roleID: the desk for every role
// of the clinic, a practitioner for their own role only.
func (s *Service) actsFor(ctx context.Context, c Caller, roleID string) (bool, error) {
	if c.desk() {
		return true, nil
	}
	v, err := s.staffViewer(ctx, c, roleID)
	return err == nil && v.PractitionerRole != "", err
}

// BookForPatient books the free slot b.SlotID of the staff caller c's clinic for the patient b.PatientID, naming the
// service b.ServiceID (empty for none), and returns the appointment with c as who booked it. The desk books for any
// role, a practitioner for their own. The slot is resolved as the directory shows it; then, in one transaction in the
// clinic's scope, the patient is read, their active care consent checked, the slot held, the appointment stored,
// what follows a booking enqueued (for a patient without an account only the statistics: they get no e-mail and no
// reminder) and the booking audited. ErrForbidden to a patient and to a practitioner for another role's slot,
// ErrNotFound for a patient or slot the clinic does not have, ErrSlotUnavailable for a slot that is not free and
// ahead, ErrInvalidService, ErrConsentRequired; each refusal is audited.
func (s *Service) BookForPatient(ctx context.Context, c Caller, b StaffBooking) (Appointment, error) {
	if c.patient() {
		return Appointment{}, s.refuse(ctx, c, auditevent.CodeCreate, ErrForbidden)
	}
	now := s.Clock.Now()
	sl, err := s.Slots.Bookable(ctx, b.SlotID)
	switch {
	case errors.Is(err, slot.ErrUnavailable), err == nil && !sl.Start.After(now):
		return Appointment{}, s.refuse(ctx, c, auditevent.CodeCreate, ErrSlotUnavailable)
	case err != nil:
		return Appointment{}, fmt.Errorf("appointment: book for patient: %w", err)
	case sl.OrganizationID != c.OrganizationID:
		return Appointment{}, s.refuse(ctx, c, auditevent.CodeCreate, ErrNotFound)
	}
	if offered, err := s.offers(ctx, sl, b.ServiceID); err != nil {
		return Appointment{}, fmt.Errorf("appointment: book for patient: %w", err)
	} else if !offered {
		return Appointment{}, s.refuse(ctx, c, auditevent.CodeCreate, ErrInvalidService)
	}
	var out Appointment
	var refused error
	err = s.Tx.WithTx(ctx, func(ctx context.Context) error {
		var err error
		if out, refused, err = s.bookFor(ctx, c, b, sl, now); err != nil || refused != nil {
			return err
		}
		return s.record(ctx, c, auditevent.CodeCreate, "Appointment", out.ID, auditevent.OutcomeSuccess)
	})
	switch {
	case errors.Is(err, ErrSlotUnavailable): // the insert aborted the transaction
		return Appointment{}, s.refuse(ctx, c, auditevent.CodeCreate, ErrSlotUnavailable)
	case err != nil:
		return Appointment{}, fmt.Errorf("appointment: book for patient: %w", err)
	case refused != nil:
		return Appointment{}, s.refuse(ctx, c, auditevent.CodeCreate, refused)
	}
	return out, nil
}

// bookFor checks and books the slot sl for b in the transaction in ctx, or gives the refusal.
func (s *Service) bookFor(ctx context.Context, c Caller, b StaffBooking, sl slot.Bookable, now time.Time) (out Appointment, refused, err error) {
	if err := s.Repo.SetClinic(ctx, c.OrganizationID); err != nil {
		return Appointment{}, nil, err
	}
	if ok, err := s.actsFor(ctx, c, sl.PractitionerRoleID); err != nil || !ok {
		return Appointment{}, ErrForbidden, err
	}
	p, err := s.Patients.Lookup(ctx, b.PatientID)
	if errors.Is(err, patient.ErrNotFound) {
		return Appointment{}, ErrNotFound, nil
	} else if err != nil {
		return Appointment{}, nil, err
	}
	consent, err := s.Consents.State(ctx, p.ID, c.OrganizationID)
	if err != nil || !consent.Active {
		return Appointment{}, ErrConsentRequired, err
	}
	held, err := s.Slots.Hold(ctx, c.OrganizationID, sl.ID)
	if errors.Is(err, slot.ErrUnavailable) {
		return Appointment{}, ErrSlotUnavailable, nil
	} else if err != nil {
		return Appointment{}, nil, err
	}
	out, err = s.Repo.Create(ctx, Appointment{
		PatientID: p.ID, PractitionerRoleID: sl.PractitionerRoleID, OrganizationID: c.OrganizationID, SlotID: sl.ID,
		Start: sl.Start, End: sl.End, Created: now, BookedVia: BookedDirect, Mode: held.Mode, ServiceID: b.ServiceID, BookedBy: c.AccountID,
	})
	if err != nil {
		return Appointment{}, nil, err
	}
	if p.AccountID == "" { // registered by the clinic: no e-mail, no questionnaire in the app, no reminder
		return out, nil, s.Stats.EnqueueStatsRefresh(ctx, out.OrganizationID, out.Start)
	}
	return out, nil, s.booked(ctx, out, now)
}

// CancelForPatient cancels the booked appointment id at version for its patient as the staff caller c (reason prov),
// as Cancel does for the clinic: the desk any of the clinic's, a practitioner their own. ErrForbidden to a patient,
// ErrVersionConflict when version is not the current one (nothing changes), and Cancel's refusals; each is audited.
func (s *Service) CancelForPatient(ctx context.Context, c Caller, id string, version int32) (Appointment, error) {
	if c.patient() {
		return Appointment{}, s.refuse(ctx, c, auditevent.CodeUpdate, ErrForbidden)
	}
	c = c.asDesk()
	var out Appointment
	var refused error
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		var err error
		if out, refused, err = s.cancel(ctx, c, id); err != nil || refused != nil {
			return err
		}
		if out.VersionID != version+1 { // cancelled from another version: rolled back
			return ErrVersionConflict
		}
		return s.record(ctx, c, auditevent.CodeUpdate, "Appointment", out.ID, auditevent.OutcomeSuccess)
	})
	switch {
	case errors.Is(err, ErrVersionConflict):
		return Appointment{}, s.refuse(ctx, c, auditevent.CodeUpdate, ErrVersionConflict)
	case err != nil:
		return Appointment{}, fmt.Errorf("appointment: cancel for patient: %w", err)
	case refused != nil:
		return Appointment{}, s.refuse(ctx, c, auditevent.CodeUpdate, refused)
	}
	return out, nil
}

// MoveForPatient moves the appointment id to the slot slotID as the staff caller c. Not built yet (package F2's
// DEFERRED in docs/clinic-cms/PROGRESS.md): a staff move must not count against the patient's own reschedules nor
// confirm the visit in the app, which the patient's Reschedule does.
func (s *Service) MoveForPatient(ctx context.Context, c Caller, id, slotID string) (Appointment, error) {
	return Appointment{}, errors.ErrUnsupported
}

// Arrive marks the booked in-person appointment id arrived at the desk (StatusArrived, W6) as the clinic's admin or
// receptionist c, in the clinic's scope, and returns it; audited, and the clinic's statistics refreshed. ErrForbidden
// to anyone else, ErrNotFound for an appointment the clinic does not see, ErrNotInPerson for a video visit, ErrClosed
// for one not booked any more; each refusal is audited.
func (s *Service) Arrive(ctx context.Context, c Caller, id string) (Appointment, error) {
	if !c.desk() {
		return Appointment{}, s.refuse(ctx, c, auditevent.CodeUpdate, ErrForbidden)
	}
	var out Appointment
	var refused error
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.Repo.SetClinic(ctx, c.OrganizationID); err != nil {
			return err
		}
		a, err := s.Repo.ByID(ctx, id)
		switch {
		case errors.Is(err, ErrNotFound), err == nil && a.OrganizationID != c.OrganizationID:
			refused = ErrNotFound
			return nil
		case err != nil:
			return err
		case visitMode(a.Mode) != schedule.ModeInPerson:
			refused = ErrNotInPerson
			return nil
		}
		if out, err = s.Repo.Arrive(ctx, id, s.Clock.Now()); errors.Is(err, ErrClosed) {
			refused = ErrClosed
			return nil
		} else if err != nil {
			return err
		}
		if err := s.Stats.EnqueueStatsRefresh(ctx, out.OrganizationID, out.Start); err != nil {
			return err
		}
		return s.record(ctx, c, auditevent.CodeUpdate, "Appointment", out.ID, auditevent.OutcomeSuccess)
	})
	switch {
	case err != nil:
		return Appointment{}, fmt.Errorf("appointment: arrive: %w", err)
	case refused != nil:
		return Appointment{}, s.refuse(ctx, c, auditevent.CodeUpdate, refused)
	}
	return out, nil
}

// DeskDay is the clinic's visits, but the cancelled ones, of the calendar day day (its date) in the clinic's zone, by
// practitioner role and start, each with its patient, for the desk (package F3's screen): to the clinic's admin and
// receptionists only (ErrForbidden, audited, to anyone else). One search of Appointment is audited, and each patient
// shown as a read of Patient.
func (s *Service) DeskDay(ctx context.Context, c Caller, day time.Time) ([]DeskVisit, error) {
	if !c.desk() {
		return nil, s.refuse(ctx, c, auditevent.CodeSearchType, ErrForbidden)
	}
	var out []DeskVisit
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		clinic, err := s.Clinics.Read(ctx, organization.Caller{OrganizationID: c.OrganizationID}, c.OrganizationID)
		if err != nil {
			return err
		}
		loc, err := time.LoadLocation(clinic.TimeZone)
		if err != nil {
			return err
		}
		if err := s.Repo.SetClinic(ctx, c.OrganizationID); err != nil {
			return err
		}
		y, m, d := day.Date()
		from := time.Date(y, m, d, 0, 0, 0, 0, loc)
		all, err := s.Repo.Desk(ctx, c.OrganizationID, from, from.AddDate(0, 0, 1))
		if err != nil {
			return err
		}
		if out, err = s.deskVisits(ctx, c, all); err != nil {
			return err
		}
		return s.record(ctx, c, auditevent.CodeSearchType, "", "", auditevent.OutcomeSuccess)
	})
	if err != nil {
		return nil, fmt.Errorf("appointment: desk day: %w", err)
	}
	return out, nil
}

// deskVisits pairs each appointment with its patient, read once each and audited as a read of Patient.
func (s *Service) deskVisits(ctx context.Context, c Caller, all []Appointment) ([]DeskVisit, error) {
	patients := map[string]patient.Patient{}
	out := make([]DeskVisit, 0, len(all))
	for _, a := range all {
		p, ok := patients[a.PatientID]
		if !ok {
			var err error
			if p, err = s.Patients.Lookup(ctx, a.PatientID); err != nil {
				return nil, err
			}
			patients[a.PatientID] = p
			if err := s.record(ctx, c, auditevent.CodeRead, "Patient", p.ID, auditevent.OutcomeSuccess); err != nil {
				return nil, err
			}
		}
		out = append(out, DeskVisit{Appointment: a, Patient: p})
	}
	return out, nil
}
