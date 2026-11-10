# v1-finish: progress

The running record of docs/plans/v1-finish.md, one section per merged ticket, written at merge time.

## SAL-17: the patient's signed documents and transcript copy (PR #47, merged 2026-11-10)

As built (agent decisions, for the owner to review):

| Topic | Decision |
| --- | --- |
| What the patient reads | The signed documents of their own consultations: the final `Composition`, its `DocumentReference` and `Provenance`, and the PDF (`GET /document-references/{id}/content`); never a draft. `GET /fhir/R4/DocumentReference?encounter=<id>` lists them (CapabilityStatement updated). Rules `access.SignedDocument` and `access.TranscriptCopy`; row-level security `document_draft_own` (final rows only) and `transcript_segment_own`, migration 00038 (reserved as 00040, renumbered at merge). Audited with purpose PATRQT |
| Consent withdrawal | Does not take a signed document or the transcript copy from the patient: their own record (GDPR Art. 15), as the existing rule for their profile, appointments and questionnaire responses. The practitioner loses access at once, as before |
| Transcript copy | `GET /encounters/{id}/transcript` answers the patient only while the clinic shares transcripts: `organization.patient_transcript_copy`, off by default, set by the clinic admin in web/ (one checkbox). An Organization extension (`organization-patient-transcript-copy`, StructureDefinition in api/fhir), like the clinic's other settings, not an OpenAPI endpoint. Checked in the service, not in row-level security (the patient's scope cannot see the clinic's row). The live stream stays the practitioner's. This is the patient transcript access docs/plans/consult-intelligence.md "Personal data" left to "the export feature"; `Patient/$everything` is still to come |
| Language | Documents in the app's language first; a signed document is never translated or changed |
| Notification | E-mail only (`document_signed_email`, River, appointment and clinic ids; enqueued in the signing transaction), ten languages, naming only the clinic, the date and the app link. No push channel in the stack yet |
| Mobile | A "Documents" button on started, not cancelled appointments; each document rendered as text (one `Text` per label and line, for SAL-13's tap-to-explain in `DocumentScreen`, `mobile/lib/documents_screen.dart`); PDF and transcript to the share sheet through `share_plus` 13.3.1 (the one new dependency), the temporary PDF deleted after sharing and swept at start |

Copy waiting for the owner's approval: web `admin.transcriptCopy`, `admin.transcriptCopyHint`; mobile `documents`,
`noDocuments`, `documentSignedOn`, `documentLanguage`, `sharePdf`, `transcript`, `transcriptNote`, `transcriptEmpty`,
`speakerYou`, `speakerPractitioner`, `shareTranscript`, `transcriptTranslation`; the e-mail `signed.txt` (subject and
body), all in their languages (PR #47 lists the English).

Copilot: one comment fixed (the refused read's OperationOutcome wording), one declined (an explicit
`dart:typed_data` import: re-exported by `flutter/services.dart`; the explicit import trips `unnecessary_import`).

## SAL-14: the terminology review panel (PR #48, merged 2026-11-10)

As built (agent decisions, for the owner to review):

| Topic | Decision |
| --- | --- |
| Permission | `term_reviewer` (00039, owner `auth`, personal data, retention `keep` while granted; revoking or deleting the account deletes the row). Granted and revoked only with `salusctl term-reviewer grant\|revoke <account-id>` (audited, refuses a patient); `salus_app` reads it only. `GET /auth/me` says `termReviewer` |
| Grants (00039) | `salus_app` inserts and deletes `term_label` rows; inserts into `term_translation` and `term_explanation` and updates only their review and text columns; updates only `term`'s review columns. 00039 also fills the translation rows missing for terms imported after 00037 |
| Entry | One term in one language: its labels (the preferred one and synonyms; at most 20 synonyms, 300 characters, no `\|`; a preferred label is required) and its explanation. needs_changes if either part needs changes; else unreviewed if the term has no label there or either part is unreviewed; else approved. A missing explanation does not count. A term with no label in a language is in that language's unreviewed queue |
| Transitions | Any edit (also of an entry that needs changes) sets it back to unreviewed and keeps its last reviewer, time and note; an unchanged save changes nothing. Approve and needs changes set `reviewed_by` and `reviewed_at`; needs changes requires a note (`note_required`). Every write takes If-Match (`W/"0"` for a new entry) and locks the term; stale is 412 |
| **The term's own status (for the owner to confirm)** | `term.review_status` (the cheatsheet's mark and the CSV) follows its translations in Latin and the five staff languages only, so a term can be approved without German, Arabic, Turkish, Italian and French reviews; explanations do not change it |
| CSV round trip | A term has a translation row in exactly the languages it has labels in (seed import, CSV import and panel keep it so); a language added by the CSV takes the term's status; a status changed by the CSV is set on every translation of the term, with no reviewer |
| Audit | Every edit, approval and request for changes is an AuditEvent (`entity.name` the table, as the document templates; a term is no FHIR resource), in the write's transaction; a refusal to a non-reviewer is audited as a minor failure. Reads are not audited (no patient data) |
| API | `GET /terminology/review` (keyset paging with `after`/`next`), `GET /terminology/review/progress`, `PUT` and `POST …/review` on `/terminology/terms/{id}/translations/{language}` and `/terminology/terms/{id}/explanations/{language}` |
| Web | `/term-review`, "Term review" in the navigation for reviewers (others see it read-only); language and status filters (unreviewed by default); progress for the selected language, the others behind a disclosure; keyboard: ↓/↑ or j/k, a/e/n on the labels, Shift+A/E/N on the explanation, Enter saves, Escape cancels |

Copy waiting for the owner's approval: about 45 strings in `web/src/i18n/review/*.json` and the navigation label
"Term review" (`web/src/i18n/shell`), five staff languages (PR #48 lists them), including the names of the five
patient-only languages.

Copilot: two comments, both fixed (a page token that is no UUID is now a 400; the update grants are column-level).

Noted: `internal/terminology/repository_postgres.go` is about 404 lines and `web/src/terminology/Review.tsx` 568,
over CLAUDE.md's "about 400" (archcheck lets only `repository_postgres.go` import the sqlc package).
