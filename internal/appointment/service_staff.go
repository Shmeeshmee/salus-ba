package appointment

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/internal/patient"
	"github.com/Shmeeshmee/salus/internal/practitionerrole"
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
	ServiceName string // the booked service's, in the clinic's language; empty for none
	ReminderOff string // why no reminder will go: ReminderOffSMS or ReminderOffNoPhone; empty when one will
	Patient     patient.Patient
	Receipts    []VisitReceipt // the visit's receipts, whoever issued them (reversals left out), by number
	Appointment Appointment
	Reminder    bool // a reminder reaches the patient (docs/clinic-cms R3): an account, or SMS offered and a phone
}

// VisitReceipt is a receipt of a visit as the desk shows it: its number and id (for its PDF), and whether it was
// reversed. Nothing else of a receipt reaches the desk's day.
type VisitReceipt struct {
	ID       string
	Number   int64
	Reversed bool
}

// DeskReceipts gives the receipts of the clinic org's visits, by visit, in the transaction in ctx, unaudited
// (invoice.Service): the desk's day authorizes its caller and audits each receipt it shows.
type DeskReceipts interface {
	VisitReceipts(ctx context.Context, org string, appointmentIDs []string) (map[string][]VisitReceipt, error)
}

// Desk is the desk's day: its date (midnight in the clinic's zone) and its visits.
type Desk struct {
	Day    time.Time
	Visits []DeskVisit
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
// service b.ServiceID (empty for none, else an active service of the clinic's own price list offered in the slot's
// mode), and returns the appointment with c as who booked it. The desk books for any role, a practitioner for their
// own. A clinic whose bookings are stopped (a suspended or cancelled membership) books nothing; the slot is resolved
// in the clinic's own scope, listed or not, and must be of an active role; then, in one transaction in the
// clinic's scope, the patient is read, their active care consent checked, the slot held, the appointment stored,
// what follows a booking enqueued (for a patient without an account the statistics and the reminders, which reach
// them by SMS only: no e-mail or questionnaire request) and the booking audited. ErrForbidden to a patient and to a practitioner for another role's slot,
// ErrNotFound for a patient the clinic does not have, ErrSlotUnavailable for a slot it does not have, one that is
// not free and ahead or of an inactive role, and while its bookings are stopped, ErrInvalidService,
// ErrConsentRequired; each refusal is audited.
func (s *Service) BookForPatient(ctx context.Context, c Caller, b StaffBooking) (Appointment, error) {
	if c.patient() {
		return Appointment{}, s.refuse(ctx, c, auditevent.CodeCreate, ErrForbidden)
	}
	if s.Stops == nil {
		return Appointment{}, errors.New("appointment: book for patient: no booking stops")
	}
	if stopped, err := s.Stops.BookingsStopped(ctx, c.OrganizationID); err != nil {
		return Appointment{}, fmt.Errorf("appointment: book for patient: %w", err)
	} else if stopped {
		return Appointment{}, s.refuse(ctx, c, auditevent.CodeCreate, ErrSlotUnavailable)
	}
	now := s.Clock.Now()
	sl, err := s.Slots.BookableInClinic(ctx, c.OrganizationID, b.SlotID)
	switch {
	case errors.Is(err, slot.ErrUnavailable), err == nil && !sl.Start.After(now):
		return Appointment{}, s.refuse(ctx, c, auditevent.CodeCreate, ErrSlotUnavailable)
	case err != nil:
		return Appointment{}, fmt.Errorf("appointment: book for patient: %w", err)
	}
	if offered, err := s.offersInClinic(ctx, sl, b.ServiceID); err != nil {
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
	case errors.Is(err, ErrSlotUnavailable), errors.Is(err, ErrResourceBusy): // the insert aborted the transaction
		return Appointment{}, s.refuse(ctx, c, auditevent.CodeCreate, err)
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
	role, err := s.Roles.Lookup(ctx, c.OrganizationID, sl.PractitionerRoleID)
	if errors.Is(err, practitionerrole.ErrNotFound) || err == nil && !role.Active { // the directory hides such a slot
		return Appointment{}, ErrSlotUnavailable, nil
	} else if err != nil {
		return Appointment{}, nil, err
	}
	p, err := s.Patients.Lookup(ctx, b.PatientID)
	if errors.Is(err, patient.ErrNotFound) {
		return Appointment{}, ErrNotFound, nil
	} else if err != nil {
		return Appointment{}, nil, err
	}
	consent, err := s.Consents.State(ctx, p.ID, c.OrganizationID)
	switch {
	case err != nil || !consent.Active:
		return Appointment{}, ErrConsentRequired, err
	case consent.Restricted: // S7: no new use of a restricted record
		return Appointment{}, ErrNotFound, nil
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
	if p.AccountID == "" { // registered by the clinic: no e-mail, no questionnaire in the app; reminders by SMS (R3)
		return out, nil, s.bookedAtDesk(ctx, out, now)
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
		if err := s.recordChange(ctx, Change{
			At: out.ArrivedAt, AppointmentID: id, Kind: ChangeStatusSet, Via: ViaStaff, ByAccountID: c.AccountID,
			FromStatus: StatusBooked, ToStatus: StatusArrived,
		}); err != nil {
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

// DeskDay is the clinic's visits, but the cancelled ones, of the calendar day day (its date; zero for today in the
// clinic's zone) in the clinic's zone, by practitioner role and start, each with its patient and its booked service's
// name, and whether a reminder reaches the patient, for the desk (package F3's screen): to the clinic's admin and
// receptionists only (ErrForbidden, audited, to
// anyone else). One search of Appointment is audited, and each patient shown as a read of Patient.
func (s *Service) DeskDay(ctx context.Context, c Caller, day time.Time, locationID string) (Desk, error) {
	if !c.desk() {
		return Desk{}, s.refuse(ctx, c, auditevent.CodeSearchType, ErrForbidden)
	}
	var out Desk
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		clinic, err := s.Clinics.Read(ctx, organization.Caller{OrganizationID: c.OrganizationID}, c.OrganizationID)
		if err != nil {
			return err
		}
		loc, err := time.LoadLocation(clinic.TimeZone)
		if err != nil {
			return err
		}
		if day.IsZero() {
			day = s.Clock.Now().In(loc)
		}
		if err := s.Repo.SetClinic(ctx, c.OrganizationID); err != nil {
			return err
		}
		y, m, d := day.Date()
		out.Day = time.Date(y, m, d, 0, 0, 0, 0, loc)
		all, err := s.Repo.Desk(ctx, c.OrganizationID, out.Day, out.Day.AddDate(0, 0, 1))
		if err != nil {
			return err
		}
		if locationID != "" { // that location's in-person visits and every video visit (docs/clinic-cms/WAVE6.md U2)
			all = slices.DeleteFunc(all, func(a Appointment) bool { return a.LocationID != "" && a.LocationID != locationID })
		}
		if out.Visits, err = s.deskVisits(ctx, c, all); err != nil {
			return err
		}
		if err := s.withReceipts(ctx, c, out.Visits); err != nil {
			return err
		}
		if err := s.withReminders(ctx, c.OrganizationID, out.Visits); err != nil {
			return err
		}
		return s.record(ctx, c, auditevent.CodeSearchType, "", "", auditevent.OutcomeSuccess)
	})
	if err != nil {
		return Desk{}, fmt.Errorf("appointment: desk day: %w", err)
	}
	return out, nil
}

// deskVisits pairs each appointment with its patient, read once each and audited as a read of Patient, and its booked
// service's name; a patient the scope hides or whose record is restricted at the clinic is left out.
func (s *Service) deskVisits(ctx context.Context, c Caller, all []Appointment) ([]DeskVisit, error) {
	names, err := s.serviceNames(ctx, c.OrganizationID, all)
	if err != nil {
		return nil, err
	}
	patients, hidden := map[string]patient.Patient{}, map[string]bool{}
	out := make([]DeskVisit, 0, len(all))
	for _, a := range all {
		p, ok := patients[a.PatientID]
		if hidden[a.PatientID] {
			continue
		}
		if !ok {
			state, err := s.Consents.State(ctx, a.PatientID, c.OrganizationID)
			if err != nil {
				return nil, err
			}
			p, err = s.Patients.Lookup(ctx, a.PatientID)
			// The scope no longer shows them (a revocation), or their record is restricted at the clinic
			// (docs/clinic-cms/WAVE5.md S7): nothing of them reaches the desk.
			if errors.Is(err, patient.ErrNotFound) || (err == nil && state.Restricted) {
				hidden[a.PatientID] = true
				continue
			} else if err != nil {
				return nil, err
			}
			p.ConsentActive = state.Active
			patients[a.PatientID] = p
			if err := s.record(ctx, c, auditevent.CodeRead, "Patient", p.ID, auditevent.OutcomeSuccess); err != nil {
				return nil, err
			}
		}
		out = append(out, DeskVisit{Appointment: a, Patient: p, ServiceName: names[a.ServiceID]})
	}
	return out, nil
}

// withReceipts gives each visit its receipts, each audited as a read by c; nothing without a Receipts port.
func (s *Service) withReceipts(ctx context.Context, c Caller, visits []DeskVisit) error {
	if s.Receipts == nil || len(visits) == 0 {
		return nil
	}
	ids := make([]string, len(visits))
	for i, v := range visits {
		ids[i] = v.Appointment.ID
	}
	all, err := s.Receipts.VisitReceipts(ctx, c.OrganizationID, ids)
	if err != nil {
		return err
	}
	for i := range visits {
		visits[i].Receipts = all[visits[i].Appointment.ID]
		for _, r := range visits[i].Receipts {
			if err := s.record(ctx, c, auditevent.CodeRead, "Invoice", r.ID, auditevent.OutcomeSuccess); err != nil {
				return err
			}
		}
	}
	return nil
}

// serviceNames gives the names of the services the appointments booked, by id, from the clinic's own price list; none
// is read when no appointment booked one.
func (s *Service) serviceNames(ctx context.Context, org string, all []Appointment) (map[string]string, error) {
	if !slices.ContainsFunc(all, func(a Appointment) bool { return a.ServiceID != "" }) {
		return nil, nil
	}
	list, err := s.Services.OfClinic(ctx, org)
	if err != nil {
		return nil, err
	}
	names := make(map[string]string, len(list))
	for _, d := range list {
		names[d.ID] = d.Name
	}
	return names, nil
}
