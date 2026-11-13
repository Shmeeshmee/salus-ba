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
			PractitionerRoleID: row.PractitionerRoleID.String(), Status: Status(row.Status), BookedVia: BookedVia(row.BookedVia)}
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

// SAL-34: the patient's confirmation and reschedule, and the use of a one-time link.

// NewPostgresSelfServiceRepository returns the SelfServiceRepository on db, using the transaction in the context when
// there is one.
func NewPostgresSelfServiceRepository(db postgres.Querier) SelfServiceRepository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) ByIDForUpdate(ctx context.Context, id string) (Appointment, error) {
	var u pgtype.UUID
	if u.Scan(id) != nil {
		return Appointment{}, ErrNotFound
	}
	row, err := r.q(ctx).AppointmentByIDForUpdate(ctx, u)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return Appointment{}, ErrNotFound
	case err != nil:
		return Appointment{}, err
	}
	return toAppointment(appointmentdb.AppointmentByIDRow(row)), nil
}

func (r *postgresRepository) Confirm(ctx context.Context, id string, via Via, at time.Time) (Appointment, error) {
	var u pgtype.UUID
	if u.Scan(id) != nil {
		return Appointment{}, ErrClosed
	}
	row, err := r.q(ctx).ConfirmAppointment(ctx, appointmentdb.ConfirmAppointmentParams{ID: u, Via: text(string(via)), At: ts(at)})
	return changed(appointmentdb.AppointmentByIDRow(row), err)
}

func (r *postgresRepository) Reschedule(ctx context.Context, id string, m Move, at time.Time) (Appointment, error) {
	var u, sl pgtype.UUID
	if u.Scan(id) != nil {
		return Appointment{}, ErrClosed
	}
	if sl.Scan(m.SlotID) != nil {
		return Appointment{}, ErrSlotUnavailable
	}
	row, err := r.q(ctx).RescheduleAppointment(ctx, appointmentdb.RescheduleAppointmentParams{
		ID: u, SlotID: sl, StartAt: ts(m.Start), EndAt: ts(m.End), At: ts(at), FromCount: int16(m.From),
	})
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && (pgErr.Code == exclusionViolation || pgErr.Code == deadlockDetected) {
		return Appointment{}, ErrSlotUnavailable
	}
	return changed(appointmentdb.AppointmentByIDRow(row), err)
}

func (r *postgresRepository) FindToken(ctx context.Context, hash [32]byte) (Token, error) {
	row, err := r.q(ctx).FindActionToken(ctx, hash[:])
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return Token{}, ErrLinkInvalid
	case err != nil:
		return Token{}, err
	}
	return Token{
		ID: row.ID.String(), AppointmentID: row.AppointmentID.String(), OrganizationID: row.OrganizationID.String(),
		PatientID: row.PatientID.String(), AccountID: row.AccountID.String(),
	}, nil
}

func (r *postgresRepository) TokenByID(ctx context.Context, id string) (Token, error) {
	var u pgtype.UUID
	if u.Scan(id) != nil {
		return Token{}, ErrLinkInvalid
	}
	row, err := r.q(ctx).ActionTokenByID(ctx, u)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return Token{}, ErrLinkInvalid
	case err != nil:
		return Token{}, err
	}
	t := Token{
		CreatedAt: row.CreatedAt.Time, StartAt: row.StartAt.Time, ID: row.ID.String(), AppointmentID: row.AppointmentID.String(),
		OrganizationID: row.OrganizationID.String(), PatientID: row.PatientID.String(), AccountID: row.AccountID.String(),
		Language: string(row.Language), Channel: Channel(row.Channel), UsedAction: LinkAction(row.UsedAction.String),
	}
	copy(t.Hash[:], row.TokenHash)
	if row.ReminderID.Valid {
		t.ReminderID = row.ReminderID.String()
	}
	if row.UsedAt.Valid {
		t.UsedAt = row.UsedAt.Time
	}
	return t, nil
}

