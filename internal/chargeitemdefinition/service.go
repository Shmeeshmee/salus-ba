package chargeitemdefinition

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/internal/schedule"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

// The limits of a service; the table's checks and api/openapi.yaml say the same, but for the price's ceiling, which
// keeps a line's total (an int32 quantity times the price) inside an int64.
const (
	maxNameLength = 200
	maxCodeLength = 40
	maxPriceMinor = 1_000_000_000
	minDuration   = 5
	maxDuration   = 480
	maxVATRateBP  = 10000
	currencyBAM   = "BAM"
)

// languages are the tags of the Postgres enum language (CLAUDE.md "Languages"): a patient's languages.
var languages = []string{"bs", "hr", "sr-Latn", "sr-Cyrl", "en", "de", "ar", "tr", "it", "fr"}

// Transactor runs fn in a transaction that travels in the context (postgres.Transactor).
type Transactor interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// Auditor records an AuditEvent inside the caller's transaction (auditevent.Service).
type Auditor interface {
	Record(ctx context.Context, e auditevent.AuditEvent) error
}

// Clinics is the public directory of listed clinics (organization.Service).
type Clinics interface {
	ListListed(ctx context.Context) ([]organization.Organization, error)
}

// Ports are what the Service runs on.
type Ports struct {
	Repo    Repository
	Audit   Auditor
	Tx      Transactor
	Clock   clock.Clock
	Clinics Clinics
}

// Service holds the price list's rules: the clinic's staff see its services, only its admin changes them (audited,
// also when refused), the public directory sees the active services of listed clinics, and a receipt's lines are
// priced from it. Reads are not audited: nothing in a price list is personal data.
type Service struct{ Ports }

// NewService returns a Service on its ports.
func NewService(p Ports) *Service { return &Service{Ports: p} }

// List returns every service of the caller's clinic, active or not, to its staff (GET /clinic/services). ErrNotFound
// for a caller of no clinic.
func (s *Service) List(ctx context.Context, c Caller) ([]Definition, error) {
	if c.OrganizationID == "" {
		return nil, ErrNotFound
	}
	var out []Definition
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.Repo.SetOrganization(ctx, c.OrganizationID); err != nil {
			return err
		}
		var err error
		out, err = s.Repo.List(ctx, c.OrganizationID)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("chargeitemdefinition: list: %w", err)
	}
	return out, nil
}

// Create adds a service to the clinic's price list, by its admin (POST /clinic/services), audited. ErrForbidden for
// its other staff and ErrNotFound for anyone else, audited too; ErrInvalid for a service outside the limits.
func (s *Service) Create(ctx context.Context, c Caller, d Definition) (Definition, error) {
	return s.save(ctx, c, d, auditevent.CodeCreate, auditevent.ActionCreate, s.Repo.Create)
}

// Update replaces the service d.ID at version d.VersionID, by the clinic's admin (PUT /clinic/services/{id}), audited.
// Refusals as Create's; ErrNotFound for a service that is not the clinic's and ErrConflict when d.VersionID is not the
// current version, both audited as refused. An issued receipt keeps the lines it copied.
func (s *Service) Update(ctx context.Context, c Caller, d Definition) (Definition, error) {
	return s.save(ctx, c, d, auditevent.CodeUpdate, auditevent.ActionUpdate, s.Repo.Update)
}

// save is Create and Update: the admin's check, the validation, the write and its audit in one transaction.
func (s *Service) save(ctx context.Context, c Caller, d Definition, interaction string, action auditevent.Action,
	write func(context.Context, Definition, time.Time) (Definition, error),
) (Definition, error) {
	if err := adminOf(c); err != nil {
		return Definition{}, s.refused(ctx, c, interaction, action, err)
	}
	d, err := normalized(d)
	if err != nil {
		return Definition{}, err
	}
	d.OrganizationID = c.OrganizationID
	var out Definition
	err = s.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.Repo.SetOrganization(ctx, c.OrganizationID); err != nil {
			return err
		}
		var err error
		if out, err = write(ctx, d, s.Clock.Now()); err != nil {
			return err
		}
		return s.Audit.Record(ctx, event(interaction, action, c, out.ID, auditevent.OutcomeSuccess))
	})
	switch {
	case errors.Is(err, ErrNotFound), errors.Is(err, ErrConflict): // refused: the write's transaction rolled back
		return Definition{}, s.refused(ctx, c, interaction, action, err)
	case err != nil:
		return Definition{}, fmt.Errorf("chargeitemdefinition: save: %w", err)
	}
	return out, nil
}

// refused audits a refused write and returns why it was refused, or the auditor's error.
func (s *Service) refused(ctx context.Context, c Caller, interaction string, action auditevent.Action, why error) error {
	if err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		return s.Audit.Record(ctx, event(interaction, action, c, "", auditevent.OutcomeMinorFailure))
	}); err != nil {
		return fmt.Errorf("chargeitemdefinition: audit: %w", err)
	}
	return why
}

