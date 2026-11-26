import AxeBuilder from '@axe-core/playwright'
import { expect, type Locator, type Page } from '@playwright/test'

// The checks of the accessibility journey (a11y.spec.ts, docs/clinic-cms/WAVE7.md RD3). Each adds what it finds to a
// list of findings, so one run names every failure on every screen; the spec expects the list empty at the end.

export const findings: string[] = []

// The axe engine's WCAG 2.2 A and AA rules.
const wcag = ['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa', 'wcag22a', 'wcag22aa']

// axe runs the engine over the page as it is now (a dialog open included).
export async function axe(page: Page, where: string) {
  const result = await new AxeBuilder({ page }).withTags(wcag).analyze()
  if (result.passes.length === 0) findings.push(`${where}: axe checked nothing`)
  for (const v of result.violations) {
    for (const n of v.nodes) {
      const why = (n.failureSummary ?? '').split('\n').slice(1).join(' ').trim()
      findings.push(`${where}: axe ${v.id}: ${n.target.join(' ')}: ${why}`)
    }
  }
}

type Stop = {
  name: string
  order: number // the element's place in the document
  visible: boolean
  covered: string // what is on top of it, or ''
  indicator: string // why the focus is not seen, or ''
  inDialog: boolean
  segmented: boolean
}

// tabWalk presses Tab from the start of the page (or, in a dialog, from where the dialog put the focus) and checks
// every stop: each control comes after the one before it in the document (no jump back, no positive tabindex), it is
// on screen and not covered by something else, and it shows a focus indicator of at least 3:1 against what is around
// it. It ends when the focus leaves the page, comes back to the first stop (a modal keeps Tab inside), or after max.
export async function tabWalk(page: Page, where: string, { dialog = false, modal = false, max = 60 } = {}) {
  if (!dialog) {
    await page.evaluate(() => {
      ;(document.activeElement as HTMLElement | null)?.blur()
      window.scrollTo(0, 0)
    })
  }
  const stops: Stop[] = []
  for (let i = 0; i < max; i++) {
    await page.keyboard.press('Tab')
    const stop = await page.evaluate(focusStop)
    if (!stop) break
    if (stops.length > 0 && stop.order === stops[0].order) break
    if (modal && !stop.inDialog) {
      findings.push(`${where}: Tab leaves the dialog for ${stop.name}`)
      break
    }
    const before = stops[stops.length - 1]
    if (before && stop.order === before.order) {
      // A date or time input takes one Tab per part (day, month, year) and its picker: one control, checked once.
      if (!stop.segmented) findings.push(`${where}: Tab stays on ${stop.name}`)
      continue
    }
    if (before && stop.order < before.order && !dialog) findings.push(`${where}: Tab goes back from ${before.name} to ${stop.name}`)
    if (!stop.visible) findings.push(`${where}: the focus is on ${stop.name}, which is not visible`)
    else if (stop.covered) findings.push(`${where}: the focus is on ${stop.name}, which ${stop.covered} covers`)
    if (stop.indicator) findings.push(`${where}: ${stop.name}: ${stop.indicator}`)
    stops.push(stop)
  }
  return stops.length
}

