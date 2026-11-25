-- name: DeleteFreeSlotsExcept :execrows
-- Generation: the schedule's future free slots its rules no longer give, at that time, in that visit mode or at that
-- location (its room's when it names none) or room ('' for none). A busy slot is never deleted (here, and by the
-- slot_delete_free_only policy).
DELETE FROM slot
WHERE schedule_id = @schedule_id
  AND status = 'free'
  AND lower(period) > @after::timestamptz
  AND (lower(period), upper(period), visit_mode, coalesce(location_id::text, ''), coalesce(room_id::text, '')) NOT IN (
      SELECT g.s, g.e, g.m, coalesce(NULLIF(g.l, ''), room.location_id::text, ''), g.r
      FROM (SELECT unnest(@starts::timestamptz[]) AS s, unnest(@ends::timestamptz[]) AS e, unnest(@modes::text[]) AS m,
                   unnest(@locations::text[]) AS l, unnest(@rooms::text[]) AS r) AS g
      LEFT JOIN room ON room.id = NULLIF(g.r, '')::uuid);

-- name: InsertSlots :execrows
-- Generation: a free slot per period, in its rule's visit mode, at its location (its room's when it names none) and in
-- its room ('' for none); one that overlaps a slot of the schedule (slot_no_overlap) or another's in its room
-- (slot_room_no_overlap, which the rules' check at save leaves to a race) is skipped. The arrays are as long as each
-- other, so the unnest calls pair them up.
INSERT INTO slot (schedule_id, organization_id, period, visit_mode, location_id, room_id)
SELECT @schedule_id::uuid, @organization_id::uuid, tstzrange(g.s, g.e), g.m, coalesce(NULLIF(g.l, '')::uuid, room.location_id),
       room.id
FROM (SELECT unnest(@starts::timestamptz[]) AS s, unnest(@ends::timestamptz[]) AS e, unnest(@modes::text[]) AS m,
             unnest(@locations::text[]) AS l, unnest(@rooms::text[]) AS r) AS g
LEFT JOIN room ON room.id = NULLIF(g.r, '')::uuid
ON CONFLICT DO NOTHING;

-- name: SearchFreeSlots :many
-- Slot?schedule=…&start=ge…&start=lt…&status=free; the overlap lets the exclusion constraint's index serve it.
SELECT id, updated_at, schedule_id, organization_id, version_id, status, visit_mode, location_id, lower(period)::timestamptz AS start_at,
       upper(period)::timestamptz AS end_at
FROM slot
WHERE schedule_id = @schedule_id
  AND status = 'free'
  AND period && tstzrange(@from_at::timestamptz, @to_at::timestamptz)
  AND lower(period) >= @from_at::timestamptz
  AND lower(period) < @to_at::timestamptz
ORDER BY lower(period);

-- name: SlotByID :one
-- Booking: the slot to book, as the directory shows it (free, of an active role of a listed clinic).
SELECT id, updated_at, schedule_id, organization_id, version_id, status, visit_mode, location_id, lower(period)::timestamptz AS start_at,
       upper(period)::timestamptz AS end_at
FROM slot
WHERE id = @id;

-- name: HoldSlot :one
-- Booking: free to busy, in the appointment's transaction; no row when the slot is not free.
UPDATE slot
SET status = 'busy', version_id = version_id + 1, updated_at = @updated_at
WHERE id = @id AND status = 'free'
RETURNING id, updated_at, schedule_id, organization_id, version_id, status, visit_mode, location_id, lower(period)::timestamptz AS start_at,
          upper(period)::timestamptz AS end_at;

-- name: ReleaseSlot :execrows
-- Cancellation: busy to free while the slot is still ahead, in the appointment's transaction.
UPDATE slot
SET status = 'free', version_id = version_id + 1, updated_at = @updated_at
WHERE id = @id AND status = 'busy' AND lower(period) > @updated_at;

-- name: DeleteFreeSlotsEndedBefore :execrows
-- Retention, as salus_retention: free slots whose end is past their period, in every clinic.
DELETE FROM slot
WHERE status = 'free' AND upper(period) < @before::timestamptz;

-- SAL-35: the waitlist.

-- name: LockSlot :one
-- The waitlist: the slot to hand on, locked until the transaction ends (two jobs of one slot run one after the other).
SELECT id, updated_at, schedule_id, organization_id, version_id, status, visit_mode, location_id, lower(period)::timestamptz AS start_at,
       upper(period)::timestamptz AS end_at
FROM slot
WHERE id = @id
FOR UPDATE;

-- name: TransitionSlot :one
-- The waitlist: free to busy-tentative (an offer), busy-tentative to busy (its acceptance) or to free (none left); no
-- row when the slot is not in the status it moves from.
UPDATE slot
SET status = @to_status, version_id = version_id + 1, updated_at = @updated_at
WHERE id = @id AND status = @from_status::text
RETURNING id, updated_at, schedule_id, organization_id, version_id, status, visit_mode, location_id, lower(period)::timestamptz AS start_at,
          upper(period)::timestamptz AS end_at;
