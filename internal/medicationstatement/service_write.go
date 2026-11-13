package medicationstatement

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

// The bounds of an input (00043).
const (
	maxText      = 200 // characters
	maxDose      = 100000
	maxPeriod    = 1000
	maxFrequency = 100
)

// Add adds an medication to the calling patient's profile: theirs, unconfirmed, recorded now. Staff get ErrForbidden;
// an input outside its value sets or lengths ErrInvalid.
func (s *Service) Add(ctx context.Context, c Caller, in Input) (MedicationStatement, error) {
	return s.do(ctx, c, auditevent.CodeCreate, func(ctx context.Context) (MedicationStatement, error) {
		if c.OrganizationID != "" || c.PatientID == "" {
			return MedicationStatement{}, ErrForbidden
		}
		if err := s.Repo.SetPatient(ctx, c.PatientID); err != nil {
			return MedicationStatement{}, err
		}
		now := s.Clock.Now().UTC()
		a := MedicationStatement{ID: newID(), PatientID: c.PatientID, DateAsserted: now, LastUpdated: now, InProfile: true, VersionID: 1}
		if err := s.apply(ctx, &a, in); err != nil {
			return MedicationStatement{}, err
		}
		return a, s.Repo.Insert(ctx, a)
	})
}

// Record records a medication in the encounter id during the visit: by its practitioner while the patient's care consent
// is active (access.ConsultationRecord), confirmed by them, belonging to the visit (not in the profile). Once the
// visit finished ErrForbidden; anyone else gets ErrNotFound.
func (s *Service) Record(ctx context.Context, c Caller, encounterID string, in Input) (MedicationStatement, error) {
	return s.do(ctx, c, auditevent.CodeCreate, func(ctx context.Context) (MedicationStatement, error) {
		if c.OrganizationID == "" || c.Admin || !fhir.IsUUID(encounterID) {
			return MedicationStatement{}, ErrNotFound
		}
		e, att, err := s.Encounters.Access(ctx, c, encounterID)
		switch {
		case errors.Is(err, encounter.ErrNotFound):
			return MedicationStatement{}, ErrNotFound
		case err != nil:
			return MedicationStatement{}, err
		}
		v := access.Viewer{Role: access.RolePractitioner, Clinic: c.OrganizationID}
		if att.Practitioner {
			v.PractitionerRole = e.PractitionerRoleID
		}
		visit := access.Visit{PractitionerRole: e.PractitionerRoleID, Start: att.Appointment.Start}
		f := access.Facts{Patient: e.PatientID, Clinic: e.OrganizationID, Visits: []access.Visit{visit},
			Consent: access.Consent{Active: att.ConsentActive, RevokedAt: att.RevokedAt}}
		if !access.ConsultationRecord(v, f, visit) {
			return MedicationStatement{}, ErrNotFound
		}
		if e.Status != encounter.StatusInProgress {
			return MedicationStatement{}, ErrForbidden // recorded during the visit only
		}
		if err := s.Repo.SetClinic(ctx, c.OrganizationID); err != nil {
			return MedicationStatement{}, err
		}
		now := s.Clock.Now().UTC()
		a := MedicationStatement{ID: newID(), PatientID: e.PatientID, DateAsserted: now, LastUpdated: now,
			Source:       Source{OrganizationID: e.OrganizationID, EncounterID: e.ID, RecorderRoleID: e.PractitionerRoleID},
			Confirmation: Confirmation{At: now, RoleID: e.PractitionerRoleID, OrganizationID: e.OrganizationID},
			VersionID:    1}
		if err := s.apply(ctx, &a, in); err != nil {
			return MedicationStatement{}, err
		}
		return a, s.Repo.Insert(ctx, a)
	})
}

// Update changes the medication id read at version: the patient their own entries (a confirmed one becomes unconfirmed
// again once its content or status changes), a practitioner the entries their clinic recorded (another clinic's confirmation is cleared, their clinic's
// stays). An input without a term id and with the entry's text keeps its term. ErrNotFound when the caller does not
// see it, ErrForbidden when it is not theirs to change, ErrVersionConflict when it changed since version.
func (s *Service) Update(ctx context.Context, c Caller, id string, version int32, in Input) (MedicationStatement, error) {
	return s.do(ctx, c, auditevent.CodeUpdate, func(ctx context.Context) (MedicationStatement, error) {
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
		before := content(a)
		if err := s.apply(ctx, &a, in); err != nil {
			return MedicationStatement{}, err
		}
		if content(a) != before && !a.Confirmation.At.IsZero() && (v.Role == access.RolePatient || a.Confirmation.OrganizationID != c.OrganizationID) {
			a.Confirmation = Confirmation{}
		}
		return s.update(ctx, a)
	})
}

// Remove removes the medication id read at version from what anyone sees: the patient's own unconfirmed entry is
// deleted; a confirmed one, or one a clinic recorded, is kept with the record as entered-in-error. The patient
// removes any entry they see, a practitioner only the entries their clinic recorded.
func (s *Service) Remove(ctx context.Context, c Caller, id string, version int32) error {
	_, err := s.do(ctx, c, auditevent.CodeDelete, func(ctx context.Context) (MedicationStatement, error) {
		v, _, a, err := s.atVersion(ctx, c, id, version)
		switch {
		case err != nil:
			return a, err
		case v.Role != access.RolePatient && a.Source.OrganizationID != c.OrganizationID:
			return named(a), ErrForbidden
		case a.Source.Patient() && a.Confirmation.At.IsZero():
			return a, s.Repo.Delete(ctx, a.ID, version)
		}
		a.Status = StatusEnteredInError
		return s.update(ctx, a)
	})
	return err
}

