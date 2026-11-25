package appointment

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

// deviceNoOverlap keeps two visits off one device (00080; U4 of docs/clinic-cms/WAVE6.md, RM1).
const deviceNoOverlap = "appointment_device_no_overlap"

// clashed is the refusal of a write the database aborted on an overlap: ErrResourceBusy on the device's constraint,
// ErrSlotUnavailable on the role's (or a deadlock of two concurrent bookings of it); nil for any other error.
func clashed(err error) error {
	var pgErr *pgconn.PgError
	switch {
	case !errors.As(err, &pgErr):
		return nil
	case pgErr.Code == exclusionViolation && pgErr.ConstraintName == deviceNoOverlap:
		return ErrResourceBusy
	case pgErr.Code == exclusionViolation || pgErr.Code == deadlockDetected:
		return ErrSlotUnavailable
	}
	return nil
}
