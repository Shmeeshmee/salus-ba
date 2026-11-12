-- The notification domain's queries (migration 00046): the push devices of the accounts, and the clinics' webhook
-- endpoints with the dead letters of their deliveries.

-- name: RegisterDevice :exec
-- A token belongs to the account that registered it last: a device signed into another account moves to it.
INSERT INTO push_device (account_id, platform, token, updated_at)
VALUES (@account_id, @platform, @token, @updated_at)
ON CONFLICT (token) DO UPDATE
SET account_id = EXCLUDED.account_id, platform = EXCLUDED.platform, updated_at = EXCLUDED.updated_at;

-- name: UnregisterDevice :exec
DELETE FROM push_device WHERE account_id = @account_id AND token = @token;

-- name: DevicesOfAccount :many
SELECT id, account_id, platform, token FROM push_device WHERE account_id = @account_id ORDER BY created_at, id;

-- name: DeviceByID :one
SELECT id, account_id, platform, token FROM push_device WHERE id = @id;

-- name: DeleteDevice :exec
-- The platform said the token is gone: only if it still is this one (a re-registration keeps the row).
DELETE FROM push_device WHERE id = @id AND token = @token;

-- name: LockWebhookEndpoints :exec
-- A clinic's endpoint creations count and insert one after the other, until the transaction ends.
SELECT pg_advisory_xact_lock(hashtextextended('webhook_endpoint:' || @organization_id::text, 0));

-- name: CreateWebhookEndpoint :one
INSERT INTO webhook_endpoint (id, organization_id, created_by, include_clinical, url, events, secret_sealed)
VALUES (@id, @organization_id, @created_by, @include_clinical, @url, @events, @secret_sealed)
RETURNING created_at;

-- name: WebhookEndpoints :many
-- Row-level security shows only the clinic set.
SELECT id, created_at, organization_id, include_clinical, url, events, secret_sealed
FROM webhook_endpoint
WHERE organization_id = @organization_id
ORDER BY created_at, id;

-- name: WebhookEndpointByID :one
SELECT id, created_at, organization_id, include_clinical, url, events, secret_sealed
FROM webhook_endpoint
WHERE id = @id;

-- name: DeleteWebhookEndpoint :execrows
DELETE FROM webhook_endpoint WHERE id = @id;

-- name: AddDeadLetter :exec
INSERT INTO webhook_dead_letter (endpoint_id, organization_id, event_id, conversation_id, message_id, attempts, last_status, event_type)
VALUES (@endpoint_id, @organization_id, @event_id, @conversation_id, @message_id, @attempts, @last_status, @event_type)
ON CONFLICT (endpoint_id, event_id) DO NOTHING;

-- name: DeleteDeadLettersBefore :execrows
-- The retention role only (it bypasses row-level security).
DELETE FROM webhook_dead_letter WHERE created_at < @before;
