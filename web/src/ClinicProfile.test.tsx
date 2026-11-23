import { cleanup, fireEvent, screen } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'
import { ClinicProfile } from './ClinicProfile'
import { newApiClient } from './api/client'
import { renderInLanguage } from './i18n/renderInLanguage'
import { apiError, expectCsrfOnWrites, fakeFetch, type Route } from './test/fakeFetch'

afterEach(cleanup)

const profile = {
  languages: ['bs'],
  description: 'Synthetic clinic description',
  translations: [{ language: 'de', description: 'Synthetische Beschreibung' }],
  welcomesInternational: false,
}

function renderProfile(routes: Record<string, Route>) {
  const fake = fakeFetch({
    'GET /clinic/profile': () => Response.json(profile, { headers: { ETag: 'W/"3"' } }),
    ...routes,
  })
  renderInLanguage(<ClinicProfile api={newApiClient(fake.fetch)} />, 'en')
  return fake
}

describe('ClinicProfile', () => {
  it('saves the languages spoken, the welcome and the description with its translations, at the version read', async () => {
    const fake = renderProfile({
      'PUT /clinic/profile': async (r) => Response.json(await r.json(), { headers: { ETag: 'W/"4"' } }),
    })
    fireEvent.click(await screen.findByRole('checkbox', { name: 'Deutsch' }))
    fireEvent.click(screen.getByRole('checkbox', { name: 'We welcome patients from abroad' }))
    fireEvent.change(screen.getByLabelText('Description'), { target: { value: 'Updated description' } })
    fireEvent.change(screen.getByLabelText('Description in English'), { target: { value: 'In English' } })
    fireEvent.change(screen.getByLabelText('Description in Deutsch'), { target: { value: '' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save profile' }))
    expect(await screen.findByText('Profile saved.')).toBeTruthy()
    const put = fake.requests.find((r) => r.method === 'PUT')!
    expect(put.headers.get('If-Match')).toBe('W/"3"')
    expect(await put.json()).toEqual({
      languages: ['bs', 'de'],
      description: 'Updated description',
      translations: [{ language: 'en', description: 'In English' }],
      welcomesInternational: true,
    })
    expectCsrfOnWrites(fake.requests)
  })

  it('sends the version of its last save with the next one', async () => {
    const fake = renderProfile({
      'PUT /clinic/profile': async (r) => Response.json(await r.json(), { headers: { ETag: 'W/"4"' } }),
    })
    fireEvent.click(await screen.findByRole('button', { name: 'Save profile' }))
    await screen.findByText('Profile saved.')
    fireEvent.click(screen.getByRole('button', { name: 'Save profile' }))
    await screen.findByText('Profile saved.')
    const puts = () => fake.requests.filter((r) => r.method === 'PUT')
    await expect.poll(() => puts().length).toBe(2)
    expect(puts().map((r) => r.headers.get('If-Match'))).toEqual(['W/"3"', 'W/"4"'])
  })

  it('says when someone else saved first, or the profile is invalid, keeping what was entered', async () => {
    let answer = apiError(409, 'version_conflict')
    renderProfile({ 'PUT /clinic/profile': () => answer })
    fireEvent.change(await screen.findByLabelText('Description'), { target: { value: 'Kept' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save profile' }))
    expect(await screen.findByText('Someone else changed the profile. Reload the page to see the change; what you entered stays in the form.')).toBeTruthy()
    expect((screen.getByLabelText('Description') as HTMLTextAreaElement).value).toBe('Kept')
    answer = apiError(400, 'invalid_profile')
    fireEvent.click(screen.getByRole('button', { name: 'Save profile' }))
    expect(await screen.findByText('Keep the description and each translation under 2000 characters.')).toBeTruthy()
    answer = apiError(500, 'internal')
    fireEvent.click(screen.getByRole('button', { name: 'Save profile' }))
    expect(await screen.findByText('The profile could not be saved. Try again.')).toBeTruthy()
  })

  it('refuses translations without a description before sending', async () => {
    const fake = renderProfile({})
    fireEvent.change(await screen.findByLabelText('Description'), { target: { value: ' ' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save profile' }))
    expect(await screen.findByText('Write the description before its translations.')).toBeTruthy()
    expect(fake.requests.some((r) => r.method === 'PUT')).toBe(false)
  })

  it('says when the profile could not be loaded', async () => {
    renderProfile({ 'GET /clinic/profile': () => apiError(500, 'internal') })
    expect(await screen.findByText('The profile could not be loaded. Reload the page to try again.')).toBeTruthy()
  })
})
