package appointment

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/internal/patient"
	"github.com/Shmeeshmee/salus/internal/practitionerrole"
	"github.com/Shmeeshmee/salus/internal/slot"
)

// The offer's rules (docs/plans/appointments.md "Waitlist matching", "Hold").
const (
	// offerLead: a slot is offered only when it starts at least the clinic's hold plus this ahead.
	offerLead = 60 * time.Minute
	// candidatesPerOffer bounds the entries read for one offer; those whose patient may not be offered it now (no care
	// consent, quiet hours) are passed over.
	candidatesPerOffer = 10
)

// The agents of the waitlist's jobs: their kinds.
var (
	agentSlotFreed   = SlotFreedArgs{}.Kind()
	agentOfferExpire = OfferExpireArgs{}.Kind()
)

// OfferedSlots moves a slot through the waitlist's statuses in the caller's transaction (slot.Service): ForWaitlist
// locks it, Offer holds a free one (busy-tentative), TakeOffered books an offered one, Unoffer gives one back.
type OfferedSlots interface {
	ForWaitlist(ctx context.Context, org, id string) (slot.Bookable, error)
	Offer(ctx context.Context, org, id string) (slot.Slot, error)
	TakeOffered(ctx context.Context, org, id string) (slot.Slot, error)
	Unoffer(ctx context.Context, org, id string) (bool, error)
}

// OfferJobs enqueues an offer's expiry at its end (waitlist_offer_expire), in the transaction in ctx.
type OfferJobs interface {
	EnqueueOfferExpire(ctx context.Context, offerID, org string, at time.Time) error
}

// OfferNotices tells the patient of a new offer, in the offer's transaction: notification.Service's deliveries (SAL-33),
// with the patient's channels and quiet hours, the offer's end the deadline.
type OfferNotices interface {
	NotifyOffer(ctx context.Context, o Offer, accountID string) error
}

// QuietHours tells whether the patient of the account is inside their quiet hours at at (notification.Service): such an
// entry is passed over for that offer.
type QuietHours interface {
	Quiet(ctx context.Context, org, accountID string, at time.Time) (bool, error)
}

// HandOn offers the slot slotID of the clinic org to the next matching entry of its waitlist (the waitlist_slot_freed
// job, enqueued by a cancellation, a reschedule, a decline or a withdrawal): see handOn. One transaction in the
// clinic's scope; a slot already booked or offered changes nothing.
func (s *Service) HandOn(ctx context.Context, org, slotID string) error {
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.Repo.SetClinic(ctx, org); err != nil {
			return err
		}
		return s.handOn(ctx, agentSlotFreed, org, slotID)
	})
	if err != nil {
		return fmt.Errorf("appointment: hand a slot on: %w", err)
	}
	return nil
}

// handOn locks the slot; a busy one, or one an open offer holds, stays as it is. Otherwise the slot is free again
// (an offered slot whose offer closed is given back), and offered to the first matching entry in offer order, held
// for the clinic's hold with the expiry and the patient's notice enqueued; with none, it stays free in the directory.
func (s *Service) handOn(ctx context.Context, agent, org, slotID string) error {
	b, err := s.Offered.ForWaitlist(ctx, org, slotID)
	switch {
	case errors.Is(err, slot.ErrUnavailable):
		return nil
	case err != nil:
		return err
	case b.Status == slot.StatusBusy:
		return nil
	case b.Status == slot.StatusTentative:
		if _, open, err := s.Waitlist.OpenOffer(ctx, slotID, ""); err != nil || open {
			return err
		}
		if _, err := s.Offered.Unoffer(ctx, org, slotID); err != nil {
			return err
		}
	}
	now := s.Clock.Now()
	e, hold, err := s.nextEntry(ctx, agent, org, b, now)
	if err != nil || e.ID == "" {
		return err
	}
	if _, err := s.Offered.Offer(ctx, org, slotID); err != nil {
		return err
	}
	o, err := s.Waitlist.InsertOffer(ctx, Offer{
		Created: now, ExpiresAt: now.Add(hold), Start: b.Start, End: b.End, EntryID: e.ID, SlotID: b.ID, OrganizationID: org,
		PatientID: e.PatientID, PractitionerRoleID: b.PractitionerRoleID,
	})
	if err != nil {
		return err
	}
	if err := s.OfferJobs.EnqueueOfferExpire(ctx, o.ID, org, o.ExpiresAt); err != nil {
		return err
	}
	if err := s.Notices.NotifyOffer(ctx, o, e.accountID); err != nil {
		return err
	}
	return s.recordJob(ctx, agent, org, auditevent.CodeUpdate, "Appointment", e.ID, auditevent.OutcomeSuccess)
}

