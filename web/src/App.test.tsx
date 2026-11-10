import { cleanup, fireEvent, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { renderInLanguage } from './i18n/renderInLanguage'
import App from './App'
import { newApiClient, type Account } from './api/client'
import { newFhirClient } from './api/fhir'
import { loadSavedLanguage } from './i18n/saved'
import { apiError, expectCsrfOnWrites, fakeFetch, fhirJSON, type Route } from './test/fakeFetch'

afterEach(() => {
  cleanup()
  localStorage.clear()
})

const org = 'a0000000-0000-4000-8000-00000000000a'
const admin: Account = { id: 'a1000000-0000-4000-8000-000000000001', email: 'admin@example.test', role: 'admin', organizationId: org, mfaEnrolled: true }
const emptyBundle = () => fhirJSON({ resourceType: 'Bundle', type: 'searchset', total: 0 })
const organization = () =>
  fhirJSON({ resourceType: 'Organization', id: org, name: 'Synthetic clinic', active: true, meta: { versionId: '1', lastUpdated: '2026-11-04T09:30:00Z' } })

function renderAt(path: string, routes: Record<string, Route>) {
  history.replaceState(null, '', path)
  const fake = fakeFetch(routes)
  renderInLanguage(<App api={newApiClient(fake.fetch)} fhir={newFhirClient(fake.fetch)} />, 'en')
  return fake
}

const staff = (account: Account): Record<string, Route> => ({
  'GET /auth/me': () => Response.json(account),
  'GET /fhir/R4/PractitionerRole': emptyBundle,
  'GET /fhir/R4/Appointment': emptyBundle,
  'GET /fhir/R4/Questionnaire': emptyBundle,
  'GET /invites': () => Response.json({ invites: [] }),
  'GET /specialties': () => Response.json({ specialties: [] }),
  [`GET /fhir/R4/Organization/${org}`]: organization,
})

describe('App', () => {
  it('sends a visitor without a session to sign in', async () => {
    renderAt('/', { 'GET /auth/me': () => apiError(401, 'unauthenticated') })

    expect(await screen.findByRole('heading', { name: 'Sign in' })).toBeTruthy()
    expect(location.pathname).toBe('/sign-in')
  })

  it('opens the calendar after signing in', async () => {
    renderAt('/sign-in', { ...staff(admin), 'POST /auth/login': () => Response.json({ next: 'done' }) })

    fireEvent.change(screen.getByLabelText('E-mail'), { target: { value: 'admin@example.test' } })
    fireEvent.change(screen.getByLabelText('Password'), { target: { value: 'correct horse battery' } })
    fireEvent.click(screen.getByRole('button', { name: 'Sign in' }))

    expect(await screen.findByRole('heading', { name: 'Calendar' })).toBeTruthy()
    expect(location.pathname).toBe('/')
  })

  it("opens an appointment's consultation on its call path", async () => {
    const practitioner: Account = { ...admin, role: 'practitioner', practitionerId: 'a3000000-0000-4000-8000-0000000000a1' }
    renderAt('/appointments/e1000000-0000-4000-8000-000000000001/call', staff(practitioner))

    expect(await screen.findByRole('heading', { name: 'Check your camera and microphone' })).toBeTruthy()
  })

  it('shows the accept invite screen on /invite', () => {
    renderAt('/invite#tok-synthetic', {})

    expect(screen.getByRole('heading', { name: 'Accept your invite' })).toBeTruthy()
  })

  it.each([
    ['Working hours', '/availability'],
    ['Practitioners', '/practitioners'],
    ['Questionnaires', '/questionnaires'],
  ])('gives staff the calendar and %s', async (screenName, path) => {
    renderAt('/', staff({ ...admin, role: 'practitioner', practitionerId: 'a3000000-0000-4000-8000-0000000000a1' }))

    expect(await screen.findByRole('heading', { name: 'Calendar' })).toBeTruthy()
    fireEvent.click(screen.getByRole('link', { name: screenName }))

    expect(await screen.findByRole('heading', { name: screenName })).toBeTruthy()
    expect(location.pathname).toBe(path)
  })

  it("opens a questionnaire's builder from its path", async () => {
    renderAt('/questionnaires/q1', {
      ...staff(admin),
      'GET /fhir/R4/Questionnaire/q1': () =>
        fhirJSON({
          resourceType: 'Questionnaire',
          id: 'q1',
          url: 'https://salus.invalid/fhir/Questionnaire/s1',
          version: '1',
          title: 'Placeholder intake',
          status: 'draft',
          meta: { versionId: '1', lastUpdated: '2026-11-04T09:30:00Z' },
        }),
    })

    expect(await screen.findByRole('heading', { name: 'Placeholder intake' })).toBeTruthy()
    expect(screen.getByRole('heading', { name: 'Add an item' })).toBeTruthy()
  })

  it('opens an appointment from its path', async () => {
    const id = 'e1000000-0000-4000-8000-000000000001'
    renderAt(`/appointments/${id}`, {
      ...staff(admin),
      [`GET /fhir/R4/Appointment/${id}`]: () =>
        fhirJSON({ resourceType: 'Appointment', id, status: 'booked', start: '2026-11-09T08:00:00Z', end: '2026-11-09T08:30:00Z', participant: [], meta: { versionId: '1', lastUpdated: '2026-11-04T09:30:00Z' } }),
    })

    expect(await screen.findByRole('heading', { name: 'Appointment' })).toBeTruthy()
  })

  it('gives an admin the clinic settings', async () => {
    renderAt('/', staff(admin))

    expect(await screen.findByRole('heading', { name: 'Calendar' })).toBeTruthy()
    fireEvent.click(screen.getByRole('link', { name: 'Clinic settings' }))

    expect(await screen.findByRole('heading', { name: 'Clinic settings' })).toBeTruthy()
    expect(location.pathname).toBe('/clinic')
  })

  it('gives a practitioner no clinic settings', async () => {
    renderAt('/clinic', staff({ ...admin, role: 'practitioner' }))

    expect(await screen.findByText('This page does not exist.')).toBeTruthy()
    expect(screen.queryByRole('link', { name: 'Clinic settings' })).toBeNull()
  })

  it('tells a patient the web app is for clinic staff', async () => {
    renderAt('/', { 'GET /auth/me': () => Response.json({ id: admin.id, email: 'pat@example.test', role: 'patient', mfaEnrolled: false }) })

    expect(await screen.findByText('The web app is for clinic staff. Patients use the Salus mobile app.')).toBeTruthy()
    expect(screen.queryByRole('heading', { name: 'Calendar' })).toBeNull()
  })

  it('says when a page does not exist', async () => {
    renderAt('/nowhere', staff(admin))

    expect(await screen.findByText('This page does not exist.')).toBeTruthy()
  })

  it('says when the account cannot be loaded', async () => {
    renderAt('/', { 'GET /auth/me': () => apiError(500, 'internal error') })

    expect((await screen.findByRole('alert')).textContent).toBe('Something went wrong. Try again.')
  })

  it('signs out with the CSRF header and returns to sign in', async () => {
    const { requests } = renderAt('/', { ...staff(admin), 'POST /auth/logout': () => new Response(null, { status: 204 }) })

    fireEvent.click(await screen.findByRole('button', { name: 'Sign out' }))

    expect(await screen.findByRole('heading', { name: 'Sign in' })).toBeTruthy()
    expectCsrfOnWrites(requests)
  })

  it('stays signed in when signing out fails', async () => {
    renderAt('/', { ...staff(admin), 'POST /auth/logout': () => apiError(500, 'internal error') })

    fireEvent.click(await screen.findByRole('button', { name: 'Sign out' }))

    await vi.waitFor(() => expect(screen.getByRole('alert').textContent).toBe('Signing out failed. Try again.'))
    expect(screen.getByRole('heading', { name: 'Calendar' })).toBeTruthy()
  })
})

describe('the language', () => {
  const me = (account: Account) => ({ ...staff(account), 'PATCH /auth/me': async (r: Request) => Response.json({ ...account, ...(await r.json()) }) })

  it("shows the app in the account's language", async () => {
    renderAt('/', staff({ ...admin, language: 'hr' }))

    expect(await screen.findByRole('heading', { name: 'Kalendar' })).toBeTruthy()
    expect(screen.getByRole('link', { name: 'Postavke klinike' })).toBeTruthy()
    expect(screen.getByRole('button', { name: 'Odjavi se' })).toBeTruthy()
    expect(loadSavedLanguage()).toBe('hr')
  })

  it('saves a language picked in the header on the account', async () => {
    const { requests } = renderAt('/', me(admin))

    fireEvent.change(await screen.findByRole('combobox', { name: 'Language' }), { target: { value: 'sr-Cyrl' } })

    expect(await screen.findByText('Језик је сачуван.')).toBeTruthy()
    expect(screen.getByRole('heading', { name: 'Календар' })).toBeTruthy()
    const patch = requests.find((r) => r.method === 'PATCH')!
    expect(new URL(patch.url).pathname).toBe('/auth/me')
    expect(await patch.json()).toEqual({ language: 'sr-Cyrl' })
    expectCsrfOnWrites(requests)
  })

  it("goes back to the browser's language when the choice is cleared", async () => {
    const { requests } = renderAt('/', me({ ...admin, language: 'hr' }))

    fireEvent.change(await screen.findByRole('combobox', { name: 'Jezik' }), { target: { value: '' } })

    expect(await screen.findByText('Language saved.')).toBeTruthy()
    expect(await requests.find((r) => r.method === 'PATCH')!.json()).toEqual({ language: null })
    expect(loadSavedLanguage()).toBeNull()
  })

  it('says when the language could not be saved', async () => {
    renderAt('/', { ...staff(admin), 'PATCH /auth/me': () => apiError(500, 'internal error') })

    fireEvent.change(await screen.findByRole('combobox', { name: 'Language' }), { target: { value: 'bs' } })

    expect((await screen.findByRole('alert')).textContent).toBe('Nešto nije u redu. Pokušajte ponovo.')
  })

  it('gives a reviewer the term review in the navigation, and keeps it out of everyone else\'s', async () => {
    const reviewRoutes: Record<string, Route> = {
      'GET /terminology/review': () => Response.json({ entries: [] }),
      'GET /terminology/review/progress': () => Response.json({ languages: [] }),
    }
    renderAt('/', { ...staff({ ...admin, termReviewer: true }), ...reviewRoutes })
    expect(await screen.findByRole('heading', { name: 'Calendar' })).toBeTruthy()
    fireEvent.click(screen.getByRole('link', { name: 'Term review' }))
    expect(await screen.findByRole('heading', { name: 'Term review' })).toBeTruthy()
    expect(location.pathname).toBe('/term-review')
    cleanup()

    renderAt('/term-review', { ...staff(admin), ...reviewRoutes })
    expect(await screen.findByRole('heading', { name: 'Term review' })).toBeTruthy()
    expect(screen.queryByRole('link', { name: 'Term review' })).toBeNull()
    expect(screen.getByText('Only reviewers can edit and approve terms.')).toBeTruthy()
  })
})
