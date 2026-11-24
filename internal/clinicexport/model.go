// Package clinicexport is the clinic's data export (docs/clinic-cms/WAVE4.md, M2): the admin asks, a job builds one
// encrypted archive in the object store, the admin downloads it until it expires, and the retention job deletes it.
package clinicexport

import "time"

// Status is where an export stands (api/openapi.yaml ClinicExportStatus).
type Status string

// The statuses. Expired is never stored: a ready export past ExpiresAt reads as it.
const (
	StatusQueued  Status = "queued"
	StatusReady   Status = "ready"
	StatusFailed  Status = "failed"
	StatusExpired Status = "expired"
)

// Window is how long after the request an export can be downloaded; the retention job deletes it after.
const Window = 7 * 24 * time.Hour

// Export is one export of a clinic. WrappedKey and KEKID are the archive's data key, wrapped by the key-encryption
// key and bound to the export's id; Size is the archive's plaintext bytes. All three are set once it is ready.
type Export struct {
	RequestedAt    time.Time
	ExpiresAt      time.Time
	ReadyAt        time.Time // zero until ready
	ID             string
	OrganizationID string
	RequestedBy    string // the admin's account
	Status         Status
	KEKID          string
	WrappedKey     []byte
	Size           int64
}

// at is the export as it reads at now: a ready one past its expiry is expired.
func (e Export) at(now time.Time) Export {
	if e.Status == StatusReady && !now.Before(e.ExpiresAt) {
		e.Status = StatusExpired
	}
	return e
}

// Caller is who asks: the signed-in account, its clinic and whether it is that clinic's admin.
type Caller struct {
	AccountID      string
	OrganizationID string
	Admin          bool
}

// BuildArgs is the job that builds an export's archive: opaque ids only.
type BuildArgs struct {
	ExportID       string `json:"export_id"`
	OrganizationID string `json:"organization_id"`
}

// Kind names the job in the river_job table.
func (BuildArgs) Kind() string { return "clinic_export_build" }

// ObjectKey is where the archive of the export id is stored.
func ObjectKey(id string) string { return "clinicexport/" + id }
