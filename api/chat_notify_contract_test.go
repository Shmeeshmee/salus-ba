package api_test

import (
	"maps"
	"slices"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/notification"
	"github.com/Shmeeshmee/salus/internal/organization"
)

// SAL-29's schemas are what the Go types encode (docs/plans/chat.md "Contract of wave 1"): every key, and as required
// every key a value always has; a request's required keys are the contract's.
func TestChatNotifications_SchemasMatchTheGoTypes(t *testing.T) {
	t.Parallel()
	s := schemas(t)
	at := time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)
	hour := organization.ChatHour{Day: "mon", Start: "08:00", End: "16:00"}
	endpoint := notification.Endpoint{CreatedAt: at, ID: "x", URL: "https://x.example.test", Events: []string{"message.created"}}
	for _, tt := range []struct {
		name          string
		full, minimal any
		required      []string // a request's, when it differs from what the type always encodes
	}{
		{"ChatHour", hour, organization.ChatHour{}, nil},
		{"ChatSettings", organization.ChatSettings{Hours: []organization.ChatHour{hour}}, organization.ChatSettings{}, nil},
		{"ChatHours", organization.ChatHours{TimeZone: "Europe/Sarajevo", Hours: []organization.ChatHour{hour}}, organization.ChatHours{}, nil},
		{"Webhook", endpoint, notification.Endpoint{}, nil},
		{"CreatedWebhook", notification.CreatedWebhook{CreatedAt: at, Secret: "whsec_x"}, notification.CreatedWebhook{}, nil},
		{"NewWebhook", notification.NewEndpoint{URL: "https://x.example.test"}, notification.NewEndpoint{}, []string{"events", "url"}},
	} {
		sc, ok := s[tt.name]
		if !ok {
			t.Errorf("%s: no schema", tt.name)
			continue
		}
		if props, want := slices.Sorted(maps.Keys(sc.Properties)), keys(t, tt.full); !slices.Equal(props, want) {
			t.Errorf("%s: properties %v, Go encodes %v", tt.name, props, want)
		}
		want := keys(t, tt.minimal)
		if tt.required != nil {
			want = tt.required
		}
		if got := slices.Sorted(slices.Values(s[tt.name].Required)); !slices.Equal(got, want) {
			t.Errorf("%s: required %v, want %v", tt.name, got, want)
		}
	}
	if got := slices.Sorted(maps.Keys(s["PushDevice"].Properties)); !slices.Equal(got, []string{"platform", "token"}) {
		t.Errorf("PushDevice: properties %v", got)
	}
	if got := s["PushDevice"].Properties["platform"].Enum; !slices.Equal(got, []string{"android", "ios"}) {
		t.Errorf("platform enum %v", got)
	}
	if got := s["ChatHour"].Properties["day"].Enum; !slices.Equal(got, []string{"mon", "tue", "wed", "thu", "fri", "sat", "sun"}) {
		t.Errorf("day enum %v", got)
	}
	if got := s["WebhookEventType"].Enum; !slices.Equal(got, []string{notification.EventMessageCreated}) {
		t.Errorf("event types %v", got)
	}
}
