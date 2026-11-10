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