func (r *postgresRepository) UseToken(ctx context.Context, id string, action LinkAction, at time.Time) error {
	var u pgtype.UUID
	if u.Scan(id) != nil {
		return ErrLinkInvalid
	}
	n, err := r.q(ctx).UseActionToken(ctx, appointmentdb.UseActionTokenParams{ID: u, Action: text(string(action)), At: ts(at)})
	switch {
	case err != nil:
		return err
	case n == 0:
		return ErrLinkInvalid
	}
	return nil
}

// SAL-33: the reminders.

// NewPostgresReminderRepository returns the ReminderRepository on db, using the transaction in the context when there
// is one: salus_app's pool, or the retention role's for DeleteRemindersDueBefore.
func NewPostgresReminderRepository(db postgres.Querier) ReminderRepository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) AddReminder(ctx context.Context, x Reminder) (Reminder, bool, error) {
	var p appointmentdb.InsertReminderParams
	if p.AppointmentID.Scan(x.AppointmentID) != nil || p.OrganizationID.Scan(x.OrganizationID) != nil || p.PatientID.Scan(x.PatientID) != nil {
		return Reminder{}, false, errors.New("appointment: an id of the reminder is not a UUID")
	}
	p.OffsetMinutes, p.StartAt, p.DueAt = int32(x.OffsetMinutes), ts(x.StartAt), ts(x.DueAt)
	row, err := r.q(ctx).InsertReminder(ctx, p)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return Reminder{}, false, nil
	case err != nil:
		return Reminder{}, false, err
	}
	x.ID, x.State = row.ID.String(), ReminderScheduled
	return x, true, nil
}

func (r *postgresRepository) CancelReminders(ctx context.Context, appointmentID string) (int64, error) {
	var u pgtype.UUID
	if u.Scan(appointmentID) != nil {
		return 0, nil
	}
	return r.q(ctx).CancelReminders(ctx, u)
}

func (r *postgresRepository) LockReminder(ctx context.Context, id string) (Reminder, error) {
	var u pgtype.UUID
	if u.Scan(id) != nil {
		return Reminder{}, ErrNotFound
	}
	row, err := r.q(ctx).LockReminder(ctx, u)
	return toReminder(appointmentdb.ReminderByIDRow(row), err)
}

func (r *postgresRepository) Reminder(ctx context.Context, id string) (Reminder, error) {
	var u pgtype.UUID
	if u.Scan(id) != nil {
		return Reminder{}, ErrNotFound
	}
	return toReminder(r.q(ctx).ReminderByID(ctx, u))
}

func toReminder(row appointmentdb.ReminderByIDRow, err error) (Reminder, error) {
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return Reminder{}, ErrNotFound
	case err != nil:
		return Reminder{}, err
	}
	return Reminder{
		StartAt: row.StartAt.Time, DueAt: row.DueAt.Time, DispatchedAt: row.DispatchedAt.Time,
		ID: row.ID.String(), AppointmentID: row.AppointmentID.String(), OrganizationID: row.OrganizationID.String(),
		PatientID: row.PatientID.String(), State: ReminderState(row.State), OffsetMinutes: int(row.OffsetMinutes),
	}, nil
}

func (r *postgresRepository) SetReminderState(ctx context.Context, id string, state ReminderState, at time.Time) error {
	p := appointmentdb.SetReminderStateParams{State: string(state)}
	if p.ID.Scan(id) != nil {
		return ErrNotFound
	}
	if state == ReminderDispatched {
		p.DispatchedAt = ts(at)
	}
	n, err := r.q(ctx).SetReminderState(ctx, p)
	if err == nil && n == 0 {
		return ErrReminderGone
	}
	return err
}

func (r *postgresRepository) OverdueReminders(ctx context.Context, at, afterDue time.Time, afterID string, max int) ([]Reminder, error) {
	// From the first page on: a zero cursor is before every due time.
	p := appointmentdb.OverdueRemindersParams{At: ts(at), AfterDue: pgtype.Timestamptz{Valid: true, InfinityModifier: pgtype.NegativeInfinity}, Max: int32(max)}
	if !afterDue.IsZero() {
		p.AfterDue = ts(afterDue)
	}
	if p.AfterID.Scan(afterID) != nil {
		return nil, errors.New("appointment: the sweep's cursor is not a UUID")
	}
	rows, err := r.q(ctx).OverdueReminders(ctx, p)
	if err != nil {
		return nil, err
	}
	out := make([]Reminder, len(rows))
	for i, row := range rows {
		out[i] = Reminder{ID: row.ID.String(), OrganizationID: row.OrganizationID.String(), DueAt: row.DueAt.Time}
	}
	return out, nil
}

