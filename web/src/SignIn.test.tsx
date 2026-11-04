import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { SignIn } from './SignIn'
import { newApiClient } from './api/client'
import { apiError, expectCsrfOnWrites, fakeFetch, paths, type Route } from './test/fakeFetch'

afterEach(cleanup)

const secret = 'JBSWY3DPEHPK3PXPJBSWY3DPEHPK3PXP'
const uri = `otpauth://totp/Salus:staff%40example.test?secret=${secret}&issuer=Salus`

function renderSignIn(routes: Record<string, Route>) {
  const fake = fakeFetch(routes)
  const onSignedIn = vi.fn()
  render(<SignIn api={newApiClient(fake.fetch)} onSignedIn={onSignedIn} />)
  return { ...fake, onSignedIn }
}

function signInWithPassword() {
  fireEvent.change(screen.getByLabelText('E-mail'), { target: { value: 'staff@example.test' } })
  fireEvent.change(screen.getByLabelText('Password'), { target: { value: 'correct horse battery' } })
  fireEvent.click(screen.getByRole('button', { name: 'Sign in' }))
}

async function enterCode(button: string) {
  fireEvent.change(await screen.findByLabelText('Code'), { target: { value: '123456' } })
  fireEvent.click(screen.getByRole('button', { name: button }))
}

describe('SignIn', () => {
  it('signs in a patient without MFA in one step', async () => {
    const { requests, onSignedIn } = renderSignIn({ 'POST /auth/login': () => Response.json({ next: 'done' }) })

    signInWithPassword()

    await vi.waitFor(() => expect(onSignedIn).toHaveBeenCalled())
    expect(await requests[0].json()).toEqual({ email: 'staff@example.test', password: 'correct horse battery' })
    expectCsrfOnWrites(requests)
  })

  it('disables the button while the password is checked', async () => {
    renderSignIn({ 'POST /auth/login': () => new Promise<Response>(() => {}) })

    signInWithPassword()

    expect((await screen.findByRole('button', { name: 'Signing in' })).hasAttribute('disabled')).toBe(true)
  })

  it('asks for the code after the password and verifies it with the challenge', async () => {
    const { requests, onSignedIn } = renderSignIn({
      'POST /auth/login': () => Response.json({ next: 'verify', challenge: 'chal-1' }),
      'POST /auth/mfa/verify': () => Response.json({ next: 'done' }),
    })

    signInWithPassword()
    await enterCode('Continue')

    await vi.waitFor(() => expect(onSignedIn).toHaveBeenCalled())
    expect(paths(requests)).toEqual(['POST /auth/login', 'POST /auth/mfa/verify'])
    expect(await requests[1].json()).toEqual({ challenge: 'chal-1', code: '123456' })
    expectCsrfOnWrites(requests)
  })

  it('enrols staff at their first sign-in: QR code and secret, then the first code', async () => {
    const { requests, onSignedIn } = renderSignIn({
      'POST /auth/login': () => Response.json({ next: 'enrol', challenge: 'chal-2' }),
      'POST /auth/mfa/enrol': () => Response.json({ secret, uri }),
      'POST /auth/mfa/confirm': () => Response.json({ next: 'done' }),
    })

    signInWithPassword()

    expect(await screen.findByRole('img', { name: 'QR code for your authenticator app' })).toBeTruthy()
    expect(screen.getByText(secret)).toBeTruthy()
    await enterCode('Confirm')

    await vi.waitFor(() => expect(onSignedIn).toHaveBeenCalled())
    expect(paths(requests)).toEqual(['POST /auth/login', 'POST /auth/mfa/enrol', 'POST /auth/mfa/confirm'])
    expect(await requests[1].json()).toEqual({ challenge: 'chal-2' })
    expect(await requests[2].json()).toEqual({ challenge: 'chal-2', code: '123456' })
    expectCsrfOnWrites(requests)
    expect(document.body.innerHTML).not.toContain(secret)
    expect(screen.queryByRole('img', { name: 'QR code for your authenticator app' })).toBeNull()
  })

  it('says when the e-mail or password is wrong', async () => {
    renderSignIn({ 'POST /auth/login': () => apiError(401, 'wrong_credentials') })

    signInWithPassword()

    expect(await screen.findByRole('alert')).toHaveProperty('textContent', 'The e-mail or password is wrong.')
  })

  it('says how long to wait when rate limited', async () => {
    renderSignIn({ 'POST /auth/login': () => apiError(429, 'rate_limited', { 'Retry-After': '120' }) })

    signInWithPassword()

    expect(await screen.findByRole('alert')).toHaveProperty(
      'textContent',
      'Too many attempts. Try again in 2 minutes.',
    )
  })

  it('keeps the code step after a wrong code', async () => {
    renderSignIn({
      'POST /auth/login': () => Response.json({ next: 'verify', challenge: 'chal-1' }),
      'POST /auth/mfa/verify': () => apiError(401, 'wrong_code'),
    })

    signInWithPassword()
    await enterCode('Continue')

    expect(await screen.findByRole('alert')).toHaveProperty(
      'textContent',
      'The code is wrong. Check the time on your device and try again.',
    )
    expect(screen.getByLabelText('Code')).toBeTruthy()
  })

  it('starts again when the challenge has expired', async () => {
    renderSignIn({
      'POST /auth/login': () => Response.json({ next: 'verify', challenge: 'chal-1' }),
      'POST /auth/mfa/verify': () => apiError(401, 'invalid_challenge'),
    })

    signInWithPassword()
    await enterCode('Continue')

    expect(await screen.findByRole('status')).toHaveProperty('textContent', 'Your sign-in timed out. Sign in again.')
    expect(screen.getByLabelText('Password')).toBeTruthy()
  })

  it('gives a fixed message for anything unexpected', async () => {
    renderSignIn({ 'POST /auth/login': () => apiError(500, 'internal error') })

    signInWithPassword()

    expect(await screen.findByRole('alert')).toHaveProperty('textContent', 'Something went wrong. Try again.')
  })

  it('gives the same message when the API cannot be reached', async () => {
    renderSignIn({ 'POST /auth/login': () => Promise.reject(new TypeError('network')) })

    signInWithPassword()

    expect(await screen.findByRole('alert')).toHaveProperty('textContent', 'Something went wrong. Try again.')
  })
})
