package pushtest_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Shmeeshmee/salus/pkg/push"
	"github.com/Shmeeshmee/salus/pkg/push/pushtest"
)

// The fake records what it is sent, answers ErrTokenGone for the tokens marked gone and Err otherwise when set, and
// keeps at most Keep messages (a long-running process that uses it as the default must not grow without bound).
func TestFake_RecordsAndFailsAsTold(t *testing.T) {
	f := &pushtest.Sender{Keep: 2}
	f.Gone("synthetic-gone")
	ctx := context.Background()
	if err := f.Send(ctx, push.Message{Token: "synthetic-gone"}); !errors.Is(err, push.ErrTokenGone) {
		t.Errorf("Send(gone) = %v, want ErrTokenGone", err)
	}
	for _, tok := range []string{"a", "b", "c"} {
		if err := f.Send(ctx, push.Message{Token: tok}); err != nil {
			t.Fatal(err)
		}
	}
	if got := f.Sent(); len(got) != 2 || got[0].Token != "b" || got[1].Token != "c" {
		t.Errorf("Sent() = %+v, want the last two", got)
	}
	down := errors.New("down")
	f.Err = down
	if err := f.Send(ctx, push.Message{Token: "d"}); !errors.Is(err, down) {
		t.Errorf("Send() = %v, want Err", err)
	}
}