// Confirm confirms the medication id read at version as the calling practitioner's clinic: an entry in the profile, or
// one their clinic recorded. One their clinic confirmed stays as it is; another clinic's confirmation is never
// overwritten, a shared visit's entry is read-only, and a patient never confirms (ErrForbidden).
func (s *Service) Confirm(ctx context.Context, c Caller, id string, version int32) (MedicationStatement, error) {
	return s.do(ctx, c, auditevent.CodeUpdate, func(ctx context.Context) (MedicationStatement, error) {
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
		return s.update(ctx, a)
	})
}

// AddToProfile adds the medication id read at version, one a clinic recorded in the calling patient's visit, to their
// profile, so every clinic with their care consent sees it (without its source). One already there stays as it is;
// staff get ErrForbidden.
func (s *Service) AddToProfile(ctx context.Context, c Caller, id string, version int32) (MedicationStatement, error) {
	return s.do(ctx, c, auditevent.CodeUpdate, func(ctx context.Context) (MedicationStatement, error) {
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

// atVersion reads the medication id the caller sees, at version: ErrNotFound when they do not see it,
// ErrVersionConflict (naming it) when it is at another version.
func (s *Service) atVersion(ctx context.Context, c Caller, id string, version int32) (access.Viewer, access.Facts, MedicationStatement, error) {
	v, f, a, err := s.visible(ctx, c, id)
	switch {
	case err != nil:
		return v, f, MedicationStatement{}, err
	case a.VersionID != version:
		return v, f, named(a), ErrVersionConflict
	}
	return v, f, a, nil
}

// edit is what a change compares: the content and the status (the confirmation vouches for them).
type edit struct {
	Term, Text string
	Status     Status
	Dosage     Dosage
}

func content(m MedicationStatement) edit {
	return edit{Term: m.Code.TermID, Text: m.Code.Text, Status: m.Status, Dosage: m.Dosage}
}

// named is the entry as a refusal names it: its id only.
func named(a MedicationStatement) MedicationStatement { return MedicationStatement{ID: a.ID} }

// update writes a as its next version, now.
func (s *Service) update(ctx context.Context, a MedicationStatement) (MedicationStatement, error) {
	a.LastUpdated = s.Clock.Now().UTC()
	version, err := s.Repo.Update(ctx, a)
	if err != nil {
		return named(a), err
	}
	a.VersionID = version
	return a, nil
}

// apply validates in and writes it into a: the text trimmed, the term a drug of the terminology labelled text (with its ATC code),
// the dosage as Dosage says (a dose with its unit, a timing complete, each within the table's bounds).
func (s *Service) apply(ctx context.Context, a *MedicationStatement, in Input) error {
	in.Text = strings.TrimSpace(in.Text)
	if in.Text == "" || utf8.RuneCountInString(in.Text) > maxText || (in.Status != StatusActive && in.Status != StatusStopped) ||
		(in.TermID != "" && !fhir.IsUUID(in.TermID)) || !validDosage(in.Dosage) {
		return ErrInvalid
	}
	code, err := s.code(ctx, a.Code, in.TermID, in.Text)
	if err != nil {
		return err
	}
	a.Code, a.Status, a.Dosage = code, in.Status, in.Dosage
	return nil
}

// validDosage reports whether d is a dose (a value in (0, 100000) with a UCUM unit) or none, and a timing (1 to 100
// times per period in (0, 1000] of a period unit) or none.
func validDosage(d Dosage) bool {
	dose := (d.DoseUnit == "" && d.DoseValue == 0) ||
		(slices.Contains(DoseUnits(), d.DoseUnit) && d.DoseValue > 0 && d.DoseValue < maxDose)
	timing := (d.PeriodUnit == "" && d.Period == 0 && d.Frequency == 0) ||
		(slices.Contains(PeriodUnits(), d.PeriodUnit) && d.Period > 0 && d.Period <= maxPeriod && d.Frequency >= 1 && d.Frequency <= maxFrequency)
	return dose && timing
}

// code is the code of an entry whose code was old given termID and text: text only without a term; a term picked
// now carries its ATC code and has text as one of its labels (ErrInvalid otherwise); the term and text old already
// had stay as they are, though the term's labels changed since.
func (s *Service) code(ctx context.Context, old Code, termID, text string) (Code, error) {
	code := Code{TermID: termID, Text: text}
	var ok bool
	switch {
	case termID == "":
		return code, nil
	case termID == old.TermID && text == old.Text:
		atc, err := s.Terms.DrugCodes(ctx, []string{termID})
		if err != nil {
			return Code{}, err
		}
		code.ATC, ok = atc[termID]
	default:
		var err error
		if code.ATC, ok, err = s.Terms.DrugLabelled(ctx, termID, text); err != nil {
			return Code{}, err
		}
	}
	if !ok {
		return Code{}, ErrInvalid
	}
	return code, nil
}

// newID returns a random (version 4) UUID.
func newID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