// nextEntry returns the first entry the free slot b may be offered to at now, and the clinic's hold; none (an empty
// ID) while the clinic's waitlist is off, the slot starts within the hold and offerLead, its role is not active, or no
// entry matches. An entry whose patient's care consent is not active, or who is inside their quiet hours, is passed
// over. The read of the candidates is audited with the job as agent.
func (s *Service) nextEntry(ctx context.Context, agent, org string, b slot.Bookable, now time.Time) (candidate, time.Duration, error) {
	set, err := s.Rules.For(ctx, org)
	if err != nil {
		return candidate{}, 0, err
	}
	hold := time.Duration(set.WaitlistHoldMinutes) * time.Minute
	if !set.WaitlistEnabled || b.Start.Before(now.Add(hold+offerLead)) {
		return candidate{}, 0, nil
	}
	m, ok, err := s.match(ctx, org, b, now)
	if err != nil || !ok {
		return candidate{}, 0, err
	}
	candidates, err := s.Waitlist.Candidates(ctx, m, candidatesPerOffer)
	if err != nil || len(candidates) == 0 {
		return candidate{}, 0, err
	}
	if err := s.recordJob(ctx, agent, org, auditevent.CodeSearchType, "", "", auditevent.OutcomeSuccess); err != nil {
		return candidate{}, 0, err
	}
	for _, e := range candidates {
		if e.OrganizationID != org {
			continue // not on row-level security alone
		}
		consent, err := s.Consents.State(ctx, e.PatientID, org)
		if err != nil {
			return candidate{}, 0, err
		}
		if !consent.Active {
			continue
		}
		p, err := s.Patients.Lookup(ctx, e.PatientID)
		if err != nil {
			return candidate{}, 0, err
		}
		if err := s.recordJob(ctx, agent, org, auditevent.CodeRead, "Patient", p.ID, auditevent.OutcomeSuccess); err != nil {
			return candidate{}, 0, err
		}
		if quiet, err := s.Quiet.Quiet(ctx, org, p.AccountID, now); err != nil || quiet {
			if err != nil {
				return candidate{}, 0, err
			}
			continue
		}
		return candidate{WaitlistEntry: e, accountID: p.AccountID}, hold, nil
	}
	return candidate{}, 0, nil
}

// candidate is an entry chosen for an offer, with its patient's account (to tell them).
type candidate struct {
	accountID string
	WaitlistEntry
}

// match is what the slot b of the clinic org asks of an entry at now: its role (active) and the role's specialty, its
// day and times in the clinic's zone. False for a slot whose role is not an active one of the clinic.
func (s *Service) match(ctx context.Context, org string, b slot.Bookable, now time.Time) (Match, bool, error) {
	role, err := s.Roles.Lookup(ctx, org, b.PractitionerRoleID)
	switch {
	case errors.Is(err, practitionerrole.ErrNotFound):
		return Match{}, false, nil
	case err != nil:
		return Match{}, false, err
	case !role.Active:
		return Match{}, false, nil
	}
	clinic, err := s.Clinics.Read(ctx, organization.Caller{OrganizationID: org}, org)
	if err != nil {
		return Match{}, false, err
	}
	loc, err := time.LoadLocation(clinic.TimeZone)
	if err != nil {
		return Match{}, false, fmt.Errorf("the clinic's time zone: %w", err)
	}
	start, end := b.Start.In(loc), b.End.In(loc)
	day := dateOf(start)
	times := Times{Start: start.Hour()*60 + start.Minute(), End: minutesPerDay}
	if dateOf(end).Equal(day) {
		times.End = end.Hour()*60 + end.Minute()
	}
	return Match{
		Start: b.Start, End: b.End, Day: day, Now: now, OrganizationID: org, SlotID: b.ID,
		PractitionerRoleID: role.ID, Specialty: role.SpecialtyCode, Times: times,
	}, true, nil
}

