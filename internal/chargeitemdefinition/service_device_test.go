package chargeitemdefinition_test

import (
	"errors"
	"testing"

	"github.com/Shmeeshmee/salus/internal/chargeitemdefinition"
)

// A service may need a device of its clinic that is in use (U4 of docs/clinic-cms/WAVE6.md): an unknown, another
// clinic's or an inactive one is ErrInvalid, on a new service and on a change, and nothing is stored.
func TestSave_TakesOnlyADeviceInUseOfTheClinic(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		want   error
		device string
	}{
		"none":                {device: ""},
		"a device in use":     {device: deviceInUse},
		"a device not in use": {device: deviceUnused, want: chargeitemdefinition.ErrInvalid},
		"an unknown device":   {device: "a8000000-0000-4000-8000-0000000000ff", want: chargeitemdefinition.ErrInvalid},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			repo, devices := &fakeRepository{}, &fakeDevices{}
			svc := newServiceWith(repo, &fakeAuditor{}, devices)
			d := exam()
			d.DeviceID = tc.device
			got, err := svc.Create(t.Context(), admin, d)
			if !errors.Is(err, tc.want) {
				t.Fatalf("Create() = %+v, %v; want %v", got, err, tc.want)
			}
			if tc.want != nil && len(repo.rows) != 0 {
				t.Errorf("stored %+v after a refusal", repo.rows)
			}
			if devices.outside {
				t.Error("the device was read outside the save's transaction")
			}
			stored := newServiceWith(&fakeRepository{rows: []chargeitemdefinition.Definition{{ID: "d0000000-0000-4000-8000-000000000001", OrganizationID: clinicA, VersionID: 2}}}, &fakeAuditor{}, devices)
			d.ID, d.VersionID = "d0000000-0000-4000-8000-000000000001", 2
			if _, err := stored.Update(t.Context(), admin, d); !errors.Is(err, tc.want) {
				t.Errorf("Update() = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestSave_ReturnsTheDevicesError(t *testing.T) {
	t.Parallel()
	d := exam()
	d.DeviceID = deviceInUse
	if _, err := newServiceWith(&fakeRepository{}, &fakeAuditor{}, &fakeDevices{err: errBroken}).Create(t.Context(), admin, d); !errors.Is(err, errBroken) {
		t.Errorf("Create() = %v, want the devices' error", err)
	}
	if _, err := newServiceWith(&fakeRepository{}, &fakeAuditor{}, nil).Create(t.Context(), admin, d); !errors.Is(err, chargeitemdefinition.ErrInvalid) {
		t.Errorf("Create() without the devices port = %v, want ErrInvalid", err)
	}
}
