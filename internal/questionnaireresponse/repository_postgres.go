package questionnaireresponse

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Shmeeshmee/salus/internal/questionnaireresponse/questionnaireresponsedb"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

type postgresRepository struct {
	db postgres.Querier
}

// NewPostgresRepository returns the Repository on db (the salus_app pool, or salus_retention's for the
// purger), using the transaction in the context when there is one; row-level security shows only the scope
// set.
func NewPostgresRepository(db postgres.Querier) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) q(ctx context.Context) *questionnaireresponsedb.Queries {
	return questionnaireresponsedb.New(postgres.Conn(ctx, r.db))
}

func (r *postgresRepository) SetPatient(ctx context.Context, id string) error {
	return postgres.SetLocal(ctx, "app.patient_id", id)
}

func (r *postgresRepository) SetClinic(ctx context.Context, id string) error {
	return postgres.SetLocal(ctx, "app.organization_id", id)
}

func (r *postgresRepository) SetNotice(ctx context.Context, id string) error {
	return postgres.SetLocal(ctx, "app.appointment_id", id)
}

func (r *postgresRepository) DeleteOfAppointment(ctx context.Context, id string) ([]string, error) {
	var u pgtype.UUID
	if u.Scan(id) != nil {
		return nil, nil
	}
	rows, err := r.q(ctx).DeleteQuestionnaireResponsesOfAppointment(ctx, u)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.String())
	}
	return ids, nil
}

func (r *postgresRepository) Insert(ctx context.Context, x Response, at time.Time) (Response, bool, error) {
	var p questionnaireresponsedb.InsertQuestionnaireResponseParams
	if p.OrganizationID.Scan(x.OrganizationID) != nil || p.AppointmentID.Scan(x.AppointmentID) != nil || p.PatientID.Scan(x.PatientID) != nil ||
		p.PractitionerRoleID.Scan(x.PractitionerRoleID) != nil || p.QuestionnaireID.Scan(x.QuestionnaireID) != nil || p.SeriesID.Scan(x.SeriesID) != nil {
		return Response{}, false, errors.New("questionnaireresponse: an id is not a UUID")
	}
	p.Source, p.DueAt, p.At = string(x.Source), ts(x.DueAt), ts(at)
	row, err := r.q(ctx).InsertQuestionnaireResponse(ctx, p)
	switch {
	case errors.Is(err, pgx.ErrNoRows): // a live request of the series exists
		return Response{}, false, nil
	case err != nil:
		return Response{}, false, err
	}
	out, err := toResponse(row)
	return out, err == nil, err
}

func (r *postgresRepository) ByID(ctx context.Context, id string) (Response, error) {
	return r.one(ctx, id, r.q(ctx).QuestionnaireResponseByID)
}

func (r *postgresRepository) Lock(ctx context.Context, id string) (Response, error) {
	return r.one(ctx, id, r.q(ctx).LockQuestionnaireResponse)
}

// one reads the response id with read; ErrNotFound for none or an id that is not a UUID.
func (r *postgresRepository) one(ctx context.Context, id string, read func(context.Context, pgtype.UUID) (questionnaireresponsedb.QuestionnaireResponse, error)) (Response, error) {
	var u pgtype.UUID
	if u.Scan(id) != nil {
		return Response{}, ErrNotFound
	}
	row, err := read(ctx, u)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return Response{}, ErrNotFound
	case err != nil:
		return Response{}, err
	}
	return toResponse(row)
}

func (r *postgresRepository) Stop(ctx context.Context, id string, at time.Time) (Response, error) {
	var u pgtype.UUID
	if u.Scan(id) != nil {
		return Response{}, ErrNotFound
	}
	row, err := r.q(ctx).StopQuestionnaireResponse(ctx, questionnaireresponsedb.StopQuestionnaireResponseParams{ID: u, At: ts(at)})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return Response{}, ErrNotFound
	case err != nil:
		return Response{}, err
	}
	return toResponse(row)
}

