package communication_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"slices"
	"sync"
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

// GrantChat ends an earlier active one for the conversation and role first, as consent.Service.GrantChat does.
func (o scopedChatConsents) GrantChat(ctx context.Context, by consent.Caller, g consent.ChatGrant) (consent.Consent, error) {
	if _, err := o.RevokeChat(ctx, by, g.ConversationID, g.PractitionerRoleID); err != nil && !errors.Is(err, consent.ErrInactive) {
		return consent.Consent{}, err
	}
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

func (o scopedChatConsents) EndChat(ctx context.Context, by consent.Caller, conv, role string) (consent.Consent, error) {
	return o.RevokeChat(ctx, by, conv, role)
}

func (o scopedChatConsents) ChatGranted(ctx context.Context, conv, role string) (bool, error) {
	var ok bool
	err := postgres.Conn(ctx, o.db).QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM chat_consent WHERE conversation_id = $1 AND practitioner_role_id = $2
		AND status = 'active')`, conv, role).Scan(&ok)
	return ok, err
}

// racingRepo is the Postgres repository that runs race once, right after the service read a participant and before
// it locks the conversation: what another request commits in between (race runs its own transactions).
type racingRepo struct {
	communication.Repository
	race func()
	mu   sync.Mutex
}

func (r *racingRepo) Participant(ctx context.Context, id string) (communication.Participant, error) {
	p, err := r.Repository.Participant(ctx, id)
	r.mu.Lock()
	race := r.race
	r.race = nil
	r.mu.Unlock()
	if race != nil {
		race()
	}
	return p, err
}

// pgGroupFixture is the fixture's service on Postgres as the app role, with the chat consents in Postgres (the
// guest's row-level security reads them), clinic B with its role B1 and the guest's account.
func pgGroupFixture(t *testing.T) (*fixture, *pgxpool.Pool) {
	t.Helper()
	f, owner, _ := pgRaceFixture(t)
	return f, owner
}

// pgRaceFixture is pgGroupFixture with its repository's race to set.
func pgRaceFixture(t *testing.T) (*fixture, *pgxpool.Pool, *racingRepo) {
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
	repo, tx := &racingRepo{Repository: communication.NewPostgresRepository(app)}, postgres.NewTransactor(app)
	f := newFixtureOn(t, repo, tx)
	f.svc = communication.NewService(communication.Ports{
		Repo: repo, Publisher: f.pub, Consents: fakeConsents{f}, Visits: fakeVisits{f}, Roles: fakeRoles{}, Encounters: fakeEncounters{},
		ChatConsents: scopedChatConsents{app}, Directory: fakeDirectory{}, Audit: f.audit, Tx: tx, Clock: f.clock, Jobs: f.jobs,
		Log: slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	return f, owner, repo
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

	invitations, err := f.svc.Invitations(ctx, me, "")
	if err != nil || len(invitations) != 1 || invitations[0].ID != invited.ID || invitations[0].Reason != communication.ReasonSecondOpinion {
		t.Fatalf("Invitations = %+v, %v", invitations, err)
	}
	if err := f.svc.DecideInvitation(ctx, me, invited.ID, communication.DecisionGrant); err != nil {
		t.Fatal(err)
	}
	granted, err := f.svc.Invitations(ctx, me, communication.StatusActive)
	if err != nil || len(granted) != 1 || granted[0].ID != invited.ID || granted[0].Status != communication.StatusActive {
		t.Fatalf("the granted invitations = %+v, %v", granted, err)
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

// systemCount is how many system messages of code the conversation holds, and activeChat how many of its chat consents
// are active, read as the owner.
func systemCount(t *testing.T, owner *pgxpool.Pool, conv, code string) (systems, activeChat int) {
	t.Helper()
	if err := owner.QueryRow(t.Context(), `SELECT (SELECT count(*) FROM communication WHERE conversation_id = $1 AND system_code = $2),
		(SELECT count(*) FROM chat_consent WHERE conversation_id = $1 AND status = 'active')`, conv, code).Scan(&systems, &activeChat); err != nil {
		t.Fatal(err)
	}
	return systems, activeChat
}

// A decision or a revocation that reads the invitation while another request changes it decides on the participant
// as it is once the conversation is locked: a grant after a removal grants nothing, a second grant adds nothing, and a
// revocation after a removal removes nothing twice (SAL-28 review, finding 2).
func TestPostgres_ADecisionRacingAnotherChangeOfTheInvitationActsOnItsStateUnderTheLock(t *testing.T) {
	t.Parallel()
	for name, tt := range map[string]struct {
		race    func(ctx context.Context, f *fixture, conv, part string) error // the other request
		act     func(ctx context.Context, f *fixture, part string) error       // the request that read the invitation first
		code    string                                                         // the system message there is one of
		chat    int                                                            // the active chat consents after both
		granted bool                                                           // the invitation is granted before the race
	}{
		"a grant after a removal": {
			race: func(ctx context.Context, f *fixture, conv, part string) error {
				return f.svc.RemoveParticipant(ctx, pract, conv, part)
			},
			act: func(ctx context.Context, f *fixture, part string) error {
				return f.svc.DecideInvitation(ctx, me, part, communication.DecisionGrant)
			},
			code: communication.SystemParticipantRemoved,
		},
		"two grants": {
			race: func(ctx context.Context, f *fixture, _, part string) error {
				return f.svc.DecideInvitation(ctx, me, part, communication.DecisionGrant)
			},
			act: func(ctx context.Context, f *fixture, part string) error {
				return f.svc.DecideInvitation(ctx, me, part, communication.DecisionGrant)
			},
			code: communication.SystemConsentGranted, chat: 1,
		},
		"a revocation after a removal": {
			granted: true,
			race: func(ctx context.Context, f *fixture, conv, part string) error {
				return f.svc.RemoveParticipant(ctx, pract, conv, part)
			},
			act:  func(ctx context.Context, f *fixture, part string) error { return f.svc.RevokeInvitation(ctx, me, part) },
			code: communication.SystemParticipantRemoved,
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			f, owner, repo := pgRaceFixture(t)
			conv := f.open(t)
			part := f.add(t, conv.ID, roleB1, clinicB)
			if tt.granted {
				if err := f.svc.DecideInvitation(t.Context(), me, part.ID, communication.DecisionGrant); err != nil {
					t.Fatal(err)
				}
			}
			var raced error
			repo.mu.Lock()
			repo.race = func() { raced = tt.race(t.Context(), f, conv.ID, part.ID) }
			repo.mu.Unlock()
			if err := tt.act(t.Context(), f, part.ID); !errors.Is(err, communication.ErrNotFound) {
				t.Errorf("the request that read first: %v, want ErrNotFound", err)
			}
			if raced != nil {
				t.Fatalf("the other request: %v", raced)
			}
			if systems, chat := systemCount(t, owner, conv.ID, tt.code); systems != 1 || chat != tt.chat {
				t.Errorf("%d system messages %s and %d active chat consents, want 1 and %d", systems, tt.code, chat, tt.chat)
			}
		})
	}
}

// Activating or removing a participant who is no longer pending, or no longer there, changes no row and says so.
func TestPostgres_ActivatingOrRemovingAParticipantTwiceIsNotFound(t *testing.T) {
	t.Parallel()
	f, _, repo := pgRaceFixture(t)
	conv := f.open(t)
	part := f.add(t, conv.ID, roleB1, clinicB)
	colleague := f.add(t, conv.ID, roleA2, clinicA)
	if err := f.svc.RemoveParticipant(t.Context(), pract, conv.ID, part.ID); err != nil {
		t.Fatal(err)
	}
	for name, change := range map[string]func(ctx context.Context) error{
		"activate a removed invitation": func(ctx context.Context) error { return repo.ActivateParticipant(ctx, part.ID) },
		"activate an active doctor":     func(ctx context.Context) error { return repo.ActivateParticipant(ctx, colleague.ID) },
		"remove a removed invitation":   func(ctx context.Context) error { return repo.RemoveParticipant(ctx, part.ID, start) },
	} {
		err := f.svc.Tx.WithTx(t.Context(), func(ctx context.Context) error {
			if err := repo.SetClinic(ctx, clinicA); err != nil {
				return err
			}
			return change(ctx)
		})
		if !errors.Is(err, communication.ErrNotFound) {
			t.Errorf("%s: %v, want ErrNotFound", name, err)
		}
	}
}

// On Postgres, the notifier's read in the clinic's scope gives the granted guest and the clinic's own (SAL-29 after
// SAL-28): the invitation pending, the guest is left out; granted, they are in; revoked, out again.
func TestPostgres_NoticeConversationGivesTheGuestOnlyWhileGranted(t *testing.T) {
	t.Parallel()
	f, _ := pgGroupFixture(t)
	conv := f.open(t)
	p := f.add(t, conv.ID, roleB1, clinicB)
	roles := func() []string {
		c, err := f.svc.NoticeConversation(t.Context(), "chat_message_sent", clinicA, conv.ID)
		if err != nil {
			t.Fatal(err)
		}
		var out []string
		for _, q := range c.Participants {
			out = append(out, q.PractitionerRoleID)
		}
		return out
	}
	if got := roles(); slices.Contains(got, roleB1) || len(got) != 2 {
		t.Errorf("pending: %v, want the patient and role A1", got)
	}
	if err := f.svc.DecideInvitation(t.Context(), me, p.ID, communication.DecisionGrant); err != nil {
		t.Fatal(err)
	}
	if got := roles(); !slices.Contains(got, roleB1) || len(got) != 3 {
		t.Errorf("granted: %v, want the guest too", got)
	}
	if err := f.svc.RevokeInvitation(t.Context(), me, p.ID); err != nil {
		t.Fatal(err)
	}
	if got := roles(); slices.Contains(got, roleB1) {
		t.Errorf("revoked: %v, want the guest left out", got)
	}
}
