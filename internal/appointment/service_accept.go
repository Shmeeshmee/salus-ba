package appointment

import (
	"context"
	"errors"
	"fmt"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/internal/slot"
	"github.com/Shmeeshmee/salus/pkg/payments"
)

// Accept books the slot of the calling patient's open offer id and returns the appointment (booked via the waitlist).
// The offer is read first and paid for before the booking's transaction (purchase, with the offer's id as the key;
// refunded when the booking then fails, ADR 0015). In one transaction in the patient's scope that locks the offer, as
// its expiry does: the offer is open and its hold not over, the clinic listed, the care consent active; the slot moves
// from offered to busy, the appointment is stored, the offer accepted and its entry booked, the booking's e-mail,
// questionnaire request, reminders and statistics enqueued, and the booking audited. ErrForbidden to staff,
// ErrProfileRequired without a profile, ErrNotFound for an offer not theirs, ErrOfferExpired, ErrConsentRequired,
// ErrSlotUnavailable, ErrPaymentDeclined; each refusal is audited.
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
		// Not cut short by the request's end: a charge must not stay without its booking.
		if rerr := s.refund(context.WithoutCancel(ctx), receipt); rerr != nil {
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
		if out, refused, err = s.accept(ctx, c, pre); err != nil || refused != nil {
			return err
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

// accept checks and books the offer pre in the transaction in ctx, or gives the refusal.
func (s *Service) accept(ctx context.Context, c Caller, pre Offer) (out Appointment, refused, err error) {
	org := pre.OrganizationID
	if err := s.Repo.SetPatient(ctx, c.PatientID); err != nil {
		return Appointment{}, nil, err
	}
	if err := s.Repo.SetClinic(ctx, org); err != nil {
		return Appointment{}, nil, err
	}
	o, err := s.Waitlist.OfferForUpdate(ctx, pre.ID)
	now := s.Clock.Now()
	switch {
	case errors.Is(err, ErrNotFound):
		return Appointment{}, ErrNotFound, nil
	case err != nil:
		return Appointment{}, nil, err
	case o.PatientID != c.PatientID || o.OrganizationID != org:
		return Appointment{}, ErrNotFound, nil
	case o.State != OfferOpen || !now.Before(o.ExpiresAt): // an expiry or another acceptance committed first
		return Appointment{}, ErrOfferExpired, nil
	}
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
