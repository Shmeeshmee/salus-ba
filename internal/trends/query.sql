-- name: InsertPeriod :execrows
-- A period is built once: a second build inserts nothing (A3).
INSERT INTO trends_period (period, kind, status, code_level, min_count)
VALUES (@period, @kind, @status, @code_level, @min_count)
ON CONFLICT (period) DO NOTHING;

-- name: PeriodBuilt :one
SELECT EXISTS (SELECT 1 FROM trends_period WHERE period = @period);

-- name: InsertCells :exec
-- The arrays are as long as each other, so the unnest calls pair them up; a count of -1 is a suppressed cell.
INSERT INTO trends_cell (period, tbl, entity, region, city, grp, subgroup, code, specialty, weekday, hour, count, suppressed)
SELECT @period::text, c.tbl, c.entity, c.region, c.city, c.grp, c.subgroup, c.code, c.specialty, c.weekday, c.hour,
       NULLIF(c.count, -1), c.count = -1
FROM (SELECT unnest(@tbls::text[]) AS tbl, unnest(@entities::text[]) AS entity, unnest(@regions::text[]) AS region,
             unnest(@cities::text[]) AS city, unnest(@groups::text[]) AS grp, unnest(@subgroups::text[]) AS subgroup,
             unnest(@codes::text[]) AS code, unnest(@specialties::text[]) AS specialty,
             unnest(@weekdays::text[]) AS weekday, unnest(@hours::text[]) AS hour, unnest(@counts::integer[]) AS count) AS c;

-- name: InsertRises :exec
INSERT INTO trends_rise (period, region, grp)
SELECT @period::text, r.region, r.grp
FROM (SELECT unnest(@regions::text[]) AS region, unnest(@groups::text[]) AS grp) AS r;

-- name: PeriodsOfKind :many
-- A month or a quarter whose cells retention has deleted is no longer listed (a week has flags, not cells).
SELECT created_at, min_count, code_level, period, kind, status
FROM trends_period p
WHERE kind = @kind AND (kind = 'week' OR EXISTS (SELECT 1 FROM trends_cell c WHERE c.period = p.period))
ORDER BY period;

-- name: CellsOfTable :many
-- A table's cells over a range of its periods ('2026-01' to '2026-10', '2026-Q3'): the service picks the query's.
SELECT period, tbl, entity, region, city, grp, subgroup, code, specialty, weekday, hour, count, suppressed
FROM trends_cell
WHERE tbl = @tbl AND period >= @from_period AND period <= @to_period
ORDER BY period, entity, region, city, grp, subgroup, code, specialty, weekday, hour;

-- name: RisesOfPeriods :many
SELECT period, region, grp FROM trends_rise WHERE period = ANY(@periods::text[]) ORDER BY period, region, grp;

-- name: DeleteCellsBefore :execrows
-- The retention job, as salus_retention: cells built more than five years ago.
DELETE FROM trends_cell WHERE created_at < @before;

-- name: DeleteRisesBefore :execrows
DELETE FROM trends_rise WHERE created_at < @before;
