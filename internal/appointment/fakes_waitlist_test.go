package appointment_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/internal/slot"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

// fakeWaitlist holds entries and offers, showing them as row-level security does (the patient's own; the clinic's
// entries with the consent, every offer of the clinic). Candidates gives the entries of cands that have no open offer
// and were never offered the slot: the matching itself is the query's, tested on Postgres.
type fakeWaitlist struct {
	err      error
	repo     *fakeRepository
	consents *fakeConsents
	entries  []appointment.WaitlistEntry
	offers   []appointment.Offer
	cands    []string // the ids Candidates gives, in order
	matched  []appointment.Match
	queries  []appointment.ClinicQuery
	locked   []string // "<patient> <clinic>" of each LockEntries
}

func (w *fakeWaitlist) inTx(ctx context.Context) error {
	if ctx.Value(txKey{}) == nil {
		return errors.New("the waitlist outside a transaction")
	}
	return w.err
}

func (w *fakeWaitlist) visible(e appointment.WaitlistEntry) bool {
	if w.repo.patient != "" {
		return e.PatientID == w.repo.patient
	}
	return w.repo.clinic != "" && e.OrganizationID == w.repo.clinic && w.consents.state[e.PatientID+e.OrganizationID].Active
}

func (w *fakeWaitlist) offerVisible(o appointment.Offer) bool {
	if w.repo.patient != "" {
		return o.PatientID == w.repo.patient
	}
	return w.repo.clinic != "" && o.OrganizationID == w.repo.clinic
}

func (w *fakeWaitlist) InsertEntry(ctx context.Context, e appointment.WaitlistEntry) (appointment.WaitlistEntry, error) {
	if err := w.inTx(ctx); err != nil {
		return appointment.WaitlistEntry{}, err
	}
	if w.repo.patient != e.PatientID {
		return appointment.WaitlistEntry{}, errors.New("an entry inserted outside its patient's scope")
	}
	e.ID = fmt.Sprintf("e%07d-0000-4000-8000-000000000000", len(w.entries)+1)
	w.entries = append(w.entries, e)
	return e, nil
}

func (w *fakeWaitlist) CountActiveEntries(ctx context.Context, patientID, org string, now time.Time) (int, error) {
	n := 0
	for _, e := range w.entries {
		if e.PatientID == patientID && e.OrganizationID == org && e.Status == appointment.WaitlistActive && e.EndsAt.After(now) && w.visible(e) {
			n++
		}
	}
	return n, w.inTx(ctx)
}

func (w *fakeWaitlist) withOffer(e appointment.WaitlistEntry) appointment.WaitlistEntry {
	for _, o := range w.offers {
		if o.EntryID == e.ID && o.State == appointment.OfferOpen {
			e.Offer = o
		}
	}
	return e
}

func (w *fakeWaitlist) PatientEntries(ctx context.Context, patientID string) ([]appointment.WaitlistEntry, error) {
	var out []appointment.WaitlistEntry
	for i := len(w.entries) - 1; i >= 0; i-- {
		if e := w.entries[i]; e.PatientID == patientID && w.visible(e) {
			out = append(out, w.withOffer(e))
		}
	}
	return out, w.inTx(ctx)
}

func (w *fakeWaitlist) ClinicEntries(ctx context.Context, q appointment.ClinicQuery) ([]appointment.WaitlistEntry, error) {
	w.queries = append(w.queries, q)
	var out []appointment.WaitlistEntry
	for _, e := range w.entries {
		status := e.Status
		if status == appointment.WaitlistActive && !e.EndsAt.After(q.Now) {
			status = appointment.WaitlistExpired
		}
		if e.OrganizationID == q.OrganizationID && w.visible(e) && status == q.Status &&
			(q.PractitionerRoleID == "" || e.PractitionerRoleID == q.PractitionerRoleID || (q.Specialty != "" && e.Specialty == q.Specialty)) {
			out = append(out, w.withOffer(e))
		}
	}
	if q.Offset >= len(out) {
		return nil, w.inTx(ctx)
	}
	out = out[q.Offset:]
	return out[:min(len(out), q.Limit)], w.inTx(ctx)
}

