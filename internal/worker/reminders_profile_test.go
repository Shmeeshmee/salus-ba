package worker_test

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/schedule"
	"github.com/Shmeeshmee/salus/internal/slot"
)

// docs/clinic-cms/WAVE4.md R3: TestReminder_SentOnceAtTheRightInstant's cases for a patient the clinic registered
// (no account) at a clinic in Europe/Sarajevo that offers SMS, around the October change there (25 October 2026): the
// SMS goes at its absolute due time, held by the default quiet hours (22:00 to 08:00) in the clinic's zone until they
// end, or dropped when that is too close to the start; the first dispatch is rolled back and every job runs twice, so
// one SMS goes, counted once in the clinic's usage, with the phone read for the send only (audited with the send's job
// as agent). A visit cancelled or moved sends nothing more; without the SMS option or a phone, nothing goes. A visit
// the desk moves (M1) gets its reminders again for the new time, and the same cases hold for them.
func TestReminder_APatientWithoutAnAccountGetsOneSMSAtTheRightInstant(t *testing.T) {
	t.Parallel()
	s := newReminderStack(t)
	b := seedBooking(t, s.owner)
	ctx := t.Context()
	var role string
	if err := s.owner.QueryRow(ctx, `SELECT practitioner_role_id::text FROM appointment WHERE id = $1`, b.future).Scan(&role); err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{
		`UPDATE patient SET account_id = NULL, registered_by_organization_id = $2, phone = '+38761000009' WHERE id = $1`,
		`INSERT INTO organization_appointment_settings (organization_id, sms_reminders) SELECT $2, true WHERE $1::uuid IS NOT NULL
			ON CONFLICT (organization_id) DO UPDATE SET sms_reminders = true`,
	} {
		if _, err := s.owner.Exec(ctx, q, b.patient, clinicID); err != nil {
			t.Fatal(err)
		}
	}

	late := checkProfileAtHome(t, s, b, role)
	checkProfileSentOnce(t, s, b, late)
	checkProfileTooEarly(t, s, b, role)
	checkProfileCancelledAndMoved(t, s, b, role)
	checkProfileNoOptionNoPhone(t, s, b, role)
	checkProfileMovedByStaff(t, s, b, role)
}

