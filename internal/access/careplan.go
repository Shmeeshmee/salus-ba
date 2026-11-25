package access

// CarePlan reports whether viewer may work with the facts' patient's treatment plans at the facts' clinic (write,
// book, cancel, read; docs/clinic-cms/WAVE6.md U6): the clinic's admin for any patient of the clinic (a care consent
// with it, active or revoked, or a visit there), a practitioner only for a patient they have or had a visit with, of
// any status (the care relationship). The consent a write needs is the service's check. Nobody while the record is
// restricted (staffOf), never a receptionist or the patient: the patient reads their booked plans as their own.
func CarePlan(v Viewer, f Facts) bool {
	if !staffOf(v, f) {
		return false
	}
	if v.Role == RolePractitioner {
		return hasVisit(v, f)
	}
	return f.Consent.Active || !f.Consent.RevokedAt.IsZero() || len(f.Visits) > 0
}
