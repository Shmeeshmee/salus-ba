# Terminology sources and licences

Where the terms in `assets/terminology/seed.jsonl` come from, under which terms, and what is deliberately not
used (SPEC.md §3 "Data", "What is not used"). Checked by package 0.1 on 2026-11-07 against the pages linked
below. Not legal advice: SPEC.md §7 has a lawyer check this file, with the other licences, before production.

## Used

| Source | What we take | Licence | Checked |
| --- | --- | --- | --- |
| [Wikidata](https://www.wikidata.org/) | Items, their labels and aliases in `la` and the ten app languages, the identifier properties below, and the number of sitelinks (for the rank) | Structured data (main, Property, Lexeme and EntitySchema namespaces) is [CC0 1.0](https://creativecommons.org/publicdomain/zero/1.0/): "All structured data … is released into the public domain" ([Wikidata:Licensing](https://www.wikidata.org/wiki/Wikidata:Licensing)). CC0 waives every right, commercial use included. Text in other namespaces (talk and project pages) is CC BY-SA 4.0: we take none | As the SPEC says |
| Wikidata identifier properties | Terminologia Anatomica 98 ID ([P1323](https://www.wikidata.org/wiki/Property:P1323)), ICD-10 ([P494](https://www.wikidata.org/wiki/Property:P494)), ICD-10-PCS ([P1690](https://www.wikidata.org/wiki/Property:P1690)), ICD-9-CM procedure ([P1692](https://www.wikidata.org/wiki/Property:P1692)), ATC ([P267](https://www.wikidata.org/wiki/Property:P267)) | Values of Wikidata statements, so CC0 as above. A code is used as an identifier of the concept (a fact), never with the classification's own titles or text | See the notes on ICD and ATC below |

The fetch date and the exact SPARQL queries are recorded here by package 1.2 when it writes the seed.

## Not used

| Source | Why not | Checked |
| --- | --- | --- |
| SNOMED CT | Use needs a licence: free in SNOMED International member countries, an affiliate licence with possible fees elsewhere, and web applications need "special permission and fees" ([Get SNOMED CT](https://www.snomed.org/get-snomed)). Bosnia and Herzegovina is not a member (the [member list](https://www.snomed.org/members), 54 members on 2026-11-07, has Croatia and Slovenia, not BiH or Serbia) | As the SPEC says |
| WHO ICD-10 and ICD-11 titles and descriptions | WHO's classifications are under the [WHO Classifications License, CC BY-ND 3.0 IGO](https://www.who.int/about/policies/publishing/copyright): commercial use is allowed "provided there is no adaptation", translations and mappings need a written agreement with WHO ([ICD-11 licence](https://icd.who.int/en/docs/icd11-license.pdf)). Our labels are Wikidata's and the partner's, in languages WHO has not licensed to us, and sit beside codes as a crosswalk: that is an adaptation and a translation, so we take no WHO text. The ICD-10 code itself is taken only as an identifier from Wikidata | As the SPEC says (WHO's text is protected; the code as an identifier is a fact). One nuance: the classification licence is CC BY-ND, not "all rights reserved", which does not change what we take |
| WHOCC ATC/DDD index | The index states "Copying and distribution for commercial purposes is not allowed" and that the material may not be changed ([WHOCC copyright](https://atcddd.fhi.no/copyright_disclaimer/)). We take nothing from the index: the ATC code comes from Wikidata (CC0) as an identifier of a drug item, with Wikidata's labels | Not in the SPEC's list; recorded in PROGRESS.md (0.1) as a point for the lawyer before production |
| Machine translation | SPEC.md §3 "Gaps stay gaps": a missing label stays empty | n/a |

## For the lawyer (SPEC.md §7)

- Whether showing ICD-10 and ATC codes taken from Wikidata (CC0) next to our own labels needs anything from
  WHO or the WHOCC, given their terms above.
- Whether a list of about 5,300 Wikidata items selected by these identifiers is free of any database right of
  the classifications' owners (the selection is ours; the values are CC0).
