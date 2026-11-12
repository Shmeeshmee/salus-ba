package communication_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/communication"
	"github.com/Shmeeshmee/salus/internal/consent"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// scopedChatConsents writes and reads the patient's chat consents in the caller's transaction and scope, as the consent
// service does, so the guest's row-level security reads them (the consent domain's repository is its own;
// internal/api's test runs the real service).
type scopedChatConsents struct{ db *pgxpool.Pool }

func (o scopedChatConsents) GrantChat(ctx context.Context, by consent.Caller, g consent.ChatGrant) (consent.Consent, error) {
	_, err := postgres.Conn(ctx, o.db).Exec(ctx, `INSERT INTO chat_consent (patient_id, organization_id, conversation_id, practitioner_role_id,
		practitioner_organization_id, status, date_time) VALUES ($1, $2, $3, $4, $5, 'active', now())`,
		by.PatientID, g.OrganizationID, g.ConversationID, g.PractitionerRoleID, g.PractitionerOrganizationID)
	return consent.Consent{Kind: consent.KindChat}, err
}

func (o scopedChatConsents) RevokeChat(ctx context.Context, _ consent.Caller, conv, role string) (consent.Consent, error) {
	tag, err := postgres.Conn(ctx, o.db).Exec(ctx, `UPDATE chat_consent SET status = 'inactive', revoked_at = now()
		WHERE conversation_id = $1 AND practitioner_role_id = $2 AND status = 'active'`, conv, role)
	if err == nil && tag.RowsAffected() == 0 {
		err = consent.ErrInactive
	}
	return consent.Consent{Kind: consent.KindChat}, err
}

