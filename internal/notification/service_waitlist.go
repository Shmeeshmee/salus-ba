package notification

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/pkg/push"
)

// WaitlistOffers gives the notifier's job a waitlist offer and its patient's account, audited with the job as agent
// (appointment.Service).
type WaitlistOffers interface {
	OfferNotice(ctx context.Context, agent, org, id string) (appointment.OfferNotice, error)
}

// WaitlistPorts are what Waitlist runs on, and the base of its link.
type WaitlistPorts struct {
	Offers   WaitlistOffers
	Accounts Accounts
	Clinics  Clinics
	Repo     Repository // the account's push devices
	Pusher   push.Sender
	Sender   Sender
	Audit    Auditor
	Tx       Transactor
	// PatientAppURL is PATIENT_APP_URL: the offer's e-mail links to <PatientAppURL>/waitlist.
	PatientAppURL string
}

// Waitlist tells a patient of a waitlist offer (the waitlist_offer_notify job, SAL-35): an e-mail with the clinic's
// name, the time offered and the end of its hold in the clinic's zone, and a link to the app, and a push with a fixed
// text to each of their devices; never a practitioner, a specialty or anything clinical, and accepting happens in the
// app only, signed in. Until SAL-33's deliveries (notification.Service.Deliver, with the patient's channels, quiet hours
// and SMS) it sends through the plain e-mail and push paths, so a retry may repeat a message.
type Waitlist struct {
	letters map[string]letters
	WaitlistPorts
}

// NewWaitlist returns Waitlist on its ports.
func NewWaitlist(p WaitlistPorts) *Waitlist {
	p.PatientAppURL = strings.TrimRight(p.PatientAppURL, "/")
	return &Waitlist{WaitlistPorts: p, letters: loadLetters()}
}

// agentOffer is the agent of the offer job's reads: its kind.
var agentOffer = appointment.OfferNotifyArgs{}.Kind()

// NotifyOffer tells the patient of the offer id of the clinic org, while it is open: the e-mail, then a push per
// device (a token the platform says is gone is forgotten). An offer the clinic no longer has, or whose patient it no
// longer sees, sends nothing; appointment.ErrNotFound passes through for the job to cancel. Errors carry no address,
// token or text.
func (s *Waitlist) NotifyOffer(ctx context.Context, id, org string) error {
	n, err := s.Offers.OfferNotice(ctx, agentOffer, org, id)
	if err != nil {
		return fmt.Errorf("notification: waitlist offer: %w", err)
	}
	if n.Offer.State != appointment.OfferOpen {
		return nil
	}
	clinic, err := s.Clinics.Read(ctx, organization.Caller{OrganizationID: org}, org)
	if err != nil {
		return fmt.Errorf("notification: waitlist offer: %w", err)
	}
	loc, err := time.LoadLocation(clinic.TimeZone)
	if err != nil {
		return fmt.Errorf("notification: waitlist offer: the clinic's time zone: %w", err)
	}
	to, language, err := s.Accounts.AccountEmail(ctx, agentOffer, n.AccountID)
	if err != nil {
		return fmt.Errorf("notification: waitlist offer: %w", err)
	}
	l := pick(s.letters, language)
	when, err := l.when(n.Offer.Start.In(loc), clinic.TimeZone)
	if err != nil {
		return fmt.Errorf("notification: waitlist offer: %w", err)
	}
	msg, err := l.message(to, "waitlist_offer", letter{
		Clinic: clinic.Name, When: when, Until: n.Offer.ExpiresAt.In(loc).Format("15:04"), Link: s.PatientAppURL + "/waitlist",
	})
	if err != nil {
		return fmt.Errorf("notification: waitlist offer: %w", err)
	}
	if err := s.Sender.Send(ctx, msg); err != nil {
		return fmt.Errorf("notification: waitlist offer: %w", err)
	}
	return s.push(ctx, org, n.AccountID, l)
}

// push sends the fixed offer alert in l to each device of the account, audited as a read of its devices.
func (s *Waitlist) push(ctx context.Context, org, account string, l letters) error {
	var devices []Device
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		var err error
		if devices, err = s.Repo.Devices(ctx, account); err != nil || len(devices) == 0 {
			return err
		}
		return s.Audit.Record(ctx, auditevent.AuditEvent{
			Type:            auditevent.Coding{System: auditevent.SystemAuditEventType, Code: auditevent.CodeRest},
			Subtype:         auditevent.Coding{System: auditevent.SystemRestfulInteraction, Code: auditevent.CodeRead},
			Purpose:         auditevent.Coding{System: auditevent.SystemActReason, Code: auditevent.CodeTREAT},
			Action:          auditevent.ActionRead,
			Outcome:         auditevent.OutcomeSuccess,
			AgentName:       agentOffer,
			OrganizationID:  org,
			EntityReference: "Account/" + account,
			EntityName:      "push_device",
			EntityCount:     int64(len(devices)),
		})
	})
	if err != nil {
		return fmt.Errorf("notification: waitlist offer: push: %w", err)
	}
	alert, err := l.message("", "waitlist_offer_push", letter{})
	if err != nil {
		return fmt.Errorf("notification: waitlist offer: push: %w", err)
	}
	var errs []error
	for _, d := range devices {
		err := s.Pusher.Send(ctx, push.Message{Token: d.Token, Platform: d.Platform, Title: alert.Subject, Body: strings.TrimSuffix(alert.Body, "\n")})
		switch {
		case errors.Is(err, push.ErrTokenGone):
			if err := s.Repo.DeleteDevice(ctx, d.ID, d.Token); err != nil {
				errs = append(errs, err)
			}
		case err != nil:
			errs = append(errs, err)
		}
	}
	if err := errors.Join(errs...); err != nil {
		return fmt.Errorf("notification: waitlist offer: push: %w", err)
	}
	return nil
}
