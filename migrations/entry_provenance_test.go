package migrations_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// provenanceTables are the Provenance table of each entry table (00053), its column naming the entry, and the
// entry's removal.
var provenanceTables = map[string][3]string{
	"allergy_intolerance":  {"allergy_intolerance_provenance", "allergy_intolerance_id", "verification_status = 'entered-in-error'"},
	"medication_statement": {"medication_statement_provenance", "medication_statement_id", "status = 'entered-in-error'"},
}

// provenanceRows are the seeded Provenances: clinic A's record of its visit's entry, clinic B's of its visit's,
// clinic A's and clinic B's confirmations of the patient's own entry, and the patient's removal of clinic B's entry.
type provenanceRows struct {
	atA, atB, ownByA, ownByB, byPatient string
}

// insertProvenance is an insert into the Provenance table: $1 the entry, $2 the role, $3 the clinic, $4 the
// patient (an agent is a role of a clinic or the patient), $5 the version, $6 the activity, $7 the agent type.
func insertProvenance(table, column string) string {
	return `INSERT INTO ` + table + ` (recorded, ` + column + `, practitioner_role_id, organization_id, patient_id, target_version, activity, agent_type)
		VALUES (now(), $1, $2, $3, $4, $5, $6, $7) RETURNING id`
}

// seedProvenances creates, as the owner, the rows of provenanceRows on the entries r.
func seedProvenances(t *testing.T, owner *pgxpool.Pool, h historyRows, r entryRows, table, column string) provenanceRows {
	t.Helper()
	var p provenanceRows
	for _, row := range []struct {
		role, clinic, patient any
		id                    *string
		entry                 string
		activity, kind        string
		version               int
	}{
		{h.r.roles[clinicA], clinicA, nil, &p.atA, r.atA, "CREATE", "author", 1},
		{h.r.roles[clinicB], clinicB, nil, &p.atB, r.atB, "CREATE", "author", 1},
		{h.r.roles[clinicA], clinicA, nil, &p.ownByA, r.own, "UPDATE", "verifier", 2},
		{h.r.roles[clinicB], clinicB, nil, &p.ownByB, r.own, "UPDATE", "verifier", 3},
		{nil, nil, h.patients[0].patient, &p.byPatient, r.atB, "NULLIFY", "author", 2},
	} {
		if err := owner.QueryRow(t.Context(), insertProvenance(table, column), row.entry, row.role, row.clinic, row.patient,
			row.version, row.activity, row.kind).Scan(row.id); err != nil {
			t.Fatal(err)
		}
	}
	return p
}

// A clinic's Provenance names the clinic that wrote, so only the patient and that clinic, while it sees the entry,
// see it (access.Attributed): never another clinic that sees the entry in the profile or through a shared visit. The
// patient's Provenances are the patient's scope's only.
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
				"patient A":  {map[string]string{"app.patient_id": h.patients[0].patient}, sorted(p.atA, p.atB, p.ownByA, p.ownByB, p.byPatient)},
				"patient B":  {map[string]string{"app.patient_id": h.patients[1].patient}, nil},
				"clinic A":   {clinic, sorted(p.atA, p.ownByA)},                                  // not B's confirmation, not the patient's
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

// provenanceInsert is one insert of TestEntryProvenance_AnInsertIsTiedToTheWriteItAttests; keep commits it, else it
// is rolled back so the next case finds the same state.
type provenanceInsert struct {
	scope                 map[string]string
	role, clinic, patient any
	name, entry           string
	activity, kind        string
	version               int
	ok, keep              bool
}

// errRollback rolls back a case that was accepted.
var errRollback = errors.New("rolled back")

// runProvenanceInserts runs each insert in its scope, in the entries' state, and checks it is accepted as it wants.
func runProvenanceInserts(t *testing.T, app *pgxpool.Pool, insert, state string, tests []provenanceInsert) {
	t.Helper()
	for _, tt := range tests {
		err := asApp(t, app, tt.scope, func(ctx context.Context, db postgres.Querier) error {
			if _, err := db.Exec(ctx, insert, tt.entry, tt.role, tt.clinic, tt.patient, tt.version, tt.activity, tt.kind); err != nil || tt.keep {
				return err
			}
			return errRollback
		})
		if accepted := err == nil || errors.Is(err, errRollback); accepted != tt.ok {
			t.Errorf("%s, %s: err %v, want ok %v", state, tt.name, err, tt.ok)
		}
	}
}

