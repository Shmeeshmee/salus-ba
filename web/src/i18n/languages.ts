// The staff languages (docs/i18n/SPEC.md §3 "Languages"): the web app offers these five, as BCP 47 tags.
export const staffLanguages = ['bs', 'hr', 'sr-Latn', 'sr-Cyrl', 'en'] as const

export type Language = (typeof staffLanguages)[number]

export function isStaffLanguage(value: unknown): value is Language {
  return (staffLanguages as readonly unknown[]).includes(value)
}

// The rules of SPEC §3 "Matching", narrowed to the staff languages. A tag matches a prefix when it is the
// prefix or the prefix followed by a subtag ("hr" and "hr-HR", not "hrx").
const rules: readonly [readonly string[], Language][] = [
  [['bs'], 'bs'],
  [['hr'], 'hr'],
  [['sr-latn', 'sr-me', 'sh', 'cnr'], 'sr-Latn'],
  [['sr'], 'sr-Cyrl'], // sr, sr-Cyrl, sr-RS, sr-BA: CLDR's default script for Serbian is Cyrillic
  [['en'], 'en'],
]

function hasPrefix(tag: string, prefix: string): boolean {
  return tag === prefix || tag.startsWith(`${prefix}-`)
}

// matchLanguage is the first of the browser's languages, in its order, that the app offers; Bosnian when none is.
export function matchLanguage(browser: readonly string[]): Language {
  for (const tag of browser) {
    const lower = tag.toLowerCase()
    for (const [prefixes, language] of rules) if (prefixes.some((p) => hasPrefix(lower, p))) return language
  }
  return 'bs'
}
