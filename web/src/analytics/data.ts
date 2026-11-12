import type { Account, ApiClient } from '../api/client'
import type { FhirClient } from '../api/fhir'
import { attempt } from '../api/result'
import type { components } from '../api/schema'
import type { Language } from '../i18n/languages'
import { loadStaff, type Member } from '../staff'
import { fixtureStats } from './fixtures'

// data.ts is the analytics screens' one way to their data (docs/plans/analytics.md §8, SAL-22): the shapes of the
// plan's API contract, in camelCase, behind one interface. Until ANA-2 (SAL-20) and ANA-3 (SAL-21) are merged,
// the statistics come from fixtures.ts (synthetic, placeholder); the reference data (groups, places, specialties,
// the clinic's practitioners) is already the API's. The switch to the generated client touches only this file.

export type Month = string // "2026-11"
export type Count = number | null // null: hidden (suppressed), on the platform only

export type AppointmentMonth = { period: Month; booked: number; fulfilled: number; noshow: number; cancelled: number }
export type HeatCell = { weekday: number; hour: number; count: Count } // ISO weekday, 1 = Monday; local hour 0-23
export type ClinicAppointments = {
  series: AppointmentMonth[]
  heatmap: HeatCell[]
  practitioners?: { practitionerRole: string; fulfilled: number }[] // admins only
}
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
export type PlatformCells = { rows: Row[]; partial: boolean }
export type PlatformMonths = { months: Month[]; minCount: number }
export type Rise = { week: string; region: string; group: string } // week "2026-W44"

export type DiagnosisGroup = components['schemas']['DiagnosisGroup']
export type Place = components['schemas']['Place']
export type Specialty = components['schemas']['Specialty']
export type { Member }

// AnalyticsError is a refused call, with the API's code (analytics_not_contributing, invalid_request, ...).
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

// AnalyticsAccount is the account with the features of GET /auth/me (plan §2a), typed here until the client has it.
export type AnalyticsAccount = Account & { features?: string[] }

export function platformEnabled(account: AnalyticsAccount): boolean {
  return account.features?.includes('analytics_platform') === true
}

// analyticsData is the app's data: the API's reference data and, for now, the fixtures' statistics over it.
export function analyticsData(api: ApiClient, fhir: FhirClient, account: Account): AnalyticsData {
  const reference = apiReference(api, fhir, account.organizationId ?? '')
  return { ...reference, ...fixtureStats(reference, { admin: account.role === 'admin' }) }
}

function apiReference(api: ApiClient, fhir: FhirClient, organizationId: string): Reference {
  async function read<T>(call: Promise<{ data?: T; error?: unknown; response: Response }>): Promise<T> {
    const res = await attempt(call)
    if (!res.ok) throw new AnalyticsError(res.code ?? 'unavailable')
    return res.data
  }
  return {
    groups: async (language) => (await read(api.GET('/terminology/diagnosis-groups', { params: { query: { language } } }))).groups,
    places: async () => (await read(api.GET('/places'))).places,
    specialties: async () => (await read(api.GET('/specialties'))).specialties,
    staff: () => loadStaff(fhir, organizationId),
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

// quartersOf is the quarters ("2026-Q3") whose three months are all in months, newest first: the published
// heatmaps (plan §5: built after the quarter).
export function quartersOf(months: Month[]): string[] {
  const have = new Set(months)
  const quarters = new Set<string>()
  for (const month of months) {
    const [y, m] = month.split('-').map(Number)
    const q = Math.ceil(m / 3)
    const all = [0, 1, 2].every((i) => have.has(`${y}-${String((q - 1) * 3 + 1 + i).padStart(2, '0')}`))
    if (all) quarters.add(`${y}-Q${q}`)
  }
  return [...quarters].sort().reverse()
}
