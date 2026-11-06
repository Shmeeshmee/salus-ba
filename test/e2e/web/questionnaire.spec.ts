import { expect, test, type Locator, type Page } from '@playwright/test'
import { required, signIn } from './signin'

// The practitioner's side of TestQuestionnaire_TheBuilderPublishesAQuestionnaire (test/e2e/questionnaire_test.go),
// which makes the synthetic clinic and passes everything in E2E_* variables. Three items are dragged in from the
// palette with the mouse, one is moved by a mouse drag, one by the keyboard and one with "Move up"; the questionnaire
// is published and found in the library. The texts are the approved ones (docs/questionnaire/COPY.md); the
// questionnaire's own wording is a placeholder.
test('the practitioner builds a questionnaire by drag and drop, publishes it and finds it in the library', async ({ page }) => {
  const title = required('E2E_QUESTIONNAIRE_TITLE')
  await signIn(page)

  await page.getByRole('link', { name: 'Questionnaires' }).click() // W1
  await page.getByRole('button', { name: 'New questionnaire' }).click() // W2
  await expect(page.getByRole('heading', { name: 'Untitled questionnaire' })).toBeVisible() // W8
  await page.getByLabel('Title').fill(title) // W9

  // Each type is dragged from the palette (W11) to the end of the list, named, and its card closed.
  for (const [kind, text] of [
    ['Yes or no', 'Placeholder question A'],
    ['Short answer', 'Placeholder question B'],
    ['Date', 'Placeholder question C'],
  ]) {
    await drag(page, page.getByRole('button', { name: kind, exact: true }), page.locator('.list-end'))
    const fresh = rows(page).last()
    await fresh.getByLabel('Question', { exact: true }).fill(text) // W13
    await fresh.getByRole('button', { name: text, exact: true }).click()
  }
  await expect(texts(page)).toHaveText(['Placeholder question A', 'Placeholder question B', 'Placeholder question C'])

  // C dragged by its handle (W18) onto A's place.
  await drag(page, handle(page, 'Placeholder question C'), handle(page, 'Placeholder question A'))
  await expect(texts(page)).toHaveText(['Placeholder question C', 'Placeholder question A', 'Placeholder question B'])
  await expect(announced(page, 'Placeholder question C moved to position 1 of 3.')).toBeAttached() // W42

  // B moved up one place with the keyboard: space, arrow up, space (W41).
  await handle(page, 'Placeholder question B').focus()
  await page.keyboard.press('Space')
  await expect(announced(page, 'Moving Placeholder question B.')).toBeAttached() // W42
  await page.keyboard.press('ArrowUp')
  await expect(announced(page, 'Placeholder question B is at position 2 of 3.')).toBeAttached() // W42, then dropped
  await page.keyboard.press('Space')
  await expect(texts(page)).toHaveText(['Placeholder question C', 'Placeholder question B', 'Placeholder question A'])
  await expect(announced(page, 'Placeholder question B moved to position 2 of 3.')).toBeAttached()

  // A moved up with its button (W18).
  await row(page, 'Placeholder question A').getByRole('button', { name: 'Move up' }).click()
  await expect(texts(page)).toHaveText(['Placeholder question C', 'Placeholder question A', 'Placeholder question B'])
  await expect(announced(page, 'Placeholder question A moved to position 2 of 3.')).toBeAttached()

  await page.getByRole('button', { name: 'Publish', exact: true }).click() // W22
  const confirm = page.getByRole('group', { name: 'Publish this questionnaire?' })
  await expect(confirm.getByText('Once published, it cannot be changed. To change it later, you create a new version.')).toBeVisible()
  await confirm.getByRole('button', { name: 'Publish' }).click()
  await expect(page.getByText('This questionnaire is published and cannot be changed.')).toBeVisible() // W23

  await page.getByRole('link', { name: 'Questionnaires' }).first().click()
  const library = page.getByRole('list', { name: 'Questionnaires' })
  const entry = library.getByRole('listitem').filter({ has: page.getByRole('link', { name: title }) })
  await expect(entry.getByText('Active', { exact: true })).toBeVisible() // W4
})

// drag moves the mouse from the middle of from to the middle of to in small steps, as a hand does: @dnd-kit starts
// a drag after 5 px and finds the drop place on each move.
async function drag(page: Page, from: Locator, to: Locator) {
  const a = await box(from)
  const b = await box(to)
  await page.mouse.move(a.x, a.y)
  await page.mouse.down()
  await page.mouse.move(a.x, a.y + 10, { steps: 5 })
  await page.mouse.move(b.x, b.y, { steps: 20 })
  await page.mouse.up()
}

async function box(l: Locator): Promise<{ x: number; y: number }> {
  await l.scrollIntoViewIfNeeded()
  const r = await l.boundingBox()
  if (!r) throw new Error('the element to drag is not on the page')
  return { x: r.x + r.width / 2, y: r.y + r.height / 2 }
}

const rows = (page: Page) => page.locator('ul.items > li.item')
const texts = (page: Page) => rows(page).locator('.item-text')
const row = (page: Page, text: string) => rows(page).filter({ has: page.getByRole('button', { name: text, exact: true }) })
const handle = (page: Page, text: string) => row(page, text).getByRole('button', { name: 'Drag to reorder' })

// announced is a W42 announcement in a live region: the builder's own, or the one @dnd-kit renders.
const announced = (page: Page, text: string) => page.locator('[aria-live]').getByText(text, { exact: true })
