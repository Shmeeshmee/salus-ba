package api_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/questionnaire"
	"github.com/Shmeeshmee/salus/internal/questionnaireresponse"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// heldQuestionnaires is questionnaire.Service whose Read, the step of a request after it locked the
// appointment, signals reached and waits for release (or the test's end) before it reads.
type heldQuestionnaires struct {
	*questionnaire.Service
	reached, release chan struct{}
}

func (q heldQuestionnaires) Read(ctx context.Context, c questionnaire.Caller, id string) (questionnaire.Questionnaire, error) {
	close(q.reached)
	select {
	case <-q.release:
	case <-ctx.Done():
	}
	return q.Service.Read(ctx, c, id)
}

// A request and a cancellation of one appointment at once, on the salus_app login (row-level security on):
// the cancellation waits while the request's transaction holds the appointment's lock, and commits after it;
// a request after the committed cancellation is refused with ErrAppointmentClosed, and audited.
func TestQuestionnaireRequest_AndACancellationOfTheAppointmentDoNotInterleave(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	c := seedClinic(t, owner, 1, 48*time.Hour)
	b := newBooking(t, app)
	pract := appointment.Caller{AccountID: c.practitionerAccount, OrganizationID: inviteClinic, PractitionerID: c.practitioner}
	booked, err := b.appointments.Book(t.Context(), appointment.Caller{AccountID: c.accounts[0], PatientID: c.patients[0]}, c.slots[0])
	if err != nil {
		t.Fatalf("Book() = %v", err)
	}
	first, second := activeQuestionnaire(t, owner), activeQuestionnaire(t, owner)
	held := heldQuestionnaires{reached: make(chan struct{}), release: make(chan struct{})}
	responses := newResponses(app, b.appointments, held)

	requested := make(chan error, 1)
	go func() {
		_, err := responses.Request(t.Context(), pract, booked.ID, first)
		requested <- err
	}()
	<-held.reached
	cancelled := make(chan error, 1)
	go func() {
		_, err := b.appointments.Cancel(t.Context(), pract, booked.ID)
		cancelled <- err
	}()
	pgtest.WaitForLockWait(t, owner)
	select {
	case err := <-cancelled:
		t.Fatalf("Cancel() returned %v while the request held the appointment, want it to wait", err)
	default:
	}
	close(held.release)
	if err := <-requested; err != nil {
		t.Fatalf("the request that held the lock = %v, want it made", err)
	}
	if err := <-cancelled; err != nil {
		t.Fatalf("Cancel() after the request = %v, want it made", err)
	}

	held = heldQuestionnaires{reached: make(chan struct{}), release: make(chan struct{})}
	close(held.release)
	responses = newResponses(app, b.appointments, held)
	if _, err := responses.Request(t.Context(), pract, booked.ID, second); !errors.Is(err, questionnaireresponse.ErrAppointmentClosed) {
		t.Errorf("a request after the cancellation = %v, want ErrAppointmentClosed", err)
	}
	var refusals int
	if err := owner.QueryRow(t.Context(), `SELECT count(*) FROM audit_event WHERE subtype_code = 'create' AND outcome = 4
		AND entity_reference IS NULL AND agent_account_id = $1`, c.practitionerAccount).Scan(&refusals); err != nil || refusals != 1 {
		t.Errorf("audited %d refused requests (%v), want the one", refusals, err)
	}
}

// newResponses is the questionnaireresponse service as internal/api composes it, on app, with questionnaires
// as its questionnaire service.
func newResponses(app *pgxpool.Pool, appointments *appointment.Service, questionnaires heldQuestionnaires) *questionnaireresponse.Service {
	tx := postgres.NewTransactor(app)
	audit := auditevent.NewService(auditevent.NewPostgresRepository(app), tx, clock.System{})
	questionnaires.Service = questionnaire.NewService(questionnaire.Ports{
		Repo: questionnaire.NewPostgresRepository(app), Audit: audit, Tx: tx, Clock: clock.System{},
	})
	return questionnaireresponse.NewService(questionnaireresponse.Ports{
		Repo: questionnaireresponse.NewPostgresRepository(app), Audit: audit, Tx: tx, Clock: clock.System{},
		Appointments: appointments, Questionnaires: questionnaires,
	})
}

// activeQuestionnaire stores, as the owner, an active version of a new series of the clinic, and returns its id.
func activeQuestionnaire(t *testing.T, owner *pgxpool.Pool) string {
	t.Helper()
	var id string
	if err := owner.QueryRow(t.Context(), `INSERT INTO questionnaire (series_id, organization_id, version, status, title, published_at)
		VALUES (gen_random_uuid(), $1, 1, 'active', 'Placeholder questionnaire', now()) RETURNING id::text`, inviteClinic).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}
