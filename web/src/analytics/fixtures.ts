import {
  AnalyticsError,
  dimensionsOf,
  monthsBetween,
  type AnalyticsData,
  type ClinicAppointments,
  type DiagnosisGroup,
  type Dimension,
  type HeatCell,
  type Member,
  type Month,
  type Place,
  type PlatformCellsQuery,
  type Reference,
  type Row,
  type Specialty,
  type Stats,
} from './data'

// fixtures.ts: PLACEHOLDER statistics for the analytics screens' tests and render benchmark (SAL-22, plan §8), in the
// shapes of api/openapi.yaml. Every number is synthetic, made up from a hash of its keys: no clinic, practitioner or
// patient is behind any of them. They follow the API's visible rules (a practitioner who is not an admin refused a
// specialty or another role; the platform's counts rounded to 5, a small or dominated cell hidden as null, partial
// sums, the quarters whose three months are published), not its suppression (plan §5).

// The diagnosis groups of plan appendix A (key and parent), with synthetic ICD-10 codes inside their ranges.
const groupTable: [code: string, parent: string | undefined, codes: string[]][] = [
  ['intestinal-infections', undefined, ['A04.7', 'A08.4', 'A09']],
  ['gastroenteritis', 'intestinal-infections', ['A08.4', 'A09']],
  ['other-infections', undefined, ['B01.9', 'B34.9']],
  ['covid-19', undefined, ['U07.1', 'U09.9']],
  ['tumours', undefined, ['C50.9', 'D22.5']],
  ['blood-immune', undefined, ['D50.9']],
  ['diabetes', undefined, ['E10.9', 'E11.9']],
  ['endocrine-metabolic', undefined, ['E03.9', 'E78.0']],
  ['mental-health', undefined, ['F32.1', 'F41.1']],
  ['nervous-system', undefined, ['G43.9', 'G47.0']],
  ['eye', undefined, ['H10.3']],
  ['ear', undefined, ['H66.9']],
  ['high-blood-pressure', undefined, ['I10']],
  ['heart-circulation', undefined, ['I25.1', 'I48.9']],
  ['respiratory-infections', undefined, ['J00', 'J02.9', 'J06.9', 'J11.1', 'J18.9', 'J20.9']],
  ['common-cold', 'respiratory-infections', ['J00']],
  ['upper-airway-infections', 'respiratory-infections', ['J02.9', 'J06.9']],
  ['flu', 'respiratory-infections', ['J11.1']],
  ['pneumonia', 'respiratory-infections', ['J18.9']],
  ['chest-infections', 'respiratory-infections', ['J20.9']],
  ['asthma-chronic-lung', undefined, ['J44.9', 'J45.0']],
  ['other-respiratory', undefined, ['J30.1']],
  ['digestive', undefined, ['K21.9', 'K29.7']],
  ['skin', undefined, ['L20.9', 'L30.9']],
  ['musculoskeletal', undefined, ['M17.1', 'M54.5']],
  ['back-neck', 'musculoskeletal', ['M54.5']],
  ['kidney-urinary', undefined, ['N39.0']],
  ['reproductive-breast', undefined, ['N76.0']],
  ['pregnancy', undefined, ['O26.9']],
  ['newborn-congenital', undefined, ['P59.9']],
  ['symptoms', undefined, ['R05', 'R51']],
  ['injuries', undefined, ['S93.4']],
  ['checkups', undefined, ['Z00.0']],
  ['other', undefined, ['W19']],
]
const codesOf = new Map(groupTable.map(([code, , codes]) => [code, codes]))

// A few places of GET /places (ISO 3166-2:BA regions; the cities' ids are made up, not Wikidata's).
const placeTable: [id: string, kind: Place['kind'], parent: string | undefined, latn: string, cyrl: string][] = [
  ['BA-BIH', 'entity', undefined, 'Federacija Bosne i Hercegovine', 'Федерација Босне и Херцеговине'],
  ['BA-01', 'region', 'BA-BIH', 'Unsko-sanski kanton', 'Унско-сански кантон'],
  ['BA-03', 'region', 'BA-BIH', 'Tuzlanski kanton', 'Тузлански кантон'],
  ['BA-04', 'region', 'BA-BIH', 'Zeničko-dobojski kanton', 'Зеничко-добојски кантон'],
  ['BA-07', 'region', 'BA-BIH', 'Hercegovačko-neretvanski kanton', 'Херцеговачко-неретвански кантон'],
  ['BA-09', 'region', 'BA-BIH', 'Kanton Sarajevo', 'Кантон Сарајево'],
  ['BA-SRP', 'region', undefined, 'Republika Srpska', 'Република Српска'],
  ['BA-BRC', 'region', undefined, 'Brčko distrikt', 'Брчко дистрикт'],
  ['Q-fixture-1', 'city', 'BA-09', 'Centar', 'Центар'],
  ['Q-fixture-2', 'city', 'BA-09', 'Novi Grad', 'Нови Град'],
  ['Q-fixture-3', 'city', 'BA-09', 'Trnovo', 'Трново'],
  ['Q-fixture-4', 'city', 'BA-03', 'Tuzla', 'Тузла'],
  ['Q-fixture-5', 'city', 'BA-SRP', 'Banja Luka', 'Бања Лука'],
  ['Q-fixture-6', 'city', 'BA-SRP', 'Bijeljina', 'Бијељина'],
]

