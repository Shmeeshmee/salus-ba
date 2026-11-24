//go:build e2e

package e2e_test

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"
	"time"
)

// TestDesk_TheClinicActsOnTheRightsOfAPatientItRegistered (docs/clinic-cms R2, V2): the receptionist registers a
// patient without an account and books them; corrects their name at the version read; records that they withdrew the
// consent given in person, after which the visit ahead is cancelled by the revocation's job and leaves the desk's day;
// the admin downloads the record as a FHIR collection, which the receptionist and another clinic are refused.
func TestDesk_TheClinicActsOnTheRightsOfAPatientItRegistered(t *testing.T) {
	clinic := newBookableClinic(t)
	atTheClinic(t, clinic)
	deskAddr := email(t, "reception")
	clinic.admin.must(http.StatusCreated, http.MethodPost, "/invites", map[string]string{"email": deskAddr, "role": "reception"}, nil)
	desk, _ := enrolFromInvite(t, deskAddr)

	slotID, start := deskFreeTime(t, desk, clinic)
	var reg struct {
		ID        string
		VersionID int
	}
	desk.must(http.StatusCreated, http.MethodPost, "/clinic/patients", map[string]any{
		"givenName": "Ranka", "familyName": "Registrovana", "birthDate": "1970-05-06", "consentGivenInPerson": true,
	}, &reg)
	var visit struct{ ID string }
	desk.must(http.StatusCreated, http.MethodPost, "/clinic/appointments", map[string]string{"patientId": reg.ID, "slotId": slotID}, &visit)

	var corrected struct {
		GivenName string
		VersionID int
	}
	desk.mustWith(http.Header{"If-Match": {`W/"` + strconv.Itoa(reg.VersionID) + `"`}}, http.StatusOK, http.MethodPut, "/clinic/patients/"+reg.ID,
		map[string]string{"givenName": "Radmila", "familyName": "Registrovana", "birthDate": "1970-06-05"}, &corrected)
	if corrected.GivenName != "Radmila" || corrected.VersionID != reg.VersionID+1 {
		t.Errorf("the correction = %+v, want the new name at the next version", corrected)
	}
	if code, _ := desk.send(http.Header{"If-Match": {`W/"` + strconv.Itoa(reg.VersionID) + `"`}}, http.MethodPut, "/clinic/patients/"+reg.ID,
		map[string]string{"givenName": "Stale", "familyName": "Registrovana", "birthDate": "1970-06-05"}); code != http.StatusConflict {
		t.Errorf("a correction at the old version = %d, want 409", code)
	}

	other := newBookableClinic(t)
	if code := other.admin.do(http.MethodGet, "/clinic/patients/"+reg.ID+"/record", nil, nil); code != http.StatusNotFound {
		t.Errorf("another clinic's admin downloads the record: %d, want 404", code)
	}
	if code := other.admin.do(http.MethodPost, "/clinic/patients/"+reg.ID+"/consent-withdrawal", nil, nil); code != http.StatusNotFound {
		t.Errorf("another clinic's admin records a withdrawal: %d, want 404", code)
	}

	desk.must(http.StatusNoContent, http.MethodPost, "/clinic/patients/"+reg.ID+"/consent-withdrawal", nil, nil)
	if code := desk.do(http.MethodPost, "/clinic/patients/"+reg.ID+"/consent-withdrawal", nil, nil); code != http.StatusConflict {
		t.Errorf("a second withdrawal = %d, want 409", code)
	}
	sarajevo, err := time.LoadLocation("Europe/Sarajevo")
	if err != nil {
		t.Fatal(err)
	}
	eventually(t, "the visit ahead is cancelled after the withdrawal", func() bool {
		var day struct {
			Visits []struct{ Appointment struct{ ID string } }
		}
		desk.must(http.StatusOK, http.MethodGet, "/clinic/desk/today?date="+start.In(sarajevo).Format(time.DateOnly), nil, &day)
		for _, v := range day.Visits {
			if v.Appointment.ID == visit.ID {
				return false
			}
		}
		return true
	})

	if code := desk.do(http.MethodGet, "/clinic/patients/"+reg.ID+"/record", nil, nil); code != http.StatusForbidden {
		t.Errorf("the receptionist downloads the record: %d, want 403", code)
	}
	code, raw := clinic.admin.send(nil, http.MethodGet, "/clinic/patients/"+reg.ID+"/record", nil)
	var record struct {
		ResourceType, Type string
		Entry              []struct {
			Resource struct {
				ResourceType, ID, Status string
				Name                     []struct{ Given []string }
			}
		}
	}
	if code != http.StatusOK || json.Unmarshal(raw, &record) != nil || record.Type != "collection" || len(record.Entry) != 2 {
		t.Fatalf("the admin's record = %d %s, want a collection of the patient and the consent", code, raw)
	}
	if p, c := record.Entry[0].Resource, record.Entry[1].Resource; p.ResourceType != "Patient" || p.ID != reg.ID || p.Name[0].Given[0] != "Radmila" ||
		c.ResourceType != "Consent" || c.Status != "inactive" {
		t.Errorf("the record's entries = %+v, want the corrected patient and the withdrawn consent", record.Entry)
	}
}
