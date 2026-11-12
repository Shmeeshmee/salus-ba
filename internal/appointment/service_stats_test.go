package appointment_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// fakeStatsRepo answers Booked with its rows.
type fakeStatsRepo struct {
	err  error
	rows []appointment.Booked
}

func (f fakeStatsRepo) Booked(context.Context, string, time.Time, time.Time) ([]appointment.Booked, error) {
	return f.rows, f.err
}

// The aggregate builds get every appointment of a clinic with its patient as a pseudonym under the build's key, or
// none without a key (docs/plans/analytics.md §3).
func TestStatsStarts_PseudonymisesThePatient(t *testing.T) {
	at := time.Date(2026, 11, 2, 9, 0, 0, 0, time.UTC)
	stats := appointment.NewStats(fakeStatsRepo{rows: []appointment.Booked{
		{Start: at, PatientID: "p1", PractitionerRoleID: "r", Status: appointment.StatusFulfilled},
		{Start: at, PatientID: "p1", PractitionerRoleID: "r", Status: appointment.StatusBooked},
		{Start: at, PatientID: "p2", PractitionerRoleID: "r", Status: appointment.StatusNoShow},
	}})
	got, err := stats.Starts(context.Background(), "org", at, at, []byte("synthetic build key, 32 bytes ok"))
	if err != nil {
		t.Fatal(err)
	}
	var zero [32]byte
	if len(got) != 3 || got[0].Patient != got[1].Patient || got[0].Patient == got[2].Patient || got[0].Patient == zero ||
		got[2].Status != appointment.StatusNoShow || !got[0].Start.Equal(at) || got[0].PractitionerRoleID != "r" {
		t.Errorf("Starts = %+v", got)
	}
	if _, err := stats.Starts(context.Background(), "org", at, at, []byte{}); !errors.Is(err, appointment.ErrShortKey) {
		t.Errorf("an empty key: %v, want ErrShortKey", err)
	}
	none, _ := stats.Starts(context.Background(), "org", at, at, nil)
	if none[0].Patient != zero {
		t.Error("no key gives a pseudonym")
	}
	boom := errors.New("boom")
	if _, err := appointment.NewStats(fakeStatsRepo{err: boom}).Starts(context.Background(), "org", at, at, nil); !errors.Is(err, boom) {
		t.Errorf("Starts = %v", err)
	}
}

// salus_stats reads a clinic's appointments starting in the range, every status, oldest first; only that clinic's.
func TestStatsRepository_ReadsTheClinicsAppointmentsInTheRange(t *testing.T) {
	t.Parallel()
	owner, _ := pgtest.NewAppDatabase(t)
	s := seedDB(t, owner)
	at := time.Date(2026, 11, 2, 9, 0, 0, 0, time.UTC)
	for i, status := range []string{"fulfilled", "booked", "noshow", "booked"} {
		start := at.Add(time.Duration(i) * time.Hour)
		if i == 3 {
			start = at.Add(-time.Hour) // before the range
		}
		if _, err := owner.Exec(t.Context(), `INSERT INTO appointment (patient_id, practitioner_role_id, organization_id, slot_id, period, status)
			VALUES ($1, $2, $3, gen_random_uuid(), tstzrange($4::timestamptz, $4::timestamptz + interval '30 minutes'), $5)`,
			s.patientA, s.roleA1, clinicA, start, status); err != nil {
			t.Fatal(err)
		}
	}
	pool, err := postgres.Open(t.Context(), pgtest.StatsURL(t, owner.Config().ConnString()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	got, err := appointment.NewPostgresStatsRepository(pool).Booked(t.Context(), clinicA, at, at.Add(24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || got[0].Status != "fulfilled" || got[2].Status != "noshow" || got[0].PatientID != s.patientA ||
		got[0].PractitionerRoleID != s.roleA1 || !got[0].Start.Equal(at) {
		t.Errorf("Booked = %+v", got)
	}
	other, err := appointment.NewPostgresStatsRepository(pool).Booked(t.Context(), "00000000-0000-4000-8000-0000000000ff", at, at.Add(24*time.Hour))
	if err != nil || len(other) != 0 {
		t.Errorf("another clinic: %+v, %v", other, err)
	}
}
