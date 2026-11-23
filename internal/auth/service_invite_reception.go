package auth

// validReception checks a receptionist's invite (docs/clinic-cms, F1): the e-mail alone, as an admin's. A
// receptionist has no Practitioner, so the invite names no one.
func validReception(in Invite) error {
	if in.GivenName != "" || in.FamilyName != "" {
		return ErrInvalidName
	}
	return nil
}
