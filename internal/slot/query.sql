-- name: DeleteFreeSlotsExcept :execrows
-- Generation: the schedule's future free slots its rules no longer give. A busy slot is never deleted (here,
-- and by the slot_delete_free_only policy).
DELETE FROM slot
WHERE schedule_id = @schedule_id
  AND status = 'free'
  AND lower(period) > @after::timestamptz
  AND (lower(period), upper(period)) NOT IN (
      SELECT g.s, g.e FROM (SELECT unnest(@starts::timestamptz[]) AS s, unnest(@ends::timestamptz[]) AS e) AS g);

-- name: InsertSlots :execrows
-- Generation: a free slot per period; one that overlaps a slot of the schedule (slot_no_overlap) is skipped.
-- The two arrays are as long as each other, so the two unnest calls pair them up.
INSERT INTO slot (schedule_id, organization_id, period)
SELECT @schedule_id::uuid, @organization_id::uuid, tstzrange(g.s, g.e)
FROM (SELECT unnest(@starts::timestamptz[]) AS s, unnest(@ends::timestamptz[]) AS e) AS g
ON CONFLICT DO NOTHING;

-- name: SearchFreeSlots :many
-- Slot?schedule=…&start=ge…&start=lt…&status=free; the overlap lets the exclusion constraint's index serve it.
SELECT id, updated_at, schedule_id, organization_id, version_id, status, lower(period)::timestamptz AS start_at,
       upper(period)::timestamptz AS end_at
FROM slot
WHERE schedule_id = @schedule_id
  AND status = 'free'
  AND period && tstzrange(@from_at::timestamptz, @to_at::timestamptz)
  AND lower(period) >= @from_at::timestamptz
  AND lower(period) < @to_at::timestamptz
ORDER BY lower(period);

-- name: HoldSlot :one
-- Booking: free to busy, in the appointment's transaction; no row when the slot is not free.
UPDATE slot
SET status = 'busy', version_id = version_id + 1, updated_at = @updated_at
WHERE id = @id AND status = 'free'
RETURNING id, updated_at, schedule_id, organization_id, version_id, status, lower(period)::timestamptz AS start_at,
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
