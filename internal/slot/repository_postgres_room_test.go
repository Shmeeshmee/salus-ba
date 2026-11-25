package slot_test

import (
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/schedule"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

const (
	roomSiteA = "a7000000-0000-4000-8000-0000000000a1"
	roomA     = "a7100000-0000-4000-8000-0000000000a1"
	roomB     = "a7100000-0000-4000-8000-0000000000a2"
)

// seedRooms adds, as the owner, clinic A's site with rooms A and B.
func seedRooms(t *testing.T, owner *pgxpool.Pool) {
	t.Helper()
	if _, err := owner.Exec(t.Context(), `INSERT INTO location (id, organization_id, name) VALUES ($1, $2, 'Synthetic site')`, roomSiteA, clinicA); err != nil {
		t.Fatal(err)
	}
	if _, err := owner.Exec(t.Context(), `INSERT INTO room (id, organization_id, location_id, name) VALUES ($1, $3, $4, 'Room A'), ($2, $3, $4, 'Room B')`,
		roomA, roomB, clinicA, roomSiteA); err != nil {
		t.Fatal(err)
	}
}

// roomsOf lists, as the owner, the room and location of the schedule's slots, "" for none, by start.
func roomsOf(t *testing.T, owner *pgxpool.Pool, scheduleID string) [][2]string {
	t.Helper()
	rows, err := owner.Query(t.Context(), `SELECT coalesce(room_id::text, ''), coalesce(location_id::text, '') FROM slot
		WHERE schedule_id = $1 ORDER BY lower(period)`, scheduleID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out [][2]string
	for rows.Next() {
		var r [2]string
		if err := rows.Scan(&r[0], &r[1]); err != nil {
			t.Fatal(err)
		}
		out = append(out, r)
	}
	return out
}

// An in-person rule's room goes on its slots with the room's location (U3, RM1); a rule moved to another room gives
// its free slots again in that room, and a rule without a room gives slots without one.
func TestGenerate_ARulesRoomIsOnItsSlots(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seed(t, owner)
	seedRooms(t, owner)
	inRoom := func(room string) []schedule.Schedule {
		s := seeded()
		s[0].Rules = []schedule.Rule{{Day: 1, Start: "10:00", End: "12:00", Mode: schedule.ModeInPerson, RoomID: room}}
		return s
	}
	if got, err := newService(app, monday, inRoom(roomA)...).Generate(t.Context(), clinicA, scheduleA1); err != nil || got.Inserted != 8 {
		t.Fatalf("Generate() = %+v, %v; want 8 inserted", got, err)
	}
	for _, r := range roomsOf(t, owner, scheduleA1) {
		if r != [2]string{roomA, roomSiteA} {
			t.Fatalf("slot room and location %v, want room A at its site", r)
		}
	}
	got, err := newService(app, monday, inRoom(roomB)...).Generate(t.Context(), clinicA, scheduleA1)
	if err != nil || got.Deleted != 8 || got.Inserted != 8 {
		t.Fatalf("moved to room B: %+v, %v; want 8 deleted and 8 inserted", got, err)
	}
	if r := roomsOf(t, owner, scheduleA1); len(r) != 8 || r[0] != [2]string{roomB, roomSiteA} {
		t.Errorf("slots %v, want 8 in room B", r)
	}
	if got, err := newService(app, monday, inRoom(roomB)...).Generate(t.Context(), clinicA, scheduleA1); err != nil || got.Deleted != 0 || got.Inserted != 0 {
		t.Errorf("a re-run: %+v, %v; want nothing changed", got, err)
	}
	if got, err := newService(app, monday, inRoom("")...).Generate(t.Context(), clinicA, scheduleA1); err != nil || got.Deleted != 8 || got.Inserted != 8 {
		t.Errorf("no room: %+v, %v; want 8 deleted and 8 inserted", got, err)
	}
	if r := roomsOf(t, owner, scheduleA1); r[0] != [2]string{"", ""} {
		t.Errorf("slots %v, want no room", r)
	}
}
