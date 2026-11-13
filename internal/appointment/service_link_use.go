package appointment

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/netip"
	"time"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/organization"
)

// The one-time links' use (ADR 0015 "One-tap links"): the agent of their audit events, beside the patient's account,
// and the lookups an address may make in a window (Redis, by an HMAC of the address).
const (
	agentActionLink = "action_link"
	linkLookups     = 30
	linkWindow      = 10 * time.Minute
)

// errLinkRefused rolls back a link's use whose action was refused: the link stays unused.
var errLinkRefused = errors.New("appointment: the link's action was refused")

// opened is a link found by its token, with what the page shows of it, read in its patient's scope.
type opened struct {
	a      Appointment
	clinic organization.Organization
	caller Caller // the link's patient through the link
	tok    Token
	valid  bool // unused, unexpired, issued for the appointment's start, booked, under an active consent
}

// Link returns what the page of the one-time link token shows, for a lookup from the address addr (GET /a/{token}):
// for a used link its outcome only; for a usable one the clinic, the start and the actions still possible. Changes
// nothing however often it is fetched. In one transaction in the link's patient's scope, audited as a read of the
// appointment by the patient's account through the link. ErrLinkInvalid for an unknown, expired or used-up link, one
// issued for another start than the appointment's now, an appointment no longer booked, or a consent no longer active
// (the page then says the link no longer works, in the link's language when it is known: LinkPage.Language);
// ErrLinkRateLimited when the address looked up too many links.
func (s *Service) Link(ctx context.Context, token, addr string) (LinkPage, error) {
	if err := s.allowLookup(ctx, addr); err != nil {
		return LinkPage{}, err
	}
	hash, ok := tokenHash(token)
	if !ok {
		return LinkPage{}, ErrLinkInvalid
	}
	var out LinkPage
	var refused error
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		o, err := s.open(ctx, hash)
		switch {
		case errors.Is(err, ErrLinkInvalid):
			refused = err
			return nil
		case err != nil:
			return err
		}
		now := s.Clock.Now()
		out = page(o, now)
		if !o.valid && o.tok.UsedAt.IsZero() {
			refused = ErrLinkInvalid
			return s.record(ctx, o.caller, auditevent.CodeRead, "", "", auditevent.OutcomeMinorFailure)
		}
		return s.record(ctx, o.caller, auditevent.CodeRead, "Appointment", o.a.ID, auditevent.OutcomeSuccess)
	})
	switch {
	case err != nil:
		return LinkPage{}, fmt.Errorf("appointment: link: %w", err)
	case refused != nil:
		return LinkPage{Language: out.Language}, refused
	}
	return out, nil
}

// UseLink confirms or cancels the appointment of the one-time link token as its patient, for a request from the
// address addr (POST /a/{token}): a cancel only with its box ticked (confirmCancel). In one transaction in the
// patient's scope the link is used (once: a used link never changes) and the appointment confirmed (via the link) or
// cancelled as the patient would in their app (until 24 hours before the start, recorded via the link), audited as the
// patient's account through the link. A refused action leaves the link unused. ErrConfirmCancelRequired for a cancel
// without its box (nothing is looked up), ErrLinkInvalid for a link Link refuses, used already or for another action,
// ErrClosed for a confirmation already made, ErrCancellationWindowClosed inside the 24 hours, ErrLinkRateLimited;
// each refusal of a known link is audited.
func (s *Service) UseLink(ctx context.Context, token, addr string, action LinkAction, confirmCancel bool) error {
	if action == LinkCancel && !confirmCancel {
		return ErrConfirmCancelRequired
	}
	if action != LinkConfirm && action != LinkCancel {
		return ErrLinkInvalid
	}
	if err := s.allowLookup(ctx, addr); err != nil {
		return err
	}
	hash, ok := tokenHash(token)
	if !ok {
		return ErrLinkInvalid
	}
	var caller Caller
	var refused error
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		o, err := s.open(ctx, hash)
		if err != nil {
			return err
		}
		caller = o.caller
		if refused, err = s.use(ctx, o, action); err != nil {
			return err
		}
		if refused != nil {
			return errLinkRefused
		}
		return s.record(ctx, o.caller, auditevent.CodeUpdate, "Appointment", o.a.ID, auditevent.OutcomeSuccess)
	})
	switch {
	case errors.Is(err, ErrLinkInvalid):
		return ErrLinkInvalid
	case errors.Is(err, errLinkRefused):
		return s.refuse(ctx, caller, auditevent.CodeUpdate, refused)
	case err != nil:
		return fmt.Errorf("appointment: use link: %w", err)
	}
	return nil
}

