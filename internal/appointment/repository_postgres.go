package appointment

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Shmeeshmee/salus/internal/appointment/appointmentdb"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

// The answers of appointment_no_double_booking to an overlapping booking: 23P01 when the other booking had
// committed; 40P01 when both were inserting at once, as two concurrent inserts into an exclusion constraint
// each wait for the other and Postgres aborts one of them (the other commits). Both are a clash.
const (
	exclusionViolation = "23P01"
	deadlockDetected   = "40P01"
)

type postgresRepository struct {
	db postgres.Querier
}

// NewPostgresRepository returns the Repository on db, using the transaction in the context when there is one.
func NewPostgresRepository(db postgres.Querier) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) q(ctx context.Context) *appointmentdb.Queries {
	return appointmentdb.New(postgres.Conn(ctx, r.db))
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

func (r *postgresRepository) Create(ctx context.Context, a Appointment) (Appointment, error) {
	var p, role, org, sl pgtype.UUID
	if p.Scan(a.PatientID) != nil || role.Scan(a.PractitionerRoleID) != nil || org.Scan(a.OrganizationID) != nil || sl.Scan(a.SlotID) != nil {
		return Appointment{}, errors.New("appointment: an id is not a UUID")
	}
	row, err := r.q(ctx).CreateAppointment(ctx, appointmentdb.CreateAppointmentParams{
		PatientID: p, PractitionerRoleID: role, OrganizationID: org, SlotID: sl,
		StartAt: ts(a.Start), EndAt: ts(a.End), CreatedAt: ts(a.Created), BookedVia: string(bookedVia(a.BookedVia)),
	})
	var pgErr *pgconn.PgError
	switch {
	case errors.As(err, &pgErr) && (pgErr.Code == exclusionViolation || pgErr.Code == deadlockDetected):
		return Appointment{}, ErrSlotUnavailable
	case err != nil:
		return Appointment{}, err
	}
	return toAppointment(appointmentdb.AppointmentByIDRow(row)), nil
}

func (r *postgresRepository) ByID(ctx context.Context, id string) (Appointment, error) {
	var u pgtype.UUID
	if u.Scan(id) != nil {
		return Appointment{}, ErrNotFound
	}
	row, err := r.q(ctx).AppointmentByID(ctx, u)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return Appointment{}, ErrNotFound
	case err != nil:
		return Appointment{}, err
	}
	return toAppointment(row), nil
}

func (r *postgresRepository) ByIDForShare(ctx context.Context, id string) (Appointment, error) {
	var u pgtype.UUID
	if u.Scan(id) != nil {
		return Appointment{}, ErrNotFound
	}
	row, err := r.q(ctx).AppointmentByIDForShare(ctx, u)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return Appointment{}, ErrNotFound
	case err != nil:
		return Appointment{}, err
	}
	return toAppointment(appointmentdb.AppointmentByIDRow(row)), nil
}

func (r *postgresRepository) Search(ctx context.Context, f Filter) ([]Appointment, error) {
	var p appointmentdb.SearchAppointmentsParams
	for _, id := range []struct {
		dst *pgtype.UUID
		v   string
	}{{&p.PatientID, f.PatientID}, {&p.OrganizationID, f.OrganizationID}, {&p.PractitionerRoleID, f.PractitionerRoleID}} {
		if id.v != "" && id.dst.Scan(id.v) != nil {
			return nil, nil // an id that is not a UUID names no appointment
		}
	}
	p.Status = pgtype.Text{String: string(f.Status), Valid: f.Status != ""}
	p.FromAt = pgtype.Timestamptz{Time: f.From, Valid: !f.From.IsZero()}
	p.ToAt = pgtype.Timestamptz{Time: f.To, Valid: !f.To.IsZero()}
	rows, err := r.q(ctx).SearchAppointments(ctx, p)
	if err != nil {
		return nil, err
	}
	out := make([]Appointment, 0, len(rows))
	for _, row := range rows {
		out = append(out, toAppointment(appointmentdb.AppointmentByIDRow(row)))
	}
	return out, nil
}

