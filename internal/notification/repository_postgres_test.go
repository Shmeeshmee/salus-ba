package notification_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/notification"
	"github.com/Shmeeshmee/salus/migrations"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
	"github.com/Shmeeshmee/salus/pkg/push"
)

func TestMain(m *testing.M) { pgtest.Main(m, migrations.FS) }

const (
	clinicB  = "b0000000-0000-4000-8000-00000000000b"
	accountB = "a9000000-0000-4000-8000-0000000000a9"
)

// seedAccounts creates, as the owner, the clinics A and B, a patient account and another, and clinic A's admin.
func seedAccounts(t *testing.T, owner *pgxpool.Pool) {
	t.Helper()
	_, err := owner.Exec(t.Context(), `
		INSERT INTO organization (id, name) VALUES ('`+clinicID+`', 'Synthetic clinic A'), ('`+clinicB+`', 'Synthetic clinic B');
		INSERT INTO account (id, email, password_hash, role) VALUES ('`+patientAccount+`', 'ana@example.test', 'h', 'patient'),
			('`+accountB+`', 'bo@example.test', 'h', 'patient');
		INSERT INTO account (id, email, password_hash, role, organization_id) VALUES ('`+clinicAdmin+`', 'admin@example.test', 'h', 'admin', '`+clinicID+`');`)
	if err != nil {
		t.Fatal(err)
	}
}

func inTx(t *testing.T, app *pgxpool.Pool, fn func(ctx context.Context) error) {
	t.Helper()
	if err := postgres.NewTransactor(app).WithTx(t.Context(), fn); err != nil {
		t.Fatal(err)
	}
}

// A token is one row: registering it again moves it to the account that registers it; unregistering or the
// platform's "gone" deletes it, the latter only while it is still the same token.
func TestDevices_RegisterMoveAndDelete(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seedAccounts(t, owner)
	repo := notification.NewPostgresRepository(app)
	at := time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)
	devices := func(account string) []notification.Device {
		var out []notification.Device
		inTx(t, app, func(ctx context.Context) error {
			var err error
			out, err = repo.Devices(ctx, account)
			return err
		})
		return out
	}
	inTx(t, app, func(ctx context.Context) error {
		if err := repo.RegisterDevice(ctx, notification.Device{AccountID: patientAccount, Token: "synthetic-1", Platform: push.Android}, at); err != nil {
			return err
		}
		return repo.RegisterDevice(ctx, notification.Device{AccountID: patientAccount, Token: "synthetic-2", Platform: push.IOS}, at)
	})
	got := devices(patientAccount)
	slices.SortFunc(got, func(a, b notification.Device) int { return strings.Compare(a.Token, b.Token) })
	if len(got) != 2 || got[0].Token != "synthetic-1" || got[1].Platform != push.IOS || got[0].AccountID != patientAccount {
		t.Fatalf("devices %+v", got)
	}
	inTx(t, app, func(ctx context.Context) error {
		return repo.RegisterDevice(ctx, notification.Device{AccountID: accountB, Token: "synthetic-1", Platform: push.Android}, at.Add(time.Hour))
	})
	if a, b := devices(patientAccount), devices(accountB); len(a) != 1 || len(b) != 1 || b[0].ID != got[0].ID {
		t.Errorf("after B registers token 1: A %+v, B %+v; want it moved", a, b)
	}
	var d notification.Device
	inTx(t, app, func(ctx context.Context) error {
		var err error
		d, err = repo.Device(ctx, got[1].ID)
		return err
	})
	if d.Token != "synthetic-2" {
		t.Errorf("Device() = %+v", d)
	}
	inTx(t, app, func(ctx context.Context) error {
		if err := repo.DeleteDevice(ctx, got[1].ID, "another-token"); err != nil {
			return err
		}
		return repo.UnregisterDevice(ctx, patientAccount, "synthetic-1") // B's now: not A's to unregister
	})
	if len(devices(patientAccount)) != 1 || len(devices(accountB)) != 1 {
		t.Error("a delete of another token, or another account's unregister, deleted a device")
	}
	inTx(t, app, func(ctx context.Context) error {
		if err := repo.DeleteDevice(ctx, got[1].ID, "synthetic-2"); err != nil {
			return err
		}
		return repo.UnregisterDevice(ctx, accountB, "synthetic-1")
	})
	if len(devices(patientAccount))+len(devices(accountB)) != 0 {
		t.Error("devices left")
	}
	err := postgres.NewTransactor(app).WithTx(t.Context(), func(ctx context.Context) error {
		_, err := repo.Device(ctx, got[1].ID)
		return err
	})
	if !errors.Is(err, notification.ErrNotFound) {
		t.Errorf("Device(deleted) = %v", err)
	}
}

