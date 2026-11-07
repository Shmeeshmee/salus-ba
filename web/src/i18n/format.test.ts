import { describe, expect, it } from 'vitest'
import { formatDate, formatNumber, formatTime } from './format'

const afternoon = new Date('2026-11-09T14:05:00Z') // 15:05 in Sarajevo

describe('formatTime', () => {
  it.each(['bs', 'hr', 'sr-Latn', 'sr-Cyrl', 'en'] as const)('is 24-hour in %s', (language) => {
    expect(formatTime(afternoon, 'Europe/Sarajevo', language)).toBe('15:05')
  })

  it('pads the hour', () => {
    expect(formatTime(new Date('2026-11-09T08:30:00Z'), 'Europe/Sarajevo', 'en')).toBe('09:30')
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
  })

  it('writes Serbian ijekavian in both scripts', () => {
    expect(formatDate(afternoon, 'Europe/Sarajevo', 'sr-Latn', 'long')).toMatch(/ponedjeljak/i)
  })

  it('uses Latin digits', () => {
    expect(formatDate(afternoon, 'Europe/Sarajevo', 'sr-Cyrl', 'long')).toMatch(/9.*2026/)
  })
})

describe('formatNumber', () => {
  it('groups and separates decimals the way the language does', () => {
    expect(formatNumber(1234.5, 'en')).toBe('1,234.5')
    expect(formatNumber(1234.5, 'bs')).toBe('1.234,5')
  })
})
