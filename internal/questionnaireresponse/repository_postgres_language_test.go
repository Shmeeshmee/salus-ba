package questionnaireresponse_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/questionnaireresponse"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// docs/i18n/SPEC.md §3 "The patient's language": the language answered in is set when the patient starts and
// cannot change after the first saved answer.
func TestPostgres_SetLanguageWhileNothingIsAnswered(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	s := seedDB(t, owner, dbStart)
	other := seedDB(t, owner, dbStart)
	id := insert(t, app, s, 0, 0)
	no := false
	saved := []questionnaireresponse.Answer{{LinkID: "q1", Values: []questionnaireresponse.Value{{Boolean: &no}}}}
	at := dbStart.Add(-time.Hour)

	var fresh, english, german questionnaireresponse.Response
	err := inScope(t, app, "", other.patient, func(ctx context.Context, r questionnaireresponse.Repository) error {
		if _, err := r.SetLanguage(ctx, id, "en", at); !errors.Is(err, questionnaireresponse.ErrNotFound) {
			return errors.Join(err, errors.New("another patient set the language"))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	err = inScope(t, app, "", s.patient, func(ctx context.Context, r questionnaireresponse.Repository) error {
		var err error
		if fresh, err = r.ByID(ctx, id); err != nil {
			return err
		}
		if english, err = r.SetLanguage(ctx, id, "en", at); err != nil {
			return err
		}
		if german, err = r.SetLanguage(ctx, id, "de", at.Add(time.Minute)); err != nil {
			return err
		}
		if _, err = r.SaveAnswers(ctx, id, saved, at.Add(2*time.Minute)); err != nil {
			return err
		}
		if _, err := r.SetLanguage(ctx, id, "en", at); !errors.Is(err, questionnaireresponse.ErrNotFound) {
			return errors.Join(err, errors.New("changed the language after an answer was saved"))
		}
		if _, err := r.SetLanguage(ctx, "not-a-uuid", "en", at); !errors.Is(err, questionnaireresponse.ErrNotFound) {
			return errors.Join(err, errors.New("a malformed id"))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if fresh.Language != "" {
		t.Errorf("a new response's language: %q, want none", fresh.Language)
	}
	if english.Language != "en" || english.VersionID != 2 || !english.LastUpdated.Equal(at) {
		t.Errorf("SetLanguage(en) = %+v; want English at version 2", english)
	}
	if german.Language != "de" || german.VersionID != 3 {
		t.Errorf("SetLanguage(de) before any answer = %+v; want German at version 3", german)
	}
}
