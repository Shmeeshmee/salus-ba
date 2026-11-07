import { expect, test } from '@playwright/test'
import { required, signIn } from './signin'

// The practitioner's side of TestAttachments_ThePatientAttachesFilesAndThePractitionerOpensThem
// (test/e2e/attachments_test.go), which makes the synthetic clinic, the submitted response with a generated photo
// and a generated video, and passes everything in E2E_* variables. The texts are the approved ones
// (docs/attachments/COPY.md, docs/questionnaire/COPY.md); the titles are placeholders.
//
// Firefox, not Chromium: Playwright's Chromium is built without the proprietary codecs, so it plays no H.264, and
// the files are H.264 MP4 only (docs/attachments/SPEC.md §3 O1). Firefox decodes it with the image's libavcodec.
// Chrome itself is checked in the owner's acceptance walk. The config's fake media devices are Chromium's.
test.use({ browserName: 'firefox', permissions: [], launchOptions: {} })

test('the practitioner opens the submitted answers, the photo in the viewer, and plays and seeks the video', async ({ page }) => {
  const photo = required('E2E_PHOTO_TITLE')
  const video = required('E2E_VIDEO_TITLE')
  const seconds = Number(required('E2E_VIDEO_SECONDS'))
  const content = `/binaries/${required('E2E_VIDEO_ID')}/content`
  const partial: number[] = []
  page.on('response', (r) => {
    if (new URL(r.url()).pathname === content && r.status() === 206) partial.push(r.status())
  })
  await signIn(page)

  await page.goto(`/appointments/${required('E2E_APPOINTMENT_ID')}`)
  await page.getByRole('button', { name: 'View answers' }).click() // questionnaire W37
  await expect(page.getByText('1 file')).toHaveCount(2) // W7, one per item: the rejected document was detached
  await expect(page.getByText(photo, { exact: true })).toBeVisible() // W8
  await expect(page.getByText(video, { exact: true })).toBeVisible()

  // The photo in the viewer (W9, W10, W11), decoded by the browser.
  await page.locator('.files li').filter({ hasText: photo }).getByRole('button', { name: 'Open' }).click()
  const viewer = page.getByRole('dialog')
  await expect(viewer.getByText('File 1 of 1')).toBeVisible()
  const img = viewer.getByRole('img', { name: `Photo: ${photo}` })
  await expect.poll(() => img.evaluate((e: HTMLImageElement) => e.complete && e.naturalWidth)).toBe(64)
  await viewer.getByRole('button', { name: 'Close' }).click()
  await expect(viewer).toHaveCount(0)

  // The video in the browser's player on the content URL: it plays, and a seek lands where it was sent. The
  // browser asks for the bytes with Range requests, which the API answers 206.
  await page.locator('.files li').filter({ hasText: video }).getByRole('button', { name: 'Open' }).click()
  const player = page.getByLabel(`Video: ${video}`)
  await expect.poll(() => player.evaluate((v: HTMLVideoElement) => v.duration)).toBe(seconds)
  await player.evaluate((v: HTMLVideoElement) => {
    v.muted = true
    return v.play()
  })
  await expect.poll(() => player.evaluate((v: HTMLVideoElement) => v.currentTime)).toBeGreaterThan(0.5)
  const target = seconds - 2
  await player.evaluate(
    (v: HTMLVideoElement, to: number) =>
      new Promise<void>((resolve) => {
        v.addEventListener('seeked', () => resolve(), { once: true })
        v.currentTime = to
      }),
    target,
  )
  expect(await player.evaluate((v: HTMLVideoElement) => v.currentTime)).toBeGreaterThanOrEqual(target)
  await expect.poll(() => player.evaluate((v: HTMLVideoElement) => v.currentTime)).toBeGreaterThan(target + 0.5)
  expect(partial.length).toBeGreaterThan(0)
})