func (o scopedChatConsents) ChatGranted(ctx context.Context, conv, role string) (bool, error) {
	var ok bool
	err := postgres.Conn(ctx, o.db).QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM chat_consent WHERE conversation_id = $1 AND practitioner_role_id = $2
		AND status = 'active')`, conv, role).Scan(&ok)
	return ok, err
}

// pgGroupFixture is the fixture's service on Postgres as the app role, with the chat consents in Postgres (the
// guest's row-level security reads them), clinic B with its role B1 and the guest's account.
func pgGroupFixture(t *testing.T) (*fixture, *pgxpool.Pool) {
	t.Helper()
	owner, app := pgtest.NewAppDatabase(t)
	seedChat(t, owner)
	for _, stmt := range []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO organization (id, name, listed) VALUES ($1, 'Synthetic clinic B', true)`, []any{clinicB}},
		{`WITH p AS (INSERT INTO practitioner (organization_id, given_name, family_name) VALUES ($1, 'Ed', 'Example') RETURNING id)
			INSERT INTO practitioner_role (id, organization_id, practitioner_id) SELECT $2, $1, id FROM p`, []any{clinicB, roleB1}},
		{`INSERT INTO account (id, email, password_hash, role, organization_id) VALUES ($1, 'b1@example.test', 'h', 'practitioner', $2)`,
			[]any{guest.AccountID, clinicB}},
	} {
		if _, err := owner.Exec(t.Context(), stmt.sql, stmt.args...); err != nil {
			t.Fatal(err)
		}
	}
	repo, tx := communication.NewPostgresRepository(app), postgres.NewTransactor(app)
	f := newFixtureOn(t, repo, tx)
	f.svc = communication.NewService(communication.Ports{
		Repo: repo, Publisher: f.pub, Consents: fakeConsents{f}, Visits: fakeVisits{f}, Roles: fakeRoles{}, Encounters: fakeEncounters{},
		ChatConsents: scopedChatConsents{app}, Directory: fakeDirectory{}, Audit: f.audit, Tx: tx, Clock: f.clock,
		Log: slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	return f, owner
}

// The group consult on Postgres (SAL-28 acceptance): the invited doctor of clinic B sees nothing before the patient
// consents, then the history and their side thread; the patient never sees the side thread; the patient's revocation
// ends the doctor's access to both at once.
func TestPostgres_AGuestSeesTheConversationOnlyWhileTheirInvitationIsGranted(t *testing.T) {
	t.Parallel()
	f, owner := pgGroupFixture(t)
	ctx := t.Context()
	conv := f.open(t)
	f.send(t, me, conv.ID, "Synthetic placeholder question")
	invited := f.add(t, conv.ID, roleB1, clinicB)
	if invited.Status != communication.StatusPendingConsent || invited.OrganizationID != clinicB {
		t.Fatalf("invited = %+v", invited)
	}
	checkNothing(t, f, guest, conv.ID)

	invitations, err := f.svc.Invitations(ctx, me)
	if err != nil || len(invitations) != 1 || invitations[0].ID != invited.ID || invitations[0].Reason != communication.ReasonSecondOpinion {
		t.Fatalf("Invitations = %+v, %v", invitations, err)
	}
	if err := f.svc.DecideInvitation(ctx, me, invited.ID, communication.DecisionGrant); err != nil {
		t.Fatal(err)
	}
	page, err := f.svc.Messages(ctx, guest, conv.ID, nil, 0)
	if err != nil || len(page.Messages) != 3 || page.Messages[0].SystemCode != communication.SystemConsentGranted {
		t.Fatalf("the guest reads %v, %+v", err, page.Messages)
	}
	f.send(t, guest, conv.ID, "Synthetic placeholder opinion")
	got, err := f.svc.Get(ctx, guest, conv.ID)
	if err != nil || len(got.Participants) != 3 || got.Participants[2].Status != communication.StatusActive {
		t.Errorf("the guest's Get = %+v, %v", got, err)
	}
	thread, err := f.svc.OpenThread(ctx, guest, conv.ID)
	if err != nil || len(thread.Participants) != 2 {
		t.Fatalf("the guest's thread = %+v, %v", thread, err)
	}
	f.send(t, guest, thread.ID, "Synthetic placeholder between doctors")
	if threads, err := f.svc.Threads(ctx, pract, conv.ID); err != nil || len(threads) != 1 || threads[0].Unread != 1 {
		t.Errorf("the treating practitioner's threads = %+v, %v", threads, err)
	}
	checkNothing(t, f, me, thread.ID)
	if _, err := f.svc.Threads(ctx, me, conv.ID); !errors.Is(err, communication.ErrForbidden) {
		t.Errorf("the patient's Threads = %v, want ErrForbidden", err)
	}

	if err := f.svc.RevokeInvitation(ctx, me, invited.ID); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{conv.ID, thread.ID} {
		checkNothing(t, f, guest, id)
	}
	var left int
	if err := owner.QueryRow(ctx, `SELECT count(*) FROM conversation_participant WHERE practitioner_role_id = $1 AND removed_at IS NULL`,
		roleB1).Scan(&left); err != nil || left != 0 {
		t.Errorf("the guest's rows left: %d (%v), want 0", left, err)
	}
	var consents int
	if err := owner.QueryRow(ctx, `SELECT count(*) FROM chat_consent WHERE status = 'inactive'`).Scan(&consents); err != nil || consents != 1 {
		t.Errorf("revoked chat consents: %d (%v), want 1", consents, err)
	}

	// Invited again and granted, the care consent's revocation with the treating clinic ends the access too.
	again := f.invite(t, conv.ID)
	if _, err := f.svc.Get(ctx, guest, conv.ID); err != nil {
		t.Fatalf("the guest granted again: %v", err)
	}
	if _, err := owner.Exec(ctx, `UPDATE consent SET status = 'inactive', revoked_at = now() WHERE patient_id = $1`, patientA); err != nil {
		t.Fatal(err)
	}
	f.revoke() // the fake's state, which the service reads
	checkNothing(t, f, guest, conv.ID)
	if err := f.svc.RevokeInvitation(ctx, me, again.ID); err != nil {
		t.Errorf("a revocation after the care consent's: %v", err)
	}
}

// A doctor of the clinic is added from a chosen point and removed again, on Postgres.
func TestPostgres_ADoctorOfTheClinicIsAddedFromAPointAndRemoved(t *testing.T) {
	t.Parallel()
	f, _ := pgGroupFixture(t)
	ctx := t.Context()
	conv := f.open(t)
	for range 3 {
		f.send(t, me, conv.ID, "Synthetic placeholder")
	}
	p, err := f.svc.AddParticipant(ctx, pract, conv.ID, communication.AddParticipant{
		PractitionerRoleID: roleA2, OrganizationID: clinicA, Reason: communication.ReasonHandover, History: communication.HistoryFrom, FromSeq: int64p(2),
	})
	if err != nil || p.Status != communication.StatusActive || p.Reason != communication.ReasonHandover {
		t.Fatalf("AddParticipant = %+v, %v", p, err)
	}
	if _, err := f.svc.AddParticipant(ctx, pract, conv.ID, communication.AddParticipant{
		PractitionerRoleID: roleA2, OrganizationID: clinicA, Reason: communication.ReasonHandover, History: communication.HistoryFull,
	}); !errors.Is(err, communication.ErrAlreadyParticipant) {
		t.Errorf("added twice: %v, want ErrAlreadyParticipant", err)
	}
	page, err := f.svc.Messages(ctx, pract2, conv.ID, nil, 0)
	if err != nil || len(page.Messages) != 2 {
		t.Fatalf("the doctor reads %v, %+v; want messages 3 and 4", err, page.Messages)
	}
	if got, err := f.svc.Get(ctx, pract2, conv.ID); err != nil || got.Unread != 2 {
		t.Errorf("unread = %d, %v; want 2", got.Unread, err)
	}
	if err := f.svc.RemoveParticipant(ctx, pract, conv.ID, p.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Get(ctx, pract2, conv.ID); !errors.Is(err, communication.ErrForbidden) {
		t.Errorf("the removed doctor's Get = %v, want ErrForbidden", err)
	}
	m := f.lastMessage(t, me, conv.ID)
	checkSystem(t, m, communication.SystemParticipantRemoved, map[string]string{"practitioner_role_id": roleA2})
}