func (w *fakeWaitlist) LockEntries(ctx context.Context, patientID, org string) error {
	w.locked = append(w.locked, patientID+" "+org)
	return w.inTx(ctx)
}

func (w *fakeWaitlist) find(id string) (int, bool) {
	for i, e := range w.entries {
		if e.ID == id && w.visible(e) {
			return i, true
		}
	}
	return 0, false
}

func (w *fakeWaitlist) EntryForUpdate(ctx context.Context, id string) (appointment.WaitlistEntry, error) {
	if err := w.inTx(ctx); err != nil {
		return appointment.WaitlistEntry{}, err
	}
	if i, ok := w.find(id); ok {
		return w.entries[i], nil
	}
	return appointment.WaitlistEntry{}, appointment.ErrNotFound
}

func (w *fakeWaitlist) update(ctx context.Context, id string, fn func(*appointment.WaitlistEntry)) (appointment.WaitlistEntry, error) {
	if err := w.inTx(ctx); err != nil {
		return appointment.WaitlistEntry{}, err
	}
	i, ok := w.find(id)
	if !ok || w.entries[i].Status != appointment.WaitlistActive {
		return appointment.WaitlistEntry{}, appointment.ErrNotFound
	}
	fn(&w.entries[i])
	return w.entries[i], nil
}

func (w *fakeWaitlist) CloseEntry(ctx context.Context, id string, status appointment.WaitlistStatus, at time.Time) (appointment.WaitlistEntry, error) {
	return w.update(ctx, id, func(e *appointment.WaitlistEntry) { e.Status, e.ClosedAt = status, at })
}

func (w *fakeWaitlist) SetPriority(ctx context.Context, id string, priority int, _ time.Time) (appointment.WaitlistEntry, error) {
	return w.update(ctx, id, func(e *appointment.WaitlistEntry) { e.Priority = priority })
}

func (w *fakeWaitlist) ActiveEntries(ctx context.Context, patientID, org string) ([]appointment.WaitlistEntry, error) {
	var out []appointment.WaitlistEntry
	for _, e := range w.entries {
		if e.PatientID == patientID && e.OrganizationID == org && e.Status == appointment.WaitlistActive && w.visible(e) {
			out = append(out, e)
		}
	}
	return out, w.inTx(ctx)
}

func (w *fakeWaitlist) Candidates(ctx context.Context, m appointment.Match, n int) ([]appointment.WaitlistEntry, error) {
	w.matched = append(w.matched, m)
	var out []appointment.WaitlistEntry
	for _, id := range w.cands {
		i, ok := w.find(id)
		if !ok || len(out) == n {
			continue
		}
		e, offered := w.entries[i], slices.Contains(m.Skip, id)
		for _, o := range w.offers {
			offered = offered || o.EntryID == id && (o.State == appointment.OfferOpen || o.SlotID == m.SlotID)
		}
		if !offered && e.Status == appointment.WaitlistActive {
			out = append(out, e)
		}
	}
	return out, w.inTx(ctx)
}

func (w *fakeWaitlist) InsertOffer(ctx context.Context, o appointment.Offer) (appointment.Offer, error) {
	if err := w.inTx(ctx); err != nil {
		return appointment.Offer{}, err
	}
	for _, x := range w.offers {
		if x.State == appointment.OfferOpen && (x.SlotID == o.SlotID || x.EntryID == o.EntryID) {
			return appointment.Offer{}, errors.New("a second open offer")
		}
	}
	o.ID, o.State = fmt.Sprintf("0f%06d-0000-4000-8000-000000000000", len(w.offers)+1), appointment.OfferOpen
	w.offers = append(w.offers, o)
	return o, nil
}

func (w *fakeWaitlist) Offer(ctx context.Context, id string) (appointment.Offer, error) {
	if err := w.inTx(ctx); err != nil {
		return appointment.Offer{}, err
	}
	for _, o := range w.offers {
		if o.ID == id && w.offerVisible(o) {
			return o, nil
		}
	}
	return appointment.Offer{}, appointment.ErrNotFound
}

func (w *fakeWaitlist) OfferForUpdate(ctx context.Context, id string) (appointment.Offer, error) {
	return w.Offer(ctx, id)
}

