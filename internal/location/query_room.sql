-- Rooms and devices of a location (RM1, docs/clinic-cms/WAVE6.md U3, U4), in the clinic's scope.

-- name: LocationVisible :one
SELECT EXISTS (SELECT 1 FROM location WHERE id = @id);

-- name: Rooms :many
SELECT id, location_id, name, active, version_id FROM room WHERE location_id = @location_id ORDER BY name, id;

-- name: InsertRoom :many
-- From the location, so a room is always in its location's clinic; none for a location outside the scope.
INSERT INTO room (organization_id, location_id, name, active)
SELECT l.organization_id, l.id, @name, @active FROM location l WHERE l.id = @location_id
RETURNING id, location_id, name, active, version_id;

-- name: UpdateRoom :many
UPDATE room SET name = @name, active = @active, updated_at = now(), version_id = version_id + 1
WHERE id = @id AND version_id = @version_id
RETURNING id, location_id, name, active, version_id;

-- name: LockRoom :many
-- A rule's room at a save of availability: two saves that name one room run one after the other (RM1).
SELECT id, location_id, name, active, version_id FROM room WHERE id = @id FOR UPDATE;

-- name: RoomVisible :one
SELECT EXISTS (SELECT 1 FROM room WHERE id = @id);

-- name: Devices :many
SELECT id, location_id, name, active, version_id FROM device WHERE location_id = @location_id ORDER BY name, id;

-- name: InsertDevice :many
INSERT INTO device (organization_id, location_id, name, active)
SELECT l.organization_id, l.id, @name, @active FROM location l WHERE l.id = @location_id
RETURNING id, location_id, name, active, version_id;

-- name: UpdateDevice :many
UPDATE device SET name = @name, active = @active, updated_at = now(), version_id = version_id + 1
WHERE id = @id AND version_id = @version_id
RETURNING id, location_id, name, active, version_id;

-- name: DeviceVisible :one
SELECT EXISTS (SELECT 1 FROM device WHERE id = @id);

-- name: Device :many
-- The device a price-list service names at its save, held until the save ends so it is not taken out of use
-- meanwhile (W6).
SELECT id, location_id, name, active, version_id FROM device WHERE id = @id FOR SHARE;
