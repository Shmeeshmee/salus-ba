-- The migration drill's checks at head (scripts/migration-drill.sh) on the database restored from the 00059 dump of
-- seed-00059.sql and migrated: the rows of 00059 are all there, unchanged in number, and the backfills of the later
-- migrations hold. :before is the counts at 00059, in the same form as the first query's result. Any failed check
-- raises an error, so psql (ON_ERROR_STOP) exits non-zero.
\set ON_ERROR_STOP on
SELECT string_agg(t || '=' || n, ' ' ORDER BY t) AS after FROM (
	SELECT 'account' t, count(*) n FROM account UNION ALL SELECT 'patient', count(*) FROM patient
	UNION ALL SELECT 'appointment', count(*) FROM appointment UNION ALL SELECT 'encounter', count(*) FROM encounter
	UNION ALL SELECT 'questionnaire_response', count(*) FROM questionnaire_response
	UNION ALL SELECT 'communication', count(*) FROM communication UNION ALL SELECT 'audit_event', count(*) FROM audit_event
	UNION ALL SELECT 'consent', count(*) FROM consent) c \gset
\echo at head: :after
SELECT :'before' = :'after' AS same \gset
\if :same
\else
  \echo the counts changed between 00059 and head
  SELECT 1/0;
\endif

DO $$
DECLARE n bigint;
BEGIN
  -- 00059: everything that existed is a video visit, and every availability rule carries its mode.
  SELECT count(*) INTO n FROM appointment WHERE visit_mode <> 'video';
  IF n <> 0 THEN RAISE EXCEPTION '% appointments are not video', n; END IF;
  SELECT count(*) INTO n FROM slot WHERE visit_mode <> 'video';
  IF n <> 0 THEN RAISE EXCEPTION '% slots are not video', n; END IF;
  SELECT count(*) INTO n FROM schedule, jsonb_array_elements(availability) r WHERE r->>'mode' IS NULL;
  IF n <> 0 THEN RAISE EXCEPTION '% availability rules have no mode', n; END IF;
  -- 00080: every clinic that existed has its first location.
  SELECT count(*) INTO n FROM organization o WHERE NOT EXISTS (SELECT 1 FROM location l WHERE l.organization_id = o.id);
  IF n <> 0 THEN RAISE EXCEPTION '% clinics have no location', n; END IF;
  -- The visits keep their states, slots and periods.
  SELECT count(*) INTO n FROM appointment a JOIN slot s ON s.id = a.slot_id WHERE a.period = s.period;
  IF n <> 3 THEN RAISE EXCEPTION 'appointments on their slots: %, want 3', n; END IF;
  SELECT count(*) INTO n FROM appointment WHERE status = 'cancelled' AND cancellation_reason = 'pat';
  IF n <> 1 THEN RAISE EXCEPTION 'cancelled appointments: %, want 1', n; END IF;
  -- The consents keep their state; the revoked one stays revoked.
  SELECT count(*) INTO n FROM consent WHERE status = 'inactive' AND revoked_at IS NOT NULL;
  IF n <> 1 THEN RAISE EXCEPTION 'revoked consents: %, want 1', n; END IF;
END $$;

-- The application role reads through row-level security as before: a clinic sees its own patients' visits only.
SET ROLE salus_app;
SET app.organization_id = '00000000-0000-4000-8000-0000000000a2';
DO $$
DECLARE n bigint;
BEGIN
  SELECT count(*) INTO n FROM appointment;
  IF n <> 0 THEN RAISE EXCEPTION 'clinic two sees % of clinic one''s appointments', n; END IF;
END $$;
SET app.organization_id = '00000000-0000-4000-8000-0000000000a1';
DO $$
DECLARE n bigint;
BEGIN
  SELECT count(*) INTO n FROM appointment;
  IF n <> 3 THEN RAISE EXCEPTION 'clinic one sees % appointments, want 3', n; END IF;
END $$;
RESET ROLE;
\echo checks at head: passed
