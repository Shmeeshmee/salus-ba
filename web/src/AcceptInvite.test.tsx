import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { AcceptInvite } from './AcceptInvite'
import { newApiClient } from './api/client'
import { apiError, expectCsrfOnWrites, fakeFetch, paths, type Route } from './test/fakeFetch'

afterEach(cleanup)
beforeEach(() => history.replaceState(null, '', '/invite#tok-synthetic'))

const secret = 'GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ'
const acceptance = { challenge: 'chal-3', secret, uri: `otpauth://totp/Salus:new%40example.test?secret=${secret}` }

function renderAccept(routes: Record<string, Route>) {
  const fake = fakeFetch(routes)
  const onDone = vi.fn()
  render(<AcceptInvite api={newApiClient(fake.fetch)} onDone={onDone} />)
  return { ...fake, onDone }
}

function choosePassword(password: string, repeat = password) {
  fireEvent.change(screen.getByLabelText('New password'), { target: { value: password } })
  fireEvent.change(screen.getByLabelText('Repeat password'), { target: { value: repeat } })
  fireEvent.click(screen.getByRole('button', { name: 'Accept invite' }))
}

describe('AcceptInvite', () => {
  it('takes the token from the fragment and removes it from the address', () => {
    renderAccept({})

    expect(location.hash).toBe('')
    expect(location.pathname).toBe('/invite')
  })

  it('says the link is incomplete without a token', () => {
    history.replaceState(null, '', '/invite')

    renderAccept({})

    expect(screen.getByRole('alert').textContent).toBe(
      'This link is incomplete. Open the link in your invite e-mail again.',
    )
    expect(screen.queryByLabelText('New password')).toBeNull()
  })

  it('checks the password length and the repetition before sending anything', async () => {
    const { requests } = renderAccept({})

    choosePassword('short')
    expect((await screen.findByRole('alert')).textContent).toBe('Use 12 to 128 characters.')

    choosePassword('a long enough password', 'a different password')
    expect((await screen.findByRole('alert')).textContent).toBe('The passwords do not match.')
    expect(requests).toHaveLength(0)
  })

  it('accepts, enrols the authenticator app and confirms the first code', async () => {
    const { requests, onDone } = renderAccept({
      'POST /invites/accept': () => Response.json(acceptance, { status: 201 }),
      'POST /auth/mfa/confirm': () => Response.json({ next: 'done' }),
    })

    choosePassword('a long enough password')
    expect(await screen.findByRole('img', { name: 'QR code for your authenticator app' })).toBeTruthy()
    expect(screen.getByText(secret)).toBeTruthy()
    fireEvent.change(screen.getByLabelText('Code'), { target: { value: '654321' } })
    fireEvent.click(screen.getByRole('button', { name: 'Confirm' }))

    await vi.waitFor(() => expect(onDone).toHaveBeenCalled())
    expect(paths(requests)).toEqual(['POST /invites/accept', 'POST /auth/mfa/confirm'])
    expect(await requests[0].json()).toEqual({ token: 'tok-synthetic', password: 'a long enough password' })
    expect(await requests[1].json()).toEqual({ challenge: 'chal-3', code: '654321' })
    expectCsrfOnWrites(requests)
    expect(document.body.innerHTML).not.toContain(secret)
  })

  it('keeps the enrolment after a wrong code', async () => {
    renderAccept({
      'POST /invites/accept': () => Response.json(acceptance, { status: 201 }),
      'POST /auth/mfa/confirm': () => apiError(401, 'wrong_code'),
    })

    choosePassword('a long enough password')
    fireEvent.change(await screen.findByLabelText('Code'), { target: { value: '000000' } })
    fireEvent.click(screen.getByRole('button', { name: 'Confirm' }))

    expect((await screen.findByRole('alert')).textContent).toBe(
      'The code is wrong. Check the time on your device and try again.',
    )
    expect(screen.getByText(secret)).toBeTruthy()
  })

  it('sends the user to sign in when the set-up timed out', async () => {
    renderAccept({
      'POST /invites/accept': () => Response.json(acceptance, { status: 201 }),
      'POST /auth/mfa/confirm': () => apiError(401, 'invalid_challenge'),
    })

    choosePassword('a long enough password')
    fireEvent.change(await screen.findByLabelText('Code'), { target: { value: '000000' } })
    fireEvent.click(screen.getByRole('button', { name: 'Confirm' }))

    expect((await screen.findByRole('status')).textContent).toBe(
      'The set-up timed out. Your account exists: sign in with your new password to finish it.',
    )
    expect(screen.getByRole('link', { name: 'Sign in' }).getAttribute('href')).toBe('/sign-in')
    expect(document.body.innerHTML).not.toContain(secret)
  })

  it.each([
    [404, 'invalid_invite', 'This invite no longer works. Ask your clinic admin for a new one.'],
    [409, 'email_taken', 'This e-mail address already has a Salus account. Ask your clinic admin for help.'],
    [400, 'invalid_password', 'Use 12 to 128 characters.'],
    [500, 'internal error', 'Something went wrong. Try again.'],
  ])('maps %i %s to a message', async (status, code, message) => {
    renderAccept({ 'POST /invites/accept': () => apiError(status, code) })

    choosePassword('a long enough password')

    expect((await screen.findByRole('alert')).textContent).toBe(message)
  })
})
