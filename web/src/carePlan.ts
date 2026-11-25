import type { ErrorCode } from './api/client'
import type { components } from './api/schema'
import type { useFormat, useT } from './i18n/useT'
import { addDays, dayIn, startOfDay } from './time'

// The treatment plan's types and the helpers its list, dialog and booking share (clinic-cms U6, package TP2).
export type CarePlan = components['schemas']['CarePlan']
export type CarePlanStep = components['schemas']['CarePlanStep']
type Service = components['schemas']['Service']
type T = ReturnType<typeof useT>

// stepText is a step as one line: its number, service and visit type, and its day: once the plan is booked, its
// visit's date (a moved visit's new day), the day from the start in a draft. A service gone from the price list keeps
// the name the server gives it.
export function stepText(t: T, f: ReturnType<typeof useFormat>, s: CarePlanStep, i: number, services: Service[], startsOn: string | undefined, zone: string): string {
  const service = services.find((x) => x.id === s.serviceId)?.name ?? s.serviceName ?? t('carePlans.serviceGone')
  const mode = t(s.visitMode === 'video' ? 'carePlans.mode.video' : 'carePlans.mode.inPerson')
  const values = { n: i + 1, service, mode }
  if (!startsOn) return t('carePlans.stepDay', { ...values, day: s.dayOffset })
  const day = s.visitStart ? new Date(s.visitStart) : dayStart(startsOn, s.dayOffset, zone)
  return t('carePlans.stepDate', { ...values, date: f.date(day, zone, 'long') })
}

// dayStart is the instant the day offset days after the ISO date day begins in the clinic's zone.
export function dayStart(day: string, offset: number, zone: string): Date {
  const [year, month, d] = day.split('-').map(Number)
  return startOfDay(addDays({ year, month, day: d }, offset), zone)
}

// isoDay is the instant's date in the zone, as YYYY-MM-DD.
export function isoDay(instant: Date, zone: string): string {
  const d = dayIn(instant, zone)
  return `${d.year}-${String(d.month).padStart(2, '0')}-${String(d.day).padStart(2, '0')}`
}

// planMessage says why the server refused a plan's write, booking or cancellation.
export function planMessage(t: T, code?: ErrorCode): string {
  if (code === 'invalid_care_plan' || code === 'invalid_request') return t('carePlans.invalid')
  if (code === 'care_plan_not_draft') return t('carePlans.notDraft')
  if (code === 'care_plan_ended') return t('carePlans.ended')
  if (code === 'version_conflict') return t('carePlans.changed')
  if (code === 'slot_unavailable') return t('carePlans.slotTaken')
  if (code === 'resource_busy') return t('carePlans.deviceBusy')
  if (code === 'forbidden' || code === 'not_found') return t('carePlans.refused')
  return t('common.somethingWrong')
}
