package schedule_test

import (
	"context"
	"slices"
	"testing"

	"github.com/Shmeeshmee/salus/internal/schedule"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// A rule's room is stored under room_id (00080's key) and read back; a rule without one stores none.
func TestPostgres_KeepsARulesRoom(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seed(t, owner)
	repo := schedule.NewPostgresRepository(app)
	clinic := func(ctx context.Context) error { return repo.SetOrganization(ctx, clinicA) }
	rules := []schedule.Rule{{Day: 1, Start: "09:00", End: "12:00", Mode: schedule.ModeInPerson, RoomID: room1}, {Day: 2, Start: "09:00", End: "10:00", Mode: schedule.ModeVideo}}
	var got schedule.Schedule
	err := in(t, app, clinic, func(ctx context.Context) error {
		created, err := repo.Create(ctx, schedule.Schedule{OrganizationID: clinicA, PractitionerRoleID: roleA1, SlotMinutes: 30, Rules: rules})
		if err != nil {
			return err
		}
		got, err = repo.ByID(ctx, created.ID)
		return err
	})
	if err != nil || !slices.Equal(got.Rules, rules) {
		t.Fatalf("ByID() = %+v, %v; want the rules with room 1", got.Rules, err)
	}
	var stored string
	if err := owner.QueryRow(t.Context(), `SELECT availability::text FROM schedule WHERE id = $1`, got.ID).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	want := `[{"day": 1, "end": "12:00", "mode": "in-person", "start": "09:00", "room_id": "` + room1 + `"}, {"day": 2, "end": "10:00", "mode": "video", "start": "09:00"}]`
	if stored != want {
		t.Errorf("stored %s, want %s", stored, want)
	}
}
