package main

import (
	"context"
	"encoding/base32"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/Shmeeshmee/salus/internal/auth"
	"github.com/Shmeeshmee/salus/internal/subscription"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

// demoStaffUser is a signed-in staff member: the sheet's line, and for a practitioner their directory ids.
type demoStaffUser struct {
	user                               *demoUser
	practitioner                       *demoPractitioner
	signIn                             demoSignIn
	practitionerID, roleID, scheduleID string
}

// demoSignIn is a line of the sign-in sheet the seed prints: synthetic accounts of a developer's stack only.
type demoSignIn struct {
	who, email, totpSecret string
}

// clinic creates the clinic as salusctl create-clinic does (its trial included, once the default plan exists), signs
// its first admin in through the invite, and sets the clinic up as its admin would in the web app.
func (d *demoSeed) clinic(ctx context.Context) error {
	if ok, err := demoPlanSet(ctx, d.db, subscription.DefaultPlan); err != nil {
		return err
	} else if !ok {
		if err := d.members.PutPlan(ctx, subscription.Plan{Code: subscription.DefaultPlan, Name: "Demo (placeholder price)",
			Currency: demoCurrency, PricePerSeatMinor: demoPlanPrice, Active: true}); err != nil {
			return err
		}
	}
	created, err := d.signups.CreateClinic(ctx, demoClinicName, demoAdminEmail)
	if err != nil {
		return err
	}
	d.org = created.OrganizationID
	if d.admin, err = d.accept(ctx, created.InviteID, "Clinic admin", demoAdminEmail); err != nil {
		return err
	}
	if _, err := d.admin.user.must(ctx, http.StatusOK, http.MethodPut, "/clinic/billing-details", map[string]any{
		"vatRegistered": false, "legalName": "Poliklinika Primjer d.o.o. (demo)", "taxId": "4200000000000",
		"billingEmail": "racuni@" + demoDomain,
	}, nil); err != nil {
		return err
	}
	if err := d.organization(ctx); err != nil {
		return err
	}
	if err := d.places(ctx); err != nil {
		return err
	}
	return d.priceList(ctx)
}

// accept issues the invite's token as the invite e-mail would carry it, accepts it with the demo password, and
// confirms the first TOTP code: the staff member is signed in, and their secret goes on the sheet.
func (d *demoSeed) accept(ctx context.Context, inviteID, who, email string) (demoStaffUser, error) {
	u := d.user()
	var acc struct{ Challenge, Secret string }
	if err := d.acceptToken(ctx, u, inviteID, &acc); err != nil {
		return demoStaffUser{}, err
	}
	secret, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(acc.Secret)
	if err != nil {
		return demoStaffUser{}, errors.New("demo-seed: the enrolment's secret is not base32")
	}
	var login struct{ Token string }
	if _, err := u.must(ctx, http.StatusOK, http.MethodPost, "/auth/mfa/confirm",
		map[string]string{"challenge": acc.Challenge, "code": auth.TOTPCode(secret, clock.System{}.Now())}, &login); err != nil {
		return demoStaffUser{}, err
	}
	u.token = login.Token
	return demoStaffUser{user: u, signIn: demoSignIn{who: who, email: email, totpSecret: acc.Secret}}, nil
}

// acceptToken issues the invite's token and accepts it. The invite e-mail's job issues a token of its own for the
// e-mail, which replaces an earlier one: when it ran between the two, the seed waits a second and issues its token
// again, for at most demoWait.
func (d *demoSeed) acceptToken(ctx context.Context, u *demoUser, inviteID string, acc any) error {
	deadline := clock.System{}.Now().Add(demoWait)
	for {
		_, token, err := d.accounts.IssueInviteToken(ctx, inviteID, d.org)
		if err != nil {
			return err
		}
		res, err := u.call(ctx, http.MethodPost, "/invites/accept",
			map[string]string{"token": token, "password": demoPassword, "language": demoLanguage}, "")
		switch {
		case err != nil:
			return err
		case res.status == http.StatusCreated:
			return json.Unmarshal(res.body, acc)
		case res.status != http.StatusNotFound || clock.System{}.Now().After(deadline):
			return fmt.Errorf("demo-seed: POST /invites/accept = %d, want 201", res.status)
		}
		time.Sleep(time.Second)
	}
}

// demoWait is the longest the seed waits on the stack's workers.
const demoWait = time.Minute

// organization gives the clinic its address in Sarajevo and lists it in the public directory (FHIR Organization
// update, as the web app's clinic settings send it), and its profile for the patient app.
func (d *demoSeed) organization(ctx context.Context) error {
	path := "/fhir/R4/Organization/" + d.org
	res, err := d.admin.user.must(ctx, http.StatusOK, http.MethodGet, path, nil, nil)
	if err != nil {
		return err
	}
	listed := true
	body := map[string]any{
		"resourceType": "Organization", "id": d.org, "name": demoClinicName,
		"extension": []map[string]any{{"url": "https://salus.invalid/fhir/StructureDefinition/organization-listed", "valueBoolean": listed}},
		"address": []map[string]any{{
			"state": "BA-09", "postalCode": demoLocations[0].postalCode, "line": []string{demoLocations[0].addressLine},
			"extension": []map[string]any{{"url": "https://salus.invalid/fhir/StructureDefinition/organization-city", "valueCode": demoLocations[0].cityID}},
		}},
	}
	if _, err := d.admin.user.mustIf(ctx, http.StatusOK, http.MethodPut, path, res.etag, body, nil); err != nil {
		return err
	}
	res, err = d.admin.user.must(ctx, http.StatusOK, http.MethodGet, "/clinic/profile", nil, nil)
	if err != nil {
		return err
	}
	_, err = d.admin.user.mustIf(ctx, http.StatusOK, http.MethodPut, "/clinic/profile", res.etag, map[string]any{
		"languages": []string{"bs", "hr", "sr-Latn", "en", "de"}, "welcomesInternational": true, "translations": []any{},
		"description": "Demo klinika sa izmišljenim podacima: nijedna osoba, nalaz ni račun nisu stvarni.",
	}, nil)
	return err
}

// demoPlace is a location's id with its rooms' and devices' ids, in demoLocations' order.
type demoPlace struct {
	devices map[string]string // name to id
	id      string
	rooms   []string
}

// places adds the two locations with their rooms and the device.
func (d *demoSeed) places(ctx context.Context) error {
	for _, l := range demoLocations {
		var loc struct{ ID string }
		if _, err := d.admin.user.must(ctx, http.StatusCreated, http.MethodPost, "/clinic/locations", map[string]any{
			"name": l.name, "addressLine": l.addressLine, "postalCode": l.postalCode, "cityId": l.cityID, "phone": l.phone, "active": true,
		}, &loc); err != nil {
			return err
		}
		p := demoPlace{id: loc.ID, devices: map[string]string{}}
		for _, room := range l.rooms {
			var r struct{ ID string }
			if _, err := d.admin.user.must(ctx, http.StatusCreated, http.MethodPost, "/clinic/locations/"+loc.ID+"/rooms",
				map[string]any{"name": room, "active": true}, &r); err != nil {
				return err
			}
			p.rooms = append(p.rooms, r.ID)
		}
		for _, device := range l.devices {
			var dev struct{ ID string }
			if _, err := d.admin.user.must(ctx, http.StatusCreated, http.MethodPost, "/clinic/locations/"+loc.ID+"/devices",
				map[string]any{"name": device, "active": true}, &dev); err != nil {
				return err
			}
			p.devices[device] = dev.ID
		}
		d.locations = append(d.locations, p)
	}
	return nil
}

// priceList adds the clinic's services.
func (d *demoSeed) priceList(ctx context.Context) error {
	d.services = map[string]string{}
	for _, s := range demoServices {
		body := map[string]any{
			"name": s.name, "code": s.code, "priceMinor": s.price, "currency": demoCurrency, "durationMinutes": s.minutes,
			"modes": s.modes, "active": s.active, "translations": []any{},
		}
		if s.device != "" {
			body["deviceId"] = d.locations[0].devices[s.device]
		}
		var out struct{ ID string }
		if _, err := d.admin.user.must(ctx, http.StatusCreated, http.MethodPost, "/clinic/services", body, &out); err != nil {
			return err
		}
		d.services[s.code] = out.ID
	}
	return nil
}

// staff invites the four practitioners and the receptionist as the admin, and signs each in through their invite.
func (d *demoSeed) staff(ctx context.Context) error {
	for i := range demoPractitioners {
		p := &demoPractitioners[i]
		st, err := d.invite(ctx, "practitioner", p.email, p.given, p.family, "Dr. "+p.given+" "+p.family)
		if err != nil {
			return err
		}
		st.practitioner = p
		var me struct{ PractitionerID string }
		if _, err := st.user.must(ctx, http.StatusOK, http.MethodGet, "/auth/me", nil, &me); err != nil {
			return err
		}
		st.practitionerID = me.PractitionerID
		if st.roleID, err = demoRole(ctx, d.db, me.PractitionerID); err != nil {
			return err
		}
		d.practitioners = append(d.practitioners, st)
	}
	var err error
	d.desk, err = d.invite(ctx, "reception", demoReceptionEmail, "", "", "Reception") // a receptionist's invite names no one
	return err
}

// invite invites one staff member as the admin and accepts the invite.
func (d *demoSeed) invite(ctx context.Context, role, email, given, family, who string) (demoStaffUser, error) {
	var inv struct{ ID string }
	if _, err := d.admin.user.must(ctx, http.StatusCreated, http.MethodPost, "/invites",
		map[string]string{"email": email, "role": role, "givenName": given, "familyName": family}, &inv); err != nil {
		return demoStaffUser{}, err
	}
	return d.accept(ctx, inv.ID, who, email)
}

// hours gives each practitioner their specialty and weekly hours as the admin, and generates their slots for the
// four weeks ahead as the worker does after an edit of the hours (slot.Service.Generate).
func (d *demoSeed) hours(ctx context.Context) error {
	for i := range d.practitioners {
		st := &d.practitioners[i]
		path := "/practitioner-roles/" + st.roleID
		role, err := d.admin.user.must(ctx, http.StatusOK, http.MethodGet, "/fhir/R4/PractitionerRole/"+st.roleID, nil, nil)
		if err != nil {
			return err
		}
		if _, err := d.admin.user.mustIf(ctx, http.StatusOK, http.MethodPut, path+"/specialty", role.etag,
			map[string]string{"code": st.practitioner.specialty}, nil); err != nil {
			return err
		}
		var rules []map[string]any
		for _, h := range st.practitioner.hours {
			r := map[string]any{"day": h.day, "start": h.start, "end": h.end, "mode": h.mode}
			if h.mode == "in-person" {
				r["locationId"], r["roomId"] = d.locations[h.location].id, d.locations[h.location].rooms[h.room]
			}
			rules = append(rules, r)
		}
		var sched struct{ ID string }
		if _, err := d.admin.user.mustIf(ctx, http.StatusOK, http.MethodPut, path+"/availability", `W/"0"`,
			map[string]any{"slotMinutes": 30, "availability": rules}, &sched); err != nil {
			return err
		}
		st.scheduleID = sched.ID
		if err := d.generate(ctx, sched.ID); err != nil {
			return err
		}
	}
	return nil
}

// generate runs the schedule's slot generation; the worker may run the same generation at the same time, after the
// edit of the hours, and a generation that lost that race is run again.
func (d *demoSeed) generate(ctx context.Context, scheduleID string) error {
	var err error
	for range 3 {
		if _, err = d.slots.Generate(ctx, d.org, scheduleID); err == nil {
			return nil
		}
		time.Sleep(time.Second)
	}
	return fmt.Errorf("demo-seed: generate slots: %w", err)
}
