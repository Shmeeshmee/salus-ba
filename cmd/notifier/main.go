// Command notifier sends e-mail (ADR 0007): it runs a River client that works the email queue, through SMTP:
// Mailpit locally, an EU relay with TLS and a login in production (ADR 0011); and the chat's push queue (SAL-29),
// through FCM and APNs once their credentials are set (docs/deploy/push.md), the fake until then. It logs in as
// salus_app.
package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata" // the clinics' time zones, independent of the runtime image

	"github.com/Shmeeshmee/salus/internal/config"
	"github.com/Shmeeshmee/salus/internal/notification"
	"github.com/Shmeeshmee/salus/internal/worker"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/push"
	"github.com/Shmeeshmee/salus/pkg/push/apns"
	"github.com/Shmeeshmee/salus/pkg/push/fcm"
	"github.com/Shmeeshmee/salus/pkg/push/pushtest"
)

// stopTimeout is how long a running send gets to finish after a stop signal.
const stopTimeout = 30 * time.Second

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	err := run(ctx, os.Stderr)
	stop()
	if err != nil {
		fmt.Fprintln(os.Stderr, "notifier:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, logOut io.Writer) error {
	log := slog.New(slog.NewJSONHandler(logOut, nil))
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	relay, err := config.LoadSMTP()
	if err != nil {
		return err
	}
	pushCfg, err := config.LoadPush()
	if err != nil {
		return err
	}
	pusher, engines, err := newPusher(pushCfg)
	if err != nil {
		return err
	}
	pool, err := postgres.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	sender := notification.NewSMTPSender(relay.Addr, notification.SMTPSecurity(relay.TLS), relay.Username, relay.Password, relay.From, clock.System{})
	client, err := worker.NewNotifier(pool, log, sender, pusher, cfg.WebBaseURL, cfg.PatientAppURL)
	if err != nil {
		return err
	}
	// River stops hard when its start context ends; the signal triggers a soft Stop below instead.
	if err := client.Start(context.WithoutCancel(ctx)); err != nil {
		return err
	}
	log.Info("started", "push_android", engines[push.Android], "push_ios", engines[push.IOS])

	<-ctx.Done()
	log.Info("shutting down")
	stopCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), stopTimeout)
	defer cancel()
	if err := client.Stop(stopCtx); err != nil {
		return err
	}
	log.Info("stopped")
	return nil
}

// pushTimeout bounds one request to a push service.
const pushTimeout = 10 * time.Second

// newPusher is the push sender of each platform: FCM for Android and APNs for iOS once their credentials are set,
// the fake (it sends nothing) until then; and which each is, for the start's log line.
func newPusher(cfg config.Push) (push.Sender, map[push.Platform]string, error) {
	fake := &pushtest.Sender{}
	senders := push.ByPlatform{push.Android: fake, push.IOS: fake}
	engines := map[push.Platform]string{push.Android: "fake", push.IOS: "fake"}
	if cfg.FCMCredentials != nil {
		c, err := fcm.New(cfg.FCMCredentials, &http.Client{Timeout: pushTimeout}, clock.System{})
		if err != nil {
			return nil, nil, err
		}
		senders[push.Android], engines[push.Android] = c, "fcm"
	}
	if a := cfg.APNs; a != nil {
		hc := &http.Client{Timeout: pushTimeout, Transport: &http.Transport{ForceAttemptHTTP2: true, TLSHandshakeTimeout: pushTimeout}}
		c, err := apns.New(apns.Config{Key: a.Key, KeyID: a.KeyID, TeamID: a.TeamID, BundleID: a.BundleID, Sandbox: a.Sandbox}, hc, clock.System{})
		if err != nil {
			return nil, nil, err
		}
		senders[push.IOS], engines[push.IOS] = c, "apns"
	}
	return senders, engines, nil
}
