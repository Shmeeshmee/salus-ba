import { readFileSync } from 'node:fs'
import { expect, test, type Locator, type Page } from '@playwright/test'
import { required, signIn } from './signin'

// The clinic's handling of data subject requests (docs/clinic-cms/WAVE5.md D4), as the clinic's second admin the Go
// test signs in (TestPrivacy_TheAdminHandlesARestrictionInTheWebApp): the queue naming the patient with their birth
// date, the request's due date and what it needs; the detail with the restriction's state and only the action that
// applies; the restriction, which takes the patient out of the desk's patient search (FX, S7) until it is lifted; and
// the hand-over downloaded from the detail, the admin's way to the restricted record (wave 5 walk, findings 3 to 5).
// At 1280 and 390 px, nothing wider than the screen. E2E_WALK_DIR, when set, keeps a screenshot of each step there
// (the walk's record).
for (const width of [1280, 390]) {
  test(`the admin opens a restriction request, restricts the record and the desk search leaves the patient out at ${width} px`, async ({ page }) => {
    await page.setViewportSize({ width, height: width > 400 ? 900 : 844 })
    const name = required('E2E_PRIVACY_PATIENT')
    await signIn(page)

    await page.goto('/desk')
    const search = await patients(page, name)
    const found = search.getByRole('list', { name: 'Patients found' })
    await expect(found.getByRole('button', { name })).toBeVisible()
    await found.getByRole('button', { name }).click()
    const card = page.getByRole('dialog', { name: 'Patient card' })
    await expect(card).toBeVisible()
    await shot(page, width, '01-desk-card-before')
    await card.getByRole('button', { name: 'Close' }).click()
    await search.getByRole('button', { name: 'Close' }).click()

    await page.goto('/data-requests')
    const queue = page.getByRole('list', { name: 'Data requests' })
    const row = queue.locator('li').filter({ hasText: 'Restriction' })
    await expect(row.getByText(name, { exact: true })).toBeVisible()
    await expect(row.getByText('Born 9 August 1975')).toBeVisible()
    await expect(row.getByText(/Restriction · Due /)).toBeVisible()
    await expect(row.getByText("Restrict the patient's record at the clinic.")).toBeVisible()
    await shot(page, width, '02-queue')
    await row.getByRole('link', { name: 'Open' }).click()
    await expect(page.getByRole('heading', { name: /Restriction/ })).toBeVisible()
    await expect(page.getByText(name, { exact: true })).toBeVisible()
    await expect(page.getByText('The record is not restricted.')).toBeVisible()
    await expect(page.getByRole('button', { name: 'Lift the restriction' })).toHaveCount(0)
    const detail = page.url()
    await shot(page, width, '03-detail')
    await page.getByRole('button', { name: 'Restrict the record' }).click()
    await expect(page.getByRole('status').filter({ hasText: 'The record is restricted.' })).toBeVisible()
    await expect(page.getByText(/^Restricted by the clinic since /)).toBeVisible()
    await expect(page.getByRole('button', { name: 'Restrict the record' })).toHaveCount(0)
    await expect(page.getByRole('button', { name: 'Lift the restriction' })).toBeVisible()
    await shot(page, width, '04-restricted')
    // The restricted record still opens for the admin, from the request: its hand-over downloads.
    expect(await handOver(page, page.getByRole('link', { name: 'Open the hand-over' }))).toContain(name.split(' ')[1])

    await page.goto('/desk')
    const after = await patients(page, name)
    await expect(after.getByText('No patient found.')).toBeVisible() // the search answered: the patient is left out
    await shot(page, width, '05-desk-search-after')
    await after.getByRole('button', { name: 'Close' }).click()

    await page.goto(detail)
    await page.getByRole('button', { name: 'Lift the restriction' }).click()
    await expect(page.getByRole('status').filter({ hasText: 'The restriction is lifted.' })).toBeVisible()
    await expect(page.getByText('The record is not restricted.')).toBeVisible()
    await expect(page.getByRole('button', { name: 'Restrict the record' })).toBeVisible()
    await shot(page, width, '06-lifted')
  })
}

// patients opens the desk's patient search and searches for the family name.
async function patients(page: Page, name: string) {
  await page.getByRole('button', { name: 'Patients', exact: true }).click()
  const search = page.getByRole('dialog', { name: 'Patients' })
  await search.getByLabel('Name').fill(name.split(' ')[1])
  await search.getByRole('button', { name: 'Search' }).click()
  return search
}

// handOver downloads the hand-over the link offers and returns its text.
async function handOver(page: Page, link: Locator): Promise<string> {
  const [download] = await Promise.all([page.waitForEvent('download'), link.click()])
  const path = await download.path()
  return readFileSync(path, 'utf8')
}

// shot checks that nothing is wider than the screen, and keeps a screenshot when E2E_WALK_DIR is set.
async function shot(page: Page, width: number, name: string) {
  expect(await page.evaluate(() => document.documentElement.scrollWidth)).toBe(width)
  const dir = process.env.E2E_WALK_DIR
  if (dir) await page.screenshot({ path: `${dir}/${width}-${name}.png`, animations: 'disabled' })
}
