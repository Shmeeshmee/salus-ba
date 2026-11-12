import { useId } from 'react'
import { useT } from '../i18n/useT'
import { TipTag } from '../ui'
import { ChartFrame, HatchPattern, Tooltip } from './chart'
import { niceMax, useRoving, useValue, useWidth } from './marks'
import type { Count } from './data'

export type Bar = { key: string; label: string; count: Count }

type Props = {
  title: string
  bars: Bar[]
  unit?: string // "Patient-months" for sums over several months
  partial?: boolean // a hidden cell is left out of the sums
  onSelect?: (key: string) => void // opens the bar's breakdown (drill-down)
}

const row = 30
const valueW = 64
const top = 4

// BarChart is one count per key, horizontal bars largest first, hidden ones last with the hidden texture and the
// word "Hidden". Every bar is a mark: one tab stop, up and down move, Enter opens the bar when onSelect is given.
export function BarChart({ title, bars, unit, partial, onSelect }: Props) {
  const t = useT()
  const value = useValue()
  const hatch = useId()
  const { ref, width } = useWidth()
  const labelW = Math.min(230, Math.round(width * 0.4)) // narrower on a phone
  const sorted = [...bars].sort((a, b) => (b.count ?? -1) - (a.count ?? -1) || a.label.localeCompare(b.label))
  const { shown, markProps } = useRoving(sorted.length, 1, onSelect && ((i) => onSelect(sorted[i].key)))
  const empty = bars.every((b) => b.count === 0)

  const height = top * 2 + sorted.length * row
  const max = niceMax(Math.max(0, ...sorted.map((b) => b.count ?? 0)))
  const plotW = width - labelW - valueW
  const len = (c: number) => Math.max(c > 0 ? 3 : 0, (c / max) * plotW)

  const note = (unit || partial) && (
    <>
      {unit && <span className="muted small">{unit}</span>}
      {partial && <TipTag className="partial" label={t('analytics.partial')} tip={t('analytics.partialTip')} />}
    </>
  )
  const table = { columns: [t('analytics.name'), unit ?? t('analytics.count')], rows: sorted.map((b) => [b.label, value(b.count)]) }
  const tip = shown === null || empty ? undefined : sorted[shown]
  return (
    <ChartFrame plotRef={ref} title={title} empty={empty} note={note || undefined} table={table}>
      <svg viewBox={`0 0 ${width} ${height}`} className="chart-svg">
        <HatchPattern id={hatch} />
        {sorted.map((b, i) => {
          const y = top + i * row
          const name = `${b.label}: ${value(b.count)}`
          return (
            <g
              key={b.key}
              className={[onSelect ? 'mark bar openable' : 'mark bar', shown === i ? 'active' : ''].join(' ')}
              role={onSelect ? 'button' : 'img'}
              aria-label={onSelect ? t('analytics.openBar', { name }) : name}
              onClick={onSelect && (() => onSelect(b.key))}
              {...markProps(i)}
            >
              <rect className="hit" x={0} y={y} width={width} height={row} />
              <text className="bar-label" x={labelW - 10} y={y + row / 2 + 5} textAnchor="end">
                {short(b.label, Math.floor(labelW / 7.5))}
              </text>
              {b.count === null ? (
                <>
                  <rect x={labelW} y={y + 6} width={56} height={row - 12} rx={4} fill={`url(#${hatch})`} />
                  <text className="bar-value muted" x={labelW + 64} y={y + row / 2 + 5}>
                    {t('analytics.hidden')}
                  </text>
                </>
              ) : (
                <>
                  <rect className="bar-fill" x={labelW} y={y + 6} width={len(b.count)} height={row - 12} rx={4} />
                  <text className="bar-value" x={labelW + len(b.count) + 8} y={y + row / 2 + 5}>
                    {value(b.count)}
                  </text>
                </>
              )}
            </g>
          )
        })}
      </svg>
      {tip && (
        <Tooltip
          at={[labelW + (tip.count === null ? 56 : len(tip.count)), top + sorted.indexOf(tip) * row + row]}
          width={width}
          height={height}
          title={tip.label}
          lines={[value(tip.count)]}
          hidden={tip.count === null}
        />
      )}
    </ChartFrame>
  )
}

// short cuts a long name to fit the label column (max characters); the full name is in the mark's name, its
// tooltip and the table.
function short(label: string, max: number): string {
  return label.length > max ? `${label.slice(0, max - 1).trimEnd()}…` : label
}
