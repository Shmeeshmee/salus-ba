package careplan

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/chargeitemdefinition"
	"github.com/Shmeeshmee/salus/internal/schedule"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

// MaxSteps is the most steps a plan has (SaveCarePlanRequest).
const MaxSteps = 50

// Transactor runs fn in one transaction (postgres.Transactor).
type Transactor interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// Auditor records an AuditEvent inside the caller's transaction (auditevent.Service).
type Auditor interface {
	Record(ctx context.Context, e auditevent.AuditEvent) error
}

// Consents gives the state of a patient's consents with a clinic in the caller's transaction (consent.Service).
type Consents interface {
	State(ctx context.Context, patientID, organizationID string) (access.Consent, error)
}

// Prices gives the clinic's price list (chargeitemdefinition.Service).
type Prices interface {
	OfClinic(ctx context.Context, org string) ([]chargeitemdefinition.Definition, error)
}

// Visits books, reads and cancels a plan's visits in the plan's transaction (appointment.Service).
type Visits interface {
	BookPlan(ctx context.Context, c appointment.Caller, patientID string, steps []appointment.StaffBooking) ([]appointment.Appointment, error)
	PlanVisits(ctx context.Context, org string, ids []string) ([]appointment.Appointment, error)
	CancelPlan(ctx context.Context, c appointment.Caller, ids []string) error
}

// Ports are what the Service consumes.
type Ports struct {
	Repo     Repository
	Tx       Transactor
	Audit    Auditor
	Consents Consents
	Prices   Prices
	Visits   Visits
	Clock    clock.Clock
}

// Service is the treatment plans' business rules (docs/clinic-cms/WAVE6.md U6): the clinic's admin and its
// practitioners lay out, book and cancel a plan for a patient with the clinic's active care consent; a receptionist is
// refused (no clinical content); the patient reads their booked plans. Every read and write is audited, a refusal too.
type Service struct {
	p Ports
}

// NewService builds a Service on p.
func NewService(p Ports) *Service {
	return &Service{p: p}
}

// refusals are the errors a caller is answered with; any other is a failure.
var refusals = []error{ErrInvalid, ErrNotFound, ErrForbidden, ErrNotDraft, ErrEnded, ErrVersionConflict, ErrSlotUnavailable}

// Clinical reports whether c may work with the clinic's plans: its admin or a practitioner, never a receptionist.
func (c Staff) Clinical() bool {
	return c.OrganizationID != "" && !c.Reception && (c.Admin || c.Practitioner)
}

// Write writes a new draft for a patient of the staff's clinic with its active care consent, from its own price list,
// with its quote (one of the patient's at the clinic, or none). ErrForbidden, ErrInvalid, ErrNotFound; audited.
func (s *Service) Write(ctx context.Context, c Staff, d Draft) (Plan, error) {
	return s.staffTx(ctx, c, auditevent.CodeCreate, func(ctx context.Context) (Plan, error) {
		steps, err := s.checkDraft(ctx, c, d.PatientID, d)
		if err != nil {
			return Plan{}, err
		}
		return s.p.Repo.Insert(ctx, Plan{
			OrganizationID: c.OrganizationID, PatientID: d.PatientID, Status: StatusDraft, Title: strings.TrimSpace(d.Title),
			QuoteID: d.QuoteID, Steps: steps,
		}, c.AccountID)
	})
}

// Change changes a draft at its version: its title, quote and steps. ErrNotDraft once booked or ended,
// ErrVersionConflict, and Write's refusals; audited.
func (s *Service) Change(ctx context.Context, c Staff, id string, d Draft) (Plan, error) {
	return s.staffTx(ctx, c, auditevent.CodeUpdate, func(ctx context.Context) (Plan, error) {
		p, err := s.current(ctx, id, d.Version)
		if err != nil {
			return Plan{}, err
		}
		if p.Status != StatusDraft {
			return Plan{}, ErrNotDraft
		}
		if p.Steps, err = s.checkDraft(ctx, c, p.PatientID, d); err != nil {
			return Plan{}, err
		}
		p.Title, p.QuoteID = strings.TrimSpace(d.Title), d.QuoteID
		return s.store(ctx, c, p)
	})
}