// checkProfileAtHome dispatches the reminders of a visit at 22:00 CET on 25 October, the first dispatch rolled back.
func checkProfileAtHome(t *testing.T, s *reminderStack, b booking, role string) visit {
	t.Helper()
	late := s.seedVisit(t, role, b.patient, time.Date(2026, 10, 25, 21, 0, 0, 0, time.UTC))
	// The day before, 21:00 UTC is 23:00 CEST: quiet until 08:00 the next morning, which after the change is 07:00 UTC.
	s.jobs.failed = false
	dayBefore := time.Date(2026, 10, 24, 21, 0, 0, 0, time.UTC)
	if err := s.dispatch(t, late, 1440, dayBefore); err == nil {
		t.Fatal("the first dispatch did not fail")
	}
	if st := s.state(t, late.reminders[1440]); st != "scheduled" {
		t.Fatalf("after the rolled-back dispatch: %s, want scheduled", st)
	}
	for range 2 {
		if err := s.dispatch(t, late, 1440, dayBefore); err != nil {
			t.Fatal(err)
		}
	}
	if jobs, _ := s.sends(t, late.reminders[1440]); !slices.Equal(jobs, []string{"reminder_sms at 2026-10-25T07:00Z"}) {
		t.Errorf("the day before: %v, want the SMS at 07:00 UTC", jobs)
	}
	// An hour before, 20:00 UTC is 21:00 CET: before the quiet hours, at once.
	if err := s.dispatch(t, late, 60, time.Date(2026, 10, 25, 20, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	if jobs, _ := s.sends(t, late.reminders[60]); !slices.Equal(jobs, []string{"reminder_sms at 2026-10-25T20:00Z"}) {
		t.Errorf("an hour before: %v, want the SMS at once", jobs)
	}
	var withAccount int
	if err := s.owner.QueryRow(t.Context(), `SELECT count(*) FROM notification_delivery WHERE source_id = ANY($1::uuid[]) AND (account_id IS NOT NULL OR patient_id <> $2)`,
		[]string{late.reminders[1440], late.reminders[60]}, b.patient).Scan(&withAccount); err != nil || withAccount != 0 {
		t.Errorf("%d deliveries not to the patient (%v), want none", withAccount, err)
	}
	return late
}

// checkProfileSentOnce runs the day-before SMS's send twice: one SMS to the phone, no link, counted once.
func checkProfileSentOnce(t *testing.T, s *reminderStack, b booking, late visit) {
	t.Helper()
	ctx := t.Context()
	s.sendAll(t, late.reminders[1440], time.Date(2026, 10, 25, 7, 0, 0, 0, time.UTC))
	if _, deliveries := s.sends(t, late.reminders[1440]); !slices.Equal(deliveries, []string{"sms sent"}) {
		t.Errorf("deliveries %v, want the SMS sent", deliveries)
	}
	sent := s.sms.Sent()
	if len(sent) != 1 || sent[0].To != "+38761000009" {
		t.Fatalf("sent %+v, want one SMS to the profile's phone", sent)
	}
	for _, never := range []string{"Ana", "Cy", "/a/", late.id} {
		if strings.Contains(sent[0].Body, never) {
			t.Errorf("the SMS holds %q: %s", never, sent[0].Body)
		}
	}
	if !strings.Contains(sent[0].Body, "25.10.2026. u 22:00 (Europe/Sarajevo)") {
		t.Errorf("the SMS %q lacks the clinic's time", sent[0].Body)
	}
	var messages, reads, tokens int
	if err := s.owner.QueryRow(ctx, `SELECT coalesce(sum(messages), 0) FROM sms_usage WHERE organization_id = $1`, clinicID).Scan(&messages); err != nil || messages != 1 {
		t.Errorf("%d messages in the clinic's usage (%v), want 1", messages, err)
	}
	if err := s.owner.QueryRow(ctx, `SELECT count(*) FROM audit_event WHERE agent_name = 'reminder_sms' AND entity_reference = 'Patient/' || $1`, b.patient).Scan(&reads); err != nil || reads == 0 {
		t.Errorf("%d audited reads of the patient by the send (%v), want some", reads, err)
	}
	if err := s.owner.QueryRow(ctx, `SELECT count(*) FROM appointment_action_token WHERE reminder_id = $1`, late.reminders[1440]).Scan(&tokens); err != nil || tokens != 0 {
		t.Errorf("%d one-time links (%v), want none", tokens, err)
	}
}

// checkProfileTooEarly: an hour before a 05:30 CET visit is 04:30 CET, quiet until 08:00, after the start: no SMS.
func checkProfileTooEarly(t *testing.T, s *reminderStack, b booking, role string) {
	t.Helper()
	early := s.seedVisit(t, role, b.patient, time.Date(2026, 10, 25, 4, 30, 0, 0, time.UTC))
	if err := s.dispatch(t, early, 60, time.Date(2026, 10, 25, 3, 30, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	if jobs, _ := s.sends(t, early.reminders[60]); len(jobs) != 0 {
		t.Errorf("an hour before 05:30: %v, want nothing", jobs)
	}
	if st := s.state(t, early.reminders[60]); st != "dispatched" {
		t.Errorf("the reminder is %s, want dispatched", st)
	}
}

// checkProfileCancelledAndMoved: a visit cancelled before its dispatch sends nothing; one moved after its first
// reminder went has that SMS cancelled at the send.
func checkProfileCancelledAndMoved(t *testing.T, s *reminderStack, b booking, role string) {
	t.Helper()
	ctx := t.Context()
	cancelled := s.seedVisit(t, role, b.patient, time.Date(2026, 10, 27, 9, 0, 0, 0, time.UTC))
	s.clock.set(time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC))
	var admin string
	if err := s.owner.QueryRow(ctx, `INSERT INTO account (email, password_hash, role, organization_id) VALUES ('admin-p@example.test', 'h', 'admin', $1) RETURNING id::text`,
		clinicID).Scan(&admin); err != nil {
		t.Fatal(err)
	}
	if _, err := s.appointments.Cancel(ctx, appointment.Caller{AccountID: admin, OrganizationID: clinicID, Admin: true}, cancelled.id); err != nil {
		t.Fatal(err)
	}
	if err := s.dispatch(t, cancelled, 1440, time.Date(2026, 10, 26, 9, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	if jobs, _ := s.sends(t, cancelled.reminders[1440]); len(jobs) != 0 {
		t.Errorf("a cancelled visit's reminder sends %v", jobs)
	}

	moved := s.seedVisit(t, role, b.patient, time.Date(2026, 10, 28, 3, 0, 0, 0, time.UTC)) // 04:00 CET: quiet
	if err := s.dispatch(t, moved, 1440, time.Date(2026, 10, 27, 3, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	if _, err := s.owner.Exec(ctx, `UPDATE appointment SET period = tstzrange(lower(period) + interval '1 day', upper(period) + interval '1 day'),
		slot_id = gen_random_uuid(), reschedule_count = reschedule_count + 1 WHERE id = $1`, moved.id); err != nil {
		t.Fatal(err)
	}
	s.sendAll(t, moved.reminders[1440], time.Date(2026, 10, 27, 7, 0, 0, 0, time.UTC))
	if _, deliveries := s.sends(t, moved.reminders[1440]); !slices.Equal(deliveries, []string{"sms cancelled"}) {
		t.Errorf("a moved visit's earlier reminder: %v, want its SMS cancelled", deliveries)
	}
	if len(s.sms.Sent()) != 1 {
		t.Errorf("sent %d SMS in all, want the first reminder's only", len(s.sms.Sent()))
	}
}

// checkProfileNoOptionNoPhone: without the clinic's SMS option the dispatch stores nothing; without a phone the SMS
// is cancelled at its send.
func checkProfileNoOptionNoPhone(t *testing.T, s *reminderStack, b booking, role string) {
	t.Helper()
	ctx := t.Context()
	off := s.seedVisit(t, role, b.patient, time.Date(2026, 10, 30, 10, 0, 0, 0, time.UTC))
	if _, err := s.owner.Exec(ctx, `UPDATE organization_appointment_settings SET sms_reminders = false WHERE organization_id = $1`, clinicID); err != nil {
		t.Fatal(err)
	}
	if err := s.dispatch(t, off, 1440, time.Date(2026, 10, 29, 10, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	if jobs, _ := s.sends(t, off.reminders[1440]); len(jobs) != 0 {
		t.Errorf("without the SMS option: %v, want nothing", jobs)
	}
	for _, q := range []string{
		`UPDATE organization_appointment_settings SET sms_reminders = true WHERE organization_id = $2 AND $1::uuid IS NOT NULL`,
		`UPDATE patient SET phone = NULL WHERE id = $1 AND $2::uuid IS NOT NULL`,
	} {
		if _, err := s.owner.Exec(ctx, q, b.patient, clinicID); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.dispatch(t, off, 60, time.Date(2026, 10, 30, 9, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	s.sendAll(t, off.reminders[60], time.Date(2026, 10, 30, 9, 0, 0, 0, time.UTC))
	if _, deliveries := s.sends(t, off.reminders[60]); !slices.Equal(deliveries, []string{"sms cancelled"}) {
		t.Errorf("without a phone: %v, want the SMS cancelled", deliveries)
	}
	if len(s.sms.Sent()) != 1 {
		t.Errorf("sent %d SMS in all, want one", len(s.sms.Sent()))
	}
}

// checkProfileMovedByStaff: the desk moves a visit a day later (docs/clinic-cms wave 4, M1 with R3). The reminders of
// the old time are cancelled and send nothing; the new time's are scheduled, and the day-before one, its first
// dispatch rolled back and every job run twice, sends one SMS with the new time, counted once.
func checkProfileMovedByStaff(t *testing.T, s *reminderStack, b booking, role string) {
	t.Helper()
	ctx := t.Context()
	if _, err := s.owner.Exec(ctx, `UPDATE patient SET phone = '+38761000009' WHERE id = $1`, b.patient); err != nil {
		t.Fatal(err)
	}
	var admin string
	if err := s.owner.QueryRow(ctx, `INSERT INTO account (email, password_hash, role, organization_id) VALUES ('admin-m@example.test', 'h', 'admin', $1) RETURNING id::text`,
		clinicID).Scan(&admin); err != nil {
		t.Fatal(err)
	}
	from := s.seedVisit(t, role, b.patient, time.Date(2026, 11, 3, 10, 0, 0, 0, time.UTC))
	if _, err := s.owner.Exec(ctx, `UPDATE appointment SET visit_mode = 'in-person' WHERE id = $1`, from.id); err != nil { // a desk visit
		t.Fatal(err)
	}
	to := time.Date(2026, 11, 4, 10, 0, 0, 0, time.UTC) // 11:00 CET
	s.slots.next = slot.Bookable{PractitionerRoleID: role, Slot: slot.Slot{
		ID: "5e000000-0000-4000-8000-00000000005e", OrganizationID: clinicID,
		Period: slot.Period{Start: to, End: to.Add(30 * time.Minute), Mode: schedule.ModeInPerson},
	}}
	s.clock.set(time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC))
	desk := appointment.Caller{AccountID: admin, OrganizationID: clinicID, Admin: true}
	if _, err := s.appointments.MoveForPatient(ctx, desk, from.id, s.slots.next.ID, 1); err != nil {
		t.Fatalf("MoveForPatient = %v", err)
	}
	for _, old := range from.reminders {
		if st := s.state(t, old); st != "cancelled" {
			t.Errorf("the old time's reminder is %s, want cancelled", st)
		}
	}
	moved := visit{id: from.id, reminders: map[int]string{}}
	rows, err := s.owner.Query(ctx, `SELECT offset_minutes, id::text FROM appointment_reminder WHERE appointment_id = $1 AND state = 'scheduled' AND start_at = $2`, from.id, to)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var offset int
		var id string
		if err := rows.Scan(&offset, &id); err != nil {
			t.Fatal(err)
		}
		moved.reminders[offset] = id
	}
	rows.Close()
	if moved.reminders[1440] == "" || moved.reminders[60] == "" {
		t.Fatalf("reminders at the new time %v, want the day before and the hour before", moved.reminders)
	}
	if err := s.dispatch(t, from, 1440, time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	if jobs, _ := s.sends(t, from.reminders[1440]); len(jobs) != 0 {
		t.Errorf("the old time's reminder sends %v", jobs)
	}
	before, sent := s.smsMessages(t), len(s.sms.Sent())
	s.jobs.failed = false
	dayBefore := time.Date(2026, 11, 3, 10, 0, 0, 0, time.UTC)
	if err := s.dispatch(t, moved, 1440, dayBefore); err == nil {
		t.Fatal("the first dispatch did not fail")
	}
	for range 2 {
		if err := s.dispatch(t, moved, 1440, dayBefore); err != nil {
			t.Fatal(err)
		}
	}
	if jobs, _ := s.sends(t, moved.reminders[1440]); !slices.Equal(jobs, []string{"reminder_sms at 2026-11-03T10:00Z"}) {
		t.Errorf("the new day before: %v, want the SMS at 10:00 UTC", jobs)
	}
	s.sendAll(t, moved.reminders[1440], dayBefore)
	all := s.sms.Sent()
	if len(all) != sent+1 || all[len(all)-1].To != "+38761000009" || !strings.Contains(all[len(all)-1].Body, "04.11.2026. u 11:00 (Europe/Sarajevo)") {
		t.Fatalf("sent %+v after the move, want one more SMS with the new time", all[sent:])
	}
	if after := s.smsMessages(t); after != before+1 {
		t.Errorf("%d messages in the clinic's usage after the move's SMS, want %d", after, before+1)
	}
}

// smsMessages is the clinic's SMS usage: the messages counted so far.
func (s *reminderStack) smsMessages(t *testing.T) int {
	t.Helper()
	var n int
	if err := s.owner.QueryRow(t.Context(), `SELECT coalesce(sum(messages), 0) FROM sms_usage WHERE organization_id = $1`, clinicID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}
