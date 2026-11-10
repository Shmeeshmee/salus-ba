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

## SAL-12: the transcription consent (PR #49, merged 2026-11-10)

As built (agent decisions, for the owner to review; the copy waits for the owner, the doctor partner and the
lawyer's Q16):

| Topic | Decision |
| --- | --- |
| Consent | A second FHIR Consent per patient and clinic (`consent.KindTranscription`) in its own table `transcription_consent` (00040; reserved as 00038, renumbered at merge), the shape, row-level security and grants of `consent`; owner `consent`, retention `keep`. Its own table because 14 policies read `consent` as the care consent, which a transcription consent must never open. JSON: `kind` (`care`, the default, or `transcription`) on `POST /consents` and every Consent; `/me/consents` lists both. Codes as the care consent's (`patient-privacy`, `OPTIN`, `TREAT`, the clinic as `IRCP`) plus category `v3-ActCode#ICOL` and `provision.action` `consentaction#collect` and `#use` |
| Rule | `access.Transcribed`: both consents active. Revoking the care consent also revokes the transcription consent with that clinic (same transaction, audited), so it is asked again after care is given again |
| The agent | `transcription_join` is enqueued only when both are active; `AgentJoin` and `AgentAdmitted` (also the check once connected) refuse with `ErrConsentRequired`. A withdrawal enqueues `transcription_consent_revoked`, whose job removes only the agent from the patient's open rooms with the clinic (`videosession.EndAgentAccess`); the call goes on |
| What else needs it | Storing the transcript, the suggestions (record and worker) and drafting (`403 transcription_consent_required`) need the transcription consent too: they process the transcript. Reading the transcript or the stream, signing and reading documents stay on the care consent: they are the care record |
| Mobile | The notice before the device check while the consultation's status has no `transcription`: "Agree and continue" gives it, "Continue without transcription" asks for nothing. A decline is not stored: asked before every call while none is active. The "Transcription is on" badge withdraws it after a confirmation; My consents shows each kind with its own withdraw dialog. Android M35 and the iOS camera and microphone purposes changed. No "turn it back on" in the call: the next call's notice asks again. A clinic no longer in the directory gives the app no clinic to grant to: its call starts without the notice and without the agent |
| Web | Patients never use it; the practitioner sees "Transcription is off…" while the agent is absent for want of the consent, and a refused draft says why |
| Docs | docs/plans/consult-intelligence.md ("Gates", SAL-4 "When the agent joins"), docs/legal/BRIEF.md Q16, docs/privacy/dpia.md, docs/consultation/SPEC.md and COPY.md (M37–M48, W42, W43), docs/attachments/COPY.md (M15, M16), docs/booking/SPEC.md, docs/i18n/TRANSLATIONS.md |

Merged with SAL-17 at merge time: `composition`'s access check takes SAL-17's `signed` (a patient reads signed
documents) and SAL-12's transcription consent for drafting.

Copy waiting for approval: mobile M37–M48 (the notice, agree and decline, the withdraw dialog and message, the kind
labels), changed M35 and iOS M15/M16 ("not recorded as audio or video"; "if you agree, what is said is transcribed
and translated"), web W42 and W43; ten mobile and five web languages, `legal` where it applies (PR #49 lists them).

Copilot: one comment fixed (a stale consultation status could update the call screen after leaving).
