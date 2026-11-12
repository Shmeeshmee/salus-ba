import { useCallback, useRef, useState, type KeyboardEvent } from 'react'
import { useFormat, useT } from '../i18n/useT'
import type { Count } from './data'

// marks.ts: the charts' geometry and the roving focus over their marks (chart.tsx has the shared components).

// useWidth is the plot's width in CSS pixels, measured as it changes, so a chart draws at its real size and its
// text never scales with the column; 720 until measured (and in tests, which have no layout).
export function useWidth() {
  const [width, setWidth] = useState(720)
  const observer = useRef<ResizeObserver | null>(null)
  const ref = useCallback((el: HTMLDivElement | null) => {
    observer.current?.disconnect()
    if (!el || typeof ResizeObserver === 'undefined') return
    observer.current = new ResizeObserver(([entry]) => {
      const w = Math.round(entry.contentRect.width)
      if (w > 0) setWidth(w)
    })
    observer.current.observe(el)
  }, [])
  return { ref, width }
}

// useValue writes a count: the number in the UI language, or "Hidden".
export function useValue() {
  const t = useT()
  const { number } = useFormat()
  return (count: Count) => (count === null ? t('analytics.hidden') : number(count))
}

// niceMax is the axis top: the smallest 1, 2, 2.5 or 5 times a power of ten at or above max.
export function niceMax(max: number): number {
  if (max <= 0) return 1
  const power = 10 ** Math.floor(Math.log10(max))
  return [1, 2, 2.5, 5, 10].map((f) => f * power).find((v) => v >= max)!
}

// useRoving gives the marks one tab stop: the arrow keys move between them on a grid of cols columns (one column:
// up and down, left and right alike), Home and End to the row's ends; shown is the mark with its tooltip.
export function useRoving(count: number, cols: number, onEnter?: (i: number) => void) {
  const [active, setActive] = useState(0)
  const [shown, setShown] = useState<number | null>(null)
  const refs = useRef<(SVGElement | null)[]>([])
  const current = Math.min(active, Math.max(count - 1, 0))

  function target(i: number, key: string): number | undefined {
    const col = i % cols
    const one = cols === 1
    switch (key) {
      case 'ArrowRight':
        return one || col < cols - 1 ? i + 1 : undefined
      case 'ArrowLeft':
        return one || col > 0 ? i - 1 : undefined
      case 'ArrowDown':
        return i + cols
      case 'ArrowUp':
        return i - cols
      case 'Home':
        return one ? 0 : i - col
      case 'End':
        return one ? count - 1 : i - col + cols - 1
    }
    return undefined
  }

  function onKeyDown(event: KeyboardEvent, i: number) {
    if ((event.key === 'Enter' || event.key === ' ') && onEnter) {
      event.preventDefault()
      onEnter(i)
      return
    }
    const to = target(i, event.key)
    if (to === undefined) return
    event.preventDefault()
    if (to < 0 || to >= count) return
    setActive(to)
    setShown(to)
    refs.current[to]?.focus()
  }

  function markProps(i: number) {
    return {
      ref: (el: SVGElement | null) => {
        refs.current[i] = el
      },
      tabIndex: i === current ? 0 : -1,
      onFocus: () => {
        setActive(i)
        setShown(i)
      },
      onBlur: () => setShown(null),
      onMouseEnter: () => setShown(i),
      onMouseLeave: () => setShown(null),
      onKeyDown: (event: KeyboardEvent) => onKeyDown(event, i),
    }
  }
  return { shown, markProps }
}

