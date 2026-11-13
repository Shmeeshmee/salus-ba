import { describe, expect, it } from 'vitest'
import { newApiClient } from '../api/client'
import { apiError, fakeFetch } from '../test/fakeFetch'
import { admin } from '../test/fixtures'
import { AnalyticsError, apiStats, dimensionsOf, monthsBetween, platformEnabled } from './data'
import { fixtureData } from './fixtures'

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

describe('monthsBetween', () => {
  it('lists the months from to to, both included', () => {
    expect(monthsBetween('2025-11', '2026-02')).toEqual(['2025-11', '2025-12', '2026-01', '2026-02'])
    expect(monthsBetween('2026-02', '2026-01')).toEqual([])
  })
})

// query is a request's query parameters, in their order.
const query = (r: Request) => [...new URL(r.url).searchParams.entries()]

describe('apiStats', () => {
  it("asks the clinic's figures from the first day of the first month to the last day of the last", async () => {
    const series = [{ period: '2026-01', booked: 9, fulfilled: 7, noshow: 1, cancelled: 1 }]
    const { fetch, requests } = fakeFetch({
      'GET /analytics/clinic/appointments': () => Response.json({ series, heatmap: [{ weekday: 1, hour: 9, count: 4 }] }),
      'GET /analytics/clinic/diagnoses': () => Response.json({ rows: [{ key: 'flu', count: 3 }] }),
    })
    const stats = apiStats(newApiClient(fetch))

    expect(await stats.clinicAppointments({ from: '2026-01', to: '2026-02', specialty: '394814009' })).toEqual({ series, heatmap: [{ weekday: 1, hour: 9, count: 4 }] })
    expect(await stats.clinicDiagnoses({ from: '2024-01', to: '2024-02', practitionerRole: 'r1', group: 'respiratory-infections', by: 'subgroup' })).toEqual({ rows: [{ key: 'flu', count: 3 }] })

    expect(query(requests[0])).toEqual([['from', '2026-01-01'], ['to', '2026-02-28'], ['specialty', '394814009']])
    expect(query(requests[1])).toEqual([['from', '2024-01-01'], ['to', '2024-02-29'], ['by', 'subgroup'], ['group', 'respiratory-infections'], ['practitionerRole', 'r1']])
  })

  it('asks the platform for one table with its filters, a quarter, and the rises', async () => {
    const { fetch, requests } = fakeFetch({
      'GET /analytics/platform/months': () => Response.json({ months: ['2026-07', '2026-08', '2026-09'], quarters: ['2026-Q3'], minCount: 10 }),
      'GET /analytics/platform/cells': () => Response.json({ rows: [{ key: 'BA-09', count: null }], partial: true }),
      'GET /analytics/platform/heatmap': () => Response.json({ cells: [{ weekday: 2, hour: 10, count: 15 }] }),
      'GET /analytics/platform/rises': () => Response.json({ rises: [{ week: '2026-W44', region: 'BA-09', group: 'flu' }] }),
    })
    const stats = apiStats(newApiClient(fetch))

    expect(await stats.platformMonths()).toEqual({ months: ['2026-07', '2026-08', '2026-09'], quarters: ['2026-Q3'], minCount: 10 })
    expect(await stats.platformCells({ table: 'gd', from: '2026-07', to: '2026-09', filter: { entity: 'BA-BIH', group: 'flu' }, by: 'region' })).toEqual({ rows: [{ key: 'BA-09', count: null }], partial: true })
    expect((await stats.platformHeatmap({ quarter: '2026-Q3', region: 'BA-09' })).cells).toHaveLength(1)
    expect((await stats.platformRises({ weeks: 8 })).rises).toHaveLength(1)

    expect(query(requests[1])).toEqual([['table', 'gd'], ['from', '2026-07'], ['to', '2026-09'], ['by', 'region'], ['entity', 'BA-BIH'], ['group', 'flu']])
    expect(query(requests[2])).toEqual([['quarter', '2026-Q3'], ['region', 'BA-09']])
    expect(query(requests[3])).toEqual([['weeks', '8']])
  })

  it("turns a refusal into the API's code", async () => {
    const { fetch } = fakeFetch({
      'GET /analytics/platform/months': () => apiError(403, 'analytics_not_contributing'),
      'GET /analytics/clinic/appointments': () => apiError(403, 'forbidden'),
    })
    const stats = apiStats(newApiClient(fetch))
    await expect(stats.platformMonths()).rejects.toMatchObject({ code: 'analytics_not_contributing' })
    await expect(stats.clinicAppointments({ from: '2026-01', to: '2026-01', specialty: '394814009' })).rejects.toMatchObject({ code: 'forbidden' })
  })
})

describe('fixtureData', () => {
  const data = fixtureData()

  it('gives a month series for every month of the range', async () => {
    const res = await data.clinicAppointments({ from: '2024-11', to: '2026-10' })
    expect(res.series.map((s) => s.period)).toEqual(monthsBetween('2024-11', '2026-10'))
    expect(res.heatmap.every((c) => c.weekday >= 1 && c.weekday <= 7 && c.hour >= 0 && c.hour <= 23)).toBe(true)
  })

  it('refuses a practitioner who is not an admin a specialty or another role, as the API does', async () => {
    const mine = fixtureData({ admin: false, role: 'r-own' })
    const range = { from: '2026-01', to: '2026-10' }
    await expect(mine.clinicAppointments({ ...range, specialty: '394814009' })).rejects.toMatchObject({ code: 'forbidden' })
    await expect(mine.clinicDiagnoses({ ...range, specialty: '394814009', by: 'group' })).rejects.toMatchObject({ code: 'forbidden' })
    await expect(mine.clinicAppointments({ ...range, practitionerRole: 'r-other' })).rejects.toMatchObject({ code: 'forbidden' })
    await expect(mine.clinicAppointments({ ...range, practitionerRole: 'r-own' })).resolves.toBeDefined()
    await expect(fixtureData({ admin: true }).clinicAppointments({ ...range, specialty: '394814009' })).resolves.toBeDefined()
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
    const published = await data.platformMonths()
    expect(published.months.at(-1)).toBe('2026-10')
    expect(published.quarters.at(-1)).toBe('2026-Q3')
    expect(published.quarters[0]).toBe('2024-Q1')
  })
})
