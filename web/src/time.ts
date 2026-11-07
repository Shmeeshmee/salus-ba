// Calendar days and times in the clinic's time zone (SPEC: stored in UTC, shown in the clinic's zone). A Day
// is a date on the clinic's calendar; instants are Dates.

export type Day = { year: number; month: number; day: number }

const partsCache = new Map<string, Intl.DateTimeFormat>()

function parts(instant: Date, zone: string): Record<string, number> {
  let format = partsCache.get(zone)
  if (!format) {
    format = new Intl.DateTimeFormat('en-GB', {
      timeZone: zone,
      hourCycle: 'h23',
      year: 'numeric',
      month: 'numeric',
      day: 'numeric',
      hour: 'numeric',
      minute: 'numeric',
      second: 'numeric',
    })
    partsCache.set(zone, format)
  }
  const out: Record<string, number> = {}
  for (const p of format.formatToParts(instant)) if (p.type !== 'literal') out[p.type] = Number(p.value)
  return out
}

// dayIn is the day instant falls on in zone.
export function dayIn(instant: Date, zone: string): Day {
  const p = parts(instant, zone)
  return { year: p.year, month: p.month, day: p.day }
}

function utc(day: Day): Date {
  return new Date(Date.UTC(day.year, day.month - 1, day.day))
}

function fromUTC(d: Date): Day {
  return { year: d.getUTCFullYear(), month: d.getUTCMonth() + 1, day: d.getUTCDate() }
}

export function addDays(day: Day, n: number): Day {
  const d = utc(day)
  d.setUTCDate(d.getUTCDate() + n)
  return fromUTC(d)
}

// weekOf is the Monday of day's week.
export function weekOf(day: Day): Day {
  return addDays(day, -((utc(day).getUTCDay() + 6) % 7))
}

export function sameDay(a: Day, b: Day): boolean {
  return a.year === b.year && a.month === b.month && a.day === b.day
}

// offset is how far zone's wall clock is ahead of UTC at instant, in milliseconds.
function offset(instant: number, zone: string): number {
  const p = parts(new Date(instant), zone)
  return Date.UTC(p.year, p.month - 1, p.day, p.hour, p.minute, p.second) - Math.floor(instant / 1000) * 1000
}

// startOfDay is the instant day begins in zone: local midnight, found by correcting a UTC guess with the
// zone's offset twice (once more when the offset changes in between, as on a daylight saving day).
export function startOfDay(day: Day, zone: string): Date {
  const wall = utc(day).getTime()
  const first = wall - offset(wall, zone)
  return new Date(wall - offset(first, zone))
}

export function timeOf(instant: Date, zone: string): string {
  return new Intl.DateTimeFormat('en-GB', { timeZone: zone, hour: '2-digit', minute: '2-digit', hourCycle: 'h23' }).format(instant)
}

// longDate is "Monday 9 November 2026" in zone.
export function longDate(instant: Date, zone: string): string {
  const p = new Intl.DateTimeFormat('en-GB', { timeZone: zone, weekday: 'long', day: 'numeric', month: 'long', year: 'numeric' }).formatToParts(instant)
  const get = (type: string) => p.find((x) => x.type === type)?.value ?? ''
  return `${get('weekday')} ${get('day')} ${get('month')} ${get('year')}`
}
