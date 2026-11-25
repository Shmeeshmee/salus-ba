package binary_test

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/binary"
	"github.com/Shmeeshmee/salus/pkg/envelope"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// seededReport is seedDB's patient with a preliminary lab result on their visit, written by a practitioner's account
// of the clinic, and a second report of the other patient. Synthetic.
type seededReport struct {
	seededDB
	report, otherReport, doctor string
}

func seedReportDB(t testing.TB, owner *pgxpool.Pool) seededReport {
	t.Helper()
	ctx := t.Context()
	s := seededReport{seededDB: seedDB(t, owner)}
	if err := owner.QueryRow(ctx, `INSERT INTO account (email, password_hash, role, organization_id)
		VALUES (gen_random_uuid()::text || '@example.test', 'h', 'practitioner', $1) RETURNING id::text`, s.clinic).Scan(&s.doctor); err != nil {
		t.Fatal(err)
	}
	report := func(patient, response string) string {
		var id string
		if err := owner.QueryRow(ctx, `INSERT INTO diagnostic_report (organization_id, patient_id, appointment_id, created_by_account_id, title)
			SELECT organization_id, patient_id, appointment_id, $2, 'Placeholder blood count' FROM questionnaire_response WHERE id = $1
			RETURNING id::text`, response, s.doctor).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	s.report, s.otherReport = report(s.patient, s.response), report(s.other, s.otherResponse)
	return s
}

// reportFile is the report's PDF as UploadReport builds it.
func (s seededReport) reportFile(report, patient string) binary.Binary {
	return binary.Binary{
		ID: binaryID(), OrganizationID: s.clinic, PatientID: patient, DiagnosticReportID: report, ContentType: "application/pdf",
		Title: "placeholder.pdf", KEKID: "test-1", SHA1: bytes.Repeat([]byte{4}, 20), WrappedKey: bytes.Repeat([]byte{1}, 60),
		Size: 700, SegmentSize: envelope.SegmentSize,
	}
}

func asReportJob(report, org string) func(context.Context, binary.Repository) error {
	return func(ctx context.Context, r binary.Repository) error { return r.SetReportJob(ctx, report, org) }
}

func (s seededReport) release(t *testing.T, owner *pgxpool.Pool) {
	t.Helper()
	if _, err := owner.Exec(t.Context(), `UPDATE diagnostic_report SET status = 'final', released_at = now(), released_by_account_id = $2,
		version_id = version_id + 1 WHERE id = $1`, s.report, s.doctor); err != nil {
		t.Fatal(err)
	}
}

// A report's PDF is stored in the clinic's scope while the report is preliminary, and seen there and by the scan job
// of the report; the patient sees it only once the report is released, and then it is frozen: no file is added or
// deleted, by the clinic or by the owner.
func TestPostgres_AReportsPDFIsTheClinicsUntilReleasedThenThePatientsAndFrozen(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	s := seedReportDB(t, owner)
	at := time.Date(2030, 1, 6, 9, 0, 0, 0, time.UTC)
	var stored binary.Binary
	if err := in(t, app, asClinic(s.clinic), func(ctx context.Context, r binary.Repository) error {
		var err error
		stored, err = r.Insert(ctx, s.reportFile(s.report, s.patient), at)
		return err
	}); err != nil {
		t.Fatalf("Insert() = %v", err)
	}
	if stored.DiagnosticReportID != s.report || stored.ResponseID != "" || stored.AppointmentID != "" || stored.ConversationID != "" ||
		stored.Status != binary.StatusQuarantined {
		t.Errorf("Insert() = %+v, want the report's quarantined file", stored)
	}
	if err := in(t, app, asClinic(s.clinic), func(ctx context.Context, r binary.Repository) error {
		_, err := r.Insert(ctx, s.reportFile(s.report, s.other), at)
		return err
	}); err == nil {
		t.Error("a file of the report naming another patient was stored")
	}
	sees := func(scope func(context.Context, binary.Repository) error) bool {
		t.Helper()
		var got []binary.Binary
		if err := in(t, app, scope, func(ctx context.Context, r binary.Repository) error {
			var err error
			got, err = r.OfReports(ctx, []string{s.report, s.otherReport})
			return err
		}); err != nil {
			t.Fatal(err)
		}
		return len(got) == 1 && got[0].ID == stored.ID
	}
	for name, sc := range map[string]struct {
		scope func(context.Context, binary.Repository) error
		sees  bool
	}{
		"the clinic":                   {asClinic(s.clinic), true},
		"another clinic":               {asClinic(unknownID), false},
		"the patient, not yet":         {asPatient(s.patient), false},
		"the scan job of the report":   {asReportJob(s.report, s.clinic), true},
		"the scan job of another":      {asReportJob(s.otherReport, s.clinic), false},
		"a questionnaire's job":        {asJob(s.response, s.clinic), false},
		"the clinic in a conversation": {inConversation(asClinic(s.clinic), unknownID), true},
	} {
		if got := sees(sc.scope); got != sc.sees {
			t.Errorf("%s sees the file: %v, want %v", name, got, sc.sees)
		}
	}
	if err := in(t, app, asReportJob(s.report, s.clinic), func(ctx context.Context, r binary.Repository) error {
		ok, err := r.MarkScanned(ctx, stored.ID, binary.StatusClean, at)
		if err == nil && !ok {
			t.Error("the scan job did not mark the report's file")
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}

	s.release(t, owner)
	if !sees(asPatient(s.patient)) || sees(asPatient(s.other)) {
		t.Error("once released, the patient must see the file, and only they")
	}
	checkFrozen(t, owner, app, s, stored.ID)
}

// checkFrozen: the file id of the released report is neither deleted nor joined by another, by the clinic or the
// owner; a rejected one is purged.
func checkFrozen(t *testing.T, owner, app *pgxpool.Pool, s seededReport, id string) {
	t.Helper()
	frozenAt := time.Date(2030, 1, 7, 9, 0, 0, 0, time.UTC)
	if err := in(t, app, asClinic(s.clinic), func(ctx context.Context, r binary.Repository) error {
		ok, err := r.Delete(ctx, id)
		if ok {
			t.Error("the clinic deleted the file of a released report")
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if err := in(t, app, asClinic(s.clinic), func(ctx context.Context, r binary.Repository) error {
		_, err := r.Insert(ctx, s.reportFile(s.report, s.patient), frozenAt)
		return err
	}); err == nil {
		t.Error("a file was added to a released report")
	}
	if _, err := owner.Exec(t.Context(), `DELETE FROM "binary" WHERE id = $1`, id); err == nil {
		t.Error("the owner deleted the file of a released report")
	}
	if _, err := owner.Exec(t.Context(), `UPDATE "binary" SET status = 'rejected', wrapped_key = NULL WHERE id = $1`, id); err != nil {
		t.Fatal(err)
	}
	if _, err := owner.Exec(t.Context(), `DELETE FROM "binary" WHERE id = $1`, id); err != nil {
		t.Errorf("a rejected file of a released report is purged: %v", err)
	}
}

// The clinic deletes the file of its preliminary report (a replacement), in its own scope only.
func TestPostgres_TheClinicDeletesThePDFOfItsPreliminaryReport(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	s := seedReportDB(t, owner)
	b := s.reportFile(s.report, s.patient)
	at := time.Date(2030, 1, 6, 9, 0, 0, 0, time.UTC)
	if err := in(t, app, asClinic(s.clinic), func(ctx context.Context, r binary.Repository) error {
		_, err := r.Insert(ctx, b, at)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	for name, sc := range map[string]struct {
		scope   func(context.Context, binary.Repository) error
		deletes bool
	}{
		"another clinic": {asClinic(unknownID), false},
		"the patient":    {asPatient(s.patient), false},
		"the clinic":     {asClinic(s.clinic), true},
	} {
		if err := in(t, app, sc.scope, func(ctx context.Context, r binary.Repository) error {
			ok, err := r.Delete(ctx, b.ID)
			if ok != sc.deletes {
				t.Errorf("%s deleted the file: %v, want %v", name, ok, sc.deletes)
			}
			return err
		}); err != nil {
			t.Fatal(err)
		}
	}
	var n int
	if err := postgres.NewTransactor(owner).WithTx(t.Context(), func(ctx context.Context) error {
		return owner.QueryRow(ctx, `SELECT count(*) FROM "binary" WHERE diagnostic_report_id = $1`, s.report).Scan(&n)
	}); err != nil || n != 0 {
		t.Errorf("%d rows left (%v)", n, err)
	}
}
