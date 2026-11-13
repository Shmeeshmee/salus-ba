package notification_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"

	"github.com/Shmeeshmee/salus/internal/notification"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// A delivery is stored once per source, channel and device, locked and moved from pending once; the deliveries go
// with the account and after their period (the retention role).
func TestDeliveries_StoredOnceSentOncePurged(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seedAccounts(t, owner)
	repo := notification.NewPostgresDeliveryRepository(app)
	at := time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)
	d := notification.Delivery{Kind: notification.KindReminder, SourceID: reminderID, OrganizationID: clinicID, AccountID: patientAccount}
	var email, push notification.Delivery
	inTx(t, app, func(ctx context.Context) error {
		var ok bool
		var err error
		email, ok, err = repo.AddDelivery(ctx, withChannel(d, notification.ChannelEmail, ""))
		if err != nil || !ok {
			return errors.Join(err, errors.New("e-mail not stored"))
		}
		if push, ok, err = repo.AddDelivery(ctx, withChannel(d, notification.ChannelPush, device)); err != nil || !ok {
			return errors.Join(err, errors.New("push not stored"))
		}
		for _, again := range []notification.Delivery{withChannel(d, notification.ChannelEmail, ""), withChannel(d, notification.ChannelPush, device)} {
			if _, ok, err := repo.AddDelivery(ctx, again); ok || err != nil {
				t.Errorf("%s again: stored %v, %v", again.Channel, ok, err)
			}
		}
		return nil
	})
	inTx(t, app, func(ctx context.Context) error {
		got, err := repo.LockDelivery(ctx, push.ID)
		if err != nil || got.DeviceID != device || got.State != notification.DeliveryPending || got.Channel != notification.ChannelPush || got.SourceID != reminderID {
			t.Errorf("LockDelivery() = %+v, %v", got, err)
		}
		if err := repo.SetDeliveryState(ctx, email.ID, notification.DeliverySent, at, 0); err != nil {
			return err
		}
		if err := repo.SetDeliveryState(ctx, email.ID, notification.DeliveryFailed, at, 0); !errors.Is(err, notification.ErrDeliveryDone) {
			t.Errorf("a sent delivery failed: %v, want ErrDeliveryDone", err)
		}
		if _, err := repo.LockDelivery(ctx, "not-a-uuid"); !errors.Is(err, notification.ErrNotFound) {
			t.Errorf("a malformed id: %v", err)
		}
		return nil
	})
	n, err := notification.NewDeliveryPurger(notification.NewPostgresDeliveryRepository(owner)).Purge(t.Context(), at.Add(time.Minute))
	if err != nil || n != 1 {
		t.Errorf("Purge() = %d, %v; want the sent e-mail", n, err)
	}
}

func withChannel(d notification.Delivery, ch notification.Channel, device string) notification.Delivery {
	d.Channel, d.DeviceID = ch, device
	return d
}

// The patient's settings are stored by account, quiet hours and zone included or not; a clinic's SMS usage adds up per
// month in its own scope.
func TestPreferencesAndSMSUsage_KeptByAccountAndByClinicMonth(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seedAccounts(t, owner)
	repo := notification.NewPostgresDeliveryRepository(app)
	at := time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)
	for _, p := range []notification.Preferences{
		{Channels: notification.Channels{Push: true, SMS: true}, Quiet: &notification.QuietHours{Start: "23:30", End: "07:05"}, TimeZone: "Asia/Dubai"},
		{Channels: notification.Channels{Email: true}},
	} {
		inTx(t, app, func(ctx context.Context) error {
			if _, ok, err := repo.Preferences(ctx, accountB); ok || err != nil {
				t.Errorf("another account's: %v, %v", ok, err)
			}
			if err := repo.SetPreferences(ctx, patientAccount, p, at); err != nil {
				return err
			}
			got, ok, err := repo.Preferences(ctx, patientAccount)
			if err != nil || !ok || got.Channels != p.Channels || got.TimeZone != p.TimeZone || (got.Quiet == nil) != (p.Quiet == nil) || got.Quiet != nil && *got.Quiet != *p.Quiet {
				t.Errorf("Preferences() = %+v, %v, %v; want %+v", got, ok, err, p)
			}
			return nil
		})
	}
	nov := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	chat := notification.NewPostgresRepository(app)
	for _, segments := range []int{1, 2} {
		inTx(t, app, func(ctx context.Context) error {
			if err := chat.SetClinic(ctx, clinicID); err != nil {
				return err
			}
			return repo.AddSMSUsage(ctx, clinicID, nov, segments)
		})
	}
	inTx(t, app, func(ctx context.Context) error {
		if err := chat.SetClinic(ctx, clinicB); err != nil {
			return err
		}
		if got, err := repo.SMSUsage(ctx, clinicID, nov.AddDate(0, -1, 0), nov); err != nil || len(got) != 0 {
			t.Errorf("clinic B sees clinic A's usage: %+v, %v", got, err)
		}
		return nil
	})
	inTx(t, app, func(ctx context.Context) error {
		if err := chat.SetClinic(ctx, clinicID); err != nil {
			return err
		}
		got, err := repo.SMSUsage(ctx, clinicID, nov.AddDate(0, -1, 0), nov)
		if want := []notification.UsageMonth{{Month: "2026-11", Messages: 2, Segments: 3}}; err != nil || !slices.Equal(got, want) {
			t.Errorf("SMSUsage() = %+v, %v; want %+v", got, err, want)
		}
		return nil
	})
}

// A delivery's send goes on its channel's queue at its time, the delivery's id only, once.
func TestRiverDeliveryJobs_EnqueueEachSendOnItsChannelsQueue(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	client, err := river.NewClient(riverpgxv5.New(app), &river.Config{})
	if err != nil {
		t.Fatal(err)
	}
	jobs := notification.NewRiverDeliveryJobs(client)
	at := time.Date(2030, 1, 1, 4, 0, 0, 0, time.UTC)
	for range 2 {
		inTx(t, app, func(ctx context.Context) error {
			for _, ch := range []notification.Channel{notification.ChannelEmail, notification.ChannelPush, notification.ChannelSMS} {
				if err := jobs.EnqueueDelivery(ctx, ch, reminderID, at); err != nil {
					return err
				}
			}
			return nil
		})
	}
	rows, err := owner.Query(t.Context(), "SELECT kind || ' ' || queue || ' ' || args::text, scheduled_at FROM river_job ORDER BY id")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var got []string
	for rows.Next() {
		var s string
		var when time.Time
		if err := rows.Scan(&s, &when); err != nil {
			t.Fatal(err)
		}
		if !when.Equal(at) {
			t.Errorf("%s at %v, want %v", s, when, at)
		}
		got = append(got, s)
	}
	args := `{"delivery_id": "` + reminderID + `"}`
	if want := []string{"reminder_email email " + args, "reminder_push push " + args, "reminder_sms sms " + args}; !slices.Equal(got, want) {
		t.Errorf("jobs %v,\nwant %v", got, want)
	}
	if err := jobs.EnqueueDelivery(t.Context(), notification.ChannelSMS, reminderID, at); !errors.Is(err, postgres.ErrNoTx) {
		t.Errorf("outside a transaction: %v", err)
	}
}
