import { existsSync } from 'node:fs'
import { expect, test, type Page } from '@playwright/test'
import { required, signIn } from './signin'

// The practitioner's side of TestConsultation_ThePractitionerEndsTheCallFromTheWebAppAndTheAppointmentIsFulfilled
// (test/e2e/consultation_test.go), which makes the synthetic clinic, patient and appointment, connects the patient
// and passes everything in E2E_* variables. The texts are the approved ones (docs/consultation/COPY.md).
test('the practitioner signs in, joins from the appointment, sees and hears the patient and ends the consultation', async ({ page }) => {
  const patient = required('E2E_PATIENT_NAME')
  await countInboundAudio(page)

  required('E2E_LIVEKIT_PUBLIC_URL') // forwarded by the global setup (livekit-forward.ts)
  await signIn(page)

  await page.goto(`/appointments/${required('E2E_APPOINTMENT_ID')}`)
  await page.getByRole('button', { name: 'Join consultation' }).click() // W1
  await expect(page.getByText('Check your camera and microphone')).toBeVisible() // W3
  await page.getByRole('button', { name: 'Join', exact: true }).click() // W8, enabled once both fake devices opened

  // The patient is in the room with a microphone and no camera: W16 and not W17, and their tone arrives.
  await expect(page.getByText(`${patient}'s camera is off`)).toBeVisible()
  await expect(page.getByText(`${patient} is muted`)).toHaveCount(0)
  await expect.poll(() => inboundAudioPackets(page), { timeout: 30_000 }).toBeGreaterThan(0)

  // The transcription agent hears the tone and the fake speech engine says its placeholder line, which reaches the
  // page over the encounter's event stream (SAL-6): the patient's line in the transcript beside the video
  // (docs/consult-intelligence/COPY.md W1, W2, W5).
  const side = page.getByRole('complementary', { name: 'Transcript, suggestions and documents' })
  const transcript = side.getByRole('list', { name: 'Transcript' })
  const line = transcript.getByRole('listitem').filter({ hasText: 'Placeholder caption from the test speech engine' })
  await expect(line.getByText('Patient', { exact: true })).toBeVisible({ timeout: 30_000 })

  // SAL-13: the term the line says (the Go test added it with its explanation) is explained on hover, labelled
  // informational and not reviewed yet; the line itself reads as it was said.
  const label = required('E2E_TERM_LABEL')
  const patientLine = line.filter({ has: page.getByText('Patient', { exact: true }) }) // the practitioner's mic says it too
  await expectExplained(page, patientLine.locator('.term', { hasText: label }).first())
  await expect(patientLine).toContainText('Placeholder caption from the test speech engine')

  // Drafting (SAL-8) on the fake model, which leaves every field missing: the required field blocks the sign until
  // the practitioner fills it in, and a diagnosis code until they confirm it (SAL-19); then the document is signed
  // (W15 to W24).
  await side.getByRole('tab', { name: 'Documents' }).click()
  await side.getByRole('combobox', { name: 'Template' }).selectOption({ label: required('E2E_TEMPLATE_TITLE') })
  await side.getByRole('button', { name: 'Generate' }).click()
  const reason = side.getByRole('textbox', { name: 'Placeholder reason' })
  await expect(reason).toHaveValue('')
  await expect(side.getByText('Not covered in the conversation yet')).toHaveCount(3)
  await side.getByRole('button', { name: 'Sign' }).click()
  await expect(side.getByRole('alert')).toHaveText('1 required field is empty. Fill it in before you sign.')
  await reason.fill(`Placeholder reason typed in the e2e journey about the ${label}`)
  // A typed code counts as confirmed; unconfirmed, it blocks the sign.
  await side.getByRole('textbox', { name: 'Placeholder diagnosis' }).fill('J06.9')
  const confirmCode = side.getByRole('checkbox', { name: 'I confirm this diagnosis code' })
  await expect(confirmCode).toBeChecked()
  await confirmCode.uncheck()
  await side.getByRole('button', { name: 'Sign' }).click()
  await expect(side.getByRole('alert')).toHaveText('Confirm each diagnosis code before you sign.')
  await confirmCode.check()
  await side.getByRole('button', { name: 'Sign' }).click()
  const ask = side.getByRole('alertdialog', { name: 'Sign this document?' })
  await expect(ask.getByText('1 field is empty and stays empty.')).toBeVisible()
  await ask.getByRole('button', { name: 'Sign' }).click()
  await expect(side.getByText('Signed. This document is final.')).toBeVisible()
  await expect(side.getByText('Placeholder reason typed in the e2e journey')).toBeVisible()
  // The signed document's term is explained on hover too; the document is shown as it was signed.
  const signed = side.locator('.draft-signed')
  await expectExplained(page, signed.locator('.term', { hasText: label }))
  await expect(signed).toContainText(`Placeholder reason typed in the e2e journey about the ${label}`)

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

// expectExplained hovers term and expects its tooltip: the explanation, informational, not reviewed yet (the texts
// are the catalogs', web/src/i18n/terminology/en.json).
async function expectExplained(page: Page, term: ReturnType<Page['locator']>) {
  await term.hover()
  const tip = term.getByRole('tooltip')
  await expect(tip).toBeVisible()
  await expect(tip).toContainText(required('E2E_TERM_EXPLANATION'))
  await expect(tip).toContainText('Informational, not a diagnosis · Not reviewed yet')
  await page.mouse.move(0, 0)
}

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
