package questionnaireresponse

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Shmeeshmee/salus/internal/binary"
	"github.com/Shmeeshmee/salus/internal/questionnaireresponse/questionnaireresponsedb"
	"github.com/Shmeeshmee/salus/pkg/keyset"
)

// historyArgs are the parameters of the timeline's queries, or false when an id is not a UUID (which matches
// nothing).
func historyArgs(patientID, org string, p keyset.Page) (questionnaireresponsedb.CompletedOfParams, bool) {
	arg := questionnaireresponsedb.CompletedOfParams{
		FromAt: optTS(p.From), ToAt: optTS(p.To), AfterAt: optTS(p.At), AfterRef: p.Ref, Lim: int32(min(p.Limit, 1000)), //nolint:gosec // bounded
	}
	ok := arg.PatientID.Scan(patientID) == nil && (org == "" || arg.OrganizationID.Scan(org) == nil)
	return arg, ok
}

func (r *postgresRepository) CompletedOf(ctx context.Context, patientID, org string, p keyset.Page) ([]Response, error) {
	arg, ok := historyArgs(patientID, org, p)
	if !ok {
		return nil, nil
	}
	rows, err := r.q(ctx).CompletedOf(ctx, arg)
	if err != nil {
		return nil, err
	}
	out := make([]Response, 0, len(rows))
	for _, row := range rows {
		x, err := toResponse(questionnaireresponsedb.QuestionnaireResponse(row))
		if err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, nil
}

func (r *postgresRepository) FilesOf(ctx context.Context, patientID, org string, p keyset.Page) ([]Attached, error) {
	arg, ok := historyArgs(patientID, org, p)
	if !ok {
		return nil, nil
	}
	rows, err := r.q(ctx).FilesOf(ctx, questionnaireresponsedb.FilesOfParams(arg))
	if err != nil {
		return nil, err
	}
	out := make([]Attached, 0, len(rows))
	for _, row := range rows {
		x, err := toResponse(questionnaireresponsedb.QuestionnaireResponse{
			ID: row.ID, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt, DueAt: row.DueAt, AuthoredAt: row.AuthoredAt,
			OrganizationID: row.OrganizationID, AppointmentID: row.AppointmentID, PatientID: row.PatientID,
			PractitionerRoleID: row.PractitionerRoleID, QuestionnaireID: row.QuestionnaireID, SeriesID: row.SeriesID,
			VersionID: row.VersionID, Source: row.Source, Status: row.Status, Answers: row.Answers, Attachments: row.Attachments,
			Language: row.Language,
		})
		if err != nil {
			return nil, err
		}
		var file binary.Attachment
		if err := json.Unmarshal(row.File, &file); err != nil {
			return nil, fmt.Errorf("questionnaireresponse: stored attachment: %w", errors.New("not an attachment"))
		}
		out = append(out, Attached{Response: x, File: file})
	}
	return out, nil
}

// optTS is t, or null when it is zero.
func optTS(t time.Time) pgtype.Timestamptz { return pgtype.Timestamptz{Time: t, Valid: !t.IsZero()} }
