# 12. ICD-10 codes as identifiers, our own diagnosis groups, pre-computed aggregates, suppressed platform figures

Status: proposed, 2026-11-12 (docs/plans/analytics.md, epic SAL-18). Waits for the owner's approval. The points
under "Open legal questions" in the plan are for a lawyer before real patient data reaches the platform view.

## Context

SAL-18 asks for two analytics views: a clinic's own statistics (SAL-20) and platform-wide trends by region with
no clinic or patient identifiers (SAL-21), both drawn from coded diagnoses on signed documents and from
appointments, never from transcripts, and grouped into diagnosis groups based on ICD-10 blocks (SAL-19).

What exists: a document template can have `coded` fields holding one ICD-10 code
(`internal/composition/model.go`, system `http://hl7.org/fhir/sid/icd-10`, format checked on save); signed
documents are `document_draft` rows with status `final`; the terminology list (`internal/terminology`) has
Wikidata items with their ICD-10 code (P494, CC0) and a review panel for labels and translations. There is no
FHIR `Condition`, no clinic address, and no aggregate of any kind. Every clinic table is row-level secured by
`app.organization_id`. Nothing is deployed yet (ADR 0011), so there is no production data to backfill.

Three things need a decision that outlasts the feature: where the codes and groups come from (a licence
question), how the figures are computed (a performance and correctness question), and what makes the platform
figures anonymous (a GDPR question, CLAUDE.md top rule 4).

## Options for the ICD-10 source

