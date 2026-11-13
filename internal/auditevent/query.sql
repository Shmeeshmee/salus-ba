-- name: InsertAuditEvent :exec
INSERT INTO audit_event (
    recorded, agent_account_id, agent_name, organization_id, outcome, action,
    type_system, type_code, subtype_system, subtype_code, purpose_system, purpose_code,
    entity_reference, entity_name, entity_count, request_id
) VALUES (
    @recorded, sqlc.narg('agent_account_id'), sqlc.narg('agent_name'), sqlc.narg('organization_id'), @outcome, @action,
    @type_system, @type_code, @subtype_system, @subtype_code, @purpose_system, @purpose_code,
    sqlc.narg('entity_reference'), sqlc.narg('entity_name'), sqlc.narg('entity_count'), sqlc.narg('request_id')
);

-- name: CountRecordedBefore :one
-- The retention job's daily overdue check. A sequential scan: no index leads with recorded, and partitioning
-- by year (docs/booking/SPEC.md §6) will make this a partition lookup.
SELECT count(*) FROM audit_event WHERE recorded < @before;

-- name: InsertAuditEvents :exec
-- Several events in one statement (a page of a patient's history, SAL-39), given as a JSON array of their columns;
-- an absent or null value is NULL.
INSERT INTO audit_event (
    recorded, agent_account_id, agent_name, organization_id, outcome, action,
    type_system, type_code, subtype_system, subtype_code, purpose_system, purpose_code,
    entity_reference, entity_name, entity_count, request_id
)
SELECT e.recorded, e.agent_account_id, e.agent_name, e.organization_id, e.outcome, e.action,
       e.type_system, e.type_code, e.subtype_system, e.subtype_code, e.purpose_system, e.purpose_code,
       e.entity_reference, e.entity_name, e.entity_count, e.request_id
FROM jsonb_to_recordset(@events::jsonb) AS e (
    recorded timestamptz, agent_account_id uuid, agent_name text, organization_id uuid, outcome smallint, action text,
    type_system text, type_code text, subtype_system text, subtype_code text, purpose_system text, purpose_code text,
    entity_reference text, entity_name text, entity_count bigint, request_id text);
