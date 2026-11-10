import { expect, test } from '@playwright/test'
import { required, signIn } from './signin'

// The review panel's side of the term review journey (test/e2e/terminology_test.go), which adds a synthetic term with
// a rank above every seed term's (so it is the first unreviewed entry of the Croatian queue), makes the practitioner
// a reviewer and passes the texts in E2E_* variables (docs/terminology/SPEC.md §3 "Review"). With the keyboard only:
// the Croatian label is edited, the labels approved, an explanation written and sent back for changes with a note,
// and a synonym added to the approved labels, which puts them back to unreviewed. The Go test then checks the
// stored review through the API. The texts are the catalogs' (web/src/i18n/review/en.json).
test.use({ locale: 'en' })

test('a reviewer edits, approves and requests changes with the keyboard', async ({ page }) => {
  await signIn(page)
  await page.getByRole('link', { name: 'Term review' }).click()
  const main = page.getByRole('main')
  await expect(main.getByRole('heading', { name: 'Term review' })).toBeVisible()
  await main.getByLabel('Language').selectOption('hr')
  await expect(main.getByText(/approved in Croatian/)).toBeVisible()

  const entry = main.getByRole('article').first()
  await expect(entry).toContainText(required('E2E_TERM_LATIN'))
  await expect(entry).toContainText(required('E2E_TERM_CROATIAN'))
  const labels = entry.getByRole('region', { name: 'Labels' })
  const explanation = entry.getByRole('region', { name: 'Explanation for patients (informational)' })
  await expect(explanation).toContainText('No explanation yet')

  // e edits the labels; Enter saves.
  await entry.focus()
  await page.keyboard.press('e')
  const preferred = entry.getByLabel('Preferred label')
  await expect(preferred).toBeFocused()
  await preferred.fill(required('E2E_TERM_EDITED'))
  await page.keyboard.press('Enter')
  await expect(entry.getByRole('heading', { level: 2 })).toHaveText(required('E2E_TERM_EDITED'))
  await expect(labels).toContainText('Unreviewed')
  await expect(entry).toBeFocused()

  // a approves them; the focus moves on to the next entry.
  await page.keyboard.press('a')
  await expect(labels).toContainText('Approved')

  // Shift+E writes the explanation; Shift+N sends it back with a note.
  await entry.focus()
  await page.keyboard.press('Shift+E')
  const text = entry.getByLabel('Explanation text')
  await expect(text).toBeFocused()
  await text.fill(required('E2E_EXPLANATION'))
  await page.keyboard.press('Enter')
  await expect(explanation).toContainText(required('E2E_EXPLANATION'))
  await expect(explanation).toContainText('Unreviewed')
  await expect(entry).toBeFocused()
  await page.keyboard.press('Shift+N')
  const note = entry.getByLabel('What needs to change')
  await expect(note).toBeFocused()
  await note.fill('Placeholder note')
  await page.keyboard.press('Enter')
  await expect(explanation).toContainText('Needs changes')
  await expect(explanation).toContainText('Note: Placeholder note')

  // Editing the approved labels returns them to the queue.
  await expect(entry).toBeFocused()
  await page.keyboard.press('e')
  const synonyms = entry.getByLabel('Synonyms, separated by |')
  await synonyms.fill('Sinonim e2e')
  await page.keyboard.press('Enter')
  await expect(labels).toContainText('Sinonim e2e')
  await expect(labels).toContainText('Unreviewed')
})
