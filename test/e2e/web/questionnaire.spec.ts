import { expect, test } from '@playwright/test'
import { announced, drag, handle, row, rows, texts, toggle } from './builder'
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
    // The drop lands only when the drop place was measured on a settled list: each drag waits for the card before it
    // to be closed, and is checked to have added its row before the row is filled in (CI flake, PR #25).
    const before = await rows(page).count()
    await drag(page, page.getByRole('button', { name: kind, exact: true }), page.locator('.list-end'))
    await expect(rows(page)).toHaveCount(before + 1)
    const fresh = rows(page).last()
    await fresh.getByLabel('Question', { exact: true }).fill(text) // W13
    await toggle(fresh, text).click()
    await expect(fresh.getByLabel('Question', { exact: true })).toBeHidden()
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
  // With no translation, the builder first asks whether to add one (i18n COPY.md B5).
  await page.getByRole('alertdialog', { name: 'Add a translation before publishing?' }).getByRole('button', { name: 'Publish without' }).click()
  const confirm = page.getByRole('alertdialog', { name: 'Publish this questionnaire?' })
  await expect(confirm.getByText('Once published, it cannot be changed. To change it later, you create a new version.')).toBeVisible()
  await confirm.getByRole('button', { name: 'Publish' }).click()
  await expect(page.getByText('This questionnaire is published and cannot be changed.')).toBeVisible() // W23

  await page.getByRole('link', { name: 'Questionnaires' }).first().click()
  const library = page.getByRole('list', { name: 'Questionnaires' })
  const entry = library.getByRole('listitem').filter({ has: page.getByRole('link', { name: title }) })
  await expect(entry.getByText('Active', { exact: true })).toBeVisible() // W4
})