func (r *postgresRepository) DeleteRemindersDueBefore(ctx context.Context, before time.Time) (int64, error) {
	return r.q(ctx).DeleteRemindersDueBefore(ctx, ts(before))
}

// SAL-35: the waitlist.

// NewPostgresWaitlistRepository returns the WaitlistRepository on db, using the transaction in the context when there
// is one; on the retention role's pool it purges every clinic's.
func NewPostgresWaitlistRepository(db postgres.Querier) WaitlistRepository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) InsertEntry(ctx context.Context, e WaitlistEntry) (WaitlistEntry, error) {
	p := appointmentdb.InsertWaitlistEntryParams{
		CreatedAt: ts(e.Created), EndsAt: ts(e.EndsAt), Specialty: text(e.Specialty),
		Earliest: pgDate(e.Earliest), Latest: pgDate(e.Latest), TimesStartMinute: minute(e.Times, e.Times.Start), TimesEndMinute: minute(e.Times, e.Times.End),
	}
	if p.PatientID.Scan(e.PatientID) != nil || p.OrganizationID.Scan(e.OrganizationID) != nil ||
		(e.PractitionerRoleID != "" && p.PractitionerRoleID.Scan(e.PractitionerRoleID) != nil) {
		return WaitlistEntry{}, errors.New("appointment: an id of the entry is not a UUID")
	}
	row, err := r.q(ctx).InsertWaitlistEntry(ctx, p)
	if err != nil {
		return WaitlistEntry{}, err
	}
	return toEntry(appointmentdb.WaitlistEntryForUpdateRow(row)), nil
}

func (r *postgresRepository) CountActiveEntries(ctx context.Context, patientID, org string, now time.Time) (int, error) {
	var p appointmentdb.CountActiveWaitlistEntriesParams
	if p.PatientID.Scan(patientID) != nil || p.OrganizationID.Scan(org) != nil {
		return 0, nil
	}
	p.Now = ts(now)
	n, err := r.q(ctx).CountActiveWaitlistEntries(ctx, p)
	return int(n), err
}

func (r *postgresRepository) PatientEntries(ctx context.Context, patientID string) ([]WaitlistEntry, error) {
	var u pgtype.UUID
	if u.Scan(patientID) != nil {
		return nil, nil
	}
	rows, err := r.q(ctx).ListPatientWaitlistEntries(ctx, u)
	if err != nil {
		return nil, err
	}
	out := make([]WaitlistEntry, 0, len(rows))
	for _, row := range rows {
		out = append(out, toEntryWithOffer(appointmentdb.ListClinicWaitlistEntriesRow(row)))
	}
	return out, nil
}

func (r *postgresRepository) ClinicEntries(ctx context.Context, q ClinicQuery) ([]WaitlistEntry, error) {
	p := appointmentdb.ListClinicWaitlistEntriesParams{
		Specialty: text(q.Specialty), Status: text(string(q.Status)), Now: ts(q.Now),
		MaxRows: int32(q.Limit), SkipRows: int32(q.Offset), //nolint:gosec // bounded by the service
	}
	if p.OrganizationID.Scan(q.OrganizationID) != nil || (q.PractitionerRoleID != "" && p.PractitionerRoleID.Scan(q.PractitionerRoleID) != nil) {
		return nil, nil
	}
	rows, err := r.q(ctx).ListClinicWaitlistEntries(ctx, p)
	if err != nil {
		return nil, err
	}
	out := make([]WaitlistEntry, 0, len(rows))
	for _, row := range rows {
		out = append(out, toEntryWithOffer(row))
	}
	return out, nil
}

func (r *postgresRepository) LockEntries(ctx context.Context, patientID, org string) error {
	return r.q(ctx).LockWaitlistEntries(ctx, appointmentdb.LockWaitlistEntriesParams{PatientID: patientID, OrganizationID: org})
}

