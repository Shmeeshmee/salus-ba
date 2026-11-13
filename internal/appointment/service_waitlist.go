package appointment

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/internal/practitionerrole"
)

// The waitlist's limits (docs/plans/appointments.md "Entry lifetime", "Waitlist order").
const (
	maxActiveEntries = 5   // per patient and clinic
	maxWaitlistDays  = 90  // the latest date at most this many days ahead
	maxPriority      = 100 // staff move an entry up from 0 to this
	minutesPerDay    = 24 * 60
)

// Settings gives a clinic's appointment rules in the caller's transaction, setting the clinic's scope on it
// (organization.Appointments).
type Settings interface {
	For(ctx context.Context, org string) (organization.AppointmentSettings, error)
}

// Join puts the calling patient on the waitlist of a clinic for one of its practitioner roles or a specialty, between
// two dates of the clinic's zone (from today, at most 90 days ahead) and, optionally, a time of day. In one transaction
// in the patient's scope: the clinic is listed and has the waitlist on, the role is an active one of the clinic, the
// patient's care consent there is active and they have fewer than five active entries there. Audited, refusals too.
// ErrForbidden to staff, ErrProfileRequired without a profile, ErrInvalidEntry, ErrWaitlistOff, ErrConsentRequired,
// ErrTooManyEntries.
func (s *Service) Join(ctx context.Context, c Caller, in NewEntry) (WaitlistEntry, error) {
	switch {
	case !c.patient():
		return WaitlistEntry{}, s.refuse(ctx, c, auditevent.CodeCreate, ErrForbidden)
	case c.PatientID == "":
		return WaitlistEntry{}, s.refuse(ctx, c, auditevent.CodeCreate, ErrProfileRequired)
	case !validEntry(in):
		return WaitlistEntry{}, ErrInvalidEntry
	}
	var out WaitlistEntry
	var refused error
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		var err error
		if out, refused, err = s.join(ctx, c, in); err != nil {
			return err
		}
		if refused != nil {
			return s.record(ctx, c, auditevent.CodeCreate, "", "", auditevent.OutcomeMinorFailure)
		}
		return s.record(ctx, c, auditevent.CodeCreate, "Appointment", out.ID, auditevent.OutcomeSuccess)
	})
	switch {
	case err != nil:
		return WaitlistEntry{}, fmt.Errorf("appointment: join the waitlist: %w", err)
	case refused != nil:
		return WaitlistEntry{}, refused
	}
	return out, nil
}

// validEntry reports whether in names one of a role and a known specialty, ordered dates and, when given, ordered times.
func validEntry(in NewEntry) bool {
	if (in.PractitionerRoleID == "") == (in.Specialty == "") || in.Earliest.After(in.Latest) || in.Earliest.IsZero() {
		return false
	}
	if _, ok := practitionerrole.SpecialtyByCode(in.Specialty); in.Specialty != "" && !ok {
		return false
	}
	t := in.Times
	return t.Any() && t.Start == 0 || t.Start >= 0 && t.Start < t.End && t.End <= minutesPerDay
}

