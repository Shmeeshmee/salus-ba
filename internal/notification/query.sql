-- The notification domain's queries (migration 00047): the push devices of the accounts, and the clinics' webhook
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

-- SAL-33 (docs/plans/appointments.md): the deliveries of reminders and offers, the patients' settings, SMS usage.

-- name: InsertDelivery :one
-- A pending delivery; none when its source has one on that channel and device already.
INSERT INTO notification_delivery (kind, source_id, organization_id, account_id, channel, device_id)
VALUES (@kind, @source_id, @organization_id, @account_id, @channel, sqlc.narg(device_id))
ON CONFLICT ON CONSTRAINT notification_delivery_once DO NOTHING
RETURNING id, created_at;

-- name: LockDelivery :one
SELECT id, created_at, sent_at, source_id, organization_id, account_id, device_id, segments, kind, channel, state
FROM notification_delivery
WHERE id = @id
FOR UPDATE;

-- name: SetDeliveryState :execrows
UPDATE notification_delivery SET state = @state, sent_at = sqlc.narg(sent_at), segments = sqlc.narg(segments)
WHERE id = @id AND state = 'pending';

-- name: DeleteDeliveriesBefore :execrows
-- The retention role only.
DELETE FROM notification_delivery WHERE coalesce(sent_at, created_at) < @before::timestamptz;

-- name: NotificationPreference :one
SELECT push, email, sms, quiet_start, quiet_end, time_zone
FROM notification_preference
WHERE account_id = @account_id;

-- name: SetNotificationPreference :exec
INSERT INTO notification_preference (account_id, push, email, sms, quiet_start, quiet_end, time_zone, updated_at)
VALUES (@account_id, @push, @email, @sms, sqlc.narg(quiet_start), sqlc.narg(quiet_end), sqlc.narg(time_zone), @updated_at)
ON CONFLICT (account_id) DO UPDATE SET push = excluded.push, email = excluded.email, sms = excluded.sms,
    quiet_start = excluded.quiet_start, quiet_end = excluded.quiet_end, time_zone = excluded.time_zone, updated_at = excluded.updated_at;

-- name: AddSMSUsage :exec
-- In the clinic's scope (row-level security), in the SMS send's transaction.
INSERT INTO sms_usage (organization_id, month, messages, segments)
VALUES (@organization_id, @month, 1, @segments)
ON CONFLICT (organization_id, month) DO UPDATE SET messages = sms_usage.messages + 1, segments = sms_usage.segments + excluded.segments;

-- name: SMSUsage :many
SELECT month, messages, segments
FROM sms_usage
WHERE organization_id = @organization_id AND month >= @from_month::date AND month <= @to_month::date
ORDER BY month;

-- name: ClinicScope :one
SELECT coalesce(current_setting('app.organization_id', true), '')::text AS organization_id;
