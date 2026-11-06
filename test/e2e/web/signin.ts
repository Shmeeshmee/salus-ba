import { createHmac } from 'node:crypto'
import { expect, type Page } from '@playwright/test'

// signIn signs the practitioner in to the web app with the password and a TOTP code, as the Go test that starts
// the spec passes them (E2E_PRACTITIONER_EMAIL, E2E_PASSWORD, E2E_TOTP_SECRET, E2E_TOTP_LAST_STEP).
export async function signIn(page: Page) {
  await page.goto('/sign-in')
  await page.getByLabel('E-mail').fill(required('E2E_PRACTITIONER_EMAIL'))
  await page.getByLabel('Password').fill(required('E2E_PASSWORD'))
  await page.getByRole('button', { name: 'Sign in' }).click()
  await page.getByLabel('Code').fill(await totpCode())
  await page.getByRole('button', { name: 'Continue' }).click()
  await expect(page).toHaveURL(/:5173\/$/)
}

// totpCode is the authenticator's code (RFC 6238: SHA-1, 6 digits, 30 s) of a step the API has not seen used: the
// current one, or the one after the enrolment's (E2E_TOTP_LAST_STEP), which it accepts as one step of drift.
async function totpCode(): Promise<string> {
  const now = () => Math.floor(Date.now() / 30_000)
  const step = Math.max(now(), Number(required('E2E_TOTP_LAST_STEP')) + 1)
  await expect.poll(now, { timeout: 60_000 }).toBeGreaterThanOrEqual(step - 1)
  const counter = Buffer.alloc(8)
  counter.writeBigInt64BE(BigInt(step))
  const sum = createHmac('sha1', Buffer.from(required('E2E_TOTP_SECRET'), 'hex')).update(counter).digest()
  const value = sum.readUInt32BE(sum[sum.length - 1] & 0x0f) & 0x7fffffff
  return String(value % 1_000_000).padStart(6, '0')
}

export function required(name: string): string {
  const value = process.env[name]
  if (!value) throw new Error(`${name} is not set; make e2e runs the specs from the Go tests in test/e2e`)
  return value
}
