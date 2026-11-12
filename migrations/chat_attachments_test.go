package migrations_test

import (
	"context"
	"fmt"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// chatFileRows is seedChat (chat_test.go) with what migration 00045 adds (docs/plans/chat.md, SAL-30): a practitioner
// participant in each conversation, of the conversation's clinic, who joined inA after its first message
// (history_from_seq 1); a second message in inA; one file per conversation attached to its first message, uploaded
// by the patient (in staffA by the practitioner); and one file in inA not attached yet.
type chatFileRows struct {
	chatRows
	pract      map[string]string // the practitioner's participant row, by conversation
	file       map[string]string // the attached file, by conversation
	second     string            // inA's second message
	unattached string            // inA's file not attached yet
}

// insertChatFile is a synthetic file of a conversation, uploaded by the participant, as binary's service writes it.
const insertChatFile = `INSERT INTO "binary" (organization_id, patient_id, conversation_id, conversation_participant_id, content_type,
		size, title, sha1, kek_id, wrapped_key, segment_size)
	VALUES ($1, $2, $3, $4::uuid, 'image/png', 1024, 'placeholder.png', digest($4::text || gen_random_uuid()::text, 'sha1'), 'local-1',
		gen_random_bytes(60), 65536)
	RETURNING id`

func seedChatFiles(t *testing.T, owner *pgxpool.Pool) chatFileRows {
	t.Helper()
	ctx := t.Context()
	r := chatFileRows{chatRows: seedChat(t, owner), pract: map[string]string{}, file: map[string]string{}}
	for name, clinic := range map[string]string{"inA": clinicA, "staffA": clinicA, "inB": clinicB, "other": clinicA} {
		history := 0
		if name == "inA" {
			history = 1
		}
		var part string
		if err := owner.QueryRow(ctx, `INSERT INTO conversation_participant (conversation_id, practitioner_role_id, organization_id, role, added_by,
			history_from_seq) VALUES ($1, $2, $3, 'practitioner', $4, $5) RETURNING id`, r.conv[name], r.dir[clinic].role, clinic, r.p[0].account,
			history).Scan(&part); err != nil {
			t.Fatal(err)
		}
		r.pract[name] = part
	}
	if err := owner.QueryRow(ctx, `INSERT INTO communication (conversation_id, organization_id, patient_id, staff_only, seq, sent_at,
			sender_participant_id, sender_account_id, sender_role, client_message_id, kind, body)
		VALUES ($1, $2, $3, false, 2, '2026-11-12T10:05:00Z', $4, $5, 'patient', gen_random_uuid(), 'text', 'Synthetic placeholder two')
		RETURNING id`, r.conv["inA"], clinicA, r.p[0].patient, r.part["inA"], r.p[0].account).Scan(&r.second); err != nil {
		t.Fatal(err)
	}
	for name, c := range map[string]struct{ clinic, patient string }{
		"inA": {clinicA, r.p[0].patient}, "staffA": {clinicA, r.p[0].patient}, "inB": {clinicB, r.p[0].patient}, "other": {clinicA, r.p[1].patient},
	} {
		uploader := r.part[name]
		if name == "staffA" {
			uploader = r.pract[name]
		}
		var id string
		if err := owner.QueryRow(ctx, insertChatFile, c.clinic, c.patient, r.conv[name], uploader).Scan(&id); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if _, err := owner.Exec(ctx, `UPDATE "binary" SET communication_id = $2, communication_sent_at = '2026-11-12T10:00:00Z' WHERE id = $1`,
			id, r.msg[name]); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		r.file[name] = id
	}
	if err := owner.QueryRow(ctx, insertChatFile, clinicA, r.p[0].patient, r.conv["inA"], r.part["inA"]).Scan(&r.unattached); err != nil {
		t.Fatal(err)
	}
	return r
}

func (r chatFileRows) files(names ...string) []string { return r.pick(r.file, names) }

// chatRole is the scope of a practitioner's search across their conversations, as the service sets it: their clinic,
// their role (00044's app.practitioner_role_id) and the search's role.
func chatRole(clinic, role string) map[string]string {
	return map[string]string{"app.organization_id": clinic, "app.practitioner_role_id": role, "app.chat_practitioner_role_id": role}
}

// A conversation's files are seen exactly where its messages are (00041): by its patient, never in a doctor-only
// thread, and before it is sent only their own upload; with the conversation set in a scope that sees it; by the
// practitioner role's scope in the conversations it takes part in, while the care consent is active, those of the
// messages after its history_from_seq and its own uploads not yet sent (SAL-30 review, finding 6); by the scan job's
// scope of the conversation. A clinic without a conversation set sees none of them (its questionnaire files' policy
// needs a response).
func TestChatFiles_EachScopeSeesTheFilesOfTheConversationsItSees(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	r := seedChatFiles(t, owner)
	pA, pB := r.p[0].patient, r.p[1].patient
	// In inA, besides the file of its first message (before the practitioner's history_from_seq) and the patient's
	// upload not sent: a file of its second message, and the practitioner's upload not sent.
	var second, practUnsent string
	if err := owner.QueryRow(t.Context(), insertChatFile, clinicA, pA, r.conv["inA"], r.part["inA"]).Scan(&second); err != nil {
		t.Fatal(err)
	}
	if _, err := owner.Exec(t.Context(), `UPDATE "binary" SET communication_id = $2, communication_sent_at = '2026-11-12T10:05:00Z' WHERE id = $1`,
		second, r.second); err != nil {
		t.Fatal(err)
	}
	if err := owner.QueryRow(t.Context(), insertChatFile, clinicA, pA, r.conv["inA"], r.pract["inA"]).Scan(&practUnsent); err != nil {
		t.Fatal(err)
	}
	inA := sorted(r.file["inA"], second, r.unattached, practUnsent)
	both := sorted(r.file["inA"], second, r.unattached, r.file["inB"])
	roleA := sorted(second, practUnsent, r.file["staffA"])

	for _, tt := range []struct {
		scope map[string]string
		name  string
		want  []string
	}{
		{name: "no scope"},
		{name: "patient A", scope: map[string]string{"app.patient_id": pA}, want: both},
		{name: "patient A with their doctor-only thread set", scope: map[string]string{"app.patient_id": pA, "app.conversation_id": r.conv["staffA"]}, want: both},
		{name: "patient B", scope: map[string]string{"app.patient_id": pB}, want: r.files("other")},
		{name: "clinic A", scope: map[string]string{"app.organization_id": clinicA}},
		{name: "clinic A with its conversation set", scope: map[string]string{"app.organization_id": clinicA, "app.conversation_id": r.conv["inA"]}, want: inA},
		{name: "clinic A with its doctor-only thread set", scope: map[string]string{"app.organization_id": clinicA, "app.conversation_id": r.conv["staffA"]},
			want: r.files("staffA")},
		{name: "clinic A with a conversation without consent set", scope: map[string]string{"app.organization_id": clinicA, "app.conversation_id": r.conv["other"]}},
		{name: "clinic B after the revocation", scope: map[string]string{"app.organization_id": clinicB, "app.conversation_id": r.conv["inB"]}},
		{name: "role A of clinic A", scope: chatRole(clinicA, r.dir[clinicA].role), want: roleA},
		{name: "role B of clinic B after the revocation", scope: chatRole(clinicB, r.dir[clinicB].role)},
		{name: "role A without its clinic", scope: map[string]string{"app.chat_practitioner_role_id": r.dir[clinicA].role}},
		{name: "role A with an account set", scope: map[string]string{"app.organization_id": clinicA, "app.chat_practitioner_role_id": r.dir[clinicA].role,
			"app.account_id": r.p[0].account}},
		{name: "role A with patient A set", scope: map[string]string{"app.patient_id": pA, "app.chat_practitioner_role_id": r.dir[clinicA].role}, want: both},
		{name: "the scan job of inA", scope: map[string]string{"app.organization_id": clinicA, "app.binary_conversation_id": r.conv["inA"]}, want: inA},
		{name: "the scan job of inA in another clinic", scope: map[string]string{"app.organization_id": clinicB, "app.binary_conversation_id": r.conv["inA"]}},
		{name: "the scan job of inB after the revocation", scope: map[string]string{"app.organization_id": clinicB, "app.binary_conversation_id": r.conv["inB"]},
			want: r.files("inB")},
		{name: "a conversation alone", scope: map[string]string{"app.conversation_id": r.conv["inA"]}},
		{name: "account A with clinic A and inA set", scope: map[string]string{"app.account_id": r.p[0].account, "app.organization_id": clinicA,
			"app.conversation_id": r.conv["inA"]}},
	} {
		if got := idsIn(t, app, tt.scope, `SELECT id::text FROM "binary"`); !slices.Equal(got, tt.want) {
			t.Errorf("%s: sees %v, want %v", tt.name, got, tt.want)
		}
	}
}

// The practitioner role's scope (search across a doctor's conversations, SAL-30) shows the messages of the
// conversations the role takes part in, of numbers after its history_from_seq, while the care consent is active;
// nothing once it is removed; nothing to a patient's scope it does not belong to.
func TestChatSearch_ThePractitionerRoleScopeSeesOnlyTheMessagesItMayRead(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	r := seedChatFiles(t, owner)
	roleA := chatRole(clinicA, r.dir[clinicA].role)
	query := `SELECT id::text FROM communication`
	if got, want := idsIn(t, app, roleA, query), sorted(r.second, r.msg["staffA"]); !slices.Equal(got, want) {
		t.Errorf("role A: sees %v, want inA's second message and staffA's (%v)", got, want)
	}
	if got := idsIn(t, app, chatRole(clinicB, r.dir[clinicB].role), query); len(got) != 0 {
		t.Errorf("role B after the revocation: sees %v, want none", got)
	}
	if got := idsIn(t, app, chatRole(clinicB, r.dir[clinicA].role), query); len(got) != 0 {
		t.Errorf("role A in clinic B: sees %v, want none", got)
	}
	if got := idsIn(t, app, map[string]string{"app.chat_practitioner_role_id": r.dir[clinicA].role}, query); len(got) != 0 {
		t.Errorf("role A without its clinic: sees %v, want none", got)
	}
	if _, err := owner.Exec(t.Context(), `UPDATE conversation_participant SET removed_at = now() WHERE id = $1`, r.pract["staffA"]); err != nil {
		t.Fatal(err)
	}
	if got, want := idsIn(t, app, roleA, query), sorted(r.second); !slices.Equal(got, want) {
		t.Errorf("role A once removed from staffA: sees %v, want %v", got, want)
	}
	// The scope only reads: a send or an edit needs its conversation set.
	err := asApp(t, app, roleA, insertMessage(t, owner, r.conv["inA"], r.p[0].account))
	if sqlState(err) != "42501" {
		t.Errorf("role A inserts a message: %v, want a row-level security violation (42501)", err)
	}
	if n, err := exec(t, app, roleA, `UPDATE communication SET body = 'x', version_id = version_id + 1, edited_at = now()`); err != nil || n != 0 {
		t.Errorf("role A edits: %d rows, %v; want none", n, err)
	}
}

// A file is inserted only into a conversation the scope sees, with that conversation set, for one of its active
// participants, with the conversation's clinic and patient; it is attached to a message once, by its conversation's
// scope; an attached file is never deleted (but a rejected one, by the purger), and the patient deletes none.
func TestChatFiles_AreWrittenOnlyIntoTheirConversationAndAttachedOnce(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	r := seedChatFiles(t, owner)
	pA, pB := r.p[0].patient, r.p[1].patient
	inA := map[string]string{"app.organization_id": clinicA, "app.conversation_id": r.conv["inA"]}
	for _, tt := range []struct {
		scope                     map[string]string
		name                      string
		clinic, patient, conv, by string
		allowed                   bool
	}{
		{name: "patient A into inA", scope: map[string]string{"app.patient_id": pA, "app.conversation_id": r.conv["inA"]},
			clinic: clinicA, patient: pA, conv: r.conv["inA"], by: r.part["inA"], allowed: true},
		{name: "patient A into inA without it set", scope: map[string]string{"app.patient_id": pA}, clinic: clinicA, patient: pA, conv: r.conv["inA"], by: r.part["inA"]},
		{name: "patient A into their doctor-only thread", scope: map[string]string{"app.patient_id": pA, "app.conversation_id": r.conv["staffA"]},
			clinic: clinicA, patient: pA, conv: r.conv["staffA"], by: r.part["staffA"]},
		{name: "patient B into inA", scope: map[string]string{"app.patient_id": pB, "app.conversation_id": r.conv["inA"]},
			clinic: clinicA, patient: pA, conv: r.conv["inA"], by: r.part["inA"]},
		{name: "clinic A into inA", scope: inA, clinic: clinicA, patient: pA, conv: r.conv["inA"], by: r.pract["inA"], allowed: true},
		{name: "clinic A into inA for another conversation's participant", scope: inA, clinic: clinicA, patient: pA, conv: r.conv["inA"], by: r.pract["staffA"]},
		{name: "clinic A into inA with another patient", scope: inA, clinic: clinicA, patient: pB, conv: r.conv["inA"], by: r.pract["inA"]},
		{name: "clinic A into inA with another clinic", scope: inA, clinic: clinicB, patient: pA, conv: r.conv["inA"], by: r.pract["inA"]},
		{name: "clinic A into a conversation without consent", scope: map[string]string{"app.organization_id": clinicA, "app.conversation_id": r.conv["other"]},
			clinic: clinicA, patient: pB, conv: r.conv["other"], by: r.pract["other"]},
		{name: "the role scope into inA", scope: chatRole(clinicA, r.dir[clinicA].role), clinic: clinicA, patient: pA, conv: r.conv["inA"], by: r.pract["inA"]},
	} {
		err := asApp(t, app, tt.scope, func(ctx context.Context, db postgres.Querier) error {
			var id string
			return db.QueryRow(ctx, insertChatFile, tt.clinic, tt.patient, tt.conv, tt.by).Scan(&id)
		})
		switch {
		case tt.allowed && err != nil:
			t.Errorf("%s: %v, want the insert", tt.name, err)
		case !tt.allowed && sqlState(err) != "42501":
			t.Errorf("%s: %v, want a row-level security violation (42501)", tt.name, err)
		}
	}

	link := `UPDATE "binary" SET communication_id = $2, communication_sent_at = '2026-11-12T10:05:00Z', updated_at = now() WHERE id = $1`
	if n, err := exec(t, app, inA, link, r.unattached, r.second); err != nil || n != 1 {
		t.Errorf("clinic A attaches inA's file: %d rows, %v; want 1", n, err)
	}
	if n, err := exec(t, app, map[string]string{"app.organization_id": clinicA, "app.conversation_id": r.conv["staffA"]}, link, r.unattached, r.msg["staffA"]); err != nil || n != 0 {
		t.Errorf("another conversation's scope attaches inA's file: %d rows, %v; want none", n, err)
	}
	if _, err := exec(t, app, inA, link, r.file["inA"], r.second); sqlState(err) != "P0001" {
		t.Errorf("an attached file attached again: %v, want the trigger's exception (P0001)", err)
	}
	if n, err := exec(t, app, map[string]string{"app.patient_id": pA, "app.conversation_id": r.conv["inA"]}, `DELETE FROM "binary"`); err != nil || n != 0 {
		t.Errorf("patient A deletes their conversation's files: %d rows, %v; want none", n, err)
	}
	if err := tryAsOwner(t, owner, `DELETE FROM "binary" WHERE communication_id IS NOT NULL`); sqlState(err) != "P0001" {
		t.Errorf("the owner deletes an attached file: %v, want the trigger's exception (P0001)", err)
	}
	for _, stmt := range []string{
		`DELETE FROM "binary" WHERE conversation_id IS NOT NULL AND communication_id IS NULL`,
		`UPDATE "binary" SET status = 'rejected', scanned_at = now(), wrapped_key = NULL WHERE conversation_id IS NOT NULL;
		 DELETE FROM "binary" WHERE conversation_id IS NOT NULL`,
	} {
		if err := tryAsOwner(t, owner, stmt); err != nil {
			t.Errorf("%s: %v, want it done", stmt, err)
		}
	}
	for _, stmt := range []string{
		`UPDATE "binary" SET conversation_id = conversation_id`, `UPDATE "binary" SET conversation_participant_id = conversation_participant_id`,
	} {
		if _, err := exec(t, app, inA, stmt); sqlState(err) != "42501" {
			t.Errorf("%s: %v, want insufficient_privilege (42501)", stmt, err)
		}
	}
}

// The checks of migration 00045: a file has exactly one kind of parent; a chat file is a photo or a PDF; a message
// it is attached to is named with its time; a thumbnail only of a clean file, whole. A text may be without a body
// only with attachments, a system message has none.
func TestChatFiles_HoldAConsistentState(t *testing.T) {
	t.Parallel()
	owner, _ := pgtest.NewAppDatabase(t)
	r := seedChatFiles(t, owner)
	file := `UPDATE "binary" SET %s WHERE id = '` + r.file["inA"] + `'`
	set := func(s string) string { return fmt.Sprintf(file, s) }
	unattached := func(s string) string { return fmt.Sprintf(`UPDATE "binary" SET %s WHERE id = '`+r.unattached+`'`, s) }
	text := func(body string, attachments int) string {
		return fmt.Sprintf(`INSERT INTO communication (conversation_id, organization_id, patient_id, staff_only, seq, sent_at, sender_participant_id,
			sender_account_id, sender_role, client_message_id, kind, body, attachment_count) VALUES ('%s', '%s', '%s', false, 3, '2026-11-12T11:00:00Z',
			'%s', '%s', 'patient', gen_random_uuid(), 'text', %s, %d)`, r.conv["inA"], clinicA, r.p[0].patient, r.part["inA"], r.p[0].account, body, attachments)
	}
	for stmt, want := range map[string]string{
		set(`appointment_id = gen_random_uuid()`):                    "23514",
		set(`questionnaire_response_id = gen_random_uuid()`):         "23514",
		set(`link_id = 'f1'`):                                        "23514",
		set(`conversation_participant_id = NULL`):                    "23514",
		set(`content_type = 'video/mp4'`):                            "23514",
		set(`thumbnail_id = gen_random_uuid(), thumbnail_size = 10`): "23514", // no key
		set(`thumbnail_id = gen_random_uuid(), thumbnail_size = 10, thumbnail_wrapped_key = gen_random_bytes(60)`):                                       "23514", // quarantined
		set(`status = 'clean', scanned_at = now(), thumbnail_id = gen_random_uuid(), thumbnail_size = 10, thumbnail_wrapped_key = gen_random_bytes(60)`): "",
		set(`content_type = 'application/pdf'`):                                                             "",
		unattached(`communication_id = '` + r.second + `'`):                                                 "23514",
		unattached(`communication_id = '` + r.second + `', communication_sent_at = '2026-11-12T10:05:01Z'`): "23503",
		unattached(`communication_id = '` + r.second + `', communication_sent_at = '2026-11-12T10:05:00Z'`): "",
		text("NULL", 1): "",
		text("'x'", 2):  "",
		text("NULL", 0): "23514",
		text("''", 1):   "23514",
		text("'x'", 11): "23514",
	} {
		if err := tryAsOwner(t, owner, stmt); sqlState(err) != want {
			t.Errorf("%s: %v, want SQLSTATE %q", stmt, err, want)
		}
	}
	if _, err := owner.Exec(t.Context(), `INSERT INTO communication (conversation_id, organization_id, patient_id, staff_only, seq, sent_at,
			sender_role, client_message_id, kind, system_code, attachment_count) VALUES ($1, $2, $3, false, 3, now(), 'system', gen_random_uuid(),
			'system', 'participant_added', 1)`, r.conv["inA"], clinicA, r.p[0].patient); sqlState(err) != "23514" {
		t.Errorf("a system message with an attachment: %v, want a check violation (23514)", err)
	}
}

// Search (SAL-30): one text search configuration for every language of the app, lower case and accents aside, so a
// word typed without its diacritics finds it, in Latin and Cyrillic; the index is on every partition, those created
// later too.
func TestChatSearch_MatchesWordsCaseAndAccentsAsideOnEveryPartition(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	ctx := t.Context()
	for _, tt := range []struct {
		body, query string
		match       bool
	}{
		{"Glavobolja i mučnina od jutros", "mucnina:*", true},
		{"Glavobolja i mučnina od jutros", "'GLAVOBOL':*", true},
		{"Главобоља и мучнина", "мучнина:*", true},
		{"Kopfschmerzen seit gestern", "kopfschmerzen", true},
		{"Kopfschmerzen seit gestern", "fieber", false},
	} {
		var got bool
		if err := app.QueryRow(ctx, `SELECT to_tsvector('public.chat_search'::regconfig, $1) @@ to_tsquery('public.chat_search'::regconfig, $2)`,
			tt.body, tt.query).Scan(&got); err != nil || got != tt.match {
			t.Errorf("%q @@ %q = %v, %v; want %v", tt.body, tt.query, got, err, tt.match)
		}
	}
	if _, err := app.Exec(ctx, `SELECT communication_add_partitions('2027-05-01', 1)`); err != nil {
		t.Fatal(err)
	}
	var missing []string
	rows, err := owner.Query(ctx, `SELECT c.relname FROM pg_inherits i JOIN pg_class c ON c.oid = i.inhrelid
		WHERE i.inhparent = 'communication'::regclass
		  AND NOT EXISTS (SELECT 1 FROM pg_indexes x WHERE x.tablename = c.relname AND x.indexdef LIKE '%USING gin%chat_search%')
		ORDER BY 1`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var n string
		_ = rows.Scan(&n)
		missing = append(missing, n)
	}
	rows.Close()
	if len(missing) > 0 {
		t.Errorf("partitions without the search index: %v", missing)
	}
}

// The search function (SAL-30) finds through the text index, which row-level security cannot use (the match operator
// is not leakproof), so it decides what a scope may read itself, as the policies do: the patient their conversations
// but no doctor-only one; the practitioner role, with its clinic, the conversations it takes part in after its
// history_from_seq while the consent is active; nothing else. It returns keys only; the app reads the rows through the
// policies.
func TestChatSearch_TheSearchFunctionFindsOnlyWhatTheScopeMayRead(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	r := seedChatFiles(t, owner)
	pA, pB := r.p[0].patient, r.p[1].patient
	search := `SELECT id::text FROM communication_search(to_tsquery('public.chat_search', 'synth:*'), NULLIF($1, '')::uuid, 50)`
	for _, tt := range []struct {
		scope map[string]string
		name  string
		only  string
		want  []string
	}{
		{name: "no scope"},
		{name: "patient A", scope: map[string]string{"app.patient_id": pA}, want: sorted(r.msg["inA"], r.second, r.msg["inB"])},
		{name: "patient A in inA", scope: map[string]string{"app.patient_id": pA}, only: r.conv["inA"], want: sorted(r.msg["inA"], r.second)},
		{name: "patient A in their doctor-only thread", scope: map[string]string{"app.patient_id": pA}, only: r.conv["staffA"]},
		{name: "patient B", scope: map[string]string{"app.patient_id": pB}, want: r.msgs("other")},
		{name: "role A of clinic A", scope: chatRole(clinicA, r.dir[clinicA].role), want: sorted(r.second, r.msg["staffA"])},
		{name: "role A in staffA", scope: chatRole(clinicA, r.dir[clinicA].role), only: r.conv["staffA"], want: r.msgs("staffA")},
		{name: "role B after the revocation", scope: chatRole(clinicB, r.dir[clinicB].role)},
		{name: "role A in clinic B", scope: chatRole(clinicB, r.dir[clinicA].role)},
		{name: "role A without its clinic", scope: map[string]string{"app.chat_practitioner_role_id": r.dir[clinicA].role}},
		{name: "role A with an account set", scope: map[string]string{"app.account_id": r.p[0].account, "app.organization_id": clinicA,
			"app.chat_practitioner_role_id": r.dir[clinicA].role}},
		{name: "clinic A with a conversation set", scope: map[string]string{"app.organization_id": clinicA, "app.conversation_id": r.conv["inA"]}},
	} {
		if got := idsWith(t, app, tt.scope, search, tt.only); !slices.Equal(got, tt.want) {
			t.Errorf("%s: finds %v, want %v", tt.name, got, tt.want)
		}
	}
	if _, err := owner.Exec(t.Context(), `UPDATE conversation_participant SET removed_at = now() WHERE id = $1`, r.pract["staffA"]); err != nil {
		t.Fatal(err)
	}
	if got := idsWith(t, app, chatRole(clinicA, r.dir[clinicA].role), search, ""); !slices.Equal(got, []string{r.second}) {
		t.Errorf("role A once removed from staffA: finds %v", got)
	}
}

// guestFiles is seedGuest (chat_group_test.go) with what a search and a download read: clinic A's treating doctor
// in inA; a second message in inA, the only one after the guest's history_from_seq (1); a side thread of inA without
// the guest, with a message; and a file attached to each of those messages, inA's first, its second, staffA's (the
// side thread the guest takes part in once granted) and the thread's without them.
type guestFiles struct {
	file map[string]string // by "inA", "second", "staffA", "without"
	guestRows
	second, without, withoutMsg string
}

func seedGuestFiles(t *testing.T, owner *pgxpool.Pool, state string) guestFiles {
	t.Helper()
	ctx := t.Context()
	g := guestFiles{guestRows: seedGuest(t, owner, state), file: map[string]string{}}
	roleA := g.dir[clinicA].role
	if _, err := owner.Exec(ctx, `UPDATE conversation_participant SET history_from_seq = 1 WHERE id = $1`, g.part); err != nil {
		t.Fatal(err)
	}
	if _, err := owner.Exec(ctx, `INSERT INTO conversation_participant (conversation_id, practitioner_role_id, organization_id, role, added_by)
		VALUES ($1, $2, $3, 'practitioner', $4)`, g.conv["inA"], roleA, clinicA, g.p[0].account); err != nil {
		t.Fatal(err)
	}
	if err := owner.QueryRow(ctx, `INSERT INTO communication (conversation_id, organization_id, patient_id, staff_only, seq, sent_at,
			sender_participant_id, sender_account_id, sender_role, client_message_id, kind, body)
		VALUES ($1, $2, $3, false, 2, '2026-11-12T10:05:00Z', $4, $5, 'patient', gen_random_uuid(), 'text', 'Synthetic placeholder two')
		RETURNING id`, g.conv["inA"], clinicA, g.p[0].patient, g.chatRows.part["inA"], g.p[0].account).Scan(&g.second); err != nil {
		t.Fatal(err)
	}
	var withoutPart string
	if err := owner.QueryRow(ctx, `INSERT INTO conversation (organization_id, patient_id, practitioner_role_id, staff_only, parent_id, created_by,
			last_seq, last_sent_at) VALUES ($1, $2, $3, true, $4, $5, 1, '2026-11-12T10:00:00Z') RETURNING id`,
		clinicA, g.p[0].patient, roleA, g.conv["inA"], g.p[0].account).Scan(&g.without); err != nil {
		t.Fatal(err)
	}
	if err := owner.QueryRow(ctx, `INSERT INTO conversation_participant (conversation_id, practitioner_role_id, organization_id, role, added_by)
		VALUES ($1, $2, $3, 'practitioner', $4) RETURNING id`, g.without, roleA, clinicA, g.p[0].account).Scan(&withoutPart); err != nil {
		t.Fatal(err)
	}
	if err := owner.QueryRow(ctx, `INSERT INTO communication (conversation_id, organization_id, patient_id, staff_only, seq, sent_at,
			sender_participant_id, sender_account_id, sender_role, client_message_id, kind, body)
		VALUES ($1, $2, $3, true, 1, '2026-11-12T10:00:00Z', $4, $5, 'practitioner', gen_random_uuid(), 'text', 'Synthetic placeholder between doctors')
		RETURNING id`, g.without, clinicA, g.p[0].patient, withoutPart, g.p[0].account).Scan(&g.withoutMsg); err != nil {
		t.Fatal(err)
	}
	for name, f := range map[string]struct{ conv, by, msg, at string }{
		"inA":     {g.conv["inA"], g.chatRows.part["inA"], g.msg["inA"], "2026-11-12T10:00:00Z"},
		"second":  {g.conv["inA"], g.chatRows.part["inA"], g.second, "2026-11-12T10:05:00Z"},
		"staffA":  {g.conv["staffA"], g.chatRows.part["staffA"], g.msg["staffA"], "2026-11-12T10:00:00Z"},
		"without": {g.without, withoutPart, g.withoutMsg, "2026-11-12T10:00:00Z"},
	} {
		var id string
		if err := owner.QueryRow(ctx, insertChatFile, clinicA, g.p[0].patient, f.conv, f.by).Scan(&id); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if _, err := owner.Exec(ctx, `UPDATE "binary" SET communication_id = $2, communication_sent_at = $3 WHERE id = $1`, id, f.msg, f.at); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		g.file[name] = id
	}
	return g
}

// A doctor of another clinic (SAL-28) searches and reads the files of a search (SAL-30, finding 7) only as an active
// participant, while the patient's chat consent for them and the care consent with the treating clinic are active:
// in inA after their history_from_seq and in the side thread they take part in, never the side thread without them;
// nothing while pending, after a decline, a revocation of either consent or a removal. The treating doctor of clinic
// A reads theirs while the care consent is active, whatever the guest's state. The search function, the messages'
// and the files' policies agree.
func TestChatSearch_AGuestFindsAndReadsOnlyWhileGranted(t *testing.T) {
	t.Parallel()
	const declined = "declined"
	for _, state := range []string{guestPending, declined, guestGranted, guestCareRevoked, guestRevoked, guestRemoved} {
		t.Run(state, func(t *testing.T) {
			t.Parallel()
			owner, app := pgtest.NewAppDatabase(t)
			seed := state
			if state == declined {
				seed = guestPending
			}
			g := seedGuestFiles(t, owner, seed)
			if state == declined {
				if _, err := owner.Exec(t.Context(), `UPDATE conversation_participant SET removed_at = now() WHERE id = $1`, g.part); err != nil {
					t.Fatal(err)
				}
			}
			g.check(t, app, state)
		})
	}
}

// check asserts what each scope finds, and reads of the messages and files, in the guest's state.
func (g guestFiles) check(t *testing.T, app *pgxpool.Pool, state string) {
	t.Helper()
	var guestMsgs, guestFiles, doctorMsgs, doctorFiles []string
	if state == guestGranted {
		guestMsgs, guestFiles = sorted(g.second, g.msg["staffA"]), sorted(g.file["second"], g.file["staffA"])
	}
	if state != guestCareRevoked {
		doctorMsgs = sorted(g.msg["inA"], g.second, g.withoutMsg)
		doctorFiles = sorted(g.file["inA"], g.file["second"], g.file["without"])
	}
	search := `SELECT id::text FROM communication_search(to_tsquery('public.chat_search', 'synth:*'), NULLIF($1, '')::uuid, 50)`
	for _, tt := range []struct {
		scope        map[string]string
		name, only   string
		msgs, files  []string
		onlyFindings []string
	}{
		{name: "the guest", scope: chatRole(clinicB, g.role), msgs: guestMsgs, files: guestFiles},
		{name: "the guest in inA", scope: chatRole(clinicB, g.role), only: g.conv["inA"], onlyFindings: g.pickOf(guestMsgs, g.second)},
		{name: "the guest in their side thread", scope: chatRole(clinicB, g.role), only: g.conv["staffA"], onlyFindings: g.pickOf(guestMsgs, g.msg["staffA"])},
		{name: "the guest in the side thread without them", scope: chatRole(clinicB, g.role), only: g.without},
		{name: "the guest's search role without their role", scope: map[string]string{"app.organization_id": clinicB, "app.chat_practitioner_role_id": g.role}},
		{name: "the guest's role with clinic A", scope: chatRole(clinicA, g.role)},
		{name: "the treating doctor", scope: chatRole(clinicA, g.dir[clinicA].role), msgs: doctorMsgs, files: doctorFiles},
		{name: "the treating doctor's search role without their role", scope: map[string]string{"app.organization_id": clinicA,
			"app.chat_practitioner_role_id": g.dir[clinicA].role}},
		{name: "the treating doctor's search role with another role", scope: map[string]string{"app.organization_id": clinicA,
			"app.chat_practitioner_role_id": g.dir[clinicA].role, "app.practitioner_role_id": g.role}},
	} {
		if tt.only != "" {
			if got := idsWith(t, app, tt.scope, search, tt.only); !slices.Equal(got, tt.onlyFindings) {
				t.Errorf("%s: finds %v, want %v", tt.name, got, tt.onlyFindings)
			}
			continue
		}
		if got := idsWith(t, app, tt.scope, search, ""); !slices.Equal(got, tt.msgs) {
			t.Errorf("%s: finds %v, want %v", tt.name, got, tt.msgs)
		}
		if got := idsIn(t, app, tt.scope, `SELECT id::text FROM communication`); !slices.Equal(got, tt.msgs) {
			t.Errorf("%s: reads the messages %v, want %v", tt.name, got, tt.msgs)
		}
		if got := idsIn(t, app, tt.scope, `SELECT id::text FROM "binary"`); !slices.Equal(got, tt.files) {
			t.Errorf("%s: reads the files %v, want %v", tt.name, got, tt.files)
		}
	}
}

// pickOf is want when it holds id, else nothing.
func (guestFiles) pickOf(want []string, id string) []string {
	if slices.Contains(want, id) {
		return []string{id}
	}
	return nil
}
