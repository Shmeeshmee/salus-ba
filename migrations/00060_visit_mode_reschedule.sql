-- +goose Up
-- Visit modes, package A of docs/plans/clinic-cms.md (ADR 0016). The number is a placeholder: the branch renumbers to
-- the next free number when it merges main.
--   - A reschedule keeps the appointment's visit mode. The app may move an appointment to another slot (00054) and may
--     not change visit_mode (00059), so nothing in the schema kept a video appointment off an in-person slot. This
--     trigger does: an appointment moves only to a slot of its own mode. The service refuses it first
--     (appointment.ErrNotReschedulable); this is the second line, as step 0's second review asked.
--   - slot_id is no foreign key (a free slot's row may be gone by the time its appointment is old), so a slot that
--     does not exist passes, as it does today.
--   - SECURITY DEFINER: the slot is busy by then and the patient's scope does not show it; the function reads its mode
--     only, as the owner.

-- +goose StatementBegin
CREATE FUNCTION appointment_guard_mode() RETURNS trigger
    LANGUAGE plpgsql
    SECURITY DEFINER
    SET search_path = pg_catalog, public
AS $$
BEGIN
    IF NEW.slot_id IS DISTINCT FROM OLD.slot_id
       AND EXISTS (SELECT 1 FROM public.slot s WHERE s.id = NEW.slot_id AND s.visit_mode <> NEW.visit_mode) THEN
        RAISE EXCEPTION 'appointment: a reschedule keeps the visit mode' USING ERRCODE = 'check_violation';
    END IF;
    RETURN NEW;
END
$$;
-- +goose StatementEnd

CREATE TRIGGER appointment_guard_mode BEFORE UPDATE OF slot_id ON appointment
    FOR EACH ROW EXECUTE FUNCTION appointment_guard_mode();