func (r *postgresRepository) Cancel(ctx context.Context, id string, reason Reason, by string, at time.Time) (Appointment, error) {
	var u, acc pgtype.UUID
	if u.Scan(id) != nil {
		return Appointment{}, ErrClosed
	}
	if by != "" && acc.Scan(by) != nil {
		return Appointment{}, errors.New("appointment: the account id is not a UUID")
	}
	row, err := r.q(ctx).CancelAppointment(ctx, appointmentdb.CancelAppointmentParams{
		ID: u, At: ts(at), Reason: pgtype.Text{String: string(reason), Valid: true}, CancelledBy: acc,
	})
	return changed(appointmentdb.AppointmentByIDRow(row), err)
}

func (r *postgresRepository) SetStatus(ctx context.Context, id string, status Status, at time.Time) (Appointment, error) {
	var u pgtype.UUID
	if u.Scan(id) != nil {
		return Appointment{}, ErrClosed
	}
	row, err := r.q(ctx).SetAppointmentStatus(ctx, appointmentdb.SetAppointmentStatusParams{ID: u, Status: string(status), At: ts(at)})
	return changed(appointmentdb.AppointmentByIDRow(row), err)
}

// changed maps the result of an update that touches only a booked appointment: no row is ErrClosed.
func changed(row appointmentdb.AppointmentByIDRow, err error) (Appointment, error) {
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return Appointment{}, ErrClosed
	case err != nil:
		return Appointment{}, err
	}
	return toAppointment(row), nil
}

func ts(t time.Time) pgtype.Timestamptz { return pgtype.Timestamptz{Time: t, Valid: true} }

// toAppointment maps a row; every appointment query selects the same columns.
func toAppointment(row appointmentdb.AppointmentByIDRow) Appointment {
	a := Appointment{
		Start: row.StartAt.Time, End: row.EndAt.Time, Created: row.CreatedAt.Time, LastUpdated: row.UpdatedAt.Time,
		ID: row.ID.String(), PatientID: row.PatientID.String(), PractitionerRoleID: row.PractitionerRoleID.String(),
		OrganizationID: row.OrganizationID.String(), SlotID: row.SlotID.String(),
		Status: Status(row.Status), Reason: Reason(row.CancellationReason.String), VersionID: row.VersionID,
	}
	if row.CancelledAt.Valid {
		a.CancelledAt = row.CancelledAt.Time
	}
	if row.CancelledByAccountID.Valid {
		a.CancelledBy = row.CancelledByAccountID.String()
	}
	if row.ConfirmedAt.Valid {
		a.ConfirmedAt, a.ConfirmedVia = row.ConfirmedAt.Time, Via(row.ConfirmedVia.String)
	}
	a.RescheduleCount, a.BookedVia = int32(row.RescheduleCount), BookedVia(row.BookedVia)
	return a
}

// bookedVia is v, or BookedDirect when unset.
func bookedVia(v BookedVia) BookedVia {
	if v == "" {
		return BookedDirect
	}
	return v
}

type postgresStatsRepository struct {
	db postgres.Querier
}

// NewPostgresStatsRepository returns the StatsRepository on db, the salus_stats pool.
func NewPostgresStatsRepository(db postgres.Querier) StatsRepository {
	return &postgresStatsRepository{db: db}
}

func (r *postgresStatsRepository) Booked(ctx context.Context, orgID string, from, to time.Time) ([]Booked, error) {
	var org pgtype.UUID
	if org.Scan(orgID) != nil {
		return nil, ErrNotFound
	}
	rows, err := appointmentdb.New(postgres.Conn(ctx, r.db)).ListBookedForStats(ctx, appointmentdb.ListBookedForStatsParams{
		OrganizationID: org, FromAt: pgtype.Timestamptz{Time: from, Valid: true}, ToAt: pgtype.Timestamptz{Time: to, Valid: true},
	})
	if err != nil {
		return nil, err
	}
	out := make([]Booked, len(rows))
	for i, row := range rows {
		out[i] = Booked{Start: row.StartAt.Time.UTC(), PatientID: row.PatientID.String(),
			PractitionerRoleID: row.PractitionerRoleID.String(), Status: Status(row.Status)}
	}
	return out, nil
}

