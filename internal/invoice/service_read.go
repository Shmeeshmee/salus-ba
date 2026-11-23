package invoice

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/Shmeeshmee/salus/internal/auditevent"
)

// Read returns a receipt of the caller's clinic to its admin and to the practitioner who issued it
// (GET /clinic/receipts/{id}); ErrNotFound to anyone else. Audited, also when refused.
func (s *Service) Read(ctx context.Context, c Caller, id string) (Invoice, error) {
	if c.OrganizationID == "" {
		return Invoice{}, s.refuse(ctx, c, auditevent.CodeRead, ErrNotFound)
	}
	var out Invoice
	err := s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.p.Repo.SetOrganization(ctx, c.OrganizationID); err != nil {
			return err
		}
		var err error
		if out, err = s.p.Repo.ByID(ctx, id); err != nil {
			return err
		}
		if !c.Admin && out.IssuedByAccountID != c.AccountID {
			return ErrNotFound
		}
		return s.p.Audit.Record(ctx, event(c, auditevent.CodeRead, out.ID, auditevent.OutcomeSuccess))
	})
	return out, s.readError(ctx, c, "read", err)
}

// own returns the patient's own receipt id, in their scope; ErrNotFound for any other. Audited, also when refused.
func (s *Service) own(ctx context.Context, c Caller, id string) (Invoice, error) {
	var out Invoice
	err := s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.p.Repo.SetPatient(ctx, c.PatientID); err != nil {
			return err
		}
		var err error
		if out, err = s.p.Repo.ByID(ctx, id); err != nil {
			return err
		}
		if out.PatientID != c.PatientID { // the owner check does not rest on row-level security alone
			return ErrNotFound
		}
		return s.p.Audit.Record(ctx, event(c, auditevent.CodeRead, out.ID, auditevent.OutcomeSuccess))
	})
	return out, s.readError(ctx, c, "own", err)
}

// readError audits a read that found nothing c may see and wraps any other failure.
func (s *Service) readError(ctx context.Context, c Caller, op string, err error) error {
	switch {
	case errors.Is(err, ErrNotFound):
		return s.refuse(ctx, c, auditevent.CodeRead, ErrNotFound)
	case err != nil:
		return fmt.Errorf("invoice: %s: %w", op, err)
	}
	return nil
}

// List returns the clinic's receipts issued in the period, newest first, to its admin (GET /clinic/receipts).
// ErrInvalid for a period that is none (not audited: nothing is read); ErrForbidden, audited, to anyone else.
func (s *Service) List(ctx context.Context, c Caller, p Period) ([]Invoice, error) {
	if p.From.IsZero() || !p.To.After(p.From) {
		return nil, ErrInvalid
	}
	if c.OrganizationID == "" || !c.Admin {
		return nil, s.refuse(ctx, c, auditevent.CodeSearchType, ErrForbidden)
	}
	var out []Invoice
	err := s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.p.Repo.SetOrganization(ctx, c.OrganizationID); err != nil {
			return err
		}
		var err error
		if out, err = s.p.Repo.Issued(ctx, c.OrganizationID, p); err != nil {
			return err
		}
		return s.p.Audit.Record(ctx, event(c, auditevent.CodeSearchType, "", auditevent.OutcomeSuccess))
	})
	if err != nil {
		return nil, fmt.Errorf("invoice: list: %w", err)
	}
	return out, nil
}

// Mine returns a patient's own receipts, from every clinic, newest first (GET /me/receipts). ErrForbidden, audited,
// to staff and to an account without a patient profile.
func (s *Service) Mine(ctx context.Context, c Caller) ([]Invoice, error) {
	if c.OrganizationID != "" || c.PatientID == "" {
		return nil, s.refuse(ctx, c, auditevent.CodeSearchType, ErrForbidden)
	}
	var out []Invoice
	err := s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.p.Repo.SetPatient(ctx, c.PatientID); err != nil {
			return err
		}
		var err error
		if out, err = s.p.Repo.OfPatient(ctx, c.PatientID); err != nil {
			return err
		}
		return s.p.Audit.Record(ctx, event(c, auditevent.CodeSearchType, "", auditevent.OutcomeSuccess))
	})
	if err != nil {
		return nil, fmt.Errorf("invoice: mine: %w", err)
	}
	return out, nil
}

// Report sums the clinic's receipts issued in the period for its accountant, to its admin
// (GET /clinic/reports/receipts): the count and total, by payment method and by service, reversals included with
// their negative amounts. It is computed from the receipts List returns, so it always equals them. It carries the
// clinic's default language for the CSV's headers.
func (s *Service) Report(ctx context.Context, c Caller, p Period) (Report, error) {
	list, err := s.List(ctx, c, p)
	if err != nil {
		return Report{}, err
	}
	clinic, err := s.p.Clinics.Clinic(ctx, c.OrganizationID) // outside the list's transaction: its own scope
	if err != nil {
		return Report{}, fmt.Errorf("invoice: report: %w", err)
	}
	r := Report{Period: p, Currency: "BAM", Language: clinic.DefaultLanguage, Count: len(list)}
	methods := map[PaymentMethod]*MethodTotal{}
	type key struct{ id, name string }
	services := map[key]*ServiceTotal{}
	for _, inv := range list {
		r.Currency = inv.Currency
		r.TotalMinor += inv.TotalMinor
		m := methods[inv.PaymentMethod]
		if m == nil {
			m = &MethodTotal{PaymentMethod: inv.PaymentMethod}
			methods[inv.PaymentMethod] = m
		}
		m.Count, m.TotalMinor = m.Count+1, m.TotalMinor+inv.TotalMinor
		for _, l := range inv.Lines {
			k := key{l.ChargeItemDefinitionID, l.Name}
			t := services[k]
			if t == nil {
				t = &ServiceTotal{ChargeItemDefinitionID: l.ChargeItemDefinitionID, Name: l.Name}
				services[k] = t
			}
			t.Quantity, t.TotalMinor = t.Quantity+int64(l.Quantity), t.TotalMinor+l.TotalMinor
		}
	}
	for _, m := range []PaymentMethod{PaymentCash, PaymentCard, PaymentTransfer} {
		if t := methods[m]; t != nil {
			r.ByPaymentMethod = append(r.ByPaymentMethod, *t)
		}
	}
	for _, t := range services {
		r.ByService = append(r.ByService, *t)
	}
	slices.SortFunc(r.ByService, func(a, b ServiceTotal) int {
		return cmp.Or(cmp.Compare(a.Name, b.Name), cmp.Compare(a.ChargeItemDefinitionID, b.ChargeItemDefinitionID))
	})
	return r, nil
}
