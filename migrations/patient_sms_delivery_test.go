package migrations_test

import (
	"context"
	"testing"

	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// docs/clinic-cms/WAVE4.md R3: a delivery goes to an account or, by SMS only, to a patient without one; never both,
// never neither. It goes with the patient's profile.
func TestNotificationDelivery_ToAnAccountOrBySMSToAPatient(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	p, _, _ := seedAppointments(t, owner)
	ctx := t.Context()
	var profile string
	if err := owner.QueryRow(ctx, `INSERT INTO patient (registered_by_organization_id, given_name, family_name, birth_date)
		VALUES ($1, 'Bo', 'Example', '1980-02-03') RETURNING id::text`, clinicA).Scan(&profile); err != nil {
		t.Fatal(err)
	}
	for i, tt := range []struct {
		name, account, patient, channel string
		ok                              bool
	}{
		{"an SMS to the patient", "", profile, "sms", true},
		{"an e-mail to the patient", "", profile, "email", false},
		{"to both", p[0].account, profile, "sms", false},
		{"to neither", "", "", "sms", false},
		{"an SMS to the account", p[0].account, "", "sms", true},
	} {
		err := asApp(t, app, nil, func(ctx context.Context, db postgres.Querier) error {
			_, err := db.Exec(ctx, `INSERT INTO notification_delivery (kind, source_id, organization_id, account_id, patient_id, channel)
				VALUES ('reminder', gen_random_uuid(), $1, NULLIF($2, '')::uuid, NULLIF($3, '')::uuid, $4)`, clinicA, tt.account, tt.patient, tt.channel)
			return err
		})
		if (err == nil) != tt.ok {
			t.Errorf("%d %s: err %v, want stored %v", i, tt.name, err, tt.ok)
		}
	}
	if _, err := owner.Exec(ctx, `DELETE FROM patient WHERE id = $1`, profile); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := owner.QueryRow(ctx, `SELECT count(*) FROM notification_delivery WHERE patient_id IS NOT NULL`).Scan(&n); err != nil || n != 0 {
		t.Errorf("%d deliveries of a deleted profile (%v), want none", n, err)
	}
}
