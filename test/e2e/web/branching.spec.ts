import { expect, test, type Locator, type Page } from '@playwright/test'
import { drag, row, rows, texts, toggle } from './builder'
import { required, signIn } from './signin'

// The practitioner's side of TestQuestionnaire_TheBuilderBranchesTheGynaecologyExample (test/e2e/questionnaire_web_test.go):
// SAL-43's example built in the builder with "Ask a question if the answer is Yes" on the yes or no question (a month
// and an age, as whole numbers) and a main question after the branch, where the branches join; the preview takes the
// yes path, warns when a change to no would discard the month, takes the no path, and the questionnaire is published.
// The questions' wording is a placeholder.
const had = 'Placeholder: have you had an abortion?'
const month = 'Placeholder: in which month of pregnancy?'
const age = 'Placeholder: how old were you?'
const main = 'Placeholder main question'

test('the practitioner builds the branching example, previews both paths and publishes it', async ({ page }) => {
  const title = required('E2E_QUESTIONNAIRE_TITLE')
  await signIn(page)
  await page.getByRole('link', { name: 'Questionnaires' }).click()
  await page.getByRole('button', { name: 'New questionnaire' }).click()
  await page.getByLabel('Title').fill(title)

  await add(page, 'Yes or no', had)
  for (const text of [month, age]) {
    const question = row(page, had)
    if (await question.getByLabel('Question', { exact: true }).isHidden()) await toggle(question, had).click()
    const before = await rows(page).count()
    await question.getByRole('button', { name: 'Ask a question if the answer is Yes' }).click()
    await expect(rows(page)).toHaveCount(before + 1)
    const asked = rows(page).nth(before) // at the end of the yes branch, which is the end of the list so far
    await asked.getByLabel('Question', { exact: true }).fill(text)
    await asked.getByLabel('Type').selectOption({ label: 'Whole number' })
  }
  await add(page, 'Short answer', main)

  await expect(texts(page)).toHaveText([had, month, age, main])
  await expect(row(page, month).getByText('Conditional')).toBeVisible()
  await expect(row(page, main).getByText('Branches join here')).toBeVisible()
  await expect(row(page, main).getByText('Conditional')).toHaveCount(0)

  await page.getByRole('button', { name: 'Preview', exact: true }).click()
  const preview = page.locator('.q-preview')
  await expect.poll(() => asked(preview)).toEqual([had, main])
  await preview.getByLabel('Yes', { exact: true }).check()
  await expect.poll(() => asked(preview)).toEqual([had, month, age, main])
  await preview.getByLabel(month).fill('3')

  // The changed answer would discard the month: asked first, kept, then changed.
  await preview.getByLabel('No', { exact: true }).click()
  const confirm = page.getByRole('alertdialog', { name: 'Change this answer?' })
  await expect(confirm.getByText('1 answer already given will be discarded, because its question will no longer be asked.')).toBeVisible()
  await confirm.getByRole('button', { name: 'Keep answer' }).click()
  await expect(preview.getByLabel(month)).toHaveValue('3')
  await preview.getByLabel('No', { exact: true }).click()
  await page.getByRole('alertdialog', { name: 'Change this answer?' }).getByRole('button', { name: 'Change answer' }).click()
  await expect.poll(() => asked(preview)).toEqual([had, main])
  await preview.getByLabel('Yes', { exact: true }).check()
  await expect(preview.getByLabel(month)).toHaveValue('')
  await page.getByRole('button', { name: 'Back to editing' }).click()

  await page.getByRole('button', { name: 'Publish', exact: true }).click()
  await page.getByRole('alertdialog', { name: 'Add a translation before publishing?' }).getByRole('button', { name: 'Publish without' }).click()
  await page.getByRole('alertdialog', { name: 'Publish this questionnaire?' }).getByRole('button', { name: 'Publish' }).click()
  await expect(page.getByText('This questionnaire is published and cannot be changed.')).toBeVisible()
})

// add drags a type from the palette to the end of the list and names it; its card stays open.
async function add(page: Page, kind: string, text: string) {
  const before = await rows(page).count()
  await drag(page, page.getByRole('button', { name: kind, exact: true }), page.locator('.list-end'))
  await expect(rows(page)).toHaveCount(before + 1)
  await rows(page).last().getByLabel('Question', { exact: true }).fill(text)
}

// asked is the questions the preview asks now, in order: each item's legend or label (the unit tests' reading).
const asked = (preview: Locator) =>
  preview.locator('.preview-items > li').evaluateAll((items) => items.map((li) => li.querySelector('legend, label')?.textContent))
