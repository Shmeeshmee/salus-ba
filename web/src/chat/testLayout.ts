// testLayout stands in for the browser's layout in jsdom, for the tests of the virtualized list: the scroller
// (role log) is viewport pixels high and each row ([data-row]) is as high as rowHeight says. Test code only.

export type Layout = {
  // top is where the row of key sits from the top of the scroller's view.
  top(scroller: HTMLElement, key: number): number
  // bottomGap is how far the view's bottom is above the end of the list.
  bottomGap(scroller: HTMLElement): number
  restore(): void
}

export function stubLayout({ viewport, rowHeight }: { viewport: number; rowHeight: (row: HTMLElement) => number }): Layout {
  const proto = HTMLElement.prototype
  const rect = proto.getBoundingClientRect
  Object.defineProperty(proto, 'clientHeight', {
    configurable: true,
    get(this: HTMLElement) {
      return this.getAttribute('role') === 'log' ? viewport : 0
    },
  })
  proto.getBoundingClientRect = function (this: HTMLElement) {
    if (this.dataset.row === undefined) return rect.call(this)
    const height = rowHeight(this)
    return { x: 0, y: 0, top: 0, left: 0, right: 0, bottom: height, width: 0, height, toJSON: () => ({}) } as DOMRect
  }
  return {
    top(scroller, key) {
      const row = scroller.querySelector<HTMLElement>(`[data-row="${key}"]`)
      if (!row?.parentElement) return Number.NaN
      let y = parseFloat(row.parentElement.style.top || '0')
      for (let r = row.previousElementSibling; r; r = r.previousElementSibling) y += rowHeight(r as HTMLElement)
      return y - scroller.scrollTop
    },
    bottomGap(scroller) {
      const total = parseFloat((scroller.firstElementChild as HTMLElement).style.height)
      return total - (scroller.scrollTop + viewport)
    },
    restore() {
      delete (proto as { clientHeight?: number }).clientHeight
      proto.getBoundingClientRect = rect
    },
  }
}
