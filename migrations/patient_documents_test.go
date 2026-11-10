package migrations_test

import (
	"context"
	"slices"
	"testing"

	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// 00038 (SAL-17): the patient's own scope sees the final (signed) documents of their own encounters, whatever the
// consent, never an open or discarded draft and never another patient's; a video room's scope sees none.
func TestPatientDocuments_ThePatientSeesOnlyTheirOwnSignedDocuments(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	c := seedConsult(t, owner)
	seedVideoSessions(t, owner, c.a)
	e, a := c.encounters, c.a
	// Sign the seeded (open) draft of patient A's appointment in clinic B, which started before the revocation, and
	// patient B's; discard patient A's in clinic A.
	if _, err := owner.Exec(t.Context(), `UPDATE document_draft SET status = 'final' WHERE encounter_id = ANY($1)`,
		[]string{e[a.bPast], e[a.other]}); err != nil {
		t.Fatal(err)
	}
	if _, err := owner.Exec(t.Context(), `UPDATE document_draft SET status = 'entered-in-error' WHERE encounter_id = $1`, e[a.aPast]); err != nil {
		t.Fatal(err)
	}
	for name, tt := range map[string]struct {
		scope map[string]string
		want  []string
	}{
		"patient A":            {map[string]string{"app.patient_id": c.patients[0].patient}, sorted(e[a.bPast])},
		"patient B":            {map[string]string{"app.patient_id": c.patients[1].patient}, sorted(e[a.other])},
		"patient A in a room":  {map[string]string{"app.patient_id": c.patients[0].patient, "app.video_room": "r_" + a.bFuture}, nil},
		"an account":           {map[string]string{"app.account_id": c.patients[0].account}, nil},
		"clinic B, as before":  {map[string]string{"app.organization_id": clinicB}, sorted(e[a.bPast])},
		"clinic A, its drafts": {map[string]string{"app.organization_id": clinicA}, sorted(e[a.aPast], e[a.aFuture])},
	} {
		if got := idsIn(t, app, tt.scope, "SELECT encounter_id::text FROM document_draft"); !slices.Equal(got, tt.want) {
			t.Errorf("%s: sees the documents of %v, want %v", name, got, tt.want)
		}
	}
	// Read-only: the patient changes nothing.
	err := asApp(t, app, map[string]string{"app.patient_id": c.patients[0].patient}, func(ctx context.Context, db postgres.Querier) error {
		tag, err := db.Exec(ctx, `UPDATE document_draft SET status = 'entered-in-error' WHERE encounter_id = $1`, e[a.bPast])
		if err == nil && tag.RowsAffected() != 0 {
			t.Errorf("the patient changed %d documents, want none", tag.RowsAffected())
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// 00038: the clinic's patient transcript copy is off unless its admin turns it on, through salus_app's grant.
func TestPatientDocuments_TheTranscriptCopyIsOffUntilTheClinicTurnsItOn(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seedDirectory(t, owner)
	var on bool
	if err := owner.QueryRow(t.Context(), `SELECT patient_transcript_copy FROM organization WHERE id = $1`, clinicA).Scan(&on); err != nil || on {
		t.Fatalf("patient_transcript_copy = %v, %v; want false", on, err)
	}
	err := asApp(t, app, map[string]string{"app.organization_id": clinicA}, func(ctx context.Context, db postgres.Querier) error {
		_, err := db.Exec(ctx, `UPDATE organization SET patient_transcript_copy = true WHERE id = $1`, clinicA)
		return err
	})
	if err != nil {
		t.Errorf("the clinic's update = %v, want it granted", err)
	}
}
