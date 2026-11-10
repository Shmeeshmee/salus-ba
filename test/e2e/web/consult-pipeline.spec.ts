import { existsSync, readFileSync, writeFileSync } from 'node:fs'
import { expect, test } from '@playwright/test'
import { required, signIn } from './signin'

// The practitioner's side of TestConsultPipeline_AGermanPatientAndTheBosnianPractitionerInTheWebApp
// (test/e2e/consult_pipeline_test.go): the Go test books the synthetic appointment, connects the German patient and
// says the scripted conversation (E2E_CONVERSATION, test/testdata/consult/de.json) as tones, which the fake engines
// turn into its lines, their Bosnian translations, two suggestions and the draft's fields. Here the practitioner sees
// the captions and the transcript, pins one suggestion and dismisses the other, drafts the document, edits a field
// and signs it. The UI texts are the approved ones (docs/consult-intelligence/COPY.md).

type Conversation = {
  lines: { speaker: string; text: string; translations: Record<string, string> }[]
  suggestions: { type: string; text: string }[]
  fields: { key: string; label: string; from?: string; value?: string }[]
}

test('the practitioner follows the translated conversation, its suggestions and signs the drafted document', async ({ page }) => {
  test.setTimeout(210_000)
  const conversation = JSON.parse(readFileSync(required('E2E_CONVERSATION'), 'utf8')) as Conversation
  const said = conversation.lines.filter((l) => l.speaker === 'patient').map((l) => ({ original: l.text, translation: l.translations.bs }))

  required('E2E_LIVEKIT_PUBLIC_URL') // forwarded by the global setup (livekit-forward.ts)
  await signIn(page)
  await page.goto(`/appointments/${required('E2E_APPOINTMENT_ID')}`)
  await page.getByRole('button', { name: 'Join consultation' }).click()
  await page.getByRole('button', { name: 'Join', exact: true }).click()

  // The patient's words in the practitioner's language on the video, while the conversation goes on (a final
  // caption goes after 6 s), and every line in the transcript: the translation, then the original.
  const captions = page.getByRole('group', { name: 'Captions' })
  await expect(captions.getByText(new RegExp(said.map((l) => escape(l.translation)).join('|')))).toBeVisible({ timeout: 60_000 })
  const side = page.getByRole('complementary', { name: 'Transcript, suggestions and documents' })
  const transcript = side.getByRole('list', { name: 'Transcript' })
  for (const l of said) {
    const line = transcript.getByRole('listitem').filter({ hasText: l.translation })
    await expect(line.getByText(l.original)).toBeVisible({ timeout: 60_000 })
    await expect(line.getByText('Patient', { exact: true })).toBeVisible()
  }

  // The suggestions, labelled as AI prompts: the first is pinned, the second dismissed (W9 to W14).
  await side.getByRole('tab', { name: /^Suggestions/ }).click()
  await expect(side.getByText('AI suggestions are prompts drawn from the transcript, not a diagnosis. Every decision is yours.')).toBeVisible()
  const [first, second] = conversation.suggestions.map((s) => side.getByRole('article').filter({ hasText: s.text }))
  await expect(first).toBeVisible({ timeout: 60_000 })
  await expect(second).toBeVisible({ timeout: 60_000 })
  await expect(first.getByText('Suggestion: a question to ask')).toBeVisible()
  await first.getByRole('button', { name: 'Pin' }).click()
  await expect(first.getByRole('button', { name: 'Unpin' })).toBeVisible()
  await second.getByRole('button', { name: 'Dismiss' }).click()
  await expect(second).toHaveCount(0)
  await expect(first.getByRole('button', { name: 'Unpin' })).toBeVisible() // the pin came back from the server

  // The draft from the transcript: filled fields with their sources, the uncovered one flagged; the practitioner
  // types over the second, signs, and the document is final (W15 to W24). Generate to the drafted values is timed.
  await side.getByRole('tab', { name: 'Documents' }).click()
  await side.getByRole('combobox', { name: 'Template' }).selectOption({ label: required('E2E_TEMPLATE_TITLE') })
  const [complaint, course, plan] = conversation.fields
  const began = Date.now()
  await side.getByRole('button', { name: 'Generate' }).click()
  await expect(side.getByRole('textbox', { name: complaint.label })).toHaveValue(complaint.value!, { timeout: 30_000 })
  const draftMs = Date.now() - began
  await expect(side.getByRole('textbox', { name: course.label })).toHaveValue(course.value!)
  await expect(side.getByRole('textbox', { name: plan.label })).toHaveValue('')
  await expect(side.getByText('Not covered in the conversation yet')).toHaveCount(1)
  await expect(side.getByRole('button', { name: 'Show the source line' })).toHaveCount(2)
  await side.getByRole('textbox', { name: course.label }).fill(required('E2E_EDITED_VALUE'))
  await side.getByRole('button', { name: 'Sign' }).click()
  const ask = side.getByRole('alertdialog', { name: 'Sign this document?' })
  await expect(ask.getByText('1 field is empty and stays empty.')).toBeVisible()
  await ask.getByRole('button', { name: 'Sign' }).click()
  await expect(side.getByText('Signed. This document is final.')).toBeVisible()
  await expect(side.getByText(required('E2E_EDITED_VALUE'))).toBeVisible()
  writeFileSync(required('E2E_DRAFT_TIMING'), JSON.stringify({ draftMs }))

  // The Go test checks what was stored, then signals the end.
  await expect.poll(() => existsSync(required('E2E_END_SIGNAL')), { timeout: 90_000 }).toBe(true)
  await page.getByRole('button', { name: 'End consultation' }).click()
  await page.locator('.end-confirm').getByRole('button', { name: 'End consultation' }).click()
  await expect(page.getByRole('heading', { name: 'The consultation has ended' })).toBeVisible()
})

function escape(text: string): string {
  return text.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}
