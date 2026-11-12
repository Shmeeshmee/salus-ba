package consent

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Shmeeshmee/salus/internal/consent/consentdb"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

func (r *postgresRepository) RevokeHistory(ctx context.Context, patientID, organizationID string, at time.Time) (Consent, error) {
	var p, o pgtype.UUID
	if p.Scan(patientID) != nil || o.Scan(organizationID) != nil {
		return Consent{}, ErrInactive
	}
	row, err := consentdb.New(postgres.Conn(ctx, r.db)).RevokeActiveHistoryConsent(ctx, consentdb.RevokeActiveHistoryConsentParams{
		PatientID: p, OrganizationID: o, RevokedAt: pgtype.Timestamptz{Time: at, Valid: true},
	})
	return toConsent(KindHistory, consentdb.GrantConsentRow(row), err, ErrInactive)
}

func (r *postgresRepository) ActiveHistory(ctx context.Context, patientID, organizationID string) (Consent, error) {
	var p, o pgtype.UUID
	if p.Scan(patientID) != nil || o.Scan(organizationID) != nil {
		return Consent{}, ErrNotFound
	}
	row, err := consentdb.New(postgres.Conn(ctx, r.db)).ActiveHistoryConsent(ctx, consentdb.ActiveHistoryConsentParams{PatientID: p, OrganizationID: o})
	c, err := toConsent(KindHistory, consentdb.GrantConsentRow(row), err, ErrNotFound)
	if err != nil {
		return Consent{}, err
	}
	return r.withShared(ctx, c)
}

func (r *postgresRepository) GrantHistory(ctx context.Context, patientID, organizationID string, encounterIDs []string, at time.Time) (Consent, error) {
	var p, o pgtype.UUID
	if err := p.Scan(patientID); err != nil {
		return Consent{}, errors.New("consent: the patient id is not a UUID")
	}
	if o.Scan(organizationID) != nil {
		return Consent{}, ErrClinicNotFound
	}
	q := consentdb.New(postgres.Conn(ctx, r.db))
	ts := pgtype.Timestamptz{Time: at, Valid: true}
	row, err := q.GrantHistoryConsent(ctx, consentdb.GrantHistoryConsentParams{PatientID: p, OrganizationID: o, DateTime: ts})
	if pgErr := (*pgconn.PgError)(nil); errors.As(err, &pgErr) && pgErr.ConstraintName == organizationFK[KindHistory] {
		return Consent{}, ErrClinicNotFound
	}
	c, err := toConsent(KindHistory, consentdb.GrantConsentRow(row), err, ErrActive)
	if err != nil {
		return Consent{}, err
	}
	if err := r.add(ctx, q, row.ID, encounterIDs, ts); err != nil {
		return Consent{}, err
	}
	return r.withShared(ctx, c)
}

func (r *postgresRepository) ChangeHistory(ctx context.Context, id string, version int32, add, remove []string, at time.Time) (Consent, error) {
	var u pgtype.UUID
	if u.Scan(id) != nil {
		return Consent{}, ErrVersionConflict
	}
	q := consentdb.New(postgres.Conn(ctx, r.db))
	ts := pgtype.Timestamptz{Time: at, Valid: true}
	row, err := q.ChangeHistoryConsent(ctx, consentdb.ChangeHistoryConsentParams{ID: u, VersionID: version, UpdatedAt: ts})
	c, err := toConsent(KindHistory, consentdb.GrantConsentRow(row), err, ErrVersionConflict)
	if err != nil {
		return Consent{}, err
	}
	if len(remove) > 0 {
		gone, ok := uuids(remove)
		if !ok {
			return Consent{}, ErrInvalidVisit
		}
		if err := q.RemoveHistoryVisits(ctx, consentdb.RemoveHistoryVisitsParams{ConsentID: u, EncounterIds: gone, RemovedAt: ts}); err != nil {
			return Consent{}, err
		}
	}
	if err := r.add(ctx, q, u, add, ts); err != nil {
		return Consent{}, err
	}
	return r.withShared(ctx, c)
}

// add lists the encounters in the history consent id from at; ErrInvalidVisit when one is not a UUID, or row-level
// security or the encounter's key refuses it.
func (r *postgresRepository) add(ctx context.Context, q *consentdb.Queries, id pgtype.UUID, encounterIDs []string, at pgtype.Timestamptz) error {
	if len(encounterIDs) == 0 {
		return nil
	}
	es, ok := uuids(encounterIDs)
	if !ok {
		return ErrInvalidVisit
	}
	err := q.AddHistoryVisits(ctx, consentdb.AddHistoryVisitsParams{ConsentID: id, EncounterIds: es, AddedAt: at})
	if pgErr := (*pgconn.PgError)(nil); errors.As(err, &pgErr) &&
		(pgErr.Code == insufficientPrivilege || pgErr.ConstraintName == "history_consent_encounter_encounter_id_fkey") {
		return ErrInvalidVisit
	}
	return err
}

// withShared is c with the visits it lists now when it is an active history consent.
func (r *postgresRepository) withShared(ctx context.Context, c Consent) (Consent, error) {
	all, err := r.withAllShared(ctx, []Consent{c})
	if err != nil {
		return Consent{}, err
	}
	return all[0], nil
}

// withAllShared is consents with the visits each active history consent of them lists now, read in one query.
func (r *postgresRepository) withAllShared(ctx context.Context, consents []Consent) ([]Consent, error) {
	var active []pgtype.UUID
	at := map[string]int{}
	for i, c := range consents {
		var u pgtype.UUID
		if c.Kind == KindHistory && c.Status == StatusActive && u.Scan(c.ID) == nil {
			active = append(active, u)
			at[c.ID] = i
			consents[i].Shared = []SharedVisit{}
		}
	}
	if len(active) == 0 {
		return consents, nil
	}
	rows, err := consentdb.New(postgres.Conn(ctx, r.db)).HistoryVisits(ctx, active)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		i := at[row.ConsentID.String()]
		consents[i].Shared = append(consents[i].Shared, SharedVisit{EncounterID: row.EncounterID.String(), AddedAt: row.AddedAt.Time})
	}
	return consents, nil
}

// uuids are ids as UUIDs, and false when one is not a UUID.
func uuids(ids []string) ([]pgtype.UUID, bool) {
	out := make([]pgtype.UUID, len(ids))
	for i, id := range ids {
		if out[i].Scan(id) != nil {
			return nil, false
		}
	}
	return out, true
}
