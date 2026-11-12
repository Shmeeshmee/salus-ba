package condition

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Shmeeshmee/salus/internal/condition/conditiondb"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

type postgresRepository struct {
	db postgres.Querier
}

// NewPostgresRepository returns the Repository on db, using the transaction in the context when there is one:
// salus_app writes in the signing transaction, salus_stats reads the diagnoses, the owner lists the unconfirmed.
func NewPostgresRepository(db postgres.Querier) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) q(ctx context.Context) *conditiondb.Queries {
	return conditiondb.New(postgres.Conn(ctx, r.db))
}

func (r *postgresRepository) Insert(ctx context.Context, cs []Condition) error {
	q := r.q(ctx)
	for _, c := range cs {
		ids, err := uuids(c.ID, c.OrganizationID, c.PatientID, c.EncounterID, c.DocumentID, c.PractitionerRoleID)
		if err != nil {
			return err
		}
		err = q.InsertCondition(ctx, conditiondb.InsertConditionParams{
			ID: ids[0], RecordedAt: pgtype.Timestamptz{Time: c.RecordedAt, Valid: true}, OrganizationID: ids[1],
			PatientID: ids[2], EncounterID: ids[3], DocumentDraftID: ids[4], PractitionerRoleID: ids[5],
			FieldKey: c.FieldKey, Code: c.Code, VerificationStatus: c.VerificationStatus,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *postgresRepository) Diagnoses(ctx context.Context, orgID string, from, to time.Time) ([]Diagnosis, error) {
	org, err := uuids(orgID)
	if err != nil {
		return nil, err
	}
	rows, err := r.q(ctx).ListDiagnoses(ctx, conditiondb.ListDiagnosesParams{
		OrganizationID: org[0], FromAt: pgtype.Timestamptz{Time: from, Valid: true}, ToAt: pgtype.Timestamptz{Time: to, Valid: true},
	})
	if err != nil {
		return nil, err
	}
	out := make([]Diagnosis, len(rows))
	for i, row := range rows {
		out[i] = Diagnosis{RecordedAt: row.RecordedAt.Time.UTC(), PatientID: row.PatientID.String(),
			PractitionerRoleID: row.PractitionerRoleID.String(), Code: row.Code}
	}
	return out, nil
}

func (r *postgresRepository) Unconfirmed(ctx context.Context) ([]string, error) {
	rows, err := r.q(ctx).ListUnconfirmed(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]string, len(rows))
	for i, id := range rows {
		out[i] = id.String()
	}
	return out, nil
}

// uuids parses the ids.
func uuids(ids ...string) ([]pgtype.UUID, error) {
	out := make([]pgtype.UUID, len(ids))
	for i, id := range ids {
		if err := out[i].Scan(id); err != nil {
			return nil, fmt.Errorf("condition: an id that is not a UUID: %w", ErrInvalid)
		}
	}
	return out, nil
}
