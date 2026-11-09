import { expect, test } from '@playwright/test'
import { required, signIn } from './signin'

// The cheatsheet's side of TestTerminology_APractitionerFindsATermAndAPatientIsRefused (test/e2e/terminology_test.go),
// which adds the synthetic term and passes its labels in E2E_TERM_* variables (docs/terminology/SPEC.md §3 "The
// cheatsheet"). Ctrl+K on the calendar opens the palette; the Latin label finds the term with its English label and
// "Not reviewed yet"; Enter copies the UI language's (English) label; Escape closes; the header button opens it on
// the questionnaires screen. The texts are the catalogs' (web/src/i18n/terminology/en.json).
test.use({ locale: 'en', permissions: ['clipboard-read', 'clipboard-write'] })

test('a practitioner finds a term in the cheatsheet and copies it', async ({ page }) => {
  await signIn(page)
  const open = page.getByRole('button', { name: 'Search medical terms (Ctrl+K)' })
  await expect(open).toBeVisible() // the shell is shown once the account is read
  await page.keyboard.press('Control+K')
  const palette = page.getByRole('dialog', { name: 'Medical terms' })
  await expect(palette).toBeVisible()

  await palette.getByRole('combobox').fill(required('E2E_TERM_LATIN'))
  const term = palette.getByRole('option').first()
  await expect(term).toContainText(required('E2E_TERM_LATIN'))
  await expect(term).toContainText(required('E2E_TERM_ENGLISH'))
  await expect(term).toContainText('Not reviewed yet')

  await palette.getByRole('combobox').press('Enter')
  await expect(palette.getByRole('status')).toHaveText('Copied')
  expect(await page.evaluate(() => navigator.clipboard.readText())).toBe(required('E2E_TERM_ENGLISH'))

  await page.keyboard.press('Escape')
  await expect(palette).toBeHidden()

  await page.getByRole('link', { name: 'Questionnaires' }).click()
  await open.click()
  await expect(palette).toBeVisible()
})
