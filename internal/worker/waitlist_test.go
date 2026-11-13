package worker_test

import (
	"strings"
	"testing"

	"github.com/riverqueue/river"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/consent"
)

// seedWaitlisted adds, as the owner, the clinic's waitlist (on), a second patient with a care consent waiting for the
// role from today for two weeks, and cancels the booking's future appointment, freeing its slot. It returns the
// patient and the entry.
func seedWaitlisted(t *testing.T, db logins, b booking) (patient, entry string) {
	t.Helper()
	ctx := t.Context()
	if _, err := db.owner.Exec(ctx, `INSERT INTO organization_appointment_settings (organization_id, waitlist_enabled) VALUES ($1, true)`, clinicID); err != nil {
		t.Fatal(err)
	}
	if err := db.owner.QueryRow(ctx, `WITH a AS (INSERT INTO account (email, password_hash, role, language) VALUES ('waiting@example.test', 'h', 'patient', 'en') RETURNING id),
		p AS (INSERT INTO patient (account_id, given_name, family_name, birth_date) SELECT id, 'Bo', 'Example', '1990-01-01' FROM a RETURNING id),
		c AS (INSERT INTO consent (patient_id, organization_id, status, date_time) SELECT id, $1, 'active', now() - interval '1 day' FROM p RETURNING patient_id),
		e AS (INSERT INTO waitlist_entry (patient_id, organization_id, practitioner_role_id, earliest, latest, ends_at)
		      SELECT c.patient_id, $1, a.practitioner_role_id, current_date, current_date + 14, current_date + 15 FROM c, appointment a WHERE a.id = $2
		      RETURNING patient_id, id)
		SELECT patient_id::text, id::text FROM e`, clinicID, b.future).Scan(&patient, &entry); err != nil {
		t.Fatal(err)
	}
	if _, err := db.owner.Exec(ctx, `UPDATE appointment SET status = 'cancelled', cancelled_at = now(), cancellation_reason = 'prov' WHERE id = $1`, b.future); err != nil {
		t.Fatal(err)
	}
	if _, err := db.owner.Exec(ctx, `UPDATE slot SET status = 'free' WHERE id = $1`, b.futureSlot); err != nil {
		t.Fatal(err)
	}
	return patient, entry
}

// A freed slot's job offers it to the waiting patient: the slot is held, the offer open with its expiry and notice
// enqueued; once the hold is over its expiry hands the slot on and, with nobody else waiting, frees it.
func TestSlotFreed_OffersTheSlotAndItsExpiryHandsItOn(t *testing.T) {
	t.Parallel()
	db := newLogins(t)
	b := seedBooking(t, db.owner)
	_, entry := seedWaitlisted(t, db, b)
	if ended, logs := runWorkerJob(t, db, appointment.SlotFreedArgs{SlotID: b.futureSlot, OrganizationID: clinicID}); ended != river.EventKindJobCompleted {
		t.Fatalf("the slot's job ended %s:\n%s", ended, logs)
	}
	var offer, state, slotStatus string
	if err := db.owner.QueryRow(t.Context(), `SELECT o.id::text, o.state, s.status FROM waitlist_offer o JOIN slot s ON s.id = o.slot_id
		WHERE o.entry_id = $1`, entry).Scan(&offer, &state, &slotStatus); err != nil {
		t.Fatal(err)
	}
	if state != "open" || slotStatus != "busy-tentative" {
		t.Fatalf("offer %s, slot %s; want it open and the slot held", state, slotStatus)
	}
	if n := countWhere(t, db, `SELECT count(*) FROM river_job WHERE kind = 'waitlist_offer_expire' AND queue = 'default' AND scheduled_at > now()`); n != 1 {
		t.Errorf("%d expiries of the offer, want one at its end", n)
	}
	// Its notice: SAL-33's deliveries, stored in the offer's transaction (the default settings: e-mail, no device).
	if n := countWhere(t, db, `SELECT count(*) FROM notification_delivery WHERE kind = 'waitlist_offer' AND source_id = $1 AND channel = 'email'`, offer); n != 1 {
		t.Errorf("%d e-mail deliveries of the offer, want one", n)
	}
	if _, err := db.owner.Exec(t.Context(), `UPDATE waitlist_offer SET created_at = now() - interval '20 minutes', expires_at = now() - interval '1 minute'
		WHERE id = $1`, offer); err != nil {
		t.Fatal(err)
	}
	if ended, logs := runWorkerJob(t, db, appointment.OfferExpireArgs{OfferID: offer, OrganizationID: clinicID}); ended != river.EventKindJobCompleted {
		t.Fatalf("the expiry ended %s:\n%s", ended, logs)
	}
	if err := db.owner.QueryRow(t.Context(), `SELECT o.state, s.status FROM waitlist_offer o JOIN slot s ON s.id = o.slot_id WHERE o.id = $1`, offer).
		Scan(&state, &slotStatus); err != nil {
		t.Fatal(err)
	}
	if state != "expired" || slotStatus != "free" {
		t.Errorf("offer %s, slot %s; want it expired and the slot free again", state, slotStatus)
	}
}

// The revocation of the care consent withdraws the patient's entries there, and logs the count only.
func TestConsentRevoked_WithdrawsTheWaitlistEntries(t *testing.T) {
	t.Parallel()
	db := newLogins(t)
	b := seedBooking(t, db.owner)
	patient, entry := seedWaitlisted(t, db, b)
	if _, err := db.owner.Exec(t.Context(), `UPDATE consent SET status = 'inactive', revoked_at = now() WHERE patient_id = $1`, patient); err != nil {
		t.Fatal(err)
	}
	ended, logs := runWorkerJob(t, db, consent.RevokedArgs{PatientID: patient, OrganizationID: clinicID})
	if ended != river.EventKindJobCompleted || !strings.Contains(logs, `"withdrawn":1`) || strings.Contains(logs, "waiting@example.test") {
		t.Fatalf("the revocation ended %s:\n%s", ended, logs)
	}
	if n := countWhere(t, db, `SELECT count(*) FROM waitlist_entry WHERE id = $1 AND status = 'withdrawn'`, entry); n != 1 {
		t.Error("the entry is still active")
	}
}
