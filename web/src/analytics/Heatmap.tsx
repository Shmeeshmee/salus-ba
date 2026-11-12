import { useId } from 'react'
import { useFormat, useT } from '../i18n/useT'
import { ChartFrame, HatchPattern, Tooltip } from './chart'
import { useRoving, useValue, useWidth } from './marks'
import type { HeatCell } from './data'

const days = [1, 2, 3, 4, 5, 6, 7]
const hours = Array.from({ length: 24 }, (_, h) => h)
const labelW = 48
const cellH = 24
const top = 4
const axisH = 22
const height = top + days.length * cellH + axisH

// The sequential ramp of one hue (the brand's green), light to dark; 0 is the page colour.
const ramp = ['#DDF7EA', '#A8E9CB', '#5FCB9D', '#219A6C', '#0A6E4F']

// Heatmap is a count per ISO weekday (rows, Monday first) and local hour (columns, 0 to 23); a cell not given is
// 0. One tab stop: left and right move by hour, up and down by day, Home and End to the day's ends.
export function Heatmap({ title, cells }: { title: string; cells: HeatCell[] }) {
  const t = useT()
  const { weekday } = useFormat()
  const value = useValue()
  const hatch = useId()
  const { ref, width } = useWidth()
  const cellW = (width - labelW) / hours.length
  const { shown, markProps } = useRoving(days.length * hours.length, hours.length)
  const byKey = new Map(cells.map((c) => [`${c.weekday}-${c.hour}`, c.count]))
  const at = (d: number, h: number) => {
    const k = `${d}-${h}`
    return byKey.has(k) ? (byKey.get(k) as number | null) : 0
  }
  const max = Math.max(0, ...cells.map((c) => c.count ?? 0))
  const empty = max === 0 && cells.every((c) => c.count !== null)
  const anyHidden = cells.some((c) => c.count === null)
  const hour = (h: number) => `${String(h).padStart(2, '0')}:00`
  const slot = (d: number, h: number) => `${weekday(d)} ${hour(h)}–${hour((h + 1) % 24)}`
  const fill = (c: number) => (c === 0 ? 'var(--color-page)' : ramp[Math.min(ramp.length - 1, Math.floor((c / max) * ramp.length))])

  const legend = [
    ...ramp.map((color, i) => ({ key: color, label: i === 0 ? t('analytics.fewer') : i === ramp.length - 1 ? t('analytics.more') : '', color })),
    ...(anyHidden ? [{ key: 'hidden', label: t('analytics.hidden'), hatched: true }] : []),
  ]
  const table = { columns: [t('analytics.weekday'), ...hours.map(hour)], rows: days.map((d) => [weekday(d), ...hours.map((h) => value(at(d, h)))]) }
  const tip = shown === null || empty ? null : { d: days[Math.floor(shown / hours.length)], h: shown % hours.length }
  return (
    <ChartFrame plotRef={ref} title={title} empty={empty} legend={legend} table={table}>
      <svg viewBox={`0 0 ${width} ${height}`} className="chart-svg">
        <HatchPattern id={hatch} />
        {days.map((d, di) => (
          <text key={d} className="axis" x={labelW - 8} y={top + di * cellH + cellH / 2 + 4} textAnchor="end" aria-hidden="true">
            {weekday(d)}
          </text>
        ))}
        {hours.map((h) =>
          h % 3 === 0 ? (
            <text key={h} className="axis" x={labelW + h * cellW + 2} y={height - 6} aria-hidden="true">
              {String(h).padStart(2, '0')}
            </text>
          ) : null,
        )}
        {days.map((d, di) =>
          hours.map((h) => {
            const k = di * hours.length + h
            const c = at(d, h)
            return (
              <rect
                key={k}
                className={shown === k ? 'mark cell active' : 'mark cell'}
                role="img"
                aria-label={`${slot(d, h)}: ${value(c)}`}
                x={labelW + h * cellW + 1}
                y={top + di * cellH + 1}
                width={cellW - 2}
                height={cellH - 2}
                rx={3}
                fill={c === null ? `url(#${hatch})` : fill(c)}
                {...markProps(k)}
              />
            )
          }),
        )}
      </svg>
      {tip && (
        <Tooltip
          at={[labelW + tip.h * cellW + cellW / 2, top + days.indexOf(tip.d) * cellH + cellH]}
          width={width}
          height={height}
          title={slot(tip.d, tip.h)}
          lines={[value(at(tip.d, tip.h))]}
          hidden={at(tip.d, tip.h) === null}
        />
      )}
    </ChartFrame>
  )
}