func (w *fakeWaitlist) OpenOffer(ctx context.Context, slotID, entryID string) (appointment.Offer, bool, error) {
	for _, o := range w.offers {
		if o.State == appointment.OfferOpen && w.offerVisible(o) && (slotID == "" || o.SlotID == slotID) && (entryID == "" || o.EntryID == entryID) {
			return o, true, w.inTx(ctx)
		}
	}
	return appointment.Offer{}, false, w.inTx(ctx)
}

func (w *fakeWaitlist) CloseOffer(ctx context.Context, id string, state appointment.OfferState, appointmentID string, _ time.Time) (appointment.Offer, error) {
	if err := w.inTx(ctx); err != nil {
		return appointment.Offer{}, err
	}
	for i, o := range w.offers {
		if o.ID == id && w.offerVisible(o) && o.State == appointment.OfferOpen {
			w.offers[i].State, w.offers[i].AppointmentID = state, appointmentID
			return w.offers[i], nil
		}
	}
	return appointment.Offer{}, appointment.ErrNotFound
}

func (w *fakeWaitlist) DeleteEntriesClosedBefore(context.Context, time.Time) (int64, error) {
	return 2, w.err
}
func (w *fakeWaitlist) DeleteOffersClosedBefore(context.Context, time.Time) (int64, error) {
	return 3, w.err
}

// RecordAll keeps the events as Record does, in a transaction only.
func (a *fakeAuditor) RecordAll(ctx context.Context, es []auditevent.AuditEvent) error {
	for _, e := range es {
		if err := a.Record(ctx, e); err != nil {
			return err
		}
	}
	return nil
}

// fakeWaitlistRules gives each clinic's appointment settings: the defaults with the waitlist on, unless off.
type fakeWaitlistRules struct {
	err error
	off map[string]bool
}

func (s *fakeWaitlistRules) For(_ context.Context, org string) (organization.AppointmentSettings, error) {
	set := organization.DefaultAppointmentSettings()
	set.WaitlistEnabled = !s.off[org]
	return set, s.err
}

// fakeOfferedSlots holds the statuses of the slots the waitlist moves.
type fakeOfferedSlots struct {
	err    error
	slots  map[string]slot.Bookable
	locked []string
}

func (s *fakeOfferedSlots) ForWaitlist(ctx context.Context, org, id string) (slot.Bookable, error) {
	if ctx.Value(txKey{}) == nil {
		return slot.Bookable{}, errors.New("ForWaitlist outside a transaction")
	}
	b, ok := s.slots[id]
	if !ok || b.OrganizationID != org {
		return slot.Bookable{}, slot.ErrUnavailable
	}
	s.locked = append(s.locked, id)
	return b, s.err
}

func (s *fakeOfferedSlots) move(ctx context.Context, org, id string, from, to slot.Status) (slot.Slot, error) {
	if ctx.Value(txKey{}) == nil {
		return slot.Slot{}, errors.New("a slot moved outside a transaction")
	}
	b, ok := s.slots[id]
	if !ok || b.OrganizationID != org || b.Status != from {
		return slot.Slot{}, slot.ErrUnavailable
	}
	b.Status = to
	s.slots[id] = b
	return b.Slot, s.err
}

func (s *fakeOfferedSlots) Offer(ctx context.Context, org, id string) (slot.Slot, error) {
	return s.move(ctx, org, id, slot.StatusFree, slot.StatusTentative)
}

func (s *fakeOfferedSlots) TakeOffered(ctx context.Context, org, id string) (slot.Slot, error) {
	return s.move(ctx, org, id, slot.StatusTentative, slot.StatusBusy)
}

func (s *fakeOfferedSlots) Unoffer(ctx context.Context, org, id string) (bool, error) {
	_, err := s.move(ctx, org, id, slot.StatusTentative, slot.StatusFree)
	if errors.Is(err, slot.ErrUnavailable) {
		return false, nil
	}
	return err == nil, err
}

func (s *fakeOfferedSlots) status(id string) slot.Status { return s.slots[id].Status }

// fakeOfferJobs records the expiries and notices into the fixture's jobs: "offer_expire <offer> <clinic> at <time>",
// "offer_notify <offer> <clinic> <account>".
type fakeOfferJobs struct{ jobs *fakeJobs }

