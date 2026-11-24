import { expect, test, type Page } from '@playwright/test'
import { required, signIn } from './signin'

// The clinic's handling of data subject requests (docs/clinic-cms/WAVE5.md D4), as the clinic's second admin the Go
// test signs in (TestPrivacy_TheAdminHandlesARestrictionInTheWebApp): the queue with its due date and what the
// request needs, the detail, and the restriction, which takes the patient out of the desk's patient search (FX) until
// it is lifted. At 1280 and 390 px, nothing wider than the screen. E2E_WALK_DIR, when set, keeps a screenshot of each
// step there (the walk's record).
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
    await expect(row.getByText(/^Due /)).toBeVisible()
    await expect(row.getByText("Restrict the patient's record at the clinic.")).toBeVisible()
    await shot(page, width, '02-queue')
    await row.getByRole('link', { name: 'Open' }).click()
    await expect(page.getByRole('heading', { name: /Restriction/ })).toBeVisible()
    const detail = page.url()
    await shot(page, width, '03-detail')
    await page.getByRole('button', { name: 'Restrict the record' }).click()
    await expect(page.getByRole('status')).toHaveText('The record is restricted.')
    await shot(page, width, '04-restricted')

    await page.goto('/desk')
    const after = await patients(page, name)
    await expect(after.getByRole('button', { name })).toHaveCount(0)
    await shot(page, width, '05-desk-search-after')
    await after.getByRole('button', { name: 'Close' }).click()

    await page.goto(detail)
    await page.getByRole('button', { name: 'Lift the restriction' }).click()
    await expect(page.getByRole('status')).toBeVisible()
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

// shot checks that nothing is wider than the screen, and keeps a screenshot when E2E_WALK_DIR is set.
async function shot(page: Page, width: number, name: string) {
  expect(await page.evaluate(() => document.documentElement.scrollWidth)).toBe(width)
  const dir = process.env.E2E_WALK_DIR
  if (dir) await page.screenshot({ path: `${dir}/${width}-${name}.png`, animations: 'disabled' })
}