// Book books every step of a draft in one transaction, or none: one free slot of the clinic per step, in the steps'
// order, each in its step's mode. The plan becomes active and its patient sees it. ErrInvalid for a slot count or
// mode that does not match, ErrSlotUnavailable (nothing booked), ErrNotDraft, ErrVersionConflict; audited.
func (s *Service) Book(ctx context.Context, c Staff, id string, b Booking) (Plan, error) {
	return s.staffTx(ctx, c, auditevent.CodeUpdate, func(ctx context.Context) (Plan, error) {
		p, err := s.current(ctx, id, b.Version)
		if err != nil {
			return Plan{}, err
		}
		if p.Status != StatusDraft {
			return Plan{}, ErrNotDraft
		}
		if len(b.SlotIDs) != len(p.Steps) {
			return Plan{}, ErrInvalid
		}
		bookings := make([]appointment.StaffBooking, len(p.Steps))
		for i, st := range p.Steps {
			bookings[i] = appointment.StaffBooking{SlotID: b.SlotIDs[i], ServiceID: st.ServiceID}
		}
		visits, err := s.p.Visits.BookPlan(ctx, visitCaller(c), p.PatientID, bookings)
		if err != nil {
			return Plan{}, bookingErr(err)
		}
		for i, v := range visits {
			if string(v.Mode) != p.Steps[i].VisitMode {
				return Plan{}, ErrInvalid
			}
			p.Steps[i].AppointmentID = v.ID
		}
		now, day := s.p.Clock.Now(), b.StartsOn
		p.Status, p.BookedAt, p.StartsOn = StatusActive, &now, &day
		return s.store(ctx, c, p)
	})
}

// Cancel revokes a draft or an active plan at its version and cancels its steps' visits that are still ahead (by the
// clinic); past ones are kept. ErrEnded for a completed or revoked plan, ErrVersionConflict; audited.
func (s *Service) Cancel(ctx context.Context, c Staff, id string, version int) (Plan, error) {
	return s.staffTx(ctx, c, auditevent.CodeUpdate, func(ctx context.Context) (Plan, error) {
		p, err := s.current(ctx, id, version)
		if err != nil {
			return Plan{}, err
		}
		if p, err = s.settle(ctx, c.OrganizationID, p); err != nil {
			return Plan{}, err
		}
		if p.Status != StatusDraft && p.Status != StatusActive {
			return Plan{}, ErrEnded
		}
		if err := s.p.Visits.CancelPlan(ctx, visitCaller(c), visitIDs(p)); err != nil {
			return Plan{}, bookingErr(err) // a visit it may not cancel: nothing is
		}
		now := s.p.Clock.Now()
		p.Status, p.RevokedAt = StatusRevoked, &now
		return s.store(ctx, c, p)
	})
}

// Read returns one plan of the staff's clinic, its steps in the order of their visits once booked; audited.
func (s *Service) Read(ctx context.Context, c Staff, id string) (Plan, error) {
	return s.staffTx(ctx, c, auditevent.CodeRead, func(ctx context.Context) (Plan, error) {
		p, err := s.p.Repo.ByID(ctx, id)
		if err != nil {
			return Plan{}, err
		}
		return s.settle(ctx, c.OrganizationID, p)
	})
}

// ForPatient returns a patient's plans at the staff's clinic, drafts included, newest first; each is audited as read.
func (s *Service) ForPatient(ctx context.Context, c Staff, patientID string) ([]Plan, error) {
	var out []Plan
	_, err := s.staffTx(ctx, c, auditevent.CodeSearchType, func(ctx context.Context) (Plan, error) {
		var err error
		out, err = s.list(ctx, c.AccountID, c.OrganizationID, patientID)
		return Plan{}, err
	})
	return out, err
}

// Mine returns the patient's booked plans at every clinic, newest first; each is audited as read.
func (s *Service) Mine(ctx context.Context, c Patient) ([]Plan, error) {
	var out []Plan
	err := s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.p.Repo.SetPatient(ctx, c.PatientID); err != nil {
			return err
		}
		var err error
		out, err = s.list(ctx, c.AccountID, "", c.PatientID)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("careplan: mine: %w", err)
	}
	return out, nil
}

