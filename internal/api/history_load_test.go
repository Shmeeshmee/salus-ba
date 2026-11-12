//go:build loadtest

package api

import (
	"context"
	"fmt"
	"os"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/binary"
	"github.com/Shmeeshmee/salus/internal/composition"
	"github.com/Shmeeshmee/salus/internal/encounter"
	"github.com/Shmeeshmee/salus/internal/history"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/internal/practitioner"
	"github.com/Shmeeshmee/salus/internal/practitionerrole"
	"github.com/Shmeeshmee/salus/internal/questionnaireresponse"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/keyset"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// The timeline's target (SAL-39, docs/plans/patient-history.md): a page of a patient's history with about 2,000
// items, p95 under 50 ms. make bench-history runs this against the compose Postgres, on the timeline as New composes
// it for the api (every source real: its scope, its access rule, its audit write); the numbers go in
// docs/plans/patient-history.PROGRESS.md.
//
// The patient: historyVisits visits at each of two clinics, each with a signed document and a completed
// questionnaire response holding one clean file: 8 items a visit pair's worth, historyVisits*8 = 2,000 in all. Around
// them, historyOthers other patients with historyOtherVisits visits each and the same records, so every query has
// other rows to pass over.

const (
	historyVisits      = 250 // per clinic
	historyOthers      = 400
	historyOtherVisits = 20
	historySamples     = 300
	historyPage        = 30
)

// historyClinic is a seeded clinic's ids.
type historyClinic struct{ id, role, template, questionnaire, series string }

func TestHistoryLoad(t *testing.T) {
	owner, app := pgtest.NewAppDatabase(t)
	ctx := t.Context()
	began := loadWall.Now()
	clinics := [2]historyClinic{seedHistoryClinic(t, owner, "A"), seedHistoryClinic(t, owner, "B")}
	me := seedHistoryPatient(t, owner, clinics, 0, historyVisits)
	others := historyOthers
	if s := os.Getenv("HISTORY_LOAD_OTHERS"); s != "" {
		others, _ = strconv.Atoi(s)
	}
	for i := 1; i <= others; i++ {
		seedHistoryPatient(t, owner, clinics, i, historyOtherVisits/2)
	}
	if _, err := owner.Exec(ctx, `VACUUM (ANALYZE)`); err != nil {
		t.Fatal(err)
	}
	t.Logf("seeded in %s", loadWall.Now().Sub(began).Round(time.Second))

	svc := historyService(app, nil)
	all, pages := walkHistory(t, svc, me)
	if len(all) != historyVisits*8 {
		types := map[history.ItemType]int{}
		for _, it := range all {
			types[it.Type]++
		}
		t.Fatalf("the walk gave %d items (%v), want %d", len(all), types, historyVisits*8)
	}
	t.Logf("the whole history: %d items in %d pages of %d", len(all), pages, historyPage)

	spent := map[string]time.Duration{}
	timed := historyService(app, spent)
	for range historySamples {
		if _, err := timed.Timeline(ctx, me, history.Filter{}, history.Cursor{}, historyPage); err != nil {
			t.Fatal(err)
		}
	}
	for name, d := range spent {
		t.Logf("first page, mean time in %s: %s", name, (d / historySamples).Round(10*time.Microsecond))
	}

	cursors := []history.Cursor{{}}
	for i := historyPage - 1; i < len(all); i += historyPage {
		cursors = append(cursors, history.CursorAt(all[i]))
	}
	for name, pick := range map[string]func(i int) (history.Cursor, history.Filter){
		"first page": func(int) (history.Cursor, history.Filter) { return history.Cursor{}, history.Filter{} },
		"any page":   func(i int) (history.Cursor, history.Filter) { return cursors[i%len(cursors)], history.Filter{} },
		"first page of one clinic": func(int) (history.Cursor, history.Filter) {
			return history.Cursor{}, history.Filter{OrganizationID: clinics[1].id}
		},
		"first page of the documents": func(int) (history.Cursor, history.Filter) {
			return history.Cursor{}, history.Filter{Types: []history.ItemType{history.TypeDocument}}
		},
	} {
		var d []time.Duration
		for i := range historySamples {
			cur, f := pick(i)
			start := loadWall.Now()
			if _, err := svc.Timeline(ctx, me, f, cur, historyPage); err != nil {
				t.Fatal(err)
			}
			d = append(d, loadWall.Now().Sub(start))
		}
		slices.Sort(d)
		t.Logf("%s, %d samples: p50 %s p95 %s p99 %s max %s", name, len(d), historyPct(d, 50), historyPct(d, 95), historyPct(d, 99), d[len(d)-1])
	}
}

// historyService is the timeline as New builds it, on the app pool.
func historyService(app *pgxpool.Pool, spent map[string]time.Duration) *history.Service {
	tx := postgres.NewTransactor(app)
	audit := auditevent.NewService(auditevent.NewPostgresRepository(app), tx, clock.System{})
	clinics := organization.NewService(organization.NewPostgresRepository(app), audit, tx, clock.System{})
	roles := practitionerrole.NewService(practitionerrole.Ports{
		Repo: practitionerrole.NewPostgresRepository(app), Audit: audit, Tx: tx, Clock: clock.System{}, Clinics: clinics,
		People: practitioner.NewService(practitioner.NewPostgresRepository(app), audit, tx),
	})
	encounters := encounter.NewService(encounter.Ports{Repo: encounter.NewPostgresRepository(app), Audit: audit, Tx: tx, Clock: clock.System{}})
	responses := questionnaireresponse.NewService(questionnaireresponse.Ports{
		Repo: questionnaireresponse.NewPostgresRepository(app), Audit: audit, Tx: tx, Clock: clock.System{},
	})
	ports := history.Ports{
		Visits: encounters,
		Documents: composition.NewService(composition.Ports{
			Repo: composition.NewPostgresRepository(app), Encounters: encounters, Audit: audit, Tx: tx, Clock: clock.System{},
		}),
		Responses: responses,
		Files:     binary.NewService(binary.Ports{Repo: binary.NewPostgresRepository(app), Holder: responses, Audit: audit, Tx: tx, Clock: clock.System{}}),
		Clinics:   clinics,
		Roles:     roles,
		Audit:     audit,
		Tx:        tx,
	}
	if spent != nil {
		ports = timedPorts(ports, spent)
	}
	return history.NewService(ports)
}

// walkHistory reads the patient's whole history page by page and checks it is newest first without a repeat.
func walkHistory(t *testing.T, svc *history.Service, c appointment.Caller) ([]history.Item, int) {
	t.Helper()
	var all []history.Item
	var cur history.Cursor
	pages := 0
	seen := map[string]bool{}
	for {
		p, err := svc.Timeline(t.Context(), c, history.Filter{}, cur, historyPage)
		if err != nil {
			t.Fatal(err)
		}
		pages++
		for _, it := range p.Items {
			if seen[it.Reference] || (len(all) > 0 && it.Date.After(all[len(all)-1].Date)) {
				t.Fatalf("item %s repeated or out of order", it.Reference)
			}
			seen[it.Reference] = true
			all = append(all, it)
		}
		if p.Next == "" {
			return all, pages
		}
		if cur, err = history.ParseCursor(p.Next); err != nil {
			t.Fatal(err)
		}
	}
}

// historyStart is the first day of patient i's visits: the patient measured in 1980, each other one 22 days (their 10
// visits, every two days) after the one before from 2000.
func historyStart(i int) time.Time {
	if i == 0 {
		return time.Date(1980, 1, 1, 9, 0, 0, 0, time.UTC)
	}
	return time.Date(2000, 1, 1, 9, 0, 0, 0, time.UTC).AddDate(0, 0, 2*(historyOtherVisits/2+1)*i)
}

func historyPct(d []time.Duration, p int) time.Duration {
	return d[min(len(d)-1, len(d)*p/100)].Round(10 * time.Microsecond)
}

// seedHistoryClinic makes a clinic with a practitioner role with a specialty, a document template and an active
// questionnaire, as the owner.
func seedHistoryClinic(t *testing.T, owner *pgxpool.Pool, name string) historyClinic {
	t.Helper()
	var c historyClinic
	err := owner.QueryRow(t.Context(), `WITH o AS (INSERT INTO organization (name) VALUES ('Synthetic clinic `+name+`') RETURNING id),
		p AS (INSERT INTO practitioner (organization_id, given_name, family_name) SELECT id, 'Cy', 'Example' FROM o RETURNING id, organization_id),
		r AS (INSERT INTO practitioner_role (organization_id, practitioner_id, specialty_code) SELECT organization_id, id, '394579002' FROM p RETURNING id, organization_id),
		tp AS (INSERT INTO document_template (organization_id, title, fields) SELECT organization_id, 'Placeholder report', '[{"key":"reason"}]' FROM r RETURNING id),
		q AS (INSERT INTO questionnaire (organization_id, series_id, version, status, title, published_at)
			SELECT organization_id, gen_random_uuid(), 1, 'active', 'Placeholder questionnaire', now() FROM r RETURNING id, series_id)
		SELECT r.organization_id::text, r.id::text, tp.id::text, q.id::text, q.series_id::text FROM r, tp, q`).
		Scan(&c.id, &c.role, &c.template, &c.questionnaire, &c.series)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// seedHistoryPatient makes patient i with a consent at both clinics and visits of them at each, every visit with an
// encounter, a signed document and a completed response with one clean file, as the owner; it returns the
// patient's caller.
func seedHistoryPatient(t *testing.T, owner *pgxpool.Pool, clinics [2]historyClinic, i, visits int) appointment.Caller {
	t.Helper()
	ctx := t.Context()
	var c appointment.Caller
	err := owner.QueryRow(ctx, `WITH a AS (INSERT INTO account (email, password_hash, role) VALUES ('history'||$1::text||'@example.test', 'h', 'patient') RETURNING id),
		p AS (INSERT INTO patient (account_id, given_name, family_name, birth_date) SELECT id, 'Synthetic', 'Patient', '1990-01-01' FROM a RETURNING id, account_id)
		SELECT account_id::text, id::text FROM p`, strconv.Itoa(i)).Scan(&c.AccountID, &c.PatientID)
	if err != nil {
		t.Fatal(err)
	}
	for k, cl := range clinics {
		stmts := []string{
			`INSERT INTO consent (patient_id, organization_id, status, date_time) VALUES ($1, $2, 'active', '2020-01-01')`,
			// A visit every two days from the patient's own start (no two patients' overlap: a role is never double
			// booked), the second clinic's an hour later.
			`INSERT INTO appointment (patient_id, practitioner_role_id, organization_id, slot_id, period, status)
				SELECT $1, $3, $2, gen_random_uuid(), tstzrange(t, t + interval '30 minutes'), 'fulfilled'
				FROM (SELECT $6::timestamptz + make_interval(days => 2 * n, hours => $4) AS t FROM generate_series(1, $5) n) s`,
			`INSERT INTO encounter (appointment_id, organization_id, patient_id, practitioner_role_id, status, period_start, period_end)
				SELECT id, organization_id, patient_id, practitioner_role_id, 'finished', lower(period), upper(period)
				FROM appointment WHERE patient_id = $1 AND organization_id = $2 AND $3::uuid IS NOT NULL AND $4::int IS NOT NULL AND $5::int IS NOT NULL`,
			`INSERT INTO document_draft (encounter_id, template_id, status, fields, signed_at, pdf_size, signer_role_id, document_reference_id,
					provenance_id, pdf_kek_id, pdf_sha1, pdf_wrapped_key, title)
				SELECT e.id, '` + cl.template + `', 'final', '[{"key":"reason"}]', e.period_end, 1000, e.practitioner_role_id, gen_random_uuid(),
					gen_random_uuid(), 'k1', '\x0101010101010101010101010101010101010101', '\x01', 'Placeholder report'
				FROM encounter e WHERE e.patient_id = $1 AND e.organization_id = $2 AND $3::uuid IS NOT NULL AND $4::int IS NOT NULL AND $5::int IS NOT NULL`,
			`WITH r AS (INSERT INTO questionnaire_response (organization_id, appointment_id, patient_id, practitioner_role_id, questionnaire_id, series_id,
					source, status, due_at, authored_at, answers)
				SELECT a.organization_id, a.id, a.patient_id, a.practitioner_role_id, '` + cl.questionnaire + `', '` + cl.series + `', 'default', 'completed',
					lower(a.period), lower(a.period) - interval '1 hour', '[]'
				FROM appointment a WHERE a.patient_id = $1 AND a.organization_id = $2 AND $3::uuid IS NOT NULL AND $4::int IS NOT NULL AND $5::int IS NOT NULL
				RETURNING id, organization_id, patient_id, appointment_id),
			b AS (INSERT INTO "binary" (organization_id, patient_id, appointment_id, questionnaire_response_id, size, segment_size, link_id,
					content_type, title, status, scanned_at, kek_id, sha1, wrapped_key)
				SELECT organization_id, patient_id, appointment_id, id, 1000, 65536, 'files', 'application/pdf', 'Placeholder.pdf', 'clean', now(),
					'k1', '\x0101010101010101010101010101010101010101', '\x01' FROM r)
			SELECT 1`,
			// The response lists its file (a statement of its own: one statement does not see its own inserts).
			`UPDATE questionnaire_response q SET attachments = jsonb_build_array(jsonb_build_object('linkId', 'files', 'binaryId', b.id,
					'contentType', 'application/pdf', 'title', 'Placeholder.pdf', 'hash', 'AQEBAQEBAQEBAQEBAQEBAQEBAQE=', 'size', 1000, 'pending', false))
				FROM "binary" b WHERE q.id = b.questionnaire_response_id AND q.patient_id = $1 AND q.organization_id = $2
					AND $3::uuid IS NOT NULL AND $4::int IS NOT NULL AND $5::int IS NOT NULL`,
		}
		for n, stmt := range stmts {
			args := []any{c.PatientID, cl.id}
			switch n {
			case 0:
			case 1:
				args = append(args, cl.role, k, visits, historyStart(i))
			default:
				args = append(args, cl.role, k, visits)
			}
			if _, err := owner.Exec(ctx, stmt, args...); err != nil {
				t.Fatalf("patient %d, clinic %d, statement %d: %v", i, k, n, fmt.Errorf("%w", err))
			}
		}
	}
	return c
}

// timed* wrap the timeline's ports to add the time each one takes to spent, by port.
type timedVisits struct {
	history.Visits
	spent map[string]time.Duration
}

func (t timedVisits) PatientHistory(ctx context.Context, c appointment.Caller, org string, p keyset.Page) ([]encounter.Encounter, error) {
	defer t.add("visits", loadWall.Now())
	return t.Visits.PatientHistory(ctx, c, org, p)
}

func (t timedVisits) add(name string, began time.Time) { t.spent[name] += loadWall.Now().Sub(began) }

type timedDocuments struct {
	history.Documents
	timedVisits
}

func (t timedDocuments) PatientHistory(ctx context.Context, c appointment.Caller, org string, p keyset.Page) ([]composition.Document, error) {
	defer t.add("documents", loadWall.Now())
	return t.Documents.PatientHistory(ctx, c, org, p)
}

type timedResponses struct {
	history.Responses
	timedVisits
}

func (t timedResponses) PatientHistory(ctx context.Context, c appointment.Caller, org string, p keyset.Page) ([]questionnaireresponse.Response, error) {
	defer t.add("responses", loadWall.Now())
	return t.Responses.PatientHistory(ctx, c, org, p)
}

func (t timedResponses) PatientAttachments(ctx context.Context, c appointment.Caller, org string, p keyset.Page) ([]questionnaireresponse.Attached, error) {
	defer t.add("attachments", loadWall.Now())
	return t.Responses.PatientAttachments(ctx, c, org, p)
}

type timedFiles struct {
	history.Files
	timedVisits
}

func (t timedFiles) Clean(ctx context.Context, c binary.Caller, ids []string) ([]string, error) {
	defer t.add("files", loadWall.Now())
	return t.Files.Clean(ctx, c, ids)
}

type timedClinics struct {
	history.Clinics
	timedVisits
}

func (t timedClinics) Names(ctx context.Context, ids []string) (map[string]string, error) {
	defer t.add("clinics", loadWall.Now())
	return t.Clinics.Names(ctx, ids)
}

type timedRoles struct {
	history.Roles
	timedVisits
}

func (t timedRoles) LookupAll(ctx context.Context, ids map[string][]string) (map[string]practitionerrole.PractitionerRole, error) {
	defer t.add("roles", loadWall.Now())
	return t.Roles.LookupAll(ctx, ids)
}

type timedAudit struct {
	history.Auditor
	timedVisits
}

func (t timedAudit) RecordAll(ctx context.Context, es []auditevent.AuditEvent) error {
	defer t.add("audit", loadWall.Now())
	return t.Auditor.RecordAll(ctx, es)
}

func timedPorts(p history.Ports, spent map[string]time.Duration) history.Ports {
	base := timedVisits{Visits: p.Visits, spent: spent}
	return history.Ports{
		Visits: base, Documents: timedDocuments{p.Documents, base}, Responses: timedResponses{p.Responses, base},
		Files: timedFiles{p.Files, base}, Clinics: timedClinics{p.Clinics, base}, Roles: timedRoles{p.Roles, base},
		Audit: timedAudit{p.Audit, base}, Tx: p.Tx,
	}
}