// focusStop describes the focused element; it runs in the page.
function focusStop(): Stop | null {
  const focused = document.activeElement as HTMLElement | null
  if (!focused || focused === document.body || focused === document.documentElement) return null
  const all = Array.from(document.querySelectorAll('*'))
  const text = (focused.getAttribute('aria-label') ?? focused.textContent ?? '').trim().replace(/\s+/g, ' ').slice(0, 40)
  const name = `<${focused.tagName.toLowerCase()}${focused.getAttribute('type') ? ` type=${focused.getAttribute('type')}` : ''}> "${text}"`
  const shown = (n: Element) => {
    const b = n.getBoundingClientRect()
    return b.width > 1 && b.height > 1 && getComputedStyle(n).visibility !== 'hidden'
  }
  // A visually hidden input pressed through its label (the chat's attach button) shows its focus on the label.
  const label = focused instanceof HTMLInputElement && !shown(focused) ? focused.labels?.[0] : undefined
  const el: Element = label ?? focused
  const r = el.getBoundingClientRect()
  const visible = shown(el)
  const own = (n: Element | null) => !!n && (n === el || el.contains(n) || n.contains(el) || (focused instanceof HTMLInputElement && !!focused.labels && Array.from(focused.labels).some((l) => l.contains(n))))
  // Covered: something else is on top at the middle and at each inner corner (a tooltip over a part is not).
  const points = [
    [0.5, 0.5],
    [0.15, 0.15],
    [0.85, 0.15],
    [0.15, 0.85],
    [0.85, 0.85],
  ].map(([fx, fy]) => document.elementFromPoint(Math.min(Math.max(r.left + r.width * fx, 0), innerWidth - 1), Math.min(Math.max(r.top + r.height * fy, 0), innerHeight - 1)))
  const top = points[0]
  // A chart's marks of two series at one value lie on each other, each with a transparent hit area: not covered.
  const covered = visible && !(el instanceof SVGElement) && !points.some(own) ? `<${top?.tagName.toLowerCase()} class="${top?.getAttribute('class') ?? ''}">` : ''

  const rgb = (c: string) => {
    const m = c.match(/rgba?\(([\d.]+),\s*([\d.]+),\s*([\d.]+)(?:,\s*([\d.]+))?/)
    return m ? { r: +m[1], g: +m[2], b: +m[3], a: m[4] === undefined ? 1 : +m[4] } : undefined
  }
  const lum = ({ r, g, b }: { r: number; g: number; b: number }) => {
    const ch = (v: number) => {
      const s = v / 255
      return s <= 0.03928 ? s / 12.92 : ((s + 0.055) / 1.055) ** 2.4
    }
    return 0.2126 * ch(r) + 0.7152 * ch(g) + 0.0722 * ch(b)
  }
  const behind = (from: Element | null) => {
    for (let n = from; n; n = n.parentElement) {
      const c = rgb(getComputedStyle(n).backgroundColor)
      if (c && c.a > 0.5) return c
    }
    return { r: 255, g: 255, b: 255, a: 1 }
  }
  const cs = getComputedStyle(el)
  let indicator = ''
  const outline = cs.outlineStyle !== 'none' && parseFloat(cs.outlineWidth) >= 1 ? rgb(cs.outlineColor) : undefined
  // A chart's mark shows its focus as a stroke around it (analytics.css).
  const stroked = el instanceof SVGElement && [el, ...Array.from(el.querySelectorAll('.hit'))].some((n) => getComputedStyle(n).stroke !== 'none' && parseFloat(getComputedStyle(n).strokeWidth) >= 2)
  if (stroked) {
    // the stroke is the indicator
  } else if (outline) {
    const a = lum(outline)
    const b = lum(behind(el.parentElement))
    const ratio = (Math.max(a, b) + 0.05) / (Math.min(a, b) + 0.05)
    if (ratio < 3) indicator = `the focus outline has a contrast of ${ratio.toFixed(2)}:1 against the page, below 3:1`
  } else if (cs.boxShadow === 'none') {
    indicator = 'no focus indicator (no outline, no box-shadow)'
  }
  const segmented = focused instanceof HTMLInputElement && ['date', 'time', 'datetime-local', 'month', 'week'].includes(focused.type)
  return { name, order: all.indexOf(focused), visible, covered, indicator, inDialog: !!focused.closest('[role=dialog], [role=alertdialog]'), segmented }
}

// targets checks every control a finger can press for a target of at least 44 × 44 px: a link inside a sentence is
// exempt (WCAG 2.5.5), and a checkbox or radio counts with its label, which presses it too.
export async function targets(page: Page, where: string) {
  const small = await page.evaluate(() => {
    const out: string[] = []
    // With a modal dialog open only its controls can be pressed.
    const modals = document.querySelectorAll('[aria-modal=true]')
    const root = modals.length > 0 ? modals[modals.length - 1] : document
    const controls = root.querySelectorAll<HTMLElement>('a[href], button, input:not([type=hidden]), select, textarea, summary, [role=button], [tabindex="0"]')
    for (const el of Array.from(controls)) {
      if ((el as HTMLButtonElement).disabled) continue
      // A chart's marks are read with a finger in its table ("Show as table"): WCAG 2.5.5's equivalent.
      if (el instanceof SVGElement) continue
      const cs = getComputedStyle(el)
      let r = el.getBoundingClientRect()
      if (r.width <= 1 || r.height <= 1 || cs.visibility === 'hidden') continue
      if (el.closest('[aria-hidden=true], [inert]')) continue
      if (el.tagName === 'A' && cs.display === 'inline') {
        const parent = el.parentElement
        const words = (parent?.textContent ?? '').trim().length - (el.textContent ?? '').trim().length
        if (words > 0) continue // a link in running text
      }
      if (el instanceof HTMLInputElement && (el.type === 'checkbox' || el.type === 'radio') && el.labels && el.labels.length > 0) {
        const l = el.labels[0].getBoundingClientRect()
        const left = Math.min(r.left, l.left)
        const top = Math.min(r.top, l.top)
        r = new DOMRect(left, top, Math.max(r.right, l.right) - left, Math.max(r.bottom, l.bottom) - top)
      }
      if (r.width < 43.5 || r.height < 43.5) {
        const text = (el.getAttribute('aria-label') ?? el.textContent ?? '').trim().replace(/\s+/g, ' ').slice(0, 40)
        out.push(`<${el.tagName.toLowerCase()}${el.getAttribute('type') ? ` type=${el.getAttribute('type')}` : ''}> "${text}" is ${Math.round(r.width)} × ${Math.round(r.height)} px`)
      }
    }
    return out
  })
  for (const s of new Set(small)) findings.push(`${where}: target below 44 px: ${s}`)
}

// dialog opens a dialog from its opener with the keyboard and checks it: the focus moves into it, the engine finds
// nothing, Tab stays inside and every stop shows its focus, and closing it (Escape, or the close button for a dialog
// in the page) gives the focus back to the opener.
export async function dialog(page: Page, where: string, opener: Locator, opened: Locator, close: (d: Locator) => Promise<void> = () => page.keyboard.press('Escape'), checks: { tab: boolean; touch: boolean } = { tab: true, touch: false }) {
  await opener.focus()
  await page.keyboard.press('Enter')
  await expect(opened).toBeVisible({ timeout: 10_000 })
  const inside = await opened.evaluate((d) => d.contains(document.activeElement))
  if (!inside) findings.push(`${where}: the focus stays outside the dialog when it opens`)
  await axe(page, where)
  if (checks.touch) await targets(page, where)
  // A modal keeps Tab inside; a question asked in place on the page (Confirm) lets it go on through the page.
  if (checks.tab) await tabWalk(page, where, { dialog: true, modal: (await opened.getAttribute('aria-modal')) === 'true' })
  await close(opened)
  await expect(opened).toBeHidden({ timeout: 5_000 })
  const back = await opener.evaluate((o) => o === document.activeElement)
  if (!back) findings.push(`${where}: the focus does not go back to the opener when the dialog closes`)
}

// report fails the spec with every finding, one per line.
export function report() {
  expect(findings, `accessibility findings:\n${findings.join('\n')}\n`).toEqual([])
}

// The header's menu button (readiness, after RD3), found by what it controls.
export const menuButton = (page: Page) => page.locator('header button[aria-controls]')
const menuPanel = async (page: Page) => page.locator(`[id="${await menuButton(page).getAttribute('aria-controls')}"]`)

// headerLinks lists the header navigation's links as they are shown now: name, address and whether it is the current page.
export async function headerLinks(page: Page): Promise<string[]> {
  return page.locator('header nav a:visible').evaluateAll((links) => links.map((a) => `${a.textContent?.trim()} ${a.getAttribute('href')}${a.getAttribute('aria-current') === 'page' ? ' (current)' : ''}`))
}

// wideHeader checks the header at 1280 px: no menu button, the links shown in the header as before. It returns them.
export async function wideHeader(page: Page, where: string): Promise<string[]> {
  if (await menuButton(page).isVisible()) findings.push(`${where}: the header shows the menu button on a wide screen`)
  const links = await headerLinks(page)
  if (links.length === 0) findings.push(`${where}: the header shows no link`)
  return links
}

// phoneMenu checks the header's menu on a phone, the disclosure pattern: the button is shown and says it is closed, the
// links are not; Space and Enter open it, Escape closes it and leaves the focus on the button; open, Tab goes from the
// button into the panel and through every link in it in order; the engine and the 44 px targets pass with it open (the
// header links are checked here now that a closed panel hides them); and its links are wide's, the same set and the same
// current page. It leaves the panel closed.
export async function phoneMenu(page: Page, where: string, wide: string[]) {
  const button = menuButton(page)
  if (!(await button.isVisible())) {
    findings.push(`${where}: no menu button in the header`)
    return
  }
  const panel = await menuPanel(page)
  const expanded = () => button.getAttribute('aria-expanded')
  if ((await expanded()) !== 'false' || (await panel.isVisible())) findings.push(`${where}: the menu is not closed when the page opens`)
  if ((await headerLinks(page)).length > 0) findings.push(`${where}: header links are shown with the menu closed`)
  for (const key of ['Space', 'Enter']) {
    await button.focus()
    await page.keyboard.press(key)
    if ((await expanded()) !== 'true' || !(await panel.isVisible())) findings.push(`${where}: ${key} on the menu button does not open the menu`)
    await page.keyboard.press('Escape')
    await expect(panel).toBeHidden({ timeout: 5_000 })
    if ((await expanded()) !== 'false') findings.push(`${where}: Escape does not close the menu`)
    if (!(await button.evaluate((b) => b === document.activeElement))) findings.push(`${where}: the focus does not go back to the menu button on Escape`)
  }
  await button.focus()
  await page.keyboard.press('Enter')
  await expect(panel).toBeVisible({ timeout: 5_000 })
  const links = await headerLinks(page)
  if (links.join('\n') !== wide.join('\n')) findings.push(`${where}: the menu's links are not the wide header's: ${links.join(', ')} against ${wide.join(', ')}`)
  const reached: string[] = []
  for (let i = 0; i < links.length; i++) {
    await page.keyboard.press('Tab')
    const at = await page.evaluate(() => {
      const a = document.activeElement
      return a instanceof HTMLAnchorElement && a.closest('header nav') ? a.getAttribute('href') : null
    })
    if (at === null) break
    reached.push(at)
  }
  const hrefs = links.map((l) => l.split(' ').find((w) => w.startsWith('/')) ?? '')
  if (reached.join(' ') !== hrefs.join(' ')) findings.push(`${where}: Tab from the menu button reaches ${reached.join(' ') || 'no link'} in the menu, not ${hrefs.join(' ')}`)
  await axe(page, `${where}, menu open`)
  await targets(page, `${where}, menu open`)
  await button.focus()
  await page.keyboard.press('Escape')
  await expect(panel).toBeHidden({ timeout: 5_000 })
}

// openMenu opens the header's menu on a phone, where the medical terms' button is, and does nothing on a wide screen.
export async function openMenu(page: Page) {
  const button = menuButton(page)
  if ((await button.isVisible()) && (await button.getAttribute('aria-expanded')) !== 'true') await button.click()
}
