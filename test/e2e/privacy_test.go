//go:build e2e

package e2e_test

import (
	"archive/zip"
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"

	"github.com/Shmeeshmee/salus/internal/datarequest"
	"github.com/Shmeeshmee/salus/internal/worker"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

// deadlineSubject begins the deadline warning's subject in English (D5: "A data subject request is due in 7 days").
const deadlineSubject = "A data subject request is due in"

// dataRequest is a data subject request as the API answers it (api/openapi.yaml, DataRequest).
type dataRequest struct {
	ID, Kind, Target, Status, PatientID, AnswerCode string
	Version                                         int
}

// fileRequest files the patient's request of kind to their account (clinic "") or to the clinic.
func (p *patient) fileRequest(kind, clinic string) dataRequest {
	p.t.Helper()
	body := map[string]string{"kind": kind, "target": "account"}
	if clinic != "" {
		body["target"], body["organizationId"] = "clinic", clinic
	}
	var r dataRequest
	p.must(http.StatusCreated, http.MethodPost, "/me/data-requests", body, &r)
	return r
}

// runJob inserts the periodic job args as the owner (DATABASE_URL) and waits until the worker completed it: the
// journeys make time pass by moving the stored instants back (as startAt does) and run the job at once instead of
// waiting for its schedule.
func runJob(t *testing.T, args river.JobArgs) {
	t.Helper()
	pool, err := pgxpool.New(t.Context(), env(t, "DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	client, err := river.NewClient(riverpgxv5.New(pool), &river.Config{})
	if err != nil {
		t.Fatal(err)
	}
	job, err := client.Insert(t.Context(), args, nil)
	if err != nil {
		t.Fatal(err)
	}
	eventually(t, "the "+args.Kind()+" job completes", func() bool {
		var state string
		if err := pool.QueryRow(t.Context(), `SELECT state FROM river_job WHERE id = $1`, job.Job.ID).Scan(&state); err != nil {
			t.Fatal(err)
		}
		if state == "discarded" || state == "cancelled" {
			t.Fatalf("the %s job ended %s", args.Kind(), state)
		}
		return state == "completed"
	})
}

// ageRequest moves the request's receipt and due date back so that it is due in left from now: the guard (00071)
// keeps both fixed for every role, so the owner turns triggers off for this one transaction.
func ageRequest(t *testing.T, id string, left time.Duration) {
	t.Helper()
	conn := ownerConn(t)
	err := pgx.BeginFunc(t.Context(), conn, func(tx pgx.Tx) error {
		if _, err := tx.Exec(t.Context(), `SET LOCAL session_replication_role = replica`); err != nil {
			return err
		}
		_, err := tx.Exec(t.Context(), `UPDATE data_request SET received_at = received_at - (due_at - (now() + make_interval(secs => $2))),
			due_at = now() + make_interval(secs => $2) WHERE id = $1`, id, left.Seconds())
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

// S7, D4: the patient files a restriction at the clinic, its admin finds it in the queue and restricts the record;
// the practitioner and the desk no longer find the patient, the admin answers the request.
func TestPrivacy_ARestrictionHidesThePatientFromThePractitionerAndTheDesk(t *testing.T) {
	clinic := newBookableClinic(t)
	deskAddr := email(t, "reception")
	clinic.admin.must(http.StatusCreated, http.MethodPost, "/invites", map[string]string{"email": deskAddr, "role": "reception"}, nil)
	desk, _ := enrolFromInvite(t, deskAddr)
	p := newPatientWithProfile(t)
	p.bookFirstTime(clinic)
	finds := func(who *api) bool {
		var found []struct{ ID string }
		who.must(http.StatusOK, http.MethodGet, "/clinic/patients?name=Syntheticova&birthDate=1990-05-17", nil, &found)
		for _, f := range found {
			if f.ID == p.patientID {
				return true
			}
		}
		return false
	}
	if !finds(clinic.practitioner) || !finds(desk) {
		t.Fatal("the practitioner or the desk does not find the patient before the restriction")
	}

	r := p.fileRequest("restriction", clinic.id)
	var queue []dataRequest
	clinic.admin.must(http.StatusOK, http.MethodGet, "/clinic/data-requests", nil, &queue)
	if len(queue) != 1 || queue[0].ID != r.ID || queue[0].Kind != "restriction" || queue[0].PatientID != p.patientID {
		t.Fatalf("the admin's queue = %+v, want the restriction request", queue)
	}
	clinic.admin.must(http.StatusNoContent, http.MethodPut, "/clinic/patients/"+p.patientID+"/restriction", map[string]string{"dataRequestId": r.ID}, nil)
	if finds(clinic.practitioner) || finds(desk) {
		t.Error("the practitioner or the desk still finds the patient whose record is restricted")
	}
	if code := clinic.practitioner.do(http.MethodGet, "/fhir/R4/Patient/"+p.patientID, nil, nil); code == http.StatusOK {
		t.Error("the practitioner reads the restricted patient's FHIR Patient")
	}
	var answered dataRequest
	clinic.admin.mustWith(ifMatch(r.Version), http.StatusOK, http.MethodPost, "/clinic/data-requests/"+r.ID+"/answer",
		map[string]string{"status": "fulfilled", "code": "done"}, &answered)
	var mine []dataRequest
	p.must(http.StatusOK, http.MethodGet, "/me/data-requests", nil, &mine)
	if len(mine) != 1 || mine[0].Status != "fulfilled" || mine[0].AnswerCode != "done" {
		t.Errorf("the patient's requests = %+v, want the restriction fulfilled", mine)
	}
}

// S10: the desk files a request on behalf of a patient the clinic registered, after checking identity there; the
// admin finds it in the queue as filed by the clinic.
func TestPrivacy_TheDeskFilesARequestForAPatientTheClinicRegistered(t *testing.T) {
	clinic := newBookableClinic(t)
	deskAddr := email(t, "reception")
	clinic.admin.must(http.StatusCreated, http.MethodPost, "/invites", map[string]string{"email": deskAddr, "role": "reception"}, nil)
	desk, _ := enrolFromInvite(t, deskAddr)
	var reg struct{ ID string }
	desk.must(http.StatusCreated, http.MethodPost, "/clinic/patients", map[string]any{
		"givenName": "Rada", "familyName": "Registrovana", "birthDate": "1972-03-04", "consentGivenInPerson": true,
	}, &reg)
	var filed dataRequest
	desk.must(http.StatusCreated, http.MethodPost, "/clinic/patients/"+reg.ID+"/data-requests", map[string]string{"kind": "rectification"}, &filed)
	if filed.Target != "clinic" || filed.PatientID != reg.ID || filed.Status != "received" {
		t.Errorf("filed on behalf = %+v", filed)
	}
	var queue []struct {
		ID             string
		FiledByPatient bool
	}
	clinic.admin.must(http.StatusOK, http.MethodGet, "/clinic/data-requests", nil, &queue)
	if len(queue) != 1 || queue[0].ID != filed.ID || queue[0].FiledByPatient {
		t.Errorf("the admin's queue = %+v, want the request the clinic filed", queue)
	}
	if code := desk.do(http.MethodGet, "/clinic/data-requests", nil, nil); code != http.StatusForbidden {
		t.Errorf("the desk lists the queue: %d, want 403", code)
	}
	if code := clinic.practitioner.do(http.MethodPost, "/clinic/patients/"+reg.ID+"/data-requests", map[string]string{"kind": "access"}, nil); code != http.StatusForbidden {
		t.Errorf("the practitioner files on behalf: %d, want 403", code)
	}
}

// accountRow is what the owner reads of an account after its erasure.
type accountRow struct {
	Email, Hash          string
	TOTP, MFA, Languages bool
}

func readAccount(t *testing.T, conn *pgx.Conn, id string) accountRow {
	t.Helper()
	var a accountRow
	if err := conn.QueryRow(t.Context(), `SELECT email, password_hash, totp_secret IS NOT NULL, mfa_enrolled_at IS NOT NULL,
		language IS NOT NULL OR detected_language IS NOT NULL FROM account WHERE id = $1`, id).Scan(&a.Email, &a.Hash, &a.TOTP, &a.MFA, &a.Languages); err != nil {
		t.Fatal(err)
	}
	return a
}

// erasureState is what a second run of the erasure must leave as it was.
func erasureState(t *testing.T, conn *pgx.Conn, account string) string {
	t.Helper()
	var s string
	if err := conn.QueryRow(t.Context(), `SELECT a.email || a.password_hash || coalesce(a.erased_at::text, '') ||
		(SELECT count(*) FROM patient_restriction r JOIN patient p ON p.id = r.patient_id WHERE p.account_id = a.id)::text ||
		(SELECT string_agg(d.status || d.version::text, ',' ORDER BY d.received_at) FROM data_request d WHERE d.account_id = a.id) ||
		(SELECT count(*) FROM data_request_change c JOIN data_request d ON d.id = c.data_request_id WHERE d.account_id = a.id)::text
		FROM account a WHERE a.id = $1`, account).Scan(&s); err != nil {
		t.Fatal(err)
	}
	return s
}

// S5, S6, D3, D5: the patient closes their account; sign-in is refused and the visit ahead cancelled; the operator
// reopens it within 21 days. Closed again, the erasure job runs 21 days later: the account's e-mail and credentials
// are gone for every role, the clinic keeps the record, restricted, and a second run changes nothing.
func TestPrivacy_APatientClosesTheAccountTheOperatorReopensItAndTheErasureRunsAt21Days(t *testing.T) {
	clinic := newBookableClinic(t)
	p := newPatientWithProfile(t)
	visit := p.bookFirstTime(clinic)
	me := p.me()
	p.must(http.StatusNoContent, http.MethodPost, "/me/account/close", map[string]string{"password": password}, nil)
	if code := newAPI(t).do(http.MethodPost, "/auth/login", map[string]string{"email": p.addr, "password": password}, nil); code == http.StatusOK {
		t.Fatalf("sign-in after closure = %d, want refused", code)
	}
	conn := ownerConn(t)
	eventually(t, "the visit ahead is cancelled", func() bool {
		var status string
		if err := conn.QueryRow(t.Context(), `SELECT status FROM appointment WHERE id = $1`, visit.ID).Scan(&status); err != nil {
			t.Fatal(err)
		}
		return status == "cancelled"
	})

	if out := salusctl(t, "datarequest", "reopen-account", me.ID); strings.Contains(out, p.addr) {
		t.Errorf("salusctl reopen-account prints the e-mail: %s", out)
	}
	again := newAPI(t)
	again.bearer = again.login(p.addr).Token
	again.must(http.StatusNoContent, http.MethodPost, "/me/account/close", map[string]string{"password": password}, nil)

	if _, err := conn.Exec(t.Context(), `UPDATE account SET closed_at = closed_at - interval '21 days',
		erase_after = erase_after - interval '21 days' WHERE id = $1`, me.ID); err != nil {
		t.Fatal(err)
	}
	runJob(t, worker.AccountErasureArgs{})
	a := readAccount(t, conn, me.ID)
	if a.Email != me.ID+"@erased.invalid" || a.Hash != "" || a.TOTP || a.MFA || a.Languages {
		t.Errorf("the account after erasure = %+v, want its e-mail, password, MFA and languages gone", a)
	}
	theErasedAccountIsGoneForEveryRole(t, clinic, p, me.ID)
	theRecordIsKeptRestricted(t, conn, clinic, p.patientID, visit.ID)

	before := erasureState(t, conn, me.ID)
	runJob(t, worker.AccountErasureArgs{})
	if after := erasureState(t, conn, me.ID); after != before {
		t.Errorf("a second run changed %q to %q", before, after)
	}
}

// theErasedAccountIsGoneForEveryRole: the old e-mail signs nobody in and is nowhere in the owner's tables that name
// an account, the clinic's staff no longer find the patient, and the operator's view of the request shows no e-mail.
func theErasedAccountIsGoneForEveryRole(t *testing.T, clinic *bookableClinic, p *patient, account string) {
	t.Helper()
	if code := newAPI(t).do(http.MethodPost, "/auth/login", map[string]string{"email": p.addr, "password": password}, nil); code == http.StatusOK {
		t.Error("the erased e-mail still signs in")
	}
	conn := ownerConn(t)
	var n int
	if err := conn.QueryRow(t.Context(), `SELECT (SELECT count(*) FROM account WHERE email = $1)
		+ (SELECT count(*) FROM audit_event WHERE to_jsonb(audit_event)::text LIKE '%' || $1 || '%')`, p.addr).Scan(&n); err != nil || n != 0 {
		t.Errorf("the erased e-mail is still in %d rows (%v)", n, err)
	}
	for name, who := range map[string]*api{"practitioner": clinic.practitioner, "admin": clinic.admin} {
		if body := string(must200(t, who, "/clinic/patients?name=Syntheticova")); strings.Contains(body, p.patientID) || strings.Contains(body, p.addr) {
			t.Errorf("the %s finds the erased patient: %s", name, body)
		}
	}
	if out := salusctl(t, "datarequest", "list"); strings.Contains(out, p.addr) {
		t.Errorf("salusctl datarequest list shows the e-mail: %s", out)
	}
}

// theRecordIsKeptRestricted: the clinic keeps the profile and the visit under the medical-record retention, the
// record restricted by the job (no staff, which the clinic cannot lift).
func theRecordIsKeptRestricted(t *testing.T, conn *pgx.Conn, clinic *bookableClinic, patientID, visitID string) {
	t.Helper()
	var visits, jobs int
	if err := conn.QueryRow(t.Context(), `SELECT (SELECT count(*) FROM appointment WHERE id = $1 AND patient_id = $2),
		(SELECT count(*) FROM patient_restriction WHERE patient_id = $2 AND organization_id = $3 AND restricted_by_account_id IS NULL)`,
		visitID, patientID, clinic.id).Scan(&visits, &jobs); err != nil {
		t.Fatal(err)
	}
	if visits != 1 || jobs != 1 {
		t.Errorf("kept visits %d, the job's restrictions %d; want 1 and 1", visits, jobs)
	}
	if code := clinic.admin.do(http.MethodDelete, "/clinic/patients/"+patientID+"/restriction", nil, nil); code != http.StatusForbidden {
		t.Errorf("the admin lifts the erasure's restriction: %d, want 403", code)
	}
}

// must200 GETs path as who and returns the body, failing unless it answers 200.
func must200(t *testing.T, who *api, path string) []byte {
	t.Helper()
	code, body := who.send(nil, http.MethodGet, path, nil)
	if code != http.StatusOK {
		t.Fatalf("GET %s = %d %s", path, code, body)
	}
	return body
}

// S3, D1, D5: the patient asks the clinic to rectify their record; 7 days and 1 day before its deadline the clinic's
// admin is warned by e-mail, naming the kind and the due date and never the patient; the admin answers it.
func TestPrivacy_TheClinicIsWarnedBeforeTheDeadlineAndAnswersARectification(t *testing.T) {
	clinic := newBookableClinic(t)
	p := newPatientWithProfile(t)
	p.bookFirstTime(clinic)
	r := p.fileRequest("rectification", clinic.id)

	for _, left := range []time.Duration{6*24*time.Hour + 12*time.Hour, 12 * time.Hour} {
		ageRequest(t, r.ID, left)
		runJob(t, datarequest.DeadlinesArgs{})
		text := mailTextWithin(t, clinic.adminAddr, deadlineSubject, time.Minute)
		for _, leak := range []string{p.name, "Syntheticova", p.addr, p.patientID} {
			if strings.Contains(text, leak) {
				t.Errorf("the warning %v ahead names the patient (%q):\n%s", left, leak, text)
			}
		}
		if !strings.Contains(strings.ToLower(text), "rectif") {
			t.Errorf("the warning %v ahead does not name the kind:\n%s", left, text)
		}
	}
	var answered dataRequest
	clinic.admin.mustWith(ifMatch(r.Version), http.StatusOK, http.MethodPost, "/clinic/data-requests/"+r.ID+"/answer",
		map[string]string{"status": "fulfilled", "code": "done"}, &answered)
	if answered.Status != "fulfilled" {
		t.Errorf("answered = %+v", answered)
	}
}

// D4 in the web app (privacy.spec.ts): the clinic's second admin opens the patient's restriction request from the
// queue, restricts the record, and the desk's patient search leaves the patient out until the restriction is lifted.
func TestPrivacy_TheAdminHandlesARestrictionInTheWebApp(t *testing.T) {
	clinic := newBookableClinic(t)
	adminAddr := email(t, "admin2")
	clinic.admin.must(http.StatusCreated, http.MethodPost, "/invites", map[string]string{"email": adminAddr, "role": "admin"}, nil)
	_, adminTOTP := enrolFromInvite(t, adminAddr)
	p := newPatientWithProfile(t)
	p.bookFirstTime(clinic)
	p.fileRequest("restriction", clinic.id)
	runDeskJourney(t, adminAddr, adminTOTP, "privacy.spec.ts", "E2E_PRIVACY_PATIENT="+p.name)
}

// S4, D2, D7: the patient files an access request to their account; the worker builds the archive and the patient
// downloads it. It holds their account and profile and, for the clinic, the record (with the visit) and the chat.
func TestPrivacy_APatientFilesAnAccessRequestAndDownloadsTheArchive(t *testing.T) {
	clinic := newBookableClinic(t)
	p := newPatientWithProfile(t)
	visit := p.bookFirstTime(clinic)
	chat := openChat(t, p.api, clinic.id, publicRole(t, clinic.id))
	p.must(http.StatusCreated, http.MethodPost, "/conversations/"+chat+"/messages",
		map[string]string{"client_message_id": newUUID(t), "body": "Synthetic placeholder question"}, nil)

	r := p.fileRequest("access", "")
	var export struct{ Status, ExpiresAt string }
	eventually(t, "the export is ready", func() bool {
		p.must(http.StatusOK, http.MethodGet, "/me/data-requests/"+r.ID+"/export", nil, &export)
		if export.Status == "failed" {
			t.Fatal("the export failed")
		}
		return export.Status == "ready"
	})
	if expires, err := time.Parse(time.RFC3339, export.ExpiresAt); err != nil || expires.Before(clock.System{}.Now().Add(6*24*time.Hour)) {
		t.Errorf("expires %q (%v), want 7 days from when it was ready (S13)", export.ExpiresAt, err)
	}
	code, body := p.send(nil, http.MethodGet, "/me/data-requests/"+r.ID+"/export/archive", nil)
	if code != http.StatusOK {
		t.Fatalf("the archive = %d %s", code, body)
	}
	zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		t.Fatalf("the archive is no zip: %v", err)
	}
	files := map[string]string{}
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		b, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			t.Fatal(err)
		}
		files[f.Name] = string(b)
	}
	dir := "clinics/" + clinic.id + "/"
	for name, want := range map[string]string{
		"manifest.txt": dir, "account.json": p.addr, "patient.json": p.patientID,
		dir + "record.json": visit.ID, dir + "chat.json": "Synthetic placeholder question",
	} {
		if !strings.Contains(files[name], want) {
			t.Errorf("%s does not hold %q:\n%.400s", name, want, files[name])
		}
	}
	var mine []dataRequest
	p.must(http.StatusOK, http.MethodGet, "/me/data-requests", nil, &mine)
	if len(mine) != 1 || mine[0].Status != "fulfilled" {
		t.Errorf("the request after the export = %+v, want fulfilled", mine)
	}
}
