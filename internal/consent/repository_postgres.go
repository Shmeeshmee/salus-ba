package consent

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/consent/consentdb"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

// organizationFK is the constraint an unknown clinic violates, per kind.
var organizationFK = map[Kind]string{
	KindCare: "consent_organization_id_fkey", KindTranscription: "transcription_consent_organization_id_fkey",
	KindHistory: "history_consent_organization_id_fkey",
}

// insufficientPrivilege is the SQLSTATE of a row-level security policy's refusal of a row.
const insufficientPrivilege = "42501"

type postgresRepository struct {
	db postgres.Querier
}

// NewPostgresRepository returns the Repository on db, using the transaction in the context when there is one.
// Its queries run as salus_app, so row-level security shows only the consents of the scope set by SetPatient
// or SetClinic.
func NewPostgresRepository(db postgres.Querier) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) SetPatient(ctx context.Context, id string) error {
	return postgres.SetLocal(ctx, "app.patient_id", id)
}

func (r *postgresRepository) SetClinic(ctx context.Context, id string) error {
	return postgres.SetLocal(ctx, "app.organization_id", id)
}

func (r *postgresRepository) Grant(ctx context.Context, kind Kind, patientID, organizationID string, at time.Time) (Consent, error) {
	var p, o pgtype.UUID
	if err := p.Scan(patientID); err != nil {
		return Consent{}, errors.New("consent: the patient id is not a UUID")
	}
	if o.Scan(organizationID) != nil {
		return Consent{}, ErrClinicNotFound
	}
	q := consentdb.New(postgres.Conn(ctx, r.db))
	ts := pgtype.Timestamptz{Time: at, Valid: true}
	var row consentdb.GrantConsentRow
	var err error
	if kind == KindTranscription {
		var t consentdb.GrantTranscriptionConsentRow
		t, err = q.GrantTranscriptionConsent(ctx, consentdb.GrantTranscriptionConsentParams{PatientID: p, OrganizationID: o, DateTime: ts})
		row = consentdb.GrantConsentRow(t)
	} else {
		row, err = q.GrantConsent(ctx, consentdb.GrantConsentParams{PatientID: p, OrganizationID: o, DateTime: ts})
	}
	if pgErr := (*pgconn.PgError)(nil); errors.As(err, &pgErr) && pgErr.ConstraintName == organizationFK[kind] {
		return Consent{}, ErrClinicNotFound
	}
	return toConsent(kind, row, err, ErrActive)
}

func (r *postgresRepository) ByID(ctx context.Context, id string) (Consent, error) {
	var u pgtype.UUID
	if u.Scan(id) != nil {
		return Consent{}, ErrNotFound
	}
	row, err := consentdb.New(postgres.Conn(ctx, r.db)).ConsentByID(ctx, u)
	c, err := toConsent(Kind(row.Kind), consentdb.GrantConsentRow{
		ID: row.ID, UpdatedAt: row.UpdatedAt, DateTime: row.DateTime, RevokedAt: row.RevokedAt, PatientID: row.PatientID,
		OrganizationID: row.OrganizationID, VersionID: row.VersionID, Status: row.Status,
	}, err, ErrNotFound)
	if err != nil {
		return Consent{}, err
	}
	return r.withShared(ctx, c)
}

func (r *postgresRepository) Revoke(ctx context.Context, kind Kind, id string, at time.Time) (Consent, error) {
	var u pgtype.UUID
	if u.Scan(id) != nil {
		return Consent{}, ErrInactive
	}
	q := consentdb.New(postgres.Conn(ctx, r.db))
	ts := pgtype.Timestamptz{Time: at, Valid: true}
	switch kind {
	case KindTranscription:
		row, err := q.RevokeTranscriptionConsent(ctx, consentdb.RevokeTranscriptionConsentParams{ID: u, RevokedAt: ts})
		return toConsent(kind, consentdb.GrantConsentRow(row), err, ErrInactive)
	case KindHistory:
		row, err := q.RevokeHistoryConsent(ctx, consentdb.RevokeHistoryConsentParams{ID: u, RevokedAt: ts})
		return toConsent(kind, consentdb.GrantConsentRow(row), err, ErrInactive)
	}
	row, err := q.RevokeConsent(ctx, consentdb.RevokeConsentParams{ID: u, RevokedAt: ts})
	return toConsent(kind, consentdb.GrantConsentRow(row), err, ErrInactive)
}

func (r *postgresRepository) RevokeTranscription(ctx context.Context, patientID, organizationID string, at time.Time) (Consent, error) {
	var p, o pgtype.UUID
	if p.Scan(patientID) != nil || o.Scan(organizationID) != nil {
		return Consent{}, ErrInactive
	}
	row, err := consentdb.New(postgres.Conn(ctx, r.db)).RevokeActiveTranscriptionConsent(ctx, consentdb.RevokeActiveTranscriptionConsentParams{
		PatientID: p, OrganizationID: o, RevokedAt: pgtype.Timestamptz{Time: at, Valid: true},
	})
	return toConsent(KindTranscription, consentdb.GrantConsentRow(row), err, ErrInactive)
}

