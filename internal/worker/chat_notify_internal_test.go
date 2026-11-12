package worker

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/Shmeeshmee/salus/internal/notification"
)

// A webhook whose endpoint is gone is cancelled, not retried; any other failure goes back to River with the job's
// attempt and its maximum, which decide the dead letter. An unread e-mail to an address SMTP cannot carry is
// cancelled.
func TestChatWorkers_CancelWhatARetryCannotDo(t *testing.T) {
	var attempt, maxAttempts int
	for name, tc := range map[string]struct {
		err    error
		cancel bool
	}{
		"endpoint gone": {fmt.Errorf("notification: webhook: %w", notification.ErrNotFound), true},
		"no answer":     {errors.New("notification: webhook: status 503"), false},
	} {
		w := &webhookWorker{deliver: func(_ context.Context, _ notification.WebhookDeliveryArgs, a, m int) error {
			attempt, maxAttempts = a, m
			return tc.err
		}}
		err := w.Work(t.Context(), &river.Job[notification.WebhookDeliveryArgs]{JobRow: &rivertype.JobRow{Attempt: 3, MaxAttempts: 8}})
		var cancel *rivertype.JobCancelError
		if !errors.Is(err, tc.err) || errors.As(err, &cancel) != tc.cancel || attempt != 3 || maxAttempts != 8 {
			t.Errorf("%s: Work() = %v (attempt %d of %d), want cancelled %v", name, err, attempt, maxAttempts, tc.cancel)
		}
	}
	for name, tc := range map[string]struct {
		err    error
		cancel bool
	}{
		"not ASCII": {fmt.Errorf("notification: unread e-mail: %w", notification.ErrRecipientNotASCII), true},
		"smtp down": {errors.New("notification: smtp down"), false},
	} {
		w := &unreadEmailWorker{send: func(context.Context, notification.ChatUnreadEmailArgs) error { return tc.err }}
		err := w.Work(t.Context(), &river.Job[notification.ChatUnreadEmailArgs]{JobRow: &rivertype.JobRow{}})
		var cancel *rivertype.JobCancelError
		if !errors.Is(err, tc.err) || errors.As(err, &cancel) != tc.cancel {
			t.Errorf("%s: Work() = %v, want cancelled %v", name, err, tc.cancel)
		}
	}
}
