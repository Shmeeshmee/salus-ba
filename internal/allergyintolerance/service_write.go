package allergyintolerance

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/encounter"
	"github.com/Shmeeshmee/salus/pkg/fhir"
)

// maxText is the longest substance and reaction text, in characters (00043).
const maxText = 200

// Add adds an allergy to the calling patient's profile: theirs, unconfirmed, recorded now. Staff get ErrForbidden;
// an input outside its value sets or lengths ErrInvalid.
func (s *Service) Add(ctx context.Context, c Caller, in Input) (AllergyIntolerance, error) {
	return s.do(ctx, c, auditevent.CodeCreate, func(ctx context.Context) (AllergyIntolerance, error) {
		if c.OrganizationID != "" || c.PatientID == "" {
			return AllergyIntolerance{}, ErrForbidden
		}
		if err := s.Repo.SetPatient(ctx, c.PatientID); err != nil {
			return AllergyIntolerance{}, err
		}
		now := s.Clock.Now().UTC()
		a := AllergyIntolerance{ID: newID(), PatientID: c.PatientID, RecordedDate: now, LastUpdated: now, InProfile: true,
			VerificationStatus: VerificationUnconfirmed, VersionID: 1}
		if err := s.apply(ctx, &a, in); err != nil {
			return AllergyIntolerance{}, err
		}
		return a, s.Repo.Insert(ctx, a)
	})
}

// Record records an allergy in the encounter id: by its practitioner while the patient's care consent is active
// (access.ConsultationRecord), confirmed by them, belonging to the visit (not in the profile). Anyone else gets
// ErrNotFound.
func (s *Service) Record(ctx context.Context, c Caller, encounterID string, in Input) (AllergyIntolerance, error) {
	return s.do(ctx, c, auditevent.CodeCreate, func(ctx context.Context) (AllergyIntolerance, error) {
		if c.OrganizationID == "" || c.Admin || !fhir.IsUUID(encounterID) {
			return AllergyIntolerance{}, ErrNotFound
		}
		e, att, err := s.Encounters.Access(ctx, c, encounterID)
		switch {
		case errors.Is(err, encounter.ErrNotFound):
			return AllergyIntolerance{}, ErrNotFound
		case err != nil:
			return AllergyIntolerance{}, err
		}
		v := access.Viewer{Role: access.RolePractitioner, Clinic: c.OrganizationID}
		if att.Practitioner {
			v.PractitionerRole = e.PractitionerRoleID
		}
		visit := access.Visit{PractitionerRole: e.PractitionerRoleID, Start: att.Appointment.Start}
		f := access.Facts{Patient: e.PatientID, Clinic: e.OrganizationID, Visits: []access.Visit{visit},
			Consent: access.Consent{Active: att.ConsentActive, RevokedAt: att.RevokedAt}}
		if !access.ConsultationRecord(v, f, visit) {
			return AllergyIntolerance{}, ErrNotFound
		}
		if err := s.Repo.SetClinic(ctx, c.OrganizationID); err != nil {
			return AllergyIntolerance{}, err
		}
		now := s.Clock.Now().UTC()
		a := AllergyIntolerance{ID: newID(), PatientID: e.PatientID, RecordedDate: now, LastUpdated: now,
			Source:             Source{OrganizationID: e.OrganizationID, EncounterID: e.ID, RecorderRoleID: e.PractitionerRoleID},
			Confirmation:       Confirmation{At: now, RoleID: e.PractitionerRoleID, OrganizationID: e.OrganizationID},
			VerificationStatus: VerificationConfirmed, VersionID: 1}
		if err := s.apply(ctx, &a, in); err != nil {
			return AllergyIntolerance{}, err
		}
		return a, s.Repo.Insert(ctx, a)
	})
}

// Update changes the allergy id read at version: the patient their own entries (a confirmed one becomes unconfirmed
// again), a practitioner the entries their clinic recorded (another clinic's confirmation is cleared, their clinic's
// stays). An input without a term id and with the entry's text keeps its term. ErrNotFound when the caller does not
// see it, ErrForbidden when it is not theirs to change, ErrVersionConflict when it changed since version.
func (s *Service) Update(ctx context.Context, c Caller, id string, version int32, in Input) (AllergyIntolerance, error) {
	return s.do(ctx, c, auditevent.CodeUpdate, func(ctx context.Context) (AllergyIntolerance, error) {
		v, _, a, err := s.atVersion(ctx, c, id, version)
		if err != nil {
			return a, err
		}
		switch {
		case v.Role == access.RolePatient && !a.Source.Patient(), v.Role != access.RolePatient && a.Source.OrganizationID != c.OrganizationID:
			return named(a), ErrForbidden
		}
		if in.TermID == "" && strings.TrimSpace(in.Text) == a.Code.Text {
			in.TermID = a.Code.TermID
		}
		if err := s.apply(ctx, &a, in); err != nil {
			return AllergyIntolerance{}, err
		}
		if !a.Confirmation.At.IsZero() && (v.Role == access.RolePatient || a.Confirmation.OrganizationID != c.OrganizationID) {
			a.Confirmation, a.VerificationStatus = Confirmation{}, VerificationUnconfirmed
		}
		return s.update(ctx, a)
	})
}

