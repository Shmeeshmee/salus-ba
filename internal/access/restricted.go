package access

// Restricted reports whether the facts' record is restricted for viewer (docs/clinic-cms/WAVE5.md S7, GDPR Art. 18):
// staff of the facts' clinic, of any role, while the patient's record there is restricted. Every rule of the care
// record and of the chat then refuses them (staffOf), and a practitioner of another clinic is no guest in the
// clinic's chat (GuestChat). Only ClinicRecord, the admin's opening with a stated purpose that its service audits,
// reads a restricted record. The patient themselves is never restricted from their own record.
func Restricted(v Viewer, f Facts) bool {
	return f.Consent.Restricted && v.Role != RolePatient && v.Clinic != "" && v.Clinic == f.Clinic
}