// join checks and stores the entry in the transaction in ctx, or gives the refusal.
func (s *Service) join(ctx context.Context, c Caller, in NewEntry) (out WaitlistEntry, refused, err error) {
	org := in.OrganizationID
	set, err := s.Settings.For(ctx, org) // sets the clinic's scope: the patient's is set after it
	if err != nil {
		return WaitlistEntry{}, nil, err
	}
	if err := s.Repo.SetPatient(ctx, c.PatientID); err != nil {
		return WaitlistEntry{}, nil, err
	}
	clinic, err := s.Clinics.Read(ctx, organization.Caller{OrganizationID: org}, org)
	switch {
	case errors.Is(err, organization.ErrNotFound):
		return WaitlistEntry{}, ErrWaitlistOff, nil
	case err != nil:
		return WaitlistEntry{}, nil, err
	case !clinic.Listed || !set.WaitlistEnabled:
		return WaitlistEntry{}, ErrWaitlistOff, nil
	}
	loc, err := time.LoadLocation(clinic.TimeZone)
	if err != nil {
		return WaitlistEntry{}, nil, fmt.Errorf("the clinic's time zone: %w", err)
	}
	now := s.Clock.Now()
	today := dateOf(now.In(loc))
	if in.Earliest.Before(today) || in.Latest.After(today.AddDate(0, 0, maxWaitlistDays)) {
		return WaitlistEntry{}, ErrInvalidEntry, nil
	}
	if refused, err := s.activeRole(ctx, org, in.PractitionerRoleID); refused != nil || err != nil {
		return WaitlistEntry{}, refused, err
	}
	consent, err := s.Consents.State(ctx, c.PatientID, org)
	if err != nil {
		return WaitlistEntry{}, nil, err
	}
	if !consent.Active {
		return WaitlistEntry{}, ErrConsentRequired, nil
	}
	n, err := s.Waitlist.CountActiveEntries(ctx, c.PatientID, org, now)
	if err != nil {
		return WaitlistEntry{}, nil, err
	}
	if n >= maxActiveEntries {
		return WaitlistEntry{}, ErrTooManyEntries, nil
	}
	out, err = s.Waitlist.InsertEntry(ctx, WaitlistEntry{
		Created: now, EndsAt: time.Date(in.Latest.Year(), in.Latest.Month(), in.Latest.Day()+1, 0, 0, 0, 0, loc),
		Earliest: in.Earliest, Latest: in.Latest, PatientID: c.PatientID, OrganizationID: org,
		PractitionerRoleID: in.PractitionerRoleID, Specialty: in.Specialty, Times: in.Times, Status: WaitlistActive,
	})
	return out, nil, err
}

// activeRole refuses with ErrInvalidEntry a role that is not an active one of the clinic org; none ("") passes.
func (s *Service) activeRole(ctx context.Context, org, roleID string) (refused, err error) {
	if roleID == "" {
		return nil, nil
	}
	role, err := s.Roles.Lookup(ctx, org, roleID)
	switch {
	case errors.Is(err, practitionerrole.ErrNotFound):
		return ErrInvalidEntry, nil
	case err != nil:
		return nil, err
	case !role.Active:
		return ErrInvalidEntry, nil
	}
	return nil, nil
}

// MyWaitlist returns the calling patient's entries, newest first, each with its open offer; an active entry past its
// latest day is expired. Each entry returned is audited (one event without an entity for none). ErrForbidden to staff.
func (s *Service) MyWaitlist(ctx context.Context, c Caller) ([]WaitlistEntry, error) {
	if !c.patient() {
		return nil, s.refuse(ctx, c, auditevent.CodeSearchType, ErrForbidden)
	}
	var out []WaitlistEntry
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		out = nil
		if c.PatientID != "" {
			if err := s.Repo.SetPatient(ctx, c.PatientID); err != nil {
				return err
			}
			all, err := s.Waitlist.PatientEntries(ctx, c.PatientID)
			if err != nil {
				return err
			}
			for _, e := range all {
				if e.PatientID == c.PatientID { // not on row-level security alone
					out = append(out, expired(e, s.Clock.Now()))
				}
			}
		}
		return s.recordEntries(ctx, c, out)
	})
	if err != nil {
		return nil, fmt.Errorf("appointment: my waitlist: %w", err)
	}
	return out, nil
}

// expired is e with its status expired when it is active and its latest day is over at now.
func expired(e WaitlistEntry, now time.Time) WaitlistEntry {
	if e.Status == WaitlistActive && !e.EndsAt.After(now) {
		e.Status = WaitlistExpired
	}
	return e
}

// recordEntries audits each entry read by c, or one search without an entity for none.
func (s *Service) recordEntries(ctx context.Context, c Caller, entries []WaitlistEntry) error {
	if len(entries) == 0 {
		return s.record(ctx, c, auditevent.CodeSearchType, "", "", auditevent.OutcomeSuccess)
	}
	for _, e := range entries {
		if err := s.record(ctx, c, auditevent.CodeSearchType, "Appointment", e.ID, auditevent.OutcomeSuccess); err != nil {
			return err
		}
	}
	return nil
}

