package worker

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/notification"
)

// An appointment e-mail to an address SMTP cannot carry is cancelled like one whose appointment is gone; any
// other failure is returned for River to retry.
func TestAppointmentEmailWorker_CancelsWhatARetryCannotSend(t *testing.T) {
	for name, tc := range map[string]struct {
		err    error
		cancel bool
	}{
		"no such appointment":    {fmt.Errorf("notification: booked: %w", appointment.ErrNotFound), true},
		"an address not ASCII":   {fmt.Errorf("notification: booked: %w", notification.ErrRecipientNotASCII), true},
		"the server unreachable": {errors.New("notification: smtp: connection refused"), false},
	} {
		t.Run(name, func(t *testing.T) {
			w := &appointmentEmailWorker[appointment.BookedEmailArgs]{send: func(context.Context, appointment.BookedEmailArgs) error { return tc.err }}

			err := w.Work(t.Context(), &river.Job[appointment.BookedEmailArgs]{JobRow: &rivertype.JobRow{}})

			var cancel *rivertype.JobCancelError
			if !errors.Is(err, tc.err) || errors.As(err, &cancel) != tc.cancel {
				t.Errorf("Work() = %v, want %v, cancelled %v", err, tc.err, tc.cancel)
			}
		})
	}
}