// A Provenance is tied to the write it attests: the entry at its current version and in the state the write left,
// a recording by the entry's recorder (who confirmed it), a confirmation of a confirmed entry by its confirmer, a
// change of the clinic's own entry not removed, a removal of it now entered-in-error, and the patient's addition to
// the profile or removal of a clinic's entry of theirs; one per version. Every other insert is refused, and nobody
// changes or deletes one.
func TestEntryProvenance_AnInsertIsTiedToTheWriteItAttests(t *testing.T) {
	t.Parallel()
	for entries, prov := range provenanceTables {
		t.Run(entries, func(t *testing.T) {
			t.Parallel()
			owner, app := pgtest.NewAppDatabase(t)
			h := seedHistory(t, owner)
			r := seedEntries(t, owner, h, entryTables[entries])
			p := seedProvenances(t, owner, h, r, prov[0], prov[1])
			var other string // a second role at clinic A, which neither recorded nor confirmed anything
			if err := owner.QueryRow(t.Context(), `WITH p AS (INSERT INTO practitioner (organization_id, given_name, family_name)
					VALUES ($1, 'Di', 'Example') RETURNING id)
				INSERT INTO practitioner_role (organization_id, practitioner_id) SELECT $1, id FROM p RETURNING id::text`, clinicA).Scan(&other); err != nil {
				t.Fatal(err)
			}
			clinic := map[string]string{"app.organization_id": clinicA}
			patientA := map[string]string{"app.patient_id": h.patients[0].patient}
			roleA, roleB, pa, pb := h.r.roles[clinicA], h.r.roles[clinicB], h.patients[0].patient, h.patients[1].patient
			run := func(state string, tests []provenanceInsert) {
				runProvenanceInserts(t, app, insertProvenance(prov[0], prov[1]), state, tests)
			}
			ownerExec(t, owner, `DELETE FROM `+prov[0]+` WHERE id = $1`, p.atA) // its recording is the first case
			run("atA recorded and confirmed by roleA (v1)", []provenanceInsert{
				{clinic, other, clinicA, nil, "a recording by another role", r.atA, "CREATE", "author", 1, false, false},
				{clinic, roleA, clinicA, nil, "a recording of the patient's entry", r.own, "CREATE", "author", 1, false, false},
				{clinic, roleB, clinicB, nil, "as another clinic", r.atA, "CREATE", "author", 1, false, false},
				{clinic, roleA, clinicA, nil, "on an entry it does not see", r.atB, "CREATE", "author", 1, false, false},
				{clinic, roleA, clinicA, nil, "at another version", r.atA, "UPDATE", "author", 2, false, false},
				{clinic, roleA, clinicA, nil, "the recording", r.atA, "CREATE", "author", 1, true, true},
				{clinic, roleA, clinicA, nil, "a second recording", r.atA, "CREATE", "author", 1, false, false},
			})
			ownerExec(t, owner, `UPDATE `+entries+` SET version_id = 2 WHERE id = ANY($1)`, []string{r.atA, r.own})
			run("v2, confirmed, not in the profile", []provenanceInsert{
				{clinic, roleA, clinicA, nil, "a change of its entry", r.atA, "UPDATE", "author", 2, true, false},
				{clinic, roleA, clinicA, nil, "the confirmer's confirmation", r.atA, "UPDATE", "verifier", 2, true, false},
				{clinic, other, clinicA, nil, "a confirmation by another role", r.atA, "UPDATE", "verifier", 2, false, false},
				{clinic, roleA, clinicA, nil, "a confirmation of an unconfirmed entry", r.own, "UPDATE", "verifier", 2, false, false},
				{clinic, roleA, clinicA, nil, "a change of the patient's entry", r.own, "UPDATE", "author", 2, false, false},
				{clinic, roleA, clinicA, nil, "a removal of an entry not removed", r.atA, "NULLIFY", "author", 2, false, false},
				{clinic, nil, nil, pa, "the patient as the agent in the clinic's scope", r.atA, "UPDATE", "author", 2, false, false},
				{patientA, nil, nil, pa, "the patient adding an entry not in the profile", r.atA, "UPDATE", "author", 2, false, false},
				{patientA, nil, nil, pa, "the patient removing an entry not removed", r.atA, "NULLIFY", "author", 2, false, false},
				{patientA, nil, nil, pa, "the patient on their own entry", r.own, "UPDATE", "author", 2, false, false},
				{patientA, roleA, clinicA, nil, "a clinic's agent in the patient's scope", r.atA, "UPDATE", "author", 2, false, false},
			})
			ownerExec(t, owner, `UPDATE `+entries+` SET in_profile = true WHERE id = $1`, r.atA)
			run("v2 in the profile", []provenanceInsert{
				{patientA, nil, nil, pb, "another patient as the agent", r.atA, "UPDATE", "author", 2, false, false},
				{patientA, nil, nil, pa, "the patient's addition to the profile", r.atA, "UPDATE", "author", 2, true, true},
				{clinic, roleA, clinicA, nil, "the clinic claiming the patient's version", r.atA, "UPDATE", "author", 2, false, false},
			})
			ownerExec(t, owner, `UPDATE `+entries+` SET version_id = 3, `+prov[2]+` WHERE id = $1`, r.atA)
			run("v3 removed", []provenanceInsert{
				{clinic, roleA, clinicA, nil, "a change of a removed entry", r.atA, "UPDATE", "author", 3, false, false},
				{clinic, roleA, clinicA, nil, "a confirmation of a removed entry", r.atA, "UPDATE", "verifier", 3, false, false},
				{patientA, nil, nil, pa, "the patient's removal", r.atA, "NULLIFY", "author", 3, true, false},
				{clinic, other, clinicA, nil, "its removal, by a colleague", r.atA, "NULLIFY", "author", 3, true, true},
				{patientA, nil, nil, pa, "the patient claiming the clinic's version", r.atA, "NULLIFY", "author", 3, false, false},
			})
			for name, stmt := range map[string]string{
				"update": `UPDATE ` + prov[0] + ` SET recorded = now() WHERE id = $1`,
				"delete": `DELETE FROM ` + prov[0] + ` WHERE id = $1`,
			} {
				for _, scope := range []map[string]string{clinic, patientA} {
					err := asApp(t, app, scope, func(ctx context.Context, db postgres.Querier) error {
						_, err := db.Exec(ctx, stmt, p.ownByA)
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

// A clinic's Provenance is never deleted with the entry: clinics confirmed the patient's own entry, the patient's
// change cleared the confirmation since, and deleting the entry fails (no action on the foreign key); the patient
// removes it as entered-in-error instead, and the Provenances stay.
func TestEntryProvenance_AnEntryWithAProvenanceIsNeverDeleted(t *testing.T) {
	t.Parallel()
	for entries, prov := range provenanceTables {
		t.Run(entries, func(t *testing.T) {
			t.Parallel()
			owner, app := pgtest.NewAppDatabase(t)
			h := seedHistory(t, owner)
			r := seedEntries(t, owner, h, entryTables[entries])
			p := seedProvenances(t, owner, h, r, prov[0], prov[1])
			patientA := map[string]string{"app.patient_id": h.patients[0].patient}
			err := asApp(t, app, patientA, func(ctx context.Context, db postgres.Querier) error {
				_, err := db.Exec(ctx, `DELETE FROM `+entries+` WHERE id = $1`, r.own)
				return err
			})
			if sqlState(err) != "23503" {
				t.Errorf("the patient deletes their entry with the clinics' Provenances: %v, want foreign_key_violation", err)
			}
			if err := asApp(t, app, patientA, func(ctx context.Context, db postgres.Querier) error {
				_, err := db.Exec(ctx, `UPDATE `+entries+` SET `+prov[2]+` WHERE id = $1`, r.own)
				return err
			}); err != nil {
				t.Fatalf("the patient removes it as entered-in-error: %v", err)
			}
			var n int
			if err := owner.QueryRow(t.Context(), `SELECT count(*) FROM `+prov[0]+` WHERE id = ANY($1)`, []string{p.ownByA, p.ownByB}).Scan(&n); err != nil {
				t.Fatal(err)
			}
			if n != 2 {
				t.Errorf("%d of the entry's 2 Provenances left, want both", n)
			}
		})
	}
}

// A Provenance's codes are from their code systems, a creation is of the first version and nothing else is, the
// agent is one practitioner role of the clinic named or the patient, the patient never creates or verifies, a
// creation is stored as its author, and a version has one Provenance.
func TestEntryProvenance_HoldsAConsistentState(t *testing.T) {
	t.Parallel()
	for entries, prov := range provenanceTables {
		t.Run(entries, func(t *testing.T) {
			t.Parallel()
			owner, _ := pgtest.NewAppDatabase(t)
			h := seedHistory(t, owner)
			r := seedEntries(t, owner, h, entryTables[entries])
			roleA, pa := h.r.roles[clinicA], h.patients[0].patient
			ownerExec(t, owner, `INSERT INTO `+prov[0]+` (recorded, `+prov[1]+`, practitioner_role_id, organization_id, target_version, activity, agent_type)
				VALUES (now(), $1, $2, $3, 1, 'CREATE', 'author')`, r.atA, roleA, clinicA)
			for _, tt := range []struct {
				role, clinic, patient any
				name                  string
				activity, kind        string
				version               int
			}{
				{roleA, clinicA, nil, "an unknown activity", "DELETE", "author", 2},
				{roleA, clinicA, nil, "an unknown agent type", "UPDATE", "enterer", 2},
				{roleA, clinicA, nil, "a creation of a later version", "CREATE", "author", 2},
				{roleA, clinicA, nil, "an update of the first version", "UPDATE", "author", 1},
				{roleA, clinicB, nil, "another clinic's role", "CREATE", "author", 1},
				{roleA, nil, nil, "a role without its clinic", "UPDATE", "author", 2},
				{nil, nil, nil, "no agent", "UPDATE", "author", 2},
				{roleA, clinicA, pa, "a role and the patient", "UPDATE", "author", 2},
				{nil, nil, pa, "the patient creating", "CREATE", "author", 1},
				{nil, nil, pa, "the patient verifying", "UPDATE", "verifier", 2},
				{roleA, clinicA, nil, "a creation stored as its verifier", "CREATE", "verifier", 1},
				{roleA, clinicA, nil, "a second Provenance of the recorded version", "CREATE", "author", 1},
			} {
				if _, err := owner.Exec(t.Context(), insertProvenance(prov[0], prov[1]), r.atA, tt.role, tt.clinic, tt.patient,
					tt.version, tt.activity, tt.kind); err == nil {
					t.Errorf("%s: accepted", tt.name)
				}
			}
		})
	}
}
