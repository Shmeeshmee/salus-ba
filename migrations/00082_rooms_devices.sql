-- +goose Up
-- Rooms and devices on a visit (docs/clinic-cms/WAVE6.md U3, U4, package RM1). The number is the next free one after
-- LC1's in WAVE6.md's order; the integration renumbers.
--
-- A visit's device is its service's when it is in person (none for video or without a service), and its room is its
-- slot's, with the slot's location when it has one: set by the guard below on every insert and on a change of the service, the mode
-- or the slot, and (the device and room) kept on any other update, so no scope can write other values (6.0's review, item 5). Two
-- bookings that need one device take a lock on it first, so they run one after the other and the second meets
-- appointment_device_no_overlap (23P01, resource_busy), never the deadlock two concurrent inserts into an exclusion
-- constraint can give. A later migration that writes these columns disables the trigger for its statement.

-- +goose StatementBegin
CREATE FUNCTION appointment_resources() RETURNS trigger
    LANGUAGE plpgsql
    SECURITY DEFINER
    SET search_path = pg_catalog, public
AS $$
BEGIN
    IF TG_OP = 'INSERT' OR NEW.service_id IS DISTINCT FROM OLD.service_id OR NEW.visit_mode IS DISTINCT FROM OLD.visit_mode THEN
        NEW.device_id := CASE WHEN NEW.visit_mode = 'in-person' THEN
            (SELECT c.device_id FROM public.charge_item_definition c WHERE c.id = NEW.service_id) END;
    ELSE
        NEW.device_id := OLD.device_id;
    END IF;
    IF TG_OP = 'INSERT' OR NEW.slot_id IS DISTINCT FROM OLD.slot_id THEN
        SELECT s.room_id, coalesce(s.location_id, NEW.location_id) INTO NEW.room_id, NEW.location_id
        FROM public.slot s WHERE s.id = NEW.slot_id;
        IF NOT FOUND THEN
            NEW.room_id := NULL;
        END IF;
    ELSE
        NEW.room_id := OLD.room_id; -- the location on other updates is LC1's (its guard)
    END IF;
    IF NEW.device_id IS NOT NULL AND NEW.status <> 'cancelled'
       AND (TG_OP = 'INSERT' OR NEW.period IS DISTINCT FROM OLD.period OR NEW.device_id IS DISTINCT FROM OLD.device_id
            OR NEW.status IS DISTINCT FROM OLD.status) THEN
        PERFORM pg_advisory_xact_lock(hashtextextended('appointment_device:' || NEW.device_id::text, 0));
    END IF;
    RETURN NEW;
END
$$;
-- +goose StatementEnd

CREATE TRIGGER appointment_resources BEFORE INSERT OR UPDATE ON appointment
    FOR EACH ROW EXECUTE FUNCTION appointment_resources();
