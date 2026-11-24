import { expect, test } from '@playwright/test'
import { required, signIn } from './signin'

// The front desk's row after a receipt (docs/clinic-cms/walk/wave3, findings 1 and 2), as the receptionist the Go test
// signs in (TestDesk_…): the visit shows its receipt and opens its PDF, and a second receipt is asked for first.
test("the desk row shows the visit's receipt and its PDF, and asks before a second one", async ({ page }) => {
  const patient = required('E2E_DESK_PATIENT')
  const number = required('E2E_DESK_RECEIPT')
  await signIn(page)
  await expect(page).toHaveURL(/\/desk$/)
  const row = page.locator('li').filter({ hasText: patient })
  await expect(row.getByText(`Receipt no. ${number}`, { exact: true })).toBeVisible()

  const href = await row.getByRole('link', { name: 'Open PDF' }).getAttribute('href')
  const pdf = await page.request.get(href!)
  expect(pdf.status()).toBe(200)
  expect(pdf.headers()['content-type']).toContain('application/pdf')

  await row.getByRole('button', { name: 'Issue another receipt' }).click()
  const dialog = page.getByRole('dialog', { name: 'New receipt' })
  await expect(dialog.getByText(`This visit already has receipt no. ${number}. Issue another receipt?`)).toBeVisible()
  await expect(dialog.getByLabel('Service')).toHaveCount(0)
  await dialog.getByRole('button', { name: 'Cancel' }).click()
  await expect(dialog).toBeHidden()
})
