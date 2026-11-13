package appointment

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/auditevent"
)

// tokenBytes is the length of a one-time link's random token (ADR 0015 "One-tap links").
const tokenBytes = 32

// recordChange appends c to its appointment's history in the transaction in ctx; every confirmation, reschedule and
// status change goes through it.
func (s *Service) recordChange(ctx context.Context, c Change) error {
	_, err := s.Repo.AddChange(ctx, c)
	return err
}

// Changes returns the history of the appointment id, oldest first, to whoever sees it in full: the patient their own
// (without who or which job made each change), its practitioner and the clinic's admin with the patient's consent.
// Audited as a read of the appointment. ErrNotFound for one the caller may not see, ErrForbidden for one seen at the
// name only (a past one after a revocation); each refusal is audited.
func (s *Service) Changes(ctx context.Context, c Caller, id string) ([]Change, error) {
	var out []Change
	var refused error
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		v, err := s.view(ctx, c, id)
		switch {
		case err != nil:
			return err
		case v.level == access.None:
			refused = ErrNotFound
		case v.level != access.Full:
			refused = ErrForbidden
		}
		if refused != nil {
			return s.record(ctx, c, auditevent.CodeRead, "", "", auditevent.OutcomeMinorFailure)
		}
		if out, err = s.Repo.Changes(ctx, id); err != nil {
			return err
		}
		if c.patient() {
			for i := range out {
				out[i].ByAccountID, out[i].Agent = "", ""
			}
		}
		return s.record(ctx, c, auditevent.CodeRead, "Appointment", id, auditevent.OutcomeSuccess)
	})
	switch {
	case err != nil:
		return nil, fmt.Errorf("appointment: changes: %w", err)
	case refused != nil:
		return nil, refused
	}
	return out, nil
}

// IssueLink stores a one-time link for l, for the job agent (a reminder's), and returns its URL,
// <LinkURL>/a/<token>: 32 random bytes, base64url, of which only the SHA-256 is stored, bound to the appointment, its
// patient and their account, expiring at the start (ADR 0015 "One-tap links"). In one transaction in the clinic's
// scope with the job's appointment set, audited with the job as agent. ErrNotFound for an appointment that is not the
// clinic's, ErrClosed for one that is not booked or has started; the refusal is audited.
func (s *Service) IssueLink(ctx context.Context, agent string, l Link) (string, error) {
	if s.Tokens == nil || s.LinkURL == "" || (l.Channel != ChannelEmail && l.Channel != ChannelSMS) {
		return "", errors.New("appointment: issue link: no token store, link base or channel")
	}
	raw := make([]byte, tokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("appointment: issue link: %w", err)
	}
	var refused error
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		var err error
		if refused, err = s.issue(ctx, l, sha256.Sum256(raw)); err != nil {
			return err
		}
		if refused != nil {
			return s.recordJob(ctx, agent, l.OrganizationID, auditevent.CodeCreate, "", "", auditevent.OutcomeMinorFailure)
		}
		return s.recordJob(ctx, agent, l.OrganizationID, auditevent.CodeCreate, "Appointment", l.AppointmentID, auditevent.OutcomeSuccess)
	})
	switch {
	case err != nil:
		return "", fmt.Errorf("appointment: issue link: %w", err)
	case refused != nil:
		return "", refused
	}
	return strings.TrimRight(s.LinkURL, "/") + "/a/" + base64.RawURLEncoding.EncodeToString(raw), nil
}

// issue stores the link l with the token hash in the transaction in ctx, or gives the refusal.
func (s *Service) issue(ctx context.Context, l Link, hash [32]byte) (refused, err error) {
	if err := s.Repo.SetClinic(ctx, l.OrganizationID); err != nil {
		return nil, err
	}
	if err := s.Repo.SetNotice(ctx, l.AppointmentID); err != nil {
		return nil, err
	}
	a, err := s.Repo.ByID(ctx, l.AppointmentID)
	now := s.Clock.Now()
	switch {
	case errors.Is(err, ErrNotFound), err == nil && a.OrganizationID != l.OrganizationID:
		return ErrNotFound, nil
	case err != nil:
		return nil, err
	case a.Status != StatusBooked || !a.Start.After(now):
		return ErrClosed, nil
	}
	p, err := s.Patients.Lookup(ctx, a.PatientID)
	if err != nil {
		return nil, err
	}
	return nil, s.Tokens.InsertToken(ctx, Token{
		CreatedAt: now, ExpiresAt: a.Start, AppointmentID: a.ID, OrganizationID: a.OrganizationID, PatientID: a.PatientID,
		AccountID: p.AccountID, ReminderID: l.ReminderID, Language: l.Language, Channel: l.Channel, Hash: hash,
	})
}

// TokenPurger applies the retention rule of appointment_action_token (internal/retention): links 30 days after they
// expired.
type TokenPurger struct {
	repo TokenRepository
}

// NewTokenPurger returns the purger of appointment_action_token on repo, built on the retention role's pool.
func NewTokenPurger(repo TokenRepository) *TokenPurger { return &TokenPurger{repo: repo} }

// Purge deletes the links that expired before before and returns how many.
func (p *TokenPurger) Purge(ctx context.Context, before time.Time) (int64, error) {
	n, err := p.repo.DeleteTokensExpiredBefore(ctx, before)
	if err != nil {
		return 0, fmt.Errorf("appointment: purge links: %w", err)
	}
	return n, nil
}
