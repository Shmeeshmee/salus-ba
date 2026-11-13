package notification

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/pkg/sms"
)

// WaitlistOffers gives a send its waitlist offer and the patient's account, audited with the job as agent
// (appointment.Service).
type WaitlistOffers interface {
	OfferNotice(ctx context.Context, agent, org, id string) (appointment.OfferNotice, error)
}

// NotifyOffer stores the deliveries of the new offer o to its patient's account, with their send jobs, in the offer's
// transaction (SAL-35; appointment.OfferNotices): e-mail, a push per device and the SMS the patient opted into, as
// Deliver decides them, the offer's end their deadline (a push or SMS held by quiet hours past it is dropped).
func (s *Service) NotifyOffer(ctx context.Context, o appointment.Offer, accountID string) error {
	if err := s.Deliver(ctx, KindWaitlistOffer, o.ID, o.OrganizationID, accountID, Content{Deadline: o.ExpiresAt}); err != nil {
		return fmt.Errorf("notification: notify an offer: %w", err)
	}
	return nil
}

// Quiet tells whether the account's patient is inside their quiet hours at at, in their zone, else the clinic org's
// (appointment.QuietHours: such an entry is passed over for an offer). The read of their settings is audited with the
// offer's delivery as agent.
func (s *Service) Quiet(ctx context.Context, org, accountID string, at time.Time) (bool, error) {
	if s.p.Deliveries == nil || s.p.Audit == nil || s.clinics == nil {
		return false, errors.New("notification: quiet hours: the Service was built without the deliveries' ports")
	}
	p, err := s.preferencesOf(ctx, agentOf(KindWaitlistOffer), org, accountID)
	if err != nil || p.Quiet == nil {
		return false, err
	}
	loc, err := s.zoneOf(ctx, org, p)
	if err != nil {
		return false, fmt.Errorf("notification: quiet hours: %w", err)
	}
	_, inside := quietEnd(at, *p.Quiet, loc)
	return inside, nil
}

// composeOffer writes d's message of a waitlist offer, or errGone when it no longer applies: the offer closed or its
// hold ended, it is another account's, or d's channel is off now. The push is generic; the e-mail and the SMS give the
// clinic, the time offered and the hold's end in the clinic's zone with its name, and the link to the app (accepting
// happens there, signed in); nothing of a practitioner or specialty.
func (s *Service) composeOffer(ctx context.Context, d Delivery) (outgoing, error) {
	agent := agentSend[d.Channel]
	n, err := s.p.Offers.OfferNotice(ctx, agent, d.OrganizationID, d.SourceID)
	switch {
	case errors.Is(err, appointment.ErrNotFound):
		return outgoing{}, errGone
	case err != nil:
		return outgoing{}, err
	case n.Offer.State != appointment.OfferOpen || !s.p.Clock.Now().Before(n.Offer.ExpiresAt) || n.AccountID != d.AccountID:
		return outgoing{}, errGone
	}
	prefs, err := s.preferencesOf(ctx, agent, d.OrganizationID, d.AccountID)
	if err != nil {
		return outgoing{}, err
	}
	if on, err := s.stillOn(ctx, d, prefs); err != nil || !on {
		return outgoing{}, errors.Join(err, errGone)
	}
	if d.Channel == ChannelPush {
		return s.composePush(ctx, agent, d, "waitlist_offer_push")
	}
	clinic, err := s.clinics.Read(ctx, organization.Caller{OrganizationID: d.OrganizationID}, d.OrganizationID)
	if err != nil {
		return outgoing{}, err
	}
	loc, err := time.LoadLocation(clinic.TimeZone)
	if err != nil {
		return outgoing{}, fmt.Errorf("the clinic's time zone: %w", err)
	}
	if d.Channel == ChannelEmail {
		return s.composeOfferEmail(ctx, agent, d, n.Offer, clinic, loc)
	}
	return s.composeOfferSMS(ctx, agent, d, n.Offer, clinic, loc)
}

// composeOfferEmail is the offer's e-mail in the account's language.
func (s *Service) composeOfferEmail(ctx context.Context, agent string, d Delivery, o appointment.Offer, clinic organization.Organization, loc *time.Location) (outgoing, error) {
	to, language, err := s.accounts.AccountEmail(ctx, agent, d.AccountID)
	if err != nil {
		return outgoing{}, err
	}
	l := s.in(language)
	when, err := l.when(o.Start.In(loc), clinic.TimeZone)
	if err != nil {
		return outgoing{}, err
	}
	until, err := holdEnd(l, o, loc, clinic.TimeZone)
	if err != nil {
		return outgoing{}, err
	}
	msg, err := l.message(to, "waitlist_offer", letter{Clinic: clinic.Name, When: when, Until: until, Link: s.patientAppURL + "/waitlist"})
	return outgoing{mail: msg, ready: true}, err
}

// composeOfferSMS is the offer's SMS to the patient's phone; no phone any more cancels it.
func (s *Service) composeOfferSMS(ctx context.Context, agent string, d Delivery, o appointment.Offer, clinic organization.Organization, loc *time.Location) (outgoing, error) {
	phone, err := s.p.Phones.NoticePhone(ctx, agent, d.OrganizationID, o.PatientID)
	if err != nil || phone == "" {
		return outgoing{}, errors.Join(err, errGone)
	}
	language, err := s.p.Languages.AccountLanguage(ctx, agent, d.AccountID)
	if err != nil {
		return outgoing{}, err
	}
	start, end := o.Start.In(loc), o.ExpiresAt.In(loc)
	until := end.Format("15:04")
	if !sameDay(end, o.Created.In(loc)) {
		until = end.Format("02.01. 15:04")
	}
	body, err := s.in(language).text("waitlist_offer_sms", letter{
		Clinic: clinic.Name, Zone: clinic.TimeZone, Day: start.Format("02"), Month: start.Format("01"), Year: start.Year(),
		Time: start.Format("15:04"), Until: until, Link: s.patientAppURL + "/waitlist",
	})
	if err != nil {
		return outgoing{}, err
	}
	return outgoing{sms: sms.Message{To: phone, Body: body, Reference: d.ID}, parts: sms.Segments(body), ready: true}, nil
}

// holdEnd writes the end of o's hold in the clinic's zone loc (named zone) with the zone's name: its time when it ends
// on the day the offer was made, and its date and time when it ends on a later day (a hold past midnight).
func holdEnd(l letters, o appointment.Offer, loc *time.Location, zone string) (string, error) {
	end := o.ExpiresAt.In(loc)
	if sameDay(end, o.Created.In(loc)) {
		return end.Format("15:04") + " (" + zone + ")", nil
	}
	return l.when(end, zone)
}

func sameDay(a, b time.Time) bool { return a.YearDay() == b.YearDay() && a.Year() == b.Year() }