const specialtyTable: [code: string, text: string][] = [
  ['394814009', 'General practice'],
  ['394537008', 'Paediatrics'],
  ['394579002', 'Cardiology'],
  ['394582007', 'Dermatology'],
]

// fixtureReference is reference data for tests: the groups' names are placeholders, not appendix A's.
export function fixtureReference(): Reference {
  const groups: DiagnosisGroup[] = groupTable.map(([code, parent]) => ({ code, parent, name: `Group ${code}`, reviewStatus: 'unreviewed' }))
  const places: Place[] = placeTable.map(([id, kind, parent, nameLatn, nameCyrl]) => ({ id, kind, parent, nameLatn, nameCyrl }))
  const specialties: Specialty[] = specialtyTable.map(([code, text]) => ({ system: 'http://snomed.info/sct', code, display: text, text }))
  const staff: Member[] = [
    { roleId: 'f4000000-0000-4000-8000-0000000000a1', practitionerId: 'f3000000-0000-4000-8000-0000000000a1', name: 'Cy Example', active: true, version: '1' },
    { roleId: 'f4000000-0000-4000-8000-0000000000a2', practitionerId: 'f3000000-0000-4000-8000-0000000000a2', name: 'Dee Sample', active: true, version: '1' },
  ]
  return { groups: async () => groups, places: async () => places, specialties: async () => specialties, staff: async () => staff, timeZone: async () => 'Europe/Sarajevo' }
}

// role is the caller's own practitioner role, the only one a practitioner who is not an admin may narrow to.
export type FixtureOptions = { admin?: boolean; role?: string; contributes?: boolean; first?: Month; last?: Month }

// fixtureData is the whole interface on fixtures, for tests and the render benchmark.
export function fixtureData(options: FixtureOptions = {}): AnalyticsData {
  const reference = fixtureReference()
  return { ...reference, ...fixtureStats(reference, options) }
}

// unit is a number in [0, 1) from the keys: the same keys give the same number.
function unit(...keys: (string | number)[]): number {
  let h = 2166136261
  for (const c of keys.join('|')) h = Math.imul(h ^ c.charCodeAt(0), 16777619)
  return ((h >>> 0) % 10007) / 10007
}

// season is higher in winter for the respiratory groups, flat for the others.
function season(group: string, month: Month): number {
  const m = Number(month.slice(5))
  const winter = m <= 2 || m >= 11 ? 1.8 : m === 3 || m === 10 ? 1.3 : 0.8
  return group.includes('respiratory') || ['common-cold', 'flu', 'pneumonia', 'chest-infections', 'upper-airway-infections'].includes(group) ? winter : 1
}

const round5 = (n: number) => Math.round(n / 5) * 5

// quartersOf is the quarters ("2026-Q3") whose three months are all in months, oldest first, as the API publishes them.
function quartersOf(months: Month[]): string[] {
  const have = new Set(months)
  const quarters = new Set<string>()
  for (const month of months) {
    const [y, m] = month.split('-').map(Number)
    const q = Math.ceil(m / 3)
    if ([1, 2, 3].every((i) => have.has(`${y}-${String((q - 1) * 3 + i).padStart(2, '0')}`))) quarters.add(`${y}-Q${q}`)
  }
  return [...quarters].sort()
}

