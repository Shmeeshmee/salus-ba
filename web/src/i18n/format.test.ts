import { afterEach, describe, expect, it, vi } from 'vitest'
import { formatCalendarDate, formatDate, formatDateTime, formatMonth, formatNumber, formatTime } from './format'
import type { Language } from './languages'

const afternoon = new Date('2026-11-09T14:05:00Z') // 15:05 in Sarajevo
const tuesday = new Date('2026-11-10T14:05:00Z')

describe('formatTime', () => {
  it.each(['bs', 'hr', 'sr-Latn', 'sr-Cyrl', 'en'] as const)('is 24-hour in %s', (language) => {
    expect(formatTime(afternoon, 'Europe/Sarajevo', language)).toBe('15:05')
  })

  it('pads the hour', () => {
    expect(formatTime(new Date('2026-11-09T08:30:00Z'), 'Europe/Sarajevo', 'en')).toBe('09:30')
  })
})

// Every format the app shows, in every staff language. hr, sr-Latn, sr-Cyrl and en are what Intl wrote before
// SAL-15; bs is CLDR's Bosnian (as Firefox and Safari write it), except that a date with a time has no "u".
const formats: Record<Language, { long: string; short: string; calendar: string; dateTime: string }> = {
  bs: { long: 'utorak, 10. novembar 2026.', short: 'uto, 10. nov', calendar: '10. novembar 2026.', dateTime: '10. nov 2026. 15:05' },
  hr: { long: 'utorak, 10. studenoga 2026.', short: 'uto, 10. stu', calendar: '10. studenoga 2026.', dateTime: '10. stu 2026. 15:05' },
  'sr-Latn': { long: 'utorak, 10. novembar 2026.', short: 'uto 10. nov', calendar: '10. novembar 2026.', dateTime: '10. 11. 2026. 15:05' },
  'sr-Cyrl': { long: 'уторак, 10. новембар 2026.', short: 'уто 10. нов', calendar: '10. новембар 2026.', dateTime: '10. 11. 2026. 15:05' },
  en: { long: 'Tuesday, 10 November 2026', short: 'Tue 10 Nov', calendar: '10 November 2026', dateTime: '10 Nov 2026, 15:05' },
}

describe.each(Object.entries(formats) as [Language, (typeof formats)[Language]][])('in %s', (language, want) => {
  it('writes the long date', () => {
    expect(formatDate(tuesday, 'Europe/Sarajevo', language, 'long')).toBe(want.long)
  })

  it('writes the short date', () => {
    expect(formatDate(tuesday, 'Europe/Sarajevo', language, 'short')).toBe(want.short)
  })

  it('writes a calendar date', () => {
    expect(formatCalendarDate('2026-11-10', language)).toBe(want.calendar)
  })

  it('writes a date with its time', () => {
    expect(formatDateTime(tuesday, 'Europe/Sarajevo', language)).toBe(want.dateTime)
  })
})

describe('formatMonth', () => {
  it.each([
    ['bs', 'novembar 2026.'],
    ['hr', 'studeni 2026.'],
    ['sr-Latn', 'novembar 2026.'],
    ['sr-Cyrl', 'новембар 2026.'],
    ['en', 'November 2026'],
  ] as const)('names the month in %s', (language, want) => {
    expect(formatMonth('2026-11', language)).toBe(want)
  })
})