// ClinicWaitlist returns the waitlist of the staff caller's clinic in offer order (priority descending, then join
// time), each entry with its open offer and its patient's name where access.Profile gives it; with the filter's role,
// the entries it could be offered (its own and its specialty's), with its status, those in it. Each entry returned and
// each name given is audited. ErrForbidden to a patient.
func (s *Service) ClinicWaitlist(ctx context.Context, c Caller, f WaitlistFilter) ([]ClinicEntry, error) {
	if c.patient() {
		return nil, s.refuse(ctx, c, auditevent.CodeSearchType, ErrForbidden)
	}
	var out []ClinicEntry
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		var err error
		out, err = s.clinicWaitlist(ctx, c, f)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("appointment: clinic waitlist: %w", err)
	}
	return out, nil
}

func (s *Service) clinicWaitlist(ctx context.Context, c Caller, f WaitlistFilter) ([]ClinicEntry, error) {
	org := c.OrganizationID
	if err := s.Repo.SetClinic(ctx, org); err != nil {
		return nil, err
	}
	specialty := ""
	if f.PractitionerRoleID != "" {
		role, err := s.Roles.Lookup(ctx, org, f.PractitionerRoleID)
		switch {
		case errors.Is(err, practitionerrole.ErrNotFound):
			return nil, s.record(ctx, c, auditevent.CodeSearchType, "", "", auditevent.OutcomeSuccess)
		case err != nil:
			return nil, err
		}
		specialty = role.SpecialtyCode
	}
	all, err := s.Waitlist.ClinicEntries(ctx, org, f.PractitionerRoleID, specialty)
	if err != nil {
		return nil, err
	}
	n := namer{s: s, c: c, facts: map[string]access.Facts{}}
	var out []ClinicEntry
	for _, e := range all {
		e = expired(e, s.Clock.Now())
		if e.OrganizationID != org || (f.Status != "" && e.Status != f.Status) {
			continue
		}
		ce, err := n.name(ctx, e)
		if err != nil {
			return nil, err
		}
		out = append(out, ce)
	}
	if len(out) == 0 {
		return nil, s.record(ctx, c, auditevent.CodeSearchType, "", "", auditevent.OutcomeSuccess)
	}
	return out, nil
}

// namer gives a clinic's entries their patients' names as access.Profile allows the staff caller, reading each
// patient's facts once.
type namer struct {
	s      *Service
	facts  map[string]access.Facts
	c      Caller
	viewer access.Viewer
	ready  bool
}

// name audits the entry e and, where the caller may see it, its patient's name.
func (n *namer) name(ctx context.Context, e WaitlistEntry) (ClinicEntry, error) {
	if err := n.s.record(ctx, n.c, auditevent.CodeSearchType, "Appointment", e.ID, auditevent.OutcomeSuccess); err != nil {
		return ClinicEntry{}, err
	}
	if !n.ready {
		v, err := n.s.ownViewer(ctx, n.c)
		if err != nil {
			return ClinicEntry{}, err
		}
		n.viewer, n.ready = v, true
	}
	f, ok := n.facts[e.PatientID]
	if !ok {
		var err error
		if f, err = n.s.facts(ctx, e.PatientID, n.c.OrganizationID); err != nil {
			return ClinicEntry{}, err
		}
		n.facts[e.PatientID] = f
	}
	out := ClinicEntry{Entry: e}
	if access.Profile(n.viewer, f) == access.None {
		return out, nil
	}
	p, err := n.s.Patients.Lookup(ctx, e.PatientID)
	if err != nil {
		return ClinicEntry{}, err
	}
	out.Patient, out.Named = Name{Given: p.GivenName, Family: p.FamilyName}, true
	return out, n.s.record(ctx, n.c, auditevent.CodeSearchType, "Patient", p.ID, auditevent.OutcomeSuccess)
}

