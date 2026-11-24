-- +goose Up
-- Staff move a visit, package M1 of docs/clinic-cms (WAVE4.md, V4): the clinic's desk moves a booked visit to any free
-- slot of the clinic of its visit type, another practitioner's too, and the move is not the patient's own reschedule.
--   - The clinic's scope may move a booked visit without counting it (00054's guard counted every move once); the
--     patient's own scope still counts each of its moves exactly once. The clinic's count rule also lets the count go
--     up by one, as 00065 allowed, so nothing that was valid becomes invalid.
--   - The practitioner role changes only in the clinic's scope, with a move of a booked visit, to a role of the same
--     clinic (appointment_guard_role). 00019's "with whom never changes" holds for every other scope: the app role's
--     column grant is new, the trigger refuses any other use of it.
--   - The visit mode still never changes (00059), and the new slot's mode is checked by 00060's guard.

GRANT UPDATE (practitioner_role_id) ON appointment TO salus_app;

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION appointment_guard_change() RETURNS trigger
    LANGUAGE plpgsql
    SET search_path = pg_catalog
AS $$
DECLARE
    own boolean := NULLIF(current_setting('app.patient_id', true), '') IS NOT NULL
                   AND NULLIF(current_setting('app.patient_id', true), '')::uuid = OLD.patient_id;
    clinic boolean := NULLIF(current_setting('app.patient_id', true), '') IS NULL
                      AND NULLIF(current_setting('app.account_id', true), '') IS NULL
                      AND NULLIF(current_setting('app.organization_id', true), '')::uuid = OLD.organization_id;
    moved boolean := NEW.period IS DISTINCT FROM OLD.period OR NEW.slot_id IS DISTINCT FROM OLD.slot_id;
BEGIN
    -- session_user, not current_user: a SECURITY DEFINER function owned by the owner runs as it, yet stays guarded.
    IF session_user = (SELECT r.rolname FROM pg_class c JOIN pg_roles r ON r.oid = c.relowner WHERE c.oid = TG_RELID) THEN
        RETURN NEW;
    END IF;
    IF moved AND NOT ((own OR clinic) AND OLD.status = 'booked' AND NEW.status = 'booked') THEN
        RAISE EXCEPTION 'appointment: only its patient or its clinic moves a booked appointment' USING ERRCODE = 'insufficient_privilege';
    END IF;
    IF (NEW.period IS DISTINCT FROM OLD.period) <> (NEW.slot_id IS DISTINCT FROM OLD.slot_id) THEN
        RAISE EXCEPTION 'appointment: a reschedule changes the period and the slot together' USING ERRCODE = 'check_violation';
    END IF;
    IF NEW.period IS DISTINCT FROM OLD.period AND own AND NEW.reschedule_count IS DISTINCT FROM OLD.reschedule_count + 1 THEN
        RAISE EXCEPTION 'appointment: a reschedule counts once' USING ERRCODE = 'check_violation';
    END IF;
    IF NEW.period IS DISTINCT FROM OLD.period AND NOT own
       AND NEW.reschedule_count IS DISTINCT FROM OLD.reschedule_count
       AND NEW.reschedule_count IS DISTINCT FROM OLD.reschedule_count + 1 THEN
        RAISE EXCEPTION 'appointment: a move by the clinic counts once at most' USING ERRCODE = 'check_violation';
    END IF;
    IF NEW.period IS NOT DISTINCT FROM OLD.period AND NEW.reschedule_count IS DISTINCT FROM OLD.reschedule_count THEN
        RAISE EXCEPTION 'appointment: the reschedule count changes only with the period' USING ERRCODE = 'check_violation';
    END IF;
    IF NEW.confirmed_at IS DISTINCT FROM OLD.confirmed_at OR NEW.confirmed_via IS DISTINCT FROM OLD.confirmed_via THEN
        IF NOT own THEN
            RAISE EXCEPTION 'appointment: only its patient confirms' USING ERRCODE = 'insufficient_privilege';
        END IF;
        IF OLD.confirmed_at IS NOT NULL THEN
            RAISE EXCEPTION 'appointment: a confirmation never changes' USING ERRCODE = 'check_violation';
        END IF;
        IF OLD.status <> 'booked' THEN
            RAISE EXCEPTION 'appointment: only a booked appointment is confirmed' USING ERRCODE = 'check_violation';
        END IF;
    END IF;
    RETURN NEW;
END
$$;
-- +goose StatementEnd

-- Fires whenever an UPDATE names practitioner_role_id, so a statement that names it outside the clinic's scope is
-- refused even when it sets the same value (as the missing grant refused it before). SECURITY DEFINER: the role's
-- clinic is read as the owner, whatever practitioner_role's policies show the scope.
-- +goose StatementBegin
CREATE FUNCTION appointment_guard_role() RETURNS trigger
    LANGUAGE plpgsql
    SECURITY DEFINER
    SET search_path = pg_catalog, public
AS $$
DECLARE
    clinic boolean := NULLIF(current_setting('app.patient_id', true), '') IS NULL
                      AND NULLIF(current_setting('app.account_id', true), '') IS NULL
                      AND NULLIF(current_setting('app.organization_id', true), '')::uuid = OLD.organization_id;
BEGIN
    IF session_user = (SELECT r.rolname FROM pg_class c JOIN pg_roles r ON r.oid = c.relowner WHERE c.oid = TG_RELID) THEN
        RETURN NEW;
    END IF;
    IF NOT clinic THEN
        RAISE EXCEPTION 'appointment: only its clinic gives a visit to another practitioner' USING ERRCODE = 'insufficient_privilege';
    END IF;
    IF NEW.practitioner_role_id IS DISTINCT FROM OLD.practitioner_role_id THEN
        IF NOT (OLD.status = 'booked' AND NEW.status = 'booked' AND NEW.slot_id IS DISTINCT FROM OLD.slot_id) THEN
            RAISE EXCEPTION 'appointment: another practitioner only with a move of a booked visit' USING ERRCODE = 'check_violation';
        END IF;
        IF NOT EXISTS (SELECT 1 FROM public.practitioner_role r
                       WHERE r.id = NEW.practitioner_role_id AND r.organization_id = OLD.organization_id) THEN
            RAISE EXCEPTION 'appointment: a practitioner of its own clinic' USING ERRCODE = 'check_violation';
        END IF;
    END IF;
    RETURN NEW;
END
$$;
-- +goose StatementEnd

CREATE TRIGGER appointment_guard_role BEFORE UPDATE OF practitioner_role_id ON appointment
    FOR EACH ROW EXECUTE FUNCTION appointment_guard_role();
