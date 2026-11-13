package appointment

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/internal/slot"
	"github.com/Shmeeshmee/salus/pkg/payments"
)

// Accept books the slot of the calling patient's open offer id and returns the appointment (booked via the waitlist).
// The offer is read first and paid for before the booking's transaction (purchase, with the offer's id as the key;
// refunded when the booking then fails, unless the offer was accepted meanwhile: the key gives a concurrent acceptance
// the same receipt, ADR 0015). In one transaction in the patient's scope that locks the entry, then the offer (the
// order a withdrawal takes; the expiry locks the offer too): the offer is open and its hold not over, the clinic
// listed, the care consent active; the slot moves from offered to busy, the appointment is stored, the offer accepted
// and its entry booked, the booking's e-mail, questionnaire request, reminders and statistics enqueued, and the booking
// audited. An offer the patient accepted already (another device) gives that appointment again, read and audited as a
// read. ErrForbidden to staff, ErrProfileRequired without a profile, ErrNotFound for an offer not theirs,
// ErrOfferExpired, ErrConsentRequired, ErrSlotUnavailable, ErrPaymentDeclined; each refusal is audited.
func (s *Service) Accept(ctx context.Context, c Caller, id string) (Appointment, error) {
	switch {
	case !c.patient():
		return Appointment{}, s.refuse(ctx, c, auditevent.CodeCreate, ErrForbidden)
	case c.PatientID == "":
		return Appointment{}, s.refuse(ctx, c, auditevent.CodeCreate, ErrProfileRequired)
	}
	o, err := s.ownOffer(ctx, c, id)
	switch {
	case errors.Is(err, ErrNotFound):
		return Appointment{}, s.refuse(ctx, c, auditevent.CodeCreate, ErrNotFound)
	case err != nil:
		return Appointment{}, fmt.Errorf("appointment: accept an offer: %w", err)
	case o.State == OfferAccepted:
		return s.acceptOffer(ctx, c, o)
	case o.State != OfferOpen || !s.Clock.Now().Before(o.ExpiresAt):
		return Appointment{}, s.refuse(ctx, c, auditevent.CodeCreate, ErrOfferExpired)
	}
	b := slot.Bookable{
		Slot:               slot.Slot{ID: o.SlotID, OrganizationID: o.OrganizationID, Status: slot.StatusTentative, Period: slot.Period{Start: o.Start, End: o.End}},
		PractitionerRoleID: o.PractitionerRoleID,
	}
	receipt, err := s.purchase(ctx, c, b, "waitlist:"+o.ID)
	switch {
	case errors.Is(err, payments.ErrDeclined):
		return Appointment{}, s.refuse(ctx, c, auditevent.CodeCreate, ErrPaymentDeclined)
	case err != nil:
		return Appointment{}, fmt.Errorf("appointment: accept an offer: %w", err)
	}
	out, err := s.acceptOffer(ctx, c, o)
	if err != nil {
		// Not cut short by the request's end: a charge must not stay without its booking, and a booking made with the
		// same receipt by another acceptance keeps it.
		ctx := context.WithoutCancel(ctx)
		if now, rerr := s.ownOffer(ctx, c, id); rerr == nil && now.State == OfferAccepted {
			return Appointment{}, err
		}
		if rerr := s.refund(ctx, receipt); rerr != nil {
			return Appointment{}, errors.Join(err, fmt.Errorf("appointment: accept an offer: refund: %w", rerr))
		}
		return Appointment{}, err
	}
	return out, nil
}

// ownOffer reads the calling patient's offer id in their scope, in a transaction of its own; ErrNotFound for another's.
func (s *Service) ownOffer(ctx context.Context, c Caller, id string) (Offer, error) {
	var o Offer
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.Repo.SetPatient(ctx, c.PatientID); err != nil {
			return err
		}
		var err error
		o, err = s.Waitlist.Offer(ctx, id)
		if err == nil && o.PatientID != c.PatientID { // not on row-level security alone
			return ErrNotFound
		}
		return err
	})
	return o, err
}

