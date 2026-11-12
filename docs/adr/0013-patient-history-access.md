# 13. Patient history: own visits, chosen visits from other clinics, a visit-scoped profile

Status: accepted, 2026-11-12 (epic SAL-38, docs/plans/patient-history.md). The owner's decisions of 2026-11-12 are
recorded under "Decisions"; this ADR builds on the care consent of docs/booking/SPEC.md and the access rules in
`internal/access`.

## Context

SAL-38 gives the patient one history of every clinic they have seen, gives the practitioner the history they need
before and during a visit, keeps allergies and current medications in the patient's profile, and lets the patient
share visits from other clinics with a clinic. Health data is special-category data (Art. 9). A patient sees
several clinics, and a visit to one (psychiatry, sexual health, an embarrassing treatment) is often none of
another's business. The constraint of the epic: nothing from another clinic is shown, hinted at or used in AI
suggestions without the patient's consent.

What exists: the care consent per patient and clinic (00016), the transcription consent (00040), row-level
security in a patient scope (`app.patient_id`) and a clinic scope (`app.organization_id`), and the pure rules of
`internal/access`, where only an appointment's own practitioner sees its consultation's record
(`access.ConsultationRecord`).

## Decisions

Owner, 2026-11-12:

1. **The profile is minimal.** It holds allergies and current medications only (safety-critical for every doctor).
   Anything specialty-specific (pregnancy and birth history, surgeries, family history) is asked in each clinic's
   questionnaires and goes only to that clinic.
2. **Cross-clinic sharing is off by default** and per visit. Turned on at booking, it shows the patient's past
   visits at other clinics, each with a checkbox; only the ticked ones are shared. An unticked visit does not exist
   for the doctor: not in the history, search, AI suggestions, the pre-visit summary or any count.
3. **Own visits only.** At their own clinic a practitioner sees only the visits they held themselves, as
   `access.ConsultationRecord` already says. A colleague's visit at the same clinic stays hidden.
4. **Doctor entries are visit-scoped.** An allergy or medication the patient enters is in the profile and seen by
   every clinic with their care consent and an appointment. One a practitioner records belongs to that visit: seen
   where the visit is seen, and elsewhere only once the patient shares the visit or adds the entry to their
   profile ("Add to my profile"). A confirmation shows elsewhere as "confirmed by a clinician", without the clinic
   or the practitioner.

## Access model

One pure rule decides what of the history a viewer sees, `access.HistoryVisit(viewer, facts, visit)`, with the
facts gathered for the viewer's clinic:

| Viewer | Sees a visit when |
| --- | --- |
| The patient | Always: every visit of theirs at every clinic (their own record, Art. 15). Never a draft |
| A practitioner, visit at their own clinic | They held it, and the patient's care consent with the clinic is active |
| A practitioner, visit at another clinic | The care consent and the history consent with the practitioner's clinic are active, the history consent lists the visit, and the practitioner has an appointment with the patient at their clinic (the profile's rule, `access.Profile`) |
| The clinic's admin, anyone else | Never |

What a seen visit brings: its encounter, its signed documents (final, never a draft), its completed questionnaire
responses and their clean files, and the entries recorded in it. Never its transcript, suggestions, drafts or the
appointment. Allergies and medications follow `access.ProfileEntry`: in the profile, where `access.Profile` is
Full; recorded in a visit, where `HistoryVisit` sees the visit. `access.Attributed` shows which clinic recorded or
confirmed an entry only to the patient and that clinic.

Everything a viewer sees of the history is built from that one rule. A service that lists, counts, searches or
summarises the history (SAL-39, SAL-40) filters with it before it does anything else, so a visit it does not pass
cannot leak through a count, an empty state or a search result.

**Cross-clinic only through a FHIR Consent.** The history consent is a third kind of Consent per patient and
receiving clinic (table `history_consent`, 00041; category v3-ActCode IDSCL, action disclose, purpose TREAT), listing
the shared visits in `provision.data` (`Encounter/<id>`, meaning `instance`: `dependents` would claim every resource
that refers to the visit, its transcript included). Each selection is a row of
`history_consent_encounter` with its `added_at` and `removed_at`. A removal or a revocation ends the clinic's access
when its transaction commits, and revoking the care consent revokes the history consent with it. An active history
consent lists at least one visit: removing the last one revokes it.

**Defence in depth.** Row-level security says the same as the rule, read-only, in the receiving clinic's scope:
`history_shared(encounter, patient)` is true only while the history consent is active and lists the visit and the
care consent is active, and the `*_shared` policies on encounter, document_draft, questionnaire_response, binary,
organization and practitioner_role use it. Opening another clinic's encounter would have opened the
policies that follow encounter's visibility (the transcript's and the suggestions'), so 00041 recreates those for the
clinic's own encounters. The patient lists only their own finished visits at other clinics; a removal and a
revocation are final; the clinic sees only the open selections of an active consent. A trigger
(`profile_entry_update_check`) says what row-level security cannot: the patient never confirms an entry or edits a
clinic's, a clinic never moves an entry into or out of the profile, confirms only as itself, never over another
clinic's confirmation, and edits only its own; an edit of a confirmed entry clears or renews the confirmation unless
the confirming clinic makes it, and a removed (entered-in-error) entry never changes again. A shared visit is read-only for the receiving clinic, its drafts included (00031's
`document_draft_clinic` is narrowed to the clinic's own encounters like the transcript's).
Which practitioner of the clinic sees a shared visit is the service's rule, as for the clinic's own rows.

**Everything audited.** Every read of a history item is an AuditEvent by its owning domain (who, the resource, when,
purpose TREAT or PATRQT), every view of the history and the summary too, and every grant, change, removal and
revocation of a history consent, in the same transaction.

**No AI.** No part of the history (shared or not) is sent to the language model in this epic: suggestions and
drafting keep reading only the consultation's own transcript. The allergy warning in drafting is shown in the
editor only. No processor is added (docs/privacy/processors.md).

## Consequences

- A practitioner sees less than a clinic-wide record would give them. A colleague's visit is not shareable
  through the history consent, which lists only other clinics' visits: if that is needed later, it is a new
  decision.
- A doctor's allergy entry may not reach the next clinic unless the patient adds it to the profile. The patient
  app offers that next to the entry.
- The tables a history reads are owned by five domains. The history package owns no table and goes through their
  services (archcheck), each with a method that applies `access.HistoryVisit` and audits.
- Row-level security now has a third way into encounter and the clinical tables. Its tests
  (`migrations/patient_history_test.go`) cover each scope, each table, the removal, both revocations and the
  insert checks.
