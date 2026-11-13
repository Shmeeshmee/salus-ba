import type { Account, ApiClient } from '../api/client'
import type { FhirClient } from '../api/fhir'
import { attempt } from '../api/result'
import type { components } from '../api/schema'
import type { Language } from '../i18n/languages'
import { loadStaff, loadTimeZone, type Member } from '../staff'

// data.ts is the analytics screens' one way to their data (docs/plans/analytics.md §8, SAL-22): the generated client
// of api/openapi.yaml (SAL-20's /analytics/clinic, SAL-21's /analytics/platform) behind one interface, which
// fixtures.ts implements for the tests. The screens speak in months; the clinic's endpoints take days.

export type Month = string // "2026-11"
export type Count = number | null // null: hidden (suppressed), on the platform only

type Schemas = components['schemas']
export type ClinicAppointments = Schemas['ClinicAppointments']
export type AppointmentMonth = ClinicAppointments['series'][number]
export type HeatCell = { weekday: number; hour: number; count: Count } // ISO weekday, 1 = Monday; local hour 0-23
export type Row = { key: string; count: Count }

export type ClinicQuery = { from: Month; to: Month; specialty?: string; practitionerRole?: string }
export type ClinicDiagnosisQuery = ClinicQuery & { group?: string; subgroup?: string; by: 'group' | 'subgroup' | 'code' | 'month' }

// The platform's tables (plan §5): one at a time, each two dimensions with their levels.
export type PlatformTable = 'gd' | 'gs' | 'sd'
export type Dimension = 'entity' | 'region' | 'city' | 'group' | 'subgroup' | 'code' | 'specialty'
export const dimensionsOf: Record<PlatformTable, Dimension[]> = {
  gd: ['entity', 'region', 'city', 'group', 'subgroup', 'code'],
  gs: ['entity', 'region', 'city', 'specialty'],
  sd: ['specialty', 'group', 'subgroup', 'code'],
}
export type PlatformCellsQuery = { table: PlatformTable; from: Month; to: Month; filter: Partial<Record<Dimension, string>>; by: Dimension | 'month' }
// partial: a count of the answer is hidden, so a sum over it is a lower bound.
export type PlatformCells = Schemas['PlatformBreakdown']
export type PlatformMonths = Schemas['PlatformMonths'] // months and quarters oldest first
export type Rise = Schemas['PlatformRises']['rises'][number] // week "2026-W44"

export type DiagnosisGroup = Schemas['DiagnosisGroup']
export type Place = Schemas['Place']
export type Specialty = Schemas['Specialty']
export type { Member }

// AnalyticsError is a refused call, with the API's code (analytics_not_contributing, forbidden, invalid_request, ...).
export class AnalyticsError extends Error {
  readonly code: string
  constructor(code: string) {
    super(code)
    this.code = code
  }
}

export interface Reference {
  groups(language: Language): Promise<DiagnosisGroup[]>
  places(): Promise<Place[]>
  specialties(): Promise<Specialty[]>
  staff(): Promise<Member[]>
  timeZone(): Promise<string> // the clinic's IANA zone, which its statistics' days and months are in
}

export interface Stats {
  clinicAppointments(q: ClinicQuery): Promise<ClinicAppointments>
  clinicDiagnoses(q: ClinicDiagnosisQuery): Promise<{ rows: Row[] }>
  platformMonths(): Promise<PlatformMonths>
  platformCells(q: PlatformCellsQuery): Promise<PlatformCells>
  platformHeatmap(q: { quarter: string; region?: string }): Promise<{ cells: HeatCell[] }>
  platformRises(q: { region?: string; weeks: number }): Promise<{ rises: Rise[] }>
}

export type AnalyticsData = Reference & Stats

// platformEnabled is the analytics_platform feature of GET /auth/me (plan §2a).
export function platformEnabled(account: Account): boolean {
  return account.features?.includes('analytics_platform') === true
}

// analyticsData is the app's data, all of it from the API.
export function analyticsData(api: ApiClient, fhir: FhirClient, account: Account): AnalyticsData {
  return { ...apiReference(api, fhir, account.organizationId ?? ''), ...apiStats(api) }
}

async function read<T>(call: Promise<{ data?: T; error?: unknown; response: Response }>): Promise<T> {
  const res = await attempt(call)
  if (!res.ok) throw new AnalyticsError(res.code ?? 'unavailable')
  return res.data
}

function apiReference(api: ApiClient, fhir: FhirClient, organizationId: string): Reference {
  return {
    groups: async (language) => (await read(api.GET('/terminology/diagnosis-groups', { params: { query: { language } } }))).groups,
    places: async () => (await read(api.GET('/places'))).places,
    specialties: async () => (await read(api.GET('/specialties'))).specialties,
    staff: () => loadStaff(fhir, organizationId),
    timeZone: () => loadTimeZone(fhir, organizationId),
  }
}

// days is a range of months as the clinic's endpoints take it: the first day of from to the last day of to.
function days(q: ClinicQuery) {
  const [y, m] = q.to.split('-').map(Number)
  const last = new Date(Date.UTC(y, m, 0)).getUTCDate()
  return { from: `${q.from}-01`, to: `${q.to}-${String(last).padStart(2, '0')}` }
}

// filters is a clinic query's narrowing, only the ones set.
function filters(q: ClinicQuery) {
  return { ...(q.specialty ? { specialty: q.specialty } : {}), ...(q.practitionerRole ? { practitionerRole: q.practitionerRole } : {}) }
}

// apiStats is the statistics of the generated client.
export function apiStats(api: ApiClient): Stats {
  return {
    clinicAppointments: (q) => read(api.GET('/analytics/clinic/appointments', { params: { query: { ...days(q), ...filters(q) } } })),
    clinicDiagnoses: (q) => {
      const diagnosis = { ...(q.group ? { group: q.group } : {}), ...(q.subgroup ? { subgroup: q.subgroup } : {}) }
      return read(api.GET('/analytics/clinic/diagnoses', { params: { query: { ...days(q), by: q.by, ...diagnosis, ...filters(q) } } }))
    },
    platformMonths: () => read(api.GET('/analytics/platform/months')),
    platformCells: (q) => read(api.GET('/analytics/platform/cells', { params: { query: { table: q.table, from: q.from, to: q.to, by: q.by, ...q.filter } } })),
    platformHeatmap: (q) => read(api.GET('/analytics/platform/heatmap', { params: { query: q } })),
    platformRises: (q) => read(api.GET('/analytics/platform/rises', { params: { query: q } })),
  }
}

// monthsBetween is every month from from to to, both included.
export function monthsBetween(from: Month, to: Month): Month[] {
  const out: Month[] = []
  let [y, m] = from.split('-').map(Number)
  const end = to
  for (;;) {
    const month = `${y}-${String(m).padStart(2, '0')}`
    if (month > end) return out
    out.push(month)
    m += 1
    if (m > 12) [y, m] = [y + 1, 1]
  }
}

// noShowRate is the share of the closed appointments that were missed, in whole percent: no-shows over fulfilled
// plus no-shows (booked ones are still ahead, cancelled ones were never due). 0 without any closed one.
export function noShowRate(noshow: number, fulfilled: number): number {
  const closed = noshow + fulfilled
  return closed === 0 ? 0 : Math.round((100 * noshow) / closed)
}
