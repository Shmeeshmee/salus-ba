-- name: InsertReport :one
INSERT INTO diagnostic_report (organization_id, patient_id, appointment_id, created_by_account_id, title, loinc_code, effective_date)
VALUES (@organization_id, @patient_id, @appointment_id, @created_by_account_id, @title, sqlc.narg(loinc_code), sqlc.narg(effective_date))
RETURNING *;

-- name: InsertResult :exec
INSERT INTO diagnostic_report_result (diagnostic_report_id, position, test, loinc_code, value_decimal, unit, value_text,
                                      reference_range, interpretation)
VALUES (@diagnostic_report_id, @position, @test, sqlc.narg(loinc_code), sqlc.narg(value_decimal), sqlc.narg(unit),
        sqlc.narg(value_text), sqlc.narg(reference_range), sqlc.narg(interpretation));

-- name: DeleteResults :exec
DELETE FROM diagnostic_report_result WHERE diagnostic_report_id = @diagnostic_report_id;

-- name: UpdateReport :execrows
UPDATE diagnostic_report
SET title = @title, loinc_code = sqlc.narg(loinc_code), effective_date = sqlc.narg(effective_date),
    version_id = version_id + 1, updated_at = now()
WHERE id = @id AND version_id = @version_id AND status = 'preliminary';

-- name: TouchReport :execrows
-- The report's PDF changed: its next version.
UPDATE diagnostic_report SET version_id = version_id + 1, updated_at = now()
WHERE id = @id AND version_id = @version_id AND status = 'preliminary';

-- name: ReleaseReport :execrows
UPDATE diagnostic_report
SET status = 'final', released_at = @released_at, released_by_account_id = @released_by_account_id,
    version_id = version_id + 1, updated_at = now()
WHERE id = @id AND version_id = @version_id AND status = 'preliminary';

-- name: ReportByID :one
SELECT * FROM diagnostic_report WHERE id = @id;

-- name: ReportsForAppointment :many
SELECT * FROM diagnostic_report WHERE appointment_id = @appointment_id ORDER BY created_at, id;

-- name: ReportsForPatient :many
SELECT * FROM diagnostic_report WHERE patient_id = @patient_id AND status = 'final' ORDER BY released_at DESC, id;

-- name: ResultsOf :many
SELECT * FROM diagnostic_report_result WHERE diagnostic_report_id = ANY(@ids::uuid[]) ORDER BY diagnostic_report_id, position;
