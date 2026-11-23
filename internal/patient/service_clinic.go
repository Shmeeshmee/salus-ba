package patient

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/consent"
)

const (
	minSearch     = 2  // characters of a clinic search's name, after trimming
	maxSearchRows = 50 // a clinic search's page: the desk narrows by the birth date
)

// ClinicSearch finds the clinic's patients (package F2 of docs/clinic-cms): by name and, when given, birth date.
// The clinic's patients are those with its care consent and those it registered.
type ClinicSearch struct {
	BirthDate time.Time // zero for any
	Name      string    // given or family name, or both
}

// desk reports whether c is the clinic's admin or receptionist: who registers patients and sees all of the clinic's.
func (c Caller) desk() bool { return !c.patient() && (c.Admin || c.Reception) }

// SearchClinic returns, by family and given name, at most 50 patients of the staff caller's clinic whose name
// contains q.Name (trimmed, at least 2 characters), born on q.BirthDate when it is set: to its admin and
// receptionists all of the clinic's, to a practitioner those with an appointment with their role. Each patient
// returned is audited (a search-type on Patient), a search that finds nobody once. ErrForbidden to a patient and
// to staff who are none of these, ErrInvalidSearch for a short name.
func (s *Service) SearchClinic(ctx context.Context, c Caller, q ClinicSearch) ([]Patient, error) {
	if !c.desk() && (c.patient() || c.PractitionerRoleID == "") {
		if err := s.record(ctx, c, auditevent.CodeSearchType, "", auditevent.OutcomeMinorFailure); err != nil {
			return nil, fmt.Errorf("patient: search: %w", err)
		}
		return nil, ErrForbidden
	}
	q.Name = strings.TrimSpace(q.Name)
	if utf8.RuneCountInString(q.Name) < minSearch {
		return nil, ErrInvalidSearch
	}
	role := c.PractitionerRoleID
	if c.desk() {
		role = ""
	}
	var out []Patient
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.repo.SetClinic(ctx, c.OrganizationID); err != nil {
			return err
		}
		var err error
		if out, err = s.repo.SearchClinic(ctx, c.OrganizationID, role, q, maxSearchRows); err != nil {
			return err
		}
		if len(out) == 0 {
			return s.record(ctx, c, auditevent.CodeSearchType, "", auditevent.OutcomeSuccess)
		}
		for _, p := range out {
			if err := s.record(ctx, c, auditevent.CodeSearchType, p.ID, auditevent.OutcomeSuccess); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("patient: search: %w", err)
	}
	return out, nil
}

// Register stores the profile of a patient who has no Salus account, registered by the clinic of its admin or
// receptionist c and seen by that clinic only, with the care consent the patient gave in person recorded by c (W1
// of docs/clinic-cms), in one transaction; both writes are audited. ErrForbidden to anyone else (audited), ErrInvalid…
// for a field that breaks SPEC.md "Profile". A patient's own profile is unchanged.
func (s *Service) Register(ctx context.Context, c Caller, in Patient) (Patient, error) {
	if !c.desk() {
		if err := s.record(ctx, c, auditevent.CodeCreate, "", auditevent.OutcomeMinorFailure); err != nil {
			return Patient{}, fmt.Errorf("patient: register: %w", err)
		}
		return Patient{}, ErrForbidden
	}
	in, err := s.valid(in)
	if err != nil {
		return Patient{}, err
	}
	var p Patient
	err = s.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.repo.SetClinic(ctx, c.OrganizationID); err != nil {
			return err
		}
		var err error
		if p, err = s.repo.Register(ctx, in, c.OrganizationID, s.clock.Now()); err != nil {
			return err
		}
		if _, err := s.consents.RecordInPerson(ctx, consent.Caller{AccountID: c.AccountID, OrganizationID: c.OrganizationID}, p.ID); err != nil {
			return err
		}
		return s.record(ctx, c, auditevent.CodeCreate, p.ID, auditevent.OutcomeSuccess)
	})
	if err != nil {
		return Patient{}, fmt.Errorf("patient: register: %w", err)
	}
	return p, nil
}
