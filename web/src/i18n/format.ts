import type { Language } from './languages'

// Dates, times and numbers in the UI language (SPEC §3 "Formatting"): times 24-hour, digits Latin in every
// language. English is written the British way, as the app wrote it before, and Serbian is ijekavian as written
// in Bosnia and Herzegovina (CLDR's sr-*-BA), as the catalogs are.
const regions: Partial<Record<Language, string>> = { en: 'en-GB', 'sr-Latn': 'sr-Latn-BA', 'sr-Cyrl': 'sr-Cyrl-BA' }

export function intlLocale(language: Language): string {
  return `${regions[language] ?? language}-u-nu-latn`
}

const cache = new Map<string, Intl.DateTimeFormat>()

function dateTimeFormat(language: Language, zone: string, style: string, options: Intl.DateTimeFormatOptions) {
  const id = `${language}|${zone}|${style}`
  let format = cache.get(id)
  if (!format) {
    format = new Intl.DateTimeFormat(intlLocale(language), { ...options, timeZone: zone })
    cache.set(id, format)
  }
  return format
}

const styles = {
  time: { hour: '2-digit', minute: '2-digit', hourCycle: 'h23' },
  long: { weekday: 'long', day: 'numeric', month: 'long', year: 'numeric' },
  short: { weekday: 'short', day: 'numeric', month: 'short' },
} satisfies Record<string, Intl.DateTimeFormatOptions>

// formatTime is the time of day of instant in zone: "15:05".
export function formatTime(instant: Date, zone: string, language: Language): string {
  return dateTimeFormat(language, zone, 'time', styles.time).format(instant)
}

// formatDate is the day instant falls on in zone: long "Monday, 9 November 2026", short "Mon 9 Nov".
export function formatDate(instant: Date, zone: string, language: Language, style: 'long' | 'short'): string {
  return dateTimeFormat(language, zone, style, styles[style]).format(instant)
}

export function formatNumber(n: number, language: Language): string {
  return new Intl.NumberFormat(intlLocale(language)).format(n)
}
