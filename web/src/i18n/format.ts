import { dayIn } from '../time'
import type { Language } from './languages'

// Dates, times and numbers in the UI language (SPEC §3 "Formatting"): times 24-hour, digits Latin in every
// language. English is written the British way, as the app wrote it before, and Serbian is ijekavian as written
// in Bosnia and Herzegovina (CLDR's sr-*-BA), as the catalogs are. Every date, time and number the app shows goes
// through this file.
const regions: Partial<Record<Language, string>> = { en: 'en-GB', 'sr-Latn': 'sr-Latn-BA', 'sr-Cyrl': 'sr-Cyrl-BA' }

export function intlLocale(language: Language): string {
  return `${regions[language] ?? language}-u-nu-latn`
}

// Chromium ships no Bosnian locale data, so Intl with bs writes "2026 M11 10, Tue" and "1,234.5" there (SAL-15).
// Bosnian is formatted with Croatian data, which every browser has: its date patterns, weekday names (ijekavian)
// and number signs are Bosnian's too. Only the month names differ, and they are replaced with Bosnian ones.
function dataLocale(language: Language): string {
  return language === 'bs' ? intlLocale('hr') : intlLocale(language)
}

const bosnianMonths = {
  long: ['januar', 'februar', 'mart', 'april', 'maj', 'juni', 'juli', 'august', 'septembar', 'oktobar', 'novembar', 'decembar'],
  short: ['jan', 'feb', 'mar', 'apr', 'maj', 'jun', 'jul', 'aug', 'sep', 'okt', 'nov', 'dec'],
}

const styles = {
  time: { hour: '2-digit', minute: '2-digit', hourCycle: 'h23' },
  long: { weekday: 'long', day: 'numeric', month: 'long', year: 'numeric' },
  short: { weekday: 'short', day: 'numeric', month: 'short' },
  calendar: { day: 'numeric', month: 'long', year: 'numeric' },
  month: { month: 'long', year: 'numeric' },
  dateTime: { dateStyle: 'medium', timeStyle: 'short', hourCycle: 'h23' },
} satisfies Record<string, Intl.DateTimeFormatOptions>

type Style = keyof typeof styles

const monthsOf: Record<Style, keyof typeof bosnianMonths | null> = { time: null, long: 'long', short: 'short', calendar: 'long', dateTime: 'short', month: 'long' }

const cache = new Map<string, Intl.DateTimeFormat>()

function dateTimeFormat(language: Language, zone: string, style: Style) {
  const id = `${language}|${zone}|${style}`
  let format = cache.get(id)
  if (!format) {
    format = new Intl.DateTimeFormat(dataLocale(language), { ...styles[style], timeZone: zone })
    cache.set(id, format)
  }
  return format
}

function format(instant: Date, zone: string, language: Language, style: Style): string {
  const f = dateTimeFormat(language, zone, style)
  const months = monthsOf[style]
  if (language !== 'bs' || !months) return f.format(instant)
  const name = bosnianMonths[months][dayIn(instant, zone).month - 1]
  return f
    .formatToParts(instant)
    .map((p) => (p.type === 'month' ? name : p.value))
    .join('')
}

// formatTime is the time of day of instant in zone: "15:05".
export function formatTime(instant: Date, zone: string, language: Language): string {
  return format(instant, zone, language, 'time')
}

// formatDate is the day instant falls on in zone: long "Monday, 9 November 2026", short "Mon 9 Nov".
export function formatDate(instant: Date, zone: string, language: Language, style: 'long' | 'short'): string {
  return format(instant, zone, language, style)
}

// formatCalendarDate is a date with no time zone, as FHIR writes a birth date or a date answer ("2026-11-09"):
// "9 November 2026".
export function formatCalendarDate(date: string, language: Language): string {
  return format(new Date(`${date}T00:00:00Z`), 'UTC', language, 'calendar')
}

// formatMonth is a calendar month written YYYY-MM, as the usage figures give it: "November 2026".
export function formatMonth(month: string, language: Language): string {
  return format(new Date(`${month}-01T00:00:00Z`), 'UTC', language, 'month')
}

// formatDateTime is instant in zone with its time: "9 Nov 2026, 15:05".
export function formatDateTime(instant: Date, zone: string, language: Language): string {
  return format(instant, zone, language, 'dateTime')
}

export function formatNumber(n: number, language: Language, options?: Intl.NumberFormatOptions): string {
  return new Intl.NumberFormat(dataLocale(language), options).format(n)
}
