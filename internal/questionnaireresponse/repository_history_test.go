package questionnaireresponse_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/questionnaireresponse"
	"github.com/Shmeeshmee/salus/pkg/keyset"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// SAL-39: CompletedOf and FilesOf page the patient's completed responses, and the files of them no longer pending,
// newest first by submission, after the position and in the range and clinic asked; never one in progress, nor
// another patient's.
func TestPostgres_CompletedOfAndFilesOfPageThePatientsTimeline(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	s := seedDB(t, owner, dbStart)
	other := seedDB(t, owner, dbStart)
	done, open := insert(t, app, s, 0, 0), insert(t, app, s, 1, 1)
	authored := dbStart.Add(-time.Hour)
	err := inScope(t, app, "", s.patient, func(ctx context.Context, r questionnaireresponse.Repository) error {
		for _, f := range [][2]any{{fileA, false}, {fileB, true}, {fileC, false}} {
			if err := r.AddAttachment(ctx, done, file(f[0].(string), f[1].(bool)), authored); err != nil {
				return err
			}
		}
		if err := r.AddAttachment(ctx, open, file("b0000000-0000-4000-8000-0000000000d4", false), authored); err != nil {
			return err
		}
		_, err := r.Complete(ctx, done, nil, authored)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	for name, tc := range map[string]struct {
		org     string
		patient string
		firstF  string
		p       keyset.Page
		wantR   int
		wantF   int
	}{
		"everything":            {p: keyset.Page{Limit: 10}, wantR: 1, wantF: 2, firstF: fileC},
		"the limit":             {p: keyset.Page{Limit: 1}, wantR: 1, wantF: 1, firstF: fileC},
		"after the first file":  {p: keyset.Page{At: authored, Ref: "Binary/" + fileC, Limit: 10}, wantF: 1, firstF: fileA},
		"after the response":    {p: keyset.Page{At: authored, Ref: "QuestionnaireResponse/" + done, Limit: 10}, wantF: 2, firstF: fileC},
		"from after":            {p: keyset.Page{From: authored.Add(time.Second), Limit: 10}},
		"to it":                 {p: keyset.Page{To: authored, Limit: 10}},
		"its clinic":            {p: keyset.Page{Limit: 10}, org: s.clinic, wantR: 1, wantF: 2, firstF: fileC},
		"another clinic":        {p: keyset.Page{Limit: 10}, org: other.clinic},
		"a clinic not a UUID":   {p: keyset.Page{Limit: 10}, org: "x"},
		"another patient scope": {p: keyset.Page{Limit: 10}, patient: other.patient},
	} {
		patient := s.patient
		if tc.patient != "" {
			patient = tc.patient
		}
		rs, fs := timelineOf(t, app, s.patient, patient, tc.org, tc.p)
		if !slices.Equal(rs, []string{done}[:tc.wantR]) || len(fs) != tc.wantF || (tc.wantF > 0 && fs[0] != tc.firstF) {
			t.Errorf("%s: responses %v, files %v; want %d and %d starting with %s", name, rs, fs, tc.wantR, tc.wantF, tc.firstF)
		}
	}
}

// timelineOf lists the responses and the files of the patient's timeline in the scope of scopePatient, and checks
// each file comes with its completed response and no answers.
func timelineOf(t *testing.T, app *pgxpool.Pool, patientID, scopePatient, org string, p keyset.Page) (responses, files []string) {
	t.Helper()
	err := inScope(t, app, "", scopePatient, func(ctx context.Context, r questionnaireresponse.Repository) error {
		rs, err := r.CompletedOf(ctx, patientID, org, p)
		if err != nil {
			return err
		}
		for _, x := range rs {
			if x.Answers != nil || x.Attachments != nil || x.Status != questionnaireresponse.StatusCompleted {
				t.Errorf("CompletedOf() = %+v, want it completed, without answers or files", x)
			}
			responses = append(responses, x.ID)
		}
		fs, err := r.FilesOf(ctx, patientID, org, p)
		for _, a := range fs {
			if a.Response.Status != questionnaireresponse.StatusCompleted || a.Response.Answers != nil || a.File.Title == "" {
				t.Errorf("FilesOf() = %+v, want each with its completed response", a)
			}
			files = append(files, a.File.BinaryID)
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return responses, files
}

// A file listed twice by its response is one file of the timeline: two rows with the same position would tie, and
// the page after the first would drop the second (the review of SAL-39, finding 9).
func TestPostgres_FilesOfListsAFileOnce(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	s := seedDB(t, owner, dbStart)
	done := insert(t, app, s, 0, 0)
	authored := dbStart.Add(-time.Hour)
	err := inScope(t, app, "", s.patient, func(ctx context.Context, r questionnaireresponse.Repository) error {
		for range 2 {
			if err := r.AddAttachment(ctx, done, file(fileA, false), authored); err != nil {
				return err
			}
		}
		_, err := r.Complete(ctx, done, nil, authored)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, files := timelineOf(t, app, s.patient, s.patient, "", keyset.Page{Limit: 10}); !slices.Equal(files, []string{fileA}) {
		t.Errorf("FilesOf() = %v, want %s once", files, fileA)
	}
}