// list reads the patient's plans in the scope set, settles each, and audits the search (naming the patient, also when
// nothing is found) and each plan as read by account.
func (s *Service) list(ctx context.Context, account, org, patientID string) ([]Plan, error) {
	all, err := s.p.Repo.ForPatient(ctx, patientID)
	if err != nil {
		return nil, err
	}
	search := entry(account, org, auditevent.CodeSearchType, "", auditevent.OutcomeSuccess)
	search.EntityReference = "Patient/" + patientID // the list's subject, also when it is empty
	if err := s.p.Audit.Record(ctx, search); err != nil {
		return nil, err
	}
	out := make([]Plan, 0, len(all))
	for _, p := range all {
		if p, err = s.settle(ctx, org, p); err != nil {
			return nil, err
		}
		if err := s.p.Audit.Record(ctx, entry(account, org, auditevent.CodeRead, p.ID, auditevent.OutcomeSuccess)); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

// staffTx runs fn for c in one transaction in the clinic's scope and audits interaction on the plan it returns; a
// refusal rolls back and is audited after.
func (s *Service) staffTx(ctx context.Context, c Staff, interaction string, fn func(ctx context.Context) (Plan, error)) (Plan, error) {
	if !c.Clinical() {
		return Plan{}, s.refuse(ctx, c, interaction, ErrForbidden)
	}
	var out Plan
	err := s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.p.Repo.SetOrganization(ctx, c.OrganizationID); err != nil {
			return err
		}
		var err error
		if out, err = fn(ctx); err != nil {
			return err
		}
		if interaction == auditevent.CodeSearchType {
			return nil // a list audits itself and each plan it shows
		}
		return s.p.Audit.Record(ctx, entry(c.AccountID, c.OrganizationID, interaction, out.ID, auditevent.OutcomeSuccess))
	})
	if i := slices.IndexFunc(refusals, func(r error) bool { return errors.Is(err, r) }); i >= 0 {
		return Plan{}, s.refuse(ctx, c, interaction, refusals[i])
	}
	if err != nil {
		return Plan{}, fmt.Errorf("careplan: %s: %w", interaction, err)
	}
	return out, nil
}

// refuse audits the refusal of c's interaction in a transaction of its own and returns refusal.
func (s *Service) refuse(ctx context.Context, c Staff, interaction string, refusal error) error {
	err := s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		return s.p.Audit.Record(ctx, entry(c.AccountID, c.OrganizationID, interaction, "", auditevent.OutcomeMinorFailure))
	})
	if err != nil {
		return fmt.Errorf("careplan: %s: %w", interaction, err)
	}
	return refusal
}

// current reads plan id for a change at version: ErrNotFound, ErrVersionConflict.
func (s *Service) current(ctx context.Context, id string, version int) (Plan, error) {
	p, err := s.p.Repo.ByID(ctx, id)
	if err != nil {
		return Plan{}, err
	}
	if p.VersionID != version {
		return Plan{}, ErrVersionConflict
	}
	return p, nil
}

// store writes p at its next version as changed by c.
func (s *Service) store(ctx context.Context, c Staff, p Plan) (Plan, error) {
	p.VersionID++
	ok, err := s.p.Repo.Update(ctx, p, c.AccountID)
	switch {
	case err != nil:
		return Plan{}, err
	case !ok:
		return Plan{}, ErrVersionConflict
	}
	return s.p.Repo.ByID(ctx, p.ID)
}

// checkDraft checks a draft for the patient: a title, 1 to MaxSteps steps, each an active service of the clinic's
// price list offered in its mode on a day in range (ErrInvalid), and the clinic's active, unrestricted care consent
// (ErrNotFound). It returns the steps numbered in order.
func (s *Service) checkDraft(ctx context.Context, c Staff, patientID string, d Draft) ([]Step, error) {
	title := strings.TrimSpace(d.Title)
	if title == "" || len([]rune(title)) > 200 || len(d.Steps) == 0 || len(d.Steps) > MaxSteps {
		return nil, ErrInvalid
	}
	prices, err := s.p.Prices.OfClinic(ctx, c.OrganizationID)
	if err != nil {
		return nil, err
	}
	steps := make([]Step, len(d.Steps))
	for i, st := range d.Steps {
		mode := schedule.VisitMode(st.VisitMode)
		offered := slices.ContainsFunc(prices, func(p chargeitemdefinition.Definition) bool {
			return p.ID == st.ServiceID && p.Active && slices.Contains(p.Modes, mode)
		})
		if !mode.Valid() || !offered || st.DayOffset < 0 || st.DayOffset > MaxDayOffset {
			return nil, ErrInvalid
		}
		steps[i] = Step{ServiceID: st.ServiceID, VisitMode: st.VisitMode, DayOffset: st.DayOffset, Position: i + 1}
	}
	consent, err := s.p.Consents.State(ctx, patientID, c.OrganizationID)
	switch {
	case err != nil:
		return nil, err
	case !consent.Active || consent.Restricted:
		return nil, ErrNotFound
	}
	return steps, nil
}

