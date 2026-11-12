# 12. ICD-10 codes as identifiers, our own diagnosis groups, pre-computed aggregates, suppressed platform figures

Status: accepted, 2026-11-12 (docs/plans/analytics.md, epic SAL-18). The platform view and its builds are behind
`ANALYTICS_PLATFORM_ENABLED`, off by default, until the owner confirms the answers to the plan's legal questions
L1 to L8; a clinic may opt out of the platform figures and then loses the platform view.

## Context

SAL-18 asks for two analytics views: a clinic's own statistics (SAL-20) and platform-wide trends by region with
no clinic or patient identifiable (SAL-21), both drawn from coded diagnoses on signed documents and from
appointments, never from transcripts, and grouped into diagnosis groups based on ICD-10 blocks (SAL-19).

What exists: a document template can have `coded` fields holding one ICD-10 code
(`internal/composition/model.go`, system `http://hl7.org/fhir/sid/icd-10`); signing refuses a malformed code
(`service_draft.go`); signed documents are `document_draft` rows with status `final`, which never change again
(entered-in-error is only the discard of a preliminary draft). The terminology list (`internal/terminology`) has
Wikidata items with their ICD-10 code (P494, CC0) and a review panel. There is no FHIR `Condition`, no clinic
address, no aggregate, and no erasure or restriction feature yet. Every clinic table is row-level secured by
`app.organization_id`. Nothing is deployed (ADR 0011), so there is no production data to backfill.

Three things need a decision that outlasts the feature: where the codes and groups come from (a licence
question), how the figures are computed (performance and correctness), and what makes the platform figures
anonymous (GDPR, CLAUDE.md top rule 4).

## Options for the ICD-10 source

