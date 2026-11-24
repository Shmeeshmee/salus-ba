import { expect, test, type Page } from '@playwright/test'
import { required, signIn } from './signin'

// The desk fixes of docs/clinic-cms/WAVE5.md FX, as the clinic's second admin the Go test signs in
// (TestDesk_TheClinicActsOnTheRightsOfAPatientItRegistered): the desk's patient search opens the card of a patient whose
// consent was withdrawn and who has no visit today (wave 4 walk, finding 5); the move dialog picks a day, then a time
// (finding 7); the clinic settings page leads to each section (finding 8). At 1280 and 390 px, nothing wider than the
// screen. E2E_WALK_DIR, when set, keeps a screenshot of each step there (the walk's record); unset, nothing is written.
for (const width of [1280, 390]) {
  test(`the desk finds a withdrawn patient, moves a visit by day and the settings lead to each section at ${width} px`, async ({ page }) => {
    await page.setViewportSize({ width, height: width > 400 ? 900 : 844 })
    const withdrawn = required('E2E_RIGHTS_PATIENT')
    const booked = required('E2E_RIGHTS_BOOKED')
    await signIn(page)
    await page.goto('/desk')

    await page.getByRole('button', { name: 'Patients', exact: true }).click()
    const search = page.getByRole('dialog', { name: 'Patients' })
    await search.getByLabel('Name').fill(withdrawn.split(' ')[0])
    await search.getByRole('button', { name: 'Search' }).click()
    const found = search.getByRole('list', { name: 'Patients found' })
    await expect(found.getByText('5 June 1970')).toBeVisible()
    await shot(page, width, '01-desk-patients-search')
    await found.getByRole('button', { name: withdrawn }).click()
    const card = page.getByRole('dialog', { name: 'Patient card' })
    await expect(card.getByRole('button', { name: 'Record consent' })).toBeVisible()
    await expect(card.getByRole('link', { name: 'Download the record' })).toBeVisible()
    await shot(page, width, '02-withdrawn-patient-card')
    await card.getByRole('button', { name: 'Close' }).click()
    await search.getByRole('button', { name: 'Close' }).click()

    const row = page.locator('li').filter({ hasText: booked })
    await expect(row.getByText(/\+387 61 000 \d{3}/)).toBeVisible()
    await row.getByRole('button', { name: 'Move' }).click()
    const move = page.getByRole('dialog', { name: 'Move visit' })
    const day = move.getByLabel('Day')
    await expect(day).toBeVisible()
    expect(await day.locator('option').count()).toBeGreaterThan(1)
    expect(await move.getByLabel('New time').locator('option').count()).toBeLessThan(60) // one day's times, not 15 days'
    await shot(page, width, '03-move-day-and-time')
    await move.getByRole('button', { name: 'Cancel' }).click()

    await page.goto('/clinic')
    const nav = page.getByRole('navigation', { name: 'On this page' })
    await expect(nav.getByRole('link')).toHaveCount(8)
    await shot(page, width, '04-clinic-settings-nav')
    await nav.getByRole('link', { name: 'Profile for patients abroad' }).click()
    await expect(page.getByRole('heading', { name: 'Profile for patients abroad' })).toBeInViewport()
    await shot(page, width, '05-clinic-settings-profile-section')
  })
}

// shot checks that nothing is wider than the screen, and keeps a screenshot when E2E_WALK_DIR is set.
async function shot(page: Page, width: number, name: string) {
  expect(await page.evaluate(() => document.documentElement.scrollWidth)).toBe(width)
  const dir = process.env.E2E_WALK_DIR
  if (dir) await page.screenshot({ path: `${dir}/${width}-${name}.png`, animations: 'disabled' })
}