// A clinic stores, lists, reads and deletes its endpoints and records dead letters; another clinic sees none; the
// retention role deletes the dead letters past their period.
func TestEndpoints_AreTheClinicsAndTheirDeadLettersGoWithThem(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seedAccounts(t, owner)
	repo := notification.NewPostgresRepository(app)
	e := notification.Endpoint{ID: "e1000000-0000-4000-8000-0000000000e1", OrganizationID: clinicID, CreatedBy: clinicAdmin, URL: "https://hooks.example.test/salus",
		Events: []string{notification.EventMessageCreated}, SecretSealed: []byte("sealed"), IncludeClinical: true}
	inTx(t, app, func(ctx context.Context) error {
		if err := repo.SetClinic(ctx, clinicID); err != nil {
			return err
		}
		created, err := repo.CreateEndpoint(ctx, e)
		if err != nil || created.CreatedAt.IsZero() {
			t.Errorf("CreateEndpoint() = %+v, %v", created, err)
		}
		return repo.AddDeadLetter(ctx, notification.DeadLetter{EndpointID: e.ID, OrganizationID: clinicID, EventID: msgID, EventType: notification.EventMessageCreated,
			ConversationID: convID, MessageID: msgID, Attempts: 8})
	})
	for _, org := range []string{clinicID, clinicB} {
		inTx(t, app, func(ctx context.Context) error {
			if err := repo.SetClinic(ctx, org); err != nil {
				return err
			}
			list, err := repo.Endpoints(ctx, clinicID)
			if err != nil {
				return err
			}
			one, oneErr := repo.Endpoint(ctx, e.ID)
			switch org {
			case clinicID:
				if len(list) != 1 || list[0].URL != e.URL || !list[0].IncludeClinical || string(list[0].SecretSealed) != "sealed" || one.ID != e.ID || oneErr != nil {
					t.Errorf("clinic A: %+v, %+v (%v)", list, one, oneErr)
				}
			default:
				if len(list) != 0 || !errors.Is(oneErr, notification.ErrNotFound) || !errors.Is(repo.DeleteEndpoint(ctx, e.ID), notification.ErrNotFound) {
					t.Errorf("clinic B sees %+v, %v", list, oneErr)
				}
			}
			return nil
		})
	}
	inTx(t, app, func(ctx context.Context) error {
		if err := repo.SetClinic(ctx, clinicID); err != nil {
			return err
		}
		return repo.DeleteEndpoint(ctx, e.ID)
	})
	var dead int
	if err := owner.QueryRow(t.Context(), "SELECT count(*) FROM webhook_dead_letter").Scan(&dead); err != nil || dead != 0 {
		t.Errorf("%d dead letters after the endpoint's deletion (%v)", dead, err)
	}
}

// The retention role deletes the dead letters recorded before the cut-off, of every clinic, and counts them.
func TestDeadLetterPurger_DeletesTheDeadLettersPastTheirPeriod(t *testing.T) {
	t.Parallel()
	owner, retention := pgtest.NewRetentionDatabase(t)
	seedAccounts(t, owner)
	_, err := owner.Exec(t.Context(), `
		INSERT INTO webhook_endpoint (id, organization_id, created_by, url, events, secret_sealed)
			VALUES ('e1000000-0000-4000-8000-0000000000e1', '`+clinicID+`', '`+clinicAdmin+`', 'https://hooks.example.test', '{message.created}', '\x00');
		INSERT INTO webhook_dead_letter (created_at, endpoint_id, organization_id, event_id, conversation_id, message_id, attempts, event_type)
			VALUES ('2026-10-01T00:00:00Z', 'e1000000-0000-4000-8000-0000000000e1', '`+clinicID+`', gen_random_uuid(), gen_random_uuid(), gen_random_uuid(), 8, 'message.created'),
			       ('2026-11-11T00:00:00Z', 'e1000000-0000-4000-8000-0000000000e1', '`+clinicID+`', gen_random_uuid(), gen_random_uuid(), gen_random_uuid(), 8, 'message.created');`)
	if err != nil {
		t.Fatal(err)
	}
	n, err := notification.NewDeadLetterPurger(notification.NewPostgresRepository(retention)).Purge(t.Context(), time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC))
	if err != nil || n != 1 {
		t.Errorf("Purge() = %d, %v; want the older one", n, err)
	}
}

// Any three-digit status an endpoint answers is recorded on the dead letter, beyond the registered 100 to 599 too:
// Go's client accepts them, and the last attempt's dead letter must not be lost to a CHECK.
func TestAddDeadLetter_RecordsAnyThreeDigitStatus(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seedAccounts(t, owner)
	repo := notification.NewPostgresRepository(app)
	e := notification.Endpoint{ID: "e1000000-0000-4000-8000-0000000000e1", OrganizationID: clinicID, CreatedBy: clinicAdmin, URL: "https://hooks.example.test/salus",
		Events: []string{notification.EventMessageCreated}, SecretSealed: []byte("sealed")}
	inTx(t, app, func(ctx context.Context) error {
		if err := repo.SetClinic(ctx, clinicID); err != nil {
			return err
		}
		if _, err := repo.CreateEndpoint(ctx, e); err != nil {
			return err
		}
		return repo.AddDeadLetter(ctx, notification.DeadLetter{EndpointID: e.ID, OrganizationID: clinicID, EventID: msgID, EventType: notification.EventMessageCreated,
			ConversationID: convID, MessageID: msgID, Attempts: 8, LastStatus: 999})
	})
	var status int
	if err := owner.QueryRow(t.Context(), "SELECT last_status FROM webhook_dead_letter").Scan(&status); err != nil || status != 999 {
		t.Errorf("last_status %d (%v), want 999", status, err)
	}
}
