-- +goose Up
-- Platform-wide anonymized trends (SAL-21, docs/plans/analytics.md §5 and §6, ADR 0013):
--   - organization: the clinic's location (O1), a city (a Wikidata item of place) under a leaf region (an ISO
--     3166-2:BA code of place), set together with the time it was set; and whether the clinic contributes to the
--     platform figures (O6, on by default), with the time that last changed. A clinic counts in a period only when
--     neither changed during it (§5 "Which clinics count"). No personal data.
--   - trends_period: every period built (a month, a quarter of the heatmap, an ISO week of the rises), once: whether
--     it was published or refused (A7), whether its code level was kept, and k. A row is never changed, so a period
--     is never built again (A3), also once its cells are past retention.
--   - trends_cell: every published or suppressed cell of a period's tables (G×D, G×S, S×D, the heatmap), its
--     dimensions ('' for "all"), its count rounded to 5 (A10), NULL when suppressed. No clinic, practitioner or
--     patient column (A1).
--   - trends_rise: the rise flags of a week (§6): region and group, no count.
-- salus_stats writes the figures once (the builds in cmd/worker); the app only reads them; nobody but the owner
-- changes or deletes them, and salus_retention deletes cells and rises after five years (internal/retention).

ALTER TABLE place ADD CONSTRAINT place_id_parent_key UNIQUE (id, parent_id);
ALTER TABLE place ADD CONSTRAINT place_id_kind_key UNIQUE (id, kind);

ALTER TABLE organization
    ADD COLUMN location_since timestamptz,
    ADD COLUMN contributes_since timestamptz DEFAULT now() NOT NULL,
    ADD COLUMN analytics_contributes boolean DEFAULT true NOT NULL,
    ADD COLUMN region_id text,
    ADD COLUMN city_id text,
    -- Constant, for the foreign key that makes region_id a region's.
    ADD COLUMN region_kind text GENERATED ALWAYS AS (CASE WHEN region_id IS NOT NULL THEN 'region' END) STORED,
    ADD CONSTRAINT organization_location CHECK (
        (region_id IS NULL) = (city_id IS NULL) AND (region_id IS NULL) = (location_since IS NULL)),
    ADD CONSTRAINT organization_region_fkey FOREIGN KEY (region_id, region_kind) REFERENCES place (id, kind),
    ADD CONSTRAINT organization_city_fkey FOREIGN KEY (city_id, region_id) REFERENCES place (id, parent_id);

GRANT UPDATE (region_id, city_id, location_since, analytics_contributes, contributes_since) ON organization TO salus_app;
GRANT SELECT (region_id, city_id, location_since, analytics_contributes, contributes_since) ON organization TO salus_stats;

CREATE TABLE trends_period (
    created_at timestamptz DEFAULT now() NOT NULL,
    min_count integer NOT NULL CHECK (min_count >= 10),
    code_level boolean NOT NULL,
    period text NOT NULL PRIMARY KEY CHECK (period ~ '^[0-9]{4}-([0-9]{2}|Q[1-4]|W[0-9]{2})$'),
    kind text NOT NULL CHECK (kind IN ('month', 'quarter', 'week')),
    status text NOT NULL CHECK (status IN ('published', 'refused')),
    CONSTRAINT trends_period_kind CHECK (
        (kind = 'month' AND substr(period, 6) BETWEEN '01' AND '12' AND substr(period, 6, 1) <> 'Q' AND substr(period, 6, 1) <> 'W')
        OR (kind = 'quarter' AND substr(period, 6, 1) = 'Q')
        OR (kind = 'week' AND substr(period, 6, 1) = 'W' AND substr(period, 7) BETWEEN '01' AND '53'))
);

CREATE TABLE trends_cell (
    id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    created_at timestamptz DEFAULT now() NOT NULL,
    count integer CHECK (count >= 0 AND count % 5 = 0),
    suppressed boolean NOT NULL,
    period text NOT NULL REFERENCES trends_period (period),
    tbl text NOT NULL CHECK (tbl IN ('gd', 'gs', 'sd', 'hm')),
    entity text DEFAULT '' NOT NULL,
    region text DEFAULT '' NOT NULL,
    city text DEFAULT '' NOT NULL,
    grp text DEFAULT '' NOT NULL,
    subgroup text DEFAULT '' NOT NULL,
    code text DEFAULT '' NOT NULL,
    specialty text DEFAULT '' NOT NULL,
    weekday text DEFAULT '' NOT NULL CHECK (weekday ~ '^[1-7]?$'),
    hour text DEFAULT '' NOT NULL CHECK (hour ~ '^([0-9]|1[0-9]|2[0-3])?$'),
    CONSTRAINT trends_cell_count CHECK ((count IS NULL) = suppressed),
    CONSTRAINT trends_cell_key UNIQUE (period, tbl, entity, region, city, grp, subgroup, code, specialty, weekday, hour)
);

CREATE INDEX trends_cell_created_at ON trends_cell (created_at);

CREATE TABLE trends_rise (
    id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    created_at timestamptz DEFAULT now() NOT NULL,
    period text NOT NULL REFERENCES trends_period (period),
    region text NOT NULL,
    grp text NOT NULL,
    CONSTRAINT trends_rise_key UNIQUE (period, region, grp)
);

CREATE INDEX trends_rise_created_at ON trends_rise (created_at);

GRANT SELECT ON trends_period, trends_cell, trends_rise TO salus_app;
GRANT SELECT, INSERT ON trends_period, trends_cell, trends_rise TO salus_stats;
GRANT SELECT, DELETE ON trends_cell, trends_rise TO salus_retention;
