import { expect, test } from '@playwright/test'
import { signIn } from './signin'

// The admin's side of TestClinic_SignsItselfUpAndIsListedOnceApproved (test/e2e/signup_test.go, docs/clinic-cms O1):
// the admin of a clinic that signed itself up signs in, finds the setup banner above the screen and opens the setup
// wizard from it. The Go test passes the admin's e-mail in E2E_PRACTITIONER_EMAIL, as signIn reads it. The texts are
// the catalog's (web/src/i18n/signup/en.json).
test.use({ locale: 'en' })

test('the signed-up admin finds the setup banner and opens the wizard', async ({ page }) => {
  await signIn(page)
  const banner = page.getByRole('status').filter({ hasText: "Your clinic's setup is not finished." })
  await expect(banner).toBeVisible()
  await banner.getByRole('link', { name: 'Continue setup' }).click()
  await expect(page).toHaveURL(/\/setup$/)
  await expect(page.getByRole('heading', { name: 'Set up your clinic' })).toBeVisible()
})