func (r *postgresRepository) EntryForUpdate(ctx context.Context, id string) (WaitlistEntry, error) {
	var u pgtype.UUID
	if u.Scan(id) != nil {
		return WaitlistEntry{}, ErrNotFound
	}
	row, err := r.q(ctx).WaitlistEntryForUpdate(ctx, u)
	return entryOrNotFound(row, err)
}

func (r *postgresRepository) CloseEntry(ctx context.Context, id string, status WaitlistStatus, at time.Time) (WaitlistEntry, error) {
	var u pgtype.UUID
	if u.Scan(id) != nil {
		return WaitlistEntry{}, ErrNotFound
	}
	row, err := r.q(ctx).CloseWaitlistEntry(ctx, appointmentdb.CloseWaitlistEntryParams{ID: u, Status: string(status), At: ts(at)})
	return entryOrNotFound(appointmentdb.WaitlistEntryForUpdateRow(row), err)
}

func (r *postgresRepository) SetPriority(ctx context.Context, id string, priority int, at time.Time) (WaitlistEntry, error) {
	var u pgtype.UUID
	if u.Scan(id) != nil {
		return WaitlistEntry{}, ErrNotFound
	}
	row, err := r.q(ctx).SetWaitlistPriority(ctx, appointmentdb.SetWaitlistPriorityParams{ID: u, Priority: int16(priority), At: ts(at)}) //nolint:gosec // 0 to 100, checked by the service
	return entryOrNotFound(appointmentdb.WaitlistEntryForUpdateRow(row), err)
}

func (r *postgresRepository) ActiveEntries(ctx context.Context, patientID, org string) ([]WaitlistEntry, error) {
	var p appointmentdb.ActiveWaitlistEntriesParams
	if p.PatientID.Scan(patientID) != nil || p.OrganizationID.Scan(org) != nil {
		return nil, nil
	}
	rows, err := r.q(ctx).ActiveWaitlistEntries(ctx, p)
	if err != nil {
		return nil, err
	}
	out := make([]WaitlistEntry, 0, len(rows))
	for _, row := range rows {
		out = append(out, toEntry(appointmentdb.WaitlistEntryForUpdateRow(row)))
	}
	return out, nil
}

func (r *postgresRepository) Candidates(ctx context.Context, m Match, n int) ([]WaitlistEntry, error) {
	p := appointmentdb.WaitlistCandidatesParams{
		Now: ts(m.Now), Specialty: text(m.Specialty), Day: pgDate(m.Day), StartMinute: int16(m.Times.Start), EndMinute: int16(m.Times.End), //nolint:gosec // minutes of a day
		StartAt: ts(m.Start), EndAt: ts(m.End), MaxRows: int32(n), //nolint:gosec // a small limit
	}
	if p.OrganizationID.Scan(m.OrganizationID) != nil || p.PractitionerRoleID.Scan(m.PractitionerRoleID) != nil || p.SlotID.Scan(m.SlotID) != nil {
		return nil, errors.New("appointment: an id of the match is not a UUID")
	}
	p.Skip = make([]pgtype.UUID, len(m.Skip))
	for i, id := range m.Skip {
		if p.Skip[i].Scan(id) != nil {
			return nil, errors.New("appointment: an entry to pass over is not a UUID")
		}
	}
	rows, err := r.q(ctx).WaitlistCandidates(ctx, p)
	if err != nil {
		return nil, err
	}
	out := make([]WaitlistEntry, 0, len(rows))
	for _, row := range rows {
		out = append(out, toEntry(appointmentdb.WaitlistEntryForUpdateRow(row)))
	}
	return out, nil
}

func (r *postgresRepository) InsertOffer(ctx context.Context, o Offer) (Offer, error) {
	p := appointmentdb.InsertWaitlistOfferParams{CreatedAt: ts(o.Created), ExpiresAt: ts(o.ExpiresAt), StartAt: ts(o.Start), EndAt: ts(o.End)}
	for _, u := range []struct {
		dst *pgtype.UUID
		v   string
	}{{&p.EntryID, o.EntryID}, {&p.SlotID, o.SlotID}, {&p.OrganizationID, o.OrganizationID}, {&p.PatientID, o.PatientID}, {&p.PractitionerRoleID, o.PractitionerRoleID}} {
		if u.dst.Scan(u.v) != nil {
			return Offer{}, errors.New("appointment: an id of the offer is not a UUID")
		}
	}
	row, err := r.q(ctx).InsertWaitlistOffer(ctx, p)
	if err != nil {
		return Offer{}, err
	}
	return toOffer(appointmentdb.WaitlistOfferByIDRow(row)), nil
}

