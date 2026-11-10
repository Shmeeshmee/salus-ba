-- name: Templates :many
-- The clinic's templates, by title. By its id, not only row-level security: salusctl runs as the owner.
SELECT id, organization_id, title, language, status, type_code, fields, version_id, updated_at
FROM document_template
WHERE organization_id = @organization_id
ORDER BY title, id;

-- name: Template :one
SELECT id, organization_id, title, language, status, type_code, fields, version_id, updated_at
FROM document_template
WHERE id = @id;

-- name: TemplateForUpdate :one
SELECT id, organization_id, title, language, status, type_code, fields, version_id, updated_at
FROM document_template
WHERE id = @id
FOR UPDATE;

-- name: Translations :many
-- The translations of the templates, in the enum's order.
SELECT template_id, language, texts
FROM document_template_translation
WHERE template_id = ANY(@template_ids::uuid[])
ORDER BY template_id, language;

-- name: InsertTemplate :one
INSERT INTO document_template (organization_id, title, language, status, type_code, fields, created_at, updated_at)
VALUES (@organization_id, @title, @language::language, @status, @type_code, @fields, @at, @at)
RETURNING id, organization_id, title, language, status, type_code, fields, version_id, updated_at;

-- name: UpdateTemplate :one
UPDATE document_template
SET title = @title, language = @language::language, status = @status, type_code = @type_code, fields = @fields,
    updated_at = @at, version_id = version_id + 1
WHERE id = @id
RETURNING id, organization_id, title, language, status, type_code, fields, version_id, updated_at;

-- name: DeleteTranslations :exec
DELETE FROM document_template_translation WHERE template_id = @template_id;

-- name: InsertTranslation :exec
INSERT INTO document_template_translation (template_id, language, texts)
VALUES (@template_id, @language::language, @texts);

-- name: SaveDraft :one
-- The encounter's open draft of the template: a new one, or the open one's drafted content replaced as its next
-- version. inserted tells which.
INSERT INTO document_draft (encounter_id, template_id, language, title, type_code, fields, created_at, updated_at)
VALUES (@encounter_id, @template_id, @language::language, @title, @type_code, @fields, @at, @at)
ON CONFLICT (encounter_id, template_id) WHERE status = 'preliminary'
DO UPDATE SET language = EXCLUDED.language, title = EXCLUDED.title, type_code = EXCLUDED.type_code, fields = EXCLUDED.fields,
    updated_at = EXCLUDED.updated_at, version_id = document_draft.version_id + 1
RETURNING id, (xmax = 0)::bool AS inserted;

-- name: Draft :one
SELECT id, encounter_id, template_id, status, language, title, type_code, fields, version_id, updated_at,
       signed_at, signer_role_id, document_reference_id, provenance_id, pdf_size, pdf_kek_id, pdf_sha1, pdf_wrapped_key
FROM document_draft
WHERE id = @id;

-- name: DraftForUpdate :one
SELECT id, encounter_id, template_id, status, language, title, type_code, fields, version_id, updated_at,
       signed_at, signer_role_id, document_reference_id, provenance_id, pdf_size, pdf_kek_id, pdf_sha1, pdf_wrapped_key
FROM document_draft
WHERE id = @id
FOR UPDATE;

-- name: ByDocumentReference :one
SELECT id, encounter_id, template_id, status, language, title, type_code, fields, version_id, updated_at,
       signed_at, signer_role_id, document_reference_id, provenance_id, pdf_size, pdf_kek_id, pdf_sha1, pdf_wrapped_key
FROM document_draft
WHERE document_reference_id = @id;

-- name: ByProvenance :one
SELECT id, encounter_id, template_id, status, language, title, type_code, fields, version_id, updated_at,
       signed_at, signer_role_id, document_reference_id, provenance_id, pdf_size, pdf_kek_id, pdf_sha1, pdf_wrapped_key
FROM document_draft
WHERE provenance_id = @id;

-- name: Drafts :many
SELECT id, encounter_id, template_id, status, language, title, type_code, fields, version_id, updated_at,
       signed_at, signer_role_id, document_reference_id, provenance_id, pdf_size, pdf_kek_id, pdf_sha1, pdf_wrapped_key
FROM document_draft
WHERE encounter_id = @encounter_id
ORDER BY created_at, id;

-- name: Sign :one
-- The draft's fields as signed, final, with its signature; only from the version read.
UPDATE document_draft
SET status = 'final', fields = @fields, signed_at = @at, signer_role_id = @signer_role_id,
    document_reference_id = @document_reference_id, provenance_id = @provenance_id, pdf_size = @pdf_size,
    pdf_kek_id = @pdf_kek_id, pdf_sha1 = @pdf_sha1, pdf_wrapped_key = @pdf_wrapped_key,
    updated_at = @at, version_id = version_id + 1
WHERE id = @id AND version_id = @version_id AND status = 'preliminary'
RETURNING id;
