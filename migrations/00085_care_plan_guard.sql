-- +goose Up
-- A treatment plan's status moves forward only, and a booked plan's steps stay as booked (docs/clinic-cms/WAVE6.md U6,
-- 6.0's note 8; the careplan service already keeps both, this holds every role to them, the owner too).
--   - care_plan: written as a draft; draft to active (booked) or revoked, active to completed or revoked, never back,
--     and a completed or revoked plan's status never changes again.
--   - care_plan_activity: a step with its visit (a booked plan's) is never deleted, nor moved to another plan or another
--     visit, and its visit is never dropped; a booked plan takes no new step. A draft's steps are laid out freely, and
--     booking gives each its visit. Renumbering (a moved visit) stays allowed.
-- Refusals are check violations (23514), as diagnostic_report_guard's.

-- +goose StatementBegin
CREATE FUNCTION care_plan_guard() RETURNS trigger
    LANGUAGE plpgsql
    SET search_path = pg_catalog, public
AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        IF NEW.status <> 'draft' THEN
            RAISE EXCEPTION 'a plan is written as a draft' USING ERRCODE = 'check_violation';
        END IF;
    ELSIF NEW.status IS DISTINCT FROM OLD.status
          AND NOT (OLD.status = 'draft' AND NEW.status IN ('active', 'revoked')
                   OR OLD.status = 'active' AND NEW.status IN ('completed', 'revoked')) THEN
        RAISE EXCEPTION 'a plan''s status moves forward only' USING ERRCODE = 'check_violation';
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd
CREATE TRIGGER care_plan_guard BEFORE INSERT OR UPDATE ON care_plan
    FOR EACH ROW EXECUTE FUNCTION care_plan_guard();

-- +goose StatementBegin
CREATE FUNCTION care_plan_activity_guard() RETURNS trigger
    LANGUAGE plpgsql
    SET search_path = pg_catalog, public
AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        IF EXISTS (SELECT 1 FROM care_plan p WHERE p.id = NEW.care_plan_id AND p.booked_at IS NOT NULL) THEN
            RAISE EXCEPTION 'a booked plan takes no new step' USING ERRCODE = 'check_violation';
        END IF;
        RETURN NEW;
    END IF;
    IF OLD.appointment_id IS NULL THEN
        RETURN CASE WHEN TG_OP = 'DELETE' THEN OLD ELSE NEW END;
    END IF;
    IF TG_OP = 'DELETE' THEN
        RAISE EXCEPTION 'a booked plan''s step is never deleted' USING ERRCODE = 'check_violation';
    END IF;
    IF NEW.care_plan_id IS DISTINCT FROM OLD.care_plan_id OR NEW.appointment_id IS DISTINCT FROM OLD.appointment_id THEN
        RAISE EXCEPTION 'a booked plan''s step keeps its plan and its visit' USING ERRCODE = 'check_violation';
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd
CREATE TRIGGER care_plan_activity_guard BEFORE INSERT OR UPDATE OR DELETE ON care_plan_activity
    FOR EACH ROW EXECUTE FUNCTION care_plan_activity_guard();
