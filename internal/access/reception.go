package access

// Desk reports whether viewer sees the front desk's view of the facts' patient (docs/clinic-cms W2): the name, birth
// date and phone, and the appointments in the clinic with their service and visit type, never questionnaire answers,
// documents, history, transcripts, chat or analytics. The clinic's receptionist and its admin, with an active care
// consent or, after a revocation, while the patient has an appointment that started before it (Profile's Name);
// nobody else. Every other rule of this package refuses a receptionist.
func Desk(v Viewer, f Facts) bool {
	if (v.Role != RoleReception && v.Role != RoleAdmin) || v.Clinic == "" || v.Clinic != f.Clinic {
		return false
	}
	if f.Consent.Active {
		return true
	}
	for _, a := range f.Visits {
		if past(f, a) {
			return true
		}
	}
	return false
}
