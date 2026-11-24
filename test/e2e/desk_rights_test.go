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
// the admin downloads the record as a FHIR collection, the cancelled visit included (WAVE5 S11), which the receptionist
// and another clinic are refused. The web journey (desk-rights.spec.ts, WAVE5 FX) then finds the withdrawn patient from
// the desk's patient search, opens the move of another patient's visit, and walks the clinic settings' navigation.
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

	anotherClinicIsRefused(t, reg.ID)

	desk.must(http.StatusNoContent, http.MethodPost, "/clinic/patients/"+reg.ID+"/consent-withdrawal", nil, nil)
	if code := desk.do(http.MethodPost, "/clinic/patients/"+reg.ID+"/consent-withdrawal", nil, nil); code != http.StatusConflict {
		t.Errorf("a second withdrawal = %d, want 409", code)
	}
	sarajevo := sarajevoZone(t)
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
	theRecordHoldsTheCancelledVisit(t, clinic, reg.ID, visit.ID)

	walkTheDeskFixes(t, desk, clinic)
}

// walkTheDeskFixes runs the web journey of WAVE5 FX (desk-rights.spec.ts) as the clinic's second admin: another
// registered patient with a visit ahead, to move from the desk, their phone typed as the desk types it.
func walkTheDeskFixes(t *testing.T, desk *api, clinic *bookableClinic) {
	t.Helper()
	adminAddr := email(t, "admin2")
	clinic.admin.must(http.StatusCreated, http.MethodPost, "/invites", map[string]string{"email": adminAddr, "role": "admin"}, nil)
	_, adminTOTP := enrolFromInvite(t, adminAddr)
	var kept struct{ ID string }
	desk.must(http.StatusCreated, http.MethodPost, "/clinic/patients", map[string]any{
		"givenName": "Mirsada", "familyName": "Pomjerena", "birthDate": "1981-02-03", "phone": "061 000 377", "consentGivenInPerson": true,
	}, &kept)
	keptSlot, _ := deskFreeTime(t, desk, clinic)
	desk.must(http.StatusCreated, http.MethodPost, "/clinic/appointments", map[string]string{"patientId": kept.ID, "slotId": keptSlot}, nil)
	runDeskJourney(t, adminAddr, adminTOTP, "desk-rights.spec.ts", "E2E_RIGHTS_PATIENT=Radmila Registrovana", "E2E_RIGHTS_BOOKED=Mirsada Pomjerena")
}

// anotherClinicIsRefused checks that another clinic's admin can neither download the record of the patient id nor
// record their withdrawal.
func anotherClinicIsRefused(t *testing.T, id string) {
	t.Helper()
	other := newBookableClinic(t)
	if code := other.admin.do(http.MethodGet, "/clinic/patients/"+id+"/record", nil, nil); code != http.StatusNotFound {
		t.Errorf("another clinic's admin downloads the record: %d, want 404", code)
	}
	if code := other.admin.do(http.MethodPost, "/clinic/patients/"+id+"/consent-withdrawal", nil, nil); code != http.StatusNotFound {
		t.Errorf("another clinic's admin records a withdrawal: %d, want 404", code)
	}
}

// theRecordHoldsTheCancelledVisit downloads, as the admin, the record of the patient id the clinic registered: the
// patient, the withdrawn consent and the visit the withdrawal cancelled (docs/clinic-cms/WAVE5.md S11), once the
// revocation's job has cancelled it (the desk's day hides it at once, by the consent rule).
func theRecordHoldsTheCancelledVisit(t *testing.T, clinic *bookableClinic, id, visitID string) {
	t.Helper()
	var record struct {
		ResourceType, Type string
		Entry              []struct {
			Resource struct {
				ResourceType, ID, Status string
				Name                     []struct{ Given []string }
			}
		}
	}
	var code int
	var raw []byte
	eventually(t, "the record shows the visit cancelled", func() bool {
		record.Entry = nil
		code, raw = clinic.admin.send(nil, http.MethodGet, "/clinic/patients/"+id+"/record", nil)
		return code == http.StatusOK && json.Unmarshal(raw, &record) == nil && len(record.Entry) == 3 && record.Entry[2].Resource.Status == "cancelled"
	})
	if record.Type != "collection" || len(record.Entry) != 3 {
		t.Fatalf("the admin's record = %d %s, want a collection of the patient, the consent and the visit", code, raw)
	}
	p, c, a := record.Entry[0].Resource, record.Entry[1].Resource, record.Entry[2].Resource
	if p.ResourceType != "Patient" || p.ID != id || p.Name[0].Given[0] != "Radmila" || c.ResourceType != "Consent" || c.Status != "inactive" ||
		a.ResourceType != "Appointment" || a.ID != visitID || a.Status != "cancelled" {
		t.Errorf("the record's entries = %+v, want the corrected patient, the withdrawn consent and the cancelled visit", record.Entry)
	}
}
