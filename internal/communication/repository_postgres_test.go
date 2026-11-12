package communication_test

import (
	"errors"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/communication"
	"github.com/Shmeeshmee/salus/migrations"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

func TestMain(m *testing.M) { pgtest.Main(m, migrations.FS) }

// seedChat creates, as the owner, the fixture's world with its synthetic ids: clinic A with roles A1 and A2,
// patient A (active consent with clinic A) and patient B (none), and the callers' accounts.
func seedChat(t testing.TB, owner *pgxpool.Pool) {
	t.Helper()
	ctx := t.Context()
	for _, stmt := range []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO organization (id, name) VALUES ($1, 'Synthetic clinic')`, []any{clinicA}},
		{`INSERT INTO account (id, email, password_hash, role, organization_id) VALUES ($1, 'a@example.test', 'h', 'patient', NULL),
			($2, 'b@example.test', 'h', 'patient', NULL), ($3, 'p1@example.test', 'h', 'practitioner', $5), ($4, 'p2@example.test', 'h', 'practitioner', $5)`,
			[]any{me.AccountID, other.AccountID, pract.AccountID, pract2.AccountID, clinicA}},
		{`INSERT INTO patient (id, account_id, given_name, family_name, birth_date) VALUES ($1, $2, 'Ada', 'Example', '1990-01-01'),
			($3, $4, 'Bo', 'Example', '1991-01-01')`, []any{patientA, me.AccountID, patientB, other.AccountID}},
		{`WITH p AS (INSERT INTO practitioner (organization_id, given_name, family_name) VALUES ($1, 'Cy', 'Example'), ($1, 'Di', 'Example') RETURNING id)
			INSERT INTO practitioner_role (id, organization_id, practitioner_id)
			SELECT (ARRAY[$2::uuid, $3::uuid])[row_number() OVER ()], $1, id FROM p`, []any{clinicA, roleA1, roleA2}},
		{`INSERT INTO consent (patient_id, organization_id, status, date_time) VALUES ($1, $2, 'active', '2026-11-01T08:00:00Z')`, []any{patientA, clinicA}},
	} {
		if _, err := owner.Exec(ctx, stmt.sql, stmt.args...); err != nil {
			t.Fatal(err)
		}
	}
}

// pgFixture is the fixture's service on Postgres as the app role.
func pgFixture(t *testing.T) (*fixture, *pgxpool.Pool) {
	t.Helper()
	owner, app := pgtest.NewAppDatabase(t)
	seedChat(t, owner)
	return newFixtureOn(t, communication.NewPostgresRepository(app), postgres.NewTransactor(app)), owner
}

func TestPostgres_OpenStoresTheOneConversationOfTheCareWithItsParticipants(t *testing.T) {
	t.Parallel()
	f, _ := pgFixture(t)
	c, created, err := f.svc.Open(t.Context(), me, communication.Open{OrganizationID: clinicA, PractitionerRoleID: roleA1})
	if err != nil || !created {
		t.Fatalf("Open = %v, created %v", err, created)
	}
	if len(c.Participants) != 2 || c.Participants[0].Role != communication.RolePatient || c.Participants[0].PatientID != patientA ||
		c.Participants[1].PractitionerRoleID != roleA1 || !c.Created.Equal(start) {
		t.Errorf("conversation = %+v", c)
	}
	again, created, err := f.svc.Open(t.Context(), pract, communication.Open{PatientID: patientA})
	if err != nil || created || again.ID != c.ID || len(again.Participants) != 2 {
		t.Errorf("again = %v, created %v, %+v", err, created, again)
	}
	got, err := f.svc.Get(t.Context(), pract, c.ID)
	if err != nil || got.ID != c.ID {
		t.Errorf("Get = %v, %+v", err, got)
	}
	if _, err := f.svc.Get(t.Context(), other, c.ID); !errors.Is(err, communication.ErrNotFound) {
		t.Errorf("another patient: %v", err)
	}
}

// The conversation's row lock orders concurrent sends (ADR 0012): every message gets its own number, 1 to n,
// and a repeated client id is one message, also when the repeats race.
func TestPostgres_ConcurrentSendsAreNumberedWithoutGapsAndRepeatsStoreOneMessage(t *testing.T) {
	t.Parallel()
	f, owner := pgFixture(t)
	c := f.open(t)
	var wg sync.WaitGroup
	const n = 12
	for i := range n {
		wg.Go(func() {
			who := me
			if i%2 == 1 {
				who = pract
			}
			f.send(t, who, c.ID, "Synthetic placeholder")
		})
	}
	for range 4 {
		wg.Go(func() {
			if _, _, err := f.svc.Send(t.Context(), me, c.ID, clientID1, "Synthetic placeholder repeat"); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	var seqs []int64
	rows, err := owner.Query(t.Context(), `SELECT seq FROM communication WHERE conversation_id = $1 ORDER BY sent_at, seq`, c.ID)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var s int64
		if err := rows.Scan(&s); err != nil {
			t.Fatal(err)
		}
		seqs = append(seqs, s)
	}
	rows.Close()
	want := make([]int64, n+1)
	for i := range want {
		want[i] = int64(i + 1)
	}
	if !slices.Equal(seqs, want) {
		t.Errorf("numbers in time order = %v, want %v", seqs, want)
	}
	if got, _ := f.svc.Get(t.Context(), me, c.ID); got.LastSeq != n+1 {
		t.Errorf("last_seq = %d, want %d", got.LastSeq, n+1)
	}
}

// Pages cross the monthly partitions in order, newest first, and the replay goes oldest first.
func TestPostgres_PagesAndReplayCrossTheMonthsInOrder(t *testing.T) {
	t.Parallel()
	f, _ := pgFixture(t)
	c := f.open(t)
	at := time.Date(2026, 11, 30, 23, 59, 0, 0, time.UTC)
	var sent []communication.Message
	for i := range 6 {
		f.clock.set(at.Add(time.Duration(i) * 20 * time.Second)) // three in November, three in December
		sent = append(sent, f.send(t, me, c.ID, "Synthetic placeholder"))
	}
	page, err := f.svc.Messages(t.Context(), pract, c.ID, nil, 4)
	if err != nil || len(page.Messages) != 4 || page.Messages[0].Seq != 6 || page.Messages[3].Seq != 3 || page.Next == nil {
		t.Fatalf("first page = %v, %+v", err, page)
	}
	older, err := f.svc.Messages(t.Context(), pract, c.ID, page.Next, 4)
	if err != nil || len(older.Messages) != 2 || older.Messages[0].Seq != 2 || older.Next != nil {
		t.Errorf("older page = %v, %+v", err, older)
	}
	replay, err := f.svc.After(t.Context(), pract, c.ID, sent[1].Cursor())
	if err != nil || len(replay) != 4 || replay[0].Seq != 3 || replay[3].Seq != 6 {
		t.Errorf("replay = %v, %+v", err, replay)
	}
	if !replay[0].SentAt.Equal(sent[2].SentAt) || replay[0].Body != "Synthetic placeholder" || replay[0].SenderRole != communication.RolePatient {
		t.Errorf("replayed message = %+v, want %+v", replay[0], sent[2])
	}
}

func TestPostgres_AnEditKeepsTheEarlierBodyAndAStaleOneConflicts(t *testing.T) {
	t.Parallel()
	f, _ := pgFixture(t)
	c := f.open(t)
	m := f.send(t, pract, c.ID, "Synthetic placeholder")
	f.clock.set(start.Add(time.Minute))
	edited, err := f.svc.Edit(t.Context(), pract, c.ID, m.ID, "Synthetic placeholder, edited", 1)
	if err != nil || edited.VersionID != 2 {
		t.Fatalf("Edit = %v, %+v", err, edited)
	}
	if _, err := f.svc.Edit(t.Context(), pract, c.ID, m.ID, "Synthetic placeholder, stale", 1); !errors.Is(err, communication.ErrVersionConflict) {
		t.Errorf("stale edit: %v", err)
	}
	revs, err := f.svc.Revisions(t.Context(), me, c.ID, m.ID)
	if err != nil || len(revs) != 1 || revs[0].Body != "Synthetic placeholder" || !revs[0].EditedAt.Equal(start) {
		t.Errorf("Revisions = %v, %+v", err, revs)
	}
	page, err := f.svc.Messages(t.Context(), me, c.ID, nil, 0)
	if err != nil || page.Messages[0].Body != "Synthetic placeholder, edited" || !page.Messages[0].EditedAt.Equal(start.Add(time.Minute)) {
		t.Errorf("the page shows %+v", page.Messages)
	}
}

func TestPostgres_ListAndReadPosition(t *testing.T) {
	t.Parallel()
	f, _ := pgFixture(t)
	c := f.open(t)
	for range 3 {
		f.send(t, pract, c.ID, "Synthetic placeholder")
	}
	if err := f.svc.MarkRead(t.Context(), me, c.ID, 2); err != nil {
		t.Fatal(err)
	}
	if err := f.svc.MarkRead(t.Context(), me, c.ID, 1); err != nil {
		t.Fatal(err)
	}
	mine, next, err := f.svc.List(t.Context(), me, nil)
	if err != nil || len(mine) != 1 || mine[0].Unread != 1 || mine[0].LastSeq != 3 || next != nil {
		t.Errorf("patient's list = %v, %+v", err, mine)
	}
	theirs, _, err := f.svc.List(t.Context(), pract, &communication.ListCursor{At: start.Add(time.Hour), ID: unknownID})
	if err != nil || len(theirs) != 1 || theirs[0].Unread != 0 {
		t.Errorf("practitioner's list = %v, %+v", err, theirs)
	}
	if none, _, err := f.svc.List(t.Context(), pract, &communication.ListCursor{At: start.Add(-time.Hour), ID: unknownID}); err != nil || len(none) != 0 {
		t.Errorf("before every conversation = %v, %+v", err, none)
	}
	if none, _, err := f.svc.List(t.Context(), pract2, nil); err != nil || len(none) != 0 {
		t.Errorf("another practitioner = %v, %+v", err, none)
	}
}

// After the revocation the clinic's row-level security hides the conversation; the patient keeps reading.
func TestPostgres_ARevocationHidesTheConversationFromTheClinic(t *testing.T) {
	t.Parallel()
	f, owner := pgFixture(t)
	c := f.open(t)
	f.send(t, pract, c.ID, "Synthetic placeholder")
	if _, err := owner.Exec(t.Context(), `UPDATE consent SET status = 'inactive', revoked_at = now() WHERE patient_id = $1`, patientA); err != nil {
		t.Fatal(err)
	}
	f.revoke()
	if _, err := f.svc.Messages(t.Context(), pract, c.ID, nil, 0); !errors.Is(err, communication.ErrNotFound) {
		t.Errorf("the practitioner: %v", err)
	}
	if page, err := f.svc.Messages(t.Context(), me, c.ID, nil, 0); err != nil || len(page.Messages) != 1 {
		t.Errorf("the patient: %v, %+v", err, page)
	}
}

func TestPostgres_AddPartitionsCreatesTheMissingMonthsAsTheAppRole(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	repo := communication.NewPostgresRepository(app)
	for range 2 {
		if err := repo.AddPartitions(t.Context(), time.Date(2027, 6, 15, 0, 0, 0, 0, time.UTC), 3); err != nil {
			t.Fatal(err)
		}
	}
	var n int
	if err := owner.QueryRow(t.Context(), `SELECT count(*) FROM pg_inherits WHERE inhparent = 'communication'::regclass
		AND inhrelid::regclass::text IN ('communication_y2027m06', 'communication_y2027m07', 'communication_y2027m08')`).Scan(&n); err != nil || n != 3 {
		t.Errorf("June to August: %d partitions, %v", n, err)
	}
}