// ownViewer is staff caller c as access sees them, with their own role in the clinic for a practitioner.
func (s *Service) ownViewer(ctx context.Context, c Caller) (access.Viewer, error) {
	if c.Admin || c.PractitionerID == "" {
		return s.staffViewer(ctx, c, "")
	}
	role, err := s.Roles.OfPractitioner(ctx, c.OrganizationID, c.PractitionerID)
	switch {
	case errors.Is(err, practitionerrole.ErrNotFound):
		return s.staffViewer(ctx, c, "")
	case err != nil:
		return access.Viewer{}, err
	}
	return s.staffViewer(ctx, c, role.ID)
}

// Withdraw ends the active entry id: the patient their own, the clinic's staff one of the clinic; its open offer is
// withdrawn and its slot handed on (waitlist_slot_freed). Audited, refusals too. ErrNotFound for an entry the caller
// may not change.
func (s *Service) Withdraw(ctx context.Context, c Caller, id string) error {
	var refused error
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		var err error
		if refused, err = s.withdraw(ctx, c, id); err != nil {
			return err
		}
		if refused != nil {
			return s.record(ctx, c, auditevent.CodeUpdate, "", "", auditevent.OutcomeMinorFailure)
		}
		return s.record(ctx, c, auditevent.CodeUpdate, "Appointment", id, auditevent.OutcomeSuccess)
	})
	switch {
	case err != nil:
		return fmt.Errorf("appointment: withdraw from the waitlist: %w", err)
	case refused != nil:
		return refused
	}
	return nil
}

func (s *Service) withdraw(ctx context.Context, c Caller, id string) (refused, err error) {
	e, refused, err := s.entryOf(ctx, c, id)
	if refused != nil || err != nil {
		return refused, err
	}
	now := s.Clock.Now()
	if _, err := s.Waitlist.CloseEntry(ctx, e.ID, WaitlistWithdrawn, now); err != nil {
		return nil, err
	}
	return nil, s.closeOpenOffer(ctx, e.ID, OfferWithdrawn, now)
}

// entryOf returns the active entry id, locked, of the patient caller or of the staff caller's clinic, in its scope;
// ErrNotFound as the refusal for any other.
func (s *Service) entryOf(ctx context.Context, c Caller, id string) (out WaitlistEntry, refused, err error) {
	switch {
	case c.patient() && c.PatientID == "":
		return WaitlistEntry{}, ErrNotFound, nil
	case c.patient():
		err = s.Repo.SetPatient(ctx, c.PatientID)
	default:
		err = s.Repo.SetClinic(ctx, c.OrganizationID)
	}
	if err != nil {
		return WaitlistEntry{}, nil, err
	}
	e, err := s.Waitlist.EntryForUpdate(ctx, id)
	switch {
	case errors.Is(err, ErrNotFound):
		return WaitlistEntry{}, ErrNotFound, nil
	case err != nil:
		return WaitlistEntry{}, nil, err
	case c.patient() && e.PatientID != c.PatientID, !c.patient() && e.OrganizationID != c.OrganizationID, e.Status != WaitlistActive:
		return WaitlistEntry{}, ErrNotFound, nil // not on row-level security alone
	}
	return e, nil, nil
}

// closeOpenOffer closes the open offer of the entry, if any, with state and hands its slot on.
func (s *Service) closeOpenOffer(ctx context.Context, entryID string, state OfferState, now time.Time) error {
	o, open, err := s.Waitlist.OpenOffer(ctx, "", entryID)
	if err != nil || !open {
		return err
	}
	if _, err := s.Waitlist.CloseOffer(ctx, o.ID, state, "", now); err != nil {
		return err
	}
	return s.Jobs.EnqueueSlotFreed(ctx, o.SlotID, o.OrganizationID)
}

