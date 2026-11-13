import { expect, test } from '@playwright/test'
import { required, signIn } from './signin'

// The practitioner's side of TestAllergies_ThePractitionerSeesThePatientsAllergyOnTheCallPage
// (test/e2e/allergies_web_test.go), which makes the synthetic clinic, patient and appointment, adds the patient's
// allergy over the API and passes it in E2E_* variables (SAL-41). The warning is on the call page from the device
// check on, before anyone joins. The texts are the catalogs' (web/src/i18n/allergies/en.json).
test.use({ locale: 'en' })

test("the practitioner sees the patient's allergy on the call page", async ({ page }) => {
  await signIn(page)

  await page.goto(`/appointments/${required('E2E_APPOINTMENT_ID')}/call`)
  await expect(page.getByText('Check your camera and microphone')).toBeVisible()
  const warning = page.getByRole('region', { name: 'Active allergies' })
  const item = warning.getByRole('listitem').filter({ hasText: required('E2E_ALLERGY_TEXT') })
  await expect(item).toBeVisible()
  await expect(item).toContainText(`${required('E2E_ALLERGY_REACTION')} (severe)`)
})
