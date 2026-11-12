package migrations_test

import (
	"context"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// chatRows is the seeded chat (docs/plans/chat.md "Contract"). Patient A: inA with clinic A (active consent), its
// doctor-only side thread staffA, and inB with clinic B (consent revoked). Patient B: other with clinic A (no
// consent). Each conversation has one message, named by the conversation, and one edit of it.
type chatRows struct {
	conv map[string]string // by name: inA, staffA, inB, other
	msg  map[string]string
	part map[string]string // the patient's participant row
	dir  map[string]directory
	p    [2]patientRow
}

// seedChat creates, as the owner, the conversations, a participant each and a message each, edited once.
func seedChat(t *testing.T, owner *pgxpool.Pool) chatRows {
	t.Helper()
	ctx := t.Context()
	p, dir, _ := seedAppointments(t, owner)
	r := chatRows{conv: map[string]string{}, msg: map[string]string{}, part: map[string]string{}, p: p, dir: dir}
	for _, c := range []struct {
		name, patient, account, clinic string
		staffOnly                      bool
	}{
		{"inA", p[0].patient, p[0].account, clinicA, false},
		{"staffA", p[0].patient, p[0].account, clinicA, true},
		{"inB", p[0].patient, p[0].account, clinicB, false},
		{"other", p[1].patient, p[1].account, clinicA, false},
	} {
		var id string
		err := owner.QueryRow(ctx, `INSERT INTO conversation (organization_id, patient_id, practitioner_role_id, staff_only, parent_id,
				created_by, last_seq, last_sent_at)
			VALUES ($1, $2, $3, $4, CASE WHEN $4 THEN NULLIF($6, '')::uuid END, $5, 1, '2026-11-12T10:00:00Z') RETURNING id`,
			c.clinic, c.patient, dir[c.clinic].role, c.staffOnly, c.account, r.conv["inA"]).Scan(&id)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		r.conv[c.name] = id
		var part string
		if err := owner.QueryRow(ctx, `INSERT INTO conversation_participant (conversation_id, patient_id, role, added_by)
			VALUES ($1, $2, 'patient', $3) RETURNING id`, id, c.patient, c.account).Scan(&part); err != nil {
			t.Fatal(err)
		}
		r.part[c.name] = part
		var msg string
		err = owner.QueryRow(ctx, `INSERT INTO communication (conversation_id, organization_id, patient_id, staff_only, seq, sent_at,
				sender_participant_id, sender_account_id, sender_role, client_message_id, kind, body)
			VALUES ($1, $2, $3, $4, 1, '2026-11-12T10:00:00Z', $6, $5, 'patient', gen_random_uuid(), 'text', 'Synthetic placeholder')
			RETURNING id`, id, c.clinic, c.patient, c.staffOnly, c.account, part).Scan(&msg)
		if err != nil {
			t.Fatal(err)
		}
		r.msg[c.name] = msg
		if _, err := owner.Exec(ctx, `UPDATE communication SET body = 'Synthetic placeholder, edited', version_id = 2,
			edited_at = '2026-11-12T10:01:00Z' WHERE id = $1`, msg); err != nil {
			t.Fatal(err)
		}
	}
	return r
}

func (r chatRows) convs(names ...string) []string { return r.pick(r.conv, names) }
func (r chatRows) msgs(names ...string) []string  { return r.pick(r.msg, names) }

func (r chatRows) pick(from map[string]string, names []string) []string {
	ids := make([]string, 0, len(names))
	for _, n := range names {
		ids = append(ids, from[n])
	}
	slices.Sort(ids)
	if len(ids) == 0 {
		return nil
	}
	return ids
}

// The scopes of docs/plans/chat.md "Row-level security": the patient their conversations but never a doctor-only
// one; a clinic its conversations while the care consent is active; a conversation's messages only to the
// patient or with that conversation set in a scope that sees it.
func TestChat_EachScopeSeesOnlyItsConversationsAndMessages(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	r := seedChat(t, owner)
	pA, pB := r.p[0].patient, r.p[1].patient

	tests := []struct {
		scope     map[string]string
		name      string
		convs     []string // conversations and their participants
		messages  []string // messages and their revisions
		staffOnly bool
	}{
		{name: "no scope"},
		{name: "patient A", scope: map[string]string{"app.patient_id": pA}, convs: r.convs("inA", "inB"), messages: r.msgs("inA", "inB")},
		{name: "patient B", scope: map[string]string{"app.patient_id": pB}, convs: r.convs("other"), messages: r.msgs("other")},
		{name: "patient A with their doctor-only thread set", scope: map[string]string{"app.patient_id": pA, "app.conversation_id": r.conv["staffA"]},
			convs: r.convs("inA", "inB"), messages: r.msgs("inA", "inB")},
		{name: "patient B with patient A's conversation set", scope: map[string]string{"app.patient_id": pB, "app.conversation_id": r.conv["inA"]},
			convs: r.convs("other"), messages: r.msgs("other")},
		{name: "clinic A: the consenting patient's conversations, no messages", scope: map[string]string{"app.organization_id": clinicA},
			convs: r.convs("inA", "staffA")},
		{name: "clinic A with its conversation set", scope: map[string]string{"app.organization_id": clinicA, "app.conversation_id": r.conv["inA"]},
			convs: r.convs("inA", "staffA"), messages: r.msgs("inA")},
		{name: "clinic A with its doctor-only thread set", scope: map[string]string{"app.organization_id": clinicA, "app.conversation_id": r.conv["staffA"]},
			convs: r.convs("inA", "staffA"), messages: r.msgs("staffA")},
		{name: "clinic A with a conversation without consent set", scope: map[string]string{"app.organization_id": clinicA, "app.conversation_id": r.conv["other"]},
			convs: r.convs("inA", "staffA")},
		{name: "clinic A with clinic B's conversation set", scope: map[string]string{"app.organization_id": clinicA, "app.conversation_id": r.conv["inB"]},
			convs: r.convs("inA", "staffA")},
		{name: "clinic B after the revocation", scope: map[string]string{"app.organization_id": clinicB, "app.conversation_id": r.conv["inB"]}},
		{name: "a conversation alone", scope: map[string]string{"app.conversation_id": r.conv["inA"]}},
		{name: "account A with clinic A set", scope: map[string]string{"app.account_id": r.p[0].account, "app.organization_id": clinicA, "app.conversation_id": r.conv["inA"]}},
	}
	queries := []struct {
		sql      string
		messages bool
	}{
		{sql: "SELECT id::text FROM conversation"},
		{sql: "SELECT conversation_id::text FROM conversation_participant"},
		{sql: "SELECT id::text FROM communication", messages: true},
		{sql: "SELECT communication_id::text FROM communication_revision", messages: true},
	}
	for _, tt := range tests {
		for _, q := range queries {
			want := tt.convs
			if q.messages {
				want = tt.messages
			}
			if got := idsIn(t, app, tt.scope, q.sql); !slices.Equal(got, want) {
				t.Errorf("%s, %s: sees %v, want %v", tt.name, q.sql, got, want)
			}
		}
	}
}

// insertMessage is the insert of the next message of the conversation, as the service makes it, with the
// conversation's values read by the owner (a scope that hides the conversation must still be refused).
func insertMessage(t *testing.T, owner *pgxpool.Pool, conv, sender string) func(ctx context.Context, db postgres.Querier) error {
	t.Helper()
	var org, patient, part string
	var staffOnly bool
	if err := owner.QueryRow(t.Context(), `SELECT c.organization_id::text, c.patient_id::text, c.staff_only, p.id::text
		FROM conversation c JOIN conversation_participant p ON p.conversation_id = c.id WHERE c.id = $1`,
		conv).Scan(&org, &patient, &staffOnly, &part); err != nil {
		t.Fatal(err)
	}
	return func(ctx context.Context, db postgres.Querier) error {
		_, err := db.Exec(ctx, `INSERT INTO communication (conversation_id, organization_id, patient_id, staff_only, seq, sent_at,
				sender_participant_id, sender_account_id, sender_role, client_message_id, kind, body)
			VALUES ($1, $2, $3, $4, 2, '2026-11-12T11:00:00Z', $6, $5, 'patient', gen_random_uuid(), 'text', 'Synthetic placeholder')`,
			conv, org, patient, staffOnly, sender, part)
		return err
	}
}

func TestChat_EachScopeWritesOnlyIntoTheConversationsItSees(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	r := seedChat(t, owner)
	pA, pB, accA, accB := r.p[0].patient, r.p[1].patient, r.p[0].account, r.p[1].account

	tests := []struct {
		scope   map[string]string
		name    string
		conv    string
		sender  string
		allowed bool
	}{
		{name: "patient A into their conversation", scope: map[string]string{"app.patient_id": pA}, conv: r.conv["inA"], sender: accA, allowed: true},
		{name: "patient A into their doctor-only thread", scope: map[string]string{"app.patient_id": pA}, conv: r.conv["staffA"], sender: accA},
		{name: "patient B into patient A's conversation", scope: map[string]string{"app.patient_id": pB}, conv: r.conv["inA"], sender: accB},
		{name: "clinic A into its conversation", scope: map[string]string{"app.organization_id": clinicA, "app.conversation_id": r.conv["inA"]},
			conv: r.conv["inA"], sender: accA, allowed: true},
		{name: "clinic A into a conversation other than the one set", scope: map[string]string{"app.organization_id": clinicA, "app.conversation_id": r.conv["inA"]},
			conv: r.conv["staffA"], sender: accA},
		{name: "clinic A into a conversation without consent", scope: map[string]string{"app.organization_id": clinicA, "app.conversation_id": r.conv["other"]},
			conv: r.conv["other"], sender: accB},
		{name: "clinic B after the revocation", scope: map[string]string{"app.organization_id": clinicB, "app.conversation_id": r.conv["inB"]},
			conv: r.conv["inB"], sender: accA},
	}
	for _, tt := range tests {
		err := asApp(t, app, tt.scope, insertMessage(t, owner, tt.conv, tt.sender))
		switch {
		case tt.allowed && err != nil:
			t.Errorf("%s: %v, want the insert", tt.name, err)
		case !tt.allowed && sqlState(err) != "42501":
			t.Errorf("%s: %v, want a row-level security violation (42501)", tt.name, err)
		}
	}
}

// Messages are never deleted (ADR 0012): the app role has no privilege, and the triggers refuse every role, the
// owner included. An edit changes only the body, its version and its time, and keeps the earlier body.
func TestChat_MessagesAreNeverDeletedAndAnEditKeepsTheEarlierBody(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	r := seedChat(t, owner)
	scope := map[string]string{"app.patient_id": r.p[0].patient}
	for _, stmt := range []string{
		"DELETE FROM communication", "TRUNCATE communication", "DELETE FROM communication_revision", "TRUNCATE communication_revision",
		"DELETE FROM conversation", "DELETE FROM conversation_participant",
		"UPDATE communication SET seq = seq", "UPDATE communication SET sent_at = sent_at", "UPDATE communication SET conversation_id = conversation_id",
		"UPDATE communication SET sender_account_id = sender_account_id", "UPDATE communication SET sender_participant_id = sender_participant_id", "UPDATE communication SET kind = kind",
		"UPDATE communication SET patient_id = patient_id", "UPDATE communication SET staff_only = staff_only",
		"UPDATE communication_revision SET body = body",
		"UPDATE conversation SET patient_id = patient_id", "UPDATE conversation SET organization_id = organization_id",
		"UPDATE conversation SET staff_only = staff_only", "UPDATE conversation_participant SET patient_id = patient_id", "UPDATE conversation_participant SET practitioner_role_id = practitioner_role_id",
		"UPDATE conversation_participant SET history_from_seq = 0",
	} {
		err := asApp(t, app, scope, func(ctx context.Context, db postgres.Querier) error {
			_, err := db.Exec(ctx, stmt)
			return err
		})
		if sqlState(err) != "42501" {
			t.Errorf("app: %s: %v, want insufficient_privilege (42501)", stmt, err)
		}
	}
	for _, stmt := range []string{
		"DELETE FROM communication", "DELETE FROM communication_revision", "TRUNCATE communication_revision",
		"DELETE FROM communication_y2026m11", "TRUNCATE communication CASCADE", "TRUNCATE communication_y2026m11 CASCADE",
	} {
		if _, err := owner.Exec(t.Context(), stmt); sqlState(err) != "P0001" {
			t.Errorf("owner: %s: %v, want the trigger's exception (P0001)", stmt, err)
		}
	}

	err := asApp(t, app, scope, func(ctx context.Context, db postgres.Querier) error {
		_, err := db.Exec(ctx, `UPDATE communication SET body = 'Synthetic placeholder, edited twice', version_id = version_id + 1,
			edited_at = '2026-11-12T10:02:00Z' WHERE id = $1`, r.msg["inA"])
		return err
	})
	if err != nil {
		t.Fatalf("the patient edits their message: %v", err)
	}
	rows, err := owner.Query(t.Context(), `SELECT version_id, body FROM communication_revision WHERE communication_id = $1 ORDER BY version_id`, r.msg["inA"])
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for rows.Next() {
		var v int
		var body string
		if err := rows.Scan(&v, &body); err != nil {
			t.Fatal(err)
		}
		got = append(got, body)
	}
	rows.Close()
	if want := []string{"Synthetic placeholder", "Synthetic placeholder, edited"}; !slices.Equal(got, want) {
		t.Errorf("revisions = %q, want %q", got, want)
	}
}

func TestChat_HoldsAConsistentState(t *testing.T) {
	t.Parallel()
	owner, _ := pgtest.NewAppDatabase(t)
	r := seedChat(t, owner)
	ctx := t.Context()
	for _, tt := range []struct {
		name, stmt string
		args       []any
	}{
		{"a message's clinic differs from its conversation's", `INSERT INTO communication (conversation_id, organization_id, patient_id, staff_only, seq, sent_at,
			sender_role, client_message_id, kind, body, sender_account_id, sender_participant_id) VALUES ($1, $2, $3, false, 2, now(), 'patient', gen_random_uuid(), 'text', 'x', $4, $5)`,
			[]any{r.conv["inA"], clinicB, r.p[0].patient, r.p[0].account, r.part["inA"]}},
		{"a message claims not to be doctor-only in a doctor-only thread", `INSERT INTO communication (conversation_id, organization_id, patient_id, staff_only, seq,
			sent_at, sender_role, client_message_id, kind, body, sender_account_id, sender_participant_id) VALUES ($1, $2, $3, false, 2, now(), 'practitioner', gen_random_uuid(), 'text', 'x', $4, $5)`,
			[]any{r.conv["staffA"], clinicA, r.p[0].patient, r.p[0].account, r.part["staffA"]}},
		{"an empty text", `INSERT INTO communication (conversation_id, organization_id, patient_id, staff_only, seq, sent_at,
			sender_role, client_message_id, kind, body, sender_account_id, sender_participant_id) VALUES ($1, $2, $3, false, 2, now(), 'patient', gen_random_uuid(), 'text', '', $4, $5)`,
			[]any{r.conv["inA"], clinicA, r.p[0].patient, r.p[0].account, r.part["inA"]}},
		{"a text without a sender", `INSERT INTO communication (conversation_id, organization_id, patient_id, staff_only, seq, sent_at,
			sender_role, client_message_id, kind, body) VALUES ($1, $2, $3, false, 2, now(), 'patient', gen_random_uuid(), 'text', 'x')`,
			[]any{r.conv["inA"], clinicA, r.p[0].patient}},
		{"a text without its participant", `INSERT INTO communication (conversation_id, organization_id, patient_id, staff_only, seq, sent_at,
			sender_role, client_message_id, kind, body, sender_account_id) VALUES ($1, $2, $3, false, 2, now(), 'patient', gen_random_uuid(), 'text', 'x', $4)`,
			[]any{r.conv["inA"], clinicA, r.p[0].patient, r.p[0].account}},
		{"a system message without its code", `INSERT INTO communication (conversation_id, organization_id, patient_id, staff_only, seq, sent_at,
			sender_role, client_message_id, kind) VALUES ($1, $2, $3, false, 2, now(), 'system', gen_random_uuid(), 'system')`,
			[]any{r.conv["inA"], clinicA, r.p[0].patient}},
		{"a message before its partitions", `INSERT INTO communication (conversation_id, organization_id, patient_id, staff_only, seq, sent_at,
			sender_role, client_message_id, kind, body, sender_account_id, sender_participant_id) VALUES ($1, $2, $3, false, 2, '2001-01-01', 'patient', gen_random_uuid(), 'text', 'x', $4, $5)`,
			[]any{r.conv["inA"], clinicA, r.p[0].patient, r.p[0].account, r.part["inA"]}},
		{"a second conversation for the same patient, role and no encounter", `INSERT INTO conversation (organization_id, patient_id, practitioner_role_id, created_by)
			VALUES ($1, $2, $3, $4)`, []any{clinicA, r.p[0].patient, r.dir[clinicA].role, r.p[0].account}},
		{"a doctor-only thread without the conversation it belongs to", `INSERT INTO conversation (organization_id, patient_id, practitioner_role_id, staff_only, created_by)
			VALUES ($1, $2, $3, true, $4)`, []any{clinicA, r.p[0].patient, r.dir[clinicA].role, r.p[0].account}},
		{"the patient twice in a conversation", `INSERT INTO conversation_participant (conversation_id, patient_id, role, added_by)
			VALUES ($1, $2, 'patient', $3)`, []any{r.conv["inA"], r.p[0].patient, r.p[0].account}},
		{"a practitioner twice in a conversation", `INSERT INTO conversation_participant (conversation_id, practitioner_role_id, role, added_by)
			VALUES ($1, $2, 'practitioner', $3), ($1, $2, 'practitioner', $3)`, []any{r.conv["inA"], r.dir[clinicA].role, r.p[0].account}},
		{"a practitioner without a role", `INSERT INTO conversation_participant (conversation_id, patient_id, role, added_by)
			VALUES ($1, $2, 'practitioner', $3)`, []any{r.conv["inA"], r.p[0].patient, r.p[0].account}},
		{"an edit of a message that skips its version", `UPDATE communication SET body = 'y', version_id = version_id + 2, edited_at = now() WHERE id = $1`,
			[]any{r.msg["inA"]}},
	} {
		if _, err := owner.Exec(ctx, tt.stmt, tt.args...); err == nil {
			t.Errorf("%s: accepted, want a refusal", tt.name)
		}
	}
}

// The partitions (ADR 0012): the app role adds months through the function, which is idempotent, and never reads a
// partition directly.
func TestChat_PartitionsAreAddedThroughTheFunctionOnly(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	ctx := t.Context()
	for range 2 {
		if _, err := app.Exec(ctx, `SELECT communication_add_partitions('2027-05-20', 3)`); err != nil {
			t.Fatalf("the app adds partitions: %v", err)
		}
	}
	var names []string
	rows, err := owner.Query(ctx, `SELECT c.relname FROM pg_inherits i JOIN pg_class c ON c.oid = i.inhrelid
		WHERE i.inhparent = 'communication'::regclass ORDER BY c.relname`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			t.Fatal(err)
		}
		names = append(names, n)
	}
	rows.Close()
	want := []string{"communication_y2026m11", "communication_y2026m12", "communication_y2027m01", "communication_y2027m02",
		"communication_y2027m03", "communication_y2027m04", "communication_y2027m05", "communication_y2027m06", "communication_y2027m07"}
	if !slices.Equal(names, want) {
		t.Errorf("partitions = %v, want %v", names, want)
	}
	if _, err := app.Exec(ctx, `SELECT count(*) FROM communication_y2026m11`); sqlState(err) != "42501" {
		t.Errorf("the app reads a partition directly: %v, want insufficient_privilege (42501)", err)
	}
	if _, err := app.Exec(ctx, `SELECT communication_add_partitions('2027-01-01', 0)`); err == nil {
		t.Error("zero months: accepted, want a refusal")
	}
}
