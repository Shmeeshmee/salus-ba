-- +goose Up
-- Package I of docs/plans/clinic-cms.md (Q4): a clinic whose membership is suspended (30 days past due) or cancelled
-- is not in the public directory. The directory's policy on organization gains the membership; the directory policies
-- of practitioner, practitioner_role, charge_item_definition and (through the roles) schedule and slot read organization
-- under its own policy, so they hide the clinic's practitioners, services, schedules and free slots with it. Only the
-- directory scope (app.directory = 'public') changes: the clinic's own scope, the patients' and the shared lookups do
-- not, so existing appointments, records, documents, exports and subject rights go on. Nothing here is personal data.
--   - clinic_subscription_directory: the directory scope reads the membership of a stopped clinic only (its id and
--     status), for the policy below; no grant changes, and the app still has no UPDATE on subscription_invoice.

DROP POLICY IF EXISTS clinic_subscription_directory ON clinic_subscription;
CREATE POLICY clinic_subscription_directory ON clinic_subscription FOR SELECT
    USING (status IN ('suspended', 'cancelled') AND NULLIF(current_setting('app.directory', true), '') = 'public');

DROP POLICY IF EXISTS organization_directory ON organization;
CREATE POLICY organization_directory ON organization FOR SELECT
    USING (listed
           AND NULLIF(current_setting('app.directory', true), '') = 'public'
           AND NOT EXISTS (SELECT 1 FROM clinic_subscription s
                           WHERE s.organization_id = organization.id AND s.status IN ('suspended', 'cancelled')));