func (r *postgresRepository) AddChange(ctx context.Context, c Change) (string, error) {
	var id pgtype.UUID
	if id.Scan(c.AppointmentID) != nil {
		return "", ErrNotFound
	}
	p := appointmentdb.InsertAppointmentChangeParams{
		At: ts(c.At), AppointmentID: id, Kind: string(c.Kind), Via: string(c.Via),
		Agent: text(c.Agent), FromStatus: text(string(c.FromStatus)), ToStatus: text(string(c.ToStatus)),
		FromStart: optionalTS(c.FromStart), FromEnd: optionalTS(c.FromEnd), ToStart: optionalTS(c.ToStart), ToEnd: optionalTS(c.ToEnd),
	}
	for _, u := range []struct {
		dst *pgtype.UUID
		v   string
	}{{&p.ByAccountID, c.ByAccountID}, {&p.FromSlotID, c.FromSlotID}, {&p.ToSlotID, c.ToSlotID}} {
		if u.v != "" && u.dst.Scan(u.v) != nil {
			return "", errors.New("appointment: an id of the change is not a UUID")
		}
	}
	out, err := r.q(ctx).InsertAppointmentChange(ctx, p)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return "", ErrNotFound
	case err != nil:
		return "", err
	}
	return out.String(), nil
}

func (r *postgresRepository) Changes(ctx context.Context, id string) ([]Change, error) {
	var u pgtype.UUID
	if u.Scan(id) != nil {
		return nil, nil
	}
	rows, err := r.q(ctx).ListAppointmentChanges(ctx, u)
	if err != nil {
		return nil, err
	}
	out := make([]Change, 0, len(rows))
	for _, row := range rows {
		c := Change{
			At: row.At.Time, ID: row.ID.String(), AppointmentID: row.AppointmentID.String(), Kind: ChangeKind(row.Kind), Via: Via(row.Via),
			Agent: row.Agent.String, FromStatus: Status(row.FromStatus.String), ToStatus: Status(row.ToStatus.String),
			FromStart: row.FromStart.Time, FromEnd: row.FromEnd.Time, ToStart: row.ToStart.Time, ToEnd: row.ToEnd.Time,
		}
		for _, u := range []struct {
			dst *string
			v   pgtype.UUID
		}{{&c.ByAccountID, row.ByAccountID}, {&c.FromSlotID, row.FromSlotID}, {&c.ToSlotID, row.ToSlotID}} {
			if u.v.Valid {
				*u.dst = u.v.String()
			}
		}
		out = append(out, c)
	}
	return out, nil
}

// NewPostgresTokenRepository returns the TokenRepository on db, using the transaction in the context when there is
// one.
func NewPostgresTokenRepository(db postgres.Querier) TokenRepository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) InsertToken(ctx context.Context, t Token) error {
	p := appointmentdb.InsertActionTokenParams{
		CreatedAt: ts(t.CreatedAt), StartAt: ts(t.StartAt), TokenHash: t.Hash[:], Channel: string(t.Channel),
		Language: appointmentdb.Language(t.Language),
	}
	for _, u := range []struct {
		dst *pgtype.UUID
		v   string
	}{{&p.AppointmentID, t.AppointmentID}, {&p.OrganizationID, t.OrganizationID}, {&p.PatientID, t.PatientID}, {&p.AccountID, t.AccountID}, {&p.ReminderID, t.ReminderID}} {
		if u.v != "" && u.dst.Scan(u.v) != nil {
			return errors.New("appointment: an id of the link is not a UUID")
		}
	}
	return r.q(ctx).InsertActionToken(ctx, p)
}

func (r *postgresRepository) DeleteTokensExpiredBefore(ctx context.Context, before time.Time) (int64, error) {
	return r.q(ctx).DeleteActionTokensExpiredBefore(ctx, ts(before))
}

func text(s string) pgtype.Text { return pgtype.Text{String: s, Valid: s != ""} }

func optionalTS(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: !t.IsZero()}
}
func (r *postgresStatsRepository) Fulfilled(ctx context.Context, orgID string, from, to time.Time) ([]Booked, error) {
	var org pgtype.UUID
	if org.Scan(orgID) != nil {
		return nil, ErrNotFound
	}
	rows, err := appointmentdb.New(postgres.Conn(ctx, r.db)).ListFulfilledForStats(ctx, appointmentdb.ListFulfilledForStatsParams{
		OrganizationID: org, FromAt: pgtype.Timestamptz{Time: from, Valid: true}, ToAt: pgtype.Timestamptz{Time: to, Valid: true},
	})
	if err != nil {
		return nil, err
	}
	out := make([]Booked, len(rows))
	for i, row := range rows {
		out[i] = Booked{Start: row.StartAt.Time.UTC(), PatientID: row.PatientID.String(),
			PractitionerRoleID: row.PractitionerRoleID.String(), Status: Status(row.Status)}
	}
	return out, nil
}
