# Translations

The status of every language's catalogs (docs/i18n/SPEC.md §3 "Catalogs", "Who translates"). Claude Code drafts
every language; a native speaker reviews it; a lawyer reviews every string tagged `legal` in each language
before production (SPEC §8).

| Tag | Language | Apps | Status | Reviewed by | Reviewed on | `legal` strings reviewed by a lawyer |
| --- | --- | --- | --- | --- | --- | --- |
| `bs` | Bosanski | web, mobile, e-mail | drafted | | | no |
| `hr` | Hrvatski | web, mobile, e-mail | drafted | | | no |
| `sr-Latn` | Srpski (latinica) | web, mobile, e-mail | drafted | | | no |
| `sr-Cyrl` | Српски (ћирилица) | web, mobile, e-mail | drafted | | | no |
| `en` | English | web, mobile, e-mail | source | | | no |
| `de` | Deutsch | mobile, e-mail | drafted | | | no |
| `ar` | العربية | mobile, e-mail | drafted | | | no |
| `tr` | Türkçe | mobile, e-mail | drafted | | | no |
| `it` | Italiano | mobile, e-mail | drafted | | | no |
| `fr` | Français | mobile, e-mail | drafted | | | no |

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
