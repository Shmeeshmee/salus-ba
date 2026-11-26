package api_test

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/api"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// roleWorld is the role matrix's world (docs/clinic-cms/WAVE7.md RD1): two clinics on one database, row-level security
// on as in production, each with its admin, practitioner and receptionist, two patients with their visits and the
// clinic's records of every kind the fixture can make, and a signed-in account without a patient profile.
type roleWorld struct {
	owner    *pgxpool.Pool
	count    func(string, ...any) int
	h        http.Handler
	a, b     *matrixClinic
	stranger *client
	out      *client // signed out
}

// matrixClinic is one clinic of the matrix. ids holds a real resource of each kind, by kind: the clinic's own, read
// by its staff; the patient's, read by them; neighbour's ids are the clinic's other patient's.
type matrixClinic struct {
	admin, practitioner, receptionist *client
	patient, neighbour                *matrixPatient
	ids                               map[string]string
	org, tag                          string
}

// matrixPatient is an app patient of a clinic with a booked visit and the records that name them.
type matrixPatient struct {
	c    *client
	ids  map[string]string
	data []string // their name, birth date and phone: no caller but them and their clinic's staff may see it
}

func newRoleWorld(t *testing.T) *roleWorld {
	t.Helper()
	cfg := testConfig(t)
	owner := ownerPool(t, cfg)
	count := openDB(t, cfg)
	cfg.DatabaseURL = pgtest.AppURL(t, cfg.DatabaseURL) // row-level security on, as in production
	cfg.Analytics.PlatformEnabled = true                // the platform view's routes are served, to be held too
	app, err := api.New(t.Context(), cfg, testLiveKit(), testObjectStore(t), testDrafter(), testSummarizer(), slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(app.Close)
	m := &roleWorld{owner: owner, count: count, h: app.Handler, out: &client{t: t, h: app.Handler}}
	m.exec(t, `INSERT INTO subscription_plan (code, name, price_per_seat_minor) VALUES ('basic', 'Basic', 10000)`)
	m.a = m.clinic(t, "c0000000-0000-4000-8000-0000000000a1", "alfa")
	m.b = m.clinic(t, "c0000000-0000-4000-8000-0000000000b2", "bravo")
	m.stranger = m.signUp(t, "stranger")
	return m
}

func (m *roleWorld) exec(t *testing.T, q string, args ...any) {
	t.Helper()
	if _, err := m.owner.Exec(t.Context(), q, args...); err != nil {
		t.Fatalf("%s: %v", q, err)
	}
}

func (m *roleWorld) id(t *testing.T, q string, args ...any) string {
	t.Helper()
	var id string
	if err := m.owner.QueryRow(t.Context(), q, args...).Scan(&id); err != nil {
		t.Fatalf("%s: %v", q, err)
	}
	return id
}

// created sends the request as c, wants want, and returns the answer's id.
func created(t *testing.T, c *client, want int, method, path, body string) string {
	t.Helper()
	rec := c.do(method, path, body)
	if rec.Code != want {
		t.Fatalf("%s %s = %d %s, want %d", method, path, rec.Code, rec.Body, want)
	}
	var out struct {
		ID      string `json:"id"`
		DraftID string `json:"draft_id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil || out.ID+out.DraftID == "" {
		t.Fatalf("%s %s = %d %s: no id", method, path, rec.Code, rec.Body)
	}
	return out.ID + out.DraftID
}

func (m *roleWorld) signUp(t *testing.T, name string) *client {
	t.Helper()
	c := &client{t: t, h: m.h}
	email := name + "@example.test"
	c.do(http.MethodPost, "/auth/register", `{"email":"`+email+`","password":"`+testPassword+`"}`)
	var login struct{ Token string }
	decode(t, c.do(http.MethodPost, "/auth/login", `{"email":"`+email+`","password":"`+testPassword+`"}`), &login)
	c.bearer = login.Token
	return c
}

// clinic builds a clinic with its staff, its settings and one record of each kind.
func (m *roleWorld) clinic(t *testing.T, org, tag string) *matrixClinic {
	t.Helper()
	c := &matrixClinic{org: org, tag: tag, ids: map[string]string{"org": org}}
	m.exec(t, `INSERT INTO organization (id, name, listed, legal_name, tax_id, region_id, city_id, location_since, address_line)
		VALUES ($1, 'Synthetic clinic '||$2, true, 'Synthetic '||$2||' d.o.o.', $3, 'BA-09', 'Q1270190', now(), 'Synthetic street 1')`,
		org, tag, map[string]string{"alfa": "4200000000001", "bravo": "4200000000002"}[tag])
	m.exec(t, `INSERT INTO organization_appointment_settings (organization_id, waitlist_enabled) VALUES ($1, true)`, org)
	m.exec(t, `INSERT INTO clinic_subscription (organization_id, plan_code, status, trial_ends_at) VALUES ($1, 'basic', 'trial', now() + interval '30 days')`, org)
	exec := func(q string, args ...any) { m.exec(t, q, args...) }
	exec(`INSERT INTO invite (organization_id, expires_at, role, email) VALUES ($1, now() + interval '72 hours', 'admin', $2)`, org, tag+".admin@example.test")
	c.admin = &client{t: t, h: m.h}
	c.admin.bearer = accept(t, c.admin, issueToken(t, exec, tag+".admin@example.test"))
	for _, s := range []struct {
		who  **client
		role string
	}{{&c.practitioner, "practitioner"}, {&c.receptionist, "reception"}} {
		email := tag + "." + s.role + "@example.test"
		names := "" // a receptionist is invited without a name
		if s.role == "practitioner" {
			names = `,"givenName":"Synthetic","familyName":"Example"`
		}
		c.ids["invite"] = created(t, c.admin, http.StatusCreated, http.MethodPost, "/invites", `{"email":"`+email+`","role":"`+s.role+`"`+names+`}`)
		*s.who = &client{t: t, h: m.h}
		(*s.who).bearer = accept(t, *s.who, issueToken(t, exec, email))
	}
	c.ids["role"] = m.id(t, `SELECT r.id::text FROM practitioner_role r WHERE r.organization_id = $1`, org)
	c.ids["practitioner"] = m.id(t, `SELECT practitioner_id::text FROM practitioner_role WHERE id = $1`, c.ids["role"])
	c.ids["schedule"] = m.id(t, `INSERT INTO schedule (organization_id, practitioner_role_id, slot_minutes) VALUES ($1, $2, 30) RETURNING id::text`, org, c.ids["role"])
	c.ids["service"] = created(t, c.admin, http.StatusCreated, http.MethodPost, "/clinic/services",
		`{"name":"Synthetic exam","priceMinor":5000,"modes":["video","in-person"],"active":true,"translations":[]}`)
	c.ids["slot"] = m.slot(t, c, "72 hours")
	m.records(t, c)
	c.patient = m.patient(t, c, "patient", 24)
	c.neighbour = m.patient(t, c, "neighbour", 26)
	for k, v := range c.patient.ids {
		c.ids[k] = v // the clinic's records of its patient
	}
	return c
}

// slot adds a free slot to the clinic's schedule, after from now.
func (m *roleWorld) slot(t *testing.T, c *matrixClinic, after string) string {
	t.Helper()
	return m.id(t, `INSERT INTO slot (schedule_id, organization_id, period, visit_mode)
		VALUES ($1, $2, tstzrange(date_trunc('hour', now()) + $3::interval, date_trunc('hour', now()) + $3::interval + interval '30 minutes'), 'in-person')
		RETURNING id::text`, c.ids["schedule"], c.org, after)
}

// records are the clinic's own records that name no patient.
func (m *roleWorld) records(t *testing.T, c *matrixClinic) {
	t.Helper()
	c.ids["location"] = created(t, c.admin, http.StatusCreated, http.MethodPost, "/clinic/locations", `{"name":"Synthetic site","active":true}`)
	c.ids["room"] = created(t, c.admin, http.StatusCreated, http.MethodPost, "/clinic/locations/"+c.ids["location"]+"/rooms", `{"name":"Room 1","active":true}`)
	c.ids["device"] = created(t, c.admin, http.StatusCreated, http.MethodPost, "/clinic/locations/"+c.ids["location"]+"/devices", `{"name":"Device 1","active":true}`)
	c.ids["webhook"] = created(t, c.admin, http.StatusCreated, http.MethodPost, "/clinic/webhooks", `{"url":"https://hooks.example.test/salus","events":["message.created"]}`)
	c.ids["template"] = created(t, c.practitioner, http.StatusCreated, http.MethodPost, "/document-templates", templateBody)
	c.ids["questionnaire"] = publishQuestionnaire(t, c.practitioner, qrBody)
	c.ids["files"] = publishQuestionnaire(t, c.practitioner, filesBody)
	c.ids["export"] = created(t, c.admin, http.StatusAccepted, http.MethodPost, "/clinic/export", "")
	c.ids["invoice"] = m.id(t, `INSERT INTO subscription_invoice (organization_id, number, period_start, period_end, seats, amount_minor, issued_at, due_at)
		VALUES ($1, $2, '2026-12-01', '2027-01-01', 1, 10000, now(), now() + interval '14 days') RETURNING id::text`,
		c.org, map[string]int{"alfa": 900001, "bravo": 900002}[c.tag])
}

// patient signs up an app patient of the clinic who books a visit hours ahead, and gives them the records of every
// kind that name a patient.
func (m *roleWorld) patient(t *testing.T, c *matrixClinic, who string, hours int) *matrixPatient {
	t.Helper()
	p := &matrixPatient{c: m.signUp(t, c.tag+"."+who), ids: map[string]string{}}
	family := "Matrix" + c.tag + who
	birth := map[string]string{"patient": "1971-01-0", "neighbour": "1972-02-0"}[who] + map[string]string{"alfa": "1", "bravo": "2"}[c.tag]
	phone := "+387610" + map[string]string{"alfa": "1", "bravo": "2"}[c.tag] + map[string]string{"patient": "1111", "neighbour": "2222"}[who]
	p.data = []string{family, birth, phone}
	p.ids["patient"] = created(t, p.c, http.StatusCreated, http.MethodPut, "/me/patient",
		`{"givenName":"Synthetic","familyName":"`+family+`","birthDate":"`+birth+`","phone":"`+phone+`"}`)
	p.ids["consent"] = created(t, p.c, http.StatusCreated, http.MethodPost, "/consents", `{"organizationId":"`+c.org+`"}`)
	slot := m.slot(t, c, strconv.Itoa(hours)+" hours")
	p.ids["visit"] = created(t, p.c, http.StatusCreated, http.MethodPost, "/appointments", `{"slotId":"`+slot+`"}`)
	past := m.id(t, `INSERT INTO appointment (patient_id, practitioner_role_id, organization_id, slot_id, period)
		VALUES ($1, $2, $3, gen_random_uuid(), tstzrange(now() - $4::interval, now() - $4::interval + interval '30 minutes')) RETURNING id::text`,
		p.ids["patient"], c.ids["role"], c.org, strconv.Itoa(hours*2)+" hours") // a past visit, apart from the other patient's
	p.ids["encounter"] = m.id(t, `INSERT INTO encounter (appointment_id, organization_id, patient_id, practitioner_role_id, status, period_start, period_end)
		SELECT id, organization_id, patient_id, practitioner_role_id, 'in-progress', lower(period), NULL
		FROM appointment WHERE id = $1 RETURNING id::text`, past) // in progress while the practitioner records, then finished
	p.ids["allergy"] = created(t, c.practitioner, http.StatusCreated, http.MethodPost, "/encounters/"+p.ids["encounter"]+"/allergies",
		`{"text":"Placeholder substance","clinicalStatus":"active"}`)
	p.ids["medication"] = created(t, c.practitioner, http.StatusCreated, http.MethodPost, "/encounters/"+p.ids["encounter"]+"/medications",
		`{"text":"Placeholder medication","status":"active","dosage":{}}`)
	m.exec(t, `UPDATE encounter SET status = 'finished', period_end = period_start + interval '20 minutes' WHERE id = $1`, p.ids["encounter"])
	p.ids["conversation"] = created(t, p.c, http.StatusCreated, http.MethodPost, "/conversations",
		`{"organization_id":"`+c.org+`","practitioner_role_id":"`+c.ids["role"]+`"}`)
	p.ids["receipt"] = created(t, c.receptionist, http.StatusCreated, http.MethodPost, "/clinic/receipts",
		`{"kind":"receipt","appointmentId":"`+p.ids["visit"]+`","paymentMethod":"cash","lines":[{"serviceId":"`+c.ids["service"]+`","quantity":1}]}`)
	p.ids["quote"] = created(t, c.practitioner, http.StatusCreated, http.MethodPost, "/clinic/quotes",
		`{"patientId":"`+p.ids["patient"]+`","appointmentId":"`+p.ids["visit"]+`","note":"Placeholder note","lines":[{"serviceId":"`+c.ids["service"]+`","quantity":1}]}`)
	p.ids["report"] = created(t, c.practitioner, http.StatusCreated, http.MethodPost, "/clinic/appointments/"+p.ids["visit"]+"/diagnostic-reports",
		`{"title":"Placeholder blood count","results":[]}`)
	c.practitioner.ifMatch = `W/"1"`
	if rec := c.practitioner.do(http.MethodPost, "/clinic/diagnostic-reports/"+p.ids["report"]+"/release", ""); rec.Code != http.StatusOK {
		t.Fatalf("release = %d %s", rec.Code, rec.Body)
	}
	c.practitioner.ifMatch = ""
	p.ids["plan"] = created(t, c.practitioner, http.StatusCreated, http.MethodPost, "/clinic/care-plans",
		`{"patientId":"`+p.ids["patient"]+`","title":"Synthetic plan","steps":[{"serviceId":"`+c.ids["service"]+`","dayOffset":0,"visitMode":"video"}]}`)
	p.ids["response"] = created(t, c.practitioner, http.StatusCreated, http.MethodPost, "/appointments/"+p.ids["visit"]+"/questionnaires",
		`{"questionnaireId":"`+c.ids["questionnaire"]+`"}`)
	p.ids["binary"] = m.upload(t, p, created(t, c.practitioner, http.StatusCreated, http.MethodPost, "/appointments/"+p.ids["visit"]+"/questionnaires",
		`{"questionnaireId":"`+c.ids["files"]+`"}`)+"/items/f1/attachments", "/questionnaire-responses/")
	p.ids["attachment"] = m.upload(t, p, p.ids["conversation"]+"/attachments", "/conversations/")
	p.ids["message"] = created(t, p.c, http.StatusCreated, http.MethodPost, "/conversations/"+p.ids["conversation"]+"/messages",
		`{"client_message_id":"c1000000-0000-4000-8000-0000000000c1","body":"Synthetic placeholder question","attachment_ids":["`+p.ids["attachment"]+`"]}`)
	p.ids["waitlist"] = created(t, p.c, http.StatusCreated, http.MethodPost, "/waitlist",
		`{"organization_id":"`+c.org+`","practitioner_role_id":"`+c.ids["role"]+`","earliest":"`+day(5)+`","latest":"`+day(20)+`"}`)
	p.ids["request"] = created(t, p.c, http.StatusCreated, http.MethodPost, "/me/data-requests",
		`{"kind":"access","target":"clinic","organizationId":"`+c.org+`"}`)
	return p
}

// upload sends a synthetic PDF as the patient to the files route at prefix+path and returns the file's id.
func (m *roleWorld) upload(t *testing.T, p *matrixPatient, path, prefix string) string {
	t.Helper()
	rec := send(p.c, http.MethodPost, prefix+path, filePDF(), map[string]string{
		"Content-Disposition": "attachment; filename*=UTF-8''Synthetic%20placeholder%20scan.pdf", "Content-Type": "application/octet-stream",
	})
	var out struct{ ID string }
	if rec.Code != http.StatusCreated || json.Unmarshal(rec.Body.Bytes(), &out) != nil || out.ID == "" {
		t.Fatalf("the upload to %s = %d %s", prefix+path, rec.Code, rec.Body)
	}
	return out.ID
}

// day is the date days from today, in UTC.
func day(days int) string {
	return clock.System{}.Now().UTC().AddDate(0, 0, days).Format(time.DateOnly)
}
