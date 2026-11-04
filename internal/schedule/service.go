package schedule

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/practitionerrole"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

// Transactor runs fn in a transaction carried by the context, joining one already there
// (postgres.Transactor).
type Transactor interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// Auditor records an AuditEvent inside the caller's transaction (auditevent.Service).
type Auditor interface {
	Record(ctx context.Context, e auditevent.AuditEvent) error
}

// Roles reads a practitioner role for staff of its clinic (practitionerrole.Service).
type Roles interface {
	Read(ctx context.Context, c practitionerrole.Caller, id string) (practitionerrole.PractitionerRole, error)
}

// Jobs enqueues background work in the transaction in ctx.
type Jobs interface {
	// EnqueueSlotsGenerate enqueues slots_generate for the schedule of the clinic org.
	EnqueueSlotsGenerate(ctx context.Context, scheduleID, org string) error
}

// Ports are the Service's dependencies. The worker builds it for generation only (ForGeneration, Listed),
// which needs neither Roles nor Jobs.
type Ports struct {
	Repo  Repository
	Audit Auditor
	Tx    Transactor
	Clock clock.Clock
	Roles Roles
	Jobs  Jobs
}

// Service holds the schedule rules: the practitioner reads and edits their own role's availability, the
// clinic's admin any role's; an edit is valid (validate) and names the version it changes, and enqueues the
// schedule's slot generation in its transaction. Every staff read and edit is audited, also when refused.
// The directory reads (Read, ByActor) and the generation's (ForGeneration, Listed) are not: they disclose
// only what a listed clinic publishes, or nothing.
type Service struct {
	Ports
}

// NewService returns a Service on its ports.
func NewService(p Ports) *Service {
	return &Service{Ports: p}
}

// Availability returns the schedule of the role roleID, or an empty one at version 0 before its first edit.
// The practitioner of the role and the clinic's admin may; another member of the clinic gets ErrForbidden,
// anyone else ErrNotFound. Audited, also when refused.
func (s *Service) Availability(ctx context.Context, c Caller, roleID string) (Schedule, error) {
	var out Schedule
	var failed error // a refusal that is audited, and so commits
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		var err error
		if failed, err = s.allowed(ctx, c, roleID); err != nil {
			return err
		}
		if failed != nil {
			return s.record(ctx, c, auditevent.CodeRead, "", auditevent.OutcomeMinorFailure)
		}
		if out, err = s.byRole(ctx, c, roleID); err != nil {
			return err
		}
		return s.record(ctx, c, auditevent.CodeRead, out.ID, auditevent.OutcomeSuccess)
	})
	switch {
	case err != nil:
		return Schedule{}, fmt.Errorf("schedule: availability: %w", err)
	case failed != nil:
		return Schedule{}, failed
	}
	return out, nil
}

// SaveAvailability sets the slot length and the rules of the role roleID at version, the version the caller
// last read (0 for a role without a schedule, which creates it), returns the schedule at its new version and
// enqueues its slot generation, in one transaction. Who may is as for Availability. An invalid length or rule
// gives ErrInvalidSlotMinutes or ErrInvalidAvailability before anything is accessed; a version that is not
// the current one ErrConflict. Audited, also when refused.
func (s *Service) SaveAvailability(ctx context.Context, c Caller, roleID string, in Availability, version int32) (Schedule, error) {
	rules, err := validate(in)
	if err != nil {
		return Schedule{}, err
	}
	interaction := auditevent.CodeUpdate
	if version == 0 {
		interaction = auditevent.CodeCreate
	}
	var out Schedule
	var failed error // a refusal that is audited, and so commits
	err = s.Tx.WithTx(ctx, func(ctx context.Context) error {
		var err error
		if failed, err = s.allowed(ctx, c, roleID); err != nil {
			return err
		}
		if failed == nil {
			next := Schedule{OrganizationID: c.OrganizationID, PractitionerRoleID: roleID, SlotMinutes: in.SlotMinutes, Rules: rules, VersionID: version}
			if out, failed, err = s.save(ctx, c, next); err != nil {
				return err
			}
		}
		if failed != nil {
			return s.record(ctx, c, auditevent.CodeUpdate, "", auditevent.OutcomeMinorFailure)
		}
		if err := s.Jobs.EnqueueSlotsGenerate(ctx, out.ID, out.OrganizationID); err != nil {
			return err
		}
		return s.record(ctx, c, interaction, out.ID, auditevent.OutcomeSuccess)
	})
	switch {
	case err != nil:
		return Schedule{}, fmt.Errorf("schedule: save availability: %w", err)
	case failed != nil:
		return Schedule{}, failed
	}
	return out, nil
}

