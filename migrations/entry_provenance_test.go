package migrations_test

import (
	"context"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// provenanceTables are the Provenance table of each entry table (00053) and its column naming the entry.
var provenanceTables = map[string][2]string{
	"allergy_intolerance":  {"allergy_intolerance_provenance", "allergy_intolerance_id"},
	"medication_statement": {"medication_statement_provenance", "medication_statement_id"},
}

// provenanceRows are the seeded Provenances: clinic A's record of its visit's entry, clinic B's of its visit's,
// clinic A's and clinic B's confirmations of the patient's own entry.
type provenanceRows struct {
	atA, atB, ownByA, ownByB string
}

// seedProvenances creates, as the owner, the rows of provenanceRows on the entries r.
func seedProvenances(t *testing.T, owner *pgxpool.Pool, h historyRows, r entryRows, table, column string) provenanceRows {
	t.Helper()
	insert := `INSERT INTO ` + table + ` (recorded, ` + column + `, practitioner_role_id, organization_id, target_version, activity, agent_type)
		VALUES (now(), $1, $2, $3, $4, $5, $6) RETURNING id`
	var p provenanceRows
	for _, row := range []struct {
		id             *string
		entry, clinic  string
		activity, kind string
		version        int
	}{
		{&p.atA, r.atA, clinicA, "CREATE", "author", 1},
		{&p.atB, r.atB, clinicB, "CREATE", "author", 1},
		{&p.ownByA, r.own, clinicA, "UPDATE", "verifier", 2},
		{&p.ownByB, r.own, clinicB, "UPDATE", "verifier", 3},
	} {
		if err := owner.QueryRow(t.Context(), insert, row.entry, h.r.roles[row.clinic], row.clinic, row.version, row.activity, row.kind).Scan(row.id); err != nil {
			t.Fatal(err)
		}
	}
	return p
}

// A Provenance names the clinic that wrote, so only the patient and that clinic, while it sees the entry, see it
// (access.Attributed): never another clinic that sees the entry in the profile or through a shared visit.
func TestEntryProvenance_OnlyThePatientAndTheWritingClinicSeeIt(t *testing.T) {
	t.Parallel()
	for entries, prov := range provenanceTables {
		t.Run(entries, func(t *testing.T) {
			t.Parallel()
			owner, app := pgtest.NewAppDatabase(t)
			h := seedHistory(t, owner)
			r := seedEntries(t, owner, h, entryTables[entries])
			p := seedProvenances(t, owner, h, r, prov[0], prov[1])
			query := `SELECT id::text FROM ` + prov[0]
			clinic := map[string]string{"app.organization_id": clinicA}
			for name, tt := range map[string]struct {
				scope map[string]string
				want  []string
			}{
				"no scope":   {nil, nil},
				"patient A":  {map[string]string{"app.patient_id": h.patients[0].patient}, sorted(p.atA, p.atB, p.ownByA, p.ownByB)},
				"patient B":  {map[string]string{"app.patient_id": h.patients[1].patient}, nil},
				"clinic A":   {clinic, sorted(p.atA, p.ownByA)},                                  // not B's confirmation of the profile entry
				"clinic B":   {map[string]string{"app.organization_id": clinicB}, sorted(p.atB)}, // revoked: its visit's entry, not the profile's
				"an account": {map[string]string{"app.account_id": h.patients[0].account}, nil},
			} {
				if got := idsIn(t, app, tt.scope, query); !slices.Equal(got, tt.want) {
					t.Errorf("%s: sees %v, want %v", name, got, tt.want)
				}
			}
			share(t, owner, h.consent, h.encounters[h.a.bPast])
			if got := idsIn(t, app, clinic, query); !slices.Equal(got, sorted(p.atA, p.ownByA)) {
				t.Errorf("clinic A with the visit at B shared: sees %v, want only its own (the entry, never clinic B's Provenance)", got)
			}
			ownerExec(t, owner, `UPDATE consent SET status = 'inactive', revoked_at = now() WHERE patient_id = $1 AND organization_id = $2`,
				h.patients[0].patient, clinicA)
			if got := idsIn(t, app, clinic, query); !slices.Equal(got, sorted(p.atA)) {
				t.Errorf("clinic A after the care consent's revocation: sees %v, want its visit's only (the profile is closed)", got)
			}
		})
	}
}

// A clinic writes only its own Provenance, on an entry it sees, at the entry's current version; the patient's scope
// writes none; nobody changes or deletes one.
func TestEntryProvenance_AClinicAppendsOnlyItsOwn(t *testing.T) {
	t.Parallel()
	for entries, prov := range provenanceTables {
		t.Run(entries, func(t *testing.T) {
			t.Parallel()
			owner, app := pgtest.NewAppDatabase(t)
			h := seedHistory(t, owner)
			r := seedEntries(t, owner, h, entryTables[entries])
			p := seedProvenances(t, owner, h, r, prov[0], prov[1])
			insert := `INSERT INTO ` + prov[0] + ` (recorded, ` + prov[1] + `, practitioner_role_id, organization_id, target_version, activity, agent_type)
				VALUES (now(), $1, $2, $3, $4, $5, 'author')`
			clinic := map[string]string{"app.organization_id": clinicA}
			patientA := map[string]string{"app.patient_id": h.patients[0].patient}
			for _, tt := range []struct {
				scope    map[string]string
				name     string
				entry    string
				org      string
				activity string
				version  int
				ok       bool
			}{
				{clinic, "its write on its visit's entry", r.atA, clinicA, "CREATE", 1, true},
				{clinic, "as another clinic", r.atA, clinicB, "CREATE", 1, false},
				{clinic, "on an entry it does not see", r.atB, clinicA, "CREATE", 1, false},
				{clinic, "at another version", r.atA, clinicA, "UPDATE", 2, false},
				{patientA, "the patient's scope", r.own, clinicA, "CREATE", 1, false},
			} {
				err := asApp(t, app, tt.scope, func(ctx context.Context, db postgres.Querier) error {
					_, err := db.Exec(ctx, insert, tt.entry, h.r.roles[tt.org], tt.org, tt.version, tt.activity)
					return err
				})
				if (err == nil) != tt.ok {
					t.Errorf("%s: err %v, want ok %v", tt.name, err, tt.ok)
				}
			}
			for name, stmt := range map[string]string{
				"update": `UPDATE ` + prov[0] + ` SET recorded = now() WHERE id = $1`,
				"delete": `DELETE FROM ` + prov[0] + ` WHERE id = $1`,
			} {
				for _, scope := range []map[string]string{clinic, patientA} {
					err := asApp(t, app, scope, func(ctx context.Context, db postgres.Querier) error {
						_, err := db.Exec(ctx, stmt, p.atA)
						return err
					})
					if sqlState(err) != "42501" {
						t.Errorf("%s in %v: %v, want insufficient_privilege (append-only)", name, scope, err)
					}
				}
			}
		})
	}
}

// A Provenance goes with its entry only: when the patient deletes their own unconfirmed entry, its Provenances (a
// confirmation the patient's change cleared since) go with it.
func TestEntryProvenance_GoesWithThePatientsDeletedEntry(t *testing.T) {
	t.Parallel()
	for entries, prov := range provenanceTables {
		t.Run(entries, func(t *testing.T) {
			t.Parallel()
			owner, app := pgtest.NewAppDatabase(t)
			h := seedHistory(t, owner)
			r := seedEntries(t, owner, h, entryTables[entries])
			seedProvenances(t, owner, h, r, prov[0], prov[1])
			err := asApp(t, app, map[string]string{"app.patient_id": h.patients[0].patient}, func(ctx context.Context, db postgres.Querier) error {
				tag, err := db.Exec(ctx, `DELETE FROM `+entries+` WHERE id = $1`, r.own)
				if err == nil && tag.RowsAffected() != 1 {
					t.Errorf("deleted %d entries, want 1", tag.RowsAffected())
				}
				return err
			})
			if err != nil {
				t.Fatal(err)
			}
			var n int
			if err := owner.QueryRow(t.Context(), `SELECT count(*) FROM `+prov[0]+` WHERE `+prov[1]+` = $1`, r.own).Scan(&n); err != nil {
				t.Fatal(err)
			}
			if n != 0 {
				t.Errorf("%d Provenances of the deleted entry left, want none", n)
			}
		})
	}
}

// A Provenance's codes are from their code systems, a creation is of the first version and nothing else is, and
// the agent is a practitioner role of the clinic named.
func TestEntryProvenance_HoldsAConsistentState(t *testing.T) {
	t.Parallel()
	for entries, prov := range provenanceTables {
		t.Run(entries, func(t *testing.T) {
			t.Parallel()
			owner, _ := pgtest.NewAppDatabase(t)
			h := seedHistory(t, owner)
			r := seedEntries(t, owner, h, entryTables[entries])
			insert := `INSERT INTO ` + prov[0] + ` (recorded, ` + prov[1] + `, practitioner_role_id, organization_id, target_version, activity, agent_type)
				VALUES (now(), $1, $2, $3, $4, $5, $6)`
			for _, tt := range []struct {
				name           string
				org            string
				activity, kind string
				version        int
			}{
				{"an unknown activity", clinicA, "DELETE", "author", 2},
				{"an unknown agent type", clinicA, "UPDATE", "enterer", 2},
				{"a creation of a later version", clinicA, "CREATE", "author", 2},
				{"an update of the first version", clinicA, "UPDATE", "author", 1},
				{"another clinic's role", clinicB, "CREATE", "author", 1},
			} {
				role := h.r.roles[clinicA]
				if _, err := owner.Exec(t.Context(), insert, r.atA, role, tt.org, tt.version, tt.activity, tt.kind); err == nil {
					t.Errorf("%s: accepted", tt.name)
				}
			}
		})
	}
}
