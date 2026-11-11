-- Synthetic rows only, in a table of the test's own beside the real schema (the migrate service applies the
-- repository's migrations after this).
CREATE TABLE synthetic_backup_rows (id uuid PRIMARY KEY DEFAULT gen_random_uuid(), name text NOT NULL);
INSERT INTO synthetic_backup_rows (name) SELECT 'Synthetic Testperson ' || n FROM generate_series(1, 500) AS n;