func (r *postgresRepository) ByPatient(ctx context.Context, patientID string) ([]Consent, error) {
	var u pgtype.UUID
	if u.Scan(patientID) != nil {
		return nil, nil
	}
	rows, err := consentdb.New(postgres.Conn(ctx, r.db)).ConsentsByPatient(ctx, u)
	if err != nil {
		return nil, err
	}
	consents := make([]Consent, 0, len(rows))
	for _, row := range rows {
		c, _ := toConsent(Kind(row.Kind), consentdb.GrantConsentRow{
			ID: row.ID, UpdatedAt: row.UpdatedAt, DateTime: row.DateTime, RevokedAt: row.RevokedAt, PatientID: row.PatientID,
			OrganizationID: row.OrganizationID, VersionID: row.VersionID, Status: row.Status,
		}, nil, nil)
		consents = append(consents, c)
	}
	return r.withAllShared(ctx, consents)
}

func (r *postgresRepository) State(ctx context.Context, patientID, organizationID string) (access.Consent, error) {
	var p, o pgtype.UUID
	if p.Scan(patientID) != nil || o.Scan(organizationID) != nil {
		return access.Consent{}, nil
	}
	q := consentdb.New(postgres.Conn(ctx, r.db))
	row, err := q.ConsentState(ctx, consentdb.ConsentStateParams{PatientID: p, OrganizationID: o})
	if err != nil {
		return access.Consent{}, err
	}
	st := access.Consent{Active: row.Active, RevokedAt: row.RevokedAt.Time, Transcription: row.Transcription, History: row.History}
	if !st.History {
		return st, nil
	}
	shared, err := q.HistoryShared(ctx, consentdb.HistorySharedParams{PatientID: p, OrganizationID: o})
	if err != nil {
		return access.Consent{}, err
	}
	st.Shared = make([]string, 0, len(shared))
	for _, e := range shared {
		st.Shared = append(st.Shared, e.String())
	}
	return st, nil
}

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

// toConsent maps a row of kind to the model, and no row to noRows.
func toConsent(kind Kind, row consentdb.GrantConsentRow, err, noRows error) (Consent, error) {
	if errors.Is(err, pgx.ErrNoRows) {
		return Consent{}, noRows
	}
	if err != nil {
		return Consent{}, err
	}
	return Consent{
		ID: row.ID.String(), PatientID: row.PatientID.String(), OrganizationID: row.OrganizationID.String(), Kind: kind,
		Status: Status(row.Status), DateTime: row.DateTime.Time, RevokedAt: row.RevokedAt.Time,
		LastUpdated: row.UpdatedAt.Time, VersionID: row.VersionID,
	}, nil
}

func (r *postgresRepository) GrantChat(ctx context.Context, patientID string, g ChatGrant, at time.Time) (Consent, error) {
	var p, o, c, role, roleOrg pgtype.UUID
	if p.Scan(patientID) != nil || o.Scan(g.OrganizationID) != nil || c.Scan(g.ConversationID) != nil || role.Scan(g.PractitionerRoleID) != nil ||
		roleOrg.Scan(g.PractitionerOrganizationID) != nil {
		return Consent{}, errors.New("consent: a chat grant id is not a UUID")
	}
	row, err := consentdb.New(postgres.Conn(ctx, r.db)).GrantChatConsent(ctx, consentdb.GrantChatConsentParams{
		PatientID: p, OrganizationID: o, ConversationID: c, PractitionerRoleID: role, PractitionerOrganizationID: roleOrg,
		DateTime: pgtype.Timestamptz{Time: at, Valid: true},
	})
	if err != nil {
		return Consent{}, fmt.Errorf("consent: grant chat: %w", err)
	}
	return chatConsent(row), nil
}

func (r *postgresRepository) RevokeChat(ctx context.Context, conversationID, roleID string, at time.Time) (Consent, error) {
	var c, role pgtype.UUID
	if c.Scan(conversationID) != nil || role.Scan(roleID) != nil {
		return Consent{}, ErrInactive
	}
	row, err := consentdb.New(postgres.Conn(ctx, r.db)).RevokeChatConsent(ctx, consentdb.RevokeChatConsentParams{
		ConversationID: c, PractitionerRoleID: role, RevokedAt: pgtype.Timestamptz{Time: at, Valid: true},
	})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return Consent{}, ErrInactive
	case err != nil:
		return Consent{}, fmt.Errorf("consent: revoke chat: %w", err)
	}
	return chatConsent(consentdb.GrantChatConsentRow(row)), nil
}

func (r *postgresRepository) ChatGranted(ctx context.Context, conversationID, roleID string) (bool, error) {
	var c, role pgtype.UUID
	if c.Scan(conversationID) != nil || role.Scan(roleID) != nil {
		return false, nil
	}
	ok, err := consentdb.New(postgres.Conn(ctx, r.db)).ChatConsentGranted(ctx, consentdb.ChatConsentGrantedParams{ConversationID: c, PractitionerRoleID: role})
	if err != nil {
		return false, fmt.Errorf("consent: chat granted: %w", err)
	}
	return ok, nil
}

func chatConsent(row consentdb.GrantChatConsentRow) Consent {
	return Consent{
		ID: row.ID.String(), PatientID: row.PatientID.String(), OrganizationID: row.OrganizationID.String(),
		ConversationID: row.ConversationID.String(), PractitionerRoleID: row.PractitionerRoleID.String(),
		PractitionerOrganizationID: row.PractitionerOrganizationID.String(), Kind: KindChat,
		Status: Status(row.Status), DateTime: row.DateTime.Time, RevokedAt: row.RevokedAt.Time, LastUpdated: row.UpdatedAt.Time,
		VersionID: row.VersionID,
	}
}