describe('formatDate', () => {
  it('writes the long date in the language', () => {
    expect(formatDate(afternoon, 'Europe/Sarajevo', 'en', 'long')).toBe('Monday, 9 November 2026')
    expect(formatDate(afternoon, 'Europe/Sarajevo', 'hr', 'long')).toMatch(/ponedjeljak/i)
    expect(formatDate(afternoon, 'Europe/Sarajevo', 'sr-Cyrl', 'long')).toMatch(/понедјељак/i)
  })

  it('writes the short date in the language', () => {
    expect(formatDate(afternoon, 'Europe/Sarajevo', 'en', 'short')).toBe('Mon 9 Nov')
  })

  it('uses the day in the zone', () => {
    expect(formatDate(new Date('2026-11-09T23:30:00Z'), 'Europe/Sarajevo', 'en', 'short')).toBe('Tue 10 Nov')
    expect(formatDate(new Date('2026-11-30T23:30:00Z'), 'Europe/Sarajevo', 'bs', 'short')).toBe('uto, 1. dec')
  })

  it('writes Serbian ijekavian in both scripts', () => {
    expect(formatDate(afternoon, 'Europe/Sarajevo', 'sr-Latn', 'long')).toMatch(/ponedjeljak/i)
  })

  it('uses Latin digits', () => {
    expect(formatDate(afternoon, 'Europe/Sarajevo', 'sr-Cyrl', 'long')).toMatch(/9.*2026/)
  })

  // Bosnian months are the Latin ones, neither Croatian (siječanj) nor Serbian (jun, jul, avgust).
  it('names every month in Bosnian', () => {
    const long = Array.from({ length: 12 }, (_, i) => formatCalendarDate(`2026-${String(i + 1).padStart(2, '0')}-15`, 'bs'))
    expect(long).toEqual(
      ['januar', 'februar', 'mart', 'april', 'maj', 'juni', 'juli', 'august', 'septembar', 'oktobar', 'novembar', 'decembar'].map((m) => `15. ${m} 2026.`),
    )
    const short = Array.from({ length: 12 }, (_, i) => formatDate(new Date(Date.UTC(2026, i, 15, 12)), 'UTC', 'bs', 'short').replace(/^\S+ 15\. /, ''))
    expect(short).toEqual(['jan', 'feb', 'mar', 'apr', 'maj', 'jun', 'jul', 'aug', 'sep', 'okt', 'nov', 'dec'])
  })

  // Bosnian weekdays are ijekavian: ponedjeljak, srijeda, nedjelja.
  it('names every weekday in Bosnian', () => {
    const days = Array.from({ length: 7 }, (_, i) => formatDate(new Date(Date.UTC(2026, 10, 9 + i, 12)), 'UTC', 'bs', 'long').split(',')[0])
    expect(days).toEqual(['ponedjeljak', 'utorak', 'srijeda', 'četvrtak', 'petak', 'subota', 'nedjelja'])
  })
})

describe('formatNumber', () => {
  it('groups and separates decimals the way the language does', () => {
    expect(formatNumber(1234.5, 'en')).toBe('1,234.5')
    expect(formatNumber(1234.5, 'bs')).toBe('1.234,5')
  })

  it('takes Intl options', () => {
    expect(formatNumber(1.25, 'bs', { minimumFractionDigits: 1, maximumFractionDigits: 1 })).toBe('1,3')
    expect(formatNumber(1234.5, 'en', { useGrouping: false })).toBe('1234.5')
  })
})

// Chromium ships no Bosnian locale data: Intl with bs falls back to the root locale and writes "2026 M11 10, Tue"
// and "1,234.5" (SAL-15). Node has Bosnian, so the tests above would pass without the fix; here Intl is given
// Chromium's gap, and the formatter, loaded afresh so that nothing is cached, must still write Bosnian.
describe('in a browser without Bosnian locale data', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
    vi.resetModules()
  })

  it('still writes Bosnian dates and numbers', async () => {
    vi.stubGlobal('Intl', withoutBosnian())
    vi.resetModules()
    const format = await import('./format')
    expect(new Intl.DateTimeFormat('bs', { month: 'long', timeZone: 'UTC' }).format(tuesday)).toBe('November') // the gap is there
    expect(format.formatDate(tuesday, 'Europe/Sarajevo', 'bs', 'long')).toBe(formats.bs.long)
    expect(format.formatDate(tuesday, 'Europe/Sarajevo', 'bs', 'short')).toBe(formats.bs.short)
    expect(format.formatCalendarDate('2026-11-10', 'bs')).toBe(formats.bs.calendar)
    expect(format.formatDateTime(tuesday, 'Europe/Sarajevo', 'bs')).toBe(formats.bs.dateTime)
    expect(format.formatTime(tuesday, 'Europe/Sarajevo', 'bs')).toBe('15:05')
    expect(format.formatNumber(1234.5, 'bs')).toBe('1.234,5')
  })
})

// withoutBosnian is Intl as Chromium has it: a bs locale is formatted with data that is not Bosnian. Chromium's is
// the root locale's ("M11"), which Node cannot give, so US English ("November", "1,234.5") stands in for it.
function withoutBosnian(): typeof Intl {
  const root = (locales: unknown) => (String(locales).toLowerCase().startsWith('bs') ? 'en-US-u-nu-latn' : locales)
  const fallback = <T extends object>(Real: T): T =>
    new Proxy(Real, { construct: (target, [locales, options]) => Reflect.construct(target as never, [root(locales), options]) as object })
  const intl = Object.create(Intl) as typeof Intl
  Object.defineProperty(intl, 'DateTimeFormat', { value: fallback(Intl.DateTimeFormat) })
  Object.defineProperty(intl, 'NumberFormat', { value: fallback(Intl.NumberFormat) })
  return intl
}