// use uses the link o for action and does it, in the transaction in ctx, or gives the refusal.
func (s *Service) use(ctx context.Context, o opened, action LinkAction) (refused, err error) {
	switch {
	case !o.valid:
		return ErrLinkInvalid, nil
	case action == LinkConfirm && !o.a.ConfirmedAt.IsZero():
		return ErrClosed, nil // nothing to confirm: the link stays for a cancel
	}
	// The appointment as it is now, locked: a reschedule or cancellation that committed after open read it is seen, and
	// one that starts now waits for this transaction.
	a, err := s.SelfService.ByIDForUpdate(ctx, o.a.ID)
	now := s.Clock.Now()
	switch {
	case errors.Is(err, ErrNotFound):
		return ErrLinkInvalid, nil
	case err != nil:
		return nil, err
	case a.Status != StatusBooked || !a.Start.Equal(o.tok.StartAt) || !now.Before(o.tok.StartAt):
		return ErrLinkInvalid, nil
	case action == LinkConfirm && !a.ConfirmedAt.IsZero():
		return ErrClosed, nil
	}
	if err := s.SelfService.UseToken(ctx, o.tok.ID, action, now); errors.Is(err, ErrLinkInvalid) {
		return ErrLinkInvalid, nil // a concurrent use committed first
	} else if err != nil {
		return nil, err
	}
	if action == LinkConfirm {
		_, refused, err = s.confirm(ctx, o.caller, o.a.ID)
		return refused, err
	}
	_, refused, err = s.cancel(ctx, o.caller, o.a.ID)
	return refused, err
}

// open finds the link with the token hash and reads it, its appointment and its clinic in the link's patient's scope,
// in the transaction in ctx; ErrLinkInvalid when no link has the hash, or its appointment is not the patient's.
func (s *Service) open(ctx context.Context, hash [32]byte) (opened, error) {
	ids, err := s.SelfService.FindToken(ctx, hash)
	if err != nil {
		return opened{}, err
	}
	caller := Caller{AccountID: ids.AccountID, PatientID: ids.PatientID, Via: ViaLink}
	if err := s.Repo.SetPatient(ctx, ids.PatientID); err != nil {
		return opened{}, err
	}
	tok, err := s.SelfService.TokenByID(ctx, ids.ID)
	if err != nil {
		return opened{}, err
	}
	v, err := s.view(ctx, caller, tok.AppointmentID)
	switch {
	case err != nil:
		return opened{}, err
	case v.level != access.Full || v.a.ID != tok.AppointmentID:
		return opened{}, ErrLinkInvalid
	}
	o := opened{tok: tok, a: v.a, caller: caller}
	if o.clinic, err = s.Clinics.Read(ctx, organization.Caller{OrganizationID: v.a.OrganizationID}, v.a.OrganizationID); err != nil {
		return opened{}, err
	}
	consent, err := s.Consents.State(ctx, v.a.PatientID, v.a.OrganizationID)
	if err != nil {
		return opened{}, err
	}
	now := s.Clock.Now()
	o.valid = tok.UsedAt.IsZero() && now.Before(tok.StartAt) && tok.StartAt.Equal(v.a.Start) && v.a.Status == StatusBooked && consent.Active
	return o, nil
}

// page is what the page of the link o shows at now: a used link's outcome at the start it was issued for; a valid
// one's state and actions; for any other, its language only.
func page(o opened, now time.Time) LinkPage {
	loc, err := time.LoadLocation(o.clinic.TimeZone)
	if err != nil {
		loc = time.UTC
	}
	out := LinkPage{Clinic: o.clinic.Name, TimeZone: o.clinic.TimeZone, Language: o.tok.Language, Start: o.tok.StartAt.In(loc)}
	switch {
	case !o.tok.UsedAt.IsZero():
		out.Used = o.tok.UsedAction
	case !o.valid:
		return LinkPage{Language: o.tok.Language}
	default:
		out.Confirmed = !o.a.ConfirmedAt.IsZero()
		out.CanConfirm = !out.Confirmed
		out.CanCancel = o.a.Start.Sub(now) >= patientCancelBefore
		out.CancelClosed = !out.CanCancel
	}
	return out
}

// allowLookup counts a link lookup from addr, under an HMAC of the address (never the address itself);
// ErrLinkRateLimited past the limit. Without a limiter every lookup is allowed.
func (s *Service) allowLookup(ctx context.Context, addr string) error {
	if s.Limiter == nil {
		return nil
	}
	mac := hmac.New(sha256.New, s.LinkKey)
	mac.Write([]byte(lookupKey(addr)))
	ok, err := s.Limiter.Allow(ctx, "link:"+hex.EncodeToString(mac.Sum(nil)), linkLookups, linkWindow)
	switch {
	case err != nil:
		return fmt.Errorf("appointment: link lookups: %w", err)
	case !ok:
		return ErrLinkRateLimited
	}
	return nil
}

// lookupKey is what an address's lookups are counted under: an IPv6 address's /64 (the block one subscriber holds,
// which they can cycle through), an IPv4 address itself, and anything unparsable as it is.
func lookupKey(addr string) string {
	ip, err := netip.ParseAddr(addr)
	if err != nil {
		return addr
	}
	if ip = ip.Unmap(); ip.Is6() {
		p, _ := ip.Prefix(64) // never fails for a /64 of an IPv6 address
		return p.String()
	}
	return ip.String()
}

// tokenHash is the SHA-256 of the raw bytes of a link's token (base64url, 32 bytes), and false for anything else.
func tokenHash(token string) ([32]byte, bool) {
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil || len(raw) != tokenBytes {
		return [32]byte{}, false
	}
	return sha256.Sum256(raw), true
}
