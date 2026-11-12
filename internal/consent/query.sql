-- consent holds the care consents, transcription_consent the transcription consents (migration 00040) and
-- history_consent the history consents (00043): the same shape, a kind each. A read of all is their UNION ALL, each table on its own index and under its own
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
WHERE t.id = @id
UNION ALL
SELECT 'history'::text AS kind, h.id, h.updated_at, h.date_time, h.revoked_at, h.patient_id, h.organization_id, h.version_id, h.status
FROM history_consent h
WHERE h.id = @id;

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

-- name: RevokeHistoryConsent :one
UPDATE history_consent
SET status = 'inactive', revoked_at = @revoked_at, updated_at = @revoked_at, version_id = version_id + 1
WHERE id = @id AND status = 'active'
RETURNING id, updated_at, date_time, revoked_at, patient_id, organization_id, version_id, status;

-- The patient's active history consent with the clinic, revoked by the patient or with their care consent there.
-- name: RevokeActiveHistoryConsent :one
UPDATE history_consent
SET status = 'inactive', revoked_at = @revoked_at, updated_at = @revoked_at, version_id = version_id + 1
WHERE patient_id = @patient_id AND organization_id = @organization_id AND status = 'active'
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
UNION ALL
SELECT 'history'::text AS kind, h.id, h.updated_at, h.date_time, h.revoked_at, h.patient_id, h.organization_id, h.version_id, h.status
FROM history_consent h
WHERE h.patient_id = @patient_id
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

-- chat_consent (migration 00044, SAL-28): the patient's consent for one conversation and one invited role.
-- name: GrantChatConsent :one
INSERT INTO chat_consent (patient_id, organization_id, conversation_id, practitioner_role_id, practitioner_organization_id, status,
    date_time, updated_at)
VALUES (@patient_id, @organization_id, @conversation_id, @practitioner_role_id, @practitioner_organization_id, 'active', @date_time, @date_time)
RETURNING id, updated_at, date_time, revoked_at, patient_id, organization_id, conversation_id, practitioner_role_id,
    practitioner_organization_id, version_id, status;

-- name: RevokeChatConsent :one
UPDATE chat_consent
SET status = 'inactive', revoked_at = @revoked_at, updated_at = @revoked_at, version_id = version_id + 1
WHERE conversation_id = @conversation_id AND practitioner_role_id = @practitioner_role_id AND status = 'active'
RETURNING id, updated_at, date_time, revoked_at, patient_id, organization_id, conversation_id, practitioner_role_id,
    practitioner_organization_id, version_id, status;

-- name: ChatConsentGranted :one
SELECT EXISTS (SELECT 1 FROM chat_consent
               WHERE conversation_id = @conversation_id AND practitioner_role_id = @practitioner_role_id AND status = 'active')::boolean;

-- The visits the patient's active history consent with the clinic lists now (migration 00043), oldest selection
-- first.
-- name: HistoryShared :many
SELECT s.encounter_id
FROM history_consent_encounter s
JOIN history_consent h ON h.id = s.consent_id
WHERE h.patient_id = @patient_id AND h.organization_id = @organization_id AND h.status = 'active' AND s.removed_at IS NULL
ORDER BY s.added_at, s.encounter_id;

-- A second active history consent with the clinic inserts nothing (no row).
-- name: GrantHistoryConsent :one
INSERT INTO history_consent (patient_id, organization_id, status, date_time, updated_at)
VALUES (@patient_id, @organization_id, 'active', @date_time, @date_time)
ON CONFLICT (patient_id, organization_id) WHERE status = 'active' DO NOTHING
RETURNING id, updated_at, date_time, revoked_at, patient_id, organization_id, version_id, status;

-- name: ActiveHistoryConsent :one
SELECT id, updated_at, date_time, revoked_at, patient_id, organization_id, version_id, status
FROM history_consent
WHERE patient_id = @patient_id AND organization_id = @organization_id AND status = 'active';

-- A change of the selection names the version it changes: a concurrent change or revocation finds no row.
-- name: ChangeHistoryConsent :one
UPDATE history_consent
SET updated_at = @updated_at, version_id = version_id + 1
WHERE id = @id AND version_id = @version_id AND status = 'active'
RETURNING id, updated_at, date_time, revoked_at, patient_id, organization_id, version_id, status;

-- Row-level security refuses a visit that is not a finished one of the patient's at another clinic.
-- name: AddHistoryVisits :exec
INSERT INTO history_consent_encounter (consent_id, encounter_id, added_at)
SELECT @consent_id::uuid, e, @added_at::timestamptz FROM unnest(@encounter_ids::uuid[]) AS e;

-- A removal is final: the selection keeps its period, adding the visit again adds a row.
-- name: RemoveHistoryVisits :exec
UPDATE history_consent_encounter
SET removed_at = @removed_at
WHERE consent_id = @consent_id AND encounter_id = ANY(@encounter_ids::uuid[]) AND removed_at IS NULL;

-- The visits the history consents list now, oldest selection first.
-- name: HistoryVisits :many
SELECT consent_id, encounter_id, added_at
FROM history_consent_encounter
WHERE consent_id = ANY(@consent_ids::uuid[]) AND removed_at IS NULL
ORDER BY added_at, encounter_id;