export function fixtureStats(reference: Reference, { admin = true, role, contributes = true, first = '2024-01', last = '2026-10' }: FixtureOptions = {}): Stats {
  // allowed is the API's rule C2: a practitioner who is not an admin narrows to their own role only, never a specialty.
  function allowed(q: { specialty?: string; practitionerRole?: string }) {
    if (!admin && (q.specialty || (q.practitionerRole && q.practitionerRole !== role))) throw new AnalyticsError('forbidden')
  }
  // groupCount is the synthetic count of a group (or subgroup or code) in a month, scaled by the filters' share.
  const groupCount = (key: string, month: Month, scale: number) => Math.round((2 + 30 * unit(key)) * season(key, month) * scale * (0.8 + 0.4 * unit(key, month)))
  const leaves = (q: { group?: string; subgroup?: string }) => codesOf.get(q.subgroup ?? q.group ?? '') ?? []

  async function diagnosisKeys(by: Dimension, filter: Partial<Record<Dimension, string>>): Promise<string[]> {
    const groups = await reference.groups('en')
    if (by === 'group') return groups.filter((g) => !g.parent).map((g) => g.code)
    if (by === 'subgroup') return groups.filter((g) => g.parent === filter.group).map((g) => g.code)
    return leaves(filter)
  }

  async function keysOf(by: Dimension, filter: Partial<Record<Dimension, string>>): Promise<string[]> {
    if (by === 'group' || by === 'subgroup' || by === 'code') return diagnosisKeys(by, filter)
    if (by === 'specialty') return (await reference.specialties()).map((s) => s.code)
    const places = await reference.places()
    if (by === 'entity') return places.filter((p) => p.kind === 'entity').map((p) => p.id)
    if (by === 'region') return places.filter((p) => p.kind === 'region' && (!filter.entity || p.parent === filter.entity)).map((p) => p.id)
    return places.filter((p) => p.kind === 'city' && p.parent === filter.region).map((p) => p.id)
  }

  function checkTable(q: PlatformCellsQuery) {
    const allowed = dimensionsOf[q.table]
    const used = [...(Object.keys(q.filter) as Dimension[]), ...(q.by === 'month' ? [] : [q.by])]
    if (used.some((d) => !allowed.includes(d))) throw new AnalyticsError('invalid_request')
  }

  // platformMonth is one published cell: rounded to 5, or null when small (under 10) or "dominated".
  function platformMonth(key: string, filterKey: string, month: Month): number | null {
    const geography = filterKey.includes('city') ? 0.05 : filterKey.includes('region') ? 0.2 : filterKey.includes('entity') ? 0.6 : 1
    const raw = groupCount(key + filterKey, month, 12 * geography * (filterKey.includes('specialty') ? 0.4 : 1))
    if (raw < 10 || unit('dominated', key, filterKey, month) < 0.04) return null
    return round5(raw)
  }

  return {
    async clinicAppointments(q) {
      allowed(q)
      const scale = (q.specialty ? 0.4 : 1) * (q.practitionerRole ? 0.5 : 1)
      const series = monthsBetween(q.from, q.to).map((period) => {
        const booked = Math.round((90 + 40 * unit('booked', period)) * scale)
        const noshow = Math.round(booked * 0.05)
        const cancelled = Math.round(booked * (0.08 + 0.06 * unit('cancelled', period)))
        return { period, booked, fulfilled: booked - noshow - cancelled, noshow, cancelled }
      })
      const heatmap: ClinicAppointments['heatmap'] = []
      for (let weekday = 1; weekday <= 6; weekday++) {
        const [open, close] = weekday === 6 ? [9, 13] : [8, 19]
        for (let hour = open; hour < close; hour++) heatmap.push({ weekday, hour, count: Math.round((4 + 20 * unit('h', weekday, hour)) * scale * series.length) })
      }
      const staff = await reference.staff()
      const practitioners = admin ? staff.map((m) => ({ practitionerRole: m.roleId, fulfilled: Math.round(40 * series.length * unit(m.roleId)) })) : undefined
      return { series, heatmap, practitioners }
    },

    async clinicDiagnoses(q) {
      allowed(q)
      const scale = (q.specialty ? 0.4 : 1) * (q.practitionerRole || !admin ? 0.5 : 1)
      const months = monthsBetween(q.from, q.to)
      const own = q.subgroup ?? q.group ?? 'all'
      if (q.by === 'month') return { rows: months.map((m) => ({ key: m, count: groupCount(own, m, scale) })) }
      const keys = await diagnosisKeys(q.by, q)
      return { rows: keys.map((key) => ({ key, count: months.reduce((sum, m) => sum + groupCount(key, m, scale), 0) })) }
    },

    async platformMonths() {
      if (!contributes) throw new AnalyticsError('analytics_not_contributing')
      const months = monthsBetween(first, last)
      return { months, quarters: quartersOf(months), minCount: 10 }
    },

    async platformCells(q) {
      checkTable(q)
      const months = monthsBetween(q.from, q.to)
      const filterKey = Object.entries(q.filter).sort().join(';')
      const keys = q.by === 'month' ? months : await keysOf(q.by, q.filter)
      let partial = false
      const rows: Row[] = keys.map((key) => {
        const cells = (q.by === 'month' ? [key] : months).map((m) => platformMonth(key, `${filterKey};${q.by}`, m))
        const shown = cells.filter((c): c is number => c !== null)
        if (shown.length < cells.length) partial = true
        return { key, count: shown.length === 0 ? null : shown.reduce((a, b) => a + b, 0) }
      })
      return { rows, partial }
    },

    async platformHeatmap(q) {
      const scale = q.region ? 12 : 40
      const cells: HeatCell[] = []
      for (let weekday = 1; weekday <= 7; weekday++) {
        for (let hour = 0; hour < 24; hour++) {
          const busy = weekday <= 5 && hour >= 8 && hour < 19 ? 1 : weekday === 6 && hour >= 9 && hour < 14 ? 0.4 : 0.02
          const raw = Math.round(scale * busy * (1 + unit(q.quarter, q.region ?? '', weekday, hour)))
          cells.push({ weekday, hour, count: raw < 10 ? null : round5(raw) })
        }
      }
      return { cells }
    },

    async platformRises(q) {
      const rises = [
        { week: '2026-W44', region: 'BA-09', group: 'respiratory-infections' },
        { week: '2026-W43', region: 'BA-03', group: 'intestinal-infections' },
      ]
      return { rises: rises.filter((r) => !q.region || r.region === q.region) }
    },
  }
}
