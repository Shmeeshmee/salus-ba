import { useId, type ReactNode } from 'react'
import { useT } from '../i18n/useT'
import type { Load } from '../load'
import { Alert } from '../ui'

// The parts the three charts share (plan §2 "Charts": hand-written SVG, no chart library): the frame with its
// title, legend and table view, the tooltip, the hidden texture and the loading state. marks.ts has the rest.

// Tooltip is the shown mark's text at (x, y) of the viewBox: a title, then lines.
export function Tooltip({ at, width, height, title, lines, hidden }: { at: [number, number]; width: number; height: number; title: string; lines: string[]; hidden?: boolean }) {
  const t = useT()
  const [x, y] = at
  return (
    <div className={x > width / 2 ? 'chart-tooltip left' : 'chart-tooltip'} role="tooltip" style={{ left: `${(x / width) * 100}%`, top: `${(y / height) * 100}%` }}>
      <span className="strong">{title}</span>
      {lines.map((l) => (
        <span key={l}>{l}</span>
      ))}
      {hidden && <span className="muted-tip">{t('analytics.hiddenTip')}</span>}
    </div>
  )
}

// HiddenNote is the privacy explanation of a hidden mark for a screen reader: a hidden mark is described by it
// (aria-describedby), so the reason is heard without the tooltip. Not shown; the tooltip shows it.
export function HiddenNote({ id }: { id: string }) {
  const t = useT()
  return (
    <span id={id} hidden>
      {t('analytics.hiddenTip')}
    </span>
  )
}

export type LegendItem = { key: string; label: string; color?: string; hatched?: boolean }

type FrameProps = {
  title: string
  empty: boolean
  legend?: LegendItem[]
  note?: ReactNode
  table: { columns: string[]; rows: string[][] }
  plotRef?: (el: HTMLDivElement | null) => void // measures the plot (useWidth)
  children: ReactNode
}

// ChartFrame is a figure named by its title, with the legend above the plot and the same data as a table below.
export function ChartFrame({ title, empty, legend, note, table, plotRef, children }: FrameProps) {
  const t = useT()
  const id = useId()
  return (
    <figure className="chart" aria-labelledby={id}>
      <figcaption id={id} className="chart-title">
        {title}
      </figcaption>
      {note && <div className="chart-note">{note}</div>}
      {empty ? (
        <p className="muted chart-empty">{t('analytics.noData')}</p>
      ) : (
        <>
          {legend && legend.length > 0 && <Legend items={legend} />}
          <div className="chart-plot" ref={plotRef}>
            {children}
          </div>
          <details className="chart-table">
            <summary>{t('analytics.showTable')}</summary>
            <table>
              <thead>
                <tr>
                  {table.columns.map((c) => (
                    <th key={c} scope="col">
                      {c}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {table.rows.map((r) => (
                  <tr key={r[0]}>
                    {r.map((cell, i) => (i === 0 ? <th key={i} scope="row">{cell}</th> : <td key={i}>{cell}</td>))}
                  </tr>
                ))}
              </tbody>
            </table>
          </details>
        </>
      )}
    </figure>
  )
}

function Legend({ items }: { items: LegendItem[] }) {
  const t = useT()
  return (
    <ul className="chart-legend" aria-label={t('analytics.legend')}>
      {items.map((item) => (
        <li key={item.key} aria-hidden={item.label ? undefined : true}>
          <span className={item.hatched ? 'swatch hatched' : 'swatch'} style={item.color ? { background: item.color } : undefined} aria-hidden="true" />
          {item.label}
        </li>
      ))}
    </ul>
  )
}

// HatchPattern is the fill of a hidden mark: grey diagonal lines, so hidden never reads as a small value.
export function HatchPattern({ id }: { id: string }) {
  return (
    <defs>
      <pattern id={id} width="6" height="6" patternUnits="userSpaceOnUse" patternTransform="rotate(45)">
        <rect width="6" height="6" fill="var(--color-page)" />
        <line x1="0" y1="0" x2="0" y2="6" stroke="var(--color-border-strong)" strokeWidth="2" />
      </pattern>
    </defs>
  )
}

// Loaded shows a chart once its data is in: "Loading" before, the failure as an alert.
export function Loaded<T>({ state, children }: { state: Load<T>; children: (value: T) => ReactNode }) {
  const t = useT()
  if (state.name === 'loading') return <p className="muted chart-empty">{t('common.loading')}</p>
  if (state.name === 'failed') return <Alert message={t('analytics.loadFailed')} />
  return <>{children(state.value)}</>
}
