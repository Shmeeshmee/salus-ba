-- A clinic's locations (docs/clinic-cms/WAVE6.md LC1, U1). Row-level security scopes every query: the clinic's own in
-- its scope, the active locations of listed clinics in the directory's.

-- name: Locations :many
-- The locations of a clinic the scope shows, active first, by name.
SELECT id, organization_id, name, address_line, postal_code, city_id, phone, time_zone, active, version_id
FROM location
WHERE organization_id = @organization_id
ORDER BY active DESC, name, id;

-- name: InsertLocation :one
-- A location of the scope's clinic, in the clinic's time zone when it names none.
INSERT INTO location (organization_id, name, address_line, postal_code, city_id, phone, time_zone, active)
VALUES (@organization_id, @name, sqlc.narg(address_line), sqlc.narg(postal_code), sqlc.narg(city_id), sqlc.narg(phone),
        coalesce(sqlc.narg(time_zone)::text, (SELECT o.time_zone FROM organization o WHERE o.id = @organization_id), 'Europe/Sarajevo'), @active)
RETURNING id, organization_id, name, address_line, postal_code, city_id, phone, time_zone, active, version_id;

-- name: UpdateLocation :one
-- Optimistic locking: no row when the location is not at version_id or not the scope's clinic's. A time zone left out
-- keeps the current one.
UPDATE location
SET name = @name, address_line = sqlc.narg(address_line), postal_code = sqlc.narg(postal_code), city_id = sqlc.narg(city_id),
    phone = sqlc.narg(phone), time_zone = coalesce(sqlc.narg(time_zone)::text, time_zone), active = @active, updated_at = now(),
    version_id = version_id + 1
WHERE id = @id AND organization_id = @organization_id AND version_id = @version_id
RETURNING id, organization_id, name, address_line, postal_code, city_id, phone, time_zone, active, version_id;

-- name: LocationExists :one
-- Whether the location is the scope's clinic's, whatever its version.
SELECT EXISTS (SELECT 1 FROM location WHERE id = @id AND organization_id = @organization_id);