// settle orders a booked plan's steps by their visits' start (a moved step shows in its new place) and shows an active
// plan whose every visit has ended or was cancelled as completed. org is the clinic's scope, empty for the patient's.
func (s *Service) settle(ctx context.Context, org string, p Plan) (Plan, error) {
	if p.BookedAt == nil {
		return p, nil
	}
	visits, err := s.p.Visits.PlanVisits(ctx, org, visitIDs(p))
	if err != nil {
		return Plan{}, err
	}
	starts := make(map[string]time.Time, len(visits))
	ended := len(visits) == len(p.Steps)
	now := s.p.Clock.Now()
	for _, v := range visits {
		starts[v.ID] = v.Start
		ended = ended && (!v.End.After(now) || v.Status == appointment.StatusCancelled)
	}
	slices.SortStableFunc(p.Steps, func(a, b Step) int { return starts[a.AppointmentID].Compare(starts[b.AppointmentID]) })
	for i := range p.Steps {
		p.Steps[i].Position = i + 1
	}
	if p.Status == StatusActive && ended {
		p.Status = StatusCompleted
	}
	return p, nil
}

func visitIDs(p Plan) []string {
	ids := make([]string, 0, len(p.Steps))
	for _, st := range p.Steps {
		if st.AppointmentID != "" {
			ids = append(ids, st.AppointmentID)
		}
	}
	return ids
}

// visitCaller is c as the appointments' staff caller.
func visitCaller(c Staff) appointment.Caller {
	return appointment.Caller{AccountID: c.AccountID, OrganizationID: c.OrganizationID, PractitionerID: c.PractitionerID, Admin: c.Admin}
}

// bookingErr maps a refused booking or cancellation of the plan's visits to the plan's refusal: nothing changed.
func bookingErr(err error) error {
	switch {
	case errors.Is(err, appointment.ErrSlotUnavailable):
		return ErrSlotUnavailable
	case errors.Is(err, appointment.ErrForbidden):
		return ErrForbidden
	case errors.Is(err, appointment.ErrNotFound), errors.Is(err, appointment.ErrConsentRequired):
		return ErrNotFound
	case errors.Is(err, appointment.ErrInvalidService):
		return ErrInvalid
	}
	return err
}

// entry is the audit event of a FHIR restful interaction on CarePlan/id ("" when none was disclosed) by account, in
// the clinic org (empty for a patient: their own request).
func entry(account, org, interaction, id string, outcome auditevent.Outcome) auditevent.AuditEvent {
	action, purpose := auditevent.ActionRead, auditevent.CodeTREAT
	switch interaction {
	case auditevent.CodeCreate:
		action = auditevent.ActionCreate
	case auditevent.CodeUpdate:
		action = auditevent.ActionUpdate
	case auditevent.CodeSearchType:
		action = auditevent.ActionExecute
	}
	if org == "" {
		purpose = auditevent.CodePATRQT
	}
	e := auditevent.AuditEvent{
		Type:           auditevent.Coding{System: auditevent.SystemAuditEventType, Code: auditevent.CodeRest},
		Subtype:        auditevent.Coding{System: auditevent.SystemRestfulInteraction, Code: interaction},
		Purpose:        auditevent.Coding{System: auditevent.SystemActReason, Code: purpose},
		Action:         action,
		Outcome:        outcome,
		AgentAccountID: account,
		OrganizationID: org,
	}
	if id != "" {
		e.EntityReference = "CarePlan/" + id
	}
	return e
}
