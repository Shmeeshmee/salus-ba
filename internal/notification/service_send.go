package notification

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Shmeeshmee/salus/pkg/push"
	"github.com/Shmeeshmee/salus/pkg/sms"
)

// errGone marks a delivery that no longer applies: it is cancelled, nothing sent.
var errGone = errors.New("notification: the delivery no longer applies")

// errUnsupported marks a delivery of a kind no composer writes yet (a waitlist offer before SAL-35): it fails for good
// rather than retry.
var errUnsupported = errors.New("notification: no composer for the delivery's kind")

// outgoing is one composed message of a delivery.
type outgoing struct {
	mail  Message
	push  push.Message
	sms   sms.Message
	parts int
	ready bool
}

// SendDelivery sends the delivery id (the reminder_email, reminder_push and reminder_sms jobs): locked and checked in a
// transaction, the reminder and its appointment read again, the channel checked still on (the patient's settings, and
// the clinic's SMS option for an SMS) and the message composed (with a new one-time link for an e-mail or SMS); sent
// after that commits, then marked sent, an SMS counted in its clinic's usage in the same transaction. A delivery no
// longer pending sends nothing; one whose reminder, appointment, device, phone or channel is gone is cancelled; one of
// a kind no composer writes, or a refusal no retry can change (an address, number or token the provider will not
// take), fails it, as does the last attempt (last). Errors carry no address, number, token or text.
func (s *Service) SendDelivery(ctx context.Context, id string, last bool) error {
	if err := s.delivers(true); err != nil {
		return err
	}
	var d Delivery
	var out outgoing
	err := s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		var err error
		if d, err = s.p.Deliveries.LockDelivery(ctx, id); err != nil {
			return err
		}
		if d.State != DeliveryPending {
			return ErrDeliveryDone
		}
		out, err = s.compose(ctx, d)
		switch {
		case errors.Is(err, errGone):
			return s.p.Deliveries.SetDeliveryState(ctx, id, DeliveryCancelled, s.p.Clock.Now(), 0)
		case errors.Is(err, errUnsupported):
			return s.p.Deliveries.SetDeliveryState(ctx, id, DeliveryFailed, s.p.Clock.Now(), 0)
		}
		return err
	})
	switch {
	case errors.Is(err, ErrNotFound), errors.Is(err, ErrDeliveryDone):
		return nil
	case err != nil:
		return fmt.Errorf("notification: send %s: %w", d.Channel, err)
	case !out.ready:
		return nil
	}
	if err := s.send(ctx, d, out); err != nil {
		if permanent(err) || last {
			if ferr := s.finish(ctx, d, DeliveryFailed, 0); ferr != nil {
				return errors.Join(err, ferr)
			}
			if permanent(err) {
				return nil
			}
		}
		return fmt.Errorf("notification: send %s: %w", d.Channel, err)
	}
	if err := s.finish(ctx, d, DeliverySent, out.parts); err != nil {
		return fmt.Errorf("notification: send %s: %w", d.Channel, err)
	}
	return nil
}

// delivers is nil when the Service has the ports of the deliveries (and of the sends, for sends): a binary that builds
// it for the settings alone (the api) gets an error here, never a nil dereference.
func (s *Service) delivers(sends bool) error {
	p := s.p
	ready := p.Deliveries != nil && p.DeliveryJobs != nil && p.Devices != nil && p.Rules != nil && p.Audit != nil && p.Tx != nil && p.Clock != nil &&
		s.clinics != nil
	if sends {
		ready = ready && p.Reminders != nil && p.Languages != nil && p.Phones != nil && p.Pusher != nil && p.SMS != nil &&
			s.sender != nil && s.accounts != nil && s.questionnaires != nil
	}
	if !ready {
		return errors.New("notification: the Service was built without the ports of the deliveries")
	}
	return nil
}

// permanent tells whether err is a refusal no retry changes.
func permanent(err error) bool {
	return errors.Is(err, ErrRecipientNotASCII) || errors.Is(err, sms.ErrRejected) || errors.Is(err, sms.ErrInvalidMessage) ||
		errors.Is(err, push.ErrTokenGone)
}

// send hands out to its channel's sender; a push token the platform says is gone is forgotten.
func (s *Service) send(ctx context.Context, d Delivery, out outgoing) error {
	switch d.Channel {
	case ChannelEmail:
		return s.sender.Send(ctx, out.mail)
	case ChannelSMS:
		_, err := s.p.SMS.Send(ctx, out.sms)
		return err
	}
	err := s.p.Pusher.Send(ctx, out.push)
	if errors.Is(err, push.ErrTokenGone) {
		if ferr := s.p.Devices.DeleteDevice(ctx, d.DeviceID, out.push.Token); ferr != nil {
			return errors.Join(err, ferr)
		}
	}
	return err
}

// finish moves d from pending to state now, counting a sent SMS in its clinic's month, in one transaction.
func (s *Service) finish(ctx context.Context, d Delivery, state DeliveryState, parts int) error {
	return s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		now := s.p.Clock.Now()
		if err := s.p.Deliveries.SetDeliveryState(ctx, d.ID, state, now, parts); err != nil {
			return err
		}
		if state != DeliverySent || d.Channel != ChannelSMS {
			return nil
		}
		if err := s.p.Devices.SetClinic(ctx, d.OrganizationID); err != nil {
			return err
		}
		u := now.UTC()
		return s.p.Deliveries.AddSMSUsage(ctx, d.OrganizationID, time.Date(u.Year(), u.Month(), 1, 0, 0, 0, 0, time.UTC), parts)
	})
}