// Offered returns the active services of the listed clinic org offered in mode, by name, each named in language where
// the clinic translated it (empty: the clinic's language): the prices shown at booking. Public, as the directory.
// ErrInvalid for an unknown mode or language, ErrNotFound for a clinic that is not listed.
func (s *Service) Offered(ctx context.Context, org string, mode schedule.VisitMode, language string) ([]Definition, error) {
	if !mode.Valid() || language != "" && !slices.Contains(languages, language) {
		return nil, ErrInvalid
	}
	listed, err := s.Clinics.ListListed(ctx)
	if err != nil {
		return nil, fmt.Errorf("chargeitemdefinition: offered: %w", err)
	}
	if !slices.ContainsFunc(listed, func(o organization.Organization) bool { return o.ID == org && o.Listed }) {
		return nil, ErrNotFound
	}
	out := []Definition{}
	err = s.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.Repo.SetDirectory(ctx); err != nil {
			return err
		}
		all, err := s.Repo.List(ctx, org)
		for _, d := range all {
			if !d.Active || !slices.Contains(d.Modes, mode) { // not on row-level security alone
				continue
			}
			if i := slices.IndexFunc(d.Translations, func(t Translation) bool { return t.Language == language }); i >= 0 {
				d.Name = d.Translations[i].Name
			}
			out = append(out, d)
		}
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("chargeitemdefinition: offered: %w", err)
	}
	return out, nil
}

// Lines prices the picks of a receipt of the clinic org from its active services, in the caller's transaction (the
// clinic's scope is set on it): internal/invoice's Prices port. Each line copies the service's name, unit price and
// VAT rate as they are now. ErrNotFound for a pick that is not an active service of the clinic, ErrInvalid for no
// picks, a quantity below one, or a discount below zero or above the line's amount.
func (s *Service) Lines(ctx context.Context, org string, picks []Pick) ([]Line, error) {
	if len(picks) == 0 {
		return nil, ErrInvalid
	}
	out := make([]Line, 0, len(picks))
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.Repo.SetOrganization(ctx, org); err != nil {
			return err
		}
		all, err := s.Repo.List(ctx, org)
		if err != nil {
			return err
		}
		for _, p := range picks {
			i := slices.IndexFunc(all, func(d Definition) bool { return d.ID == p.DefinitionID && d.Active })
			if i < 0 {
				return ErrNotFound
			}
			d := all[i]
			amount := int64(p.Quantity) * d.PriceMinor
			if p.Quantity < 1 || p.DiscountMinor < 0 || p.DiscountMinor > amount {
				return ErrInvalid
			}
			out = append(out, Line{
				VATRateBP: d.VATRateBP, DefinitionID: d.ID, Name: d.Name, Currency: d.Currency, UnitPriceMinor: d.PriceMinor,
				DiscountMinor: p.DiscountMinor, TotalMinor: amount - p.DiscountMinor, Quantity: p.Quantity,
			})
		}
		return nil
	})
	if errors.Is(err, ErrNotFound) || errors.Is(err, ErrInvalid) {
		return nil, err
	}
	if err != nil {
		return nil, fmt.Errorf("chargeitemdefinition: lines: %w", err)
	}
	return out, nil
}

// adminOf is nil for the admin of a clinic, ErrForbidden for its other staff and ErrNotFound for anyone else.
func adminOf(c Caller) error {
	switch {
	case c.OrganizationID == "":
		return ErrNotFound
	case !c.Admin:
		return ErrForbidden
	}
	return nil
}

// normalized is d with its name, code and translated names trimmed and the currency BAM when none is named, or
// ErrInvalid for a service outside the limits.
func normalized(d Definition) (Definition, error) {
	d.Name, d.Code = strings.TrimSpace(d.Name), strings.TrimSpace(d.Code)
	if d.Currency == "" {
		d.Currency = currencyBAM
	}
	modes := slices.Clone(d.Modes)
	slices.Sort(modes)
	switch {
	case d.Name == "" || utf8.RuneCountInString(d.Name) > maxNameLength || utf8.RuneCountInString(d.Code) > maxCodeLength,
		d.PriceMinor < 0 || d.PriceMinor > maxPriceMinor || d.Currency != currencyBAM,
		d.DurationMinutes != 0 && (d.DurationMinutes < minDuration || d.DurationMinutes > maxDuration),
		d.VATRateBP != nil && (*d.VATRateBP < 0 || *d.VATRateBP > maxVATRateBP),
		len(modes) == 0 || len(slices.Compact(modes)) != len(d.Modes),
		slices.ContainsFunc(modes, func(m schedule.VisitMode) bool { return !m.Valid() }):
		return Definition{}, ErrInvalid
	}
	d.Translations = slices.Clone(d.Translations)
	seen := map[string]bool{}
	for i, t := range d.Translations {
		t.Name = strings.TrimSpace(t.Name)
		if !slices.Contains(languages, t.Language) || seen[t.Language] || t.Name == "" || utf8.RuneCountInString(t.Name) > maxNameLength {
			return Definition{}, ErrInvalid
		}
		seen[t.Language], d.Translations[i] = true, t
	}
	return d, nil
}

// event is the audit of a write of the price list by c: the service id, or none for a refused write.
func event(interaction string, action auditevent.Action, c Caller, id string, outcome auditevent.Outcome) auditevent.AuditEvent {
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
		e.EntityReference = "ChargeItemDefinition/" + id
	}
	return e
}