func (j fakeOfferJobs) EnqueueOfferExpire(ctx context.Context, id, org string, at time.Time) error {
	return j.jobs.add(ctx, fmt.Sprintf("offer_expire %s %s at %s", id, org, at.Format(time.RFC3339)))
}

func (j fakeOfferJobs) EnqueueHandOn(ctx context.Context, slotID, org string, at time.Time) error {
	return j.jobs.add(ctx, fmt.Sprintf("hand_on %s %s at %s", slotID, org, at.Format(time.RFC3339)))
}

func (j fakeOfferJobs) NotifyOffer(ctx context.Context, o appointment.Offer, accountID string) error {
	return j.jobs.add(ctx, "offer_notify "+o.ID+" "+o.OrganizationID+" "+accountID)
}

// fakeReach reaches every account but those unreachable, and says the accounts in quiet are inside their quiet hours
// until quietUntil; it keeps the agents it was asked as.
type fakeReach struct {
	quiet       map[string]bool
	unreachable map[string]bool
	quietUntil  map[string]time.Time
	agents      *[]string
}

func (r fakeReach) Reach(_ context.Context, agent, _, accountID string, _ time.Time) (appointment.Reach, error) {
	*r.agents = append(*r.agents, agent)
	return appointment.Reach{Reachable: !r.unreachable[accountID], Quiet: r.quiet[accountID], QuietUntil: r.quietUntil[accountID]}, nil
}

// waitlistFixture is the fixture with the waitlist's ports: patient A's and B's active consents with clinic A, and the
// slots the waitlist moves (slotFree of role A1, at start).
type waitlistFixture struct {
	*fixture
	wl       *fakeWaitlist
	settings *fakeWaitlistRules
	offered  *fakeOfferedSlots
	quiet    fakeReach
	ports    appointment.Ports
}

func newWaitlistFixture(at time.Time) *waitlistFixture {
	f := newFixture()
	f.consents.state[patientB+clinicA] = f.consents.state[patientA+clinicA]
	w := &waitlistFixture{
		fixture:  f,
		wl:       &fakeWaitlist{repo: f.repo, consents: f.consents},
		settings: &fakeWaitlistRules{off: map[string]bool{}},
		offered:  &fakeOfferedSlots{slots: map[string]slot.Bookable{slotFree: slotAt(slotFree, roleA1, clinicA, start)}},
		quiet:    fakeReach{quiet: map[string]bool{}, unreachable: map[string]bool{}, quietUntil: map[string]time.Time{}, agents: new([]string)},
	}
	jobs := fakeOfferJobs{jobs: f.jobs}
	w.ports = appointment.Ports{
		Repo: f.repo, Audit: f.audit, Tx: fakeTransactor{f: f}, Clock: clock.Fixed{T: at},
		Consents: f.consents, Slots: f.slots, Patients: f.patients, Clinics: f.clinics, Roles: f.roles, Jobs: f.jobs, Stats: f.stats,
		Waitlist: w.wl, Rules: w.settings, Reminders: f.reminders, Offered: w.offered, OfferJobs: jobs, Notices: jobs, Reach: w.quiet,
	}
	f.svc = appointment.NewService(w.ports)
	return w
}

// with rebuilds the service with change applied to its ports.
func (w *waitlistFixture) with(change func(p *appointment.Ports)) *waitlistFixture {
	change(&w.ports)
	w.svc = appointment.NewService(w.ports)
	return w
}

// entry adds an active entry of the patient for the role of clinic A from the day of now to that of start.
func (w *waitlistFixture) entry(patientID, role string, priority int) string {
	id := fmt.Sprintf("e%07d-0000-4000-8000-000000000000", len(w.wl.entries)+1)
	w.wl.entries = append(w.wl.entries, appointment.WaitlistEntry{
		ID: id, PatientID: patientID, OrganizationID: clinicA, PractitionerRoleID: role, Priority: priority, Status: appointment.WaitlistActive,
		Earliest: day(now), Latest: day(start), EndsAt: day(start).AddDate(0, 0, 1), Created: now.Add(-time.Hour),
	})
	return id
}

func day(t time.Time) time.Time { return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC) }
