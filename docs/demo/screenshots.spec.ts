import { createHmac } from 'node:crypto'
import { expect, test, type Page } from '@playwright/test'

// Takes the screenshots of SCRIPT.md at 1280 px, signed in as the demo's staff with the password and the TOTP secrets
// the seed printed (DEMO_TOTP_ADMIN, DEMO_TOTP_RECEPTION, DEMO_TOTP_PRACTITIONER: base32), on the visits, the
// questionnaire and the chat screenshots.sh reads from the stack (DEMO_VISIT_QUESTIONNAIRE, DEMO_VISIT_LAB,
// DEMO_QUESTIONNAIRE, DEMO_CHAT). The accounts are the seed's synthetic ones; nothing here works on any other stack.

const password = 'demo-lozinka-primjer'
const shots = 'screenshots'

// totp is the authenticator's current code for a base32 secret (RFC 6238: SHA-1, 6 digits, 30 s).
function totp(base32: string): string {
  const alphabet = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ234567'
  let bits = ''
  for (const ch of base32.replace(/=+$/, '')) bits += alphabet.indexOf(ch).toString(2).padStart(5, '0')
  const key = Buffer.from(bits.match(/.{8}/g)!.map((b) => parseInt(b, 2)))
  const counter = Buffer.alloc(8)
  counter.writeBigInt64BE(BigInt(Math.floor(Date.now() / 30_000)))
  const sum = createHmac('sha1', key).update(counter).digest()
  const value = sum.readUInt32BE(sum[sum.length - 1] & 0x0f) & 0x7fffffff
  return String(value % 1_000_000).padStart(6, '0')
}

function secret(name: string): string {
  const value = process.env[name]
  if (!value) throw new Error(`${name} is not set: run docs/demo/screenshots.sh`)
  return value
}

async function signIn(page: Page, email: string, base32: string) {
  await page.goto('/sign-in')
  await page.getByLabel('E-mail').fill(email)
  await page.getByLabel('Lozinka').fill(password)
  await page.getByRole('button', { name: 'Prijavi se' }).click()
  await page.getByLabel('Kod').fill(totp(base32))
  await page.getByRole('button', { name: 'Nastavi' }).click()
  await expect(page).not.toHaveURL(/sign-in/)
}

// shot opens path and saves the page as name once the network is quiet.
async function shot(page: Page, path: string, name: string) {
  await page.goto(path)
  await page.waitForLoadState('networkidle')
  await page.waitForTimeout(500)
  await page.screenshot({ path: `${shots}/${name}.png`, fullPage: false })
}

test('the front desk', async ({ page }) => {
  await signIn(page, 'recepcija@demo.example.test', secret('DEMO_TOTP_RECEPTION'))
  await shot(page, '/desk', '1-desk')
})

test('a visit, its hours and its chat', async ({ page }) => {
  await signIn(page, 'emir.primjerovic@demo.example.test', secret('DEMO_TOTP_PRACTITIONER'))
  await shot(page, '/', '2-calendar')
  await shot(page, `/appointments/${secret('DEMO_VISIT_QUESTIONNAIRE')}`, '2-visit-questionnaire')
  await shot(page, `/appointments/${secret('DEMO_VISIT_LAB')}`, '2-visit-lab')
  await shot(page, `/questionnaires/${secret('DEMO_QUESTIONNAIRE')}`, '2-questionnaire-builder')
  await shot(page, `/chats/${secret('DEMO_CHAT')}`, '2-chat')
  await shot(page, '/availability', '4-hours')
})

test('receipts and the numbers', async ({ page }) => {
  await signIn(page, 'uprava@demo.example.test', secret('DEMO_TOTP_ADMIN'))
  await shot(page, '/receipts', '3-receipts')
  await shot(page, '/receipts/practice', '4-online-practice')
  await shot(page, '/analytics', '6-numbers')
  await shot(page, '/receipts/report', '6-receipts-report')
  await shot(page, '/membership', '6-membership')
  await shot(page, '/data-requests', '6-data-requests')
  await shot(page, '/price-list', '0-price-list')
  await shot(page, '/locations', '0-locations')
})
