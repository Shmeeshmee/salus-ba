-- Synthetic rows for the migration drill (scripts/migration-drill.sh), written against the schema at 00059, the last
-- migration before the clinic system: two clinics, their staff, three patients with consents, a schedule with slots,
-- visits booked, fulfilled and cancelled, a finished encounter, a questionnaire and its answers, a chat message and
-- audit events (dated now: their tables are partitioned by month around the migration's time). Names are plainly
-- synthetic. Run as the database owner (row-level security does not apply to it).
\set ON_ERROR_STOP on
BEGIN;
INSERT INTO organization (id, name) VALUES
  ('00000000-0000-4000-8000-0000000000a1', 'Synthetic Clinic One'),
  ('00000000-0000-4000-8000-0000000000a2', 'Synthetic Clinic Two');

INSERT INTO practitioner (id, organization_id, given_name, family_name) VALUES
  ('00000000-0000-4000-8000-0000000000d1', '00000000-0000-4000-8000-0000000000a1', 'Testdoctor', 'One'),
  ('00000000-0000-4000-8000-0000000000d2', '00000000-0000-4000-8000-0000000000a2', 'Testdoctor', 'Two');
INSERT INTO practitioner_role (id, organization_id, practitioner_id) VALUES
  ('00000000-0000-4000-8000-0000000000e1', '00000000-0000-4000-8000-0000000000a1', '00000000-0000-4000-8000-0000000000d1'),
  ('00000000-0000-4000-8000-0000000000e2', '00000000-0000-4000-8000-0000000000a2', '00000000-0000-4000-8000-0000000000d2');

INSERT INTO account (id, role, organization_id, practitioner_id, email, password_hash) VALUES
  ('00000000-0000-4000-8000-0000000000b1', 'admin', '00000000-0000-4000-8000-0000000000a1', NULL, 'admin.one@example.test', 'synthetic-hash'),
  ('00000000-0000-4000-8000-0000000000b2', 'practitioner', '00000000-0000-4000-8000-0000000000a1', '00000000-0000-4000-8000-0000000000d1', 'doctor.one@example.test', 'synthetic-hash'),
  ('00000000-0000-4000-8000-0000000000b3', 'practitioner', '00000000-0000-4000-8000-0000000000a2', '00000000-0000-4000-8000-0000000000d2', 'doctor.two@example.test', 'synthetic-hash'),
  ('00000000-0000-4000-8000-0000000000c1', 'patient', NULL, NULL, 'patient.one@example.test', 'synthetic-hash'),
  ('00000000-0000-4000-8000-0000000000c2', 'patient', NULL, NULL, 'patient.two@example.test', 'synthetic-hash'),
  ('00000000-0000-4000-8000-0000000000c3', 'patient', NULL, NULL, 'patient.three@example.test', 'synthetic-hash');

INSERT INTO patient (id, account_id, birth_date, given_name, family_name) VALUES
  ('00000000-0000-4000-8000-0000000000f1', '00000000-0000-4000-8000-0000000000c1', '1980-01-01', 'Testpatient', 'One'),
  ('00000000-0000-4000-8000-0000000000f2', '00000000-0000-4000-8000-0000000000c2', '1990-02-02', 'Testpatient', 'Two'),
  ('00000000-0000-4000-8000-0000000000f3', '00000000-0000-4000-8000-0000000000c3', '2000-03-03', 'Testpatient', 'Three');
INSERT INTO consent (patient_id, organization_id, status, date_time, revoked_at) VALUES
  ('00000000-0000-4000-8000-0000000000f1', '00000000-0000-4000-8000-0000000000a1', 'active', '2026-01-05T09:00:00Z', NULL),
  ('00000000-0000-4000-8000-0000000000f2', '00000000-0000-4000-8000-0000000000a1', 'active', '2026-01-05T09:00:00Z', NULL),
  ('00000000-0000-4000-8000-0000000000f3', '00000000-0000-4000-8000-0000000000a2', 'inactive', '2026-01-05T09:00:00Z', '2026-02-01T09:00:00Z');

INSERT INTO schedule (id, organization_id, practitioner_role_id, slot_minutes, availability) VALUES
  ('00000000-0000-4000-8000-000000000101', '00000000-0000-4000-8000-0000000000a1', '00000000-0000-4000-8000-0000000000e1', 30,
   '[{"day": 1, "start": "08:00", "end": "12:00", "mode": "video"}]');
INSERT INTO slot (id, schedule_id, organization_id, period, status) VALUES
  ('00000000-0000-4000-8000-000000000201', '00000000-0000-4000-8000-000000000101', '00000000-0000-4000-8000-0000000000a1', tstzrange('2026-01-12T08:00:00Z', '2026-01-12T08:30:00Z'), 'busy'),
  ('00000000-0000-4000-8000-000000000202', '00000000-0000-4000-8000-000000000101', '00000000-0000-4000-8000-0000000000a1', tstzrange('2026-01-12T08:30:00Z', '2026-01-12T09:00:00Z'), 'free'),
  ('00000000-0000-4000-8000-000000000203', '00000000-0000-4000-8000-000000000101', '00000000-0000-4000-8000-0000000000a1', tstzrange('2030-01-07T08:00:00Z', '2030-01-07T08:30:00Z'), 'busy'),
  ('00000000-0000-4000-8000-000000000204', '00000000-0000-4000-8000-000000000101', '00000000-0000-4000-8000-0000000000a1', tstzrange('2030-01-07T08:30:00Z', '2030-01-07T09:00:00Z'), 'free');
INSERT INTO appointment (id, patient_id, practitioner_role_id, organization_id, slot_id, period, status, cancelled_at, cancellation_reason) VALUES
  ('00000000-0000-4000-8000-000000000301', '00000000-0000-4000-8000-0000000000f1', '00000000-0000-4000-8000-0000000000e1', '00000000-0000-4000-8000-0000000000a1',
   '00000000-0000-4000-8000-000000000201', tstzrange('2026-01-12T08:00:00Z', '2026-01-12T08:30:00Z'), 'fulfilled', NULL, NULL),
  ('00000000-0000-4000-8000-000000000302', '00000000-0000-4000-8000-0000000000f2', '00000000-0000-4000-8000-0000000000e1', '00000000-0000-4000-8000-0000000000a1',
   '00000000-0000-4000-8000-000000000202', tstzrange('2026-01-12T08:30:00Z', '2026-01-12T09:00:00Z'), 'cancelled', '2026-01-10T10:00:00Z', 'pat'),
  ('00000000-0000-4000-8000-000000000303', '00000000-0000-4000-8000-0000000000f2', '00000000-0000-4000-8000-0000000000e1', '00000000-0000-4000-8000-0000000000a1',
   '00000000-0000-4000-8000-000000000203', tstzrange('2030-01-07T08:00:00Z', '2030-01-07T08:30:00Z'), 'booked', NULL, NULL);

INSERT INTO encounter (id, appointment_id, organization_id, patient_id, practitioner_role_id, status, period_start, period_end) VALUES
  ('00000000-0000-4000-8000-000000000401', '00000000-0000-4000-8000-000000000301', '00000000-0000-4000-8000-0000000000a1', '00000000-0000-4000-8000-0000000000f1',
   '00000000-0000-4000-8000-0000000000e1', 'finished', '2026-01-12T08:00:00Z', '2026-01-12T08:25:00Z');

INSERT INTO questionnaire (id, series_id, organization_id, version, status, title, items, published_at) VALUES
  ('00000000-0000-4000-8000-000000000501', '00000000-0000-4000-8000-000000000502', '00000000-0000-4000-8000-0000000000a1', 1, 'active',
   'Synthetic intake form', '[{"linkId": "q1", "text": "Placeholder question", "type": "string"}]', '2026-01-02T09:00:00Z');
INSERT INTO questionnaire_response (id, due_at, organization_id, appointment_id, patient_id, practitioner_role_id, questionnaire_id, series_id, source, status, answers, authored_at) VALUES
  ('00000000-0000-4000-8000-000000000601', '2026-01-12T08:00:00Z', '00000000-0000-4000-8000-0000000000a1', '00000000-0000-4000-8000-000000000301',
   '00000000-0000-4000-8000-0000000000f1', '00000000-0000-4000-8000-0000000000e1', '00000000-0000-4000-8000-000000000501', '00000000-0000-4000-8000-000000000502',
   'default', 'completed', '[{"linkId": "q1", "answer": [{"valueString": "Synthetic answer"}]}]', '2026-01-11T18:00:00Z');

INSERT INTO conversation (id, organization_id, patient_id, practitioner_role_id, created_by) VALUES
  ('00000000-0000-4000-8000-000000000701', '00000000-0000-4000-8000-0000000000a1', '00000000-0000-4000-8000-0000000000f1', '00000000-0000-4000-8000-0000000000e1',
   '00000000-0000-4000-8000-0000000000b2');
INSERT INTO conversation_participant (id, conversation_id, added_by, role, patient_id) VALUES
  ('00000000-0000-4000-8000-000000000711', '00000000-0000-4000-8000-000000000701', '00000000-0000-4000-8000-0000000000b2', 'patient', '00000000-0000-4000-8000-0000000000f1');
INSERT INTO communication (conversation_id, organization_id, patient_id, client_message_id, seq, staff_only, sender_role, kind, sender_account_id, sender_participant_id, body, sent_at) VALUES
  ('00000000-0000-4000-8000-000000000701', '00000000-0000-4000-8000-0000000000a1', '00000000-0000-4000-8000-0000000000f1', '00000000-0000-4000-8000-000000000721',
   1, false, 'patient', 'text', '00000000-0000-4000-8000-0000000000c1', '00000000-0000-4000-8000-000000000711', 'Synthetic message', now());

INSERT INTO audit_event (recorded, outcome, action, type_system, type_code, subtype_system, subtype_code, purpose_system, purpose_code)
SELECT now() - n * interval '1 minute', 0, 'R', 'http://dicom.nema.org/resources/ontology/DCM', '110110',
       'http://hl7.org/fhir/restful-interaction', 'read', 'http://terminology.hl7.org/CodeSystem/v3-ActReason', 'TREAT'
FROM generate_series(1, 20) AS n;
COMMIT;
