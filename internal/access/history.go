package access

import (
	"slices"
	"time"
)

// Record is one visit of the patient's history (docs/plans/patient-history.md, ADR 0013): its encounter, the clinic
// it was at and its practitioner.
type Record struct {
	Start            time.Time
	Encounter        string
	Clinic           string
	PractitionerRole string
}

// HistoryVisit reports whether viewer may see the visit r in the patient's history and what belongs to it (its
// signed documents, completed questionnaire answers, their clean files and the entries recorded in it), as Facts
// gathered for the viewer's clinic tell (ADR 0013 "Access model"): the patient always; at the viewer's own clinic, a
// practitioner only the visits they held themselves, with an active care consent; a visit at another clinic, a
// practitioner of the facts' clinic with an appointment with the patient there, while the care consent and the
// history consent with that clinic are active and the history consent lists the visit. The admin never, and nobody
// else. A visit that is not seen is not disclosed: not counted, not searched, not summarised, not given to the AI.
func HistoryVisit(v Viewer, f Facts, r Record) bool {
	switch {
	case self(v, f):
		return true
	case v.Role != RolePractitioner || !staffOf(v, f) || !f.Consent.Active:
		return false
	case r.Clinic == f.Clinic:
		return ofPractitioner(v, Visit{PractitionerRole: r.PractitionerRole})
	}
	return f.Consent.History && slices.Contains(f.Consent.Shared, r.Encounter) && hasVisit(v, f)
}

// ProfileEntry reports whether viewer may see an allergy or a medication of the patient: one in the profile
// (inProfile) where Profile is Full, one a clinic recorded in visit r and the patient has not added to the profile
// where HistoryVisit sees r.
func ProfileEntry(v Viewer, f Facts, inProfile bool, r Record) bool {
	if inProfile {
		return Profile(v, f) == Full
	}
	return HistoryVisit(v, f, r)
}

// Attributed reports whether viewer may see that clinic recorded or confirmed an entry, and who there: the patient,
// and staff of that clinic. Anyone else sees "confirmed by a clinician" at most: naming the clinic would tell that
// the patient was treated there (ADR 0013 "Doctor entries").
func Attributed(v Viewer, f Facts, clinic string) bool {
	return self(v, f) || (staffOf(v, f) && clinic != "" && clinic == f.Clinic)
}

// hasVisit reports whether the practitioner v has an appointment with the patient in the facts' clinic.
func hasVisit(v Viewer, f Facts) bool {
	return slices.ContainsFunc(f.Visits, func(a Visit) bool { return ofPractitioner(v, a) })
}
