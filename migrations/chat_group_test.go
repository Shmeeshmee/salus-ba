package migrations_test

import (
	"context"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// chatGuestLookupRole owns chat_guest_conversations (migration 00044).
const chatGuestLookupRole = "salus_chat_guest"

// The states of a doctor of clinic B invited into patient A's conversation inA with clinic A (docs/plans/chat.md,
// SAL-28).
const (
	guestPending     = "pending"         // invited, the patient has not decided
	guestGranted     = "granted"         // the patient consented: the participant is active, the chat consent active
	guestCareRevoked = "care revoked"    // granted, then the patient revoked their care consent with clinic A
	guestRevoked     = "consent revoked" // granted, then the patient revoked the chat consent
	guestRemoved     = "removed"         // granted, then the treating doctor removed them
)

// guestRows is seedChat with clinic B's role invited into inA, in a state, and active in inA's side thread staffA
// once granted.
type guestRows struct {
	chatRows
	role    string // clinic B's role, the guest
	part    string // the guest's participant row in inA
	consent string // the chat consent, "" while pending
}

func seedGuest(t *testing.T, owner *pgxpool.Pool, state string) guestRows {
	t.Helper()
	ctx := t.Context()
	g := guestRows{chatRows: seedChat(t, owner)}
	g.role = g.dir[clinicB].role
	status := "pending_consent"
	if state != guestPending {
		status = "active"
	}
	if err := owner.QueryRow(ctx, `INSERT INTO conversation_participant (conversation_id, practitioner_role_id, organization_id, role, status,
			reason, added_by) VALUES ($1, $2, $3, 'practitioner', $4, 'second_opinion', $5) RETURNING id`,
		g.conv["inA"], g.role, clinicB, status, g.p[0].account).Scan(&g.part); err != nil {
		t.Fatal(err)
	}
	if state == guestPending {
		return g
	}
	if _, err := owner.Exec(ctx, `INSERT INTO conversation_participant (conversation_id, practitioner_role_id, organization_id, role, added_by)
		VALUES ($1, $2, $3, 'practitioner', $4)`, g.conv["staffA"], g.role, clinicB, g.p[0].account); err != nil {
		t.Fatal(err)
	}
	if err := owner.QueryRow(ctx, `INSERT INTO chat_consent (patient_id, organization_id, conversation_id, practitioner_role_id, practitioner_organization_id,
			status, date_time) VALUES ($1, $2, $3, $4, $5, 'active', '2026-11-12T10:05:00Z') RETURNING id`, g.p[0].patient, clinicA, g.conv["inA"], g.role, clinicB).Scan(&g.consent); err != nil {
		t.Fatal(err)
	}
	var stmt string
	switch state {
	case guestCareRevoked:
		stmt = `UPDATE consent SET status = 'inactive', revoked_at = '2026-11-12T11:00:00Z' WHERE organization_id = '` + clinicA + `'`
	case guestRevoked:
		stmt = `UPDATE chat_consent SET status = 'inactive', revoked_at = '2026-11-12T11:00:00Z'`
	case guestRemoved:
		stmt = `UPDATE conversation_participant SET removed_at = now() + interval '1 hour' WHERE id = '` + g.part + `'`
	}
	if stmt != "" {
		if _, err := owner.Exec(ctx, stmt); err != nil {
			t.Fatal(err)
		}
	}
	return g
}

func (g guestRows) scope(extra ...string) map[string]string {
	s := map[string]string{"app.organization_id": clinicB, "app.practitioner_role_id": g.role}
	for i := 0; i+1 < len(extra); i += 2 {
		s[extra[i]] = extra[i+1]
	}
	return s
}

// A doctor of another clinic sees nothing of the conversation before the patient consents: not the conversation,
// its participants, its side threads, its messages, nor the invitation; once granted, the conversation and its side
// threads, and their messages with the conversation set; nothing again after a revocation of either consent or a
// removal (docs/plans/chat.md, SAL-28, acceptance).
func TestChatGroup_AGuestSeesTheConversationOnlyWhileGranted(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		state string
		convs []string
	}{
		{state: guestPending},
		{state: guestGranted, convs: []string{"inA", "staffA"}},
		{state: guestCareRevoked},
		{state: guestRevoked},
		{state: guestRemoved},
	} {
		t.Run(tt.state, func(t *testing.T) {
			t.Parallel()
			owner, app := pgtest.NewAppDatabase(t)
			g := seedGuest(t, owner, tt.state)
			want := g.convs(tt.convs...)
			if got := idsIn(t, app, g.scope(), "SELECT id::text FROM conversation"); !slices.Equal(got, want) {
				t.Errorf("conversations = %v, want %v", got, want)
			}
			if got := idsIn(t, app, g.scope(), "SELECT DISTINCT conversation_id::text FROM conversation_participant"); !slices.Equal(got, want) {
				t.Errorf("participants of %v, want %v", got, want)
			}
			if got := idsIn(t, app, g.scope(), "SELECT id::text FROM communication"); got != nil {
				t.Errorf("messages without a conversation set: %v", got)
			}
			for _, name := range []string{"inA", "staffA", "other"} {
				var msgs []string
				if slices.Contains(tt.convs, name) {
					msgs = g.msgs(name)
				}
				scope := g.scope("app.conversation_id", g.conv[name])
				if got := idsIn(t, app, scope, "SELECT id::text FROM communication"); !slices.Equal(got, msgs) {
					t.Errorf("messages with %s set = %v, want %v", name, got, msgs)
				}
				if got := idsIn(t, app, scope, "SELECT communication_id::text FROM communication_revision"); !slices.Equal(got, msgs) {
					t.Errorf("revisions with %s set = %v, want %v", name, got, msgs)
				}
			}
		})
	}
}

