import { cleanup, fireEvent, screen } from '@testing-library/react'
import { StrictMode } from 'react'
import { afterEach, describe, expect, it } from 'vitest'
import { newApiClient } from './api/client'
import { renderInLanguage } from './i18n/renderInLanguage'
import { Signup, VerifySignup } from './Signup'
import { apiError, expectCsrfOnWrites, fakeFetch, paths, type Route } from './test/fakeFetch'

afterEach(() => {
  cleanup()
  localStorage.clear()
})

function renderSignup(routes: Record<string, Route>) {
  const fake = fakeFetch(routes)
  renderInLanguage(<Signup api={newApiClient(fake.fetch)} />, 'en')
  return fake
}

function fill(email: string, clinic: string) {
  fireEvent.change(screen.getByLabelText('Your e-mail'), { target: { value: email } })
  fireEvent.change(screen.getByLabelText('Clinic name'), { target: { value: clinic } })
  fireEvent.click(screen.getByRole('button', { name: 'Sign up' }))
}

describe('Signup', () => {
  it('sends the clinic name, the e-mail and the language, then asks to check the e-mail', async () => {
    const { requests } = renderSignup({ 'POST /signup': () => new Response(null, { status: 202 }) })

    fill('admin@example.test', 'Synthetic clinic')

    expect((await screen.findByRole('status')).textContent).toBe(
      'Check your e-mail. If the address can be used, a link to confirm it is on its way.',
    )
    expect(paths(requests)).toEqual(['POST /signup'])
    expect(await requests[0].json()).toEqual({ email: 'admin@example.test', clinicName: 'Synthetic clinic', language: 'en' })
    expectCsrfOnWrites(requests)
  })

  it('asks for both fields before sending anything', async () => {
    const { requests } = renderSignup({})

    fill('', 'Synthetic clinic')

    expect((await screen.findByRole('alert')).textContent).toBe('Enter your e-mail and the clinic name.')
    expect(requests).toHaveLength(0)
  })

  it.each([
    [400, 'invalid_signup', 'Check the e-mail address and the clinic name.'],
    [429, 'rate_limited', 'Too many sign-ups. Try again later.'],
    [500, 'internal error', 'Something went wrong. Try again.'],
  ])('maps %i %s to a message', async (status, code, message) => {
    renderSignup({ 'POST /signup': () => apiError(status, code) })

    fill('admin@example.test', 'Synthetic clinic')

    expect((await screen.findByRole('alert')).textContent).toBe(message)
  })
})

function renderVerify(routes: Record<string, Route>) {
  const fake = fakeFetch(routes)
  renderInLanguage(<VerifySignup api={newApiClient(fake.fetch)} />, 'en')
  return fake
}

describe('VerifySignup', () => {
  it('verifies the token from the fragment, removes it from the address and says what follows', async () => {
    history.replaceState(null, '', '/signup/verify#tok-synthetic')
    const { requests } = renderVerify({ 'POST /signup/verify': () => new Response(null, { status: 204 }) })

    expect((await screen.findByRole('status')).textContent).toBe(
      'Your e-mail is confirmed. We have sent you an invitation to set up the clinic admin account.',
    )
    expect(location.hash).toBe('')
    expect(await requests[0].json()).toEqual({ token: 'tok-synthetic' })
    expectCsrfOnWrites(requests)
  })

  // StrictMode runs the effect twice in development: the token is still sent once, or the second answer would say
  // the link no longer works after the first verified it.
  it('sends the token once under StrictMode', async () => {
    history.replaceState(null, '', '/signup/verify#tok-synthetic')
    const fake = fakeFetch({ 'POST /signup/verify': () => new Response(null, { status: 204 }) })
    renderInLanguage(
      <StrictMode>
        <VerifySignup api={newApiClient(fake.fetch)} />
      </StrictMode>,
      'en',
    )

    expect((await screen.findByRole('status')).textContent).toContain('Your e-mail is confirmed.')
    expect(paths(fake.requests)).toEqual(['POST /signup/verify'])
  })

  it('says the link no longer works', async () => {
    history.replaceState(null, '', '/signup/verify#tok-synthetic')
    renderVerify({ 'POST /signup/verify': () => apiError(400, 'invalid_verification') })

    expect((await screen.findByRole('alert')).textContent).toBe('This link no longer works. Sign up again.')
    expect(screen.getByRole('link', { name: 'Sign up' }).getAttribute('href')).toBe('/signup')
  })

  it('says the link is incomplete without a token', () => {
    history.replaceState(null, '', '/signup/verify')
    const { requests } = renderVerify({})

    expect(screen.getByRole('alert').textContent).toBe('This link is incomplete. Open the link in your e-mail again.')
    expect(requests).toHaveLength(0)
  })
})
