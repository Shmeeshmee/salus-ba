import { createHmac } from 'node:crypto'
import { existsSync } from 'node:fs'
import { expect, test, type Page } from '@playwright/test'

// The practitioner's side of TestConsultation_ThePractitionerEndsTheCallFromTheWebAppAndTheAppointmentIsFulfilled
// (test/e2e/consultation_test.go), which makes the synthetic clinic, patient and appointment, connects the patient
// and passes everything in E2E_* variables. The texts are the approved ones (docs/consultation/COPY.md).
test('the practitioner signs in, joins from the appointment, sees and hears the patient and ends the consultation', async ({ page }) => {
  const patient = required('E2E_PATIENT_NAME')
  await countInboundAudio(page)

  await page.goto('/sign-in')
  await page.getByLabel('E-mail').fill(required('E2E_PRACTITIONER_EMAIL'))
  await page.getByLabel('Password').fill(required('E2E_PASSWORD'))
  await page.getByRole('button', { name: 'Sign in' }).click()
  await page.getByLabel('Code').fill(await totpCode())
  await page.getByRole('button', { name: 'Continue' }).click()
  await expect(page).toHaveURL(/:5173\/$/)

  await page.goto(`/appointments/${required('E2E_APPOINTMENT_ID')}`)
  await page.getByRole('button', { name: 'Join consultation' }).click() // W1
  await expect(page.getByText('Check your camera and microphone')).toBeVisible() // W3
  await page.getByRole('button', { name: 'Join', exact: true }).click() // W8, enabled once both fake devices opened

  // The patient is in the room with a microphone and no camera: W16 and not W17, and their tone arrives.
  await expect(page.getByText(`${patient}'s camera is off`)).toBeVisible()
  await expect(page.getByText(`${patient} is muted`)).toHaveCount(0)
  await expect.poll(() => inboundAudioPackets(page), { timeout: 30_000 }).toBeGreaterThan(0)

  // The Go test checks the Encounter is in progress, then signals the end.
  await expect.poll(() => existsSync(required('E2E_END_SIGNAL')), { timeout: 60_000 }).toBe(true)
  await page.getByRole('button', { name: 'End consultation' }).click() // W19
  const confirm = page.locator('.end-confirm')
  await expect(confirm.getByText('End the consultation?')).toBeVisible() // W20
  await expect(
    confirm.getByText('The patient will be disconnected and cannot join again. The appointment will be marked as fulfilled.'),
  ).toBeVisible() // W21
  await confirm.getByRole('button', { name: 'End consultation' }).click()
  await expect(page.getByRole('heading', { name: 'The consultation has ended' })).toBeVisible() // W27
  await expect(page.getByText('The appointment is marked as fulfilled.')).toBeVisible() // W28
})

// countInboundAudio keeps every RTCPeerConnection the page makes, so the test can read their statistics: the app
// and livekit-client expose none.
async function countInboundAudio(page: Page) {
  await page.addInitScript(() => {
    const made: RTCPeerConnection[] = []
    const Native = window.RTCPeerConnection
    class Counted extends Native {
      constructor(config?: RTCConfiguration) {
        super(config)
        made.push(this)
      }
    }
    Object.assign(window, { RTCPeerConnection: Counted, e2ePeerConnections: made })
  })
}

// inboundAudioPackets is the number of audio RTP packets the page has received.
function inboundAudioPackets(page: Page): Promise<number> {
  return page.evaluate(async () => {
    const made = (window as unknown as { e2ePeerConnections: RTCPeerConnection[] }).e2ePeerConnections
    let packets = 0
    for (const pc of made) {
      ;(await pc.getStats()).forEach((s) => {
        if (s.type === 'inbound-rtp' && s.kind === 'audio') packets += s.packetsReceived ?? 0
      })
    }
    return packets
  })
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

function required(name: string): string {
  const value = process.env[name]
  if (!value) throw new Error(`${name} is not set; make e2e runs this spec from test/e2e/consultation_test.go`)
  return value
}