// SetPriority moves the active entry id of the staff caller's clinic to priority (0 to 100) and returns it as the
// clinic sees it. Audited, refusals too. ErrInvalidPriority (unaudited, before any access), ErrForbidden to a patient,
// ErrNotFound for an entry the clinic does not see.
func (s *Service) SetPriority(ctx context.Context, c Caller, id string, priority int) (ClinicEntry, error) {
	if priority < 0 || priority > maxPriority {
		return ClinicEntry{}, ErrInvalidPriority
	}
	if c.patient() {
		return ClinicEntry{}, s.refuse(ctx, c, auditevent.CodeUpdate, ErrForbidden)
	}
	var out ClinicEntry
	var refused error
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		e, refusal, err := s.entryOf(ctx, c, id)
		if refused = refusal; err != nil || refused != nil {
			if err == nil {
				err = s.record(ctx, c, auditevent.CodeUpdate, "", "", auditevent.OutcomeMinorFailure)
			}
			return err
		}
		if e, err = s.Waitlist.SetPriority(ctx, e.ID, priority, s.Clock.Now()); err != nil {
			return err
		}
		if err := s.record(ctx, c, auditevent.CodeUpdate, "Appointment", e.ID, auditevent.OutcomeSuccess); err != nil {
			return err
		}
		n := namer{s: s, c: c, facts: map[string]access.Facts{}}
		out, err = n.name(ctx, e)
		return err
	})
	switch {
	case err != nil:
		return ClinicEntry{}, fmt.Errorf("appointment: waitlist priority: %w", err)
	case refused != nil:
		return ClinicEntry{}, refused
	}
	return out, nil
}

// WithdrawForRevokedConsent withdraws, after the patient revoked their care consent with the clinic org (the
// consent_revoked job), their active entries there and their open offers, handing each offered slot on. In one
// transaction in the patient's scope, audited with the job as agent; it returns how many entries it withdrew. Nothing
// happens while the consent is active (given again).
func (s *Service) WithdrawForRevokedConsent(ctx context.Context, patientID, org string) (int, error) {
	var n int
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		n = 0
		if err := s.Repo.SetPatient(ctx, patientID); err != nil {
			return err
		}
		if err := s.Repo.SetClinic(ctx, org); err != nil {
			return err
		}
		state, err := s.Consents.State(ctx, patientID, org)
		if err != nil || state.Active {
			return err
		}
		entries, err := s.Waitlist.ActiveEntries(ctx, patientID, org)
		if err != nil {
			return err
		}
		now := s.Clock.Now()
		for _, e := range entries {
			if e.PatientID != patientID || e.OrganizationID != org {
				continue
			}
			if _, err := s.Waitlist.CloseEntry(ctx, e.ID, WaitlistWithdrawn, now); err != nil {
				return err
			}
			if err := s.closeOpenOffer(ctx, e.ID, OfferWithdrawn, now); err != nil {
				return err
			}
			if err := s.recordJob(ctx, agentRevocation, org, auditevent.CodeUpdate, "Appointment", e.ID, auditevent.OutcomeSuccess); err != nil {
				return err
			}
			n++
		}
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("appointment: withdraw the waitlist for a revoked consent: %w", err)
	}
	return n, nil
}

// dateOf is t's calendar date in t's zone, at midnight UTC (a date, not an instant).
func dateOf(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// WaitlistPurger applies the retention rules of waitlist_entry and waitlist_offer (internal/retention): an entry 30
// days after it closed or ended, with its offers.
type WaitlistPurger struct {
	repo   WaitlistRepository
	offers bool
}

// NewWaitlistEntryPurger returns the purger of waitlist_entry on repo, built on the retention role's pool.
func NewWaitlistEntryPurger(repo WaitlistRepository) *WaitlistPurger { return &WaitlistPurger{repo: repo} }

// NewWaitlistOfferPurger returns the purger of waitlist_offer on repo: the offers of the entries the entry purger
// deletes (the foreign key also deletes them with their entry).
func NewWaitlistOfferPurger(repo WaitlistRepository) *WaitlistPurger {
	return &WaitlistPurger{repo: repo, offers: true}
}

// Purge deletes the rows whose period ended before before and returns how many.
func (p *WaitlistPurger) Purge(ctx context.Context, before time.Time) (int64, error) {
	purge := p.repo.DeleteEntriesClosedBefore
	if p.offers {
		purge = p.repo.DeleteOffersClosedBefore
	}
	n, err := purge(ctx, before)
	if err != nil {
		return 0, fmt.Errorf("appointment: purge the waitlist: %w", err)
	}
	return n, nil
}
