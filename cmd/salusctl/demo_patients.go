package main

import (
	"context"
	"net/http"
)

// demoPatient is one of the sixty: their Patient id, and for an app patient their own signed-in user.
type demoPatient struct {
	app   *demoUser
	id    string
	email string
}

// patients registers the sixty patients: the first five sign up in the patient app themselves (account, profile,
// consent to the clinic), the others at the desk, with their consent given in person, as the receptionist records
// it.
func (d *demoSeed) patients(ctx context.Context) error {
	for n := range demoPatients {
		given, family, birthDate, phone := demoPerson(n)
		profile := map[string]any{"givenName": given, "familyName": family, "birthDate": birthDate, "phone": phone}
		if n >= demoAppPatients {
			profile["consentGivenInPerson"] = true
			var p struct{ ID string }
			if _, err := d.desk.user.must(ctx, http.StatusCreated, http.MethodPost, "/clinic/patients", profile, &p); err != nil {
				return err
			}
			d.people = append(d.people, demoPatient{id: p.ID})
			continue
		}
		p, err := d.appPatient(ctx, demoPatientEmail(n), profile)
		if err != nil {
			return err
		}
		d.people = append(d.people, p)
	}
	return nil
}

// appPatient signs a patient up as the patient app does: the account, the sign-in, the profile, the consent.
func (d *demoSeed) appPatient(ctx context.Context, email string, profile map[string]any) (demoPatient, error) {
	u := d.user()
	creds := map[string]string{"email": email, "password": demoPassword, "language": demoLanguage}
	if _, err := u.must(ctx, http.StatusCreated, http.MethodPost, "/auth/register", creds, nil); err != nil {
		return demoPatient{}, err
	}
	var login struct{ Token string }
	if _, err := u.must(ctx, http.StatusOK, http.MethodPost, "/auth/login", creds, &login); err != nil {
		return demoPatient{}, err
	}
	u.token = login.Token
	var p struct{ ID string }
	if _, err := u.must(ctx, http.StatusCreated, http.MethodPut, "/me/patient", profile, &p); err != nil {
		return demoPatient{}, err
	}
	if _, err := u.must(ctx, http.StatusCreated, http.MethodPost, "/consents", map[string]string{"organizationId": d.org}, nil); err != nil {
		return demoPatient{}, err
	}
	return demoPatient{app: u, id: p.ID, email: email}, nil
}
