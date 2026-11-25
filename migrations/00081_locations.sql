-- +goose Up
-- Locations on the server (docs/clinic-cms/WAVE6.md LC1, U1 and U2). No new table and no personal data.
--   - A clinic made after 00080 gets its first location as the migration gave the clinics of before: its name and
--     address, in its time zone. One trigger serves signup, salusctl and every other insert of a clinic.
--   - A visit's location and room are its slot's, set by the database on insert and whenever its slot, location or
--     room changes (a move): a caller's own values are replaced, so no scope (a patient's included) can put a visit
--     anywhere but where its slot is (6.0's review, point 5). A video visit has neither. The device stays RM1's.

-- +goose StatementBegin
CREATE FUNCTION organization_first_location() RETURNS trigger
    LANGUAGE plpgsql
    SECURITY DEFINER
    SET search_path = public, pg_temp
AS $$
BEGIN
    INSERT INTO location (organization_id, name, address_line, postal_code, city_id, time_zone)
    VALUES (NEW.id, NEW.name, NEW.address_line, NEW.postal_code, NEW.city_id, NEW.time_zone);
    RETURN NULL;
END
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION organization_first_location() FROM PUBLIC;
CREATE TRIGGER organization_first_location AFTER INSERT ON organization
    FOR EACH ROW EXECUTE FUNCTION organization_first_location();

-- +goose StatementBegin
CREATE FUNCTION appointment_location_from_slot() RETURNS trigger
    LANGUAGE plpgsql
    SECURITY DEFINER
    SET search_path = public, pg_temp
AS $$
BEGIN
    SELECT s.location_id, s.room_id INTO NEW.location_id, NEW.room_id
    FROM slot s
    WHERE s.id = NEW.slot_id AND s.organization_id = NEW.organization_id AND NEW.visit_mode = 'in-person';
    IF NOT FOUND THEN
        NEW.location_id := NULL;
        NEW.room_id := NULL;
    END IF;
    RETURN NEW;
END
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION appointment_location_from_slot() FROM PUBLIC;
CREATE TRIGGER appointment_location_from_slot BEFORE INSERT OR UPDATE OF slot_id, visit_mode, location_id, room_id ON appointment
    FOR EACH ROW EXECUTE FUNCTION appointment_location_from_slot();
