import { useId } from 'react'
import { useFormat, useT } from '../i18n/useT'
import { ChartFrame, HatchPattern, HiddenNote, Tooltip } from './chart'
import { niceMax, useRoving, useValue, useWidth } from './marks'
import type { Count, Month } from './data'

// A value undefined is a month with nothing to count (a rate without a denominator): its line breaks, its mark is
// drawn as nothing and named by missing.
export type LineSeries = { key: string; label: string; color: string; values: (Count | undefined)[]; missing?: string }

const height = 240
const pad = { left: 48, right: 16, top: 12, bottom: 28 }

// LineChart is counts over months, one line per series (one axis, starting at 0). A hidden month breaks its line
// and is marked on the baseline with the hidden texture. Every point is a mark: one tab stop, the arrow keys move
// along a line (left, right) and between lines (up, down).
export function LineChart({ title, periods, series }: { title: string; periods: Month[]; series: LineSeries[] }) {
  const t = useT()
  const { month } = useFormat()
  const value = useValue()
  const hatch = useId()
  const note = useId()
  const { ref, width } = useWidth()
  const n = periods.length
  const { shown, markProps } = useRoving(n * series.length, Math.max(n, 1))
  const empty = n === 0 || series.length === 0

  const max = niceMax(Math.max(0, ...series.flatMap((s) => s.values.map((v) => v ?? 0))))
  const text = (s: LineSeries, v: Count | undefined) => (v === undefined ? (s.missing ?? '') : value(v))
  const plotW = width - pad.left - pad.right
  const plotH = height - pad.top - pad.bottom
  const x = (i: number) => pad.left + (n === 1 ? plotW / 2 : (i * plotW) / (n - 1))
  const y = (v: number) => pad.top + plotH - (v / max) * plotH
  const ticks = [0, 0.25, 0.5, 0.75, 1].map((f) => f * max)
  const labelEvery = Math.max(1, Math.ceil(n / Math.max(2, Math.floor(plotW / 72)))) // a month label per 72 px
  const long = periods.map((p) => month(p, 'long'))

  const table = { columns: [t('analytics.month'), ...series.map((s) => s.label)], rows: long.map((name, i) => [name, ...series.map((s) => text(s, s.values[i]))]) }
  const legend = series.length > 1 ? series.map((s) => ({ key: s.key, label: s.label, color: s.color })) : undefined

  const tip = shown === null || empty ? null : { s: Math.floor(shown / n), i: shown % n }
  return (
    <ChartFrame plotRef={ref} title={title} empty={empty} legend={legend} table={table}>
      <svg viewBox={`0 0 ${width} ${height}`} className="chart-svg">
        <HatchPattern id={hatch} />
        {ticks.map((v) => (
          <g key={v} className="axis" aria-hidden="true">
            <line x1={pad.left} x2={width - pad.right} y1={y(v)} y2={y(v)} />
            <text x={pad.left - 8} y={y(v) + 4} textAnchor="end">
              {value(v)}
            </text>
          </g>
        ))}
        {periods.map((p, i) =>
          i % labelEvery === 0 ? (
            <text key={p} className="axis" x={x(i)} y={height - 8} textAnchor="middle" aria-hidden="true">
              {month(p, 'short')}
            </text>
          ) : null,
        )}
        {series.map((s) => (
          <path key={s.key} className="line" d={pathOf(s.values, x, y)} stroke={s.color} aria-hidden="true" />
        ))}
        {series.map((s, si) =>
          s.values.map((v, i) => {
            const k = si * n + i
            const cy = v === null || v === undefined ? y(0) : y(v)
            return (
              <g key={`${s.key}-${i}`} className={shown === k ? 'mark active' : 'mark'} role="img" aria-label={`${long[i]}, ${s.label}: ${text(s, v)}`} aria-describedby={v === null ? note : undefined} {...markProps(k)}>
                <circle cx={x(i)} cy={cy} r={12} className="hit" />
                {v === undefined ? null : v === null ? (
                  <rect x={x(i) - 5} y={cy - 5} width={10} height={10} rx={2} fill={`url(#${hatch})`} stroke="var(--color-text-muted)" />
                ) : (
                  <circle cx={x(i)} cy={cy} r={shown === k ? 5 : 3} fill={s.color} stroke="var(--color-surface)" strokeWidth={2} />
                )}
              </g>
            )
          }),
        )}
      </svg>
      <HiddenNote id={note} />
      {tip && (
        <Tooltip
          at={[x(tip.i), y(series[tip.s].values[tip.i] ?? 0)]}
          width={width}
          height={height}
          title={long[tip.i]}
          lines={series.map((s) => `${s.label}: ${text(s, s.values[tip.i])}`)}
          hidden={series[tip.s].values[tip.i] === null}
        />
      )}
    </ChartFrame>
  )
}

// pathOf is the line through the values, broken at every hidden one and every one with nothing to count.
function pathOf(values: (Count | undefined)[], x: (i: number) => number, y: (v: number) => number): string {
  let d = ''
  let pen = false
  values.forEach((v, i) => {
    if (v === null || v === undefined) {
      pen = false
      return
    }
    d += `${pen ? 'L' : 'M'}${x(i).toFixed(1)},${y(v).toFixed(1)}`
    pen = true
  })
  return d
}