// save creates the role's schedule (version 0) or updates the current version of it. A version that is not
// the current one is returned as failed; err is any other error.
func (s *Service) save(ctx context.Context, c Caller, next Schedule) (out Schedule, failed, err error) {
	cur, err := s.byRole(ctx, c, next.PractitionerRoleID)
	switch {
	case err != nil:
		return Schedule{}, nil, err
	case cur.VersionID != next.VersionID:
		return Schedule{}, ErrConflict, nil
	case next.VersionID == 0:
		out, err = s.Repo.Create(ctx, next)
	default:
		next.ID = cur.ID
		out, err = s.Repo.Update(ctx, next, s.Clock.Now())
	}
	if errors.Is(err, ErrConflict) { // created or changed since the read
		return Schedule{}, ErrConflict, nil
	}
	return out, nil, err
}

// allowed sets the caller's clinic on the transaction and returns why c may not see or edit the role roleID's
// schedule, or nil when c is its practitioner or the clinic's admin; err is any other error.
func (s *Service) allowed(ctx context.Context, c Caller, roleID string) (failed, err error) {
	if c.OrganizationID == "" {
		return ErrNotFound, nil
	}
	if err := s.Repo.SetOrganization(ctx, c.OrganizationID); err != nil {
		return nil, err
	}
	role, err := s.Roles.Read(ctx, practitionerrole.Caller{AccountID: c.AccountID, OrganizationID: c.OrganizationID}, roleID)
	switch {
	case errors.Is(err, practitionerrole.ErrNotFound):
		return ErrNotFound, nil
	case err != nil:
		return nil, err
	case role.OrganizationID != c.OrganizationID:
		return ErrNotFound, nil
	case !c.Admin && (c.PractitionerID == "" || role.PractitionerID != c.PractitionerID):
		return ErrForbidden, nil
	}
	return nil, nil
}

// byRole is the schedule of the role roleID in c's clinic, or an empty one at version 0.
func (s *Service) byRole(ctx context.Context, c Caller, roleID string) (Schedule, error) {
	cur, err := s.Repo.ByRole(ctx, roleID)
	if errors.Is(err, ErrNotFound) || (err == nil && cur.OrganizationID != c.OrganizationID) { // not on row-level security alone
		return Schedule{OrganizationID: c.OrganizationID, PractitionerRoleID: roleID}, nil
	}
	return cur, err
}

// Read returns the schedule id of an active role of a listed clinic (FHIR Schedule read), or ErrNotFound.
// For anyone; not audited (docs/booking/SPEC.md "Public browse").
func (s *Service) Read(ctx context.Context, id string) (Schedule, error) {
	return s.public(ctx, func(ctx context.Context) (Schedule, error) { return s.Repo.ByID(ctx, id) })
}

// ByActor returns the schedule of the role roleID when it is an active role of a listed clinic
// (Schedule?actor=PractitionerRole/…), or ErrNotFound. For anyone; not audited.
func (s *Service) ByActor(ctx context.Context, roleID string) (Schedule, error) {
	return s.public(ctx, func(ctx context.Context) (Schedule, error) { return s.Repo.ByRole(ctx, roleID) })
}

func (s *Service) public(ctx context.Context, read func(ctx context.Context) (Schedule, error)) (Schedule, error) {
	var out Schedule
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.Repo.SetDirectory(ctx); err != nil {
			return err
		}
		var err error
		out, err = read(ctx)
		return err
	})
	switch {
	case errors.Is(err, ErrNotFound):
		return Schedule{}, ErrNotFound
	case err != nil:
		return Schedule{}, fmt.Errorf("schedule: public read: %w", err)
	}
	return out, nil
}