func (r *postgresRepository) SaveAnswers(ctx context.Context, id string, answers []Answer, at time.Time) (Response, error) {
	return r.update(ctx, id, answers, func(u pgtype.UUID, b []byte) (questionnaireresponsedb.QuestionnaireResponse, error) {
		return r.q(ctx).SaveQuestionnaireResponseAnswers(ctx, questionnaireresponsedb.SaveQuestionnaireResponseAnswersParams{ID: u, Answers: b, At: ts(at)})
	})
}

func (r *postgresRepository) Complete(ctx context.Context, id string, answers []Answer, at time.Time) (Response, error) {
	return r.update(ctx, id, answers, func(u pgtype.UUID, b []byte) (questionnaireresponsedb.QuestionnaireResponse, error) {
		return r.q(ctx).CompleteQuestionnaireResponse(ctx, questionnaireresponsedb.CompleteQuestionnaireResponseParams{ID: u, Answers: b, At: ts(at)})
	})
}

// update writes answers to the response id with write; ErrNotFound when no response in progress has that id.
func (r *postgresRepository) update(ctx context.Context, id string, answers []Answer,
	write func(pgtype.UUID, []byte) (questionnaireresponsedb.QuestionnaireResponse, error)) (Response, error) {
	var u pgtype.UUID
	if u.Scan(id) != nil {
		return Response{}, ErrNotFound
	}
	if answers == nil {
		answers = []Answer{}
	}
	b, err := json.Marshal(answers)
	if err != nil {
		return Response{}, fmt.Errorf("questionnaireresponse: encode answers: %w", errors.New("not encodable"))
	}
	row, err := write(u, b)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return Response{}, ErrNotFound
	case err != nil:
		return Response{}, err
	}
	return toResponse(row)
}

func (r *postgresRepository) Search(ctx context.Context, f Filter) ([]Response, error) {
	var p questionnaireresponsedb.SearchQuestionnaireResponsesParams
	if (f.AppointmentID != "" && p.AppointmentID.Scan(f.AppointmentID) != nil) || (f.PatientID != "" && p.PatientID.Scan(f.PatientID) != nil) {
		return nil, nil // an id that is not a UUID matches nothing
	}
	if f.Status != "" {
		p.Status = pgtype.Text{String: string(f.Status), Valid: true}
	}
	rows, err := r.q(ctx).SearchQuestionnaireResponses(ctx, p)
	if err != nil {
		return nil, err
	}
	out := make([]Response, 0, len(rows))
	for _, row := range rows {
		x, err := toResponse(row)
		if err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, nil
}

func (r *postgresRepository) DeleteUnsubmittedBefore(ctx context.Context, before time.Time) (int64, error) {
	return r.q(ctx).DeleteUnsubmittedQuestionnaireResponses(ctx, ts(before))
}

func ts(t time.Time) pgtype.Timestamptz { return pgtype.Timestamptz{Time: t, Valid: true} }

// toResponse maps a row to the model. The answers' JSON is the service's own; an error reading it names no
// value.
func toResponse(row questionnaireresponsedb.QuestionnaireResponse) (Response, error) {
	var answers []Answer
	if err := json.Unmarshal(row.Answers, &answers); err != nil {
		return Response{}, fmt.Errorf("questionnaireresponse: stored answers: %w", errors.New("not a list of answers"))
	}
	if len(answers) == 0 {
		answers = nil
	}
	return Response{
		DueAt: row.DueAt.Time, AuthoredAt: row.AuthoredAt.Time, LastUpdated: row.UpdatedAt.Time,
		ID: row.ID.String(), OrganizationID: row.OrganizationID.String(), AppointmentID: row.AppointmentID.String(),
		PatientID: row.PatientID.String(), PractitionerRoleID: row.PractitionerRoleID.String(),
		QuestionnaireID: row.QuestionnaireID.String(), SeriesID: row.SeriesID.String(),
		Source: Source(row.Source), Status: Status(row.Status), Answers: answers, VersionID: row.VersionID,
	}, nil
}