// The guest's scope opens nothing else: without its clinic or its role, with another clinic or role, or with a
// patient or account set, the granted guest's conversation is not seen.
func TestChatGroup_TheGuestScopeOpensNothingElse(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	g := seedGuest(t, owner, guestGranted)
	const elsewhere = "c0000000-0000-4000-8000-00000000000c"
	for name, tt := range map[string]struct {
		scope map[string]string
		convs []string
	}{
		"clinic B without the role":          {scope: map[string]string{"app.organization_id": clinicB}},
		"the role without its clinic":        {scope: map[string]string{"app.practitioner_role_id": g.role}},
		"the role with another clinic":       {scope: map[string]string{"app.organization_id": elsewhere, "app.practitioner_role_id": g.role}},
		"clinic B with clinic A's role":      {scope: map[string]string{"app.organization_id": clinicB, "app.practitioner_role_id": g.dir[clinicA].role}},
		"the guest with an account set":      {scope: g.scope("app.account_id", g.p[0].account)},
		"the guest with patient B set":       {scope: g.scope("app.patient_id", g.p[1].patient), convs: []string{"other"}},
		"the guest with patient A set":       {scope: g.scope("app.patient_id", g.p[0].patient), convs: []string{"inA", "inB"}},
		"clinic A with the guest's role set": {scope: map[string]string{"app.organization_id": clinicA, "app.practitioner_role_id": g.role}, convs: []string{"inA", "staffA"}},
	} {
		if got, want := idsIn(t, app, tt.scope, "SELECT id::text FROM conversation"), g.convs(tt.convs...); !slices.Equal(got, want) {
			t.Errorf("%s: conversations %v, want %v", name, got, want)
		}
	}
}

// The patient's chat consents: the patient sees and writes their own, the treating clinic reads them, the guest
// reads their own; and the guest reads the patient's care consent with the treating clinic only while granted.
func TestChatGroup_ChatConsentsAreSeenByThePatientTheClinicAndTheGuest(t *testing.T) {
	t.Parallel()
	for _, state := range []string{guestPending, guestGranted} {
		t.Run(state, func(t *testing.T) {
			t.Parallel()
			owner, app := pgtest.NewAppDatabase(t)
			g := seedGuest(t, owner, state)
			var consents []string
			if g.consent != "" {
				consents = []string{g.consent}
			}
			for name, scope := range map[string]map[string]string{
				"patient A": {"app.patient_id": g.p[0].patient}, "clinic A": {"app.organization_id": clinicA}, "the guest": g.scope(),
			} {
				if got := idsIn(t, app, scope, "SELECT id::text FROM chat_consent"); !slices.Equal(got, consents) {
					t.Errorf("%s: chat consents %v, want %v", name, got, consents)
				}
			}
			for name, scope := range map[string]map[string]string{
				"patient B": {"app.patient_id": g.p[1].patient}, "clinic B without the role": {"app.organization_id": clinicB},
				"another role": {"app.organization_id": clinicB, "app.practitioner_role_id": g.dir[clinicA].role},
			} {
				if got := idsIn(t, app, scope, "SELECT id::text FROM chat_consent"); got != nil {
					t.Errorf("%s: chat consents %v, want none", name, got)
				}
			}
			care := []string{clinicB} // clinic B's own (revoked) consent of patient A
			if state == guestGranted {
				care = sorted(clinicA, clinicB)
			}
			if got := idsIn(t, app, g.scope(), "SELECT organization_id::text FROM consent"); !slices.Equal(got, care) {
				t.Errorf("the guest reads the care consents of %v, want %v", got, care)
			}
		})
	}
}

