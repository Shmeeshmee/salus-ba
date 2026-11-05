import { describe, expect, it } from 'vitest'
import { addDays, dayIn, dayLabel, longDate, startOfDay, timeOf, weekOf } from './time'

const zone = 'Europe/Sarajevo'

describe('time in the clinic zone', () => {
  it('gives the calendar day of an instant in the zone, not in UTC', () => {
    expect(dayIn(new Date('2026-11-08T23:30:00Z'), zone)).toEqual({ year: 2026, month: 11, day: 9 })
    expect(dayIn(new Date('2026-11-08T23:30:00Z'), 'UTC')).toEqual({ year: 2026, month: 11, day: 8 })
  })

  it('starts the week on Monday', () => {
    expect(weekOf({ year: 2026, month: 11, day: 4 })).toEqual({ year: 2026, month: 11, day: 2 })
    expect(weekOf({ year: 2026, month: 11, day: 8 })).toEqual({ year: 2026, month: 11, day: 2 })
    expect(weekOf({ year: 2026, month: 11, day: 9 })).toEqual({ year: 2026, month: 11, day: 9 })
  })

  it('adds days across a month and a year', () => {
    expect(addDays({ year: 2026, month: 12, day: 28 }, 7)).toEqual({ year: 2027, month: 1, day: 4 })
    expect(addDays({ year: 2026, month: 11, day: 2 }, -7)).toEqual({ year: 2026, month: 10, day: 26 })
  })

  it('gives the instant a day starts in the zone, across both daylight saving changes', () => {
    expect(startOfDay({ year: 2026, month: 11, day: 9 }, zone).toISOString()).toBe('2026-11-08T23:00:00.000Z')
    expect(startOfDay({ year: 2027, month: 7, day: 1 }, zone).toISOString()).toBe('2027-06-30T22:00:00.000Z')
    expect(startOfDay({ year: 2027, month: 3, day: 29 }, zone).toISOString()).toBe('2027-03-28T22:00:00.000Z')
    expect(startOfDay({ year: 2027, month: 11, day: 1 }, zone).toISOString()).toBe('2027-10-31T23:00:00.000Z')
    expect(startOfDay({ year: 2026, month: 11, day: 9 }, 'America/New_York').toISOString()).toBe('2026-11-09T05:00:00.000Z')
  })

  it('formats times, days and dates for people', () => {
    expect(timeOf(new Date('2026-11-09T08:00:00Z'), zone)).toBe('09:00')
    expect(dayLabel({ year: 2026, month: 11, day: 9 })).toBe('Mon 9 Nov')
    expect(longDate(new Date('2026-11-08T23:30:00Z'), zone)).toBe('Monday 9 November 2026')
  })
})
