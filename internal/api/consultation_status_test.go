package api_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/videosession"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// occupiedProvider is the recording provider with the patient in every room until it removes them.
type occupiedProvider struct {
	identity string
	recordingProvider
}

func (p *occupiedProvider) Participants(context.Context, string) ([]string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.removed) > 0 {
		return nil, nil
	}
	return []string{p.identity}, nil
}

// A revocation before the start, with both in the room, cancels the appointment, removes the patient and ends
// the session: the clinic's scope no longer shows any of it. The practitioner who was in the room still reads
// the consultation's status and why (SPEC.md §2 "Consultation status", changed after package 6), audited; the
// clinic's other practitioner, its admin and another clinic's practitioner do not (row-level security as
// salus_app on the real database).
func TestStatus_ThePractitionerWhoWasInTheRoomReadsItAfterARevocationBeforeTheStart(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	at := time.Date(2030, 1, 7, 9, 0, 0, 0, time.UTC)
	now := at.Add(-5 * time.Minute)
	s := seedConsultation(t, owner, at)
	elsewhere := seedConsultation(t, owner, at.Add(2*time.Hour))
	pat, pra := callers(s)
	_, otherClinic := callers(elsewhere)
	var colleague, admin appointment.Caller
	if err := owner.QueryRow(t.Context(), `WITH p AS (INSERT INTO practitioner (organization_id, given_name, family_name) VALUES ($1, 'Di', 'Example') RETURNING id, organization_id),
		r AS (INSERT INTO practitioner_role (organization_id, practitioner_id) SELECT organization_id, id FROM p RETURNING practitioner_id),
		a AS (INSERT INTO account (email, password_hash, role, organization_id, practitioner_id)
			SELECT gen_random_uuid()::text || '@example.test', 'h', 'practitioner', $1, practitioner_id FROM r RETURNING id, practitioner_id),
		d AS (INSERT INTO account (email, password_hash, role, organization_id) VALUES (gen_random_uuid()::text || '@example.test', 'h', 'admin', $1) RETURNING id)
		SELECT a.id::text, a.practitioner_id::text, d.id::text FROM a, d`, s.clinic).Scan(&colleague.AccountID, &colleague.PractitionerID, &admin.AccountID); err != nil {
		t.Fatal(err)
	}
	colleague.OrganizationID, admin.OrganizationID, admin.Admin = s.clinic, s.clinic, true

	if _, err := realService(t, app, now, &recordingProvider{}).Join(t.Context(), pat, s.appointment); err != nil {
		t.Fatal(err)
	}
	var identity string
	if err := owner.QueryRow(t.Context(), `UPDATE video_session SET practitioner_joined_at = $2, patient_joined_at = $2 WHERE appointment_id = $1
		RETURNING patient_identity`, s.appointment, now.Add(-time.Minute)).Scan(&identity); err != nil {
		t.Fatal(err)
	}
	if _, err := owner.Exec(t.Context(), `UPDATE consent SET status = 'inactive', revoked_at = $2 WHERE patient_id = $1`, s.patient, now); err != nil {
		t.Fatal(err)
	}
	if _, err := owner.Exec(t.Context(), `UPDATE appointment SET status = 'cancelled', cancelled_at = $2, cancellation_reason = 'pat' WHERE id = $1`,
		s.appointment, now); err != nil {
		t.Fatal(err)
	}
	svc := realService(t, app, now, &occupiedProvider{identity: identity})
	if err := svc.EndPatientAccess(t.Context(), s.patient, s.clinic); err != nil {
		t.Fatal(err)
	}

	got, err := svc.Status(t.Context(), pra, s.appointment)
	if err != nil || got.State != videosession.StateEnded || got.EndReason != videosession.EndCancelled || got.Reason != videosession.ReasonConsentWithdrawn {
		t.Errorf("Status() by the practitioner = %+v, %v; want ended cancelled, with the reason consent_withdrawn", got, err)
	}
	if got, err := svc.Status(t.Context(), pat, s.appointment); err != nil || got.Reason != videosession.ReasonConsentWithdrawn {
		t.Errorf("Status() by the patient = %+v, %v; want the reason consent_withdrawn", got, err)
	}
	for name, c := range map[string]appointment.Caller{"another practitioner": colleague, "the admin": admin, "another clinic": otherClinic} {
		if got, err := svc.Status(t.Context(), c, s.appointment); !errors.Is(err, videosession.ErrNotFound) {
			t.Errorf("Status() by %s = %+v, %v; want ErrNotFound", name, got, err)
		}
	}
	var reads, refused int
	if err := owner.QueryRow(t.Context(), `SELECT count(*) FILTER (WHERE agent_account_id = $1 AND entity_reference = 'Appointment/' || $2 AND outcome = 0),
		count(*) FILTER (WHERE agent_account_id = ANY($3) AND entity_reference IS NULL AND outcome = 4)
		FROM audit_event WHERE subtype_code = 'read'`, pra.AccountID, s.appointment,
		[]string{colleague.AccountID, admin.AccountID, otherClinic.AccountID}).Scan(&reads, &refused); err != nil {
		t.Fatal(err)
	}
	if reads != 1 || refused != 3 {
		t.Errorf("%d reads of the appointment by the practitioner and %d refusals audited, want 1 and 3", reads, refused)
	}
}
