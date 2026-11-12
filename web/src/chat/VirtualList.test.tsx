import { act, cleanup, fireEvent, render } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { VirtualList } from './VirtualList'
import { stubLayout, type Layout } from './testLayout'

type Item = { seq: number; text: string }
const items = (from: number, to: number): Item[] => Array.from({ length: to - from + 1 }, (_, i) => ({ seq: from + i, text: `Synthetic placeholder ${from + i}` }))

let layout: Layout
beforeEach(() => {
  layout = stubLayout({ viewport: 600, rowHeight: (row) => 40 + 30 * (Number(row.dataset.row) % 3) })
})
afterEach(() => {
  cleanup()
  layout.restore()
})

function list(all: Item[], onScrolled = vi.fn()) {
  const ui = (xs: Item[]) => <VirtualList items={xs} keyOf={(x) => x.seq} render={(x) => <p>{x.text}</p>} label="Synthetic list" onScrolled={onScrolled} />
  const r = render(ui(all))
  const scroller = r.getByRole('log')
  return { ...r, scroller, rerenderWith: (xs: Item[]) => r.rerender(ui(xs)), onScrolled }
}

const shown = (el: HTMLElement) => Array.from(el.querySelectorAll<HTMLElement>('[data-row]')).map((r) => Number(r.dataset.row))

describe('VirtualList', () => {
  it('starts at the bottom and renders only the rows in view and the overscan', () => {
    const { scroller } = list(items(1, 1000))
    const rows = shown(scroller)
    expect(rows.at(-1)).toBe(1000)
    expect(rows.length).toBeLessThan(40)
    expect(scroller.scrollTop).toBeGreaterThan(0)
  })

  it('keeps the first row in view where it was when older rows come in above', () => {
    const { scroller, rerenderWith } = list(items(501, 1000))
    act(() => {
      scroller.scrollTop = 0
      fireEvent.scroll(scroller)
    })
    const top = shown(scroller).find((seq) => layout.top(scroller, seq) >= 0)
    rerenderWith(items(401, 1000))
    expect(top).toBeDefined()
    expect(layout.top(scroller, top!)).toBe(0)
  })

  it('stays at the bottom when a new row arrives there, and stays put when the reader scrolled up', () => {
    const { scroller, rerenderWith } = list(items(1, 200))
    rerenderWith(items(1, 201))
    expect(shown(scroller).at(-1)).toBe(201)
    expect(layout.bottomGap(scroller)).toBe(0)
    act(() => {
      scroller.scrollTop = 1000
      fireEvent.scroll(scroller)
    })
    const seq = shown(scroller).find((k) => layout.top(scroller, k) >= 0)!
    const at = layout.top(scroller, seq)
    rerenderWith(items(1, 202))
    expect(layout.top(scroller, seq)).toBe(at)
    expect(layout.bottomGap(scroller)).toBeGreaterThan(0)
  })

  it('tells when the reader is near the top or at the bottom', () => {
    const { scroller, onScrolled } = list(items(1, 200))
    act(() => {
      scroller.scrollTop = 100
      fireEvent.scroll(scroller)
    })
    expect(onScrolled).toHaveBeenLastCalledWith({ nearTop: true, atBottom: false })
  })
})
