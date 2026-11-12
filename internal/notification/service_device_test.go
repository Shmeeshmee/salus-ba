package notification_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/notification"
	"github.com/Shmeeshmee/salus/pkg/push"
)

var patientCaller = notification.Caller{AccountID: patientAccount}

// The patient's app registers its token (again on every start: the time moves) and unregisters it on sign-out; a
// token registered by another account moves to it. Each write is audited on the account.
func TestRegisterDevice_StoresTheTokenOfThePatientsAppAndUnregisterDeletesIt(t *testing.T) {
	f := newChatFixture(t)
	if err := f.svc.RegisterDevice(t.Context(), patientCaller, push.IOS, "synthetic-token-1"); err != nil {
		t.Fatalf("RegisterDevice() = %v", err)
	}
	devices, _ := f.repo.Devices(t.Context(), patientAccount)
	if len(devices) != 1 || devices[0].Token != "synthetic-token-1" || devices[0].Platform != push.IOS || !f.repo.registers[0].Equal(chatNow) {
		t.Fatalf("devices %+v", devices)
	}
	other := notification.Caller{AccountID: "a9000000-0000-4000-8000-0000000000a9"}
	if err := f.svc.RegisterDevice(t.Context(), other, push.IOS, "synthetic-token-1"); err != nil {
		t.Fatal(err)
	}
	if devices, _ := f.repo.Devices(t.Context(), patientAccount); len(devices) != 0 {
		t.Errorf("the token stayed with the first account: %+v", devices)
	}
	if err := f.svc.UnregisterDevice(t.Context(), other, push.IOS, "synthetic-token-1"); err != nil {
		t.Fatal(err)
	}
	if len(f.repo.devices) != 0 {
		t.Errorf("devices %+v after unregistering", f.repo.devices)
	}
	if len(f.audit.events) != 3 || f.audit.events[0].EntityReference != "Account/"+patientAccount || f.audit.events[0].EntityName != "push_device" ||
		f.audit.events[0].AgentAccountID != patientAccount || f.audit.events[2].Action != auditevent.ActionDelete {
		t.Errorf("audit %+v", f.audit.events)
	}
}

// The app is the patients' (CLAUDE.md "Product"): staff register nothing. A platform or token the contract refuses
// is invalid, and the error never holds the token.
func TestRegisterDevice_RefusesStaffAndInvalidTokens(t *testing.T) {
	f := newChatFixture(t)
	staff := notification.Caller{AccountID: practAccount, OrganizationID: clinicID}
	if err := f.svc.RegisterDevice(t.Context(), staff, push.Android, "synthetic-token"); !errors.Is(err, notification.ErrForbidden) {
		t.Errorf("staff: %v, want ErrForbidden", err)
	}
	for name, tc := range map[string]struct {
		platform push.Platform
		token    string
	}{
		"web":         {"web", "synthetic-token"},
		"empty":       {push.Android, ""},
		"a space":     {push.Android, "synthetic token"},
		"a newline":   {push.Android, "synthetic\ntoken"},
		"not ASCII":   {push.Android, "synthetic-tökén"},
		"over 4096":   {push.Android, strings.Repeat("a", 4097)},
		"signed out":  {push.Android, ""},
		"caller none": {push.Android, "x"},
	} {
		c := patientCaller
		if name == "caller none" {
			c = notification.Caller{}
		}
		err := f.svc.RegisterDevice(t.Context(), c, tc.platform, tc.token)
		if err == nil || (name != "caller none" && !errors.Is(err, notification.ErrInvalid)) || (tc.token != "" && strings.Contains(err.Error(), tc.token)) {
			t.Errorf("%s: RegisterDevice() = %v", name, err)
		}
	}
	if len(f.repo.devices) != 0 {
		t.Errorf("stored %+v", f.repo.devices)
	}
}