func (r *postgresRepository) Offer(ctx context.Context, id string) (Offer, error) {
	var u pgtype.UUID
	if u.Scan(id) != nil {
		return Offer{}, ErrNotFound
	}
	row, err := r.q(ctx).WaitlistOfferByID(ctx, u)
	return offerOrNotFound(row, err)
}

func (r *postgresRepository) OfferForUpdate(ctx context.Context, id string) (Offer, error) {
	var u pgtype.UUID
	if u.Scan(id) != nil {
		return Offer{}, ErrNotFound
	}
	row, err := r.q(ctx).WaitlistOfferForUpdate(ctx, u)
	return offerOrNotFound(appointmentdb.WaitlistOfferByIDRow(row), err)
}

func (r *postgresRepository) OpenOffer(ctx context.Context, slotID, entryID string) (Offer, bool, error) {
	var p appointmentdb.OpenWaitlistOfferParams
	if (slotID != "" && p.SlotID.Scan(slotID) != nil) || (entryID != "" && p.EntryID.Scan(entryID) != nil) {
		return Offer{}, false, nil
	}
	rows, err := r.q(ctx).OpenWaitlistOffer(ctx, p)
	if err != nil || len(rows) == 0 {
		return Offer{}, false, err
	}
	return toOffer(appointmentdb.WaitlistOfferByIDRow(rows[0])), true, nil
}

func (r *postgresRepository) CloseOffer(ctx context.Context, id string, state OfferState, appointmentID string, at time.Time) (Offer, error) {
	p := appointmentdb.CloseWaitlistOfferParams{State: string(state), At: ts(at)}
	if p.ID.Scan(id) != nil || (appointmentID != "" && p.AppointmentID.Scan(appointmentID) != nil) {
		return Offer{}, ErrNotFound
	}
	row, err := r.q(ctx).CloseWaitlistOffer(ctx, p)
	return offerOrNotFound(appointmentdb.WaitlistOfferByIDRow(row), err)
}

func (r *postgresRepository) DeleteEntriesClosedBefore(ctx context.Context, before time.Time) (int64, error) {
	return r.q(ctx).DeleteWaitlistEntriesClosedBefore(ctx, ts(before))
}

func (r *postgresRepository) DeleteOffersClosedBefore(ctx context.Context, before time.Time) (int64, error) {
	return r.q(ctx).DeleteWaitlistOffersClosedBefore(ctx, ts(before))
}

func entryOrNotFound(row appointmentdb.WaitlistEntryForUpdateRow, err error) (WaitlistEntry, error) {
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return WaitlistEntry{}, ErrNotFound
	case err != nil:
		return WaitlistEntry{}, err
	}
	return toEntry(row), nil
}

func offerOrNotFound(row appointmentdb.WaitlistOfferByIDRow, err error) (Offer, error) {
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return Offer{}, ErrNotFound
	case err != nil:
		return Offer{}, err
	}
	return toOffer(row), nil
}

// toEntry maps a row; every entry query selects the same columns.
func toEntry(row appointmentdb.WaitlistEntryForUpdateRow) WaitlistEntry {
	e := WaitlistEntry{
		Created: row.CreatedAt.Time, EndsAt: row.EndsAt.Time, ClosedAt: row.ClosedAt.Time,
		Earliest: row.Earliest.Time, Latest: row.Latest.Time,
		ID: row.ID.String(), PatientID: row.PatientID.String(), OrganizationID: row.OrganizationID.String(),
		Specialty: row.Specialty.String, Status: WaitlistStatus(row.Status), Priority: int(row.Priority),
	}
	if row.PractitionerRoleID.Valid {
		e.PractitionerRoleID = row.PractitionerRoleID.String()
	}
	if row.TimesStartMinute.Valid {
		e.Times = Times{Start: int(row.TimesStartMinute.Int16), End: int(row.TimesEndMinute.Int16)}
	}
	return e
}

