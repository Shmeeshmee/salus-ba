package questionnaireresponse_test

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/binary"
	"github.com/Shmeeshmee/salus/internal/questionnaireresponse"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// file is a synthetic attachment of the item "files".
func file(id string, pending bool) binary.Attachment {
	return binary.Attachment{
		LinkID: "files", BinaryID: id, ContentType: "application/pdf", Title: "Placeholder document.pdf",
		Hash: []byte("0123456789abcdefghij"), Size: 1234, Pending: pending,
	}
}

const (
	fileA = "b0000000-0000-4000-8000-0000000000a1"
	fileB = "b0000000-0000-4000-8000-0000000000b2"
	fileC = "b0000000-0000-4000-8000-0000000000c3"
)

// The holder's list of files, in the patient's scope (an upload, a deletion) and in binary's job scope (the
// scan and the discard), on the real database: added in order, cleared of pending, removed; neither changes
// version_id; another patient and another response's job scope see no response.
func TestPostgres_AttachmentsInThePatientsAndTheJobScope(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	s := seedDB(t, owner, dbStart)
	at := dbStart.Add(-48 * time.Hour)
	var resp questionnaireresponse.Response
	err := inScope(t, app, s.clinic, "", func(ctx context.Context, r questionnaireresponse.Repository) error {
		var err error
		resp, _, err = r.Insert(ctx, s.request(0, 0), at)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	later := at.Add(time.Minute)
	err = inScope(t, app, "", s.patient, func(ctx context.Context, r questionnaireresponse.Repository) error {
		for _, id := range []string{fileA, fileB, fileC} {
			if err := r.AddAttachment(ctx, resp.ID, file(id, true), later); err != nil {
				return err
			}
		}
		if err := r.RemoveAttachment(ctx, resp.ID, fileB, later); err != nil {
			return err
		}
		return r.RemoveAttachment(ctx, resp.ID, "b0000000-0000-4000-8000-0000000000ff", later) // none: nothing changes
	})
	if err != nil {
		t.Fatalf("in the patient's scope: %v", err)
	}
	job := func(response, org string, fn func(ctx context.Context, r questionnaireresponse.Repository) error) error {
		r := questionnaireresponse.NewPostgresRepository(app)
		return postgres.NewTransactor(app).WithTx(t.Context(), func(ctx context.Context) error {
			if err := postgres.SetLocal(ctx, "app.questionnaire_response_id", response); err != nil {
				return err
			}
			if err := postgres.SetLocal(ctx, "app.organization_id", org); err != nil {
				return err
			}
			return fn(ctx, r)
		})
	}
	err = job(resp.ID, s.clinic, func(ctx context.Context, r questionnaireresponse.Repository) error {
		if err := r.ClearAttachment(ctx, resp.ID, fileC, later.Add(time.Minute)); err != nil {
			return err
		}
		got, err := r.Attachments(ctx, resp.ID)
		if want := []binary.Attachment{file(fileA, true), file(fileC, false)}; err != nil || !reflect.DeepEqual(got, want) {
			t.Errorf("Attachments() = %+v, %v;\nwant %+v", got, err, want)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("in the job scope: %v", err)
	}
	var version int32
	var updated time.Time
	if err := owner.QueryRow(t.Context(), `SELECT version_id, updated_at FROM questionnaire_response WHERE id = $1`, resp.ID).Scan(&version, &updated); err != nil {
		t.Fatal(err)
	}
	if version != resp.VersionID || !updated.Equal(later.Add(time.Minute)) {
		t.Errorf("version %d, updated %v; want version %d kept and updated at the clear", version, updated, resp.VersionID)
	}

	// The job scope named with another clinic: no response to change or read. (Named with the response's
	// clinic, the job scope also has the clinic's own view, 0.1; binary's jobs name their response.)
	err = job(resp.ID, clinicB, func(ctx context.Context, r questionnaireresponse.Repository) error {
		if _, err := r.Attachments(ctx, resp.ID); !errors.Is(err, questionnaireresponse.ErrNotFound) {
			t.Errorf("Attachments() from another clinic's job scope = %v, want ErrNotFound", err)
		}
		if err := r.AddAttachment(ctx, resp.ID, file(fileB, true), later); !errors.Is(err, questionnaireresponse.ErrNotFound) {
			t.Errorf("AddAttachment() from another clinic's job scope = %v, want ErrNotFound", err)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r(app).Attachments(t.Context(), "not-a-uuid"); !errors.Is(err, questionnaireresponse.ErrNotFound) {
		t.Errorf("Attachments(not a UUID) = %v, want ErrNotFound", err)
	}
}

func r(app postgres.Querier) questionnaireresponse.Repository {
	return questionnaireresponse.NewPostgresRepository(app)
}

// withFile stores, as the owner, a binary row for the response id and lists it in the response's attachments,
// as binary's upload does. Synthetic.
func withFile(t *testing.T, owner postgres.Querier, id, binaryID string, pending bool) {
	t.Helper()
	_, err := owner.Exec(t.Context(), `INSERT INTO "binary" (id, organization_id, patient_id, appointment_id, questionnaire_response_id,
			size, segment_size, link_id, content_type, title, kek_id, sha1, wrapped_key)
		SELECT $2, organization_id, patient_id, appointment_id, id, 1234, 65536, 'files', 'application/pdf', 'Placeholder document.pdf',
			'synthetic', '0123456789abcdefghij'::bytea, '\x00'::bytea
		FROM questionnaire_response WHERE id = $1`, id, binaryID)
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(file(binaryID, pending))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := owner.Exec(t.Context(), `UPDATE questionnaire_response SET attachments = attachments || jsonb_build_array($2::jsonb) WHERE id = $1`, id, b); err != nil {
		t.Fatal(err)
	}
}

// files counts the binary rows of the response id.
func files(t *testing.T, owner postgres.Querier, id string) int {
	t.Helper()
	var n int
	if err := owner.QueryRow(t.Context(), `SELECT count(*) FROM "binary" WHERE questionnaire_response_id = $1`, id).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// A response is read with its files; a withdrawal clears them with the answers (docs/attachments/SPEC.md §2
// "Discard": the binary_discard job then deletes the rows), which the withdrawn-attachments check requires.
func TestPostgres_ReadsTheAttachmentsAndStopClearsThem(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	s := seedDB(t, owner, dbStart)
	id := insert(t, app, s, 0, 0)
	withFile(t, owner, id, fileA, false)
	withFile(t, owner, id, fileB, true)

	err := inScope(t, app, "", s.patient, func(ctx context.Context, r questionnaireresponse.Repository) error {
		got, err := r.ByID(ctx, id)
		if want := []binary.Attachment{file(fileA, false), file(fileB, true)}; err != nil || !reflect.DeepEqual(got.Attachments, want) {
			t.Errorf("ByID() attachments = %+v, %v;\nwant %+v", got.Attachments, err, want)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	err = inScope(t, app, s.clinic, "", func(ctx context.Context, r questionnaireresponse.Repository) error {
		got, err := r.Stop(ctx, id, dbStart.Add(-time.Hour))
		if err != nil {
			return err
		}
		if len(got.Attachments) != 0 {
			t.Errorf("Stop() attachments = %+v, want none", got.Attachments)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Stop() of a response with files: %v", err)
	}
	var stored string
	if err := owner.QueryRow(t.Context(), `SELECT attachments::text FROM questionnaire_response WHERE id = $1`, id).Scan(&stored); err != nil || stored != "[]" {
		t.Errorf("stored attachments %s (%v), want []", stored, err)
	}
	if n := files(t, owner, id); n != 2 {
		t.Errorf("%d binary rows after the withdrawal, want both until the discard job", n)
	}
}

// Deleting a response by the cancellation's discard or by the retention purger deletes its files' rows in the
// same statement, through the foreign key (docs/attachments/SPEC.md §2 "Tables").
func TestPostgres_DeletingAResponseDeletesItsFilesRows(t *testing.T) {
	t.Parallel()
	t.Run("the cancellation's discard", func(t *testing.T) {
		t.Parallel()
		owner, app := pgtest.NewAppDatabase(t)
		s := seedDB(t, owner, dbStart)
		id, kept := insert(t, app, s, 0, 0), insert(t, app, s, 1, 0)
		withFile(t, owner, id, fileA, false)
		withFile(t, owner, kept, fileB, false)
		if _, err := owner.Exec(t.Context(), `UPDATE appointment SET status = 'cancelled', cancelled_at = now(), cancellation_reason = 'pat' WHERE id = $1`, s.appts[0]); err != nil {
			t.Fatal(err)
		}
		err := inScope(t, app, s.clinic, "", func(ctx context.Context, r questionnaireresponse.Repository) error {
			if err := r.SetNotice(ctx, s.appts[0]); err != nil {
				return err
			}
			ids, err := r.DeleteOfAppointment(ctx, s.appts[0])
			if len(ids) != 1 {
				t.Errorf("deleted %v, want the response", ids)
			}
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
		if n, m := files(t, owner, id), files(t, owner, kept); n != 0 || m != 1 {
			t.Errorf("%d rows of the deleted response and %d of the other, want 0 and 1", n, m)
		}
	})
	t.Run("the retention purger", func(t *testing.T) {
		t.Parallel()
		owner, ret := pgtest.NewRetentionDatabase(t)
		s := seedDB(t, owner, dbStart)
		var id string
		if err := owner.QueryRow(t.Context(), `INSERT INTO questionnaire_response (organization_id, appointment_id, patient_id,
				practitioner_role_id, questionnaire_id, series_id, source, status, due_at)
			SELECT a.organization_id, a.id, a.patient_id, a.practitioner_role_id, q.id, q.series_id, 'added', 'in-progress', lower(a.period)
			FROM appointment a, questionnaire q WHERE a.id = $1 AND q.id = $2 RETURNING id::text`, s.appts[0], s.versions[0]).Scan(&id); err != nil {
			t.Fatal(err)
		}
		withFile(t, owner, id, fileA, true)
		n, err := questionnaireresponse.NewPurger(questionnaireresponse.NewPostgresRepository(ret)).Purge(t.Context(), dbStart.Add(time.Second))
		if err != nil || n != 1 {
			t.Fatalf("Purge() = %d, %v; want the response", n, err)
		}
		if n := files(t, owner, id); n != 0 {
			t.Errorf("%d binary rows of the purged response, want 0", n)
		}
	})
}
