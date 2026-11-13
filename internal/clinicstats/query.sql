-- name: LockClinic :exec
-- A clinic's refresh lock, held until the transaction ends.
SELECT pg_advisory_xact_lock(hashtextextended('clinicstats:' || @organization_id::text, 0));

-- name: DeleteAppointmentDays :exec
DELETE FROM clinic_appointment_stat
WHERE organization_id = @organization_id AND day BETWEEN @first_day AND @last_day;

-- name: DeleteDiagnosisDays :exec
DELETE FROM clinic_diagnosis_stat
WHERE organization_id = @organization_id AND day BETWEEN @first_day AND @last_day;

-- name: InsertAppointmentCounts :exec
INSERT INTO clinic_appointment_stat (organization_id, practitioner_role_id, day, hour, specialty_code, status, count)
-- The arrays are of one length: unnest in the select list zips them.
SELECT @organization_id::uuid, unnest(@roles::uuid[]), unnest(@days::date[]), unnest(@hours::smallint[]),
       unnest(@specialties::text[]), unnest(@statuses::text[]), unnest(@counts::integer[]);

-- name: InsertDiagnosisCounts :exec
-- An empty subgroup is none (NULL).
INSERT INTO clinic_diagnosis_stat (organization_id, practitioner_role_id, day, hour, specialty_code, group_code,
                                   subgroup_code, code, count)
SELECT @organization_id::uuid, unnest(@roles::uuid[]), unnest(@days::date[]), unnest(@hours::smallint[]),
       unnest(@specialties::text[]), unnest(@group_codes::text[]), NULLIF(unnest(@subgroup_codes::text[]), ''),
       unnest(@codes::text[]), unnest(@counts::integer[]);

-- name: AppointmentSeries :many
-- A clinic's appointments per month of the local days from_day to to_day: every one, and those of each final status.
SELECT to_char(day, 'YYYY-MM')::text AS period,
       sum(count)::bigint AS booked,
       coalesce(sum(count) FILTER (WHERE status = 'fulfilled'), 0)::bigint AS fulfilled,
       coalesce(sum(count) FILTER (WHERE status = 'noshow'), 0)::bigint AS noshow,
       coalesce(sum(count) FILTER (WHERE status = 'cancelled'), 0)::bigint AS cancelled
FROM clinic_appointment_stat
WHERE organization_id = @organization_id AND day BETWEEN @from_day AND @to_day
  AND (sqlc.narg(practitioner_role_id)::uuid IS NULL OR practitioner_role_id = sqlc.narg(practitioner_role_id))
  AND (@specialty::text = '' OR specialty_code = @specialty)
GROUP BY 1
ORDER BY 1;

-- name: AppointmentHeatmap :many
-- The appointments that were not cancelled per ISO weekday (1 is Monday) and local hour.
SELECT extract(isodow FROM day)::integer AS weekday, hour::integer AS hour, sum(count)::bigint AS count
FROM clinic_appointment_stat
WHERE organization_id = @organization_id AND day BETWEEN @from_day AND @to_day AND status <> 'cancelled'
  AND (sqlc.narg(practitioner_role_id)::uuid IS NULL OR practitioner_role_id = sqlc.narg(practitioner_role_id))
  AND (@specialty::text = '' OR specialty_code = @specialty)
GROUP BY 1, 2
ORDER BY 1, 2;

-- name: AppointmentPractitioners :many
-- The fulfilled appointments per role, most first.
SELECT practitioner_role_id, coalesce(sum(count) FILTER (WHERE status = 'fulfilled'), 0)::bigint AS fulfilled
FROM clinic_appointment_stat
WHERE organization_id = @organization_id AND day BETWEEN @from_day AND @to_day
  AND (sqlc.narg(practitioner_role_id)::uuid IS NULL OR practitioner_role_id = sqlc.narg(practitioner_role_id))
  AND (@specialty::text = '' OR specialty_code = @specialty)
GROUP BY 1
ORDER BY 2 DESC, 1;

-- name: DiagnosisCounts :many
-- The confirmed diagnoses counted by the breakdown, most first (months in their order); '' for no subgroup or no
-- specialty.
SELECT (CASE @breakdown::text
            WHEN 'group' THEN group_code
            WHEN 'subgroup' THEN coalesce(subgroup_code, '')
            WHEN 'code' THEN code
            WHEN 'practitionerRole' THEN practitioner_role_id::text
            WHEN 'specialty' THEN specialty_code
            ELSE to_char(day, 'YYYY-MM') END)::text AS key,
       sum(count)::bigint AS count
FROM clinic_diagnosis_stat
WHERE organization_id = @organization_id AND day BETWEEN @from_day AND @to_day
  AND (sqlc.narg(practitioner_role_id)::uuid IS NULL OR practitioner_role_id = sqlc.narg(practitioner_role_id))
  AND (@specialty::text = '' OR specialty_code = @specialty)
  AND (@group_code::text = '' OR group_code = @group_code)
  AND (@subgroup_code::text = '' OR subgroup_code = @subgroup_code)
GROUP BY 1
ORDER BY (CASE WHEN @breakdown::text = 'month' THEN 0 ELSE sum(count) END) DESC, 1;
