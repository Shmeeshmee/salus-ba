-- The price list (docs/plans/clinic-cms.md, package B). Step 0 holds the first query; package B adds the rest.

-- name: Definitions :many
-- The services the scope shows (the clinic's, or the directory's active ones of listed clinics), by name.
SELECT id, updated_at, organization_id, name, code, price_minor, currency, duration_minutes, vat_rate_bp, modes, active,
       version_id
FROM charge_item_definition
ORDER BY name, id;
