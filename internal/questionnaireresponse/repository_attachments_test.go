package questionnaireresponse_test

import (
	"context"
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
