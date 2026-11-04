-- name: InsertAuditEvent :exec
INSERT INTO audit_event (
    recorded, agent_account_id, agent_name, organization_id, outcome, action,
    type_system, type_code, subtype_system, subtype_code, purpose_system, purpose_code,
    entity_reference, request_id
) VALUES (
    @recorded, sqlc.narg('agent_account_id'), sqlc.narg('agent_name'), sqlc.narg('organization_id'), @outcome, @action,
    @type_system, @type_code, @subtype_system, @subtype_code, @purpose_system, @purpose_code,
    sqlc.narg('entity_reference'), sqlc.narg('request_id')
);
