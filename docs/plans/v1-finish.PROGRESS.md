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

## SAL-10: end-to-end tests of the consult pipeline (PR #46, merged 2026-11-11)

As built (agent decisions, for the owner to review):

| Topic | Decision |
| --- | --- |
| Fixtures | Synthetic, spoken by no one: `test/testdata/consult/{de,ar,tr}.json`, a German, an Arabic and a Turkish patient with a Bosnian doctor, every text labelled as a placeholder (`Platzhalter:`, `مثال:`, `Örnek:`, `Primjer:`) and checked absent from LiveKit's log. No audio files: each scripted line is a pure tone on a 50 Hz grid (500–3500 Hz), sent by the speaker's LiveKit participant as G.711 and decoded by the agent as usual |
| Test seam | `FAKE_ENGINE_SCRIPT` (internal/config), for the fake engines only (refused beside `azure` or `anthropic`): the fake recognizer (`stttest.Recognizer.Lines`) hears the tones and gives the line's partial and final in the stream's language; the fake model (`llmtest.Script`) gives the scripted translations, suggestions and draft fields. Set only by `deployments/docker-compose.e2e.yml` (api, worker, agent; fixtures mounted read-only) through `make e2e-up`; the compose default is the plain fakes. Everything between is the real pipeline |
| Journeys | German patient with the practitioner in Chromium (captions, translated lines, suggestions with the AI note pinned and dismissed, Generate, the missing-field flag, edit, sign, end); Arabic patient, both sides in Go (both directions, `suggestion_status` over the WebSocket, the stored transcript, a draft with sources, signed with If-Match); Turkish patient withdrawing SAL-12's transcription consent mid-call (the agent leaves, the call goes on, drafting is refused 403 `transcription_consent_required`). Every journey gives the transcription consent first |
| Latency | Caption: first tone frame to the partial on the practitioner's WebSocket; translation: last tone frame to the translated final; draft: Generate to the fields. A journey fails only when a hop's median is over the plan's budget (1 s, 2 s, 15 s). The report goes to `/e2e/consult-latency.md`, printed by `make e2e` (and `make e2e-report`), in CI's job summary and the `consult-latency` artifact. Local medians: caption 48–53 ms, translation 68–81 ms, draft 51–410 ms: with fakes that take no time, these are Salus's own pipeline only |
| Login limit | Every Go sign-in of the suite clears the `ratelimit:*` counters first, as `make e2e` did before the run: the suite now signs in more often than the production limit of 30 per IP in 15 minutes from the one container. The limit is unchanged |

Copilot: five comments fixed, two declined (reusing a test voice's frame buffer: about 50 small allocations a second
in a test helper, and reuse would depend on `WriteSample` not keeping it; the fake stream's mutex held while `say`
waits: `Recv` never takes it, and it keeps results in audio order). Copilot did not review the last pushes (the merge
of main with SAL-12 and the journeys' consent step); the orchestrator reviewed those, and CI was green.

No copy; no migration.

## SAL-13: plain-language explanations of medical terms (PR #50, merged 2026-11-11)

As built (agent decisions, for the owner to review; the explanations wait for the doctor partner's review in SAL-14's
panel):

| Topic | Decision |
| --- | --- |
| Drafts | `assets/terminology/explanations.draft.jsonl`: 216 terms in all ten app languages (2,160 texts), drafted by Claude (the English, then translated under one brief: Bosnian and Croatian standards, Serbian ijekavian in both scripts), marked as drafts in `assets/terminology/README.md`. The 300 most common terms by rank, minus 85 that need no explanation or don't belong in a tooltip (everyday body parts, chemical elements and lab chemicals, social events, feelings, suicide, abortion, recreational drugs), plus the tonsil (the ticket's example). `salusctl terminology import-explanations` (also in `make terminology-import`) writes them through SAL-14's service as `unreviewed`, no reviewer, never over an existing explanation; not audited, like the curator's CSV. No model is called at runtime; no processor |
| Tagging | `terminology.Service.Tag`, the glossary's matching with word positions (the glossary unchanged), in the text's own language, only terms with an explanation; UTF-16 spans, never stored. Server-side only, at reads that already exist and under their rule: the agent tags each final line and its translation (`TranscriptEvent.terms`); the stored transcript is tagged as it is read (Serbian's script detected from the letters); signed documents through `GET /compositions/{id}/terms` (the Composition's rule and AuditEvent). No endpoint takes text from a client |
| Explanations | `GET /terminology/explanations?language=` for any signed-in account: the viewer's language, else English; an explanation that needs changes is never shown. The whole list per language, so the terms found in a patient's text never leave the server |
| Web | A dotted underline and a tooltip on hover and focus in the transcript and the signed document; in captions the terms are explained in the caption's own tooltip (the caption is clipped to two lines); "Informational, not a diagnosis" (`legal`) and "Not reviewed yet" until approved. Editable draft fields get no overlay; a signed document is never changed |
| Mobile | Tap-to-explain in SAL-17's `DocumentScreen` and the transcript copy (its translations too); plain text when the terms can't be loaded. No device e2e in this repo: a widget tap test and an e2e API journey of the two calls the app makes |

Copy waiting for approval: web `terminology.explainNote` (five languages, `legal`); mobile `termExplanationNote`
(`legal`), `termNotReviewed`, `termClose` (ten languages); and the 2,160 explanations (the doctor partner, in the panel).

For the curator (seed labels the translators flagged): tr "Hepatit c" as a label of jaundice; tr "Kabartma tozu"
(baking powder) for sodium bicarbonate; hr "ugljikov oksid" (reads as carbon monoxide) for carbon dioxide; sr "grudna
žlezda" for the thymus.

Copilot did not review this PR (it ran on none of its pushes, requested four times); the orchestrator reviewed the
tagging paths (access, audit, nothing of the text logged), and CI was green.