// toEntryWithOffer maps a row of a list, with the entry's open offer when it has one.
func toEntryWithOffer(row appointmentdb.ListClinicWaitlistEntriesRow) WaitlistEntry {
	e := toEntry(appointmentdb.WaitlistEntryForUpdateRow{
		ID: row.ID, CreatedAt: row.CreatedAt, EndsAt: row.EndsAt, ClosedAt: row.ClosedAt, PatientID: row.PatientID,
		OrganizationID: row.OrganizationID, PractitionerRoleID: row.PractitionerRoleID, Earliest: row.Earliest, Latest: row.Latest,
		TimesStartMinute: row.TimesStartMinute, TimesEndMinute: row.TimesEndMinute, Priority: row.Priority, Status: row.Status, Specialty: row.Specialty,
	})
	if row.OfferID.Valid {
		e.Offer = Offer{
			ID: row.OfferID.String(), EntryID: e.ID, SlotID: row.OfferSlotID.String(), OrganizationID: e.OrganizationID, PatientID: e.PatientID,
			ExpiresAt: row.OfferExpiresAt.Time, Start: row.OfferStart.Time, End: row.OfferEnd.Time, State: OfferOpen,
		}
	}
	return e
}

// toOffer maps a row; every offer query selects the same columns.
func toOffer(row appointmentdb.WaitlistOfferByIDRow) Offer {
	o := Offer{
		Start: row.StartAt.Time, End: row.EndAt.Time, ExpiresAt: row.ExpiresAt.Time, Created: row.CreatedAt.Time,
		ID: row.ID.String(), EntryID: row.EntryID.String(), SlotID: row.SlotID.String(), OrganizationID: row.OrganizationID.String(),
		PatientID: row.PatientID.String(), PractitionerRoleID: row.PractitionerRoleID.String(), State: OfferState(row.State),
	}
	if row.AppointmentID.Valid {
		o.AppointmentID = row.AppointmentID.String()
	}
	return o
}

func pgDate(t time.Time) pgtype.Date {
	return pgtype.Date{Time: time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC), Valid: true}
}

// minute is v of t as a column, null for any time of day.
func minute(t Times, v int) pgtype.Int2 {
	return pgtype.Int2{Int16: int16(v), Valid: !t.Any()} //nolint:gosec // minutes of a day
}

// NewPostgresNoShowRepository returns the NoShowRepository on db (salus_app), in the transaction in the context when
// there is one (SAL-36).
func NewPostgresNoShowRepository(db postgres.Querier) NoShowRepository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) CorrectStatus(ctx context.Context, id string, from, to Status, at time.Time) (Appointment, error) {
	var u pgtype.UUID
	if u.Scan(id) != nil {
		return Appointment{}, ErrClosed
	}
	row, err := r.q(ctx).CorrectAppointmentStatus(ctx, appointmentdb.CorrectAppointmentStatusParams{
		ID: u, FromStatus: string(from), ToStatus: string(to), At: ts(at),
	})
	return changed(appointmentdb.AppointmentByIDRow(row), err)
}

func (r *postgresRepository) NoShowsSince(ctx context.Context, org string, patientIDs []string, since, until time.Time) (map[string]int, error) {
	var o pgtype.UUID
	if o.Scan(org) != nil {
		return map[string]int{}, nil
	}
	ids := make([]pgtype.UUID, 0, len(patientIDs))
	for _, p := range patientIDs {
		var u pgtype.UUID
		if u.Scan(p) == nil {
			ids = append(ids, u)
		}
	}
	rows, err := r.q(ctx).CountNoShowsSince(ctx, appointmentdb.CountNoShowsSinceParams{
		OrganizationID: o, PatientIds: ids, Since: ts(since), Until: ts(until),
	})
	if err != nil {
		return nil, err
	}
	out := make(map[string]int, len(rows))
	for _, row := range rows {
		out[row.PatientID.String()] = int(row.Noshows)
	}
	return out, nil
}
