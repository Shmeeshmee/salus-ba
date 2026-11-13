import { expect, test } from '@playwright/test'
import { required, signIn } from './signin'

// The practitioner's side of TestAnalytics_ThePractitionerSeesTheBookingOnTheirClinicsChart
// (test/e2e/analytics_web_test.go), which makes the synthetic clinic and a booking and passes the booking's month
// (YYYY-MM) in E2E_BOOKED_MONTH (SAL-22). The texts are the catalogs' (web/src/i18n/analytics/en.json); a month reads
// as the app formats it in English.
test.use({ locale: 'en' })

test("the practitioner sees the booking on My clinic's chart and the All of Salus tab", async ({ page }) => {
  await signIn(page)
  await page.getByRole('link', { name: 'Analytics' }).click()
  const main = page.getByRole('main')
  await expect(main.getByRole('heading', { name: 'Analytics', level: 1 })).toBeVisible()

  const month = new Intl.DateTimeFormat('en', { month: 'long', year: 'numeric', timeZone: 'UTC' }).format(new Date(`${required('E2E_BOOKED_MONTH')}-01T00:00:00Z`))
  const chart = main.getByRole('figure', { name: 'Appointments per month' })
  await expect(chart.getByRole('img', { name: `${month}, Booked: 1` })).toBeAttached()
  await expect(main.getByText('Only the diagnoses of your own patients are shown.')).toBeVisible()
  await expect(main.getByLabel('Specialty')).toHaveCount(0)
  await expect(main.getByLabel('Practitioner')).toHaveCount(0)
  await expect(main.getByRole('figure', { name: 'Top diagnosis groups' })).toBeVisible()

  await main.getByRole('tab', { name: 'All of Salus' }).click()
  await expect(main.getByRole('figure', { name: 'Patients per month' }).or(main.getByText('No month has been published yet.'))).toBeVisible()
  await expect(main.getByText('The figures could not be loaded. Try again.')).toHaveCount(0)
})
