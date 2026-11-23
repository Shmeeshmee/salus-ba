import type { Locator, Page } from '@playwright/test'

// The builder's rows and the drag of the web journeys that build a questionnaire (questionnaire.spec.ts,
// branching.spec.ts).

// drag moves the mouse from the middle of from to the middle of to in small steps, as a hand does: @dnd-kit starts
// a drag after 5 px and finds the drop place on each move.
export async function drag(page: Page, from: Locator, to: Locator) {
  const a = await box(from)
  const b = await box(to)
  await page.mouse.move(a.x, a.y)
  await page.mouse.down()
  await page.mouse.move(a.x, a.y + 10, { steps: 5 })
  await page.mouse.move(b.x, b.y, { steps: 20 })
  await page.mouse.up()
  await dropGuardLifted(page)
}

// dropGuardLifted waits until a click reaches the app again. A drop does not end dnd-kit's click guard at once: from
// the start of a drag until 50 ms after the drop, a click anywhere on the page is stopped in the capture phase, so the
// click that ends a drag does not also press what is under the pointer (@dnd-kit/core 6.3.1,
// AbstractPointerSensor.detach removes the guard on a setTimeout of 50 ms). A click inside those 50 ms is lost without
// an error (CI flake: the closed card stayed open). A timer of the same delay set after the drop runs after dnd-kit's,
// since timers of equal delay run in the order they were set (HTML, timer initialisation steps), so when it fires the
// guard is gone. Update dndGuardMs with dnd-kit.
const dndGuardMs = 50
async function dropGuardLifted(page: Page) {
  await page.evaluate((ms) => new Promise<void>((done) => setTimeout(done, ms)), dndGuardMs)
}

async function box(l: Locator): Promise<{ x: number; y: number }> {
  await l.scrollIntoViewIfNeeded()
  const r = await l.boundingBox()
  if (!r) throw new Error('the element to drag is not on the page')
  return { x: r.x + r.width / 2, y: r.y + r.height / 2 }
}

export const rows = (page: Page) => page.locator('ul.items > li.item')
export const texts = (page: Page) => rows(page).locator('.item-text')
// A card's toggle is named by what it shows (the text, the type, Required): it is found by its text.
export const itemText = (page: Page, text: string) => page.locator('.item-text').getByText(text, { exact: true })
export const toggle = (card: Locator, text: string) => card.locator('button.item-toggle').filter({ has: itemText(card.page(), text) })
export const row = (page: Page, text: string) => rows(page).filter({ has: itemText(page, text) })
export const handle = (page: Page, text: string) => row(page, text).getByRole('button', { name: 'Drag to reorder' })

// announced is a W42 announcement in a live region: the builder's own, or the one @dnd-kit renders.
export const announced = (page: Page, text: string) => page.locator('[aria-live]').getByText(text, { exact: true })
