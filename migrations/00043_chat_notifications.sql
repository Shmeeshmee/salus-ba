-- +goose Up
-- Chat notifications (docs/plans/chat.md, SAL-29): the push tokens of the patients' devices, the clinic's chat
-- settings (chat hours, the response time it states, the e-mail fallback's delay), and the clinic's outgoing
-- webhooks with the dead letters of their deliveries. No message text is stored here, and none is sent anywhere
-- from here but to a clinic's own webhook when its admin turns clinical content on.
--
-- Row-level security: the settings, the webhooks and the dead letters are the clinic's (app.organization_id), as
-- the organization itself. A push device is an account's, like the account and its sessions (not row-level
-- secured): the service always names the account, and a token moves to the account that registers it last, so a
-- device signed out of one account and into another never receives the first one's notifications.
-- NULLIF: an unset setting reads as '' on a pooled connection and must match nothing.

CREATE TABLE push_device (
    id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    -- Deleted with the account (erasure, the subject-rights feature), on unregister, and when the platform says
    -- the token is gone.
    account_id uuid NOT NULL REFERENCES account (id) ON DELETE CASCADE,
    platform text NOT NULL CHECK (platform IN ('android', 'ios')),
    token text NOT NULL CHECK (char_length(token) BETWEEN 1 AND 4096),
    CONSTRAINT push_device_token_key UNIQUE (token)
);

CREATE INDEX push_device_account ON push_device (account_id);

GRANT SELECT, INSERT, DELETE ON push_device TO salus_app;
GRANT UPDATE (account_id, platform, updated_at) ON push_device TO salus_app;

-- The notifier finds a practitioner's account by the practitioner (auth), to tell them of a new message.
CREATE INDEX account_practitioner ON account (practitioner_id) WHERE practitioner_id IS NOT NULL;

-- The clinic's chat settings; a clinic without a row has the defaults (the organization service's). The hours are
-- a list of {day: mon..sun, start: HH:MM, end: HH:MM} in the clinic's time zone, validated by the service.
CREATE TABLE organization_chat_settings (
    organization_id uuid NOT NULL PRIMARY KEY REFERENCES organization (id),
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    response_time_hours integer NOT NULL CHECK (response_time_hours BETWEEN 1 AND 168),
    email_after_minutes integer NOT NULL CHECK (email_after_minutes BETWEEN 5 AND 10080),
    hours jsonb DEFAULT '[]' NOT NULL CHECK (jsonb_typeof(hours) = 'array' AND jsonb_array_length(hours) <= 21)
);

ALTER TABLE organization_chat_settings ENABLE ROW LEVEL SECURITY;
CREATE POLICY organization_chat_settings_clinic ON organization_chat_settings
    USING (organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid);

GRANT SELECT, INSERT ON organization_chat_settings TO salus_app;
GRANT UPDATE (hours, response_time_hours, email_after_minutes, updated_at) ON organization_chat_settings TO salus_app;

-- A clinic's webhook endpoint: its own system (the clinic is the controller of what it receives). The secret is
-- sealed with WEBHOOK_KEY (AES-256-GCM, bound to the endpoint's id) and shown to the admin once, at creation.
CREATE TABLE webhook_endpoint (
    id uuid NOT NULL PRIMARY KEY,
    created_at timestamptz DEFAULT now() NOT NULL,
    organization_id uuid NOT NULL REFERENCES organization (id),
    created_by uuid NOT NULL REFERENCES account (id),
    -- Off: a delivery carries the event's type and ids only. On: also the message's text.
    include_clinical boolean DEFAULT false NOT NULL,
    url text NOT NULL CHECK (url ~ '^https://' AND char_length(url) <= 2048),
    events text[] NOT NULL CHECK (cardinality(events) > 0 AND events <@ ARRAY['message.created']),
    secret_sealed bytea NOT NULL
);

CREATE INDEX webhook_endpoint_organization ON webhook_endpoint (organization_id);

ALTER TABLE webhook_endpoint ENABLE ROW LEVEL SECURITY;
CREATE POLICY webhook_endpoint_clinic ON webhook_endpoint
    USING (organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid);

GRANT SELECT, INSERT, DELETE ON webhook_endpoint TO salus_app;

-- A delivery that failed its last attempt: which event, to which endpoint, how often it was tried and the last
-- status (null when the endpoint never answered). Ids only; no payload. Deleted 30 days after (retention), or with
-- its endpoint.
CREATE TABLE webhook_dead_letter (
    id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    created_at timestamptz DEFAULT now() NOT NULL,
    endpoint_id uuid NOT NULL REFERENCES webhook_endpoint (id) ON DELETE CASCADE,
    organization_id uuid NOT NULL REFERENCES organization (id),
    event_id uuid NOT NULL,
    conversation_id uuid NOT NULL,
    message_id uuid NOT NULL,
    attempts integer NOT NULL CHECK (attempts > 0),
    last_status integer CHECK (last_status BETWEEN 100 AND 599),
    event_type text NOT NULL CHECK (event_type IN ('message.created')),
    CONSTRAINT webhook_dead_letter_once UNIQUE (endpoint_id, event_id)
);

CREATE INDEX webhook_dead_letter_created ON webhook_dead_letter (created_at);

ALTER TABLE webhook_dead_letter ENABLE ROW LEVEL SECURITY;
CREATE POLICY webhook_dead_letter_clinic ON webhook_dead_letter
    USING (organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid);

GRANT SELECT, INSERT ON webhook_dead_letter TO salus_app;
GRANT SELECT, DELETE ON webhook_dead_letter TO salus_retention;