// ExpireOffer ends the offer id of the clinic org once its hold is over (the waitlist_offer_expire job) and hands its
// slot on to the next entry, in one transaction in the clinic's scope that locks the offer first, as an acceptance
// does: an acceptance that locked it before leaves it accepted. The entry stays active. ErrOfferNotDue before the
// offer's end, for the job to try again then; an offer that is gone (retention) changes nothing.
func (s *Service) ExpireOffer(ctx context.Context, org, id string) error {
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.Repo.SetClinic(ctx, org); err != nil {
			return err
		}
		o, err := s.Waitlist.OfferForUpdate(ctx, id)
		switch {
		case errors.Is(err, ErrNotFound):
			return nil
		case err != nil:
			return err
		case o.OrganizationID != org:
			return nil
		}
		now := s.Clock.Now()
		if o.State == OfferOpen {
			if now.Before(o.ExpiresAt) {
				return ErrOfferNotDue
			}
			if _, err := s.Waitlist.CloseOffer(ctx, o.ID, OfferExpired, "", now); err != nil {
				return err
			}
			if err := s.recordJob(ctx, agentOfferExpire, org, auditevent.CodeUpdate, "Appointment", o.EntryID, auditevent.OutcomeSuccess); err != nil {
				return err
			}
		}
		return s.handOn(ctx, agentOfferExpire, org, o.SlotID)
	})
	switch {
	case errors.Is(err, ErrOfferNotDue):
		return ErrOfferNotDue
	case err != nil:
		return fmt.Errorf("appointment: expire an offer: %w", err)
	}
	return nil
}

// Decline ends the calling patient's open offer id; the entry stays active and the slot goes to the next entry
// (waitlist_slot_freed). Audited, refusals too. ErrNotFound for an offer that is not theirs or not open.
func (s *Service) Decline(ctx context.Context, c Caller, id string) error {
	var refused error
	var entry string
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		var err error
		if entry, refused, err = s.decline(ctx, c, id); err != nil {
			return err
		}
		if refused != nil {
			return s.record(ctx, c, auditevent.CodeUpdate, "", "", auditevent.OutcomeMinorFailure)
		}
		return s.record(ctx, c, auditevent.CodeUpdate, "Appointment", entry, auditevent.OutcomeSuccess)
	})
	switch {
	case err != nil:
		return fmt.Errorf("appointment: decline an offer: %w", err)
	case refused != nil:
		return refused
	}
	return nil
}

func (s *Service) decline(ctx context.Context, c Caller, id string) (entry string, refused, err error) {
	if !c.patient() || c.PatientID == "" {
		return "", ErrNotFound, nil
	}
	if err := s.Repo.SetPatient(ctx, c.PatientID); err != nil {
		return "", nil, err
	}
	o, err := s.Waitlist.OfferForUpdate(ctx, id)
	switch {
	case errors.Is(err, ErrNotFound):
		return "", ErrNotFound, nil
	case err != nil:
		return "", nil, err
	case o.PatientID != c.PatientID || o.State != OfferOpen:
		return "", ErrNotFound, nil
	}
	if _, err := s.Waitlist.CloseOffer(ctx, o.ID, OfferDeclined, "", s.Clock.Now()); err != nil {
		return "", nil, err
	}
	return o.EntryID, nil, s.Jobs.EnqueueSlotFreed(ctx, o.SlotID, o.OrganizationID)
}

// OfferNotice gives a notifier job (agent, its kind) the offer id of the clinic org and its patient's account, in the
// clinic's scope, audited with the job as agent. ErrNotFound for an offer the clinic does not have, or whose patient it
// no longer sees (the care consent revoked).
func (s *Service) OfferNotice(ctx context.Context, agent, org, id string) (OfferNotice, error) {
	var out OfferNotice
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.Repo.SetClinic(ctx, org); err != nil {
			return err
		}
		o, err := s.Waitlist.Offer(ctx, id)
		if err != nil {
			return err
		}
		if o.OrganizationID != org {
			return ErrNotFound
		}
		p, err := s.Patients.Lookup(ctx, o.PatientID)
		if errors.Is(err, patient.ErrNotFound) {
			return ErrNotFound // the clinic no longer sees the patient
		} else if err != nil {
			return err
		}
		out = OfferNotice{Offer: o, AccountID: p.AccountID}
		if err := s.recordJob(ctx, agent, org, auditevent.CodeRead, "Appointment", o.EntryID, auditevent.OutcomeSuccess); err != nil {
			return err
		}
		return s.recordJob(ctx, agent, org, auditevent.CodeRead, "Patient", p.ID, auditevent.OutcomeSuccess)
	})
	switch {
	case errors.Is(err, ErrNotFound):
		return OfferNotice{}, ErrNotFound
	case err != nil:
		return OfferNotice{}, fmt.Errorf("appointment: offer notice: %w", err)
	}
	return out, nil
}
