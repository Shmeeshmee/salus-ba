import { expect, test, type Browser, type Page } from '@playwright/test'
import { signIn } from './signin'

// The staff's side of TestLanguage_TheWebAppFollowsTheBrowserThenTheSavedChoice (test/e2e/language_test.go), which
// makes the synthetic clinic and passes its admin in E2E_* variables (docs/i18n/SPEC.md §6, steps 1 to 3). A browser
// in Serbian Cyrillic shows the sign-in screen in Serbian Cyrillic. The admin signs in from a browser in English and
// picks Hrvatski; signed in again from another browser in English, the app is in Croatian; "Use the browser's
// language" brings English back. The texts are the catalogs' (web/src/i18n).
test('the web app follows the browser, then the language the user saves', async ({ browser }) => {
  const cyrillic = await open(browser, 'sr-Cyrl')
  await cyrillic.goto('/sign-in')
  await expect(cyrillic.getByRole('heading', { name: 'Пријава' })).toBeVisible()
  await expect(cyrillic.getByLabel('Е-пошта')).toBeVisible()
  await expect(cyrillic.getByRole('button', { name: 'Пријави се' })).toBeVisible()
  await expect(cyrillic.locator('html')).toHaveAttribute('lang', 'sr-Cyrl')

  const first = await open(browser, 'en')
  await signIn(first)
  await expect(first.getByRole('link', { name: 'Questionnaires' })).toBeVisible()
  await first.getByLabel('Language').selectOption('hr')
  await expect(first.getByText('Jezik je spremljen.')).toBeVisible() // L4
  await expect(first.getByRole('link', { name: 'Upitnici' })).toBeVisible()
  await first.getByRole('button', { name: 'Odjavi se' }).click()
  await expect(first).toHaveURL(/\/sign-in$/)

  const second = await open(browser, 'en')
  await signIn(second)
  await expect(second.getByRole('link', { name: 'Upitnici' })).toBeVisible()
  await expect(second.locator('html')).toHaveAttribute('lang', 'hr')
  await second.getByLabel('Jezik').selectOption('') // L3, "Koristi jezik preglednika"
  await expect(second.getByText('Language saved.')).toBeVisible()
  await expect(second.getByRole('link', { name: 'Questionnaires' })).toBeVisible()
})

// open is a page in a browser of its own (no storage shared with the others) whose language is locale.
async function open(browser: Browser, locale: string): Promise<Page> {
  const context = await browser.newContext({ locale })
  return context.newPage()
}