func TestChatGroup_AGrantedGuestWritesInTheConversationAndOpensASideThreadOnly(t *testing.T) {
	t.Parallel()
	for _, state := range []string{guestPending, guestGranted} {
		t.Run(state, func(t *testing.T) {
			t.Parallel()
			owner, app := pgtest.NewAppDatabase(t)
			g := seedGuest(t, owner, state)
			allowed := state == guestGranted
			exec := func(scope map[string]string, stmt string, args ...any) (int64, error) {
				var n int64
				err := asApp(t, app, scope, func(ctx context.Context, db postgres.Querier) error {
					tag, err := db.Exec(ctx, stmt, args...)
					n = tag.RowsAffected()
					return err
				})
				return n, err
			}
			err := asApp(t, app, g.scope("app.conversation_id", g.conv["inA"]), insertMessage(t, owner, g.conv["inA"], g.p[0].account))
			if allowed != (err == nil) {
				t.Errorf("a message into inA: %v, want allowed %v", err, allowed)
			}
			n, err := exec(g.scope(), "UPDATE conversation SET last_seq = last_seq WHERE id = $1", g.conv["inA"])
			if err != nil || (n == 1) != allowed {
				t.Errorf("the conversation's counters: %d rows, %v; want allowed %v", n, err, allowed)
			}
			_, err = exec(g.scope(), `INSERT INTO conversation (organization_id, patient_id, practitioner_role_id, staff_only, parent_id, created_by)
				VALUES ($1, $2, $3, true, $4, $5)`, clinicA, g.p[0].patient, g.dir[clinicA].role, g.conv["inA"], g.p[0].account)
			if allowed != (err == nil) {
				t.Errorf("a side thread of inA: %v, want allowed %v", err, allowed)
			}
			_, err = exec(g.scope(), `INSERT INTO conversation (organization_id, patient_id, practitioner_role_id, encounter_id, created_by)
				VALUES ($1, $2, $3, NULL, $4)`, clinicA, g.p[1].patient, g.dir[clinicA].role, g.p[0].account)
			if sqlState(err) != "42501" {
				t.Errorf("a conversation of its own in clinic A: %v, want a row-level security violation (42501)", err)
			}
		})
	}
}

