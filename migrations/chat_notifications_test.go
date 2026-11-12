package migrations_test

import (
	"context"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// notificationRows is the seeded state of 00043: clinics A and B with chat settings, a webhook endpoint each with a
// dead letter, and the two patients' accounts with a device each.
type notificationRows struct {
	endpoint map[string]string // by clinic
	dead     map[string]string // by clinic
	p        [2]patientRow
}

func seedNotifications(t *testing.T, owner *pgxpool.Pool) notificationRows {
	t.Helper()
	ctx := t.Context()
	seedDirectory(t, owner)
	r := notificationRows{endpoint: map[string]string{}, dead: map[string]string{}, p: seedPatients(t, owner)}
	for i, clinic := range []string{clinicA, clinicB} {
		var admin string
		if err := owner.QueryRow(ctx, `INSERT INTO account (email, password_hash, role, organization_id) VALUES ($1, 'h', 'admin', $2) RETURNING id`,
			[]string{"admin.a@example.test", "admin.b@example.test"}[i], clinic).Scan(&admin); err != nil {
			t.Fatal(err)
		}
		if _, err := owner.Exec(ctx, `INSERT INTO organization_chat_settings (organization_id, response_time_hours, email_after_minutes, hours)
			VALUES ($1, 24, 60, '[{"day":"mon","start":"08:00","end":"16:00"}]')`, clinic); err != nil {
			t.Fatal(err)
		}
		var endpoint, dead string
		if err := owner.QueryRow(ctx, `INSERT INTO webhook_endpoint (id, organization_id, created_by, url, events, secret_sealed)
			VALUES (gen_random_uuid(), $1, $2, 'https://hooks.example.test/salus', '{message.created}', '\x00') RETURNING id`, clinic, admin).Scan(&endpoint); err != nil {
			t.Fatal(err)
		}
		if err := owner.QueryRow(ctx, `INSERT INTO webhook_dead_letter (endpoint_id, organization_id, event_id, conversation_id, message_id, attempts, event_type)
			VALUES ($1, $2, gen_random_uuid(), gen_random_uuid(), gen_random_uuid(), 8, 'message.created') RETURNING id`, endpoint, clinic).Scan(&dead); err != nil {
			t.Fatal(err)
		}
		r.endpoint[clinic], r.dead[clinic] = endpoint, dead
		if _, err := owner.Exec(ctx, `INSERT INTO push_device (account_id, platform, token) VALUES ($1, 'android', $2)`,
			r.p[i].account, []string{"synthetic-token-a", "synthetic-token-b"}[i]); err != nil {
			t.Fatal(err)
		}
	}
	return r
}

// A clinic sees only its own settings, endpoints and dead letters; no other scope sees any.
func TestChatNotifications_TheClinicSeesOnlyItsSettingsWebhooksAndDeadLetters(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	r := seedNotifications(t, owner)

	for _, tt := range []struct {
		scope    map[string]string
		name     string
		clinics  []string
		endpoint []string
		dead     []string
	}{
		{name: "no scope"},
		{name: "clinic A", scope: map[string]string{"app.organization_id": clinicA}, clinics: []string{clinicA}, endpoint: []string{r.endpoint[clinicA]}, dead: []string{r.dead[clinicA]}},
		{name: "clinic B", scope: map[string]string{"app.organization_id": clinicB}, clinics: []string{clinicB}, endpoint: []string{r.endpoint[clinicB]}, dead: []string{r.dead[clinicB]}},
		{name: "a patient", scope: map[string]string{"app.patient_id": r.p[0].patient}},
	} {
		for q, want := range map[string][]string{
			"SELECT organization_id::text FROM organization_chat_settings": tt.clinics,
			"SELECT id::text FROM webhook_endpoint":                        tt.endpoint,
			"SELECT id::text FROM webhook_dead_letter":                     tt.dead,
		} {
			if got := idsIn(t, app, tt.scope, q); !slices.Equal(got, want) {
				t.Errorf("%s, %s: sees %v, want %v", tt.name, q, got, want)
			}
		}
	}
	err := asApp(t, app, map[string]string{"app.organization_id": clinicA}, func(ctx context.Context, db postgres.Querier) error {
		_, err := db.Exec(ctx, `INSERT INTO organization_chat_settings (organization_id, response_time_hours, email_after_minutes) VALUES ($1, 24, 60)`, clinicB)
		return err
	})
	if err == nil {
		t.Error("clinic A wrote clinic B's settings")
	}
}

// What the schema refuses: a webhook that is not https or names an unknown event, settings out of their ranges, a
// device of an unknown platform, a token registered twice.
func TestChatNotifications_RefusesInvalidRows(t *testing.T) {
	t.Parallel()
	owner, _ := pgtest.NewAppDatabase(t)
	r := seedNotifications(t, owner)
	var admin string
	if err := owner.QueryRow(t.Context(), `SELECT created_by::text FROM webhook_endpoint WHERE id = $1`, r.endpoint[clinicA]).Scan(&admin); err != nil {
		t.Fatal(err)
	}
	for name, sql := range map[string]string{
		"http":           `INSERT INTO webhook_endpoint (id, organization_id, created_by, url, events, secret_sealed) VALUES (gen_random_uuid(), '` + clinicA + `', '` + admin + `', 'http://hooks.example.test', '{message.created}', '\x00')`,
		"unknown event":  `INSERT INTO webhook_endpoint (id, organization_id, created_by, url, events, secret_sealed) VALUES (gen_random_uuid(), '` + clinicA + `', '` + admin + `', 'https://hooks.example.test', '{patient.read}', '\x00')`,
		"no event":       `INSERT INTO webhook_endpoint (id, organization_id, created_by, url, events, secret_sealed) VALUES (gen_random_uuid(), '` + clinicA + `', '` + admin + `', 'https://hooks.example.test', '{}', '\x00')`,
		"response time":  `UPDATE organization_chat_settings SET response_time_hours = 0`,
		"e-mail delay":   `UPDATE organization_chat_settings SET email_after_minutes = 1`,
		"hours object":   `UPDATE organization_chat_settings SET hours = '{}'`,
		"platform":       `INSERT INTO push_device (account_id, platform, token) VALUES ('` + r.p[0].account + `', 'web', 'synthetic-token-c')`,
		"token twice":    `INSERT INTO push_device (account_id, platform, token) VALUES ('` + r.p[0].account + `', 'ios', 'synthetic-token-b')`,
		"dead twice":     `INSERT INTO webhook_dead_letter (endpoint_id, organization_id, event_id, conversation_id, message_id, attempts, event_type) SELECT endpoint_id, organization_id, event_id, conversation_id, message_id, 1, event_type FROM webhook_dead_letter LIMIT 1`,
		"dead no answer": `INSERT INTO webhook_dead_letter (endpoint_id, organization_id, event_id, conversation_id, message_id, attempts, event_type, last_status) VALUES ('` + r.endpoint[clinicA] + `', '` + clinicA + `', gen_random_uuid(), gen_random_uuid(), gen_random_uuid(), 1, 'message.created', 0)`,
	} {
		if _, err := owner.Exec(t.Context(), sql); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

// The app role registers, moves and deletes devices, and deletes a clinic's endpoint with its dead letters; it
// cannot delete a dead letter or the settings, or change an endpoint (a change is a new endpoint with a new secret).
func TestChatNotifications_TheAppRolesGrants(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	r := seedNotifications(t, owner)
	allowed := []struct {
		scope map[string]string
		sql   string
	}{
		{nil, `UPDATE push_device SET account_id = '` + r.p[1].account + `', platform = 'ios', updated_at = now() WHERE token = 'synthetic-token-a'`},
		{nil, `DELETE FROM push_device WHERE token = 'synthetic-token-b'`},
		{map[string]string{"app.organization_id": clinicA}, `DELETE FROM webhook_endpoint WHERE id = '` + r.endpoint[clinicA] + `'`},
	}
	for _, a := range allowed {
		if err := asApp(t, app, a.scope, func(ctx context.Context, db postgres.Querier) error { _, err := db.Exec(ctx, a.sql); return err }); err != nil {
			t.Errorf("%s: %v", a.sql, err)
		}
	}
	var dead int
	if err := owner.QueryRow(t.Context(), `SELECT count(*) FROM webhook_dead_letter WHERE organization_id = $1`, clinicA).Scan(&dead); err != nil || dead != 0 {
		t.Errorf("clinic A's dead letters after its endpoint's deletion: %d (%v), want none", dead, err)
	}
	for _, sql := range []string{
		`DELETE FROM webhook_dead_letter`,
		`DELETE FROM organization_chat_settings`,
		`UPDATE webhook_endpoint SET url = 'https://elsewhere.example.test'`,
		`UPDATE webhook_endpoint SET include_clinical = true`,
	} {
		err := asApp(t, app, map[string]string{"app.organization_id": clinicB}, func(ctx context.Context, db postgres.Querier) error {
			_, err := db.Exec(ctx, sql)
			return err
		})
		if err == nil {
			t.Errorf("%s: allowed", sql)
		}
	}
}