| Source | Licence | Usable for us |
| --- | --- | --- |
| WHO ICD-10 (2019 edition), codes, titles, block structure | WHO classifications are licensed [CC BY-ND 3.0 IGO](https://www.who.int/about/policies/publishing/copyright): commercial use allowed with attribution and **no adaptation**; translations and mappings need WHO's written agreement ([ICD-11 licence](https://icd.who.int/en/docs/icd11-license.pdf), referred to by the classifications page). Older ICD-10 licences (e.g. the [UK TRUD terms](https://isd.hscic.gov.uk/trud/users/guest/filters/0/licence/22)) were stricter and WHO reserved the right to license ICD-10 for a fee | The codes as identifiers, yes. WHO's titles, no: we show them in nine languages WHO has not licensed to us and next to our own group names, which is an adaptation and a translation |
| Croatian MKB-10 (HZJZ) | A translation made under agreement with WHO; no open licence published (no licence terms found on hzjz.hr) | No, without a written licence from HZJZ (and WHO) |
| BiH and Serbian MKB-10 editions (entity public-health institutes; the Serbian edition is © Institut za javno zdravlje Srbije "Batut", 2013) | All rights reserved, made from WHO's original; the Slovenian analogue shows WHO assigning the translation's copyright to the national institute | No, without a written licence |
| US ICD-10-CM (NCHS) | Published by the US government, free to use, but a clinical modification of WHO's classification under WHO's authorisation, with codes and titles that differ from WHO ICD-10 beyond three characters | No: not the classification doctors in BiH use, and the rights in the underlying WHO content are not clearly free outside the US |
| Wikidata P494 (ICD-10 code of an item) | [CC0 1.0](https://www.wikidata.org/wiki/Wikidata:Licensing) | Yes; already our terminology source (docs/terminology/SOURCES.md) |

## Decision

**ICD-10 source.** We keep the decision already taken for the terminology (docs/terminology/SPEC.md §7,
SOURCES.md) and extend it: an ICD-10 code is used only as an identifier of a diagnosis (a fact, not WHO's
expression). Codes come from the doctor (a `coded` field) and from the Wikidata items of the terminology list
(CC0). We take no WHO text: no titles, no descriptions, no inclusion or exclusion notes, in any language. A code
on screen is shown with the terminology's own label (Wikidata's, or the partner's reviewed label) when there is
one, and with the code alone otherwise. A code is checked by format (`^[A-Z][0-9]{2}(\.[0-9A-Z]{1,4})?$`, as
today) and by falling into a diagnosis group's range; it is not checked against a WHO code list we do not hold.
ICD-10 is shown as "MKB-10" in Bosnian, Croatian and Serbian (terminology SPEC). Why this and not a national
edition: it is the only option that is legally usable without a licence we do not have, it needs no new data
import, and it changes nothing the terminology already decided. The cost: no official titles in the UI, and a
mistyped but well-formed code (e.g. `J99.9`) is accepted. A licence from WHO or a national institute can be added
later without a data model change (titles would go into the terminology tables beside the Wikidata labels).

**Diagnosis groups.** Our own grouping, 27 groups and 7 subgroups, each defined by ranges of ICD-10
codes that follow ICD-10 block boundaries (e.g. "Respiratory infections" J00–J22, subgroup "Common cold" J00).
The ranges are code intervals, which we treat as facts; the names are our own plain-language wording, written by
us in all ten languages and reviewed in the term review panel like term translations, never WHO's block titles
or a translation of them. A code belongs to exactly one group and at most one subgroup (tested); a code in no
range belongs to "Other". The list is reference data owned by `terminology`, seeded by a migration
(the plan's appendix A).

**Source of the figures.** A FHIR `Condition` (new domain `condition`, bounded context `clinical`), one per
coded field of a signed document, written in the signing transaction, `verificationStatus` `confirmed`
(the signing practitioner confirmed it: a model-suggested code must be confirmed field by field before signing).
Appointments come from the `appointment` service. Transcripts, suggestions and free text are never read.

**Pre-computed aggregates.** Counts per clinic, local day and hour (the clinic's `time_zone`), practitioner
role, specialty, appointment status, and diagnosis group, subgroup and code, in tables owned by a new domain
`clinicstats`. Incremental: the signing transaction, an appointment status change, an erasure and a retention
delete each enqueue a River job with opaque ids (`clinicstats_refresh{organization_id, day}`) that recomputes
that clinic-day (delete and insert in one transaction, idempotent). Nightly: the scheduler recomputes the last
35 days of every clinic (catches anything a lost job missed) and then builds the platform tables. Charts read
only aggregates; a yearly overview is a `SUM` over at most 366 × 24 rows per filter. A full recompute of one
clinic is the same job over a range (`salusctl`), for backfills and repairs.

**Reading across clinics.** The jobs that build aggregates read every clinic's rows, which `salus_app` cannot
(RLS). They log in as a new role `salus_stats`, as the retention job does with `salus_retention` (00014):
`BYPASSRLS`, owner of nothing, its own credentials loaded only by `cmd/worker`, and `SELECT` granted per column on
the source columns only (codes, times, statuses, clinic, practitioner role), never on a patient id, a name or a
field value, so the aggregate queries cannot read a patient identifier even by mistake. It writes only the
aggregate tables. The source queries stay in the owning domains (`condition`, `appointment`) and are reached
through their services.

**Platform figures** (domain `trends`, bounded context `analytics` with `clinicstats`). Built nightly from the
`condition` and `appointment` services (not from `clinicstats`, so the two can be built in parallel), into a
table that has no clinic, practitioner or patient column: ISO week,
region, city, specialty, group, subgroup, code, count, and the number of contributing clinics (used for
suppression, never returned). Every number the API returns passes the rules A1 to A10 of the plan's section 5,
applied in one pure function (`trends.Suppress`) at query time on the pre-computed cells: a minimum count of 10
(configurable, never below 10), at least 3 contributing clinics, a complement rule so that no published number
minus another published number of the same query family reveals a group below 10, time ranges snapped to whole
weeks or months with totals that are sums of published cells only, and counts rounded to the nearest 5.
Region and city are the clinic's location (Organization.address), never the patient's.

**Unusual rises.** Weekly counts per region and diagnosis group, compared with the 8 weeks before the previous
week (an EARS C2 adaptation): flagged when the week's count is at least 10, at least 1.5 times the baseline mean
and above the mean plus 3 standard deviations, the deviation floored at the square root of the mean (Poisson)
and at 1. Computed nightly on the unrounded internal cells; shown only for a cell that is itself published. The
UI says "unusual rise", never "outbreak"; it is not a public-health notification.

## Consequences

- A new personal-data table (`condition`) and new aggregate tables; each has an owner, a retention rule and a
  DPIA section (plan section 6). Erasure and restriction of a patient's record also refresh the aggregates.
- No official ICD-10 titles anywhere. Doctors see the code with our label; a clinic that needs the national
  edition's titles needs a licence first (open legal question 7).
- With clinic-based geography and the 3-clinic rule, the platform view shows little until enough clinics share
  a region; that is the intended behaviour, not a bug. Patient residence would be better for epidemiology and is
  an open question for the owner (plan question O1), not built.
- The rise detection needs no history beyond 10 weeks; it does not model seasonality, so every winter's
  respiratory rise is flagged, which is what SAL-18 wants to see. A seasonal method (Farrington flexible) is
  possible once three years of data exist; not built.
- A clinic id exists in `trends` only inside the nightly build, which maps it to the clinic's region and city
  (from `organization`) and counts distinct clinics before it writes; the table, the query service and the API
  have no clinic column and no patient column, so "no query path returns clinic or patient identifiers" holds
  by construction and is also tested on the JSON.
