package notification

import (
	"context"
	"fmt"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/pkg/push"
)

// maxToken bounds a device token (FCM's are about 160 characters, APNs' 64).
const maxToken = 4096

// RegisterDevice stores the push token of the patient's app on the device, as the signed-in account's (moving it
// from any other account the device was signed into), audited on the account. Staff get ErrForbidden: the app is
// the patients'. ErrInvalid for a platform that is not android or ios, or a token that is not 1 to 4096 visible
// ASCII characters.
func (s *Chat) RegisterDevice(ctx context.Context, c Caller, p push.Platform, token string) error {
	if err := deviceCaller(c, p, token); err != nil {
		return err
	}
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.Repo.RegisterDevice(ctx, Device{AccountID: c.AccountID, Token: token, Platform: p}, s.Clock.Now()); err != nil {
			return err
		}
		return s.recordDevice(ctx, c, auditevent.ActionUpdate, auditevent.CodeUpdate)
	})
	if err != nil {
		return fmt.Errorf("notification: register device: %w", err)
	}
	return nil
}

// UnregisterDevice deletes the account's device of the token (the app's sign-out), audited on the account; nothing
// for a token it does not have. Refusals as RegisterDevice's.
func (s *Chat) UnregisterDevice(ctx context.Context, c Caller, p push.Platform, token string) error {
	if err := deviceCaller(c, p, token); err != nil {
		return err
	}
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.Repo.UnregisterDevice(ctx, c.AccountID, token); err != nil {
			return err
		}
		return s.recordDevice(ctx, c, auditevent.ActionDelete, auditevent.CodeDelete)
	})
	if err != nil {
		return fmt.Errorf("notification: unregister device: %w", err)
	}
	return nil
}

// deviceCaller is nil for a signed-in patient with a valid platform and token.
func deviceCaller(c Caller, p push.Platform, token string) error {
	switch {
	case c.AccountID == "":
		return ErrNotFound
	case c.OrganizationID != "":
		return ErrForbidden
	case !push.ValidPlatform(p) || len(token) == 0 || len(token) > maxToken:
		return ErrInvalid
	}
	for i := range len(token) {
		if token[i] <= ' ' || token[i] > '~' {
			return ErrInvalid
		}
	}
	return nil
}

// recordDevice audits c's write of their own device.
func (s *Chat) recordDevice(ctx context.Context, c Caller, action auditevent.Action, interaction string) error {
	return s.Audit.Record(ctx, auditevent.AuditEvent{
		Type:            auditevent.Coding{System: auditevent.SystemAuditEventType, Code: auditevent.CodeRest},
		Subtype:         auditevent.Coding{System: auditevent.SystemRestfulInteraction, Code: interaction},
		Purpose:         auditevent.Coding{System: auditevent.SystemActReason, Code: auditevent.CodePATRQT},
		Action:          action,
		Outcome:         auditevent.OutcomeSuccess,
		AgentAccountID:  c.AccountID,
		EntityReference: "Account/" + c.AccountID,
		EntityName:      "push_device",
		EntityCount:     1,
	})
}
