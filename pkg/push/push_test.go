package push_test

import (
	"errors"
	"testing"

	"github.com/Shmeeshmee/salus/pkg/push"
	"github.com/Shmeeshmee/salus/pkg/push/pushtest"
)

// ByPlatform hands each message to its platform's sender and refuses a platform it has none for.
func TestByPlatform_SendsThroughThePlatformsSender(t *testing.T) {
	android, ios := &pushtest.Sender{}, &pushtest.Sender{}
	s := push.ByPlatform{push.Android: android, push.IOS: ios}

	for _, m := range []push.Message{
		{Platform: push.Android, Token: "synthetic-android-token", Title: "t", Body: "b"},
		{Platform: push.IOS, Token: "synthetic-ios-token", Title: "t", Body: "b"},
	} {
		if err := s.Send(t.Context(), m); err != nil {
			t.Fatalf("Send(%s) = %v", m.Platform, err)
		}
	}
	if got := android.Sent(); len(got) != 1 || got[0].Token != "synthetic-android-token" {
		t.Errorf("android sent %+v", got)
	}
	if got := ios.Sent(); len(got) != 1 || got[0].Token != "synthetic-ios-token" {
		t.Errorf("ios sent %+v", got)
	}
	if err := s.Send(t.Context(), push.Message{Platform: "web", Token: "x"}); !errors.Is(err, push.ErrPlatform) {
		t.Errorf("Send(web) = %v, want ErrPlatform", err)
	}
}

func TestValidPlatform(t *testing.T) {
	for p, want := range map[push.Platform]bool{push.Android: true, push.IOS: true, "": false, "web": false, "Android": false} {
		if got := push.ValidPlatform(p); got != want {
			t.Errorf("ValidPlatform(%q) = %v, want %v", p, got, want)
		}
	}
}