// acceptOffer books the offer pre in one transaction, audited; each refusal is audited in a transaction of its own.
func (s *Service) acceptOffer(ctx context.Context, c Caller, pre Offer) (Appointment, error) {
	var out Appointment
	var refused error
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		var err error
		var again bool
		if out, again, refused, err = s.accept(ctx, c, pre); err != nil || refused != nil {
			return err
		}
		if again {
			return s.record(ctx, c, auditevent.CodeRead, "Appointment", out.ID, auditevent.OutcomeSuccess)
		}
		return s.record(ctx, c, auditevent.CodeCreate, "Appointment", out.ID, auditevent.OutcomeSuccess)
	})
	switch {
	case errors.Is(err, ErrSlotUnavailable):
		// The insert aborted the transaction: the refusal is audited in a transaction of its own.
		return Appointment{}, s.refuse(ctx, c, auditevent.CodeCreate, ErrSlotUnavailable)
	case err != nil:
		return Appointment{}, fmt.Errorf("appointment: accept an offer: %w", err)
	case refused != nil:
		return Appointment{}, s.refuse(ctx, c, auditevent.CodeCreate, refused)
	}
	return out, nil
}

// accept checks and books the offer pre in the transaction in ctx, or gives the refusal; again for an offer the
// patient accepted already, whose appointment it gives.
func (s *Service) accept(ctx context.Context, c Caller, pre Offer) (out Appointment, again bool, refused, err error) {
	org := pre.OrganizationID
	if err := s.Repo.SetPatient(ctx, c.PatientID); err != nil {
		return Appointment{}, false, nil, err
	}
	if err := s.Repo.SetClinic(ctx, org); err != nil {
		return Appointment{}, false, nil, err
	}
	if _, err := s.Waitlist.EntryForUpdate(ctx, pre.EntryID); errors.Is(err, ErrNotFound) {
		return Appointment{}, false, ErrNotFound, nil
	} else if err != nil {
		return Appointment{}, false, nil, err
	}
	o, err := s.Waitlist.OfferForUpdate(ctx, pre.ID)
	now := s.Clock.Now()
	switch {
	case errors.Is(err, ErrNotFound):
		return Appointment{}, false, ErrNotFound, nil
	case err != nil:
		return Appointment{}, false, nil, err
	case o.PatientID != c.PatientID || o.OrganizationID != org || o.EntryID != pre.EntryID:
		return Appointment{}, false, ErrNotFound, nil
	case o.State == OfferAccepted: // another device of the patient's accepted it first
		out, err := s.Repo.ByID(ctx, o.AppointmentID)
		return out, true, nil, err
	case o.State != OfferOpen || !now.Before(o.ExpiresAt): // an expiry or a withdrawal committed first
		return Appointment{}, false, ErrOfferExpired, nil
	}
	out, refused, err = s.bookOffer(ctx, c, o, now)
	return out, false, refused, err
}

// bookOffer books the locked open offer o for c at now in the transaction in ctx, or gives the refusal.
func (s *Service) bookOffer(ctx context.Context, c Caller, o Offer, now time.Time) (out Appointment, refused, err error) {
	org := o.OrganizationID
	if refused, err := s.mayBook(ctx, c.PatientID, org); refused != nil || err != nil {
		return Appointment{}, refused, err
	}
	if _, err := s.Offered.TakeOffered(ctx, org, o.SlotID); errors.Is(err, slot.ErrUnavailable) {
		return Appointment{}, ErrSlotUnavailable, nil
	} else if err != nil {
		return Appointment{}, nil, err
	}
	out, err = s.Repo.Create(ctx, Appointment{
		PatientID: c.PatientID, PractitionerRoleID: o.PractitionerRoleID, OrganizationID: org, SlotID: o.SlotID,
		Start: o.Start, End: o.End, Created: now, BookedVia: BookedWaitlist,
	})
	if err != nil {
		return Appointment{}, nil, err
	}
	if _, err := s.Waitlist.CloseOffer(ctx, o.ID, OfferAccepted, out.ID, now); err != nil {
		return Appointment{}, nil, err
	}
	if _, err := s.Waitlist.CloseEntry(ctx, o.EntryID, WaitlistBooked, now); err != nil {
		return Appointment{}, nil, err
	}
	return out, nil, s.booked(ctx, out, now)
}

// mayBook refuses a booking with the clinic org that is not listed (ErrSlotUnavailable) or without the patient's
// active care consent there (ErrConsentRequired), in the transaction in ctx.
func (s *Service) mayBook(ctx context.Context, patientID, org string) (refused, err error) {
	clinic, err := s.Clinics.Read(ctx, organization.Caller{OrganizationID: org}, org)
	if err != nil {
		return nil, err
	}
	if !clinic.Listed {
		return ErrSlotUnavailable, nil
	}
	consent, err := s.Consents.State(ctx, patientID, org)
	if err != nil {
		return nil, err
	}
	if !consent.Active {
		return ErrConsentRequired, nil
	}
	return nil, nil
}
