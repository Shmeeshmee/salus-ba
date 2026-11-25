-- +goose Up
-- The start of a visit and of a slot as columns of their own (docs/clinic-cms/load/REPORT.md, WAVE7.md RD2), for the
-- reads of a range under row-level security: the desk's day, the calendar's week, the practice report, the booking's
-- free slots and the hourly slot generation. A condition on lower(period) is not leakproof (lower() of a range is not
-- marked so), so Postgres applies it only after the policies, row by row: it read a busy clinic's whole year for one
-- day (132 ms on the busy year's seed, 1 ms on this column, whose comparison is leakproof) and a schedule's every
-- slot for one week. Generated from period: the same value, never written by the app, kept with its row (no new
-- personal data; the retention rules of appointment and slot apply). The indexes on lower(period) stay: the aggregate
-- builds read as salus_stats, without the policies.
ALTER TABLE appointment ADD COLUMN start_at timestamptz GENERATED ALWAYS AS (lower(period)) STORED;
CREATE INDEX appointment_organization_start_at ON appointment (organization_id, start_at);

ALTER TABLE slot ADD COLUMN start_at timestamptz GENERATED ALWAYS AS (lower(period)) STORED;
CREATE INDEX slot_schedule_start_at ON slot (schedule_id, start_at);
