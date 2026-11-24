package notification_test

import (
	"testing"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/notification"
	"github.com/Shmeeshmee/salus/pkg/push"
)

// Closing an account removes every push device of it (docs/clinic-cms/WAVE5.md S5), audited once with the number
// removed; a second call removes nothing and audits nothing.
func TestRemoveDevices_RemovesEveryDeviceOfTheAccount(t *testing.T) {
	f := newChatFixture(t)
	for _, token := range []string{"synthetic-token-1", "synthetic-token-2"} {
		if err := f.svc.RegisterDevice(t.Context(), patientCaller, push.Android, token); err != nil {
			t.Fatal(err)
		}
	}
	other := notification.Caller{AccountID: "a9000000-0000-4000-8000-0000000000a9"}
	if err := f.svc.RegisterDevice(t.Context(), other, push.IOS, "synthetic-token-3"); err != nil {
		t.Fatal(err)
	}
	f.audit.events = nil

	n, err := f.svc.RemoveDevices(t.Context(), patientAccount)
	if err != nil || n != 2 {
		t.Fatalf("RemoveDevices() = %d, %v; want 2", n, err)
	}
	if devices, _ := f.repo.Devices(t.Context(), patientAccount); len(devices) != 0 {
		t.Errorf("devices %+v left", devices)
	}
	if devices, _ := f.repo.Devices(t.Context(), other.AccountID); len(devices) != 1 {
		t.Errorf("the other account's devices %+v, want its one", devices)
	}
	if len(f.audit.events) != 1 || f.audit.events[0].Action != auditevent.ActionDelete || f.audit.events[0].EntityCount != 2 ||
		f.audit.events[0].EntityReference != "Account/"+patientAccount {
		t.Errorf("audit events %+v, want one delete of 2 devices", f.audit.events)
	}
	if n, err := f.svc.RemoveDevices(t.Context(), patientAccount); err != nil || n != 0 || len(f.audit.events) != 1 {
		t.Errorf("a second RemoveDevices() = %d, %v with %d events; want nothing", n, err, len(f.audit.events))
	}
}
