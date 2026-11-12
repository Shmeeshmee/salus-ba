import { describe, expect, it } from 'vitest'
import { AnalyticsError, dimensionsOf, monthsBetween, platformEnabled, quartersOf } from './data'
import { fixtureData } from './fixtures'
import { admin } from '../test/fixtures'

describe('dimensionsOf', () => {
  it('has the three tables of plan §5, each two dimensions deep', () => {
    expect(dimensionsOf.gd).toEqual(['entity', 'region', 'city', 'group', 'subgroup', 'code'])
    expect(dimensionsOf.gs).toEqual(['entity', 'region', 'city', 'specialty'])
    expect(dimensionsOf.sd).toEqual(['specialty', 'group', 'subgroup', 'code'])
  })
})

describe('platformEnabled', () => {
  it('reads analytics_platform from the account features', () => {
    expect(platformEnabled(admin)).toBe(false)
    expect(platformEnabled({ ...admin, features: [] })).toBe(false)
    expect(platformEnabled({ ...admin, features: ['analytics_platform'] })).toBe(true)
  })
})

describe('monthsBetween and quartersOf', () => {
  it('lists the months from to to, both included', () => {
    expect(monthsBetween('2025-11', '2026-02')).toEqual(['2025-11', '2025-12', '2026-01', '2026-02'])
    expect(monthsBetween('2026-02', '2026-01')).toEqual([])
  })

  it('keeps the quarters whose three months are all published, newest first', () => {
    expect(quartersOf(['2026-01', '2026-02', '2026-03', '2026-04', '2026-05'])).toEqual(['2026-Q1'])
    expect(quartersOf(monthsBetween('2025-07', '2026-03'))).toEqual(['2026-Q1', '2025-Q4', '2025-Q3'])
  })
})

describe('fixtureData', () => {
  const data = fixtureData()

  it('gives a month series for every month of the range', async () => {
    const res = await data.clinicAppointments({ from: '2024-11', to: '2026-10' })
    expect(res.series.map((s) => s.period)).toEqual(monthsBetween('2024-11', '2026-10'))
    expect(res.heatmap.every((c) => c.weekday >= 1 && c.weekday <= 7 && c.hour >= 0 && c.hour <= 23)).toBe(true)
  })

  it('gives the practitioners breakdown to admins only', async () => {
    expect((await fixtureData({ admin: true }).clinicAppointments({ from: '2026-01', to: '2026-10' })).practitioners?.length).toBeGreaterThan(0)
    expect((await fixtureData({ admin: false }).clinicAppointments({ from: '2026-01', to: '2026-10' })).practitioners).toBeUndefined()
  })

  it('breaks the diagnoses down by group, then by the subgroups of a group, then by code', async () => {
    const groups = await data.clinicDiagnoses({ from: '2026-01', to: '2026-10', by: 'group' })
    expect(groups.rows.map((r) => r.key)).toContain('respiratory-infections')
    const subgroups = await data.clinicDiagnoses({ from: '2026-01', to: '2026-10', group: 'respiratory-infections', by: 'subgroup' })
    expect(subgroups.rows.map((r) => r.key)).toEqual(['common-cold', 'upper-airway-infections', 'flu', 'pneumonia', 'chest-infections'])
    const codes = await data.clinicDiagnoses({ from: '2026-01', to: '2026-10', group: 'respiratory-infections', subgroup: 'common-cold', by: 'code' })
    expect(codes.rows.map((r) => r.key)).toEqual(['J00'])
  })

  it('refuses a platform filter or breakdown its table does not have', async () => {
    const query = { table: 'sd' as const, from: '2026-01', to: '2026-10' }
    await expect(data.platformCells({ ...query, filter: { region: 'BA-09' }, by: 'group' })).rejects.toThrow(AnalyticsError)
    await expect(data.platformCells({ ...query, filter: {}, by: 'region' })).rejects.toThrow(AnalyticsError)
    await expect(data.platformCells({ ...query, filter: { specialty: '394814009' }, by: 'group' })).resolves.toBeDefined()
  })

  it('hides small platform cells as null, rounds the rest to 5 and marks the answer partial', async () => {
    const res = await data.platformCells({ table: 'gd', from: '2026-01', to: '2026-10', filter: { region: 'BA-09' }, by: 'city' })
    expect(res.rows.some((r) => r.count === null)).toBe(true)
    expect(res.rows.every((r) => r.count === null || r.count % 5 === 0)).toBe(true)
    expect(res.partial).toBe(true)
  })

  it('refuses the platform view of a clinic that does not contribute', async () => {
    await expect(fixtureData({ contributes: false }).platformMonths()).rejects.toMatchObject({ code: 'analytics_not_contributing' })
    expect((await data.platformMonths()).months.at(-1)).toBe('2026-10')
  })
})
