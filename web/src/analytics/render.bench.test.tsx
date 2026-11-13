import { cleanup } from '@testing-library/react'
import type { ReactNode } from 'react'
import { describe, expect, it } from 'vitest'
import { renderInLanguage } from '../i18n/renderInLanguage'
import { BarChart } from './BarChart'
import { monthsBetween } from './data'
import { fixtureData, fixtureReference } from './fixtures'
import { Heatmap } from './Heatmap'
import { LineChart } from './LineChart'

// The render benchmark of plan §9 ANA-4: 3 years of monthly data with every diagnosis group (27 groups and 7
// subgroups), on the fixtures, in jsdom. Vitest 5 has no bench mode, so this is a test: it renders each chart
// `runs` times and logs the median and the 95th percentile. It asserts what was drawn, not the time: inside the
// whole suite the workers share the machine and the times are not stable. Read the numbers from a run of this file
// alone (`npx vitest run src/analytics/render.bench.test.tsx` in the node service); they are in
// docs/plans/analytics.PROGRESS.md.

const periods = monthsBetween('2023-11', '2026-10')
const data = fixtureData()
const groups = await fixtureReference().groups('en')
const perGroup = await Promise.all(
  groups.map(async (g) => {
    const q = g.parent ? { group: g.parent, subgroup: g.code } : { group: g.code }
    return (await data.clinicDiagnoses({ from: periods[0], to: periods.at(-1)!, ...q, by: 'month' })).rows.map((r) => r.count)
  }),
)
const series = groups.map((g, i) => ({ key: g.code, label: g.name, color: `var(--series-${(i % 4) + 1})`, values: perGroup[i] }))
const bars = (await data.clinicDiagnoses({ from: periods[0], to: periods.at(-1)!, by: 'group' })).rows.map((r) => ({ key: r.key, label: `Group ${r.key}`, count: r.count }))
const cells = (await data.platformHeatmap({ quarter: '2026-Q3' })).cells

const runs = 30

// measure renders ui runs times (after 3 warm-up renders) and gives the median and the 95th percentile in ms, and
// the number of marks drawn.
function measure(ui: () => ReactNode): { median: number; p95: number; marks: number } {
  const times: number[] = []
  let marks = 0
  for (let i = 0; i < runs + 3; i++) {
    const start = performance.now()
    const { container } = renderInLanguage(ui(), 'en')
    const took = performance.now() - start
    marks = container.querySelectorAll('.mark').length
    cleanup()
    if (i >= 3) times.push(took)
  }
  times.sort((a, b) => a - b)
  return { median: times[Math.floor(runs / 2)], p95: times[Math.floor(runs * 0.95)], marks }
}

describe('render, 3 years monthly, every group', () => {
  const cases: [string, () => ReactNode, number][] = [
    [`line chart, ${series.length} series x ${periods.length} months`, () => <LineChart title="Diagnoses" periods={periods} series={series} />, 34 * 36],
    [`bar chart, ${bars.length} groups summed over ${periods.length} months`, () => <BarChart title="Groups" bars={bars} onSelect={() => {}} />, 27],
    ['heatmap, 7 x 24 cells', () => <Heatmap title="Hours" cells={cells} />, 7 * 24],
  ]
  it.each(cases)('renders the %s', { timeout: 60_000 }, (name, ui, want) => {
    const { median, p95, marks } = measure(ui)
    console.info(`render ${name}: median ${median.toFixed(1)} ms, p95 ${p95.toFixed(1)} ms`)
    expect(marks).toBe(want)
  })
})