| Source | Licence | Usable for us |
| --- | --- | --- |
| WHO ICD-10 (2019 edition), codes, titles, block structure | WHO classifications are licensed [CC BY-ND 3.0 IGO](https://www.who.int/about/policies/publishing/copyright): commercial use allowed with attribution and **no adaptation**; translations and mappings need WHO's written agreement ([ICD-11 licence](https://icd.who.int/en/docs/icd11-license.pdf), referred to by the classifications page). Older ICD-10 licences (e.g. the [UK TRUD terms](https://isd.hscic.gov.uk/trud/users/guest/filters/0/licence/22)) were stricter and WHO reserved the right to license ICD-10 for a fee | The codes as identifiers, yes. WHO's titles, no: we would show them in nine languages WHO has not licensed to us and next to our own group names, which is an adaptation and a translation |
| Croatian MKB-10 (HZJZ) | A translation made under agreement with WHO; no open licence published (none found on hzjz.hr) | No, without a written licence from HZJZ (and WHO) |
| BiH and Serbian MKB-10 editions (entity public-health institutes; the Serbian edition is © Institut za javno zdravlje Srbije "Batut", 2013) | All rights reserved, made from WHO's original; the Slovenian analogue shows WHO assigning the translation's copyright to the national institute | No, without a written licence |
| US ICD-10-CM (NCHS) | Free to use as published by the US government, but a clinical modification made under WHO's authorisation, with codes and titles that differ from WHO ICD-10 beyond three characters | No: not the classification doctors in BiH use, and the rights in the underlying WHO content are not clearly free outside the US |
| Wikidata P494 (ICD-10 code of an item) | [CC0 1.0](https://www.wikidata.org/wiki/Wikidata:Licensing) | Yes; already our terminology source (docs/terminology/SOURCES.md) |

## Decision

**ICD-10 source.** We keep the terminology's settled decision (docs/terminology/SPEC.md, "Data" and §7;
SOURCES.md) and extend it: an ICD-10 code is used only as an identifier of a diagnosis (a fact, not WHO's
expression). Codes come from the doctor (a `coded` field) and from the Wikidata items of the terminology list
(CC0). We take no WHO text: no titles, descriptions, inclusion or exclusion notes, in any language. A code on
screen is shown with the terminology's own label when there is one, and alone otherwise. A code is checked by
format (as today) and falls into a diagnosis group or into "Other"; it is not checked against a WHO code list
we do not hold. ICD-10 is shown as "MKB-10" in Bosnian, Croatian and Serbian. Why: it is the only option legally
usable without a licence we do not have, it needs no new data import, and it changes nothing already decided.
The cost: no official titles in the UI, and a well-formed but non-existent code is accepted. A licence from WHO
or a national institute can be added later without a data-model change.

**Diagnosis groups.** Our own grouping, 27 groups and 7 subgroups, each defined by ranges of ICD-10 codes on
block boundaries (e.g. "Respiratory infections" J00–J22, subgroup "Common cold" J00). The ranges are code
intervals, which we treat as facts; the names are our own plain-language wording in all ten languages, reviewed
in the term review panel, never WHO's block titles or a translation of them. A code belongs to exactly one group
and at most one subgroup (tested). Reference data owned by `terminology`, seeded by a migration (plan appendix A).

**Source of the figures.** A FHIR `Condition` (new domain `condition`, context `clinical`), one per filled
coded field of a signed document, written in the signing transaction with `verificationStatus` `confirmed`: the
sign request carries a per-field confirmation, and signing refuses a coded value without it. Conditions
backfilled from documents signed before this feature are `unconfirmed` and not counted. Appointments come from
the `appointment` service. Transcripts, suggestions and free text are never read.

**Pre-computed aggregates for a clinic** (domain `clinicstats`, new context `analytics`). Counts per clinic,
local day and hour, practitioner role, specialty, appointment status, and diagnosis group, subgroup and code.
Incremental: signing and every appointment write enqueue, in their transaction, a River job with opaque ids
(`clinicstats_refresh{organization_id, day}`) that recomputes that clinic-day idempotently; the producers
declare the enqueue port, so they never import `clinicstats`. Nightly: the last 35 days of every clinic are
recomputed, which repairs a lost job. Charts read only aggregates.

**Reading across clinics.** The build jobs log in as a new role `salus_stats`, as the retention job does with
`salus_retention` (migration 00014): `BYPASSRLS`, owner of nothing, its own credentials loaded only by
`cmd/worker`, `SELECT` granted per column on the source columns only (codes, times, statuses, clinic,
practitioner role, and the patient id of `condition` and `appointment`), never a name, contact detail or field
value. The source queries stay in the owning domains and are reached through their services. A patient id
never leaves the `condition` and `appointment` packages: they return a pseudonym, HMAC-SHA256 of the id under a
key generated for one build and discarded with it, so the build can count distinct patients and nothing stored
can be linked back.

**Platform figures** (domain `trends`, context `analytics`). Built from the `condition` and `appointment`
services (not from `clinicstats`, so the two can be built in parallel). The published unit is a **closed
calendar month**, computed once, seven days after the month ends, and then **frozen**. For each month, one fixed
set of tables, each with at most two non-time dimensions (geography × diagnosis, geography × specialty,
specialty × diagnosis), every dimension hierarchical (all → entity → region → city; group → subgroup → code;
specialty), counting distinct patients. Suppression is decided once for the whole set, not per query: primary
suppression (fewer than `k` = 10 patients, fewer than 3 clinics, or a p% dominance rule), then secondary
suppression over every parent-children relation of every table until no suppressed cell can be recovered from
the published ones (tested with an exact interval check). The API returns only frozen published cells; a
total over months is the sum of published month cells, marked partial when any is suppressed. Published counts
are rounded to the nearest 5. Region and city are the clinic's location, never the patient's.

**Unusual rises.** Weekly distinct-patient counts per region and diagnosis group, closed weeks only, compared
with the 8 weeks before the previous week (an EARS C2 adaptation, sample standard deviation): flagged when the
week has at least `k` patients from at least 3 clinics, the 8-week baseline has at least `k` patients in total,
the count is at least 1.5 times the baseline mean and above the mean plus 3 standard deviations (floored at the
square root of the mean and at 1). A flag carries no count: region, group and week only. Computed once per
closed week and frozen. The UI says "unusual rise", never "outbreak"; it is not a public-health notification.

## Consequences

- A new personal-data table (`condition`, kept with the clinical record) and aggregate tables; each has an owner,
  a retention rule and a DPIA section. Erasure and restriction do not exist yet; when the `datarequest` feature
  is built it must refresh the clinic aggregates and leave restricted records out of new builds (frozen platform
  months are anonymous output and stay, subject to legal question L3).
- No official ICD-10 titles anywhere. A clinic that needs the national edition's titles needs a licence first.
- The platform view is coarse: months, not weeks; two dimensions at a time; and with clinic-based geography and
  the clinic rules it shows little until enough clinics share a region. That is intended.
- The rise detection flags sudden rises against the previous weeks; it does not model seasonality, so a winter
  wave is flagged when it rises sharply and not when it builds slowly. A seasonal method (Farrington flexible)
  is possible once three years of data exist; not built.
- A clinic id exists in `trends` only inside a build, which maps it to region and city and evaluates the clinic
  rules; no table, service type or response of `trends` has a clinic or patient column, so "no query path
  returns clinic or patient identifiers" holds by construction and is also tested on the JSON.
