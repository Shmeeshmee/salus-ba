import { cleanup, fireEvent, screen, within } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'
import { newApiClient } from '../api/client'
import type { components } from '../api/schema'
import { renderInLanguage } from '../i18n/renderInLanguage'
import { apiError, expectCsrfOnWrites, fakeFetch, type Route } from '../test/fakeFetch'
import { GroupNames } from './GroupNames'

afterEach(cleanup)

type GroupName = components['schemas']['DiagnosisGroupName']

// Synthetic group names in English: a group, its subgroup (needs changes, with a note) and an approved group.
const respiratory: GroupName = { code: 'respiratory-infections', language: 'en', text: 'Respiratory infections', review: { status: 'unreviewed' }, versionId: 1 }
const cold: GroupName = {
  code: 'common-cold', parent: 'respiratory-infections', language: 'en', text: 'Common cold',
  review: { status: 'needs_changes', note: 'Placeholder note' }, versionId: 3,
}
const other: GroupName = { code: 'other', language: 'en', text: 'Other', review: { status: 'approved' }, versionId: 2 }

function setup(routes: Record<string, Route> = {}, reviewer = true) {
  const fake = fakeFetch({
    'GET /terminology/review/diagnosis-groups': () => Response.json({ groups: [respiratory, cold, other] }),
    ...routes,
  })
  renderInLanguage(<GroupNames api={newApiClient(fake.fetch)} reviewer={reviewer} />, 'en')
  return fake
}

const card = (name: string) => screen.getByRole('article', { name })

describe('GroupNames', () => {
  it('lists every group with its status, a subgroup with its group and the reviewer\'s note', async () => {
    const fake = setup()
    await screen.findByRole('article', { name: 'Common cold' })
    expect(within(card('Respiratory infections')).getByText('Unreviewed')).toBeTruthy()
    expect(within(card('Common cold')).getByText('Subgroup of Respiratory infections')).toBeTruthy()
    expect(within(card('Common cold')).getByText('Note: Placeholder note')).toBeTruthy()
    expect(within(card('Other')).getByText('Approved')).toBeTruthy()
    expect(new URL(fake.requests[0].url).searchParams.get('language')).toBe('en')
  })

  it('asks for the groups of another language', async () => {
    const fake = setup()
    await screen.findByRole('article', { name: 'Common cold' })
    fireEvent.change(screen.getByRole('combobox', { name: 'Language' }), { target: { value: 'bs' } })
    await screen.findByRole('article', { name: 'Common cold' })
    expect(new URL(fake.requests.at(-1)!.url).searchParams.get('language')).toBe('bs')
  })

  it('approves a name at the version read', async () => {
    const fake = setup({
      'POST /terminology/diagnosis-groups/respiratory-infections/names/en/review': () => Response.json({ ...respiratory, review: { status: 'approved' }, versionId: 2 }),
    })
    await screen.findByRole('article', { name: 'Respiratory infections' })
    fireEvent.click(within(card('Respiratory infections')).getByRole('button', { name: 'Approve' }))
    expect(await within(card('Respiratory infections')).findByText('Approved')).toBeTruthy()
    const post = fake.requests.find((r) => r.method === 'POST')!
    expect(post.headers.get('If-Match')).toBe('W/"1"')
    expect(await post.json()).toEqual({ status: 'approved' })
    expectCsrfOnWrites(fake.requests)
  })

  it('edits a name, which comes back unreviewed', async () => {
    const fake = setup({
      'PUT /terminology/diagnosis-groups/other/names/en': () => Response.json({ ...other, text: 'Other diagnoses', review: { status: 'unreviewed' }, versionId: 3 }),
    })
    await screen.findByRole('article', { name: 'Other' })
    fireEvent.click(within(card('Other')).getByRole('button', { name: 'Edit' }))
    fireEvent.change(screen.getByRole('textbox', { name: 'Name' }), { target: { value: 'Other diagnoses' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))
    expect(await screen.findByRole('article', { name: 'Other diagnoses' })).toBeTruthy()
    const put = fake.requests.find((r) => r.method === 'PUT')!
    expect(put.headers.get('If-Match')).toBe('W/"2"')
    expect(await put.json()).toEqual({ text: 'Other diagnoses' })
  })

  it('requests changes with a note', async () => {
    const fake = setup({
      'POST /terminology/diagnosis-groups/other/names/en/review': () => Response.json({ ...other, review: { status: 'needs_changes', note: 'Too broad' }, versionId: 3 }),
    })
    await screen.findByRole('article', { name: 'Other' })
    fireEvent.click(within(card('Other')).getByRole('button', { name: 'Request changes' }))
    fireEvent.change(screen.getByRole('textbox', { name: 'What needs to change' }), { target: { value: 'Too broad' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))
    expect(await within(card('Other')).findByText('Note: Too broad')).toBeTruthy()
    expect(await fake.requests.find((r) => r.method === 'POST')!.json()).toEqual({ status: 'needs_changes', note: 'Too broad' })
  })

  it('says so and reloads when someone changed the name meanwhile', async () => {
    const fake = setup({ 'POST /terminology/diagnosis-groups/other/names/en/review': () => apiError(412, 'version_conflict') })
    await screen.findByRole('article', { name: 'Other' })
    fireEvent.click(within(card('Other')).getByRole('button', { name: 'Approve' }))
    expect((await screen.findByRole('alert')).textContent).toBe('Someone changed this entry in the meantime. The queue has been reloaded.')
    expect(fake.requests.filter((r) => r.method === 'GET')).toHaveLength(2)
  })

  it('drops an answer that comes after a switch of language', async () => {
    let answer: (r: Response) => void = () => undefined
    setup({
      'POST /terminology/diagnosis-groups/other/names/en/review': () => new Promise<Response>((resolve) => (answer = resolve)),
    })
    await screen.findByRole('article', { name: 'Other' })
    fireEvent.click(within(card('Other')).getByRole('button', { name: 'Approve' }))
    fireEvent.change(screen.getByRole('combobox', { name: 'Language' }), { target: { value: 'bs' } })
    await screen.findByRole('article', { name: 'Other' })
    answer(Response.json({ ...other, text: 'Other (en, approved)', review: { status: 'approved' }, versionId: 3 }))
    await new Promise((r) => setTimeout(r, 0))
    expect(screen.queryByRole('article', { name: 'Other (en, approved)' })).toBeNull()
  })

  it('has no controls without the reviewer permission', async () => {
    setup({}, false)
    await screen.findByRole('article', { name: 'Other' })
    expect(screen.queryByRole('button', { name: 'Approve' })).toBeNull()
    expect(screen.queryByRole('button', { name: 'Edit' })).toBeNull()
  })

  it('says when the groups cannot be loaded', async () => {
    setup({ 'GET /terminology/review/diagnosis-groups': () => apiError(500, 'internal error') })
    expect((await screen.findByRole('alert')).textContent).toBe('The diagnosis groups could not be loaded.')
  })
})