// A practitioner removed from a conversation leaves its side threads in the same statement, whoever removes them
// (the patient's scope does not see the threads).
func TestChatGroup_ARemovalFromAConversationEndsItsSideThreadsToo(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	g := seedGuest(t, owner, guestGranted)
	err := asApp(t, app, map[string]string{"app.patient_id": g.p[0].patient}, func(ctx context.Context, db postgres.Querier) error {
		_, err := db.Exec(ctx, `UPDATE conversation_participant SET removed_at = now() + interval '1 hour' WHERE id = $1`, g.part)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	var left int
	if err := owner.QueryRow(t.Context(), `SELECT count(*) FROM conversation_participant WHERE practitioner_role_id = $1 AND removed_at IS NULL`,
		g.role).Scan(&left); err != nil || left != 0 {
		t.Errorf("the guest's active rows after the removal: %d (%v), want 0", left, err)
	}
}

func TestChatGroup_HoldsAConsistentState(t *testing.T) {
	t.Parallel()
	owner, _ := pgtest.NewAppDatabase(t)
	g := seedGuest(t, owner, guestGranted)
	conv, roleA, roleB, acc := g.conv["inA"], g.dir[clinicA].role, g.role, g.p[0].account
	for _, tt := range []struct {
		name, stmt string
		args       []any
	}{
		{"a practitioner without their clinic", `INSERT INTO conversation_participant (conversation_id, practitioner_role_id, role, added_by)
			VALUES ($1, $2, 'practitioner', $3)`, []any{conv, roleA, acc}},
		{"a practitioner with another clinic than their role's", `INSERT INTO conversation_participant (conversation_id, practitioner_role_id, organization_id,
			role, added_by) VALUES ($1, $2, $3, 'practitioner', $4)`, []any{conv, roleA, clinicB, acc}},
		{"the patient with a clinic", `INSERT INTO conversation_participant (conversation_id, patient_id, organization_id, role, added_by)
			VALUES ($1, $2, $3, 'patient', $4)`, []any{g.conv["inB"], g.p[0].patient, clinicA, acc}},
		{"the patient pending a consent", `UPDATE conversation_participant SET status = 'pending_consent' WHERE id = $1`, []any{g.chatRows.part["inA"]}},
		{"a reason not in the list", `UPDATE conversation_participant SET reason = 'curiosity' WHERE id = $1`, []any{g.part}},
		{"a second active chat consent for the role", `INSERT INTO chat_consent (patient_id, organization_id, conversation_id, practitioner_role_id,
			practitioner_organization_id, status, date_time) VALUES ($1, $2, $3, $4, $5, 'active', now())`, []any{g.p[0].patient, clinicA, conv, roleB, clinicB}},
		{"a chat consent for another patient's conversation", `INSERT INTO chat_consent (patient_id, organization_id, conversation_id, practitioner_role_id,
			practitioner_organization_id, status, date_time) VALUES ($1, $2, $3, $4, $5, 'active', now())`, []any{g.p[1].patient, clinicA, conv, roleA, clinicA}},
		{"a chat consent naming another clinic than its role's", `INSERT INTO chat_consent (patient_id, organization_id, conversation_id, practitioner_role_id,
			practitioner_organization_id, status, date_time) VALUES ($1, $2, $3, $4, $5, 'active', now())`, []any{g.p[0].patient, clinicA, conv, roleA, clinicB}},
		{"an inactive chat consent without its revocation", `UPDATE chat_consent SET status = 'inactive' WHERE id = $1`, []any{g.consent}},
	} {
		if _, err := owner.Exec(t.Context(), tt.stmt, tt.args...); err == nil {
			t.Errorf("%s: accepted, want a refusal", tt.name)
		}
	}
}

func TestChatGroup_TheAppRoleChangesOnlyWhatTheApplicationNeeds(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	g := seedGuest(t, owner, guestGranted)
	scope := map[string]string{"app.patient_id": g.p[0].patient}
	for _, stmt := range []string{
		"UPDATE conversation_participant SET organization_id = organization_id", "UPDATE conversation_participant SET reason = reason",
		"UPDATE conversation_participant SET added_by = added_by",
		"UPDATE chat_consent SET conversation_id = conversation_id", "UPDATE chat_consent SET practitioner_role_id = practitioner_role_id",
		"UPDATE chat_consent SET patient_id = patient_id", "UPDATE chat_consent SET practitioner_organization_id = practitioner_organization_id", "DELETE FROM chat_consent", "TRUNCATE chat_consent",
	} {
		err := asApp(t, app, scope, func(ctx context.Context, db postgres.Querier) error {
			_, err := db.Exec(ctx, stmt)
			return err
		})
		if sqlState(err) != "42501" {
			t.Errorf("app: %s: %v, want insufficient_privilege (42501)", stmt, err)
		}
	}
	err := asApp(t, app, scope, func(ctx context.Context, db postgres.Querier) error {
		_, err := db.Exec(ctx, `UPDATE chat_consent SET status = 'inactive', revoked_at = now(), updated_at = now(), version_id = version_id + 1`)
		if err == nil {
			_, err = db.Exec(ctx, `UPDATE conversation_participant SET status = 'active' WHERE id = $1`, g.part)
		}
		return err
	})
	if err != nil {
		t.Errorf("the patient revokes and a participant is activated: %v", err)
	}
}

func TestChatGroup_TheLookupFunctionIsOwnedByAnUnprivilegedRoleWithAFixedSearchPath(t *testing.T) {
	t.Parallel()
	owner := pgtest.NewDatabase(t)
	ctx := t.Context()
	var fnOwner string
	var definer bool
	var config []string
	if err := owner.QueryRow(ctx, `SELECT pg_get_userbyid(proowner), prosecdef, coalesce(proconfig, '{}')
		FROM pg_proc WHERE proname = 'chat_guest_conversations'`).Scan(&fnOwner, &definer, &config); err != nil {
		t.Fatal(err)
	}
	if fnOwner != chatGuestLookupRole || !definer || !slices.Contains(config, "search_path=pg_catalog, public") {
		t.Errorf("chat_guest_conversations: owner %q, SECURITY DEFINER %v, config %v", fnOwner, definer, config)
	}
	var super, bypass, login, createRole, createDB bool
	if err := owner.QueryRow(ctx, `SELECT rolsuper, rolbypassrls, rolcanlogin, rolcreaterole, rolcreatedb FROM pg_roles WHERE rolname = $1`,
		chatGuestLookupRole).Scan(&super, &bypass, &login, &createRole, &createDB); err != nil || super || bypass || login || createRole || createDB {
		t.Errorf("%s: superuser %v, BYPASSRLS %v, LOGIN %v, CREATEROLE %v, CREATEDB %v (%v); want all false",
			chatGuestLookupRole, super, bypass, login, createRole, createDB, err)
	}
	var publicExec, appExec bool
	if err := owner.QueryRow(ctx, `SELECT has_function_privilege('public', 'chat_guest_conversations(uuid, uuid)', 'EXECUTE'),
		has_function_privilege($1, 'chat_guest_conversations(uuid, uuid)', 'EXECUTE')`, pgtest.AppRole).Scan(&publicExec, &appExec); err != nil || publicExec || !appExec {
		t.Errorf("EXECUTE for PUBLIC %v, for %s %v (%v); want false and true", publicExec, pgtest.AppRole, appExec, err)
	}
	var cols []string
	if err := owner.QueryRow(ctx, `SELECT coalesce(array_agg(table_name || '.' || column_name ORDER BY table_name, column_name), '{}')
		FROM information_schema.column_privileges WHERE grantee = $1`, chatGuestLookupRole).Scan(&cols); err != nil {
		t.Fatal(err)
	}
	want := []string{"chat_consent.conversation_id", "chat_consent.organization_id", "chat_consent.patient_id", "chat_consent.practitioner_role_id",
		"chat_consent.status", "consent.organization_id", "consent.patient_id", "consent.status", "conversation.id", "conversation.organization_id",
		"conversation.patient_id", "conversation_participant.conversation_id", "conversation_participant.organization_id",
		"conversation_participant.practitioner_role_id", "conversation_participant.removed_at", "conversation_participant.status"}
	if !slices.Equal(cols, want) {
		t.Errorf("%s holds column privileges on %v, want %v", chatGuestLookupRole, cols, want)
	}
	var tables int
	if err := owner.QueryRow(ctx, `SELECT count(*) FROM information_schema.table_privileges WHERE grantee = $1`, chatGuestLookupRole).Scan(&tables); err != nil || tables != 0 {
		t.Errorf("%s holds %d table privileges (%v), want none", chatGuestLookupRole, tables, err)
	}
}

// The chat's policies read no table through another policy that leads back to them: appointment's policies read
// consent, whose guest policy reads chat_consent; chat_consent's policies compare columns only, so reading an
// appointment, a role, an encounter or a consent never recurses through practitioner_role's policies of 00043
// (SAL-28 after main's SAL-38), in any scope.
func TestChatGroup_ThePoliciesDoNotRecurseThroughTheAppointments(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	p, _, _ := seedAppointments(t, owner)
	for name, scope := range map[string]map[string]string{
		"clinic A":            {"app.organization_id": clinicA},
		"a guest of clinic B": {"app.organization_id": clinicB, "app.practitioner_role_id": "b1000000-0000-4000-8000-0000000000b1"},
		"patient A":           {"app.patient_id": p[0].patient},
	} {
		for _, table := range []string{"appointment", "practitioner_role", "encounter", "consent", "chat_consent", "conversation"} {
			err := asApp(t, app, scope, func(ctx context.Context, db postgres.Querier) error {
				var n int
				return db.QueryRow(ctx, "SELECT count(*) FROM "+table).Scan(&n)
			})
			if err != nil {
				t.Errorf("%s reads %s: %v", name, table, err)
			}
		}
	}
}

