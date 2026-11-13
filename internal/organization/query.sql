-- name: OrganizationByID :one
SELECT id, updated_at, version_id, active, listed, name, time_zone, default_language, patient_transcript_copy, region_id, city_id, location_since, analytics_contributes, contributes_since
FROM organization
WHERE id = @id;

-- name: ListedOrganizations :many
-- Organization?listed=true: row-level security shows only listed clinics in the directory scope.
SELECT id, updated_at, version_id, active, listed, name, time_zone, default_language, patient_transcript_copy
FROM organization
WHERE listed
ORDER BY name, id;

-- name: UpdateOrganization :one
-- Optimistic locking: only the current version changes.
UPDATE organization
SET name = @name, listed = @listed, time_zone = @time_zone, default_language = @default_language, patient_transcript_copy = @patient_transcript_copy,
    region_id = sqlc.narg(region_id), city_id = sqlc.narg(city_id), location_since = sqlc.narg(location_since),
    analytics_contributes = @analytics_contributes, contributes_since = @contributes_since,
    version_id = version_id + 1, updated_at = @updated_at
WHERE id = @id AND version_id = @version_id
RETURNING id, updated_at, version_id, active, listed, name, time_zone, default_language, patient_transcript_copy, region_id, city_id, location_since, analytics_contributes, contributes_since;

-- name: CreateOrganization :one
-- The operator command only (salusctl, as the owner): salus_app has no INSERT on organization.
INSERT INTO organization (name) VALUES (@name)
RETURNING id, updated_at, version_id, active, listed, name, time_zone, default_language, patient_transcript_copy, region_id, city_id, location_since, analytics_contributes, contributes_since;

-- name: ChatSettings :one
-- The clinic's chat settings (00047); row-level security shows only the clinic set.
SELECT hours, response_time_hours, email_after_minutes
FROM organization_chat_settings
WHERE organization_id = @organization_id;

-- name: PutChatSettings :exec
INSERT INTO organization_chat_settings (organization_id, hours, response_time_hours, email_after_minutes, updated_at)
VALUES (@organization_id, @hours, @response_time_hours, @email_after_minutes, @updated_at)
ON CONFLICT (organization_id) DO UPDATE
SET hours = EXCLUDED.hours, response_time_hours = EXCLUDED.response_time_hours,
    email_after_minutes = EXCLUDED.email_after_minutes, updated_at = EXCLUDED.updated_at;

-- name: AppointmentSettings :one
-- The clinic's appointment settings (00054); row-level security shows only the clinic set.
SELECT reminder_offsets_minutes, sms_reminders, reschedule_notice_hours, reschedule_max, waitlist_enabled, waitlist_hold_minutes,
       noshow_flag_threshold, noshow_flag_window_days
FROM organization_appointment_settings
WHERE organization_id = @organization_id;

-- name: PutAppointmentSettings :exec
INSERT INTO organization_appointment_settings (organization_id, reminder_offsets_minutes, sms_reminders, reschedule_notice_hours,
                                               reschedule_max, waitlist_enabled, waitlist_hold_minutes, noshow_flag_threshold,
                                               noshow_flag_window_days, updated_at)
VALUES (@organization_id, @reminder_offsets_minutes, @sms_reminders, @reschedule_notice_hours, @reschedule_max, @waitlist_enabled,
        @waitlist_hold_minutes, @noshow_flag_threshold, @noshow_flag_window_days, @updated_at)
ON CONFLICT (organization_id) DO UPDATE
SET reminder_offsets_minutes = EXCLUDED.reminder_offsets_minutes, sms_reminders = EXCLUDED.sms_reminders,
    reschedule_notice_hours = EXCLUDED.reschedule_notice_hours, reschedule_max = EXCLUDED.reschedule_max,
    waitlist_enabled = EXCLUDED.waitlist_enabled, waitlist_hold_minutes = EXCLUDED.waitlist_hold_minutes,
    noshow_flag_threshold = EXCLUDED.noshow_flag_threshold, noshow_flag_window_days = EXCLUDED.noshow_flag_window_days,
    updated_at = EXCLUDED.updated_at;

-- name: ListLocationsForStats :many
-- Every clinic with its time zone, location and contribution: the columns salus_stats may read.
SELECT id, time_zone, region_id, city_id, location_since, analytics_contributes, contributes_since
FROM organization ORDER BY id;
