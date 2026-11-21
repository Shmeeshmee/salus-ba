-- The price list (docs/plans/clinic-cms.md, package B).

-- name: Definitions :many
-- The services of a clinic that the scope shows (every one in the clinic's scope, the active ones of a listed clinic in
-- the directory's), by name.
SELECT id, updated_at, organization_id, name, code, price_minor, currency, duration_minutes, vat_rate_bp, modes, active,
       version_id
FROM charge_item_definition
WHERE organization_id = @organization_id
ORDER BY name, id;

-- name: DefinitionTranslations :many
-- The translated names of the services ids, where the scope shows the service.
SELECT charge_item_definition_id, language, name
FROM charge_item_definition_translation
WHERE charge_item_definition_id = ANY (@ids::uuid[])
ORDER BY charge_item_definition_id, language;

-- name: InsertDefinition :one
INSERT INTO charge_item_definition (created_at, updated_at, organization_id, name, code, price_minor, currency,
                                    duration_minutes, vat_rate_bp, modes, active)
VALUES (@at, @at, @organization_id, @name, @code, @price_minor, @currency, @duration_minutes, @vat_rate_bp, @modes,
        @active)
RETURNING id, version_id;

-- name: UpdateDefinition :one
-- Optimistic locking: no row when the service is not at version_id (or is not the scope's).
UPDATE charge_item_definition
SET name = @name, code = @code, price_minor = @price_minor, currency = @currency,
    duration_minutes = @duration_minutes, vat_rate_bp = @vat_rate_bp, modes = @modes, active = @active,
    version_id = version_id + 1, updated_at = @at
WHERE id = @id AND version_id = @version_id
RETURNING version_id;

-- name: DefinitionExists :one
SELECT EXISTS (SELECT 1 FROM charge_item_definition WHERE id = @id);

-- name: DeleteDefinitionTranslations :exec
-- Saving a service replaces its translations.
DELETE FROM charge_item_definition_translation
WHERE charge_item_definition_id = @charge_item_definition_id;

-- name: InsertDefinitionTranslation :exec
INSERT INTO charge_item_definition_translation (charge_item_definition_id, language, name)
VALUES (@charge_item_definition_id, @language, @name);