// Remove removes the allergy id read at version from what anyone sees: the patient's own unconfirmed entry is
// deleted; a confirmed one, or one a clinic recorded, is kept with the record as entered-in-error. The patient
// removes any entry they see, a practitioner only the entries their clinic recorded.
func (s *Service) Remove(ctx context.Context, c Caller, id string, version int32) error {
	_, err := s.do(ctx, c, auditevent.CodeDelete, func(ctx context.Context) (AllergyIntolerance, error) {
		v, _, a, err := s.atVersion(ctx, c, id, version)
		switch {
		case err != nil:
			return a, err
		case v.Role != access.RolePatient && a.Source.OrganizationID != c.OrganizationID:
			return named(a), ErrForbidden
		case a.Source.Patient() && a.Confirmation.At.IsZero():
			return a, s.Repo.Delete(ctx, a.ID, version)
		}
		a.VerificationStatus = VerificationEnteredInError
		return s.update(ctx, a)
	})
	return err
}

// Confirm confirms the allergy id read at version as the calling practitioner's clinic: an entry in the profile, or
// one their clinic recorded. One their clinic confirmed stays as it is; another clinic's confirmation is never
// overwritten, a shared visit's entry is read-only, and a patient never confirms (ErrForbidden).
func (s *Service) Confirm(ctx context.Context, c Caller, id string, version int32) (AllergyIntolerance, error) {
	return s.do(ctx, c, auditevent.CodeUpdate, func(ctx context.Context) (AllergyIntolerance, error) {
		v, _, a, err := s.atVersion(ctx, c, id, version)
		switch {
		case err != nil:
			return a, err
		case v.Role != access.RolePractitioner || v.PractitionerRole == "",
			!a.InProfile && a.Source.OrganizationID != c.OrganizationID,
			!a.Confirmation.At.IsZero() && a.Confirmation.OrganizationID != c.OrganizationID:
			return named(a), ErrForbidden
		case !a.Confirmation.At.IsZero():
			return a, nil
		}
		a.Confirmation = Confirmation{At: s.Clock.Now().UTC(), RoleID: v.PractitionerRole, OrganizationID: c.OrganizationID}
		a.VerificationStatus = VerificationConfirmed
		return s.update(ctx, a)
	})
}

// AddToProfile adds the allergy id read at version, one a clinic recorded in the calling patient's visit, to their
// profile, so every clinic with their care consent sees it (without its source). One already there stays as it is;
// staff get ErrForbidden.
func (s *Service) AddToProfile(ctx context.Context, c Caller, id string, version int32) (AllergyIntolerance, error) {
	return s.do(ctx, c, auditevent.CodeUpdate, func(ctx context.Context) (AllergyIntolerance, error) {
		v, _, a, err := s.atVersion(ctx, c, id, version)
		switch {
		case err != nil:
			return a, err
		case v.Role != access.RolePatient:
			return named(a), ErrForbidden
		case a.InProfile:
			return a, nil
		}
		a.InProfile = true
		return s.update(ctx, a)
	})
}

// atVersion reads the allergy id the caller sees, at version: ErrNotFound when they do not see it,
// ErrVersionConflict (naming it) when it is at another version.
func (s *Service) atVersion(ctx context.Context, c Caller, id string, version int32) (access.Viewer, access.Facts, AllergyIntolerance, error) {
	v, f, a, err := s.visible(ctx, c, id)
	switch {
	case err != nil:
		return v, f, AllergyIntolerance{}, err
	case a.VersionID != version:
		return v, f, named(a), ErrVersionConflict
	}
	return v, f, a, nil
}

// named is the entry as a refusal names it: its id only.
func named(a AllergyIntolerance) AllergyIntolerance { return AllergyIntolerance{ID: a.ID} }

// update writes a as its next version, now.
func (s *Service) update(ctx context.Context, a AllergyIntolerance) (AllergyIntolerance, error) {
	a.LastUpdated = s.Clock.Now().UTC()
	version, err := s.Repo.Update(ctx, a)
	if err != nil {
		return named(a), err
	}
	a.VersionID = version
	return a, nil
}

// apply validates in and writes it into a: the text trimmed, the term a drug of the terminology (with its ATC code).
func (s *Service) apply(ctx context.Context, a *AllergyIntolerance, in Input) error {
	in.Text, in.Reaction = strings.TrimSpace(in.Text), strings.TrimSpace(in.Reaction)
	switch {
	case in.Text == "", utf8.RuneCountInString(in.Text) > maxText, utf8.RuneCountInString(in.Reaction) > maxText,
		in.Category != "" && !slices.Contains([]Category{CategoryFood, CategoryMedication, CategoryEnvironment, CategoryBiologic}, in.Category),
		in.Severity != "" && !slices.Contains([]Severity{SeverityMild, SeverityModerate, SeveritySevere}, in.Severity),
		in.Severity != "" && in.Reaction == "",
		!slices.Contains([]ClinicalStatus{ClinicalActive, ClinicalInactive, ClinicalResolved}, in.ClinicalStatus),
		in.TermID != "" && !fhir.IsUUID(in.TermID):
		return ErrInvalid
	}
	code := Code{TermID: in.TermID, Text: in.Text}
	if in.TermID != "" {
		atc, err := s.Terms.DrugCodes(ctx, []string{in.TermID})
		if err != nil {
			return err
		}
		var ok bool
		if code.ATC, ok = atc[in.TermID]; !ok {
			return ErrInvalid
		}
	}
	a.Code, a.Category, a.Reaction, a.Severity, a.ClinicalStatus = code, in.Category, in.Reaction, in.Severity, in.ClinicalStatus
	return nil
}

// newID returns a random (version 4) UUID.
func newID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
