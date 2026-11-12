// Package push is the port of mobile push notifications (docs/plans/chat.md, SAL-29): a Message to one device
// token, sent by the platform's service (fcm for Android, apns for iOS). pushtest has the in-memory fake that is
// the default until the owner adds the platforms' credentials (docs/deploy/push.md). It knows nothing of chats or
// patients: a caller writes the title and body, and they never hold message text, names or anything clinical, as
// Google and Apple are processors outside the EU/EEA (docs/privacy/processors.md). Nothing here logs; errors never
// carry a token, a title or a body.
package push

import (
	"context"
	"errors"
	"fmt"
)

// Platform is the device's push service.
type Platform string

// The platforms: Firebase Cloud Messaging for Android, the Apple Push Notification service for iOS.
const (
	Android Platform = "android"
	IOS     Platform = "ios"
)

// ValidPlatform tells whether p is one of the platforms.
func ValidPlatform(p Platform) bool { return p == Android || p == IOS }

// Message is one notification to one device: its token and platform, and the alert's title and body.
type Message struct {
	Token    string
	Platform Platform
	Title    string
	Body     string
}

// Sender delivers a Message.
type Sender interface {
	Send(ctx context.Context, m Message) error
}

var (
	// ErrTokenGone is returned when the platform says the token no longer reaches a device (the app was
	// uninstalled, or the token expired or is not one): the caller forgets it.
	ErrTokenGone = errors.New("push: the device token is gone")
	// ErrPlatform is returned for a platform the sender has no service for.
	ErrPlatform = errors.New("push: no sender for the platform")
)

// ByPlatform sends each message through its platform's sender.
type ByPlatform map[Platform]Sender

// Send hands m to its platform's sender, or returns ErrPlatform.
func (b ByPlatform) Send(ctx context.Context, m Message) error {
	s, ok := b[m.Platform]
	if !ok || s == nil {
		return fmt.Errorf("%w: %q", ErrPlatform, string(m.Platform))
	}
	return s.Send(ctx, m)
}