// ForGeneration returns the schedule id of the clinic org, locked until the transaction in ctx ends (the
// slots_generate job), or ErrNotFound. Not audited: a system step that discloses nothing.
func (s *Service) ForGeneration(ctx context.Context, org, id string) (Schedule, error) {
	var out Schedule
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.Repo.SetOrganization(ctx, org); err != nil {
			return err
		}
		var err error
		out, err = s.Repo.Lock(ctx, id)
		if err == nil && out.OrganizationID != org { // not on row-level security alone
			return ErrNotFound
		}
		return err
	})
	switch {
	case errors.Is(err, ErrNotFound):
		return Schedule{}, ErrNotFound
	case err != nil:
		return Schedule{}, fmt.Errorf("schedule: for generation: %w", err)
	}
	return out, nil
}

// Listed returns the schedules of the active roles of listed clinics, for the hourly slot generation. Not
// audited, like the directory.
func (s *Service) Listed(ctx context.Context) ([]Schedule, error) {
	var out []Schedule
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.Repo.SetDirectory(ctx); err != nil {
			return err
		}
		var err error
		out, err = s.Repo.Listed(ctx)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("schedule: listed: %w", err)
	}
	return out, nil
}

// record audits a read, create or update by c of the schedule id ("" for none).
func (s *Service) record(ctx context.Context, c Caller, interaction, id string, outcome auditevent.Outcome) error {
	action := auditevent.ActionRead
	switch interaction {
	case auditevent.CodeCreate:
		action = auditevent.ActionCreate
	case auditevent.CodeUpdate:
		action = auditevent.ActionUpdate
	}
	e := auditevent.AuditEvent{
		Type:           auditevent.Coding{System: auditevent.SystemAuditEventType, Code: auditevent.CodeRest},
		Subtype:        auditevent.Coding{System: auditevent.SystemRestfulInteraction, Code: interaction},
		Purpose:        auditevent.Coding{System: auditevent.SystemActReason, Code: auditevent.CodeHOPERAT},
		Action:         action,
		Outcome:        outcome,
		AgentAccountID: c.AccountID,
		OrganizationID: c.OrganizationID,
	}
	if id != "" {
		e.EntityReference = "Schedule/" + id
	}
	return s.Audit.Record(ctx, e)
}

// Slot lengths (docs/booking/SPEC.md "Schedule shape").
const (
	minSlotMinutes = 5
	maxSlotMinutes = 240
)

// validate checks an edit and returns its rules sorted by day and start.
func validate(in Availability) ([]Rule, error) {
	if in.SlotMinutes < minSlotMinutes || in.SlotMinutes > maxSlotMinutes || in.SlotMinutes%5 != 0 {
		return nil, ErrInvalidSlotMinutes
	}
	rules := slices.Clone(in.Rules)
	for _, r := range rules {
		start, okStart := Minutes(r.Start)
		end, okEnd := Minutes(r.End)
		if r.Day < 1 || r.Day > 7 || !okStart || !okEnd || start >= end {
			return nil, ErrInvalidAvailability
		}
	}
	slices.SortFunc(rules, func(a, b Rule) int {
		return cmp.Or(cmp.Compare(a.Day, b.Day), cmp.Compare(a.Start, b.Start))
	})
	for i := 1; i < len(rules); i++ {
		if rules[i].Day == rules[i-1].Day && rules[i].Start < rules[i-1].End {
			return nil, ErrInvalidAvailability
		}
	}
	return rules, nil
}

// Minutes is the time of day hhmm ("HH:MM", 00:00 to 23:59) in minutes after midnight, and false when it is
// not one.
func Minutes(hhmm string) (int, bool) {
	if len(hhmm) != 5 || hhmm[2] != ':' {
		return 0, false
	}
	digit := func(i int) (int, bool) { return int(hhmm[i] - '0'), hhmm[i] >= '0' && hhmm[i] <= '9' }
	h1, ok1 := digit(0)
	h2, ok2 := digit(1)
	m1, ok3 := digit(3)
	m2, ok4 := digit(4)
	h, m := h1*10+h2, m1*10+m2
	if !ok1 || !ok2 || !ok3 || !ok4 || h > 23 || m > 59 {
		return 0, false
	}
	return h*60 + m, true
}
