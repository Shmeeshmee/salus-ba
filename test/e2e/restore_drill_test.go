//go:build e2e && drill

package e2e_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/base32"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"testing"
)

// The restore drill's journeys (scripts/restore-drill.sh, docs/deploy/restore-drills.md), built only with the drill
// tag beside e2e, so make e2e never runs them. The seed writes a clinic, a patient, an attachment and a receipt on the
// drill's own stack and keeps what the verify needs in RESTORE_DRILL_STATE (synthetic accounts and their TOTP
// secrets, on the drill's working directory, removed by the script); the script then backs the stack up, destroys
// its database and object store, and restores them; the verify signs in again, on the restored stack, and checks the
// attachment decrypts to the bytes uploaded and the receipt's PDF is served.

// drillState is what the seed hands to the verify.
type drillState struct {
	AdminAddr, AdminSecret               string
	PractitionerAddr, PractitionerSecret string
	PatientAddr, PatientBearer           string
	BinaryID, BinarySHA1                 string
	ReceiptID, ReceiptPDFSHA256          string
	ReceiptPDFSize                       int
}

func drillStatePath(t *testing.T) string { t.Helper(); return env(t, "RESTORE_DRILL_STATE") }

func TestRestoreDrill_Seed(t *testing.T) {
	clinic, q := newClinicWithFileItems(t)
	p := newPatientWithProfile(t)
	p.bookWithQuestionnaire(clinic, q.title, freeSlots(t, p.api, clinic.scheduleID)[0])
	r := p.waitForRequests(1)[0]
	original := photo(t)
	img := p.upload(r.ID, "a-files", "Placeholder photo "+suffix(t)+".jpg", original)
	p.waitForStatus(img, "clean")
	// Submitted, so the practitioner may open the file (an in-progress response is the patient's own).
	p.mustWith(ifMatch(mustAtoi(t, r.Meta.VersionID)), http.StatusOK, http.MethodPost, "/questionnaire-responses/"+r.ID+"/submit", nil, nil)

	slot := atTheClinic(t, clinic)
	var visit struct{ ID string }
	p.must(http.StatusCreated, http.MethodPost, "/appointments", map[string]string{"slotId": slot.ID}, &visit)
	var service struct{ ID string }
	clinic.admin.must(http.StatusCreated, http.MethodPost, "/clinic/services", map[string]any{
		"name": "Placeholder examination", "priceMinor": 5000, "modes": []string{"in-person"}, "active": true, "translations": []any{},
	}, &service)
	clinic.admin.must(http.StatusOK, http.MethodPut, "/clinic/billing-details", map[string]any{
		"legalName": "Drill synthetic clinic d.o.o.", "taxId": "4200000000001", "vatRegistered": false, "billingEmail": "billing@example.test",
	}, nil)
	var issued receipt
	clinic.admin.must(http.StatusCreated, http.MethodPost, "/clinic/receipts", map[string]any{
		"kind": "receipt", "appointmentId": visit.ID, "paymentMethod": "cash",
		"lines": []map[string]any{{"serviceId": service.ID, "quantity": 1}},
	}, &issued)
	fiscalized(t, clinic, issued.ID)
	pdf := receiptPDF(t, clinic.admin, issued.ID)

	enc := base32.StdEncoding.WithPadding(base32.NoPadding)
	pdfSum := sha256.Sum256(pdf)
	s := drillState{
		AdminAddr: clinic.adminAddr, AdminSecret: enc.EncodeToString(clinic.adminTOTP.secret),
		PractitionerAddr: clinic.practitionerAddr, PractitionerSecret: enc.EncodeToString(clinic.totp.secret),
		PatientAddr: p.addr, PatientBearer: p.bearer,
		BinaryID: img.ID, BinarySHA1: sha1Hex(original),
		ReceiptID: issued.ID, ReceiptPDFSHA256: hex.EncodeToString(pdfSum[:]), ReceiptPDFSize: len(pdf),
	}
	b, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(drillStatePath(t), b, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Logf("seeded: binary %s (%d bytes), receipt %s (PDF %d bytes)", img.ID, len(original), issued.ID, len(pdf))
}

func TestRestoreDrill_Verify(t *testing.T) {
	b, err := os.ReadFile(drillStatePath(t))
	if err != nil {
		t.Fatal(err)
	}
	var s drillState
	if err := json.Unmarshal(b, &s); err != nil {
		t.Fatal(err)
	}

	// Staff sign in again on the restored database: password, then a TOTP code, which proves TOTP_KEY opens the
	// restored MFA secrets.
	practitioner := staffSignIn(t, s.PractitionerAddr, s.PractitionerSecret)
	status, _, body := practitioner.content(s.BinaryID, "")
	if status != http.StatusOK || sha1Hex(body) != s.BinarySHA1 {
		t.Fatalf("the practitioner's download of the restored attachment = %d with sha1 %s, want 200 with %s", status, sha1Hex(body), s.BinarySHA1)
	}
	t.Logf("attachment %s: decrypted and matches (%d bytes)", s.BinaryID, len(body))

	patientAPI := newAPI(t)
	patientAPI.bearer = patientAPI.login(s.PatientAddr).Token
	if status, _, body := patientAPI.content(s.BinaryID, ""); status != http.StatusOK || sha1Hex(body) != s.BinarySHA1 {
		t.Errorf("the patient's download of the restored attachment = %d, want 200 with the bytes uploaded", status)
	}

	admin := staffSignIn(t, s.AdminAddr, s.AdminSecret)
	pdf := receiptPDF(t, admin, s.ReceiptID)
	sum := sha256.Sum256(pdf)
	t.Logf("receipt %s: PDF %d bytes, identical to the seed's: %t", s.ReceiptID, len(pdf), hex.EncodeToString(sum[:]) == s.ReceiptPDFSHA256)

	// A session of the night of the dump is in the dump; the runbook closes every one before the stack opens, so a
	// sign-out made after the dump cannot come back.
	old := newAPI(t)
	old.bearer = s.PatientBearer
	if got := old.do(http.MethodGet, "/auth/me", nil, nil); got != http.StatusUnauthorized {
		t.Errorf("the patient's session from before the backup, after the restore: GET /auth/me = %d, want 401", got)
	}
}

// staffSignIn signs a staff account in with the drill's password and a code of its TOTP secret (base32).
func staffSignIn(t *testing.T, addr, secret string) *api {
	t.Helper()
	c := newAPI(t)
	l := c.login(addr)
	if l.Next != "verify" || l.Challenge == "" {
		t.Fatalf("login of a restored staff account: next %q, want verify", l.Next)
	}
	var verified login
	c.must(http.StatusOK, http.MethodPost, "/auth/mfa/verify", map[string]string{"challenge": l.Challenge, "code": newAuthenticator(t, secret).code()}, &verified)
	if verified.Next != "done" || verified.Token == "" {
		t.Fatalf("POST /auth/mfa/verify of a restored staff account: next %q", verified.Next)
	}
	c.bearer = verified.Token
	return c
}

// receiptPDF downloads the receipt's PDF and checks it is one.
func receiptPDF(t *testing.T, admin *api, id string) []byte {
	t.Helper()
	status, pdf := admin.send(nil, http.MethodGet, "/clinic/receipts/"+id+"/pdf", nil)
	if status != http.StatusOK || !bytes.HasPrefix(pdf, []byte("%PDF-")) {
		t.Fatalf("GET /clinic/receipts/%s/pdf = %d (%d bytes), want 200 with a PDF", id, status, len(pdf))
	}
	return pdf
}
