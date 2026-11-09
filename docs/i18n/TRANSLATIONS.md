# Translations

The status of every language's catalogs (docs/i18n/SPEC.md §3 "Catalogs", "Who translates"). Claude Code drafts
every language; a native speaker reviews it; a lawyer reviews every string tagged `legal` in each language
before production (SPEC §8).

| Tag | Language | Apps | Status | `legal` strings (web + mobile + e-mail) | Reviewed by | Reviewed on | `legal` strings reviewed by a lawyer |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `bs` | Bosanski | web, mobile, e-mail | drafted | 37 (7 + 30 + 0) | | | no |
| `hr` | Hrvatski | web, mobile, e-mail | drafted | 37 (7 + 30 + 0) | | | no |
| `sr-Latn` | Srpski (latinica) | web, mobile, e-mail | drafted | 37 (7 + 30 + 0) | | | no |
| `sr-Cyrl` | Српски (ћирилица) | web, mobile, e-mail | drafted | 37 (7 + 30 + 0) | | | no |
| `en` | English | web, mobile, e-mail | source | 37 (7 + 30 + 0) | | | no |
| `de` | Deutsch | mobile, e-mail | drafted | 30 (– + 30 + 0) | | | no |
| `ar` | العربية | mobile, e-mail | drafted | 30 (– + 30 + 0) | | | no |
| `tr` | Türkçe | mobile, e-mail | drafted | 30 (– + 30 + 0) | | | no |
| `it` | Italiano | mobile, e-mail | drafted | 30 (– + 30 + 0) | | | no |
| `fr` | Français | mobile, e-mail | drafted | 30 (– + 30 + 0) | | | no |

Every language but the English source is `drafted` (2026-11-08, package 3.1); none is reviewed yet. The `legal` strings were counted with a
script on 2026-11-08 (docs/i18n/PROGRESS.md, 3.1): web, the catalog entries of `web/src/i18n/*/<tag>.json` whose
`description` is `legal` (the web app has only the five staff languages, so "–" for the others); mobile, the keys
whose description in `app_en.arb` starts with `[legal]`, which hold for every language (ARB metadata lives in the
template only); e-mail, none, since the templates carry no metadata (package 1.3). The one e-mail sentence with legal
weight, the cancellation window in `booked.txt` ("You can cancel it up to 24 hours before it starts."), goes to the
lawyer with the tagged strings in each of the ten languages.

The web count went from 6 to 7 on 2026-11-09 (SAL-9): `consultation.live.suggestionsNote`, the line saying a
suggestion is not a diagnosis (`docs/consult-intelligence/COPY.md` W9), is tagged `legal` for the lawyer's review
(agent decision, for the owner to review), and the 41 other strings of the call page's live panels were drafted in
bs, hr, sr-Latn and sr-Cyrl like every string before them; each language stays `drafted`.
The wiring follow-up to #34 (SAL-9, 2026-11-09) added one string, `consultation.live.requiredEmpty` (COPY.md W24,
approved by the owner on 2026-11-09), and changed `consultation.live.suggestionsNote` (W9, `legal`) to the owner's
wording naming AI ("AI suggestions are prompts..."); both drafted in the same four languages, for a native
speaker's check (W9 also for the lawyer's). The count of `legal` strings is unchanged.

English is the source: its text is the features' COPY.md files word for word, approved by the owner.
`mobile/lib/l10n/app_sr.arb` is a copy of `app_sr_Cyrl.arb` that Flutter needs as the base of the two Serbian
scripts; it is not a language of its own.

## Statuses

- `drafted`: written by Claude Code; not yet read by a native speaker.
- `reviewed`: a native speaker read every string of the language in the running apps and the e-mails, and
  corrected what reads wrongly. Record who and when in the table; a later change to a string of that
  language sets it back to `drafted` until it is reviewed again.
- `source`: the English source.

## Review rules

- Natural, plain and professional, as a native speaker writes for a clinic; never word for word.
- Serbian is ijekavian in both scripts, and the Cyrillic catalog is written in Cyrillic, not transliterated.
  Bosnian and Croatian follow their own standards (Bosnian "historija", "sedmica"; Croatian "povijest",
  "tjedan").
- Medical terms as a doctor in Bosnia and Herzegovina writes them in that language.
- Plurals and placeholders are ICU MessageFormat; every plural has the forms its language needs (three in
  Bosnian, Croatian and Serbian, six in Arabic). `make i18n-check` checks the shape, not the wording.
- Strings tagged `legal` (consent, privacy notice link text, permission prompts, cancellation and revocation
  effects) need a lawyer's review in each language before production, besides the native speaker's.
- A reviewer changes text in the catalogs only; the English source changes through its COPY.md.
