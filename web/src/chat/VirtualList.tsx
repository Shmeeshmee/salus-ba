import { useLayoutEffect, useMemo, useReducer, useRef, useState, type ReactNode } from 'react'

// VirtualList renders a long list (a conversation's thousands of messages) with only the rows in view and a fixed
// overscan in the DOM. A row's height is the estimate until it is rendered and measured; the offsets are the sums
// of the heights, so variable heights are fine. The view is anchored: at the bottom while the reader is there (a
// new row keeps it at the bottom), else on the first row in view, which stays where it was when rows come in above
// it (older pages) or when a row above changes height. Items are sorted by their numeric key, ascending.

type Props<T> = {
  items: T[]
  keyOf: (item: T) => number
  render: (item: T) => ReactNode
  label: string
  // onScrolled says, after each move of the view, whether it is within a screen of the top or at the bottom.
  onScrolled?: (where: { nearTop: boolean; atBottom: boolean }) => void
  estimate?: number
  overscan?: number
}

type Anchor = 'bottom' | { key: number; delta: number }

// indexAt is the index of the row at pixel y: the last offset at or above it.
function indexAt(offsets: Float64Array, y: number): number {
  let lo = 0
  let hi = offsets.length - 2
  while (lo < hi) {
    const mid = (lo + hi + 1) >> 1
    if (offsets[mid] <= y) lo = mid
    else hi = mid - 1
  }
  return Math.max(0, lo)
}

// indexOf is the index of the item with key, or -1.
function indexOf<T>(items: T[], keyOf: (item: T) => number, key: number): number {
  let lo = 0
  let hi = items.length - 1
  while (lo <= hi) {
    const mid = (lo + hi) >> 1
    const k = keyOf(items[mid])
    if (k === key) return mid
    if (k < key) lo = mid + 1
    else hi = mid - 1
  }
  return -1
}

export function VirtualList<T>({ items, keyOf, render, label, onScrolled, estimate = 64, overscan = 8 }: Props<T>) {
  const scroller = useRef<HTMLDivElement>(null)
  const heights = useRef(new Map<number, number>())
  const anchor = useRef<Anchor>('bottom')
  const [view, setView] = useState({ top: 0, height: 0 })
  const [measured, remeasure] = useReducer((n: number) => n + 1, 0)

  const offsets = useMemo(() => {
    const o = new Float64Array(items.length + 1)
    for (let i = 0; i < items.length; i++) o[i + 1] = o[i] + (heights.current.get(keyOf(items[i])) ?? estimate)
    return o
    // measured is the heights' version: the map is a ref.
    // oxlint-disable-next-line react-hooks/exhaustive-deps
  }, [items, keyOf, estimate, measured])
  const total = offsets[items.length]
  const first = Math.max(0, indexAt(offsets, view.top) - overscan)
  const last = Math.min(items.length, indexAt(offsets, view.top + view.height) + 1 + overscan)

  // anchored is the scroll position the anchor asks for.
  function anchored(height: number): number {
    const a = anchor.current
    const bottom = Math.max(0, total - height)
    if (a === 'bottom') return bottom
    const i = indexOf(items, keyOf, a.key)
    return i < 0 ? Math.min(view.top, bottom) : Math.min(offsets[i] + a.delta, bottom)
  }

  // After every render: measure the rows drawn, then put the anchor back where it was. It sets state only when a
  // height or the view changed, so it settles once every row drawn is measured.
  // oxlint-disable-next-line react-hooks/exhaustive-deps
  useLayoutEffect(() => {
    const el = scroller.current
    if (!el) return
    let changed = false
    for (const row of el.querySelectorAll<HTMLElement>('[data-row]')) {
      const h = row.getBoundingClientRect().height
      const key = Number(row.dataset.row)
      if (h > 0 && heights.current.get(key) !== h) {
        heights.current.set(key, h)
        changed = true
      }
    }
    if (changed) return remeasure()
    const height = el.clientHeight
    const top = anchored(height)
    if (Math.abs(el.scrollTop - top) > 0.5) el.scrollTop = top
    if (top !== view.top || height !== view.height) {
      setView({ top, height })
      onScrolled?.({ nearTop: top < height, atBottom: anchor.current === 'bottom' })
    }
  })

  // A row that changes height after it was drawn (an image that loaded) is measured again.
  useLayoutEffect(() => {
    const el = scroller.current
    if (!el || typeof ResizeObserver === 'undefined') return
    const observer = new ResizeObserver(() => remeasure())
    for (const row of el.querySelectorAll('[data-row]')) observer.observe(row)
    return () => observer.disconnect()
  }, [first, last, items])

  function onScroll() {
    const el = scroller.current
    if (!el) return
    const top = el.scrollTop
    const height = el.clientHeight
    const atBottom = top + height >= total - 4
    if (atBottom) anchor.current = 'bottom'
    else {
      const i = indexAt(offsets, top)
      anchor.current = items[i] === undefined ? 'bottom' : { key: keyOf(items[i]), delta: top - offsets[i] }
    }
    setView({ top, height })
    onScrolled?.({ nearTop: top < height, atBottom })
  }

  return (
    <div ref={scroller} className="chat-scroll" role="log" aria-label={label} tabIndex={0} onScroll={onScroll}>
      <div className="chat-space" style={{ height: total }}>
        <ol className="chat-rows" style={{ top: offsets[first] }}>
          {items.slice(first, last).map((item) => (
            <li key={keyOf(item)} data-row={keyOf(item)}>
              {render(item)}
            </li>
          ))}
        </ol>
      </div>
    </div>
  )
}
