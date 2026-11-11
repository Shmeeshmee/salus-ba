-- Synthetic rows only. A stand-in for the schema: goose's version table and one table with personal-looking data.
CREATE TABLE goose_db_version (id serial PRIMARY KEY, version_id bigint NOT NULL, is_applied boolean NOT NULL,
  tstamp timestamptz NOT NULL DEFAULT now());
INSERT INTO goose_db_version (version_id, is_applied) SELECT v, true FROM generate_series(0, 42) AS v;
CREATE TABLE patient (id uuid PRIMARY KEY DEFAULT gen_random_uuid(), name text NOT NULL);
INSERT INTO patient (name) SELECT 'Synthetic Testperson ' || n FROM generate_series(1, 500) AS n;
