-- consent holds the care consents, transcription_consent the transcription consents (migration 00040) and
-- history_consent the history consents (00042): the same shape, a kind each. A read of both is their UNION ALL, each table on its own index and under its own
-- row-level security.

-- A second active consent with the clinic inserts nothing (no row), so the transaction stays usable for the
-- audit.
-- name: GrantConsent :one
INSERT INTO consent (patient_id, organization_id, status, date_time, updated_at)
VALUES (@patient_id, @organization_id, 'active', @date_time, @date_time)
ON CONFLICT (patient_id, organization_id) WHERE status = 'active' DO NOTHING
RETURNING id, updated_at, date_time, revoked_at, patient_id, organization_id, version_id, status;

-- name: GrantTranscriptionConsent :one
INSERT INTO transcription_consent (patient_id, organization_id, status, date_time, updated_at)
VALUES (@patient_id, @organization_id, 'active', @date_time, @date_time)
ON CONFLICT (patient_id, organization_id) WHERE status = 'active' DO NOTHING
RETURNING id, updated_at, date_time, revoked_at, patient_id, organization_id, version_id, status;

-- name: ConsentByID :one
SELECT 'care'::text AS kind, c.id, c.updated_at, c.date_time, c.revoked_at, c.patient_id, c.organization_id, c.version_id, c.status
FROM consent c
WHERE c.id = @id
UNION ALL
SELECT 'transcription'::text AS kind, t.id, t.updated_at, t.date_time, t.revoked_at, t.patient_id, t.organization_id, t.version_id, t.status
FROM transcription_consent t
WHERE t.id = @id;

-- Only an active consent is revoked; a concurrent second revocation finds no row.
-- name: RevokeConsent :one
UPDATE consent
SET status = 'inactive', revoked_at = @revoked_at, updated_at = @revoked_at, version_id = version_id + 1
WHERE id = @id AND status = 'active'
RETURNING id, updated_at, date_time, revoked_at, patient_id, organization_id, version_id, status;

-- name: RevokeTranscriptionConsent :one
UPDATE transcription_consent
SET status = 'inactive', revoked_at = @revoked_at, updated_at = @revoked_at, version_id = version_id + 1
WHERE id = @id AND status = 'active'
RETURNING id, updated_at, date_time, revoked_at, patient_id, organization_id, version_id, status;

-- The patient's active transcription consent with the clinic, revoked with their care consent there.
-- name: RevokeActiveTranscriptionConsent :one
UPDATE transcription_consent
SET status = 'inactive', revoked_at = @revoked_at, updated_at = @revoked_at, version_id = version_id + 1
WHERE patient_id = @patient_id AND organization_id = @organization_id AND status = 'active'
RETURNING id, updated_at, date_time, revoked_at, patient_id, organization_id, version_id, status;

-- name: ConsentsByPatient :many
SELECT 'care'::text AS kind, c.id, c.updated_at, c.date_time, c.revoked_at, c.patient_id, c.organization_id, c.version_id, c.status
FROM consent c
WHERE c.patient_id = @patient_id
UNION ALL
SELECT 'transcription'::text AS kind, t.id, t.updated_at, t.date_time, t.revoked_at, t.patient_id, t.organization_id, t.version_id, t.status
FROM transcription_consent t
WHERE t.patient_id = @patient_id
ORDER BY date_time DESC, id;

-- revoked_at is the care consent's latest revocation (access reads it for the care record).
-- name: ConsentState :one
SELECT coalesce(bool_or(status = 'active'), false)::boolean AS active, max(revoked_at)::timestamptz AS revoked_at,
       EXISTS (SELECT 1 FROM transcription_consent t
               WHERE t.patient_id = @patient_id AND t.organization_id = @organization_id AND t.status = 'active')::boolean AS transcription,
       EXISTS (SELECT 1 FROM history_consent h
               WHERE h.patient_id = @patient_id AND h.organization_id = @organization_id AND h.status = 'active')::boolean AS history
FROM consent
WHERE patient_id = @patient_id AND organization_id = @organization_id;

-- The visits the patient's active history consent with the clinic lists now (migration 00042), oldest selection
-- first.
-- name: HistoryShared :many
SELECT s.encounter_id
FROM history_consent_encounter s
JOIN history_consent h ON h.id = s.consent_id
WHERE h.patient_id = @patient_id AND h.organization_id = @organization_id AND h.status = 'active' AND s.removed_at IS NULL
ORDER BY s.added_at, s.encounter_id;
