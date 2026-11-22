import type { components } from './api/schema'
import { formatNumber } from './i18n/format'
import type { Language } from './i18n/languages'
import { addDays, dayIn, startOfDay, type Day } from './time'

export type Receipt = components['schemas']['Receipt']
export type PaymentMethod = components['schemas']['PaymentMethod']

export const paymentMethods: PaymentMethod[] = ['cash', 'card', 'transfer']

// money shows an amount of the currency's minor unit in the UI language: 10670 BAM is 106,70 KM.
export function money(minor: number, currency: string, language: Language): string {
  return formatNumber(minor / 100, language, { style: 'currency', currency })
}

// A period is two calendar days of the clinic, both included, as a date input holds them (YYYY-MM-DD).
export type Period = { from: string; to: string }

function iso(day: Day): string {
  return `${day.year}-${String(day.month).padStart(2, '0')}-${String(day.day).padStart(2, '0')}`
}

function parse(date: string): Day | undefined {
  const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(date)
  return m ? { year: Number(m[1]), month: Number(m[2]), day: Number(m[3]) } : undefined
}

// thisMonth is the clinic's current month up to today.
export function thisMonth(now: Date, zone: string): Period {
  const today = dayIn(now, zone)
  return { from: iso({ ...today, day: 1 }), to: iso(today) }
}

// instants is the period as the API takes it: from the first day's start to the start of the day after the last,
// in the clinic's zone; undefined while a date is missing or the period ends before it begins.
export function instants(p: Period, zone: string): { from: string; to: string } | undefined {
  const from = parse(p.from)
  const to = parse(p.to)
  if (!from || !to || p.to < p.from) return undefined
  return { from: startOfDay(from, zone).toISOString(), to: startOfDay(addDays(to, 1), zone).toISOString() }
}

// receiptPdfURL is where a receipt's PDF is read, on the same origin as every other call, so the session cookie goes
// with it.
export function receiptPdfURL(id: string): string {
  return `${globalThis.location.origin}/clinic/receipts/${encodeURIComponent(id)}/pdf`
}

export function reportCsvURL(from: string, to: string): string {
  const query = new URLSearchParams({ from, to, format: 'csv' })
  return `${globalThis.location.origin}/clinic/reports/receipts?${query.toString()}`
}
